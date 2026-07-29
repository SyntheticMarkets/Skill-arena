'use client'

import {
  Activity,
  ArrowLeft,
  CheckCircle2,
  LoaderCircle,
  LogOut,
  Radio,
  RefreshCw,
  ShieldCheck,
  Swords,
  Volume2,
  VolumeX,
} from 'lucide-react'
import Link from 'next/link'
import { useCallback, useEffect, useMemo, useRef, useState } from 'react'
import { apiFetch } from '../../lib/api'
import {
  cancelRealtimeQueue,
  enterRealtimeQueue,
  GameActionResult,
  GameSync,
  RealtimeClient,
  RealtimeEvent,
} from '../../lib/realtime'
import { actionAnnouncement } from './accessibility/announcements'
import type { MazeAnimation } from './animation/transitions'
import { animationDuration } from './animation/transitions'
import { playMazeCue } from './audio/cues'
import { createMazeArrowAction } from './protocol/actions'
import {
  MazeReplayAction,
  MazeSnapshot,
  parseMazeActionResult,
  parseMazeReplayAction,
  parseMazeSnapshot,
} from './protocol/schemas'
import { MazeRenderer } from './renderer/MazeRenderer'
import {
  MazeReplayViewer,
  VerifiedReplayMetadata,
} from './replay/MazeReplayViewer'

type ArenaPhase =
  | 'idle'
  | 'queueing'
  | 'preparing'
  | 'live'
  | 'reconnecting'
  | 'completed'
  | 'failed'
  | 'replay'

const errorCopy: Record<string, string> = {
  GAME_SEQUENCE_GAP: 'Your action sequence was out of date. The board is being synchronized.',
  GAME_STATE_CONFLICT: 'The arena advanced before this action arrived. The authoritative board is being restored.',
  GAME_DUPLICATE_MISMATCH: 'This action identifier was already used for another intent.',
  GAME_ACTION_REJECTED: 'The server rejected that action.',
  GAME_SYNC_REJECTED: 'The authoritative board could not be synchronized.',
  GATEWAY_UNAVAILABLE: 'The live arena connection is unavailable.',
  RECONNECT_EXHAUSTED: 'The live connection could not be recovered.',
  INVALID_GATEWAY_MESSAGE: 'An invalid live message was rejected.',
}

function initialReplaySnapshot(snapshot: MazeSnapshot): MazeSnapshot {
  return {
    ...snapshot,
    removedIds: [],
    status: 'active',
    completedAtMs: -1,
    progress: {
      ...snapshot.progress,
      removed: 0,
      remaining: snapshot.arrows.length,
      successfulActions: 0,
      blockedActions: 0,
      currentCombo: 0,
      maximumCombo: 0,
      completionBps: 0,
      complete: false,
    },
  }
}

function useReducedMotion() {
  const [reduced, setReduced] = useState(false)
  useEffect(() => {
    const media = window.matchMedia('(prefers-reduced-motion: reduce)')
    const update = () => setReduced(media.matches)
    update()
    media.addEventListener('change', update)
    return () => media.removeEventListener('change', update)
  }, [])
  return reduced
}

