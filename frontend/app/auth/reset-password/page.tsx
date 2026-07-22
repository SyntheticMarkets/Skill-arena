'use client'

import Link from 'next/link'
import { KeyRound } from 'lucide-react'
import { useSearchParams } from 'next/navigation'
import { Suspense, useState } from 'react'
import { ApiError, postJSON } from '../../lib/api'
import { AuthFrame, FormMessage } from '../auth-frame'

function ResetContent() {
  const token = useSearchParams().get('token') || ''
  const [password, setPassword] = useState('')
  const [confirmPassword, setConfirmPassword] = useState('')
  const [error, setError] = useState('')
  const [complete, setComplete] = useState(false)
  const [loading, setLoading] = useState(false)

  async function submit(event: React.FormEvent<HTMLFormElement>) {
    event.preventDefault(); setError(''); setLoading(true)
    try { await postJSON('/api/v1/auth/password-reset/confirm', { token, password, confirmPassword }); setComplete(true) }
    catch (caught) { setError(caught instanceof ApiError ? caught.message : 'The password could not be changed.') }
    finally { setLoading(false) }
  }

  return (
    <AuthFrame eyebrow="Secure recovery" title={complete ? 'Password updated.' : 'Choose a new password.'} description="A successful reset revokes every existing session for this account.">
      {complete ? <><FormMessage type="success">Your new password is active and all previous sessions have been signed out.</FormMessage><Link className="entry-button auth-submit" href="/auth/login">Return to login</Link></> : (
        <form className="auth-form" onSubmit={submit}>
          <label htmlFor="new-password">New password</label><input id="new-password" type="password" autoComplete="new-password" minLength={12} value={password} onChange={(event) => setPassword(event.target.value)} required />
          <label htmlFor="confirm-password">Confirm new password</label><input id="confirm-password" type="password" autoComplete="new-password" minLength={12} value={confirmPassword} onChange={(event) => setConfirmPassword(event.target.value)} required />
          {error ? <FormMessage type="error">{error}</FormMessage> : null}
          <button className="entry-button auth-submit" disabled={loading || !token || password.length < 12 || password !== confirmPassword}><KeyRound />{loading ? 'Securing account...' : 'Set new password'}</button>
        </form>
      )}
    </AuthFrame>
  )
}

export default function ResetPasswordPage() {
  return <Suspense fallback={<main className="boot-experience"><strong>SKILL ARENA</strong></main>}><ResetContent /></Suspense>
}
