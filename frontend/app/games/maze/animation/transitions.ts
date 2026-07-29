import type { MazePresentation } from '../protocol/schemas'

export type MazeAnimation = {
  id: string
  presentation: MazePresentation
  kind: 'accepted' | 'blocked'
}

export const rendererTiming = {
  acceptedMs: 620,
  blockedMs: 760,
  reducedMs: 90,
  replayGapMs: 180,
} as const

export function animationDuration(animation: MazeAnimation, reducedMotion: boolean) {
  if (reducedMotion) return rendererTiming.reducedMs
  return animation.kind === 'accepted'
    ? rendererTiming.acceptedMs
    : rendererTiming.blockedMs
}

export function directionOffset(direction: MazePresentation['direction'], distance: number) {
  switch (direction) {
    case 'left':
      return { x: -distance, y: 0 }
    case 'right':
      return { x: distance, y: 0 }
    case 'up':
      return { x: 0, y: -distance }
    case 'down':
      return { x: 0, y: distance }
  }
}

