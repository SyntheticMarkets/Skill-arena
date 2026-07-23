import { BrainCircuit, Compass, Shield, Zap } from 'lucide-react'

export const avatarOptions = [
  { key: 'vanguard', label: 'Vanguard', icon: Shield },
  { key: 'strategist', label: 'Strategist', icon: BrainCircuit },
  { key: 'pathfinder', label: 'Pathfinder', icon: Compass },
  { key: 'accelerator', label: 'Accelerator', icon: Zap },
] as const

export function PlayerAvatar({ avatarKey, fallback }: { avatarKey?: string; fallback: string }) {
  const option = avatarOptions.find((item) => item.key === avatarKey)
  if (!option) return <>{fallback}</>
  const Icon = option.icon
  return <Icon aria-hidden="true" />
}
