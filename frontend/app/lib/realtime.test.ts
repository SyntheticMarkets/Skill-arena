import { beforeEach, describe, expect, it, vi } from 'vitest'
import { enterRealtimeQueue, RealtimeClient } from './realtime'

class MockWebSocket extends EventTarget {
  static OPEN = 1
  static instances: MockWebSocket[] = []
  readyState = MockWebSocket.OPEN
  sent: string[] = []
  constructor(public url: string | URL) {
    super()
    MockWebSocket.instances.push(this)
  }
  send(value: string) { this.sent.push(value) }
  close() { this.dispatchEvent(new Event('close')) }
  open() { this.dispatchEvent(new Event('open')) }
  message(value: unknown) { this.dispatchEvent(new MessageEvent('message', { data: JSON.stringify(value) })) }
}

describe('Realtime client', () => {
  beforeEach(() => {
    MockWebSocket.instances = []
    vi.stubGlobal('WebSocket', MockWebSocket)
    vi.restoreAllMocks()
  })

  it('enters queues through the versioned provider-neutral API', async () => {
    const fetchMock = vi.spyOn(globalThis, 'fetch').mockResolvedValue(new Response(JSON.stringify({ queue: { id: 'q1', status: 'waiting' }, match: null }), { status: 200 }))
    await enterRealtimeQueue({ gameId: 'test', mode: 'pvp', walletCategory: 'practice', region: 'af-south', jurisdiction: 'ZA', latencyMs: 20 })
    expect(String(fetchMock.mock.calls[0][0])).toContain('/api/v1/realtime/queue')
    expect(fetchMock.mock.calls[0][1]?.credentials).toBe('include')
  })

  it('recovers by sequence and acknowledges only new authoritative events', () => {
    const events: number[] = []
    const client = new RealtimeClient({ onEvent: (event) => events.push(event.sequence) })
    client.connect('match-1', 4)
    const socket = MockWebSocket.instances[0]
    socket.open()
    expect(JSON.parse(socket.sent[0])).toMatchObject({ type: 'reconnect', matchId: 'match-1', afterSequence: 4 })
    socket.message({ type: 'match.event', event: { matchId: 'match-1', sequence: 5, integrityHash: 'hash' } })
    socket.message({ type: 'match.event', event: { matchId: 'match-1', sequence: 5, integrityHash: 'hash' } })
    expect(events).toEqual([5])
    expect(JSON.parse(socket.sent[socket.sent.length - 1] ?? '{}')).toMatchObject({ type: 'ack', afterSequence: 5 })
    client.close()
  })

  it('never sends client-authored match state', () => {
    const client = new RealtimeClient()
    client.connect('match-1')
    const socket = MockWebSocket.instances[0]
    socket.open()
    client.ready()
    expect(socket.sent.map((item) => JSON.parse(item).type)).toEqual(['reconnect', 'ready'])
    client.close()
  })

  it('sends game intent and exposes only authoritative receipts and snapshots', () => {
    const receipts: string[] = []
    const snapshots: number[] = []
    const client = new RealtimeClient({
      onActionReceipt: (result) => receipts.push(result.receipt.actionId),
      onGameSync: (game) => snapshots.push(game.stateVersion),
    })
    client.connect()
    const socket = MockWebSocket.instances[0]
    socket.open()
    client.ready('match-1')
    client.submitGameAction({
      actionId: 'action-1',
      kind: 'arrow.click',
      payload: { arrowId: 'a0000' },
      clientSequence: 1,
      expectedStateVersion: 0,
    })
    const sent = JSON.parse(socket.sent[socket.sent.length - 1])
    expect(sent).toEqual({
      type: 'game.action',
      matchId: 'match-1',
      actionId: 'action-1',
      kind: 'arrow.click',
      payload: { arrowId: 'a0000' },
      clientSequence: 1,
      expectedStateVersion: 0,
      latencyMs: 0,
    })
    expect(sent).not.toHaveProperty('accepted')
    expect(sent).not.toHaveProperty('state')
    expect(sent).not.toHaveProperty('progress')

    socket.message({
      type: 'game.state.sync',
      game: {
        stateVersion: 3,
        lastClientSequence: 4,
        lastServerSequence: 9,
        snapshot: { rendererVersion: '1', stateVersion: 3, payload: {}, checksum: 'hash' },
      },
    })
    socket.message({
      type: 'game.action.receipt',
      result: { receipt: { actionId: 'action-1' }, snapshot: {}, duplicate: false },
    })
    expect(snapshots).toEqual([3])
    expect(receipts).toEqual(['action-1'])
    client.close()
  })
})
