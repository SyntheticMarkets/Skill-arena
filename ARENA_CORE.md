# Arena Core Backend Boundary

Skill Arena treats every client as compromised. Clients render state and submit player intent; the backend authenticates, authorizes, validates, applies, scores, signs, and settles.

## Core Rule

Games plug into Arena Core. Games do not directly mutate wallets, leaderboards, progression, trust, tournaments, challenges, or rewards.

## Backend Flow

1. Validate JWT and derive the actor user ID server-side.
2. Load the authoritative server session.
3. Verify session ownership or match participation.
4. Resolve the registered game module.
5. Submit client intent only, such as `click line_12`.
6. Game module validates and applies the action against server state.
7. Arena Core/store settles wallet, XP, trust, replay, challenges, tournaments, and audit.

## Game Module Contract

Backend modules implement `internal/arena/core.GameModule`.

Current module:

- `maze_arena` in `backend/internal/games/maze`

Future modules should implement the same contract without calling wallet, payment, leaderboard, tournament, or challenge services directly.

## Replay Rule

Replays store seed, rules version, game version, action stream, timing, and server signature. They do not trust client-provided board state or outcome.

## Seed Rules

- Practice: one unique seed per player/session.
- PvP: one shared seed per match, independent board state per player.
- Tournament: one shared seed per bracket match.
- Daily challenge: one shared seed per day.

## Client Must Never Submit

- score
- winner
- rewards
- coins
- XP
- trust score
- completion state
- wallet IDs
- difficulty overrides
- seeds
- replay result

Those are server-owned values.
