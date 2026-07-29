import type { MazePresentation, MazeProgress } from '../protocol/schemas'

export function actionAnnouncement(
  presentation: MazePresentation,
  progress: MazeProgress,
) {
  if (presentation.blocked) {
    return `Arrow blocked. It returned to its starting position. ${progress.remaining} arrows remain.`
  }
  if (progress.complete) {
    return 'Puzzle complete. Every arrow escaped the arena.'
  }
  const combo = progress.currentCombo > 1 ? ` Combo ${progress.currentCombo}.` : ''
  return `Arrow escaped.${combo} ${progress.remaining} arrows remain.`
}

