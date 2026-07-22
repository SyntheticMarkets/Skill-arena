'use client'

import { Check, Copy, ShieldCheck } from 'lucide-react'
import { useRouter } from 'next/navigation'
import { QRCodeSVG } from 'qrcode.react'
import { useEffect, useState } from 'react'
import { useAuth } from '../../../auth-context'
import { ApiError, postJSON } from '../../../lib/api'
import { AuthFrame, FormMessage } from '../../auth-frame'

type Setup = { secret: string; otpauthUrl: string }

export default function MFASetupPage() {
  const router = useRouter()
  const { recover } = useAuth()
  const [setup, setSetup] = useState<Setup | null>(null)
  const [code, setCode] = useState('')
  const [recoveryCodes, setRecoveryCodes] = useState<string[]>([])
  const [error, setError] = useState('')
  const [loading, setLoading] = useState(true)
  const [copied, setCopied] = useState(false)

  useEffect(() => {
    void postJSON<Setup>('/api/v1/auth/mfa/setup').then(setSetup).catch((caught) => setError(caught instanceof ApiError ? caught.message : 'MFA setup could not begin.')).finally(() => setLoading(false))
  }, [])

  async function confirm(event: React.FormEvent<HTMLFormElement>) {
    event.preventDefault(); setError(''); setLoading(true)
    try {
      const result = await postJSON<{ recoveryCodes: string[] }>('/api/v1/auth/mfa/confirm', { code })
      setRecoveryCodes(result.recoveryCodes)
      await recover()
    } catch (caught) { setError(caught instanceof ApiError ? caught.message : 'The authenticator code could not be verified.') }
    finally { setLoading(false) }
  }

  async function copyCodes() {
    await navigator.clipboard.writeText(recoveryCodes.join('\n')); setCopied(true)
  }

  return (
    <AuthFrame eyebrow="Account protection" title={recoveryCodes.length ? 'Store your recovery codes.' : 'Add an authenticator.'} description={recoveryCodes.length ? 'Each code works once. Keep them somewhere private and offline.' : 'Privileged accounts must complete this step before accessing Skill Arena.'}>
      {loading && !setup ? <div className="auth-status-visual"><span className="spinner" /><strong>Creating protected setup...</strong></div> : null}
      {error ? <FormMessage type="error">{error}</FormMessage> : null}
      {setup && recoveryCodes.length === 0 ? <div className="mfa-setup"><div className="qr-frame"><QRCodeSVG value={setup.otpauthUrl} size={176} level="M" /></div><div><p>Scan with your authenticator app, then enter the six-digit code.</p><code>{setup.secret}</code></div><form className="auth-form" onSubmit={confirm}><label htmlFor="setup-code">Authenticator code</label><input id="setup-code" className="code-input" inputMode="numeric" autoComplete="one-time-code" maxLength={6} value={code} onChange={(event) => setCode(event.target.value.replace(/\D/g, ''))} required /><button className="entry-button auth-submit" disabled={loading || code.length !== 6}><ShieldCheck />{loading ? 'Verifying...' : 'Enable MFA'}</button></form></div> : null}
      {recoveryCodes.length ? <div className="recovery-codes"><div className="code-grid">{recoveryCodes.map((item) => <code key={item}>{item}</code>)}</div><button type="button" className="secondary-button" onClick={() => void copyCodes()}>{copied ? <Check /> : <Copy />}{copied ? 'Copied' : 'Copy codes'}</button><button type="button" className="entry-button auth-submit" onClick={() => router.push('/arena')}>I stored them securely</button></div> : null}
    </AuthFrame>
  )
}
