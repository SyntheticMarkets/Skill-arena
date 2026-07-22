'use client'

import Link from 'next/link'
import { useEffect, useMemo, useState } from 'react'
import { ArrowLine, ArrowLinePuzzle, normalizeLines } from './maze-preview'

const apiBase = process.env.NEXT_PUBLIC_API_BASE_URL ?? 'http://localhost:8080'

type PlatformStats = {
  launchPhase: 'PRE_LAUNCH' | 'BETA' | 'LIVE' | string
  currentSeason?: string
  prizePool: number | null
  leaderboardsStatus: string
}

type PuzzlePreview = { lines: ArrowLine[]; animationLineIds: string[]; blockedAttemptId?: string }
type LoadState = 'loading' | 'ready' | 'error'

const trustProof = [
  ['Verified identity', 'Email ownership is confirmed before a competitor session can begin.'],
  ['Server authority', 'The platform creates and validates competitive state; the browser only presents it.'],
  ['Session control', 'Sessions can be reviewed, revoked, recovered, and protected with multi-factor authentication.'],
  ['Auditable access', 'Security-sensitive identity events are recorded for investigation and support.'],
  ['Practice before commitment', 'Players can understand the arena before any financial decision is requested.'],
]

function previewSelection(preview: PuzzlePreview | null) {
  const normalized = normalizeLines(preview?.lines)
  if (normalized.length === 0) return []
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
    if ((line.dependsOn ?? []).every((dependency) => selected.has(dependency))) selected.set(line.id, line)
  }
  const selectedIds = new Set(selected.keys())
  return Array.from(selected.values()).slice(0, 10).map((line) => ({
    ...line,
    dependsOn: (line.dependsOn ?? []).filter((dependency) => selectedIds.has(dependency)),
    removed: false,
    blocked: false,
    state: 'ready' as const,
  }))
}

