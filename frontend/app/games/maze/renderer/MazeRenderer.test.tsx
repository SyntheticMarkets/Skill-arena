import { cleanup, fireEvent, render, screen } from '@testing-library/react'
import { afterEach, describe, expect, it, vi } from 'vitest'
import type { MazeSnapshot } from '../protocol/schemas'
import { snapshotPayload } from '../protocol/schemas.test'
import { MazeRenderer } from './MazeRenderer'

const snapshot = snapshotPayload as MazeSnapshot

afterEach(cleanup)

describe('Maze renderer', () => {
  it('renders stable geometry and accessible intent controls', () => {
    const onArrow = vi.fn()
    render(
      <MazeRenderer
        snapshot={snapshot}
        animation={null}
        reducedMotion={false}
        disabled={false}
        onArrow={onArrow}
      />,
    )
    expect(screen.getByRole('img', { name: /2 of 2 arrows remain/i })).toBeVisible()
    fireEvent.click(screen.getByText('Keyboard arrow controls'))
    const arrow = screen.getByRole('button', { name: /arrow 1, points right/i })
    fireEvent.click(arrow)
    expect(onArrow).toHaveBeenCalledWith('a0000')
    expect(screen.getByRole('button', { name: 'Zoom in' })).toBeEnabled()
    expect(screen.getByRole('button', { name: 'Fit board' })).toBeEnabled()
  })

  it('disables all game intents while an authoritative action is pending', () => {
    render(
      <MazeRenderer
        snapshot={snapshot}
        animation={{
          id: 'blocked',
          kind: 'blocked',
          presentation: {
            arrowId: 'a0000',
            direction: 'right',
            blocked: true,
            approachDistance: 1,
            returnToOrigin: true,
            removeAfterExit: false,
          },
        }}
        reducedMotion
        disabled
        onArrow={() => undefined}
      />,
    )
    fireEvent.click(screen.getByText('Keyboard arrow controls'))
    expect(screen.getByRole('button', { name: /arrow 1, points right/i })).toBeDisabled()
    expect(document.querySelector('.maze-arrow.is-blocked')).toBeTruthy()
  })
})
