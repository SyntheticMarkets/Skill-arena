'use client'

import { Check, Edit3, LockKeyhole, Save, ShieldCheck } from 'lucide-react'
import { FormEvent, useEffect, useState } from 'react'
import { useAuth } from '../auth-context'
import { useHub } from '../hub-context'
import { postJSON } from '../lib/api'
import { avatarOptions, PlayerAvatar } from '../player-avatar'

type FormState = {
  username: string
  displayName: string
  avatarUrl: string
  country: string
  language: string
}

export default function ProfilePage() {
  const { user } = useAuth()
  const { data, status, error, reload } = useHub()
  const [editing, setEditing] = useState(false)
  const [form, setForm] = useState<FormState>({ username: '', displayName: '', avatarUrl: '', country: '', language: 'en' })
  const [saving, setSaving] = useState(false)
  const [notice, setNotice] = useState('')

  useEffect(() => {
    if (!data) return
    const timer = window.setTimeout(() => setForm({
      username: data.profile.username,
      displayName: data.profile.displayName,
      avatarUrl: data.profile.avatarUrl || '',
      country: data.profile.country,
      language: data.profile.language,
    }), 0)
    return () => window.clearTimeout(timer)
  }, [data])

  async function save(event: FormEvent) {
    event.preventDefault()
    setSaving(true)
    setNotice('')
    try {
      await postJSON('/api/v1/profile', form)
      await reload()
      setEditing(false)
      setNotice('Your competitor profile was updated.')
    } catch (cause) {
      setNotice(cause instanceof Error ? cause.message : 'The profile could not be updated.')
    } finally {
      setSaving(false)
    }
  }

  if (status === 'loading' || status === 'idle') return <main className="hub-page"><div className="inline-loading">Loading verified profile...</div></main>
  if (!data) return <main className="hub-page"><div className="form-message error">{error || 'Profile is unavailable.'}</div></main>

  const played = data.progression.matchesPlayed
  const winRate = played ? Math.round((data.progression.wins / played) * 100) : 0
  const initials = data.profile.displayName.split(/\s+/).map((part) => part[0]).join('').slice(0, 2).toUpperCase()

  return (
    <main className="hub-page">
      <section className="profile-identity">
        <div className="profile-avatar" aria-label={`${data.profile.displayName} avatar`}>
          <PlayerAvatar avatarKey={data.profile.avatarUrl} fallback={initials || 'SA'} />
        </div>
        <div>
          <span className="eyebrow">Competitor identity</span>
          <h1>{data.profile.displayName}</h1>
          <p>@{data.profile.username} / {data.profile.country} / {data.profile.language.toUpperCase()}</p>
        </div>
        <button className="button secondary" type="button" onClick={() => setEditing((value) => !value)}><Edit3 />{editing ? 'Cancel editing' : 'Edit profile'}</button>
      </section>

      {notice ? <p className="form-message" role="status">{notice}</p> : null}

      {editing ? (
        <form className="profile-form" onSubmit={save}>
          <div><label htmlFor="profile-username">Username</label><input id="profile-username" value={form.username} minLength={3} maxLength={30} pattern="[A-Za-z0-9_]+" onChange={(event) => setForm({ ...form, username: event.target.value })} required /></div>
          <div><label htmlFor="profile-display">Display name</label><input id="profile-display" value={form.displayName} minLength={2} maxLength={60} onChange={(event) => setForm({ ...form, displayName: event.target.value })} required /></div>
          <div><label htmlFor="profile-country">Country code</label><input id="profile-country" value={form.country} minLength={2} maxLength={2} pattern="[A-Za-z]{2}" onChange={(event) => setForm({ ...form, country: event.target.value.toUpperCase() })} required /></div>
          <div><label htmlFor="profile-language">Language</label><select id="profile-language" value={form.language} onChange={(event) => setForm({ ...form, language: event.target.value })}><option value="en">English</option></select></div>
          <fieldset className="avatar-picker profile-form-wide">
            <legend>Avatar</legend>
            {avatarOptions.map((option) => (
              <label key={option.key}>
                <input type="radio" name="avatar" value={option.key} checked={form.avatarUrl === option.key} onChange={(event) => setForm({ ...form, avatarUrl: event.target.value })} />
                <span><PlayerAvatar avatarKey={option.key} fallback="" /><small>{option.label}</small></span>
              </label>
            ))}
          </fieldset>
          <button className="button profile-form-wide" type="submit" disabled={saving}><Save />{saving ? 'Saving...' : 'Save profile'}</button>
        </form>
      ) : null}

      <section className="profile-stats" aria-label="Player statistics">
        <article><span>Level</span><strong>{data.progression.level}</strong><small>{data.progression.xp.toLocaleString()} XP</small></article>
        <article><span>Rank</span><strong>{data.progression.leagueTier}</strong><small>{data.progression.eloRating.toLocaleString()} rating</small></article>
        <article><span>Record</span><strong>{data.progression.wins}W / {data.progression.losses}L</strong><small>{winRate}% win rate</small></article>
        <article><span>Trust Score</span><strong>{data.progression.trustScore.toFixed(0)}</strong><small>{data.progression.trustTier}</small></article>
      </section>

      <section className="profile-status-grid">
        <article>
          <div className="status-title"><ShieldCheck /><div><span className="eyebrow">Verification</span><h2>Identity status</h2></div></div>
          <dl>
            <div><dt>Email</dt><dd className={data.eligibility.emailVerified ? 'verified' : ''}>{data.eligibility.emailVerified ? <><Check />Verified</> : 'Not verified'}</dd></div>
            <div><dt>Identity</dt><dd>{data.wallet.verificationStatus}</dd></div>
            <div><dt>Account</dt><dd>{data.wallet.accountStatus}</dd></div>
            <div><dt>Live competition</dt><dd>{data.eligibility.liveEligible ? 'Eligible' : 'Locked'}</dd></div>
          </dl>
        </article>
        <article>
          <div className="status-title"><LockKeyhole /><div><span className="eyebrow">Security</span><h2>Account protection</h2></div></div>
          <dl>
            <div><dt>Multi-factor authentication</dt><dd className={data.eligibility.mfaEnabled ? 'verified' : ''}>{data.eligibility.mfaEnabled ? 'Enabled' : 'Optional'}</dd></div>
            <div><dt>Protected email</dt><dd>{user?.email || 'Unavailable'}</dd></div>
            <div><dt>Profile completeness</dt><dd>{data.eligibility.profileComplete ? 'Complete' : 'Action required'}</dd></div>
          </dl>
          {!data.eligibility.mfaEnabled ? <a className="button secondary" href="/auth/mfa/setup">Enable MFA</a> : null}
        </article>
      </section>
    </main>
  )
}
