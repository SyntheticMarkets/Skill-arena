'use client'

import Link from 'next/link'
import { Check, Eye, EyeOff, UserPlus } from 'lucide-react'
import { useRouter } from 'next/navigation'
import { useMemo, useState } from 'react'
import { ApiError, postJSON } from '../../lib/api'
import { AuthFrame, FormMessage } from '../auth-frame'

const countries = [
  ['ZA', 'South Africa'], ['BW', 'Botswana'], ['GH', 'Ghana'], ['KE', 'Kenya'],
  ['NA', 'Namibia'], ['NG', 'Nigeria'], ['GB', 'United Kingdom'], ['US', 'United States'],
]

export default function RegisterPage() {
  const router = useRouter()
  const [email, setEmail] = useState('')
  const [password, setPassword] = useState('')
  const [country, setCountry] = useState('ZA')
  const [dateOfBirth, setDateOfBirth] = useState('')
  const [acceptTerms, setAcceptTerms] = useState(false)
  const [acceptFairPlay, setAcceptFairPlay] = useState(false)
  const [showPassword, setShowPassword] = useState(false)
  const [error, setError] = useState('')
  const [loading, setLoading] = useState(false)

  const passwordChecks = useMemo(() => [
    ['12+ characters', password.length >= 12],
    ['Uppercase letter', /[A-Z]/.test(password)],
    ['Number', /\d/.test(password)],
    ['Symbol', /[^A-Za-z0-9]/.test(password)],
  ] as const, [password])
  const passwordReady = passwordChecks.every(([, ready]) => ready)

  async function handleSubmit(event: React.FormEvent<HTMLFormElement>) {
    event.preventDefault()
    setError('')
    setLoading(true)
    try {
      await postJSON('/api/v1/auth/register', { email, password, country, dateOfBirth, acceptTerms, acceptFairPlay })
      router.push(`/auth/verification-pending?email=${encodeURIComponent(email)}`)
    } catch (caught) {
      setError(caught instanceof ApiError ? caught.message : 'Registration is temporarily unavailable. Please try again.')
    } finally {
      setLoading(false)
    }
  }

  return (
    <AuthFrame eyebrow="Create your identity" title="Enter as a competitor." description="Your first account starts in Practice. Deposits are never required to learn the arena.">
      <form onSubmit={handleSubmit} className="auth-form" noValidate>
        <label htmlFor="register-email">Email address</label>
        <input id="register-email" type="email" autoComplete="email" value={email} onChange={(event) => setEmail(event.target.value)} required placeholder="you@example.com" />
        <div className="form-row">
          <div><label htmlFor="register-country">Country</label><select id="register-country" value={country} onChange={(event) => setCountry(event.target.value)}>{countries.map(([code, name]) => <option key={code} value={code}>{name}</option>)}</select></div>
          <div><label htmlFor="register-birth-date">Date of birth</label><input id="register-birth-date" type="date" autoComplete="bday" value={dateOfBirth} onChange={(event) => setDateOfBirth(event.target.value)} required /></div>
        </div>
        <label htmlFor="register-password">Password</label>
        <div className="password-field">
          <input id="register-password" type={showPassword ? 'text' : 'password'} autoComplete="new-password" value={password} onChange={(event) => setPassword(event.target.value)} required placeholder="Create a strong password" aria-describedby="password-rules" />
          <button type="button" className="icon-button" aria-label={showPassword ? 'Hide password' : 'Show password'} onClick={() => setShowPassword((value) => !value)}>{showPassword ? <EyeOff /> : <Eye />}</button>
        </div>
        <ul id="password-rules" className="password-rules" aria-label="Password requirements">{passwordChecks.map(([label, ready]) => <li key={label} className={ready ? 'ready' : ''}><Check aria-hidden="true" />{label}</li>)}</ul>
        <label className="check-row"><input type="checkbox" checked={acceptTerms} onChange={(event) => setAcceptTerms(event.target.checked)} /><span>I am at least 18 and accept the Skill Arena Terms and Privacy Notice.</span></label>
        <label className="check-row"><input type="checkbox" checked={acceptFairPlay} onChange={(event) => setAcceptFairPlay(event.target.checked)} /><span>I accept the Fair Play rules and understand that competitive actions are verified.</span></label>
        {error ? <FormMessage type="error">{error}</FormMessage> : null}
        <button type="submit" className="entry-button auth-submit" disabled={loading || !email || !dateOfBirth || !passwordReady || !acceptTerms || !acceptFairPlay}><UserPlus aria-hidden="true" />{loading ? 'Creating identity...' : 'Create identity'}</button>
      </form>
      <p className="auth-switch">Already registered? <Link href="/auth/login">Return to the arena</Link></p>
    </AuthFrame>
  )
}
