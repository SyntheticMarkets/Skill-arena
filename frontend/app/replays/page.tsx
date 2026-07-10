'use client'

import { useEffect, useState } from 'react'
import { useRouter } from 'next/navigation'

const apiBase = process.env.NEXT_PUBLIC_API_BASE_URL ?? 'http://localhost:8080'
type Replay = { sessionId: string; outcome: string; moveCount: number; shortestPathLength: number; routeEfficiency: number; integrityStatus: string; createdAt: string }
const authHeaders = () => { const token = window.localStorage.getItem('skill-arena-token'); return token ? { Authorization: `Bearer ${token}` } : null }

export default function ReplaysPage() {
  const router = useRouter()
  const [replays, setReplays] = useState<Replay[]>([])
  useEffect(() => { const headers = authHeaders(); if (!headers) { router.replace('/auth/login'); return }; fetch(`${apiBase}/api/v1/replays`, { headers }).then((r) => r.json()).then((body) => setReplays(body ?? [])).catch(() => undefined) }, [router])
  return <main className="page-shell"><section className="dashboard-command"><div><span className="eyebrow">Replays</span><h1>Replay Center</h1><p>Every challenge remains reviewable and verifiable.</p></div></section><section className="panel-large"><table className="leaderboard-table"><thead><tr><th>Time</th><th>Outcome</th><th>Clicks</th><th>Efficiency</th><th>Integrity</th></tr></thead><tbody>{replays.map((replay) => <tr key={replay.sessionId}><td>{new Date(replay.createdAt).toLocaleString()}</td><td>{replay.outcome}</td><td>{replay.moveCount}</td><td>{Math.round(replay.routeEfficiency * 100)}%</td><td>{replay.integrityStatus}</td></tr>)}</tbody></table></section></main>
}
