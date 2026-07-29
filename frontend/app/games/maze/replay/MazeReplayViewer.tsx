'use client'

import { CheckCircle2, Pause, Play, RotateCcw, ShieldCheck, SkipForward } from 'lucide-react'
import { useEffect, useMemo, useRef, useState } from 'react'
import type { MazeAnimation } from '../animation/transitions'
import { animationDuration, rendererTiming } from '../animation/transitions'
import type { MazeReplayAction, MazeSnapshot } from '../protocol/schemas'
import { MazeRenderer } from '../renderer/MazeRenderer'

export type VerifiedReplayMetadata = {
  id: string
  matchId: string
  replayVersion: string
  eventCount: number
  eventRootHash: string
  status: string
  createdAt: string
}

export function MazeReplayViewer({
  initial,
  actions,
  metadata,
  reducedMotion,
}: {
  initial: MazeSnapshot
  actions: MazeReplayAction[]
  metadata: VerifiedReplayMetadata
  reducedMotion: boolean
}) {
  const [position, setPosition] = useState(0)
  const [playing, setPlaying] = useState(false)
  const [animation, setAnimation] = useState<MazeAnimation | null>(null)
  const timer = useRef<number | null>(null)

  const snapshot = useMemo(() => {
    const completed = actions.slice(0, position)
    const removedIds = completed
      .filter((action) => action.accepted)
      .map((action) => action.presentation.arrowId)
    const latest = completed[completed.length - 1]
    return {
      ...initial,
      removedIds,
      progress: latest?.progress ?? {
        ...initial.progress,
        removed: 0,
        remaining: initial.arrows.length,
        successfulActions: 0,
        blockedActions: 0,
        currentCombo: 0,
        maximumCombo: 0,
        completionBps: 0,
        complete: false,
      },
      status: latest?.progress.complete ? 'completed' as const : 'active' as const,
    }
  }, [actions, initial, position])

  useEffect(() => () => {
    if (timer.current) window.clearTimeout(timer.current)
  }, [])

  function animateAt(index: number, continuePlaying: boolean) {
    if (index >= actions.length) {
      setPlaying(false)
      return
    }
    const action = actions[index]
    const nextAnimation: MazeAnimation = {
      id: `replay-${action.sequence}-${index}`,
      presentation: action.presentation,
      kind: action.accepted ? 'accepted' : 'blocked',
    }
    setAnimation(nextAnimation)
    const duration = animationDuration(nextAnimation, reducedMotion)
    timer.current = window.setTimeout(() => {
      setAnimation(null)
      const next = index + 1
      setPosition(next)
      if (continuePlaying && next < actions.length) {
        timer.current = window.setTimeout(
          () => animateAt(next, true),
          rendererTiming.replayGapMs,
        )
      } else {
        setPlaying(false)
      }
    }, duration)
  }

  function step() {
    if (animation || position >= actions.length) return
    animateAt(position, false)
  }

  function togglePlayback() {
    if (playing) {
      if (timer.current) window.clearTimeout(timer.current)
      timer.current = null
      setPlaying(false)
      setAnimation(null)
      return
    }
    setPlaying(true)
    animateAt(position, true)
  }

  function restart() {
    if (timer.current) window.clearTimeout(timer.current)
    timer.current = null
    setPlaying(false)
    setAnimation(null)
    setPosition(0)
  }

  return (
    <section className="maze-replay" aria-labelledby="maze-replay-title">
      <header className="maze-replay-heading">
        <div>
          <span className="eyebrow">Verified replay</span>
          <h2 id="maze-replay-title">Authoritative match reconstruction</h2>
          <p>Playback follows committed server events. It does not recalculate moves in the browser.</p>
        </div>
        <span className="maze-replay-proof">
          <ShieldCheck aria-hidden="true" />
          {metadata.status}
        </span>
      </header>
      <MazeRenderer
        snapshot={snapshot}
        animation={animation}
        reducedMotion={reducedMotion}
        disabled
        onArrow={() => undefined}
      />
      <div className="maze-replay-controls">
        <button
          type="button"
          className="icon-button"
          onClick={togglePlayback}
          disabled={position >= actions.length}
          aria-label={playing ? 'Pause replay' : 'Play replay'}
          title={playing ? 'Pause replay' : 'Play replay'}
        >
          {playing ? <Pause /> : <Play />}
        </button>
        <button
          type="button"
          className="icon-button"
          onClick={step}
          disabled={playing || Boolean(animation) || position >= actions.length}
          aria-label="Next replay action"
          title="Next replay action"
        >
          <SkipForward />
        </button>
        <button
          type="button"
          className="icon-button"
          onClick={restart}
          aria-label="Restart replay"
          title="Restart replay"
        >
          <RotateCcw />
        </button>
        <span>{position} / {actions.length} actions</span>
        {position === actions.length ? <strong><CheckCircle2 />Playback complete</strong> : null}
      </div>
      <dl className="maze-replay-integrity">
        <div><dt>Replay</dt><dd>{metadata.id}</dd></div>
        <div><dt>Version</dt><dd>{metadata.replayVersion}</dd></div>
        <div><dt>Event root</dt><dd>{metadata.eventRootHash.slice(0, 16)}</dd></div>
        <div><dt>Recorded</dt><dd>{new Date(metadata.createdAt).toLocaleString()}</dd></div>
      </dl>
    </section>
  )
}
