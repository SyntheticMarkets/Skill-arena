'use client'

import { KeyboardEvent, PointerEvent, WheelEvent, useId, useMemo, useRef, useState } from 'react'

export type LineState = 'ready' | 'removed' | 'blocked' | 'exiting'
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
  const colors = ['#23d3ff', '#7c5cff', '#25d87b', '#f6c453', '#ff4f9a', '#f8fafc']
  return colors[index % colors.length]
}

function motionStyle(line: ArrowLine, index: number) {
  const [dirX, dirY] = directionVector(line.direction)
  return {
    animationDelay: `${index * 70}ms`,
    ['--line-color' as string]: lineColor(index),
    ['--move-x' as string]: `${dirX * 128}px`,
    ['--move-y' as string]: `${dirY * 128}px`,
    ['--block-x' as string]: `${dirX * 10}px`,
    ['--block-y' as string]: `${dirY * 10}px`,
  }
}

function clamp(value: number, min: number, max: number) {
  return Math.min(max, Math.max(min, value))
}

export function escapeBlocker(lines: ArrowLine[], lineId: string) {
  const idx = lines.findIndex((line) => line.id === lineId)
  if (idx < 0 || lines[idx].state === 'removed') return ''
  const line = lines[idx]
  const head = lineHead(line)
  const [dirX, dirY] = directionVector(line.direction)
  const rayEnd = boardExitPoint(head, dirX, dirY)
  let blocker = ''
  let bestDistance = Number.POSITIVE_INFINITY
  lines.forEach((other, otherIdx) => {
    if (otherIdx === idx || other.state === 'removed') return
    const hit = rayHitsLine(head, rayEnd, dirX, dirY, other)
    if (hit != null && hit < bestDistance) {
      bestDistance = hit
      blocker = other.id
    }
  })
  return blocker
}

function lineHead(line: ArrowLine) {
  const points = line.points && line.points.length > 1 ? line.points : fallbackPoints(line)
  return points[points.length - 1]
}

function lineGeometry(line: ArrowLine) {
  return line.points && line.points.length > 1 ? line.points : fallbackPoints(line)
}

function directionVector(direction: Direction): [number, number] {
  switch (direction) {
    case 'right':
      return [1, 0]
    case 'left':
      return [-1, 0]
    case 'up':
      return [0, -1]
    case 'down':
      return [0, 1]
  }
}

function boardExitPoint(head: { x: number; y: number }, dirX: number, dirY: number) {
  if (dirX > 0) return { x: 101, y: head.y }
  if (dirX < 0) return { x: -1, y: head.y }
  if (dirY > 0) return { x: head.x, y: 101 }
  return { x: head.x, y: -1 }
}

function rayHitsLine(
  rayStart: { x: number; y: number },
  rayEnd: { x: number; y: number },
  dirX: number,
  dirY: number,
  line: ArrowLine,
) {
  const points = lineGeometry(line)
  for (let index = 0; index < points.length - 1; index++) {
    const hit = rayHitsSegment(rayStart, rayEnd, dirX, dirY, points[index], points[index + 1])
    if (hit != null) return hit
  }
  return null
}

function rayHitsSegment(
  rayStart: { x: number; y: number },
  rayEnd: { x: number; y: number },
  dirX: number,
  dirY: number,
  segA: { x: number; y: number },
  segB: { x: number; y: number },
) {
  const eps = 0.001
  if (Math.abs(segA.x - segB.x) < eps) {
    const x = segA.x
    const minY = Math.min(segA.y, segB.y)
    const maxY = Math.max(segA.y, segB.y)
    if (dirX === 0) {
      if (Math.abs(rayStart.x - x) >= eps) return null
      for (const y of [minY, maxY]) {
        const hit = { x, y }
        if (pointAhead(rayStart, dirX, dirY, hit) && pointOnRayBounds(rayStart, rayEnd, hit)) return Math.abs(y - rayStart.y)
      }
      return null
    }
    if (rayStart.y < minY - eps || rayStart.y > maxY + eps) return null
    const hit = { x, y: rayStart.y }
    return pointAhead(rayStart, dirX, dirY, hit) && pointOnRayBounds(rayStart, rayEnd, hit) ? Math.abs(x - rayStart.x) : null
  }
  if (Math.abs(segA.y - segB.y) < eps) {
    const y = segA.y
    const minX = Math.min(segA.x, segB.x)
    const maxX = Math.max(segA.x, segB.x)
    if (dirY === 0) {
      if (Math.abs(rayStart.y - y) >= eps) return null
      for (const x of [minX, maxX]) {
        const hit = { x, y }
        if (pointAhead(rayStart, dirX, dirY, hit) && pointOnRayBounds(rayStart, rayEnd, hit)) return Math.abs(x - rayStart.x)
      }
      return null
    }
    if (rayStart.x < minX - eps || rayStart.x > maxX + eps) return null
    const hit = { x: rayStart.x, y }
    return pointAhead(rayStart, dirX, dirY, hit) && pointOnRayBounds(rayStart, rayEnd, hit) ? Math.abs(y - rayStart.y) : null
  }
  return null
}

function pointAhead(start: { x: number; y: number }, dirX: number, dirY: number, point: { x: number; y: number }) {
  const eps = 0.001
  return (dirX > 0 && point.x > start.x + eps) ||
    (dirX < 0 && point.x < start.x - eps) ||
    (dirY > 0 && point.y > start.y + eps) ||
    (dirY < 0 && point.y < start.y - eps)
}

function pointOnRayBounds(start: { x: number; y: number }, end: { x: number; y: number }, point: { x: number; y: number }) {
  const eps = 0.001
  return point.x >= Math.min(start.x, end.x) - eps &&
    point.x <= Math.max(start.x, end.x) + eps &&
    point.y >= Math.min(start.y, end.y) - eps &&
    point.y <= Math.max(start.y, end.y) + eps
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
        <filter id={`${markerId}-glow`} x="-40%" y="-40%" width="180%" height="180%">
          <feGaussianBlur stdDeviation="1.2" result="blur" />
          <feMerge>
            <feMergeNode in="blur" />
            <feMergeNode in="SourceGraphic" />
          </feMerge>
        </filter>
        <marker id={`${markerId}-arrow`} markerWidth="4.6" markerHeight="4.6" refX="4.1" refY="2.3" orient="auto" markerUnits="userSpaceOnUse">
          <path className="line-arrowhead" d="M0.15,0.2 L4.35,2.3 L0.15,4.4 C0.95,3.15 1.12,1.45 0.15,0.2 Z" fill="currentColor" />
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
            style={motionStyle(line, index)}
          >
            <path className="line-hitbox" d={path} />
            <path className="line-shadow" d={path} />
            <path className="line-route" d={path} markerEnd={`url(#${markerId}-arrow)`} filter={`url(#${markerId}-glow)`} />
            <path className="line-highlight" d={path} />
          </g>
        )
      })}
    </svg>
  )
}
