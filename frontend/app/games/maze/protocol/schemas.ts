import type { GameActionResult, GameSync, RealtimeEvent } from '../../../lib/realtime'

export type MazeDirection = 'up' | 'down' | 'left' | 'right'

export type MazeCell = {
  column: number
  row: number
}

export type MazeArrow = {
  id: string
  cells: MazeCell[]
  direction: MazeDirection
}

export type MazeProgress = {
  removed: number
  remaining: number
  total: number
  completionBps: number
  successfulActions: number
  blockedActions: number
  currentCombo: number
  maximumCombo: number
  complete: boolean
}

export type MazeSnapshot = {
  matchId: string
  participantId: string
  puzzleId: string
  puzzleHash: string
  columns: number
  rows: number
  arrows: MazeArrow[]
  removedIds: string[]
  progress: MazeProgress
  status: 'active' | 'completed' | 'timed_out'
  startedAtMs: number
  deadlineAtMs: number
  completedAtMs: number
}

export type MazePresentation = {
  arrowId: string
  direction: MazeDirection
  blocked: boolean
  blockerId?: string
  collisionCell?: MazeCell
  approachDistance?: number
  escapeDistance?: number
  returnToOrigin: boolean
  removeAfterExit: boolean
}

export type MazeActionReceipt = {
  accepted: boolean
  code: string
  snapshot: MazeSnapshot
  presentation: MazePresentation
  clientSequence: number
  stateVersion: number
  outcome?: GameActionResult['outcome']
}

export type MazeReplayAction = {
  sequence: number
  occurredAtMs: number
  accepted: boolean
  code: string
  progress: MazeProgress
  presentation: MazePresentation
}

type RecordValue = Record<string, unknown>

function record(value: unknown, name: string): RecordValue {
  if (!value || typeof value !== 'object' || Array.isArray(value)) {
    throw new Error(`${name} is not an object`)
  }
  return value as RecordValue
}

function string(value: unknown, name: string, optional = false) {
  if (optional && value === undefined) return ''
  if (typeof value !== 'string' || value.length === 0) {
    throw new Error(`${name} is invalid`)
  }
  return value
}

function integer(value: unknown, name: string, minimum = 0) {
  if (!Number.isSafeInteger(value) || (value as number) < minimum) {
    throw new Error(`${name} is invalid`)
  }
  return value as number
}

function bool(value: unknown, name: string) {
  if (typeof value !== 'boolean') throw new Error(`${name} is invalid`)
  return value
}

function direction(value: unknown, name: string): MazeDirection {
  if (value !== 'up' && value !== 'down' && value !== 'left' && value !== 'right') {
    throw new Error(`${name} is invalid`)
  }
  return value
}

function cell(value: unknown, name: string): MazeCell {
  const item = record(value, name)
  return {
    column: integer(item.column, `${name}.column`),
    row: integer(item.row, `${name}.row`),
  }
}

function progress(value: unknown): MazeProgress {
  const item = record(value, 'progress')
  return {
    removed: integer(item.removed, 'progress.removed'),
    remaining: integer(item.remaining, 'progress.remaining'),
    total: integer(item.total, 'progress.total', 1),
    completionBps: integer(item.completionBps, 'progress.completionBps'),
    successfulActions: integer(item.successfulActions, 'progress.successfulActions'),
    blockedActions: integer(item.blockedActions, 'progress.blockedActions'),
    currentCombo: integer(item.currentCombo, 'progress.currentCombo'),
    maximumCombo: integer(item.maximumCombo, 'progress.maximumCombo'),
    complete: bool(item.complete, 'progress.complete'),
  }
}

