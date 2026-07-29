import { apiBase, apiFetch } from './api'

export type RealtimeQueueRequest = {
  gameId: string
  mode: 'practice' | 'pvp' | 'tournament'
  walletCategory: string
  region: string
  jurisdiction: string
  latencyMs: number
}

export type RealtimeEvent = {
  id: string
  matchId: string
  userId?: string
  type: string
  sequence: number
  stateVersion: number
  serverTime: string
  payload?: unknown
  previousHash?: string
  integrityHash: string
}

export type RealtimeMatch = {
  id: string
  gameId: string
  gameVersion?: string
  replayVersion?: string
  mode: string
  status: string
  stateVersion: number
  sequence: number
}

export type RendererSnapshot = {
  rendererVersion: string
  stateVersion: number
  payload: unknown
  checksum: string
}

export type GameSync = {
  snapshot: RendererSnapshot
  stateVersion: number
  lastClientSequence: number
  lastServerSequence: number
}

export type GamePresentationEvent = {
  kind: string
  payload: unknown
}

export type GameActionResult = {
  receipt: {
    actionId: string
    matchId: string
    userId: string
    clientSequence: number
    expectedStateVersion: number
    actionKind: string
    accepted: boolean
    resultCode: string
    stateVersionBefore: number
    stateVersionAfter: number
    firstEventSequence: number
    lastEventSequence: number
    receiptHash: string
    transition: {
      accepted: boolean
      code: string
      progress?: unknown
      presentation?: unknown
      events?: GamePresentationEvent[]
      completion?: { status: string; reason?: string }
    }
  }
  snapshot: RendererSnapshot
  events?: RealtimeEvent[]
  completion?: { status: string; reason?: string }
  outcome?: { status: string; winnerIds?: string[]; loserIds?: string[]; reason?: string }
  duplicate: boolean
}

export type QueueResult = {
  queue: { id: string; status: string; matchId?: string }
  match: RealtimeMatch | null
}

type GatewayMessage = {
  type: string
  code?: string
  connectionId?: string
  serverTime?: string
  match?: RealtimeMatch
  event?: RealtimeEvent
  events?: RealtimeEvent[]
  game?: GameSync
  result?: GameActionResult
}

type RealtimeClientOptions = {
  onEvent?: (event: RealtimeEvent) => void
  onSync?: (match: RealtimeMatch, events: RealtimeEvent[], game?: GameSync) => void
  onMatch?: (match: RealtimeMatch) => void
  onGameSync?: (game: GameSync) => void
  onActionReceipt?: (result: GameActionResult) => void
  onStatus?: (status: 'connecting' | 'connected' | 'reconnecting' | 'closed') => void
  onError?: (code: string) => void
}

export function enterRealtimeQueue(request: RealtimeQueueRequest) {
  return apiFetch<QueueResult>('/api/v1/realtime/queue', { method: 'POST', body: JSON.stringify(request) })
}

export function cancelRealtimeQueue() {
  return apiFetch<{ id: string; status: string }>('/api/v1/realtime/queue', { method: 'DELETE' })
}

export class RealtimeClient {
  private socket: WebSocket | null = null
  private heartbeat: ReturnType<typeof setInterval> | null = null
  private reconnectTimer: ReturnType<typeof setTimeout> | null = null
  private reconnectAttempts = 0
  private manuallyClosed = false
  private matchId = ''
  private sequence = 0
  private region = 'global'
  private readonly options: RealtimeClientOptions

  constructor(options: RealtimeClientOptions = {}) {
    this.options = options
  }

  connect(matchId = '', afterSequence = 0, region = 'global') {
    this.matchId = matchId
    this.sequence = afterSequence
    this.region = region
    this.manuallyClosed = false
    this.open()
  }

  ready(matchId = this.matchId) {
    this.matchId = matchId
    this.send({ type: 'ready', matchId })
  }

