'use client'

import Link from 'next/link'
import { Eye, EyeOff, LogIn } from 'lucide-react'
import { useRouter, useSearchParams } from 'next/navigation'
import { Suspense, useState } from 'react'
import { useAuth } from '../../auth-context'
import { ApiError, postJSON } from '../../lib/api'
import { AuthFrame, FormMessage } from '../auth-frame'

type LoginResponse = {
  authenticated?: boolean
  mfaRequired?: boolean
  challengeToken?: string
  mfaEnrollmentRequired?: boolean
}

function LoginForm() {
  const router = useRouter()
  const searchParams = useSearchParams()
  const { recover } = useAuth()
  const [email, setEmail] = useState('')
  const [password, setPassword] = useState('')
  const [showPassword, setShowPassword] = useState(false)
  const [error, setError] = useState('')
  const [unverified, setUnverified] = useState(false)
  const [loading, setLoading] = useState(false)

  async function handleSubmit(event: React.FormEvent<HTMLFormElement>) {
    event.preventDefault()
    setError('')
    setUnverified(false)
    setLoading(true)
    try {
      const body = await postJSON<LoginResponse>('/api/v1/auth/login', { email, password })
      if (body.mfaRequired && body.challengeToken) {
        window.sessionStorage.setItem('skill-arena-mfa-challenge', body.challengeToken)
        router.push(`/auth/mfa?next=${encodeURIComponent(searchParams.get('next') || '/arena')}`)
        return
      }
      await recover()
      router.push(body.mfaEnrollmentRequired ? '/auth/mfa/setup' : (searchParams.get('next') || '/arena'))
    } catch (caught) {
      if (caught instanceof ApiError && caught.code === 'AUTH_EMAIL_UNVERIFIED') setUnverified(true)
      setError(caught instanceof ApiError ? caught.message : 'Skill Arena is temporarily unreachable. Please try again.')
    } finally {
      setLoading(false)
    }
  }

  return (
    <AuthFrame eyebrow="Competitor access" title="Return to the arena." description="Continue your progression from a verified, protected session.">
      <form onSubmit={handleSubmit} className="auth-form" noValidate>
        <label htmlFor="login-email">Email address</label>
        <input id="login-email" type="email" autoComplete="email" value={email} onChange={(event) => setEmail(event.target.value)} required placeholder="you@example.com" />
        <div className="label-row"><label htmlFor="login-password">Password</label><Link href="/auth/forgot-password">Forgot password?</Link></div>
        <div className="password-field">
          <input id="login-password" type={showPassword ? 'text' : 'password'} autoComplete="current-password" value={password} onChange={(event) => setPassword(event.target.value)} required placeholder="Your password" />
          <button type="button" className="icon-button" aria-label={showPassword ? 'Hide password' : 'Show password'} onClick={() => setShowPassword((value) => !value)}>{showPassword ? <EyeOff /> : <Eye />}</button>
        </div>
        {error ? <FormMessage type="error">{error}{unverified ? <> <Link href={`/auth/verification-pending?email=${encodeURIComponent(email)}`}>Resend verification</Link></> : null}</FormMessage> : null}
        <button type="submit" className="entry-button auth-submit" disabled={loading || !email || !password}><LogIn aria-hidden="true" />{loading ? 'Verifying...' : 'Enter securely'}</button>
      </form>
      <p className="auth-switch">New to Skill Arena? <Link href="/auth/register">Create your competitor identity</Link></p>
    </AuthFrame>
  )
}

export default function LoginPage() {
  return <Suspense fallback={<main className="boot-experience"><strong>SKILL ARENA</strong><span>Preparing secure entry</span></main>}><LoginForm /></Suspense>
}