export function parseMazeSnapshot(sync: GameSync): MazeSnapshot {
  if (sync.snapshot.rendererVersion !== '1') {
    throw new Error('Maze renderer version is unsupported')
  }
  if (sync.snapshot.stateVersion !== sync.stateVersion) {
    throw new Error('Maze snapshot version does not match the authoritative cursor')
  }
  const payload = record(sync.snapshot.payload, 'snapshot payload')
  const columns = integer(payload.columns, 'snapshot.columns', 1)
  const rows = integer(payload.rows, 'snapshot.rows', 1)
  if (!Array.isArray(payload.arrows)) {
    throw new Error('Maze geometry is unavailable')
  }
  const arrows = payload.arrows.map((value, index) => {
    const item = record(value, `arrows[${index}]`)
    if (!Array.isArray(item.cells) || item.cells.length === 0) {
      throw new Error(`arrows[${index}].cells is invalid`)
    }
    return {
      id: string(item.id, `arrows[${index}].id`),
      cells: item.cells.map((value, cellIndex) =>
        cell(value, `arrows[${index}].cells[${cellIndex}]`),
      ),
      direction: direction(item.direction, `arrows[${index}].direction`),
    }
  })
  const removedValues = payload.removedIds === undefined ? [] : payload.removedIds
  if (!Array.isArray(removedValues)) {
    throw new Error('Maze removed-arrow state is invalid')
  }
  const removedIds = removedValues.map((value, index) =>
    string(value, `removedIds[${index}]`),
  )
  const status = string(payload.status, 'snapshot.status')
  if (status !== 'active' && status !== 'completed' && status !== 'timed_out') {
    throw new Error('snapshot.status is unsupported')
  }
  const parsed: MazeSnapshot = {
    matchId: string(payload.matchId, 'snapshot.matchId'),
    participantId: string(payload.participantId, 'snapshot.participantId'),
    puzzleId: string(payload.puzzleId, 'snapshot.puzzleId'),
    puzzleHash: string(payload.puzzleHash, 'snapshot.puzzleHash'),
    columns,
    rows,
    arrows,
    removedIds,
    progress: progress(payload.progress),
    status,
    startedAtMs: integer(payload.startedAtMs, 'snapshot.startedAtMs'),
    deadlineAtMs: integer(payload.deadlineAtMs, 'snapshot.deadlineAtMs'),
    completedAtMs: integer(payload.completedAtMs, 'snapshot.completedAtMs', -1),
  }
  if (parsed.progress.total !== arrows.length) {
    throw new Error('Maze progress total does not match geometry')
  }
  for (const arrow of arrows) {
    for (const arrowCell of arrow.cells) {
      if (arrowCell.column >= columns || arrowCell.row >= rows) {
        throw new Error('Maze arrow cell is outside the renderer bounds')
      }
    }
  }
  return parsed
}

export function parseMazePresentation(value: unknown): MazePresentation {
  const item = record(value, 'presentation')
  const result: MazePresentation = {
    arrowId: string(item.arrowId, 'presentation.arrowId'),
    direction: direction(item.direction, 'presentation.direction'),
    blocked: bool(item.blocked, 'presentation.blocked'),
    returnToOrigin: bool(item.returnToOrigin, 'presentation.returnToOrigin'),
    removeAfterExit: bool(item.removeAfterExit, 'presentation.removeAfterExit'),
  }
  if (item.blockerId !== undefined) {
    result.blockerId = string(item.blockerId, 'presentation.blockerId')
  }
  if (item.collisionCell !== undefined) {
    result.collisionCell = cell(item.collisionCell, 'presentation.collisionCell')
  }
  if (item.approachDistance !== undefined) {
    result.approachDistance = integer(
      item.approachDistance,
      'presentation.approachDistance',
    )
  }
  if (item.escapeDistance !== undefined) {
    result.escapeDistance = integer(
      item.escapeDistance,
      'presentation.escapeDistance',
    )
  }
  return result
}

export function parseMazeActionResult(result: GameActionResult): MazeActionReceipt {
  const transition = result.receipt.transition
  if (!transition || transition.accepted !== result.receipt.accepted) {
    throw new Error('Maze action transition is inconsistent')
  }
  const sync: GameSync = {
    snapshot: result.snapshot,
    stateVersion: result.snapshot.stateVersion,
    lastClientSequence: result.receipt.clientSequence,
    lastServerSequence: result.receipt.lastEventSequence ?? 0,
  }
  return {
    accepted: result.receipt.accepted,
    code: result.receipt.resultCode,
    snapshot: parseMazeSnapshot(sync),
    presentation: parseMazePresentation(transition.presentation),
    clientSequence: result.receipt.clientSequence,
    stateVersion: result.receipt.stateVersionAfter,
    outcome: result.outcome,
  }
}

export function parseMazeReplayAction(event: RealtimeEvent): MazeReplayAction | null {
  if (event.type !== 'game.action.processed') return null
  const payload = record(event.payload, 'replay action')
  return {
    sequence: event.sequence,
    occurredAtMs: integer(payload.occurredAtMs, 'replay action.occurredAtMs'),
    accepted: bool(payload.accepted, 'replay action.accepted'),
    code: string(payload.code, 'replay action.code'),
    progress: progress(payload.progress),
    presentation: parseMazePresentation(payload.presentation),
  }
}
