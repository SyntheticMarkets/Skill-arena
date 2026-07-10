'use client'

import Link from 'next/link'
import { useEffect, useMemo, useState } from 'react'
import { ArrowLine, ArrowLinePuzzle, escapeBlocker, normalizeLines } from './maze-preview'

const apiBase = process.env.NEXT_PUBLIC_API_BASE_URL ?? 'http://localhost:8080'

type PlatformStats = {
  launchPhase: 'PRE_LAUNCH' | 'BETA' | 'LIVE' | string
  currentSeason?: string
  prizePool: number | null
  leaderboardsStatus: string
}

type PuzzlePreview = {
  lines: ArrowLine[]
  animationLineIds: string[]
  blockedAttemptId?: string
  unlockedRetryId?: string
}

type LoadState = 'loading' | 'ready' | 'error'
type PuzzleState = 'waiting' | 'playing' | 'blocked' | 'solved'

const fallbackLines: ArrowLine[] = [
  { id: 'practice-1', x: 16, y: 24, length: 18, direction: 'right', points: [{ x: 16, y: 24 }, { x: 34, y: 24 }] },
  { id: 'practice-2', x: 38, y: 24, length: 18, direction: 'down', points: [{ x: 38, y: 24 }, { x: 38, y: 42 }], dependsOn: ['practice-1'] },
  { id: 'practice-3', x: 38, y: 46, length: 18, direction: 'right', points: [{ x: 38, y: 46 }, { x: 56, y: 46 }], dependsOn: ['practice-2'] },
  { id: 'practice-4', x: 60, y: 46, length: 17, direction: 'up', points: [{ x: 60, y: 46 }, { x: 60, y: 29 }], dependsOn: ['practice-3'] },
  { id: 'practice-5', x: 64, y: 29, length: 17, direction: 'right', points: [{ x: 64, y: 29 }, { x: 81, y: 29 }], dependsOn: ['practice-4'] },
  { id: 'practice-6', x: 22, y: 65, length: 18, direction: 'right', points: [{ x: 22, y: 65 }, { x: 40, y: 65 }] },
  { id: 'practice-7', x: 44, y: 65, length: 18, direction: 'right', points: [{ x: 44, y: 65 }, { x: 62, y: 65 }], dependsOn: ['practice-6'] },
  { id: 'practice-8', x: 66, y: 65, length: 16, direction: 'up', points: [{ x: 66, y: 65 }, { x: 66, y: 49 }], dependsOn: ['practice-5', 'practice-7'] },
]

const trustProof = [
  ['Replay verified', 'Every competitive result can be reconstructed and reviewed.'],
  ['Server authoritative', 'Progress, scoring, and match state are owned by the backend.'],
  ['Secure wallet', 'Funds move through pending, review, settlement, and ledger states.'],
  ['Treasury reviewed', 'Withdrawals and financial risk events are auditable.'],
  ['Fair matchmaking', 'Competition is shaped around skill, eligibility, and mode.'],
]

const platformSignals = [
  ['Today’s challenge', 'Coming Soon'],
  ['Ranked availability', 'Opening with Season One'],
  ['Tournament schedule', 'Coming Soon'],
]

const communitySignals = [
  ['Current champion', 'Coming Soon'],
  ['Houses', 'Founding Houses preparing'],
  ['Leaderboards', 'Open at launch'],
]

function smallPuzzleFromPreview(preview: PuzzlePreview | null) {
  const normalized = normalizeLines(preview?.lines)
  if (normalized.length === 0) return normalizeLines(fallbackLines)

  const byId = new Map(normalized.map((line) => [line.id, line]))
  const selected = new Map<string, ArrowLine>()

  function include(lineId: string | undefined) {
    if (!lineId || selected.size >= 10 || selected.has(lineId)) return
    const line = byId.get(lineId)
    if (!line) return
    ;(line.dependsOn ?? []).forEach(include)
    selected.set(line.id, line)
  }

  include(preview?.blockedAttemptId)
  preview?.animationLineIds.slice(0, 10).forEach(include)

  for (const line of normalized) {
    if (selected.size >= 10) break
    if ((line.dependsOn ?? []).every((dependency) => selected.has(dependency))) {
      selected.set(line.id, line)
    }
  }

  const selectedIds = new Set(selected.keys())
  const lines = Array.from(selected.values()).slice(0, 10).map((line) => ({
    ...line,
    dependsOn: (line.dependsOn ?? []).filter((dependency) => selectedIds.has(dependency)),
    removed: false,
    blocked: false,
    state: 'ready' as const,
  }))

  return lines.length >= 6 ? lines : normalizeLines(fallbackLines)
}

