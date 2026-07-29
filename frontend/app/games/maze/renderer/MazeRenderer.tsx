'use client'

import { MazeAccessibleControls } from '../accessibility/controls'
import type { MazeAnimation } from '../animation/transitions'
import type { MazeSnapshot } from '../protocol/schemas'
import { MazeBoard } from './MazeBoard'

export function MazeRenderer({
  snapshot,
  animation,
  reducedMotion,
  disabled,
  onArrow,
}: {
  snapshot: MazeSnapshot
  animation: MazeAnimation | null
  reducedMotion: boolean
  disabled: boolean
  onArrow: (arrowId: string) => void
}) {
  const removed = new Set(snapshot.removedIds)
  return (
    <section className="maze-renderer" aria-label="Authoritative Maze Arena board">
      <MazeBoard
        snapshot={snapshot}
        animation={animation}
        reducedMotion={reducedMotion}
        disabled={disabled}
        onArrow={onArrow}
      />
      <MazeAccessibleControls
        arrows={snapshot.arrows}
        removed={removed}
        disabled={disabled}
        onArrow={onArrow}
      />
    </section>
  )
}

