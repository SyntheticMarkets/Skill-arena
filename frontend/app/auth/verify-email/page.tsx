'use client'

import Link from 'next/link'
import { CheckCircle2, CircleAlert, LoaderCircle } from 'lucide-react'
import { useSearchParams } from 'next/navigation'
import { Suspense, useEffect, useState } from 'react'
import { ApiError, postJSON } from '../../lib/api'
import { AuthFrame, FormMessage } from '../auth-frame'

function VerifyContent() {
  const token = useSearchParams().get('token') || ''
  const [state, setState] = useState<'verifying' | 'verified' | 'failed'>(token ? 'verifying' : 'failed')
  const [message, setMessage] = useState(token ? 'Verifying your secure link...' : 'This verification link is incomplete.')

  useEffect(() => {
    if (!token) return
    void postJSON('/api/v1/auth/verify-email', { token })
      .then(() => { setState('verified'); setMessage('Your email is verified. Your competitor identity is ready.') })
      .catch((caught) => { setState('failed'); setMessage(caught instanceof ApiError ? caught.message : 'The verification link could not be validated.') })
  }, [token])

  return (
    <AuthFrame eyebrow="Identity verification" title={state === 'verified' ? 'Identity confirmed.' : state === 'failed' ? 'Link not accepted.' : 'Verifying your identity.'} description="Verification protects progression, results, and future account recovery.">
      <div className={`auth-status-visual ${state}`}>
        {state === 'verified' ? <CheckCircle2 /> : state === 'failed' ? <CircleAlert /> : <LoaderCircle className="spin" />}
        <strong>{message}</strong>
      </div>
      {state === 'failed' ? <FormMessage type="info">Verification links expire and can only be used once. Request a new link from the login screen.</FormMessage> : null}
      {state !== 'verifying' ? <Link className="entry-button auth-submit" href="/auth/login">Continue to login</Link> : null}
    </AuthFrame>
  )
}

export default function VerifyEmailPage() {
  return <Suspense fallback={<main className="boot-experience"><strong>SKILL ARENA</strong></main>}><VerifyContent /></Suspense>
}
