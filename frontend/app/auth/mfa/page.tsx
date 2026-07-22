'use client'

import { KeyRound, ShieldCheck } from 'lucide-react'
import { useRouter, useSearchParams } from 'next/navigation'
import { Suspense, useState } from 'react'
import { useAuth } from '../../auth-context'
import { ApiError, postJSON } from '../../lib/api'
import { AuthFrame, FormMessage } from '../auth-frame'

function MFAContent() {
  const router = useRouter()
  const next = useSearchParams().get('next') || '/arena'
  const { recover } = useAuth()
  const [mode, setMode] = useState<'totp' | 'recovery'>('totp')
  const [value, setValue] = useState('')
  const [error, setError] = useState('')
  const [loading, setLoading] = useState(false)

  async function submit(event: React.FormEvent<HTMLFormElement>) {
    event.preventDefault(); setError(''); setLoading(true)
    const challengeToken = window.sessionStorage.getItem('skill-arena-mfa-challenge') || ''
    try {
      await postJSON('/api/v1/auth/mfa/challenge', { challengeToken, ...(mode === 'totp' ? { code: value } : { recoveryCode: value }) })
      window.sessionStorage.removeItem('skill-arena-mfa-challenge')
      await recover()
      router.push(next)
    } catch (caught) { setError(caught instanceof ApiError ? caught.message : 'MFA verification is temporarily unavailable.') }
    finally { setLoading(false) }
  }

  return (
    <AuthFrame eyebrow="Second factor" title="Confirm it is really you." description="Use your authenticator code or one unused recovery code.">
      <div className="segmented-control" role="group" aria-label="MFA verification method"><button type="button" aria-pressed={mode === 'totp'} onClick={() => { setMode('totp'); setValue('') }}>Authenticator</button><button type="button" aria-pressed={mode === 'recovery'} onClick={() => { setMode('recovery'); setValue('') }}>Recovery code</button></div>
      <form className="auth-form" onSubmit={submit}>
        <label htmlFor="mfa-code">{mode === 'totp' ? 'Six-digit code' : 'Recovery code'}</label>
        <input id="mfa-code" className="code-input" inputMode={mode === 'totp' ? 'numeric' : 'text'} autoComplete="one-time-code" maxLength={mode === 'totp' ? 6 : 16} value={value} onChange={(event) => setValue(event.target.value.replace(mode === 'totp' ? /\D/g : /\s/g, ''))} autoFocus required />
        {error ? <FormMessage type="error">{error}</FormMessage> : null}
        <button className="entry-button auth-submit" disabled={loading || value.length < 6}>{mode === 'totp' ? <ShieldCheck /> : <KeyRound />}{loading ? 'Verifying...' : 'Verify and enter'}</button>
      </form>
    </AuthFrame>
  )
}

export default function MFAPage() {
  return <Suspense fallback={<main className="boot-experience"><strong>SKILL ARENA</strong></main>}><MFAContent /></Suspense>
}