function livingLines(preview: PuzzlePreview | null) {
  const lines = normalizeLines(preview?.lines)
  return lines.length > 0 ? lines : normalizeLines(fallbackLines)
}

export default function Home() {
  const [stats, setStats] = useState<PlatformStats | null>(null)
  const [preview, setPreview] = useState<PuzzlePreview | null>(null)
  const [state, setState] = useState<LoadState>('loading')
  const [puzzleLines, setPuzzleLines] = useState<ArrowLine[]>(normalizeLines(fallbackLines))
  const [clearedIds, setClearedIds] = useState<Set<string>>(new Set())
  const [puzzleState, setPuzzleState] = useState<PuzzleState>('waiting')

  useEffect(() => {
    let alive = true
    async function loadLanding() {
      try {
        const [statsResponse, previewResponse] = await Promise.all([
          fetch(`${apiBase}/api/v1/platform/stats`, { cache: 'no-store' }),
          fetch(`${apiBase}/api/v1/platform/puzzle-preview`, { cache: 'no-store' }),
        ])
        if (!statsResponse.ok || !previewResponse.ok) throw new Error('landing data unavailable')
        const [statsBody, previewBody] = await Promise.all([statsResponse.json(), previewResponse.json()])
        if (!alive) return
        setStats(statsBody)
        setPreview(previewBody)
        setPuzzleLines(smallPuzzleFromPreview(previewBody))
        setState('ready')
      } catch {
        if (!alive) return
        setState('error')
        setPuzzleLines(normalizeLines(fallbackLines))
      }
    }
    loadLanding()
    return () => {
      alive = false
    }
  }, [])

  const backgroundLines = useMemo(() => livingLines(preview), [preview])
  const visiblePuzzleLines = useMemo(() => puzzleLines.filter((line) => !clearedIds.has(line.id)), [clearedIds, puzzleLines])

  const statusCards = useMemo(() => {
    if (!stats) {
      return [
        ['Season status', state === 'loading' ? 'Loading' : 'Coming Soon'],
        ['Ranked availability', 'Coming Soon'],
        ['Platform announcements', state === 'error' ? 'Backend unavailable' : 'Founder access opening'],
      ]
    }
    return [
      ['Season status', stats.currentSeason || stats.launchPhase || 'Coming Soon'],
      ['Ranked availability', stats.launchPhase === 'LIVE' ? 'Open' : 'Coming Soon'],
      ['Platform announcements', stats.leaderboardsStatus || 'Founder access opening'],
    ]
  }, [state, stats])

  function startPuzzle() {
    setClearedIds(new Set())
    setPuzzleLines((current) => current.map((line) => ({ ...line, state: 'ready' })))
    setPuzzleState('playing')
  }

  function handleLineClick(lineId: string) {
    if (puzzleState === 'waiting') setPuzzleState('playing')
    if (puzzleState === 'solved') return

    const line = puzzleLines.find((item) => item.id === lineId)
    if (!line) return
    const blocker = escapeBlocker(puzzleLines.map((item) => clearedIds.has(item.id) ? { ...item, state: 'removed' } : item), lineId)
    if (blocker) {
      setPuzzleState('blocked')
      if ('vibrate' in navigator) navigator.vibrate(35)
      setPuzzleLines((current) => current.map((item) => item.id === lineId ? { ...item, state: 'blocked' } : item))
      window.setTimeout(() => {
        setPuzzleLines((current) => current.map((item) => item.id === lineId ? { ...item, state: 'ready' } : item))
        setPuzzleState('playing')
      }, 520)
      return
    }

    const nextCleared = new Set(clearedIds)
    nextCleared.add(lineId)
    setClearedIds(nextCleared)
    setPuzzleLines((current) => current.map((item) => item.id === lineId ? { ...item, state: 'removed' } : item))
    if (nextCleared.size === puzzleLines.length) {
      setPuzzleState('solved')
    }
  }

  return (
    <main className="entry-experience">
      <header className="entry-nav">
        <Link href="/" className="constructed-logo" aria-label="Skill Arena home">
          <span className="logo-mark" aria-hidden="true">
            <i />
            <i />
            <i />
          </span>
          <span>Skill Arena</span>
        </Link>
        <nav aria-label="Skill Arena public navigation">
          <a href="#play">Play</a>
          <a href="#compete">Compete</a>
          <a href="#progress">Progress</a>
          <a href="#community">Community</a>
          <a href="#support">Support</a>
          <Link href="/auth/login">Login</Link>
          <Link className="entry-nav-cta" href="/auth/register">Enter the Arena</Link>
        </nav>
      </header>

      <section className="entry-hero" aria-labelledby="entry-title">
        <div className="living-maze" aria-hidden="true">
          <ArrowLinePuzzle lines={backgroundLines} label="Living maze background" readOnly compact animated />
        </div>
        <div className="hero-copy">
          <p className="hero-kicker">Every move matters.</p>
          <h1 id="entry-title">
            <span>Where</span>
            <span>Skill</span>
            <span>Becomes</span>
            <span>Value</span>
          </h1>
          <Link className="entry-button" href="/auth/register">Enter the Arena</Link>
          <a className="scroll-cue" href="#play">Scroll to discover</a>
        </div>
      </section>

      <section id="play" className="cinematic-section puzzle-reveal" aria-labelledby="try-title">
        <div className="maze-depth" aria-hidden="true">
          <ArrowLinePuzzle lines={backgroundLines.slice(0, 34)} label="Arena depth" readOnly compact animated />
        </div>
        <div className="puzzle-stage">
          <div className="section-label">Practice first. Deposit later.</div>
          <h2 id="try-title">Think you can solve this?</h2>
          <p>
            This puzzle is generated from the same Maze Arena engine that powers real matches.
            Clear every line. Blocked moves bounce back.
          </p>
          <div className={`interactive-puzzle ${puzzleState === 'solved' ? 'solved' : ''}`}>
            <ArrowLinePuzzle
              lines={visiblePuzzleLines}
              label="Interactive Maze Arena practice puzzle"
              compact
              animated={puzzleState === 'waiting'}
              onLineClick={handleLineClick}
            />
          </div>
          <div className="puzzle-controls">
            {puzzleState === 'solved' ? (
              <>
                <strong>Well done.</strong>
                <span>Ready to compete against real players?</span>
                <Link className="entry-button small" href="/auth/register">Enter the Arena</Link>
              </>
            ) : (
              <>
                <strong>{puzzleState === 'blocked' ? 'Blocked. A dependency is still active.' : 'Try it.'}</strong>
                <span>{clearedIds.size}/{puzzleLines.length} lines cleared</span>
                <button className="entry-button small" type="button" onClick={startPuzzle}>Try It</button>
              </>
            )}
          </div>
        </div>
      </section>

      <section id="compete" className="entry-section platform-section" aria-labelledby="live-title">
        <div>
          <span className="section-label">Live platform</span>
          <h2 id="live-title">The arena is built on real state, not fake hype.</h2>
          <p>When live data is not available, Skill Arena says so. No invented player counts. No fake match volume.</p>
        </div>
        <div className="signal-grid">
          {statusCards.map(([label, value]) => (
            <article key={label} className="signal-card">
              <span>{label}</span>
              <strong>{value}</strong>
            </article>
          ))}
          {platformSignals.map(([label, value]) => (
            <article key={label} className="signal-card muted">
              <span>{label}</span>
              <strong>{value}</strong>
            </article>
          ))}
        </div>
      </section>

      <section id="progress" className="entry-section trust-section" aria-labelledby="trust-title">
        <div>
          <span className="section-label">Trust by proof</span>
          <h2 id="trust-title">Fair competition has to be visible.</h2>
        </div>
        <div className="trust-proof-grid">
          {trustProof.map(([title, copy]) => (
            <article key={title}>
              <span />
              <h3>{title}</h3>
              <p>{copy}</p>
            </article>
          ))}
        </div>
      </section>

      <section id="community" className="entry-section community-section" aria-labelledby="community-title">
        <div>
          <span className="section-label">Community</span>
          <h2 id="community-title">Competition becomes meaningful when there is a world around it.</h2>
        </div>
        <div className="community-track">
          {communitySignals.map(([label, value]) => (
            <article key={label}>
              <span>{label}</span>
              <strong>{value}</strong>
            </article>
          ))}
        </div>
      </section>

      <section id="support" className="entry-final" aria-labelledby="final-title">
        <span className="section-label">First minute complete</span>
        <h2 id="final-title">You solved a puzzle. Now race someone.</h2>
        <p>The next step creates your competitor identity and takes you into practice, replay insight, and live competition when you are ready.</p>
        <Link className="entry-button" href="/auth/register">Enter the Arena</Link>
      </section>
    </main>
  )
}
