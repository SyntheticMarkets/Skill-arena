import { describe, expect, it } from 'vitest'
import type { GameActionResult, GameSync, RealtimeEvent } from '../../../lib/realtime'
import {
  MazeSnapshot,
  parseMazeActionResult,
  parseMazeReplayAction,
  parseMazeSnapshot,
} from './schemas'

export const snapshotPayload: MazeSnapshot = {
  matchId: 'match-1',
  participantId: 'user-1',
  puzzleId: 'puzzle-1',
  puzzleHash: 'a'.repeat(64),
  columns: 6,
  rows: 6,
  arrows: [
    {
      id: 'a0000',
      direction: 'right',
      cells: [{ column: 1, row: 2 }, { column: 2, row: 2 }],
    },
    {
      id: 'a0001',
      direction: 'up',
      cells: [{ column: 4, row: 4 }, { column: 4, row: 3 }],
    },
  ],
  removedIds: [],
  progress: {
    removed: 0,
    remaining: 2,
    total: 2,
    completionBps: 0,
    successfulActions: 0,
    blockedActions: 0,
    currentCombo: 0,
    maximumCombo: 0,
    complete: false,
  },
  status: 'active',
  startedAtMs: 1_000,
  deadlineAtMs: 121_000,
  completedAtMs: -1,
}

export function gameSync(payload: unknown = snapshotPayload): GameSync {
  return {
    snapshot: {
      rendererVersion: '1',
      stateVersion: 0,
      payload,
      checksum: 'snapshot-checksum',
    },
    stateVersion: 0,
    lastClientSequence: 0,
    lastServerSequence: 1,
  }
}

describe('Maze renderer protocol', () => {
  it('accepts a complete server renderer projection', () => {
    const parsed = parseMazeSnapshot(gameSync())
    expect(parsed.arrows).toHaveLength(2)
    expect(parsed.completedAtMs).toBe(-1)
    expect(parsed.puzzleHash).toHaveLength(64)
  })

  it('treats an omitted empty removed-arrow set as empty', () => {
    const payload = structuredClone(snapshotPayload) as unknown as Record<string, unknown>
    delete payload.removedIds

    expect(parseMazeSnapshot(gameSync(payload)).removedIds).toEqual([])
  })

  it('fails closed on version drift and out-of-bounds geometry', () => {
    const version = gameSync()
    version.snapshot.stateVersion = 2
    expect(() => parseMazeSnapshot(version)).toThrow(/version/)

    const outside = structuredClone(snapshotPayload)
    outside.arrows[0].cells[0].column = 99
    expect(() => parseMazeSnapshot(gameSync(outside))).toThrow(/outside/)
  })

  it('derives animation only from the authoritative receipt', () => {
    const next = structuredClone(snapshotPayload)
    next.removedIds = ['a0000']
    next.progress = {
      ...next.progress,
      removed: 1,
      remaining: 1,
      completionBps: 5000,
      successfulActions: 1,
      currentCombo: 1,
      maximumCombo: 1,
    }
    const result = {
      receipt: {
        actionId: 'action-1',
        matchId: 'match-1',
        userId: 'user-1',
        clientSequence: 1,
        expectedStateVersion: 0,
        actionKind: 'arrow.click',
        accepted: true,
        resultCode: 'ACTION_ACCEPTED',
        stateVersionBefore: 0,
        stateVersionAfter: 1,
        firstEventSequence: 2,
        lastEventSequence: 3,
        receiptHash: 'receipt-hash',
        transition: {
          accepted: true,
          code: 'ACTION_ACCEPTED',
          presentation: {
            arrowId: 'a0000',
            direction: 'right',
            blocked: false,
            escapeDistance: 5,
            returnToOrigin: false,
            removeAfterExit: true,
          },
        },
      },
      snapshot: {
        rendererVersion: '1',
        stateVersion: 1,
        payload: next,
        checksum: 'next-checksum',
      },
      duplicate: false,
    } satisfies GameActionResult
    const parsed = parseMazeActionResult(result)
    expect(parsed.accepted).toBe(true)
    expect(parsed.presentation.escapeDistance).toBe(5)
    expect(parsed.snapshot.removedIds).toEqual(['a0000'])
  })

  it('reads replay presentation from committed events without solving', () => {
    const event = {
      id: 'event-1',
      matchId: 'match-1',
      type: 'game.action.processed',
      sequence: 2,
      stateVersion: 1,
      serverTime: new Date().toISOString(),
      integrityHash: 'event-hash',
      payload: {
        occurredAtMs: 2_000,
        accepted: false,
        code: 'ACTION_BLOCKED',
        progress: { ...snapshotPayload.progress, blockedActions: 1 },
        presentation: {
          arrowId: 'a0001',
          direction: 'up',
          blocked: true,
          blockerId: 'a0000',
          collisionCell: { column: 4, row: 1 },
          approachDistance: 1,
          returnToOrigin: true,
          removeAfterExit: false,
        },
      },
    } satisfies RealtimeEvent
    expect(parseMazeReplayAction(event)).toMatchObject({
      accepted: false,
      presentation: { arrowId: 'a0001', blockerId: 'a0000' },
    })
  })
})
