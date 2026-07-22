'use client'

import Link from 'next/link'
import { MailCheck, RefreshCw } from 'lucide-react'
import { useSearchParams } from 'next/navigation'
import { Suspense, useState } from 'react'
import { ApiError, postJSON } from '../../lib/api'
import { AuthFrame, FormMessage } from '../auth-frame'

function PendingContent() {
  const email = useSearchParams().get('email') || ''
  const [message, setMessage] = useState('')
  const [error, setError] = useState('')
  const [loading, setLoading] = useState(false)

  async function resend() {
    setLoading(true); setError(''); setMessage('')
    try {
      await postJSON('/api/v1/auth/resend-verification', { email })
      setMessage('A fresh verification link has been queued. Check your inbox and spam folder.')
    } catch (caught) {
      setError(caught instanceof ApiError ? caught.message : 'The verification email could not be queued.')
    } finally { setLoading(false) }
  }

  return (
    <AuthFrame eyebrow="One secure step" title="Check your inbox." description={email ? `We sent a verification link to ${email}.` : 'Open the verification link sent to your email address.'}>
      <div className="auth-status-visual"><MailCheck aria-hidden="true" /><strong>Verification pending</strong><span>The link expires after 24 hours and can only be used once.</span></div>
      {message ? <FormMessage type="success">{message}</FormMessage> : null}
      {error ? <FormMessage type="error">{error}</FormMessage> : null}
      <button type="button" className="entry-button auth-submit" onClick={() => void resend()} disabled={loading || !email}><RefreshCw aria-hidden="true" />{loading ? 'Sending...' : 'Resend verification'}</button>
      <p className="auth-switch"><Link href="/auth/login">Back to secure login</Link></p>
    </AuthFrame>
  )
}

export default function VerificationPendingPage() {
  return <Suspense fallback={<main className="boot-experience"><strong>SKILL ARENA</strong></main>}><PendingContent /></Suspense>
}
