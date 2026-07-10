'use client'

import { useState } from 'react'
import { useRouter } from 'next/navigation'

const apiBase = process.env.NEXT_PUBLIC_API_BASE_URL ?? 'http://localhost:8080'

export default function RegisterPage() {
  const router = useRouter()
  const [email, setEmail] = useState('')
  const [password, setPassword] = useState('')
  const [error, setError] = useState('')
  const [message, setMessage] = useState('')
  const [loading, setLoading] = useState(false)

  async function handleSubmit(event: React.FormEvent<HTMLFormElement>) {
    event.preventDefault()
    setError('')
    setMessage('')
    setLoading(true)

    try {
      const response = await fetch(`${apiBase}/api/v1/auth/register`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify({ email, password }),
      })

      if (!response.ok) {
        const body = await response.json().catch(() => null)
        setError((body && body.message) || 'Registration failed')
        return
      }

      setMessage('Registration successful. Redirecting to login...')
      setTimeout(() => router.push('/auth/login'), 1000)
    } catch (err) {
      setError('Unable to reach the API. Check your backend.')
    } finally {
      setLoading(false)
    }
  }

  return (
    <main className="container auth-container">
      <div className="form-card">
        <h1>Create account</h1>
        <p>Start your Skill Arena journey with a secure wallet and leaderboard profile.</p>

        <form onSubmit={handleSubmit} className="form-grid">
          <label>
            Email
            <input
              type="email"
              value={email}
              onChange={(event) => setEmail(event.target.value)}
              required
              placeholder="you@example.com"
            />
          </label>
          <label>
            Password
            <input
              type="password"
              value={password}
              onChange={(event) => setPassword(event.target.value)}
              required
              minLength={12}
              placeholder="Choose a strong password"
            />
          </label>

          <button type="submit" className="button" disabled={loading}>
            {loading ? 'Creating account...' : 'Register'}
          </button>
        </form>

        {error ? <p className="form-error">{error}</p> : null}
        {message ? <p className="form-success">{message}</p> : null}
      </div>
    </main>
  )
}
