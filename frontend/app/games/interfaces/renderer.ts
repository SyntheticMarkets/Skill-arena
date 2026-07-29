import type { ComponentType } from 'react'

export type GameRendererProps = {
  replayMatchId?: string
}

export type GameRendererRegistration = {
  rendererKey: string
  gameId: string
  gameVersion: string
  component: ComponentType<GameRendererProps>
}

