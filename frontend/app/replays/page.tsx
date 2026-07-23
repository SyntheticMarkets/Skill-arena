'use client'

import { PlaySquare, ShieldCheck } from 'lucide-react'
import { useEffect, useState } from 'react'
import { apiFetch } from '../lib/api'

type Replay = {
  sessionId: string
  gameType: string
  mode?: string
  outcome: string
  moveCount: number
  integrityStatus: string
  createdAt: string
}

export default function ReplaysPage() {
  const [replays, setReplays] = useState<Replay[]>([])
  const [status, setStatus] = useState<'loading' | 'ready' | 'error'>('loading')

  useEffect(() => {
    const timer = window.setTimeout(() => {
      void apiFetch<Replay[]>('/api/v1/replays')
        .then((response) => { setReplays(response); setStatus('ready') })
        .catch(() => setStatus('error'))
    }, 0)
    return () => window.clearTimeout(timer)
  }, [])

  return (
    <main className="hub-page">
      <section className="subpage-heading">
        <div><span className="eyebrow">Replay centre</span><h1>Every result leaves evidence.</h1><p>Only server-generated replay records appear here. Playback belongs to the game module that owns the rules version.</p></div>
        <PlaySquare aria-hidden="true" />
      </section>
      {status === 'loading' ? <div className="inline-loading">Loading replay records...</div> : null}
      {status === 'error' ? <div className="form-message error">Replay records are temporarily unavailable.</div> : null}
      {status === 'ready' && replays.length === 0 ? <section className="empty-state"><PlaySquare /><h2>No replay has been recorded.</h2><p>Complete a server-verified session to create your first replay record.</p></section> : null}
      <section className="replay-list">
        {replays.map((replay) => (
          <article key={replay.sessionId}>
            <span className="replay-icon"><ShieldCheck /></span>
            <div><span>{replay.gameType} · {replay.mode || 'session'}</span><h2>{replay.outcome || 'Awaiting outcome'}</h2><p>{replay.moveCount} recorded actions · {new Date(replay.createdAt).toLocaleString()}</p></div>
            <strong>{replay.integrityStatus}</strong>
          </article>
        ))}
      </section>
    </main>
  )
}
