'use client'

import { useState } from 'react'
import { useRouter } from 'next/navigation'

const apiBase = process.env.NEXT_PUBLIC_API_BASE_URL ?? 'http://localhost:8080'

export default function LoginPage() {
  const router = useRouter()
  const [email, setEmail] = useState('')
  const [password, setPassword] = useState('')
  const [error, setError] = useState('')
  const [loading, setLoading] = useState(false)

  async function handleSubmit(event: React.FormEvent<HTMLFormElement>) {
    event.preventDefault()
    setError('')
    setLoading(true)

    try {
      const response = await fetch(`${apiBase}/api/v1/auth/login`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify({ email, password }),
      })

      if (!response.ok) {
        setError('Invalid email or password')
        return
      }

      const body = await response.json()
      window.localStorage.setItem('skill-arena-token', body.token)
      window.localStorage.setItem('skill-arena-refresh-token', body.refreshToken)
      router.push('/dashboard')
    } catch (err) {
      setError('Unable to reach the API. Check your backend.')
    } finally {
      setLoading(false)
    }
  }

  return (
    <main className="container auth-container">
      <div className="form-card">
        <h1>Login</h1>
        <p>Access your Skill Arena wallet, profile, and games.</p>

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
              placeholder="Enter your password"
            />
          </label>

          <button type="submit" className="button" disabled={loading}>
            {loading ? 'Signing in...' : 'Login'}
          </button>
        </form>

        {error ? <p className="form-error">{error}</p> : null}
      </div>
    </main>
  )
}
