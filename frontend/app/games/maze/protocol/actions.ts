export const mazeArrowClickKind = 'arrow.click'

export function createMazeArrowAction(input: {
  matchId: string
  arrowId: string
  clientSequence: number
  expectedStateVersion: number
  latencyMs?: number
}) {
  return {
    matchId: input.matchId,
    actionId: crypto.randomUUID(),
    kind: mazeArrowClickKind,
    payload: { arrowId: input.arrowId },
    clientSequence: input.clientSequence,
    expectedStateVersion: input.expectedStateVersion,
    latencyMs: input.latencyMs ?? 0,
  }
}

