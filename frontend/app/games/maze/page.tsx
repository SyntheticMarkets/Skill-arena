import { notFound } from 'next/navigation'
import { resolveRenderer } from '../registry/catalog'

export default function MazeArenaPage() {
  const registration = resolveRenderer('maze-arena', '1.0.0')
  if (!registration) notFound()
  const Renderer = registration.component
  return <Renderer />
}

