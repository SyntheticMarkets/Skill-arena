'use client'

import Link from 'next/link'
import {
  ArrowRight,
  Bell,
  Check,
  ChevronRight,
  CircleDollarSign,
  Clock3,
  LockKeyhole,
  ShieldCheck,
  Sparkles,
  Target,
  Trophy,
} from 'lucide-react'
import { useHub } from '../hub-context'

function welcomePeriod() {
  const hour = new Date().getHours()
  if (hour < 12) return 'Good morning'
  if (hour < 18) return 'Good afternoon'
  return 'Good evening'
}

function money(value: number, currency: string) {
  return new Intl.NumberFormat(undefined, { style: 'currency', currency }).format(value)
}

export default function DashboardPage() {
  const { data, status, error, reload } = useHub()

  if (status === 'loading' || status === 'idle') {
    return (
      <main className="hub-page" aria-busy="true" aria-live="polite">
        <section className="hub-loading">
          <span className="loading-pulse" />
          <strong>Synchronizing your Arena Hub</strong>
          <p>Identity, progression, wallet, and competition state are being verified.</p>
        </section>
      </main>
    )
  }

  if (status === 'error' || !data) {
    return (
      <main className="hub-page">
        <section className="hub-error" role="alert">
          <ShieldCheck />
          <h1>The Hub could not synchronize.</h1>
          <p>{error || 'The server did not return a player state.'}</p>
          <button className="button" type="button" onClick={() => void reload()}>Try again</button>
        </section>
      </main>
    )
  }

  const completedObjectives = data.objectives.filter((objective) => objective.complete).length
  const availableGame = data.games.find((game) => game.availability === 'available')

  return (
    <main className="hub-page">
      <section className="hub-welcome" aria-labelledby="hub-title">
        <div className="hub-welcome-copy">
          <span className="eyebrow">Player command</span>
          <h1 id="hub-title">{welcomePeriod()}, {data.profile.displayName}.</h1>
          <p>{data.recommendedAction.reason}</p>
        </div>
        <div className="hub-rank-signal" aria-label={`Level ${data.progression.level}, ${data.progression.leagueTier} league`}>
          <span>Level {data.progression.level}</span>
          <strong>{data.progression.leagueTier}</strong>
          <small>{data.progression.eloRating.toLocaleString()} rating</small>
        </div>
      </section>

      <section className="next-action-band" aria-labelledby="next-action-title">
        <div className="next-action-icon"><Sparkles aria-hidden="true" /></div>
        <div>
          <span>Recommended next action</span>
          <h2 id="next-action-title">{data.recommendedAction.label}</h2>
          <p>{data.recommendedAction.description}</p>
        </div>
        <Link className="button" href={data.recommendedAction.actionUrl}>
          {data.recommendedAction.label}<ArrowRight aria-hidden="true" />
        </Link>
      </section>

      <section className="hub-vitals" aria-label="Player overview">
        <article>
          <span><Target />Progress</span>
          <strong>{data.progression.xp.toLocaleString()} XP</strong>
          <small>Prestige {data.progression.prestige}</small>
        </article>
        <article>
          <span><Trophy />Competitive rank</span>
          <strong>{data.progression.leagueTier}</strong>
          <small>{data.progression.eloRating.toLocaleString()} rating</small>
        </article>
        <article>
          <span><CircleDollarSign />Available wallet</span>
          <strong>{money(data.wallet.availableBalance, data.wallet.currency)}</strong>
          <small>{data.wallet.pendingWithdrawals ? `${money(data.wallet.pendingWithdrawals, data.wallet.currency)} pending withdrawal` : 'No pending withdrawals'}</small>
        </article>
        <article>
          <span><Bell />Notifications</span>
          <strong>{data.notifications.unread}</strong>
          <small>{data.notifications.unread === 1 ? 'Unread update' : 'Unread updates'}</small>
        </article>
      </section>

      <div className="hub-layout">
        <section className="hub-primary-column">
          {data.continueActivity ? (
            <section className="hub-section" aria-labelledby="continue-title">
              <div className="hub-section-heading">
                <div><span className="eyebrow">Continue</span><h2 id="continue-title">Return without losing momentum.</h2></div>
              </div>
              <Link className="continue-row" href={data.continueActivity.actionUrl || '/games'}>
                <span className="continue-icon"><Clock3 /></span>
                <span><strong>{data.continueActivity.title}</strong><small>{data.continueActivity.description}</small></span>
                <ChevronRight />
              </Link>
            </section>
          ) : null}

          <section className="hub-section" aria-labelledby="games-title">
            <div className="hub-section-heading">
              <div><span className="eyebrow">Arena directory</span><h2 id="games-title">Choose your discipline.</h2></div>
              <Link href="/games">View games<ArrowRight /></Link>
            </div>
            {availableGame ? (
              <article className="featured-game-row">
                <div className="game-monogram" aria-hidden="true">MA</div>
                <div>
                  <span>{availableGame.category} / v{availableGame.version}</span>
                  <h3>{availableGame.name}</h3>
                  <p>{availableGame.description}</p>
                  <div className="capability-list">
                    {availableGame.capabilities.practice ? <span>Practice</span> : null}
                    {availableGame.capabilities.pvp ? <span>PvP</span> : null}
                    {availableGame.capabilities.replay ? <span>Replay</span> : null}
                    {availableGame.capabilities.tournament ? <span>Tournament</span> : null}
                  </div>
                </div>
                <Link className="button secondary" href="/games">Enter Arena</Link>
              </article>
            ) : <p className="hub-empty">No game module is enabled by the current platform configuration.</p>}
          </section>

          <section className="hub-section" aria-labelledby="challenges-title">
            <div className="hub-section-heading">
              <div><span className="eyebrow">Competition paths</span><h2 id="challenges-title">Choose what you are ready for.</h2></div>
              <Link href="/challenges">All challenges<ArrowRight /></Link>
            </div>
            <div className="challenge-rows">
              {data.challenges.map((challenge) => {
                const available = challenge.status === 'available'
                return (
                  <article key={challenge.id}>
                    <span className={`status-mark ${available ? 'available' : ''}`}>{available ? <Check /> : <LockKeyhole />}</span>
                    <span><strong>{challenge.title}</strong><small>{available ? 'Available now' : challenge.reason}</small></span>
                    {available && challenge.actionUrl ? <Link href={challenge.actionUrl} aria-label={`Open ${challenge.title}`}><ChevronRight /></Link> : <span className="challenge-state">{challenge.status}</span>}
                  </article>
                )
              })}
            </div>
          </section>

          <section className="hub-section" aria-labelledby="activity-title">
            <div className="hub-section-heading">
              <div><span className="eyebrow">Recent activity</span><h2 id="activity-title">Your verified trail.</h2></div>
            </div>
            {data.recentActivity.length ? (
              <ol className="activity-timeline">
                {data.recentActivity.map((activity) => (
                  <li key={activity.id}>
                    <span />
                    <div><strong>{activity.title}</strong><p>{activity.description}</p><time dateTime={activity.occurredAt}>{new Date(activity.occurredAt).toLocaleString()}</time></div>
                    {activity.actionUrl ? <Link href={activity.actionUrl} aria-label={`Open ${activity.title}`}><ChevronRight /></Link> : null}
                  </li>
                ))}
              </ol>
            ) : <p className="hub-empty">Your activity trail begins with your first Practice session.</p>}
          </section>
        </section>

        <aside className="hub-side-column">
          <section className="objective-panel" aria-labelledby="objectives-title">
            <div className="objective-score"><strong>{completedObjectives}/{data.objectives.length}</strong><span>Today</span></div>
            <div><span className="eyebrow">Daily objectives</span><h2 id="objectives-title">Make progress that lasts.</h2></div>
            <div className="objective-list">
              {data.objectives.map((objective) => (
                <Link key={objective.id} href={objective.actionUrl} className={objective.complete ? 'complete' : ''}>
                  <span>{objective.complete ? <Check /> : `${objective.progress}/${objective.target}`}</span>
                  <span><strong>{objective.title}</strong><small>{objective.description}</small></span>
                </Link>
              ))}
            </div>
          </section>

          <section className="eligibility-panel" aria-labelledby="eligibility-title">
            <span className="eyebrow">Live eligibility</span>
            <div className="eligibility-heading">
              <h2 id="eligibility-title">{data.eligibility.liveEligible ? 'Competition ready' : 'Build trust first'}</h2>
              <ShieldCheck />
            </div>
            {data.eligibility.blockers.length ? (
              <ul>{data.eligibility.blockers.map((blocker) => <li key={blocker}>{blocker}</li>)}</ul>
            ) : <p>Your identity, profile, and account status meet the current live-entry requirements.</p>}
            <Link href="/profile">Review status<ArrowRight /></Link>
          </section>

          <section className="tournament-panel" aria-labelledby="tournaments-title">
            <span className="eyebrow">Current tournaments</span>
            <h2 id="tournaments-title">Competition calendar</h2>
            {data.tournaments.length ? data.tournaments.slice(0, 3).map((tournament) => (
              <article key={tournament.id}>
                <span><strong>{tournament.name}</strong><small>{new Date(tournament.startsAt).toLocaleString()}</small></span>
                <span className={tournament.eligible ? 'available' : ''}>{tournament.eligible ? 'Eligible' : tournament.ineligibleReason}</span>
              </article>
            )) : <p className="hub-empty">No tournament is currently accepting players.</p>}
            <Link href="/tournaments">Tournament center<ArrowRight /></Link>
          </section>
        </aside>
      </div>
    </main>
  )
}
