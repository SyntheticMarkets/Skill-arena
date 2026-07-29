import { MazeArena } from '../../MazeArena'

export default async function MazeReplayPage({
  params,
}: {
  params: Promise<{ matchId: string }>
}) {
  const { matchId } = await params
  return <MazeArena replayMatchId={matchId} />
}

