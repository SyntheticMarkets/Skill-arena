'use client'

import { useEffect, useState } from 'react'

const apiBase = process.env.NEXT_PUBLIC_API_BASE_URL ?? 'http://localhost:8080'
type Entry = { userId: string; username: string; displayName: string; leagueTier: string; rating: number; country: string; score: number; rank: number }

export default function LeaderboardsPage() {
  const [entries, setEntries] = useState<Entry[]>([])
  useEffect(() => { fetch(`${apiBase}/api/v1/leaderboard`).then((r) => r.json()).then((body) => setEntries(body ?? [])).catch(() => undefined) }, [])
  return (
    <main className="page-shell">
      <section className="dashboard-command"><div><span className="eyebrow">Leaderboards</span><h1>Rankings</h1><p>Platform-wide competitive standing.</p></div></section>
      <section className="panel-large"><table className="leaderboard-table"><thead><tr><th>Rank</th><th>Player</th><th>League</th><th>Rating</th><th>Country</th></tr></thead><tbody>{entries.map((entry) => <tr key={entry.userId}><td>{entry.rank}</td><td>{entry.displayName || entry.username}</td><td>{entry.leagueTier}</td><td>{entry.rating || Math.round(entry.score)}</td><td>{entry.country || 'South Africa'}</td></tr>)}</tbody></table></section>
    </main>
  )
}
