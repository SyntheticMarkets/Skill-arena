'use client'

import { PointerEvent, WheelEvent, useMemo, useRef, useState } from 'react'

function clamp(value: number, minimum: number, maximum: number) {
  return Math.min(maximum, Math.max(minimum, value))
}

export function useMazeCamera(columns: number, rows: number) {
  const [zoom, setZoom] = useState(1)
  const [center, setCenter] = useState({ x: columns / 2, y: rows / 2 })
  const drag = useRef<{
    pointerId: number
    x: number
    y: number
    center: { x: number; y: number }
    moved: boolean
  } | null>(null)
  const suppressClick = useRef(false)

  const viewBox = useMemo(() => {
    const width = columns / zoom
    const height = rows / zoom
    const x = clamp(center.x - width / 2, 0, Math.max(0, columns - width))
    const y = clamp(center.y - height / 2, 0, Math.max(0, rows - height))
    return `${x} ${y} ${width} ${height}`
  }, [center.x, center.y, columns, rows, zoom])

  function changeZoom(next: number) {
    setZoom(clamp(next, 1, 3))
  }

  function reset() {
    setZoom(1)
    setCenter({ x: columns / 2, y: rows / 2 })
  }

  function onWheel(event: WheelEvent<SVGSVGElement>) {
    event.preventDefault()
    changeZoom(zoom + (event.deltaY < 0 ? 0.18 : -0.18))
  }

  function onPointerDown(event: PointerEvent<SVGSVGElement>) {
    if (event.button !== 0) return
    drag.current = {
      pointerId: event.pointerId,
      x: event.clientX,
      y: event.clientY,
      center,
      moved: false,
    }
    event.currentTarget.setPointerCapture(event.pointerId)
  }

  function onPointerMove(event: PointerEvent<SVGSVGElement>) {
    const active = drag.current
    if (!active || active.pointerId !== event.pointerId) return
    const deltaX = event.clientX - active.x
    const deltaY = event.clientY - active.y
    if (Math.hypot(deltaX, deltaY) > 5) active.moved = true
    const bounds = event.currentTarget.getBoundingClientRect()
    const unitsX = columns / zoom / Math.max(1, bounds.width)
    const unitsY = rows / zoom / Math.max(1, bounds.height)
    setCenter({
      x: clamp(active.center.x - deltaX * unitsX, 0, columns),
      y: clamp(active.center.y - deltaY * unitsY, 0, rows),
    })
  }

  function onPointerUp(event: PointerEvent<SVGSVGElement>) {
    if (drag.current?.pointerId !== event.pointerId) return
    suppressClick.current = drag.current.moved
    drag.current = null
  }

  function consumeSuppressedClick() {
    if (!suppressClick.current) return false
    suppressClick.current = false
    return true
  }

  return {
    viewBox,
    zoom,
    zoomIn: () => changeZoom(zoom + 0.25),
    zoomOut: () => changeZoom(zoom - 0.25),
    reset,
    onWheel,
    onPointerDown,
    onPointerMove,
    onPointerUp,
    consumeSuppressedClick,
  }
}

