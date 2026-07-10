'use client'

import Link from 'next/link'
import { useEffect, useMemo, useState } from 'react'
import { t } from './i18n'
import { ArrowLine, ArrowLinePuzzle, normalizeLines } from './maze-preview'

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

const arenaCards = [
  { title: t.landingMazeTitle, status: t.landingMazeStatus, copy: t.landingMazeCopy, live: true },
  { title: t.landingMemoryTitle, status: t.landingMemoryStatus, copy: t.landingMemoryCopy },
  { title: t.landingLogicTitle, status: t.landingLogicStatus, copy: t.landingLogicCopy },
  { title: t.landingReactionTitle, status: t.landingReactionStatus, copy: t.landingReactionCopy },
]

const journeySteps = [
  t.landingJourneyAccount,
  t.landingJourneyPractice,
  t.landingJourneyRanked,
  t.landingJourneyHouse,
  t.landingJourneyTournaments,
  t.landingJourneyNumberOne,
]

const trustCards = [
  [t.landingWhyServer, t.landingWhyServerCopy],
  [t.landingWhyUnique, t.landingWhyUniqueCopy],
  [t.landingWhyReplay, t.landingWhyReplayCopy],
  [t.landingWhyMatchmaking, t.landingWhyMatchmakingCopy],
  [t.landingWhyAntiCheat, t.landingWhyAntiCheatCopy],
  [t.landingWhyTrust, t.landingWhyTrustCopy],
]

const faqs = [
  [t.landingFAQSkillQ, t.landingFAQSkillA],
  [t.landingFAQReplayQ, t.landingFAQReplayA],
  [t.landingFAQGamesQ, t.landingFAQGamesA],
]

