'use client'

import Link from 'next/link'
import { KeyRound, Laptop, LockKeyhole, ShieldCheck, Smartphone } from 'lucide-react'
import { useEffect, useState } from 'react'
import { useHub } from '../hub-context'
import { apiFetch, postJSON } from '../lib/api'

type Session = {
  id: string
  userAgent?: string
  ipAddress?: string
  createdAt: string
  expiresAt: string
  current: boolean
  mfaVerified: boolean
}

type Device = {
  id: string
  deviceName?: string
  os?: string
  browser?: string
  lastSeen: string
  revokedAt?: string
}

export default function SettingsPage() {
  const { data } = useHub()
  const [sessions, setSessions] = useState<Session[]>([])
  const [devices, setDevices] = useState<Device[]>([])
  const [status, setStatus] = useState<'loading' | 'ready' | 'error'>('loading')

  async function load() {
    setStatus('loading')
    try {
      const [sessionResponse, deviceResponse] = await Promise.all([
        apiFetch<{ sessions: Session[] }>('/api/v1/auth/sessions'),
        apiFetch<{ devices: Device[] }>('/api/v1/auth/devices'),
      ])
      setSessions(sessionResponse.sessions)
      setDevices(deviceResponse.devices.filter((device) => !device.revokedAt))
      setStatus('ready')
    } catch {
      setStatus('error')
    }
  }

  useEffect(() => {
    const timer = window.setTimeout(() => void load(), 0)
    return () => window.clearTimeout(timer)
  }, [])

  async function revokeSession(sessionId: string) {
    await postJSON('/api/v1/auth/sessions/revoke', { sessionId })
    await load()
  }

  async function revokeDevice(deviceId: string) {
    await postJSON('/api/v1/auth/devices/revoke', { deviceId })
    await load()
  }

  return (
    <main className="hub-page">
      <section className="subpage-heading">
        <div><span className="eyebrow">Account settings</span><h1>Control your access.</h1><p>Review MFA, active sessions, and recognized devices from the current server security state.</p></div>
        <ShieldCheck aria-hidden="true" />
      </section>

      <section className="security-summary">
        <article><LockKeyhole /><span><strong>Multi-factor authentication</strong><small>{data?.eligibility.mfaEnabled ? 'Enabled' : 'Not enabled'}</small></span>{!data?.eligibility.mfaEnabled ? <Link href="/auth/mfa/setup">Set up MFA</Link> : <span className="verified">Protected</span>}</article>
        <article><KeyRound /><span><strong>Password and recovery</strong><small>Reset uses a signed, expiring, one-time email link.</small></span><Link href="/auth/forgot-password">Reset password</Link></article>
      </section>

      {status === 'loading' ? <div className="inline-loading">Loading security state...</div> : null}
      {status === 'error' ? <div className="form-message error">Security state could not be loaded.<button type="button" onClick={() => void load()}>Retry</button></div> : null}

      <section className="settings-columns">
        <section className="hub-section" aria-labelledby="sessions-title">
          <div className="hub-section-heading"><div><span className="eyebrow">Sessions</span><h2 id="sessions-title">Where you are signed in.</h2></div><Laptop /></div>
          <div className="security-list">
            {sessions.map((session) => (
              <article key={session.id}>
                <span><strong>{session.current ? 'Current session' : 'Active session'}</strong><small>{session.ipAddress || 'IP unavailable'} / expires {new Date(session.expiresAt).toLocaleString()}</small></span>
                {session.current ? <span className="verified">Current</span> : <button type="button" onClick={() => void revokeSession(session.id)}>Revoke</button>}
              </article>
            ))}
          </div>
        </section>

        <section className="hub-section" aria-labelledby="devices-title">
          <div className="hub-section-heading"><div><span className="eyebrow">Devices</span><h2 id="devices-title">Recognized access points.</h2></div><Smartphone /></div>
          <div className="security-list">
            {devices.map((device) => (
              <article key={device.id}>
                <span><strong>{device.deviceName || device.browser || 'Browser device'}</strong><small>{device.os || 'Unknown OS'} / last seen {new Date(device.lastSeen).toLocaleString()}</small></span>
                <button type="button" onClick={() => void revokeDevice(device.id)}>Revoke</button>
              </article>
            ))}
            {status === 'ready' && devices.length === 0 ? <p className="hub-empty">No active device record is available.</p> : null}
          </div>
        </section>
      </section>
    </main>
  )
}
