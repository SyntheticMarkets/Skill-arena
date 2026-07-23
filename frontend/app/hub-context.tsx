'use client'

import { createContext, useCallback, useContext, useEffect, useMemo, useState } from 'react'
import { useAuth } from './auth-context'
import { apiFetch } from './lib/api'

export type HubProfile = {
  userId: string
  username: string
  displayName: string
  avatarUrl?: string
  country: string
  language: string
}

export type HubGame = {
  id: string
  name: string
  description: string
  category: string
  version: string
  rendererKey: string
  modes: string[]
  averageTimeSeconds: number
  capabilities: {
    practice: boolean
    pvp: boolean
    replay: boolean
    tournament: boolean
    spectator: boolean
  }
  availability: string
  availabilityReason?: string
  rulesSummary: string[]
}

export type HubSnapshot = {
  generatedAt: string
  profile: HubProfile
  progression: {
    xp: number
    level: number
    prestige: number
    eloRating: number
    leagueTier: string
    seasonPoints: number
    matchesPlayed: number
    wins: number
    losses: number
    currentStreak: number
    trustScore: number
    trustTier: string
  }
  wallet: {
    currency: string
    availableBalance: number
    pendingDeposits: number
    pendingWithdrawals: number
    accountStatus: string
    verificationStatus: string
  }
  notifications: { unread: number; total: number }
  objectives: Array<{
    id: string
    title: string
    description: string
    progress: number
    target: number
    complete: boolean
    actionUrl: string
  }>
  recommendedAction: {
    id: string
    label: string
    description: string
    actionUrl: string
    reason: string
  }
  continueActivity?: HubActivity
  recentActivity: HubActivity[]
  tournaments: Array<{
    id: string
    name: string
    status: string
    startsAt: string
    eligible: boolean
    ineligibleReason?: string
  }>
  challenges: Array<{
    id: string
    type: string
    title: string
    status: string
    reason?: string
    actionUrl?: string
  }>
  games: HubGame[]
  eligibility: {
    emailVerified: boolean
    profileComplete: boolean
    mfaEnabled: boolean
    walletVisible: boolean
    liveEligible: boolean
    blockers: string[]
  }
}

export type HubActivity = {
  id: string
  type: string
  title: string
  description: string
  actionUrl?: string
  occurredAt: string
}

type HubState = {
  data: HubSnapshot | null
  status: 'idle' | 'loading' | 'ready' | 'error'
  error: string
  reload: () => Promise<void>
}

const HubContext = createContext<HubState | null>(null)

export function HubProvider({ children }: { children: React.ReactNode }) {
  const { status: authStatus } = useAuth()
  const [data, setData] = useState<HubSnapshot | null>(null)
  const [status, setStatus] = useState<HubState['status']>('idle')
  const [error, setError] = useState('')

  const reload = useCallback(async () => {
    if (authStatus !== 'authenticated') return
    setStatus('loading')
    setError('')
    try {
      setData(await apiFetch<HubSnapshot>('/api/v1/hub'))
      setStatus('ready')
    } catch (cause) {
      setError(cause instanceof Error ? cause.message : 'The Arena Hub could not be loaded.')
      setStatus('error')
    }
  }, [authStatus])

  useEffect(() => {
    const timer = window.setTimeout(() => {
      if (authStatus === 'authenticated') {
        void reload()
        return
      }
      setData(null)
      setStatus('idle')
      setError('')
    }, 0)
    return () => window.clearTimeout(timer)
  }, [authStatus, reload])

  const value = useMemo(() => ({ data, status, error, reload }), [data, error, reload, status])
  return <HubContext.Provider value={value}>{children}</HubContext.Provider>
}

export function useHub() {
  const value = useContext(HubContext)
  if (!value) throw new Error('useHub must be used inside HubProvider')
  return value
}
