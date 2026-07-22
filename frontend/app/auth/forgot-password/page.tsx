'use client'

import Link from 'next/link'
import { Send } from 'lucide-react'
import { useState } from 'react'
import { postJSON } from '../../lib/api'
import { AuthFrame, FormMessage } from '../auth-frame'

export default function ForgotPasswordPage() {
  const [email, setEmail] = useState('')
  const [sent, setSent] = useState(false)
  const [loading, setLoading] = useState(false)

  async function submit(event: React.FormEvent<HTMLFormElement>) {
    event.preventDefault(); setLoading(true)
    try { await postJSON('/api/v1/auth/password-reset/request', { email }) } finally { setSent(true); setLoading(false) }
  }

  return (
    <AuthFrame eyebrow="Account recovery" title="Reset your password." description="We never reveal whether an email is registered. If the account exists, a one-time link will arrive shortly.">
      {sent ? <FormMessage type="success">Check your inbox. The recovery link expires after 30 minutes and signs out existing sessions when used.</FormMessage> : (
        <form className="auth-form" onSubmit={submit}><label htmlFor="recovery-email">Email address</label><input id="recovery-email" type="email" autoComplete="email" value={email} onChange={(event) => setEmail(event.target.value)} required /><button className="entry-button auth-submit" disabled={loading || !email}><Send />{loading ? 'Requesting...' : 'Send recovery link'}</button></form>
      )}
      <p className="auth-switch"><Link href="/auth/login">Back to secure login</Link></p>
    </AuthFrame>
  )
}
