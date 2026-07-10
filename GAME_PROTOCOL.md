# Game Protocol

## Game Registry

Game modules expose:

- Metadata contract
- Renderer contract
- Replay contract
- Tournament contract

Maze Arena is registered as Game #1. Future games should use the same contracts.

## Maze Arena Session

Start:

1. Client calls `/api/v1/games/start`.
2. Backend validates user, wallet, stake, mode, and difficulty.
3. Backend locks stake when required.
4. Backend builds difficulty profile.
5. Backend derives HMAC seed.
6. Backend generates line puzzle.
7. Session enters ready state.

Finish:

1. Client submits clicked line IDs as moves.
2. Backend validates dependencies.
3. Backend marks success/blocked clicks.
4. Backend determines win/loss.
5. Backend settles reward/loss.
6. Backend records progression, achievements, metrics, and replay metadata.

## Puzzle Generation

Seed derivation input:

- Purpose
- Match ID/session ID
- Player ID
- Nonce
- Difficulty profile
- Puzzle version

Output:

- Puzzle seed
- Generation nonce
- Generation hash

Line puzzle metadata:

- ID
- Direction
- Coordinates
- Routed points
- Dependencies
- Blocked/removed state

## Difficulty

Difficulty profile includes:

- Rating
- Line count
- Dependency depth
- Branching factor
- False-route rate
- Dead-end factor
- Cross dependencies
- Noise factor

## PvP Protocol

Join:

1. Client calls `/api/v1/pvp/join`.
2. Backend checks trust score and wallet eligibility.
3. Stake is locked.
4. Redis lock coordinates compatible queue access.
5. Backend matches two compatible players.
6. Backend derives separate puzzle seeds.
7. Match becomes active.

Progress:

1. Client reports progress to `/api/v1/pvp/progress`.
2. Backend stores authoritative progress.
3. Opponent UI should read match detail from backend state.

Submit:

1. Client submits moves to `/api/v1/pvp/submit`.
2. Backend validates route/clicks.
3. Backend settles winner, prize pool, platform fee, and progression.

## Replay Protocol

Replay report includes:

- Puzzle seed
- Generation nonce
- Generation hash
- Difficulty profile
- Puzzle version
- Rules version
- Replay version
- Lines/clicks/moves
- Playback events
- Integrity flags
- HMAC signature

Replay validation regenerates the puzzle from seed and profile. A mismatch flags the replay.
