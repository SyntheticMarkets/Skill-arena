'use client'

import Link from 'next/link'
import { useEffect, useState } from 'react'
import { useRouter } from 'next/navigation'

const apiBase = process.env.NEXT_PUBLIC_API_BASE_URL ?? 'http://localhost:8080'

type Profile = {
  email: string
  liveBalance: number
  availableLiveBalance: number
  demoBalance: number
  availableDemoBalance: number
  level: number
  eloRating: number
  leagueTier: string
  seasonPoints: number
  wins: number
  losses: number
  currentStreak: number
  trustScore: number
}

type Season = {
  name: string
  theme: string
  rewardPool: number
  endsAt: string
}

type ReplayReport = {
  sessionId: string
  outcome: string
  moveCount: number
  integrityStatus: string
  createdAt: string
}

type TournamentDetail = {
  tournament: { id: string; name: string; entryFee: number; walletType: string; prizePool: number; status: string; startsAt: string }
  participants?: Array<unknown>
}

function authHeaders() {
  const token = window.localStorage.getItem('skill-arena-token')
  return token ? { Authorization: `Bearer ${token}` } : null
}

export default function DashboardPage() {
  const router = useRouter()
  const [profile, setProfile] = useState<Profile | null>(null)
  const [season, setSeason] = useState<Season | null>(null)
  const [replays, setReplays] = useState<ReplayReport[]>([])
  const [tournaments, setTournaments] = useState<TournamentDetail[]>([])
  const [error, setError] = useState('')

  useEffect(() => {
    const headers = authHeaders()
    if (!headers) {
      router.replace('/auth/login')
      return
    }
    loadDashboard(headers).catch(() => setError('Unable to load dashboard. Confirm the API is running.'))
  }, [router])

  async function loadDashboard(headers: { Authorization: string }) {
    const [profileRes, seasonRes, replaysRes, tournamentsRes] = await Promise.all([
      fetch(`${apiBase}/api/v1/profile`, { headers }),
      fetch(`${apiBase}/api/v1/seasons/current`, { headers }),
      fetch(`${apiBase}/api/v1/replays`, { headers }),
      fetch(`${apiBase}/api/v1/tournaments`, { headers }),
    ])
    if (!profileRes.ok) {
      router.replace('/auth/login')
      return
    }
    setProfile(await profileRes.json())
    setSeason(await seasonRes.json())
    setReplays(await replaysRes.json())
    setTournaments((await tournamentsRes.json()) ?? [])
  }

  return (
    <main className="page-shell">
      <section className="dashboard-command">
        <div>
          <span className="eyebrow">Dashboard</span>
          <h1>Command center</h1>
          <p>Wallet status, trust score, league progress, featured games, and daily actions before gameplay.</p>
        </div>
        <div className="quick-actions">
          <Link className="button" href="/games">Play</Link>
          <Link className="button secondary" href="/wallet">Wallet</Link>
        </div>
      </section>

      {error ? <p className="form-error">{error}</p> : null}

      <section className="metric-grid">
        <article className="metric-card">
          <span>Available Wallet</span>
          <strong>{profile ? profile.availableLiveBalance.toFixed(2) : '-'}</strong>
          <small>Live balance {profile ? profile.liveBalance.toFixed(2) : '-'}</small>
        </article>
        <article className="metric-card">
          <span>Demo Wallet</span>
          <strong>{profile ? profile.availableDemoBalance.toFixed(2) : '-'}</strong>
          <small>Practice balance {profile ? profile.demoBalance.toFixed(2) : '-'}</small>
        </article>
        <article className="metric-card">
          <span>Trust Score</span>
          <strong>{profile ? profile.trustScore.toFixed(1) : '-'}</strong>
          <small>Integrity and account confidence</small>
        </article>
        <article className="metric-card">
          <span>League</span>
          <strong>{profile?.leagueTier ?? '-'}</strong>
          <small>ELO {profile?.eloRating ?? '-'}</small>
        </article>
      </section>

      <section className="content-grid">
        <article className="panel-large">
          <div className="panel-heading">
            <div>
              <span className="eyebrow">Featured Games</span>
              <h2>Game Hub</h2>
            </div>
            <Link href="/games">View all</Link>
          </div>
          <div className="hub-grid compact">
            <article className="game-tile featured">
              <span className="tile-tag">Maze Arena</span>
              <h3>Arrow-Line Dependency Puzzle</h3>
              <p>Clear all directional lines by finding the correct dependency order.</p>
              <Link className="button" href="/games">Enter Arena</Link>
            </article>
            <article className="game-tile locked">
              <span className="tile-tag">Coming Soon</span>
              <h3>Memory Arena</h3>
              <p>Future competitive game sharing the same wallet and rankings.</p>
            </article>
          </div>
        </article>

        <article className="panel">
          <span className="eyebrow">Daily Challenges</span>
          <h2>Today</h2>
          <ul className="task-list">
            <li><span>Daily calibration</span><Link href="/challenges">Start</Link></li>
            <li><span>Bronze House clear</span><Link href="/challenges">Challenge</Link></li>
            <li><span>Play one PvP queue</span><Link href="/games">Queue</Link></li>
          </ul>
        </article>

        <article className="panel">
          <span className="eyebrow">League Progress</span>
          <h2>{profile?.seasonPoints ?? 0} SP</h2>
          <div className="progress-rail"><span style={{ width: `${Math.min(100, profile?.seasonPoints ?? 0)}%` }} /></div>
          <p>{season ? `${season.name}: ${season.theme}. Ends ${new Date(season.endsAt).toLocaleDateString()}.` : 'Loading season.'}</p>
        </article>

        <article className="panel">
          <span className="eyebrow">Upcoming Tournaments</span>
          <h2>{tournaments.length}</h2>
          <ul className="task-list">
            {tournaments.slice(0, 3).map((detail) => (
              <li key={detail.tournament.id}>
                <span>{detail.tournament.name}</span>
                <Link href="/tournaments">{detail.tournament.entryFee} {detail.tournament.walletType}</Link>
              </li>
            ))}
          </ul>
        </article>

        <article className="panel">
          <span className="eyebrow">Recent Replays</span>
          <h2>{replays.length}</h2>
          <ul className="task-list">
            {replays.slice(0, 3).map((replay) => (
              <li key={replay.sessionId}>
                <span>{replay.outcome || 'pending'} · {replay.moveCount} clicks</span>
                <Link href="/replays">{replay.integrityStatus}</Link>
              </li>
            ))}
          </ul>
        </article>
      </section>
    </main>
  )
}
