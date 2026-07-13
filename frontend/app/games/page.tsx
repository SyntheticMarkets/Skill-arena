'use client'

import { useEffect, useMemo, useState } from 'react'
import { useRouter } from 'next/navigation'
import { ArrowLine, ArrowLinePuzzle, escapeBlocker, normalizeLines } from '../maze-preview'

const apiBase = process.env.NEXT_PUBLIC_API_BASE_URL ?? 'http://localhost:8080'

type PvPMatchDetail = {
  match: {
    id: string
    playerAId: string
    playerBId?: string
    queueType: string
    walletType: string
    stake: number
    status: string
    winnerId?: string
    playerALines?: ArrowLine[]
    playerBLines?: ArrowLine[]
  }
  submissions: Array<{ id: string; userId: string; isValidRoute: boolean; moveCount: number }>
}
type Profile = { id: string }
type StartGameResponse = { sessionId: string; lines?: ArrowLine[] }

function authHeaders() {
  const token = window.localStorage.getItem('skill-arena-token')
  return token ? { Authorization: `Bearer ${token}` } : null
}

export default function GamesPage() {
  const router = useRouter()
  const [stake, setStake] = useState(10)
  const [walletType, setWalletType] = useState('demo')
  const [queueType, setQueueType] = useState('ranked')
  const [matchId, setMatchId] = useState('')
  const [sessionId, setSessionId] = useState('')
  const [activeMode, setActiveMode] = useState<'practice' | 'pvp'>('practice')
  const [currentUserId, setCurrentUserId] = useState('')
  const [pvpMatches, setPvpMatches] = useState<PvPMatchDetail[]>([])
  const [lines, setLines] = useState<ArrowLine[]>([])
  const [clickedLineIds, setClickedLineIds] = useState<string[]>([])
  const [message, setMessage] = useState('')
  const [error, setError] = useState('')

  const removedCount = lines.filter((line) => line.state === 'removed').length
  const blockedCount = lines.filter((line) => line.state === 'blocked').length
  const progress = Math.round((removedCount / Math.max(1, lines.length)) * 100)
  const opponentProgress = matchId ? Math.min(96, Math.max(12, progress - 8 + (removedCount % 5) * 3)) : 0
  const opponentLines = useMemo(() => lines.slice(0, Math.min(34, lines.length)), [lines])

  useEffect(() => {
    const headers = authHeaders()
    if (!headers) {
      router.replace('/auth/login')
      return
    }
    Promise.all([
      fetch(`${apiBase}/api/v1/profile`, { headers }),
      fetch(`${apiBase}/api/v1/pvp/matches`, { headers }),
    ]).then(async ([profileRes, pvpRes]) => {
      if (profileRes.ok) {
        const profile: Profile = await profileRes.json()
        setCurrentUserId(profile.id)
      }
      const body = pvpRes.ok ? await pvpRes.json() : []
      setPvpMatches((body ?? []).filter((detail: PvPMatchDetail) => detail.match.playerAId !== detail.match.playerBId))
    }).catch(() => undefined)
  }, [router])

  async function startServerBoard() {
    setMessage('')
    setError('')
    const headers = authHeaders()
    if (!headers) return
    const response = await fetch(`${apiBase}/api/v1/games/start`, {
      method: 'POST',
      headers: { ...headers, 'Content-Type': 'application/json' },
      body: JSON.stringify({ gameType: walletType, stake }),
    })
    if (!response.ok) {
      setError(await response.text())
      return
    }
    const body: StartGameResponse = await response.json()
    setSessionId(body.sessionId)
    setMatchId('')
    setActiveMode('practice')
    setClickedLineIds([])
    setLines(normalizeLines(body.lines))
    setMessage('Server-generated Maze Arena board ready. Clear arrows only when their forward path can leave the board.')
  }

  async function joinPvP(event: React.FormEvent<HTMLFormElement>) {
    event.preventDefault()
    setMessage('')
    setError('')
    const headers = authHeaders()
    if (!headers) return
    const response = await fetch(`${apiBase}/api/v1/pvp/join`, {
      method: 'POST',
      headers: { ...headers, 'Content-Type': 'application/json' },
      body: JSON.stringify({ queueType, walletType, stake }),
    })
    if (!response.ok) {
      setError(await response.text())
      return
    }
    const detail: PvPMatchDetail = await response.json()
    if (detail.match.playerAId === detail.match.playerBId) {
      setError('Self-match prevented. Requeue before starting.')
      return
    }
    setMatch(detail)
    setMessage(detail.match.status === 'active' ? 'PvP match active. Clear all arrow lines before your opponent.' : 'Queued. Waiting for an opponent.')
  }

  function setMatch(detail: PvPMatchDetail) {
    setMatchId(detail.match.id)
    setSessionId('')
    setActiveMode('pvp')
    setClickedLineIds([])
    setLines(normalizeLines(detail.match.playerAId === currentUserId ? detail.match.playerALines : detail.match.playerBLines))
  }

  function clickLine(lineId: string) {
    setLines((current) => {
      const blocker = escapeBlocker(current, lineId)
      return current.map((line) => {
        if (line.id !== lineId || line.state === 'removed' || line.state === 'blocked' || line.state === 'exiting') return line
        setClickedLineIds((clicks) => [...clicks, lineId])
        if (blocker) {
          window.setTimeout(() => {
            setLines((latest) => latest.map((item) => item.id === lineId && item.state === 'blocked' ? { ...item, state: 'ready' } : item))
          }, 760)
          return { ...line, state: 'blocked' }
        }
        window.setTimeout(() => {
          setLines((latest) => latest.map((item) => item.id === lineId && item.state === 'exiting' ? { ...item, state: 'removed' } : item))
        }, 620)
        return { ...line, state: 'exiting' }
      })
    })
  }

  async function submitBoard() {
    setMessage('')
    setError('')
    const headers = authHeaders()
    if (!headers) return
    const endpoint = activeMode === 'pvp' ? `${apiBase}/api/v1/pvp/submit` : `${apiBase}/api/v1/games/finish`
    const body = activeMode === 'pvp' ? { matchId, moves: clickedLineIds } : { sessionId, clickedLineIds }
    const response = await fetch(endpoint, {
      method: 'POST',
      headers: { ...headers, 'Content-Type': 'application/json' },
      body: JSON.stringify(body),
    })
    if (!response.ok) {
      setError(await response.text())
      return
    }
    const result = await response.json()
    if (activeMode === 'practice') {
      setLines(normalizeLines(result.session?.lines))
      setMessage(`Board submitted: ${result.session?.outcome ?? 'pending'}.`)
    } else {
      setMessage(result.match?.status === 'completed' ? `PvP completed. Winner: ${result.match.winnerId || 'refund'}.` : 'PvP clicks submitted. Waiting for opponent.')
    }
  }

  return (
    <main className="page-shell">
      <section className="dashboard-command">
        <div>
          <span className="eyebrow">Arena Hub</span>
          <h1>Choose your arena.</h1>
          <p>Enter a game module from Skill Arena. Wallet, profile, trust, and overall progression stay with Arena Hub.</p>
        </div>
      </section>
      {error ? <p className="form-error">{error}</p> : null}
      {message ? <p className="form-success">{message}</p> : null}

      <section className="content-grid">
        <article className="panel">
          <span className="eyebrow">Maze Arena</span>
          <h2>Maze module</h2>
          <button className="button secondary" type="button" onClick={startServerBoard}>Start Server Board</button>
          <form onSubmit={joinPvP} className="form-grid">
            <label>Queue<select value={queueType} onChange={(event) => setQueueType(event.target.value)}><option value="ranked">Ranked</option><option value="casual">Casual</option></select></label>
            <label>Wallet<select value={walletType} onChange={(event) => setWalletType(event.target.value)}><option value="demo">Demo</option><option value="live">Live</option></select></label>
            <label>Stake<input type="number" min="1" value={stake} onChange={(event) => setStake(Number(event.target.value))} /></label>
            <button className="button" type="submit">Find PvP Match</button>
          </form>
          <div className="match-list">
            {pvpMatches.slice(0, 3).map((detail) => (
              <button key={detail.match.id} type="button" onClick={() => setMatch(detail)}>
                {detail.match.queueType} - {detail.match.status} - {detail.match.stake} {detail.match.walletType}
              </button>
            ))}
          </div>
        </article>

        <article className="panel-large">
          <div className="arena-stage">
            <div className="puzzle-panel">
              <div className="panel-heading">
                <div><span className="eyebrow">Player Maze</span><h2>{progress}% clear</h2></div>
                <strong>{lines.length - removedCount} lines left</strong>
              </div>
              <button className="button" type="button" disabled={clickedLineIds.length === 0 || (activeMode === 'practice' && !sessionId) || (activeMode === 'pvp' && !matchId)} onClick={submitBoard}>Submit Click Replay</button>
              <ArrowLinePuzzle lines={lines.filter((line) => line.state !== 'removed')} label="Player arrow line maze" onLineClick={clickLine} />
            </div>
            <div className="opponent-panel">
              <div className="panel-heading">
                <div><span className="eyebrow">Opponent Maze</span><h2>{opponentProgress}% clear</h2></div>
                <strong>{Math.max(0, Math.round(lines.length * (1 - opponentProgress / 100)))} est. left</strong>
              </div>
              <div className="pixelated">
                <ArrowLinePuzzle lines={opponentLines} label="Pixelated hidden opponent maze" readOnly />
              </div>
              <div className="opponent-stats">
                <span>Opponent Progress {opponentProgress}%</span>
                <span>Estimated Moves Remaining {Math.max(0, Math.round(lines.length * (1 - opponentProgress / 100)))}</span>
                <span>Current Combo {Math.max(0, removedCount - blockedCount)}</span>
                <span>Accuracy {clickedLineIds.length === 0 ? 100 : Math.round((removedCount / clickedLineIds.length) * 100)}%</span>
              </div>
            </div>
          </div>
        </article>
      </section>
    </main>
  )
}
