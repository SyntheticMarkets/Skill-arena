'use client'

import Link from 'next/link'
import { Medal, Trophy } from 'lucide-react'
import { useEffect, useState } from 'react'
import { useAuth } from '../auth-context'
import { apiFetch } from '../lib/api'

type Entry = {
  userId: string
  username: string
  displayName: string
  leagueTier: string
  rating: number
  country: string
  rank: number
}

export default function LeaderboardsPage() {
  const { status: authStatus } = useAuth()
  const [entries, setEntries] = useState<Entry[]>([])
  const [status, setStatus] = useState<'loading' | 'ready' | 'error'>('loading')

  useEffect(() => {
    const timer = window.setTimeout(() => {
      void apiFetch<Entry[]>('/api/v1/leaderboard')
        .then((body) => { setEntries(body); setStatus('ready') })
        .catch(() => setStatus('error'))
    }, 0)
    return () => window.clearTimeout(timer)
  }, [])

  return (
    <>
      {authStatus === 'guest' ? (
        <header className="public-subnav">
          <Link href="/" className="constructed-logo"><span className="logo-mark" aria-hidden="true"><i /><i /><i /></span><span>Skill Arena</span></Link>
          <nav><Link href="/arena">Games</Link><Link href="/auth/login">Login</Link><Link className="entry-nav-cta" href="/auth/register">Create identity</Link></nav>
        </header>
      ) : null}
      <main className={authStatus === 'guest' ? 'public-ranking-page' : 'hub-page'}>
        <section className="subpage-heading">
          <div><span className="eyebrow">World ranking</span><h1>Skill, measured publicly.</h1><p>Read-only standings are available to everyone. Hidden, suspended, and privileged accounts are excluded by the server.</p></div>
          <Trophy aria-hidden="true" />
        </section>
        {status === 'loading' ? <div className="inline-loading">Loading verified rankings...</div> : null}
        {status === 'error' ? <div className="form-message error">Rankings are temporarily unavailable.</div> : null}
        {status === 'ready' && entries.length === 0 ? <section className="empty-state"><Medal /><h2>No ranked competitors yet.</h2><p>The leaderboard will populate after verified competitive results exist.</p></section> : null}
        {entries.length ? (
          <div className="ranking-table-wrap">
            <table className="leaderboard-table">
              <thead><tr><th>Rank</th><th>Player</th><th>League</th><th>Rating</th><th>Region</th></tr></thead>
              <tbody>{entries.map((entry) => <tr key={entry.userId}><td>#{entry.rank}</td><td>{entry.displayName || entry.username}</td><td>{entry.leagueTier}</td><td>{entry.rating.toLocaleString()}</td><td>{entry.country || 'Not disclosed'}</td></tr>)}</tbody>
            </table>
          </div>
        ) : null}
      </main>
    </>
  )
}
