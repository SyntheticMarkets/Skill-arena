import type { GameRendererRegistration } from '../interfaces/renderer'
import { MazeArena } from '../maze/MazeArena'

const registrations: GameRendererRegistration[] = [
  {
    rendererKey: 'maze-arena',
    gameId: 'maze_arena',
    gameVersion: '1.0.0',
    component: MazeArena,
  },
]

export function resolveRenderer(rendererKey: string, gameVersion: string) {
  return registrations.find(
    (registration) =>
      registration.rendererKey === rendererKey &&
      registration.gameVersion === gameVersion,
  )
}

export function registeredRenderers() {
  return registrations.map((registration) => ({ ...registration }))
}

