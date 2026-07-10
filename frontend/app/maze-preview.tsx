'use client'

import { KeyboardEvent, PointerEvent, WheelEvent, useId, useMemo, useRef, useState } from 'react'

export type LineState = 'ready' | 'removed' | 'blocked'
export type Direction = 'up' | 'down' | 'left' | 'right'

export type ArrowLine = {
  id: string
  x: number
  y: number
  length: number
  direction: Direction
  points?: Array<{ x: number; y: number }>
  dependsOn?: string[]
  blocked?: boolean
  removed?: boolean
  state?: LineState
}

export function normalizeLines(items: ArrowLine[] | undefined): ArrowLine[] {
  const normalized = (items ?? []).map((line) => ({
    ...line,
    points: line.points ?? fallbackPoints(line),
    dependsOn: line.dependsOn ?? [],
    state: line.removed ? 'removed' as LineState : line.blocked ? 'blocked' as LineState : 'ready' as LineState,
  }))
  return normalized
}

function fallbackPoints(line: ArrowLine) {
  const end = {
    x: line.direction === 'left' ? line.x - line.length : line.direction === 'right' ? line.x + line.length : line.x,
    y: line.direction === 'up' ? line.y - line.length : line.direction === 'down' ? line.y + line.length : line.y,
  }
  return [{ x: line.x, y: line.y }, end]
}

function roundedPath(points: Array<{ x: number; y: number }>) {
  if (points.length === 0) return ''
  if (points.length === 1) return `M ${points[0].x} ${points[0].y}`
  const radius = 1.6
  let path = `M ${points[0].x.toFixed(2)} ${points[0].y.toFixed(2)}`
  for (let index = 1; index < points.length; index++) {
    const previous = points[index - 1]
    const current = points[index]
    const next = points[index + 1]
    if (!next) {
      path += ` L ${current.x.toFixed(2)} ${current.y.toFixed(2)}`
      continue
    }
    const incoming = Math.hypot(current.x - previous.x, current.y - previous.y)
    const outgoing = Math.hypot(next.x - current.x, next.y - current.y)
    const bend = Math.min(radius, incoming / 2, outgoing / 2)
    const before = {
      x: current.x - ((current.x - previous.x) / Math.max(1, incoming)) * bend,
      y: current.y - ((current.y - previous.y) / Math.max(1, incoming)) * bend,
    }
    const after = {
      x: current.x + ((next.x - current.x) / Math.max(1, outgoing)) * bend,
      y: current.y + ((next.y - current.y) / Math.max(1, outgoing)) * bend,
    }
    path += ` L ${before.x.toFixed(2)} ${before.y.toFixed(2)} Q ${current.x.toFixed(2)} ${current.y.toFixed(2)} ${after.x.toFixed(2)} ${after.y.toFixed(2)}`
  }
  return path
}

function lineColor(index: number) {
  const colors = ['#19d3ff', '#8b5cf6', '#22c55e', '#facc15', '#f43f8b', '#f8fafc']
  return colors[index % colors.length]
}

function clamp(value: number, min: number, max: number) {
  return Math.min(max, Math.max(min, value))
}

export function ArrowLinePuzzle({
  lines,
  label,
  readOnly = false,
  compact = false,
  animated = false,
  onLineClick,
}: {
  lines: ArrowLine[]
  label: string
  readOnly?: boolean
  compact?: boolean
  animated?: boolean
  onLineClick?: (lineId: string) => void
}) {
  const markerId = useId().replace(/:/g, '')
  const [zoom, setZoom] = useState(1)
  const [pan, setPan] = useState({ x: 0, y: 0 })
  const dragRef = useRef<{ id: number; x: number; y: number; pan: { x: number; y: number } } | null>(null)
  const canNavigate = !compact
  const viewBox = useMemo(() => {
    if (!canNavigate) return '0 0 100 100'
    const width = 100 / zoom
    const height = 100 / zoom
    const x = clamp(pan.x, 0, 100 - width)
    const y = clamp(pan.y, 0, 100 - height)
    return `${x} ${y} ${width} ${height}`
  }, [canNavigate, pan.x, pan.y, zoom])

  function resetCamera() {
    setZoom(1)
    setPan({ x: 0, y: 0 })
  }

  function handleWheel(event: WheelEvent<SVGSVGElement>) {
    if (!canNavigate) return
    event.preventDefault()
    setZoom((current) => clamp(current + (event.deltaY < 0 ? 0.12 : -0.12), 1, 2.8))
  }

  function pointerDown(event: PointerEvent<SVGSVGElement>) {
    if (!canNavigate || event.button !== 0) return
    dragRef.current = { id: event.pointerId, x: event.clientX, y: event.clientY, pan }
    event.currentTarget.setPointerCapture(event.pointerId)
  }

  function pointerMove(event: PointerEvent<SVGSVGElement>) {
    const drag = dragRef.current
    if (!canNavigate || !drag) return
    const scale = 100 / (event.currentTarget.clientWidth || 1) / zoom
    setPan({
      x: drag.pan.x - (event.clientX - drag.x) * scale,
      y: drag.pan.y - (event.clientY - drag.y) * scale,
    })
  }

  function pointerUp(event: PointerEvent<SVGSVGElement>) {
    if (dragRef.current?.id === event.pointerId) {
      dragRef.current = null
    }
  }

  function lineKeyDown(event: KeyboardEvent<SVGGElement>, lineId: string) {
    if (readOnly) return
    if (event.key !== 'Enter' && event.key !== ' ') return
    event.preventDefault()
    onLineClick?.(lineId)
  }

  return (
    <svg
      className={`${compact ? 'line-puzzle landing-line-puzzle' : 'line-puzzle'}${animated ? ' animated' : ''}`}
      viewBox={viewBox}
      role="img"
      aria-label={label}
      onWheel={handleWheel}
      onPointerDown={pointerDown}
      onPointerMove={pointerMove}
      onPointerUp={pointerUp}
      onPointerCancel={pointerUp}
      onDoubleClick={resetCamera}
    >
      <defs>
        <marker id={`${markerId}-arrow`} markerWidth="3.4" markerHeight="3.4" refX="3.1" refY="1.7" orient="auto" markerUnits="userSpaceOnUse">
          <path d="M0,0 L3.4,1.7 L0,3.4 L0.8,1.7 Z" fill="currentColor" />
        </marker>
      </defs>
      {lines.map((line, index) => {
        const state = line.state ?? 'ready'
        const path = roundedPath(line.points && line.points.length > 1 ? line.points : fallbackPoints(line))
        return (
          <g
            key={line.id}
            className={`arrow-line ${state}`}
            onClick={readOnly ? undefined : () => onLineClick?.(line.id)}
            onKeyDown={readOnly ? undefined : (event) => lineKeyDown(event, line.id)}
            tabIndex={readOnly ? undefined : 0}
            role={readOnly ? undefined : 'button'}
            style={{ animationDelay: `${index * 70}ms`, ['--line-color' as string]: lineColor(index) }}
          >
            <path className="line-hitbox" d={path} />
            <path className="line-route" d={path} markerEnd={`url(#${markerId}-arrow)`} />
          </g>
        )
      })}
    </svg>
  )
}
