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
- `test_arena` in `backend/internal/games/testarena` for modularity tests only

Future modules should implement the same contract without calling wallet, payment, leaderboard, tournament, or challenge services directly.

## Manifests And Capabilities

Every game module owns a `module.json` manifest.

The manifest declares:

- game ID
- name and description
- version
- rules version
- replay version
- protocol version
- renderer key
- supported modes
- minimum and maximum players
- average match time
- capability flags

Arena Core never assumes a game supports PvP, replay, tournaments, spectator mode, AI, teams, or coins. The module manifest declares that support.

## Contexts

Game modules receive one authoritative context object.

`SessionContext` carries session, actor, wallet, season, league, trust, house, tournament, practice, and configuration data.

`ActionContext` carries authenticated actor, session, action stream, sequence number, replay position, latency, and server receive time.

The client cannot override context values. Arena Core builds them from JWT-authenticated state.

## Event Bus

Arena Core formalizes platform events. Games emit events; platform systems consume them.

Examples:

- `practice_started`
- `puzzle_generated`
- `action_accepted`
- `action_rejected`
- `puzzle_solved`
- `rewards_calculated`
- `wallet_credited`
- `challenge_updated`
- `xp_granted`
- `replay_signed`
- `statistics_updated`
- `notification_sent`

Games emit events but never settle wallet, progression, tournaments, challenges, or trust directly.

Live events flow through the Session Gateway: one authenticated WebSocket per logged-in client. REST remains the interface for account, wallet, settings, and security request/response flows.

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

## Freeze Rule

Arena Core v1.0 is an extension boundary, not a rewrite target. Future work should add modules and capabilities through the existing interfaces.
