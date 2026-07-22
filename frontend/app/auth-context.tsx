'use client'

import { createContext, useCallback, useContext, useEffect, useMemo, useState } from 'react'
import { apiFetch, postJSON } from './lib/api'

export type SessionUser = {
  id: string
  email: string
  role: string
  emailVerified: boolean
  country: string
}

type SessionResponse = {
  authenticated: boolean
  user: SessionUser
  mfaEnabled: boolean
  mfaEnrollmentRequired: boolean
}

type AuthState = {
  status: 'loading' | 'guest' | 'authenticated'
  user: SessionUser | null
  mfaEnrollmentRequired: boolean
  recover: () => Promise<void>
  logout: () => Promise<void>
}

const AuthContext = createContext<AuthState | null>(null)

export function AuthProvider({ children }: { children: React.ReactNode }) {
  const [status, setStatus] = useState<AuthState['status']>('loading')
  const [user, setUser] = useState<SessionUser | null>(null)
  const [mfaEnrollmentRequired, setMFAEnrollmentRequired] = useState(false)

  const recover = useCallback(async () => {
    try {
      const session = await apiFetch<SessionResponse>('/api/v1/auth/session')
      setUser(session.user)
      setMFAEnrollmentRequired(session.mfaEnrollmentRequired)
      setStatus('authenticated')
    } catch {
      setUser(null)
      setMFAEnrollmentRequired(false)
      setStatus('guest')
    }
  }, [])

  const logout = useCallback(async () => {
    try {
      await postJSON('/api/v1/auth/logout')
    } finally {
      setUser(null)
      setMFAEnrollmentRequired(false)
      setStatus('guest')
    }
  }, [])

  useEffect(() => {
	const timer = window.setTimeout(() => void recover(), 0)
	return () => window.clearTimeout(timer)
  }, [recover])

  const value = useMemo(() => ({ status, user, mfaEnrollmentRequired, recover, logout }), [status, user, mfaEnrollmentRequired, recover, logout])
  return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>
}

export function useAuth() {
  const value = useContext(AuthContext)
  if (!value) throw new Error('useAuth must be used inside AuthProvider')
  return value
}