export default function Home() {
  const [stats, setStats] = useState<PlatformStats | null>(null)
  const [preview, setPreview] = useState<PuzzlePreview | null>(null)
  const [previewLines, setPreviewLines] = useState<ArrowLine[]>([])
  const [state, setState] = useState<LoadState>('loading')

  useEffect(() => {
    let alive = true
    async function loadLanding() {
      try {
        const [statsResponse, previewResponse] = await Promise.all([
          fetch(`${apiBase}/api/v1/platform/stats`, { cache: 'no-store' }),
          fetch(`${apiBase}/api/v1/platform/puzzle-preview`, { cache: 'no-store' }),
        ])
        if (!statsResponse.ok || !previewResponse.ok) {
          throw new Error('landing data unavailable')
        }
        const [statsBody, previewBody] = await Promise.all([statsResponse.json(), previewResponse.json()])
        if (!alive) return
        setStats(statsBody)
        setPreview(previewBody)
        setPreviewLines(normalizeLines(previewBody.lines))
        setState('ready')
      } catch {
        if (!alive) return
        setState('error')
      }
    }
    loadLanding()
    return () => {
      alive = false
    }
  }, [])

  useEffect(() => {
    if (!preview) return
    const timers: number[] = []
    const mark = (lineId: string, state: 'removed' | 'blocked' | 'ready') => {
      setPreviewLines((current) => current.map((line) => line.id === lineId ? { ...line, state } : line))
    }

    let delay = 850
    preview.animationLineIds.slice(0, 6).forEach((lineId, index) => {
      timers.push(window.setTimeout(() => mark(lineId, 'removed'), delay))
      delay += index === 1 ? 900 : 680
    })
    if (preview.blockedAttemptId) {
      timers.push(window.setTimeout(() => mark(preview.blockedAttemptId || '', 'blocked'), 2700))
      timers.push(window.setTimeout(() => mark(preview.blockedAttemptId || '', 'ready'), 3400))
    }
    if (preview.unlockedRetryId) {
      timers.push(window.setTimeout(() => mark(preview.unlockedRetryId || '', 'removed'), delay + 420))
    }
    timers.push(window.setTimeout(async () => {
      try {
        const response = await fetch(`${apiBase}/api/v1/platform/puzzle-preview`, { cache: 'no-store' })
        if (!response.ok) return
        const body: PuzzlePreview = await response.json()
        setPreview(body)
        setPreviewLines(normalizeLines(body.lines))
      } catch {
        setPreviewLines(normalizeLines(preview.lines))
      }
    }, 13500))
    return () => timers.forEach((timer) => window.clearTimeout(timer))
  }, [preview])

  const statusCards = useMemo(() => {
    if (!stats || stats.launchPhase === 'PRE_LAUNCH') {
      return [
        [t.landingPlatformStatus, t.landingPreLaunch],
        [t.landingSeasonOne, t.landingComingSoon],
        [t.landingFounderAccess, t.landingRegistrationOpen],
      ]
    }
    return [
      [t.landingPlatformStatus, stats.launchPhase],
      [t.landingSeasonOne, stats.currentSeason || t.topbarSeason],
      [t.landingPrizePool, stats.prizePool == null ? t.landingPrizePoolTba : `$${stats.prizePool.toLocaleString()}`],
    ]
  }, [stats])

  return (
    <main className="landing-experience">
      <div className="neural-field" aria-hidden="true">
        {Array.from({ length: 18 }).map((_, index) => <span key={index} />)}
      </div>

      <header className="marketing-nav landing-nav">
        <Link href="/" className="marketing-brand" aria-label={t.brandName}>
          <span className="sa-shield">SA</span>
          <span>{t.brandName}</span>
        </Link>
        <nav aria-label={`${t.brandName} public navigation`}>
          <Link href="/auth/login">{t.navLogin}</Link>
          <Link className="button small" href="/auth/register">{t.landingStartJourney}</Link>
        </nav>
      </header>

      <section className="landing-experience-hero">
        <div className="hero-signal">
          <span className="eyebrow">{t.landingHeroEyebrow}</span>
          <h1>
            <span>{t.landingHeroCompete}</span>
            <span>{t.landingHeroThink}</span>
            <span>{t.landingHeroConquer}</span>
          </h1>
          <p>{t.landingHeroCopy}</p>
          <ul>
            <li>{t.landingHeroUnique}</li>
            <li>{t.landingHeroEarned}</li>
            <li>{t.landingHeroDeserved}</li>
          </ul>
          <div className="hero-actions">
            <Link className="button" href="/auth/register">{t.landingStartJourney}</Link>
            <Link className="button secondary" href="/games">{t.landingWatchGameplay}</Link>
          </div>
        </div>
        <div className="hero-puzzle-frame" aria-label={t.landingMazeTitle}>
          <div className="puzzle-orbit-label">
            <span>{state === 'error' ? t.statsUnavailable : t.landingLiveGeneration}</span>
            <strong>{t.landingMazeTitle}</strong>
          </div>
          <ArrowLinePuzzle lines={previewLines.filter((line) => line.state !== 'removed')} label={t.landingMazeTitle} readOnly compact animated />
          <div className="solve-caption">
            <span>{t.landingDependencyMaze}</span>
            <strong>{previewLines.filter((line) => line.state === 'removed').length}/{previewLines.length || 56}</strong>
          </div>
        </div>
      </section>

      <section className="landing-section compact-status" aria-labelledby="platform-status">
        <div className="section-heading">
          <span className="eyebrow">{t.landingStatsEyebrow}</span>
          <h2 id="platform-status">{t.landingStatusTitle}</h2>
        </div>
        <div className="status-track">
          {state === 'loading' ? (
            Array.from({ length: 3 }).map((_, index) => <div key={index} className="stat-skeleton" />)
          ) : (
            statusCards.map(([label, value]) => (
              <article key={label} className="landing-stat">
                <span>{label}</span>
                <strong>{value}</strong>
              </article>
            ))
          )}
        </div>
      </section>

      <section className="landing-section featured-arena-section" aria-labelledby="featured-arena">
        <div className="section-heading">
          <span className="eyebrow">{t.landingFeaturedEyebrow}</span>
          <h2 id="featured-arena">{t.landingFeaturedTitle}</h2>
        </div>
        <div className="arena-card-grid">
          {arenaCards.map((arena) => (
            <article key={arena.title} className={arena.live ? 'arena-card live' : 'arena-card'}>
              <span className="tile-tag">{arena.status}</span>
              <h3>{arena.title}</h3>
              <p>{arena.copy}</p>
              {arena.live ? <Link className="button small" href="/auth/register">{t.landingEnter}</Link> : null}
            </article>
          ))}
        </div>
      </section>

      <section className="landing-section flow-section" aria-labelledby="how-it-works">
        <div className="section-heading">
          <span className="eyebrow">{t.landingHowEyebrow}</span>
          <h2 id="how-it-works">{t.landingHowTitle}</h2>
        </div>
        <div className="journey-flow" aria-label={t.landingHowTitle}>
          {journeySteps.map((step, index) => (
            <div key={step} className="journey-node">
              <span>{String(index + 1).padStart(2, '0')}</span>
              <strong>{step}</strong>
            </div>
          ))}
        </div>
      </section>

      <section className="landing-section" aria-labelledby="why-skill-arena">
        <div className="section-heading">
          <span className="eyebrow">{t.landingWhyEyebrow}</span>
          <h2 id="why-skill-arena">{t.landingWhyTitle}</h2>
        </div>
        <div className="why-grid trust-grid">
          {trustCards.map(([title, copy]) => (
            <article key={title} className="why-card">
              <h3>{title}</h3>
              <p>{copy}</p>
            </article>
          ))}
        </div>
      </section>

      <section className="landing-section season-launch" aria-labelledby="season-one">
        <div>
          <span className="eyebrow">{t.landingSeasonEyebrow}</span>
          <h2 id="season-one">{t.landingSeasonTitle}</h2>
          <p>{t.landingSeasonCopy}</p>
          <Link className="button" href="/auth/register">{t.landingStartJourney}</Link>
        </div>
        <div className="countdown-panel" aria-label={t.landingCountdownLabel}>
          <span>{t.landingCountdownLabel}</span>
          <strong>{t.landingCountdownSoon}</strong>
          <small>{t.landingRegistrationOpen}</small>
        </div>
      </section>

      <section className="landing-section faq-section" aria-labelledby="faq">
        <div className="section-heading">
          <span className="eyebrow">{t.landingFAQEyebrow}</span>
          <h2 id="faq">{t.landingFAQTitle}</h2>
        </div>
        <div className="faq-list">
          {faqs.map(([question, answer]) => (
            <details key={question}>
              <summary>{question}</summary>
              <p>{answer}</p>
            </details>
          ))}
        </div>
      </section>

      <footer className="marketing-footer">
        <div>
          <strong>{t.brandName}</strong>
          <p>{t.landingFooterCopy}</p>
        </div>
        <div>
          <span>{t.landingFooterPlatform}</span>
          <Link href="/auth/register">{t.landingStartJourney}</Link>
          <Link href="/games">{t.navGames}</Link>
        </div>
        <div>
          <span>{t.landingFooterLegal}</span>
          <Link href="/auth/login">{t.navLogin}</Link>
          <Link href="/settings">{t.navSettings}</Link>
        </div>
      </footer>
    </main>
  )
}