export default function Home() {
  const [stats, setStats] = useState<PlatformStats | null>(null)
  const [preview, setPreview] = useState<PuzzlePreview | null>(null)
  const [state, setState] = useState<LoadState>('loading')

  useEffect(() => {
    let alive = true
    async function loadLanding() {
      try {
        const signal = AbortSignal.timeout(10_000)
        const [statsResponse, previewResponse] = await Promise.all([
          fetch(`${apiBase}/api/v1/platform/stats`, { cache: 'no-store', signal }),
          fetch(`${apiBase}/api/v1/platform/puzzle-preview`, { cache: 'no-store', signal }),
        ])
        if (!statsResponse.ok || !previewResponse.ok) throw new Error('landing data unavailable')
        const [statsBody, previewBody] = await Promise.all([statsResponse.json(), previewResponse.json()])
        if (!alive) return
        setStats(statsBody)
        setPreview(previewBody)
        setState('ready')
      } catch {
        if (alive) setState('error')
      }
    }
    void loadLanding()
    return () => { alive = false }
  }, [])

  const backgroundLines = useMemo(() => normalizeLines(preview?.lines), [preview])
  const puzzleLines = useMemo(() => previewSelection(preview), [preview])
  const statusCards = useMemo(() => {
    if (!stats) {
      return [
        ['Platform status', state === 'loading' ? 'Connecting' : 'Unavailable'],
        ['Arena directory', state === 'error' ? 'Connection required' : 'Loading'],
        ['Identity service', state === 'error' ? 'Connection required' : 'Loading'],
      ]
    }
    return [
      ['Season status', stats.currentSeason || stats.launchPhase || 'Not yet released'],
      ['Ranked availability', stats.launchPhase === 'LIVE' ? 'Open' : 'Not yet released'],
      ['Leaderboard service', stats.leaderboardsStatus || 'Not yet released'],
    ]
  }, [state, stats])

  return (
    <main className="entry-experience">
      <header className="entry-nav">
        <Link href="/" className="constructed-logo" aria-label="Skill Arena home"><span className="logo-mark" aria-hidden="true"><i /><i /><i /></span><span>Skill Arena</span></Link>
        <nav aria-label="Skill Arena public navigation"><a href="#play">Play</a><a href="#compete">Compete</a><a href="#progress">Progress</a><a href="#community">Community</a><a href="#support">Support</a><Link href="/auth/login">Login</Link><Link className="entry-nav-cta" href="/arena">Enter the Arena</Link></nav>
      </header>

      <section className="entry-hero" aria-labelledby="entry-title">
        <div className="living-maze" aria-hidden="true"><ArrowLinePuzzle lines={backgroundLines} label="Living maze background" readOnly compact animated /></div>
        <div className="hero-copy"><p className="hero-kicker">Every move matters.</p><h1 id="entry-title"><span>Where</span><span>Skill</span><span>Becomes</span><span>Value</span></h1><Link className="entry-button" href="/arena">Explore the Arena</Link><a className="scroll-cue" href="#play">Scroll to discover</a></div>
      </section>

      <section id="play" className="cinematic-section puzzle-reveal" aria-labelledby="try-title">
        <div className="maze-depth" aria-hidden="true"><ArrowLinePuzzle lines={backgroundLines.slice(0, 34)} label="Arena depth" readOnly compact animated /></div>
        <div className="puzzle-stage">
          <div className="section-label">Practice first. Deposit later.</div>
          <h2 id="try-title">Read the board before the clock begins.</h2>
          <p>This preview comes from the server puzzle pipeline. In competition, the server owns every seed, move decision, and result.</p>
          <div className="interactive-puzzle">
            {puzzleLines.length > 0 ? <ArrowLinePuzzle lines={puzzleLines} label="Server-generated Maze Arena preview" compact readOnly animated /> : <div className="platform-unavailable" role="status"><strong>{state === 'loading' ? 'Preparing the arena.' : 'Arena preview unavailable.'}</strong><span>{state === 'loading' ? 'The server is generating a verified board.' : 'The public platform connection could not be reached.'}</span></div>}
          </div>
          <div className="puzzle-controls"><strong>Observe. Learn. Then compete.</strong><span>No deposit is required to explore the arena.</span><Link className="entry-button small" href="/arena">View disciplines</Link></div>
        </div>
      </section>

      <section id="compete" className="entry-section platform-section" aria-labelledby="live-title">
        <div><span className="section-label">Live platform</span><h2 id="live-title">The arena is built on real state, not fake hype.</h2><p>When live data is not available, Skill Arena says so. No invented player counts. No fake match volume.</p></div>
        <div className="signal-grid">{statusCards.map(([label, value]) => <article key={label} className="signal-card"><span>{label}</span><strong>{value}</strong></article>)}</div>
      </section>

      <section id="progress" className="entry-section trust-section" aria-labelledby="trust-title">
        <div><span className="section-label">Trust by proof</span><h2 id="trust-title">Fair competition has to be visible.</h2></div>
        <div className="trust-proof-grid">{trustProof.map(([title, copy]) => <article key={title}><span /><h3>{title}</h3><p>{copy}</p></article>)}</div>
      </section>

      <section id="community" className="entry-section community-section" aria-labelledby="community-title">
        <div><span className="section-label">Community</span><h2 id="community-title">Competition becomes meaningful when there is a world around it.</h2></div>
        <div className="community-track"><article><span>Practice</span><strong>Learn without financial pressure</strong></article><article><span>Competition</span><strong>Enter only verified, server-controlled matches</strong></article><article><span>Progress</span><strong>Understand what improved after every session</strong></article></div>
      </section>

      <section id="support" className="entry-final" aria-labelledby="final-title"><span className="section-label">Your next move</span><h2 id="final-title">See the arena before choosing your next move.</h2><p>Explore the disciplines, understand how competition works, then create a verified competitor identity when you are ready.</p><Link className="entry-button" href="/arena">Explore the Arena</Link></section>
    </main>
  )
}