export function MazeArena({ replayMatchId }: { replayMatchId?: string }) {
  const systemReducedMotion = useReducedMotion()
  const [motionReduced, setMotionReduced] = useState<boolean | null>(() => {
    if (typeof window === 'undefined') return null
    const saved = window.localStorage.getItem('skill-arena-maze-motion')
    return saved === 'reduced' ? true : saved === 'full' ? false : null
  })
  const reducedMotion = motionReduced ?? systemReducedMotion
  const [soundEnabled, setSoundEnabled] = useState(() =>
    typeof window !== 'undefined' &&
    window.localStorage.getItem('skill-arena-maze-sound') === 'enabled',
  )
  const [mode, setMode] = useState<'practice' | 'pvp'>('practice')
  const [phase, setPhase] = useState<ArenaPhase>(replayMatchId ? 'preparing' : 'idle')
  const [connection, setConnection] = useState<'connecting' | 'connected' | 'reconnecting' | 'closed'>('closed')
  const [snapshot, setSnapshot] = useState<MazeSnapshot | null>(null)
  const [initialSnapshot, setInitialSnapshot] = useState<MazeSnapshot | null>(null)
  const [animation, setAnimation] = useState<MazeAnimation | null>(null)
  const [pendingArrow, setPendingArrow] = useState('')
  const [authoritativeStateVersion, setAuthoritativeStateVersion] = useState(0)
  const [lastClientSequence, setLastClientSequence] = useState(0)
  const [lastServerSequence, setLastServerSequence] = useState(0)
  const [events, setEvents] = useState<RealtimeEvent[]>([])
  const [announcement, setAnnouncement] = useState('')
  const [error, setError] = useState('')
  const [remainingMs, setRemainingMs] = useState(0)
  const [lastLatencyMs, setLastLatencyMs] = useState(0)
  const [replayMetadata, setReplayMetadata] = useState<VerifiedReplayMetadata | null>(null)
  const [replayStatus, setReplayStatus] = useState('')
  const clientRef = useRef<RealtimeClient | null>(null)
  const matchIdRef = useRef(replayMatchId ?? '')
  const readyOnConnect = useRef(false)
  const actionStartedAt = useRef(0)
  const animationTimer = useRef<number | null>(null)
  const queueTimer = useRef<number | null>(null)
  const snapshotRef = useRef<MazeSnapshot | null>(null)
  const reducedMotionRef = useRef(reducedMotion)
  const soundEnabledRef = useRef(soundEnabled)

  const applySync = useCallback((game: GameSync) => {
    try {
      const parsed = parseMazeSnapshot(game)
      snapshotRef.current = parsed
      setSnapshot(parsed)
      setInitialSnapshot((current) => current ?? initialReplaySnapshot(parsed))
      setAuthoritativeStateVersion(game.stateVersion)
      setLastClientSequence(game.lastClientSequence)
      setLastServerSequence(game.lastServerSequence)
      setPendingArrow('')
      setAnimation(null)
      setError('')
      setPhase(parsed.status === 'active' ? 'live' : 'completed')
    } catch {
      setError('The server snapshot did not match the approved Maze renderer contract.')
      setPhase('failed')
    }
  }, [])

  const loadReplay = useCallback(async (matchId: string, attempts = 12) => {
    for (let attempt = 0; attempt < attempts; attempt += 1) {
      try {
        const [metadata, eventPage] = await Promise.all([
          apiFetch<VerifiedReplayMetadata>(`/api/v1/realtime/replays/${matchId}`),
          apiFetch<{ events: RealtimeEvent[] }>(`/api/v1/realtime/events/${matchId}?after=0`),
        ])
        if (metadata.status !== 'verified') {
          throw new Error('Replay verification is incomplete')
        }
        setReplayMetadata(metadata)
        setEvents(eventPage.events)
        setReplayStatus('Replay integrity verified.')
        return
      } catch {
        if (attempt === attempts - 1) {
          setReplayStatus('Replay verification is still processing. Try again shortly.')
          return
        }
        await new Promise((resolve) => window.setTimeout(resolve, 500))
      }
    }
  }, [])

  const handleReceipt = useCallback((result: GameActionResult) => {
    try {
      const parsed = parseMazeActionResult(result)
      const nextAnimation: MazeAnimation = {
        id: result.receipt.receiptHash,
        presentation: parsed.presentation,
        kind: parsed.accepted ? 'accepted' : 'blocked',
      }
      if (animationTimer.current) window.clearTimeout(animationTimer.current)
      setAnimation(nextAnimation)
      setAuthoritativeStateVersion(parsed.stateVersion)
      setLastClientSequence(parsed.clientSequence)
      setLastServerSequence(result.receipt.lastEventSequence)
      setLastLatencyMs(Math.round(performance.now() - actionStartedAt.current))
      setAnnouncement(actionAnnouncement(parsed.presentation, parsed.snapshot.progress))
      playMazeCue(
        parsed.snapshot.progress.complete ? 'complete' : parsed.accepted ? 'accepted' : 'blocked',
        soundEnabledRef.current,
      )
      animationTimer.current = window.setTimeout(() => {
        snapshotRef.current = parsed.snapshot
        setSnapshot(parsed.snapshot)
        setAnimation(null)
        setPendingArrow('')
        if (parsed.snapshot.status !== 'active') {
          setPhase('completed')
          void loadReplay(parsed.snapshot.matchId)
        }
      }, animationDuration(nextAnimation, reducedMotionRef.current))
    } catch {
      setPendingArrow('')
      setError('The action receipt did not match the approved Maze protocol.')
      clientRef.current?.requestGameSync()
    }
  }, [loadReplay])

  const openMatch = useCallback((matchId: string, ready: boolean) => {
    clientRef.current?.close()
    matchIdRef.current = matchId
    readyOnConnect.current = ready
    const client = new RealtimeClient({
      onStatus: (status) => {
        setConnection(status)
        if (status === 'reconnecting') setPhase('reconnecting')
        if (status === 'connected' && readyOnConnect.current) {
          readyOnConnect.current = false
          setPhase('preparing')
          client.ready(matchId)
        }
      },
      onMatch: (nextMatch) => {
        if (nextMatch.status === 'live') {
          client.requestGameSync(nextMatch.id)
        } else if (nextMatch.status === 'completed' || nextMatch.status === 'abandoned') {
          setPhase('completed')
          void loadReplay(nextMatch.id)
        }
      },
      onSync: (nextMatch, nextEvents, game) => {
        setEvents((current) => {
          const known = new Set(current.map((event) => event.id))
          return [...current, ...nextEvents.filter((event) => !known.has(event.id))]
        })
        if (nextEvents.length) {
          setLastServerSequence(nextEvents[nextEvents.length - 1].sequence)
        }
        if (game) applySync(game)
        if (nextMatch.status !== 'live') void loadReplay(nextMatch.id)
      },
      onGameSync: applySync,
      onActionReceipt: handleReceipt,
      onEvent: (event) => {
        setEvents((current) => current.some((item) => item.id === event.id) ? current : [...current, event])
        setLastServerSequence(event.sequence)
        if (event.type === 'game.replay.ready') void loadReplay(matchId)
      },
      onError: (code) => {
        setPendingArrow('')
        setError(errorCopy[code] ?? 'The arena rejected an unexpected live request.')
        if (code === 'GAME_SEQUENCE_GAP' || code === 'GAME_STATE_CONFLICT') {
          client.requestGameSync(matchId)
        }
      },
    })
    clientRef.current = client
    client.connect(ready ? '' : matchId, 0, 'af-south')
  }, [applySync, handleReceipt, loadReplay])

  useEffect(() => {
    reducedMotionRef.current = reducedMotion
  }, [reducedMotion])

  useEffect(() => {
    soundEnabledRef.current = soundEnabled
  }, [soundEnabled])

  useEffect(() => {
    if (replayMatchId) openMatch(replayMatchId, false)
    return () => {
      clientRef.current?.close()
      if (animationTimer.current) window.clearTimeout(animationTimer.current)
      if (queueTimer.current) window.clearTimeout(queueTimer.current)
    }
  }, [openMatch, replayMatchId])

  useEffect(() => {
    if (!snapshot || snapshot.status !== 'active') return
    const update = () => setRemainingMs(Math.max(0, snapshot.deadlineAtMs - Date.now()))
    update()
    const timer = window.setInterval(update, 250)
    return () => window.clearInterval(timer)
  }, [snapshot])

  async function waitForMatch() {
    try {
      const queue = await apiFetch<{ matchId?: string; status: string }>('/api/v1/realtime/queue')
      if (queue.matchId) {
        openMatch(queue.matchId, true)
        return
      }
      if (queue.status !== 'waiting') {
        setPhase('failed')
        setError('The matchmaking request ended before a match was assigned.')
        return
      }
      queueTimer.current = window.setTimeout(() => void waitForMatch(), 1000)
    } catch (cause) {
      setPhase('failed')
      setError(cause instanceof Error ? cause.message : 'Matchmaking status is unavailable.')
    }
  }

  async function start() {
    setError('')
    setReplayMetadata(null)
    setReplayStatus('')
    setSnapshot(null)
    setInitialSnapshot(null)
    setEvents([])
    setPhase('queueing')
    try {
      const result = await enterRealtimeQueue({
        gameId: 'maze_arena',
        mode,
        walletCategory: mode === 'practice' ? 'practice' : 'live',
        region: 'af-south',
        jurisdiction: 'ZA',
        latencyMs: lastLatencyMs,
      })
      if (result.match) {
        openMatch(result.match.id, true)
      } else {
        setAnnouncement('Searching for an equally rated opponent.')
        void waitForMatch()
      }
    } catch (cause) {
      setPhase('failed')
      setError(cause instanceof Error ? cause.message : 'The arena could not be entered.')
    }
  }

  async function cancelQueue() {
    if (queueTimer.current) window.clearTimeout(queueTimer.current)
    try {
      await cancelRealtimeQueue()
      setPhase('idle')
      setAnnouncement('Match search canceled.')
    } catch (cause) {
      setError(cause instanceof Error ? cause.message : 'The queue could not be canceled.')
    }
  }

  function submitArrow(arrowId: string) {
    const current = snapshotRef.current
    if (!current || current.status !== 'active' || pendingArrow || animation) return
    setError('')
    setPendingArrow(arrowId)
    setAnnouncement('Action submitted. Waiting for the arena decision.')
    actionStartedAt.current = performance.now()
    clientRef.current?.submitGameAction(createMazeArrowAction({
      matchId: current.matchId,
      arrowId,
      clientSequence: lastClientSequence + 1,
      expectedStateVersion: authoritativeStateVersion,
      latencyMs: lastLatencyMs,
    }))
  }

  function leave() {
    clientRef.current?.leave(matchIdRef.current)
    setPhase('completed')
    setAnnouncement('You left the match. The server recorded the result.')
  }

  const replayActions = useMemo(() => {
    const parsed: MazeReplayAction[] = []
    for (const event of events) {
      try {
        const action = parseMazeReplayAction(event)
        if (action) parsed.push(action)
      } catch {
        return []
      }
    }
    return parsed.sort((left, right) => left.sequence - right.sequence)
  }, [events])
  const progressPercent = snapshot ? snapshot.progress.completionBps / 100 : 0
  const seconds = Math.ceil(remainingMs / 1000)
  const isTerminal = snapshot?.status === 'completed' || snapshot?.status === 'timed_out'

  if (replayMatchId && replayMetadata && initialSnapshot) {
    return (
      <main className="hub-page maze-page">
        <div className="maze-back"><Link href="/replays"><ArrowLeft />Replay Center</Link></div>
        <MazeReplayViewer
          initial={initialSnapshot}
          actions={replayActions}
          metadata={replayMetadata}
          reducedMotion={reducedMotion}
        />
      </main>
    )
  }

  return (
    <main className="hub-page maze-page">
      <header className="maze-arena-header">
        <div>
          <span className="eyebrow">Maze Arena / Rules v1</span>
          <h1>Read the board. Break the chain.</h1>
          <p>Every move is validated by the arena. Your browser renders the decision; it never decides it.</p>
        </div>
        <div className="maze-preferences" aria-label="Maze presentation preferences">
          <button
            type="button"
            role="switch"
            aria-checked={soundEnabled}
            onClick={() => {
              const next = !soundEnabled
              setSoundEnabled(next)
              window.localStorage.setItem('skill-arena-maze-sound', next ? 'enabled' : 'disabled')
            }}
            title={soundEnabled ? 'Mute game audio' : 'Enable game audio'}
          >
            {soundEnabled ? <Volume2 /> : <VolumeX />}
            <span>Sound</span>
          </button>
          <button
            type="button"
            role="switch"
            aria-checked={reducedMotion}
            onClick={() => {
              const next = !reducedMotion
              setMotionReduced(next)
              window.localStorage.setItem('skill-arena-maze-motion', next ? 'reduced' : 'full')
            }}
            title="Reduce movement animation"
          >
            <Activity />
            <span>Reduced motion</span>
          </button>
        </div>
      </header>

      <div className="sr-status" role="status" aria-live="polite">{announcement}</div>
      {error ? (
        <div className="maze-error" role="alert">
          <span>{error}</span>
          {matchIdRef.current ? (
            <button type="button" onClick={() => clientRef.current?.requestGameSync(matchIdRef.current)}>
              <RefreshCw />Restore board
            </button>
          ) : null}
        </div>
      ) : null}

      {phase === 'idle' || phase === 'failed' ? (
        <section className="maze-entry" aria-labelledby="maze-entry-title">
          <div>
            <span className="section-label">Choose your arena</span>
            <h2 id="maze-entry-title">One ruleset. Two ways to improve.</h2>
            <p>Practice creates a fresh puzzle only for you. Live Duel assigns one shared puzzle and keeps each competitor&apos;s board state independent.</p>
          </div>
          <div className="maze-mode-select" role="group" aria-label="Maze mode">
            <button type="button" className={mode === 'practice' ? 'active' : ''} onClick={() => setMode('practice')} aria-pressed={mode === 'practice'}>
              <ShieldCheck /><span><strong>Practice</strong><small>Fresh private puzzle</small></span>
            </button>
            <button type="button" className={mode === 'pvp' ? 'active' : ''} onClick={() => setMode('pvp')} aria-pressed={mode === 'pvp'}>
              <Swords /><span><strong>Live Duel</strong><small>Shared puzzle, independent state</small></span>
            </button>
          </div>
          <button type="button" className="button maze-enter" onClick={() => void start()}>
            <Radio />{mode === 'practice' ? 'Enter Practice' : 'Find an Opponent'}
          </button>
        </section>
      ) : null}

      {phase === 'queueing' ? (
        <section className="maze-waiting" aria-live="polite">
          <LoaderCircle className="spin" />
          <span className="eyebrow">{mode === 'practice' ? 'Preparing puzzle' : 'Live matchmaking'}</span>
          <h2>{mode === 'practice' ? 'The Puzzle Service is qualifying your board.' : 'Finding a fair opponent.'}</h2>
          <p>{mode === 'practice' ? 'Generate, solve, validate, score, assign.' : 'The same authoritative puzzle will be assigned to both players.'}</p>
          {mode === 'pvp' ? <button type="button" className="button secondary" onClick={() => void cancelQueue()}>Cancel search</button> : null}
        </section>
      ) : null}

      {(phase === 'preparing' || phase === 'reconnecting') && !snapshot ? (
        <section className="maze-waiting" aria-live="polite">
          <LoaderCircle className="spin" />
          <span className="eyebrow">{phase === 'reconnecting' ? 'Connection recovery' : 'Authoritative state'}</span>
          <h2>{phase === 'reconnecting' ? 'Rejoining your match.' : 'Synchronizing the board.'}</h2>
          <p>No local fallback is used. Play resumes only after the server snapshot arrives.</p>
        </section>
      ) : null}

      {snapshot ? (
        <>
          <section className="maze-hud" aria-label="Match status">
            <div><span>Time</span><strong>{snapshot.status === 'active' ? `${Math.floor(seconds / 60)}:${String(seconds % 60).padStart(2, '0')}` : 'Final'}</strong></div>
            <div><span>Progress</span><strong>{progressPercent.toFixed(0)}%</strong></div>
            <div><span>Remaining</span><strong>{snapshot.progress.remaining}</strong></div>
            <div><span>Combo</span><strong>{snapshot.progress.currentCombo}</strong></div>
            <div><span>Connection</span><strong className={`connection-${connection}`}>{connection}</strong></div>
          </section>
          <div className="maze-progress-track" aria-label={`${progressPercent.toFixed(0)} percent complete`}>
            <i style={{ width: `${progressPercent}%` }} />
          </div>
          <section className="maze-live-stage">
            <div className="maze-stage-heading">
              <div>
                <span className="eyebrow">{mode === 'pvp' ? 'Shared competitive puzzle' : 'Private Practice puzzle'}</span>
                <h2>{pendingArrow ? 'Arena decision pending' : snapshot.status === 'active' ? 'Your move' : 'Match complete'}</h2>
              </div>
              <span className="maze-integrity"><ShieldCheck />{snapshot.puzzleHash.slice(0, 12)}</span>
            </div>
            <MazeRenderer
              snapshot={snapshot}
              animation={animation}
              reducedMotion={reducedMotion}
              disabled={phase !== 'live' || Boolean(pendingArrow) || Boolean(animation) || snapshot.status !== 'active'}
              onArrow={submitArrow}
            />
            <footer className="maze-stage-footer">
              <span>State {snapshot.progress.successfulActions} / Client {lastClientSequence} / Event {lastServerSequence}</span>
              {snapshot.status === 'active' ? <button type="button" className="text-button danger" onClick={leave}><LogOut />Leave match</button> : null}
            </footer>
          </section>
        </>
      ) : null}

      {isTerminal ? (
        <section className="maze-result" aria-labelledby="maze-result-title">
          <CheckCircle2 />
          <span className="eyebrow">{snapshot.status === 'completed' ? 'Puzzle cleared' : 'Time expired'}</span>
          <h2 id="maze-result-title">{snapshot.status === 'completed' ? 'You broke the dependency chain.' : 'The arena closed this attempt.'}</h2>
          <dl>
            <div><dt>Successful</dt><dd>{snapshot.progress.successfulActions}</dd></div>
            <div><dt>Blocked</dt><dd>{snapshot.progress.blockedActions}</dd></div>
            <div><dt>Best combo</dt><dd>{snapshot.progress.maximumCombo}</dd></div>
            <div><dt>Network</dt><dd>{lastLatencyMs} ms</dd></div>
          </dl>
          <div className="maze-result-actions">
            <button type="button" className="button" onClick={() => void start()}><RefreshCw />Fresh puzzle</button>
            {replayMetadata && initialSnapshot ? (
              <button type="button" className="button secondary" onClick={() => setPhase('replay')}><ShieldCheck />Watch verified replay</button>
            ) : (
              <button type="button" className="button secondary" onClick={() => void loadReplay(snapshot.matchId)}><RefreshCw />Check replay</button>
            )}
          </div>
          {replayStatus ? <p role="status">{replayStatus}</p> : null}
        </section>
      ) : null}

      {phase === 'replay' && replayMetadata && initialSnapshot ? (
        <MazeReplayViewer
          initial={initialSnapshot}
          actions={replayActions}
          metadata={replayMetadata}
          reducedMotion={reducedMotion}
        />
      ) : null}
    </main>
  )
}