  submitGameAction(action: {
    matchId?: string
    actionId: string
    kind: string
    payload: unknown
    clientSequence: number
    expectedStateVersion: number
    latencyMs?: number
  }) {
    const matchId = action.matchId ?? this.matchId
    this.send({
      type: 'game.action',
      matchId,
      actionId: action.actionId,
      kind: action.kind,
      payload: action.payload,
      clientSequence: action.clientSequence,
      expectedStateVersion: action.expectedStateVersion,
      latencyMs: action.latencyMs ?? 0,
    })
  }

  requestGameSync(matchId = this.matchId) {
    this.send({ type: 'game.sync.request', matchId })
  }

  leave(matchId = this.matchId) {
    this.send({ type: 'leave', matchId })
  }

  close() {
    this.manuallyClosed = true
    this.stopTimers()
    this.socket?.close(1000, 'client closed')
    this.socket = null
    this.options.onStatus?.('closed')
  }

  private open() {
    this.stopTimers()
    this.options.onStatus?.(this.reconnectAttempts ? 'reconnecting' : 'connecting')
    const url = new URL(apiBase)
    url.protocol = url.protocol === 'https:' ? 'wss:' : 'ws:'
    url.pathname = '/api/v1/realtime/gateway'
    url.search = ''
    const socket = new WebSocket(url)
    this.socket = socket
    socket.addEventListener('open', () => {
      this.reconnectAttempts = 0
      this.options.onStatus?.('connected')
      if (this.matchId) this.send({ type: 'reconnect', matchId: this.matchId, afterSequence: this.sequence })
      this.heartbeat = setInterval(() => {
        this.send({ type: 'heartbeat', matchId: this.matchId, region: this.region, latencyMs: 0 })
      }, 15_000)
    })
    socket.addEventListener('message', (message) => this.receive(message.data))
    socket.addEventListener('close', () => {
      this.stopTimers()
      if (!this.manuallyClosed) this.scheduleReconnect()
    })
    socket.addEventListener('error', () => this.options.onError?.('GATEWAY_UNAVAILABLE'))
  }

  private receive(raw: string) {
    let message: GatewayMessage
    try {
      message = JSON.parse(raw) as GatewayMessage
    } catch {
      this.options.onError?.('INVALID_GATEWAY_MESSAGE')
      return
    }
    if (message.type === 'error') {
      this.options.onError?.(message.code ?? 'REALTIME_ERROR')
      return
    }
    if (message.type === 'state.sync' && message.match) {
      const events = message.events ?? []
      this.advance(events)
      this.matchId = message.match.id
      this.options.onSync?.(message.match, events, message.game)
      if (message.game) this.options.onGameSync?.(message.game)
      return
    }
    if (message.type === 'match.status' && message.match) {
      this.matchId = message.match.id
      this.options.onMatch?.(message.match)
      return
    }
    if (message.type === 'game.state.sync' && message.game) {
      this.options.onGameSync?.(message.game)
      return
    }
    if (message.type === 'game.action.receipt' && message.result) {
      this.options.onActionReceipt?.(message.result)
      return
    }
    if (message.type === 'match.event' && message.event) {
      if (message.event.sequence <= this.sequence) return
      this.sequence = message.event.sequence
      this.options.onEvent?.(message.event)
      this.send({ type: 'ack', matchId: this.matchId, afterSequence: this.sequence })
    }
  }

  private advance(events: RealtimeEvent[]) {
    for (const event of events) {
      if (event.sequence > this.sequence) this.sequence = event.sequence
    }
  }

  private send(message: Record<string, unknown>) {
    if (this.socket?.readyState === WebSocket.OPEN) this.socket.send(JSON.stringify(message))
  }

  private scheduleReconnect() {
    if (this.reconnectAttempts >= 6) {
      this.options.onStatus?.('closed')
      this.options.onError?.('RECONNECT_EXHAUSTED')
      return
    }
    this.reconnectAttempts += 1
    const delay = Math.min(500 * 2 ** (this.reconnectAttempts - 1), 8_000)
    this.options.onStatus?.('reconnecting')
    this.reconnectTimer = setTimeout(() => this.open(), delay)
  }

  private stopTimers() {
    if (this.heartbeat) clearInterval(this.heartbeat)
    if (this.reconnectTimer) clearTimeout(this.reconnectTimer)
    this.heartbeat = null
    this.reconnectTimer = null
  }
}
