'use client'

import { Gamepad2, LockKeyhole, Play, ShieldCheck, Swords } from 'lucide-react'
import Link from 'next/link'
import { useHub } from '../hub-context'
import { resolveRenderer } from './registry/catalog'

export default function GamesPage() {
  const { data, status, error } = useHub()

  if (status === 'loading' || status === 'idle') {
    return <main className="hub-page"><div className="inline-loading">Loading registered game modules...</div></main>
  }
  if (!data) {
    return <main className="hub-page"><div className="form-message error">{error || 'Game catalog is unavailable.'}</div></main>
  }

  return (
    <main className="hub-page">
      <section className="subpage-heading">
        <div>
          <span className="eyebrow">Game directory</span>
          <h1>Choose the skill to sharpen.</h1>
          <p>Every playable surface resolves from Arena Core. The game client renders server decisions and sends intent only.</p>
        </div>
        <Gamepad2 aria-hidden="true" />
      </section>

      <section className="module-directory">
        {data.games.map((game) => {
          const renderer = resolveRenderer(game.rendererKey, game.version)
          const playable = game.availability === 'available' && Boolean(renderer)
          return (
            <article key={game.id}>
              <div className="module-mark" aria-hidden="true">
                {game.name.split(' ').map((part) => part[0]).join('').slice(0, 2)}
              </div>
              <div>
                <span>{game.category} · v{game.version}</span>
                <h2>{game.name}</h2>
                <p>{game.description}</p>
                <div className="capability-list">
                  {Object.entries(game.capabilities)
                    .filter(([, enabled]) => enabled)
                    .map(([capability]) => <span key={capability}>{capability}</span>)}
                </div>
                <ul>{game.rulesSummary.map((rule) => <li key={rule}>{rule}</li>)}</ul>
              </div>
              <div className="module-actions">
                {playable && game.capabilities.practice ? (
                  <Link className="button" href="/games/maze">
                    <Play />Enter Practice
                  </Link>
                ) : (
                  <span className="module-lock">
                    <LockKeyhole />{game.availabilityReason || 'An approved renderer is unavailable.'}
                  </span>
                )}
                {playable && game.capabilities.pvp ? (
                  <Link className="button secondary" href="/games/maze">
                    <Swords />Live Duel
                  </Link>
                ) : game.capabilities.pvp ? (
                  <span className="module-lock"><ShieldCheck />Live play requires an approved renderer.</span>
                ) : null}
              </div>
            </article>
          )
        })}
      </section>
    </main>
  )
}
