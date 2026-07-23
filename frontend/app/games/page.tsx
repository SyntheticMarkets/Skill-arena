'use client'

import { Check, Gamepad2, LockKeyhole, Play, RotateCcw, ShieldCheck } from 'lucide-react'
import { useState } from 'react'
import { HubGame, useHub } from '../hub-context'
import { postJSON } from '../lib/api'
import { ArrowLine, ArrowLinePuzzle, escapeBlocker, normalizeLines } from '../maze-preview'

type StartGameResponse = {
  sessionId: string
  state: string
  difficultyRating: number
  generationHash: string
  lines?: ArrowLine[]
}

type FinishGameResponse = {
  session: { outcome: string; lines?: ArrowLine[] }
}

export default function GamesPage() {
  const { data, status, error: hubError, reload } = useHub()
  const [sessionId, setSessionId] = useState('')
  const [activeGame, setActiveGame] = useState<HubGame | null>(null)
  const [lines, setLines] = useState<ArrowLine[]>([])
  const [clickedLineIds, setClickedLineIds] = useState<string[]>([])
  const [generationHash, setGenerationHash] = useState('')
  const [difficulty, setDifficulty] = useState(0)
  const [message, setMessage] = useState('')
  const [error, setError] = useState('')
  const [busy, setBusy] = useState(false)

  const removedCount = lines.filter((line) => line.state === 'removed').length
  const progress = lines.length ? Math.round((removedCount / lines.length) * 100) : 0

  async function startPractice(game: HubGame) {
    setBusy(true)
    setError('')
    setMessage('')
    try {
      const response = await postJSON<StartGameResponse>('/api/v1/games/start', { gameType: 'demo', stake: 1 })
      setActiveGame(game)
      setSessionId(response.sessionId)
      setGenerationHash(response.generationHash)
      setDifficulty(response.difficultyRating)
      setClickedLineIds([])
      setLines(normalizeLines(response.lines))
      setMessage('The server generated and signed this Practice puzzle.')
    } catch (cause) {
      setError(cause instanceof Error ? cause.message : 'Practice could not be started.')
    } finally {
      setBusy(false)
    }
  }

  function clickLine(lineId: string) {
    setLines((current) => {
      const blocker = escapeBlocker(current, lineId)
      return current.map((line) => {
        if (line.id !== lineId || line.state === 'removed' || line.state === 'blocked' || line.state === 'exiting') return line
        setClickedLineIds((clicks) => [...clicks, lineId])
        if (blocker) {
          window.setTimeout(() => setLines((latest) => latest.map((item) => item.id === lineId && item.state === 'blocked' ? { ...item, state: 'ready' } : item)), 760)
          return { ...line, state: 'blocked' }
        }
        window.setTimeout(() => setLines((latest) => latest.map((item) => item.id === lineId && item.state === 'exiting' ? { ...item, state: 'removed' } : item)), 620)
        return { ...line, state: 'exiting' }
      })
    })
  }

  async function submitPractice() {
    setBusy(true)
    setError('')
    try {
      const response = await postJSON<FinishGameResponse>('/api/v1/games/finish', { sessionId, clickedLineIds })
      setLines(normalizeLines(response.session.lines))
      setMessage(`The server recorded this Practice result as ${response.session.outcome || 'pending'}.`)
      await reload()
    } catch (cause) {
      setError(cause instanceof Error ? cause.message : 'The Practice result could not be verified.')
    } finally {
      setBusy(false)
    }
  }

  if (status === 'loading' || status === 'idle') return <main className="hub-page"><div className="inline-loading">Loading registered game modules...</div></main>
  if (!data) return <main className="hub-page"><div className="form-message error">{hubError || 'Game catalog is unavailable.'}</div></main>

  return (
    <main className="hub-page">
      <section className="subpage-heading">
        <div><span className="eyebrow">Game directory</span><h1>Choose the skill to sharpen.</h1><p>Modules and capabilities come from Arena Core. Only server-authorized modes can be entered.</p></div>
        <Gamepad2 aria-hidden="true" />
      </section>

      {error ? <p className="form-message error" role="alert">{error}</p> : null}
      {message ? <p className="form-message success" role="status">{message}</p> : null}

      <section className="module-directory">
        {data.games.map((game) => (
          <article key={game.id} className={activeGame?.id === game.id ? 'active' : ''}>
            <div className="module-mark" aria-hidden="true">{game.name.split(' ').map((part) => part[0]).join('').slice(0, 2)}</div>
            <div>
              <span>{game.category} · v{game.version}</span>
              <h2>{game.name}</h2>
              <p>{game.description}</p>
              <div className="capability-list">
                {Object.entries(game.capabilities).filter(([, enabled]) => enabled).map(([capability]) => <span key={capability}>{capability}</span>)}
              </div>
              <ul>{game.rulesSummary.map((rule) => <li key={rule}>{rule}</li>)}</ul>
            </div>
            <div className="module-actions">
              {game.availability === 'available' && game.capabilities.practice ? <button className="button" type="button" disabled={busy} onClick={() => void startPractice(game)}><Play />Start Practice</button> : <span className="module-lock"><LockKeyhole />{game.availabilityReason || 'Practice is not supported.'}</span>}
              {game.capabilities.pvp ? <span className="module-lock"><ShieldCheck />Ranked entry is unavailable until live matchmaking is enabled.</span> : null}
            </div>
          </article>
        ))}
      </section>

      {activeGame && lines.length ? (
        <section className="practice-stage" aria-labelledby="practice-title">
          <div className="practice-heading">
            <div><span className="eyebrow">Server Practice</span><h2 id="practice-title">{activeGame.name}</h2><p>Difficulty {difficulty} · Generation {generationHash.slice(0, 12)}</p></div>
            <div><strong>{progress}%</strong><span>{lines.length - removedCount} arrows remain</span></div>
          </div>
          <ArrowLinePuzzle lines={lines.filter((line) => line.state !== 'removed')} label={`${activeGame.name} Practice puzzle`} onLineClick={clickLine} />
          <div className="practice-actions">
            <button className="button secondary" type="button" disabled={busy} onClick={() => void startPractice(activeGame)}><RotateCcw />New puzzle</button>
            <button className="button" type="button" disabled={busy || clickedLineIds.length === 0} onClick={() => void submitPractice()}><Check />Verify result</button>
          </div>
        </section>
      ) : null}
    </main>
  )
}
