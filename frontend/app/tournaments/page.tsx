'use client'

import { useEffect, useState } from 'react'
import { useRouter } from 'next/navigation'
import { ArrowLine, ArrowLinePuzzle, normalizeLines } from '../maze-preview'

const apiBase = process.env.NEXT_PUBLIC_API_BASE_URL ?? 'http://localhost:8080'

type TournamentMatch = { id: string; round: number; matchNumber: number; playerAId?: string; playerBId?: string; winnerId?: string; status: string; playerALines?: ArrowLine[]; playerBLines?: ArrowLine[] }
type TournamentDetail = {
  tournament: { id: string; name: string; type: string; status: string; entryFee: number; walletType: string; prizePool: number; startsAt: string; maxParticipants: number; description: string }
  registered: boolean
  participants?: Array<unknown>
  matches?: TournamentMatch[]
  submissions?: Array<{ matchId: string; userId: string; isComplete: boolean; moveCount: number }>
}
type Profile = { id: string }

function authHeaders() {
  const token = window.localStorage.getItem('skill-arena-token')
  return token ? { Authorization: `Bearer ${token}` } : null
}

export default function TournamentsPage() {
  const router = useRouter()
  const [items, setItems] = useState<TournamentDetail[]>([])
  const [currentUserId, setCurrentUserId] = useState('')
  const [activeTournamentId, setActiveTournamentId] = useState('')
  const [activeMatchId, setActiveMatchId] = useState('')
  const [lines, setLines] = useState<ArrowLine[]>([])
  const [clickedLineIds, setClickedLineIds] = useState<string[]>([])
  const [message, setMessage] = useState('')
  const [error, setError] = useState('')

  const removedCount = lines.filter((line) => line.state === 'removed').length
  const progress = Math.round((removedCount / Math.max(1, lines.length)) * 100)

  useEffect(() => {
    load().catch(() => setError('Unable to load tournaments.'))
  }, [])

  async function load() {
    const headers = authHeaders()
    if (!headers) {
      router.replace('/auth/login')
      return
    }
    const [profileRes, tournamentRes] = await Promise.all([
      fetch(`${apiBase}/api/v1/profile`, { headers }),
      fetch(`${apiBase}/api/v1/tournaments`, { headers }),
    ])
    if (profileRes.ok) {
      const profile: Profile = await profileRes.json()
      setCurrentUserId(profile.id)
    }
    const body = tournamentRes.ok ? await tournamentRes.json() : []
    setItems((body ?? []).map((detail: TournamentDetail) => ({ ...detail, participants: detail.participants ?? [], matches: detail.matches ?? [], submissions: detail.submissions ?? [] })))
  }

  async function register(tournamentId: string) {
    const headers = authHeaders()
    if (!headers) return
    setError('')
    const response = await fetch(`${apiBase}/api/v1/tournaments/register`, {
      method: 'POST',
      headers: { ...headers, 'Content-Type': 'application/json' },
      body: JSON.stringify({ tournamentId }),
    })
    if (!response.ok) {
      setError(await response.text())
      return
    }
    setMessage('Tournament registration confirmed.')
    load()
  }

  function loadMatch(tournamentId: string, match: TournamentMatch) {
    setActiveTournamentId(tournamentId)
    setActiveMatchId(match.id)
    setClickedLineIds([])
    setLines(normalizeLines(match.playerAId === currentUserId ? match.playerALines : match.playerBLines))
    setMessage('Tournament board loaded. Clear all lines and submit the click replay.')
  }

  function clickLine(lineId: string) {
    setLines((current) => {
      const removed = new Set(current.filter((line) => line.state === 'removed').map((line) => line.id))
      return current.map((line) => {
        if (line.id !== lineId || line.state === 'removed') return line
        setClickedLineIds((clicks) => [...clicks, lineId])
        const blocked = (line.dependsOn ?? []).some((dependency) => !removed.has(dependency))
        if (blocked) {
          window.setTimeout(() => {
            setLines((latest) => latest.map((item) => item.id === lineId && item.state === 'blocked' ? { ...item, state: 'ready' } : item))
          }, 3000)
          return { ...line, state: 'blocked' }
        }
        return { ...line, state: 'removed' }
      })
    })
  }

  async function submitMatch() {
    const headers = authHeaders()
    if (!headers) return
    setError('')
    const response = await fetch(`${apiBase}/api/v1/tournaments/submit-match`, {
      method: 'POST',
      headers: { ...headers, 'Content-Type': 'application/json' },
      body: JSON.stringify({ tournamentId: activeTournamentId, matchId: activeMatchId, clickedLineIds }),
    })
    if (!response.ok) {
      setError(await response.text())
      return
    }
    setMessage('Tournament match submitted. Result settles after both players submit.')
    setClickedLineIds([])
    await load()
  }

  return (
    <main className="page-shell">
      <section className="dashboard-command"><div><span className="eyebrow">Tournaments</span><h1>Competitive events</h1><p>Register, load bracket boards, and submit verified click replays.</p></div></section>
      {message ? <p className="form-success">{message}</p> : null}{error ? <p className="form-error">{error}</p> : null}
      <section className="content-grid">
        <div className="challenge-grid">
          {items.map((detail) => (
            <article key={detail.tournament.id} className="challenge-card">
              <span className="tile-tag">{detail.tournament.type}</span>
              <h2>{detail.tournament.name}</h2>
              <p>{detail.tournament.description}</p>
              <dl>
                <div><dt>Entry</dt><dd>{detail.tournament.entryFee} {detail.tournament.walletType}</dd></div>
                <div><dt>Prize</dt><dd>{detail.tournament.prizePool}</dd></div>
                <div><dt>Players</dt><dd>{detail.participants?.length ?? 0}/{detail.tournament.maxParticipants}</dd></div>
                <div><dt>Status</dt><dd>{detail.tournament.status}</dd></div>
              </dl>
              <button className="button" disabled={detail.registered} onClick={() => register(detail.tournament.id)}>{detail.registered ? 'Registered' : 'Register'}</button>
              <div className="match-list">
                {(detail.matches ?? []).filter((match) => match.status !== 'completed' && (match.playerAId === currentUserId || match.playerBId === currentUserId)).map((match) => (
                  <button key={match.id} type="button" onClick={() => loadMatch(detail.tournament.id, match)}>
                    Round {match.round} Match {match.matchNumber} - {match.status}
                  </button>
                ))}
              </div>
            </article>
          ))}
        </div>

        <article className="panel-large">
          <div className="panel-heading">
            <div><span className="eyebrow">Tournament Match Board</span><h2>{progress}% clear</h2></div>
            <strong>{lines.length - removedCount} lines left</strong>
          </div>
          {lines.length > 0 ? (
            <>
              <button className="button" type="button" disabled={clickedLineIds.length === 0} onClick={submitMatch}>Submit Tournament Replay</button>
              <ArrowLinePuzzle lines={lines.filter((line) => line.state !== 'removed')} label="Tournament arrow line board" onLineClick={clickLine} />
            </>
          ) : (
            <p>Register for an event and load an active bracket match.</p>
          )}
        </article>
      </section>
    </main>
  )
}
