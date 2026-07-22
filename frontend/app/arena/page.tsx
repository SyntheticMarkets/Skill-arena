'use client'

import Link from 'next/link'
import { ArrowRight, BrainCircuit, Clock3, LockKeyhole, LogOut, ShieldCheck, Swords } from 'lucide-react'
import { useEffect, useState } from 'react'
import { useAuth } from '../auth-context'
import { apiFetch } from '../lib/api'
import { ArrowLine, ArrowLinePuzzle, normalizeLines } from '../maze-preview'

type Features = { MazeArena: boolean; MemoryArena: boolean; ReactionArena: boolean; LogicArena: boolean }
type Preview = { lines: ArrowLine[] }

const futureGames = [
  { key: 'MemoryArena', name: 'Memory Arena', icon: BrainCircuit, discipline: 'Recall under pressure' },
  { key: 'LogicArena', name: 'Logic Arena', icon: ShieldCheck, discipline: 'Reasoning and deduction' },
  { key: 'ReactionArena', name: 'Reaction Arena', icon: Clock3, discipline: 'Speed and precision' },
] as const

export default function GuestArenaPage() {
  const { status, logout } = useAuth()
  const authenticated = status === 'authenticated'
  const [features, setFeatures] = useState<Features | null>(null)
  const [lines, setLines] = useState<ArrowLine[]>([])
  const [loadError, setLoadError] = useState(false)

  useEffect(() => {
    void Promise.all([
      apiFetch<Features>('/api/v1/config/features'),
      apiFetch<Preview>('/api/v1/platform/puzzle-preview'),
    ]).then(([flags, preview]) => { setFeatures(flags); setLines(normalizeLines(preview.lines).slice(0, 38)) }).catch(() => setLoadError(true))
  }, [])

  return (
    <main className="guest-arena">
      <header className="entry-nav guest-nav">
        <Link href="/" className="constructed-logo" aria-label="Skill Arena home"><span className="logo-mark" aria-hidden="true"><i /><i /><i /></span><span>Skill Arena</span></Link>
        <nav>{authenticated ? <><span className="session-active">Identity verified</span><button className="guest-logout" type="button" onClick={() => void logout()}><LogOut />Log out</button></> : <><Link href="/auth/login">Login</Link><Link className="entry-nav-cta" href="/auth/register">Create identity</Link></>}</nav>
      </header>
      <section className="guest-arena-hero">
        <div className="guest-maze" aria-hidden="true">{lines.length ? <ArrowLinePuzzle lines={lines} label="Maze Arena preview" readOnly compact animated /> : null}</div>
        <div className="guest-hero-copy"><span className="section-label">Arena directory</span><h1>Choose the skill you want to sharpen.</h1><p>{authenticated ? 'Your protected session is active. Review each discipline and prepare for the arena that fits how you think.' : 'Explore the competition before creating an account. Practice comes first; live competition unlocks only when you are ready.'}</p></div>
        <div className="arena-principles"><span><ShieldCheck />Server verified</span><span><Swords />Skill matched</span><span><LockKeyhole />Protected identity</span></div>
      </section>
      <section className="game-directory" aria-labelledby="games-heading">
        <div className="directory-heading"><div><span className="section-label">Arena disciplines</span><h2 id="games-heading">The founding arena is taking shape.</h2></div><p>Every game enters Skill Arena through the same rules for identity, fairness, progression, and replay integrity.</p></div>
        {loadError ? <div className="platform-unavailable" role="status"><strong>Arena status is temporarily unavailable.</strong><span>Refresh when the platform connection is restored.</span></div> : null}
        <div className="game-directory-grid">
          <article className="game-featured"><div className="game-index">01</div><div><span className="availability">{features?.MazeArena ? 'PREVIEW AVAILABLE' : 'UNAVAILABLE'}</span><h3>Maze Arena</h3><p>Read dependencies, release the right path, and solve faster than the competitor sharing your seed.</p><dl><div><dt>Core skill</dt><dd>Spatial logic</dd></div><div><dt>First mode</dt><dd>Practice</dd></div><div><dt>Integrity</dt><dd>Deterministic replay</dd></div></dl>{features?.MazeArena && !authenticated ? <Link className="entry-button small" href="/auth/register?intent=maze-practice">Create identity<ArrowRight /></Link> : null}</div></article>
          <div className="future-game-list">{futureGames.map((game, index) => { const Icon = game.icon; const enabled = Boolean(features?.[game.key]); return <article key={game.key}><span className="game-index">0{index + 2}</span><Icon /><div><h3>{game.name}</h3><p>{game.discipline}</p></div><strong>{enabled ? 'AVAILABLE' : 'NOT YET RELEASED'}</strong></article> })}</div>
        </div>
      </section>
      <section className="guest-conversion"><span className="section-label">Your first move</span><h2>{authenticated ? 'Your competitor identity is ready.' : 'Build skill before you risk anything.'}</h2><p>{authenticated ? 'Your email is verified and this session is protected. Review the disciplines while the founding arena prepares for competition.' : 'Create a verified competitor identity, enter Practice, and learn from your first replay. Wallet activation is a later choice, not an entry requirement.'}</p>{authenticated ? <a className="entry-button" href="#games-heading">Review disciplines<ArrowRight /></a> : <Link className="entry-button" href="/auth/register">Create competitor identity<ArrowRight /></Link>}</section>
    </main>
  )
}
