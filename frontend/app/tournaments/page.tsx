'use client'

import { CalendarDays, LockKeyhole, Trophy } from 'lucide-react'
import { useHub } from '../hub-context'

export default function TournamentsPage() {
  const { data, status, error } = useHub()
  if (status === 'loading' || status === 'idle') return <main className="hub-page"><div className="inline-loading">Loading tournament calendar...</div></main>
  if (!data) return <main className="hub-page"><div className="form-message error">{error || 'Tournament state is unavailable.'}</div></main>

  return (
    <main className="hub-page">
      <section className="subpage-heading">
        <div><span className="eyebrow">Tournament centre</span><h1>Structured competition.</h1><p>Review server-recorded events, schedules, and your current eligibility for organized competition.</p></div>
        <Trophy aria-hidden="true" />
      </section>
      {data.tournaments.length === 0 ? (
        <section className="empty-state"><CalendarDays /><h2>No tournament is accepting players.</h2><p>The calendar remains empty until an authorized tournament record is active.</p></section>
      ) : (
        <section className="tournament-directory">
          {data.tournaments.map((tournament) => (
            <article key={tournament.id}>
              <span className="tournament-date"><strong>{new Date(tournament.startsAt).getDate()}</strong><small>{new Date(tournament.startsAt).toLocaleString(undefined, { month: 'short' })}</small></span>
              <div><span>{tournament.status}</span><h2>{tournament.name}</h2><p>{new Date(tournament.startsAt).toLocaleString()}</p></div>
              <span className={tournament.eligible ? 'eligible' : 'locked'}>{tournament.eligible ? 'Eligible' : <><LockKeyhole />{tournament.ineligibleReason}</>}</span>
            </article>
          ))}
        </section>
      )}
    </main>
  )
}
