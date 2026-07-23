'use client'

import Link from 'next/link'
import { Check, Goal, LockKeyhole, ShieldAlert } from 'lucide-react'
import { useHub } from '../hub-context'

export default function ChallengesPage() {
  const { data, status, error } = useHub()

  if (status === 'loading' || status === 'idle') return <main className="hub-page"><div className="inline-loading">Loading challenge eligibility...</div></main>
  if (!data) return <main className="hub-page"><div className="form-message error">{error || 'Challenges are unavailable.'}</div></main>

  return (
    <main className="hub-page">
      <section className="subpage-heading">
        <div><span className="eyebrow">Challenge paths</span><h1>Know what you can enter.</h1><p>Availability comes from account, progression, Trust, and platform state. Locked paths always explain why.</p></div>
        <Goal aria-hidden="true" />
      </section>
      <section className="challenge-directory">
        {data.challenges.map((challenge) => {
          const available = challenge.status === 'available'
          return (
            <article key={challenge.id}>
              <span className={`challenge-symbol ${available ? 'available' : ''}`}>{available ? <Check /> : challenge.status === 'locked' ? <LockKeyhole /> : <ShieldAlert />}</span>
              <div><span>{challenge.type}</span><h2>{challenge.title}</h2><p>{available ? 'Your current account state permits this path.' : challenge.reason}</p></div>
              {available && challenge.actionUrl ? <Link className="button secondary" href={challenge.actionUrl}>Open</Link> : <strong>{challenge.status}</strong>}
            </article>
          )
        })}
      </section>
    </main>
  )
}
