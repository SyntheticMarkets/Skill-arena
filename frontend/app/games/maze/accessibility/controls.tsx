'use client'

import { ArrowDown, ArrowLeft, ArrowRight, ArrowUp } from 'lucide-react'
import type { MazeArrow } from '../protocol/schemas'

const icons = {
  up: ArrowUp,
  down: ArrowDown,
  left: ArrowLeft,
  right: ArrowRight,
}

export function MazeAccessibleControls({
  arrows,
  removed,
  disabled,
  onArrow,
}: {
  arrows: MazeArrow[]
  removed: Set<string>
  disabled: boolean
  onArrow: (arrowId: string) => void
}) {
  const active = arrows.filter((arrow) => !removed.has(arrow.id))
  return (
    <details className="maze-keyboard-controls">
      <summary>Keyboard arrow controls</summary>
      <div role="group" aria-label="Maze arrows">
        {active.map((arrow, index) => {
          const Icon = icons[arrow.direction]
          const head = arrow.cells[arrow.cells.length - 1]
          return (
            <button
              key={arrow.id}
              type="button"
              disabled={disabled}
              onClick={() => onArrow(arrow.id)}
              aria-label={`Arrow ${index + 1}, points ${arrow.direction}, row ${head.row + 1}, column ${head.column + 1}`}
              title={`Arrow ${index + 1}: ${arrow.direction}`}
            >
              <Icon aria-hidden="true" />
              <span>{index + 1}</span>
            </button>
          )
        })}
      </div>
    </details>
  )
}

