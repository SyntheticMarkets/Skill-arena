'use client'

import { Maximize2, ZoomIn, ZoomOut } from 'lucide-react'
import { useId } from 'react'
import type { MazeAnimation } from '../animation/transitions'
import { animationDuration, directionOffset } from '../animation/transitions'
import { useMazeCamera } from '../camera/useMazeCamera'
import type { MazeArrow, MazeSnapshot } from '../protocol/schemas'

const palette = ['#22d3ee', '#a78bfa', '#34d399', '#f6c453', '#fb7185', '#e2e8f0']

function arrowColor(id: string) {
  const numeric = Number.parseInt(id.replace(/\D/g, ''), 10)
  return palette[Number.isFinite(numeric) ? numeric % palette.length : 0]
}

function points(arrow: MazeArrow) {
  return arrow.cells
    .map((cell) => `${cell.column + 0.5},${cell.row + 0.5}`)
    .join(' ')
}

function rotation(direction: MazeArrow['direction']) {
  switch (direction) {
    case 'right':
      return 0
    case 'down':
      return 90
    case 'left':
      return 180
    case 'up':
      return -90
  }
}

function ArrowShape({
  arrow,
  animation,
  reducedMotion,
  disabled,
  onArrow,
}: {
  arrow: MazeArrow
  animation?: MazeAnimation
  reducedMotion: boolean
  disabled: boolean
  onArrow: (arrowId: string) => void
}) {
  const head = arrow.cells[arrow.cells.length - 1]
  const color = arrowColor(arrow.id)
  const distance = animation?.kind === 'accepted'
    ? animation.presentation.escapeDistance ?? 0
    : animation?.presentation.approachDistance ?? 0
  const offset = animation
    ? directionOffset(animation.presentation.direction, distance)
    : { x: 0, y: 0 }
  const duration = animation ? animationDuration(animation, reducedMotion) : 0
  const values = animation?.kind === 'blocked'
    ? `0 0;${offset.x} ${offset.y};${offset.x * 0.9} ${offset.y * 0.9};0 0`
    : `0 0;${offset.x * 0.14} ${offset.y * 0.14};${offset.x} ${offset.y}`

  return (
    <g
      className={`maze-arrow${animation ? ` is-${animation.kind}` : ''}`}
      style={{ color }}
      data-arrow-id={arrow.id}
      onClick={disabled ? undefined : () => onArrow(arrow.id)}
    >
      {animation && !reducedMotion ? (
        <animateTransform
          key={animation.id}
          attributeName="transform"
          type="translate"
          values={values}
          keyTimes={animation.kind === 'blocked' ? '0;0.42;0.5;1' : '0;0.18;1'}
          calcMode="spline"
          keySplines={animation.kind === 'blocked'
            ? '0.2 0.8 0.3 1;0.2 0.8 0.3 1;0.4 0 0.2 1'
            : '0.3 0 0.2 1;0.2 0.7 0.2 1'}
          dur={`${duration}ms`}
          fill={animation.kind === 'accepted' ? 'freeze' : 'remove'}
        />
      ) : null}
      <polyline className="maze-arrow-hitbox" points={points(arrow)} />
      <polyline className="maze-arrow-shadow" points={points(arrow)} />
      <polyline className="maze-arrow-shaft" points={points(arrow)} />
      <polyline className="maze-arrow-highlight" points={points(arrow)} />
      <path
        className="maze-arrow-head"
        d="M -0.48 -0.42 L 0.48 0 L -0.48 0.42 Q -0.22 0 -0.48 -0.42 Z"
        transform={`translate(${head.column + 0.5} ${head.row + 0.5}) rotate(${rotation(arrow.direction)})`}
      />
      {animation?.kind === 'blocked' && animation.presentation.collisionCell ? (
        <circle
          className="maze-impact"
          cx={animation.presentation.collisionCell.column + 0.5}
          cy={animation.presentation.collisionCell.row + 0.5}
          r="0.36"
        />
      ) : null}
    </g>
  )
}

export function MazeBoard({
  snapshot,
  animation,
  reducedMotion,
  disabled,
  onArrow,
}: {
  snapshot: MazeSnapshot
  animation: MazeAnimation | null
  reducedMotion: boolean
  disabled: boolean
  onArrow: (arrowId: string) => void
}) {
  const patternId = useId().replace(/:/g, '')
  const camera = useMazeCamera(snapshot.columns, snapshot.rows)
  const removed = new Set(snapshot.removedIds)
  const active = snapshot.arrows.filter(
    (arrow) => !removed.has(arrow.id) || animation?.presentation.arrowId === arrow.id,
  )

  return (
    <div className="maze-board-shell">
      <svg
        className="maze-board"
        viewBox={camera.viewBox}
        role="img"
        aria-label={`Maze board. ${snapshot.progress.remaining} of ${snapshot.progress.total} arrows remain.`}
        onWheel={camera.onWheel}
        onPointerDown={camera.onPointerDown}
        onPointerMove={camera.onPointerMove}
        onPointerUp={camera.onPointerUp}
        onPointerCancel={camera.onPointerUp}
      >
        <defs>
          <pattern id={patternId} width="1" height="1" patternUnits="userSpaceOnUse">
            <path d="M 1 0 L 0 0 0 1" className="maze-grid-line" />
          </pattern>
          <filter id={`${patternId}-glow`} x="-35%" y="-35%" width="170%" height="170%">
            <feGaussianBlur stdDeviation="0.11" result="blur" />
            <feMerge>
              <feMergeNode in="blur" />
              <feMergeNode in="SourceGraphic" />
            </feMerge>
          </filter>
        </defs>
        <rect width={snapshot.columns} height={snapshot.rows} className="maze-board-floor" />
        <rect
          width={snapshot.columns}
          height={snapshot.rows}
          fill={`url(#${patternId})`}
          className="maze-board-grid"
        />
        <g filter={`url(#${patternId}-glow)`}>
          {active.map((arrow) => (
            <ArrowShape
              key={arrow.id}
              arrow={arrow}
              animation={animation?.presentation.arrowId === arrow.id ? animation : undefined}
              reducedMotion={reducedMotion}
              disabled={disabled}
              onArrow={(arrowId) => {
                if (!camera.consumeSuppressedClick()) onArrow(arrowId)
              }}
            />
          ))}
        </g>
      </svg>
      <div className="maze-camera" aria-label="Board camera controls">
        <button type="button" onClick={camera.zoomOut} disabled={camera.zoom <= 1} aria-label="Zoom out" title="Zoom out"><ZoomOut /></button>
        <output aria-label="Board zoom">{Math.round(camera.zoom * 100)}%</output>
        <button type="button" onClick={camera.zoomIn} disabled={camera.zoom >= 3} aria-label="Zoom in" title="Zoom in"><ZoomIn /></button>
        <button type="button" onClick={camera.reset} aria-label="Fit board" title="Fit board"><Maximize2 /></button>
      </div>
    </div>
  )
}
