'use client'

import Link from 'next/link'

export default function ChallengesPage() {
  return (
    <main className="page-shell">
      <section className="dashboard-command">
        <div>
          <span className="eyebrow">Challenges</span>
          <h1>Daily challenges</h1>
          <p>Calibration, house attempts, and PvP tasks route through the existing game systems.</p>
        </div>
        <div className="quick-actions">
          <Link className="button" href="/games">Play</Link>
          <Link className="button secondary" href="/dashboard">Dashboard</Link>
        </div>
      </section>

      <section className="content-grid">
        <article className="panel">
          <span className="eyebrow">Calibration</span>
          <h2>Baseline run</h2>
          <p>Complete a calibration run before entering higher trust-sensitive modes.</p>
          <Link href="/games">Start</Link>
        </article>

        <article className="panel">
          <span className="eyebrow">House</span>
          <h2>Bronze House</h2>
          <p>Attempt the entry house challenge once your account is eligible.</p>
          <Link href="/games">Challenge</Link>
        </article>

        <article className="panel">
          <span className="eyebrow">PvP</span>
          <h2>Ranked queue</h2>
          <p>Join a live opponent queue from the Game Hub.</p>
          <Link href="/games">Queue</Link>
        </article>
      </section>
    </main>
  )
}
