# Game Rules

Status: Draft for review

Sprint: 1 - Product Identity & Design Foundation

This is the product rulebook for Skill Arena games.

It is not backend architecture and it is not frontend design.

Every game should eventually have its own rules section that defines how players compete, how outcomes are verified, and how spectators, replays, latency, scoring, and disputes work.

## Rulebook Purpose

Game rules exist so players, designers, developers, support staff, treasury, and admins all understand the same competitive contract.

Every game rule must answer:

- Why does this rule exist?
- What does the player see?
- What does the opponent see?
- What does the spectator see?
- How is the rule enforced?
- How is the result verified?
- What happens when something goes wrong?

## Shared Game Rules

All Skill Arena games should define:

- Legal actions.
- Invalid or blocked actions.
- Win condition.
- Loss condition.
- Draw condition.
- Scoring.
- Difficulty.
- Match timing.
- Replay requirements.
- Spectator visibility.
- Disconnect handling.
- Latency handling.
- Dispute rules.
- Rules version.
- Seed or generation hash when applicable.
- Integrity hash when applicable.

## Maze Arena Rules

Maze Arena is Skill Arena Game 1.

### Legal Move

A legal move is an action that:

- Is allowed by the current puzzle state.
- Follows the active rule set.
- Does not violate dependency requirements.
- Occurs while the match timer is active.
- Is accepted by the authoritative game state.

Player-facing explanation:

- "Move accepted."
- "Path advanced."
- "Dependency cleared."

### Blocked Move

A blocked move is an attempted action that cannot change the puzzle state.

Common reasons:

- Required dependency has not been cleared.
- Target node or path is locked.
- Move would violate the puzzle route rules.
- Move arrives after match completion or timeout.
- Move conflicts with authoritative game state.

Player-facing behavior:

- Explain why the move was blocked.
- Show the relevant dependency or rule when possible.
- Do not punish honest exploration unless the game mode explicitly scores penalties.

### Dependency

A dependency is a rule relationship where one route, node, or action must be completed before another becomes valid.

Dependencies create:

- Strategic planning.
- Puzzle readability requirements.
- Difficulty scaling.
- Replay learning moments.

Dependencies must be visible enough for skilled players to reason about them.

### Difficulty

Difficulty may be created through:

- Number of dependencies.
- Dependency depth.
- Cross dependencies.
- Dead ends.
- Fake paths.
- Route length.
- Timing pressure.
- Visual complexity.
- Required planning depth.

Difficulty should not be created through unreadable UI, unclear rules, hidden information, or inconsistent interactions.

### APCE

APCE should be treated as a competitive evaluation layer.

It may influence:

- Puzzle calibration.
- Difficulty profile.
- Fairness checks.
- Replay verification.
- Challenge validation.
- Ranked suitability.

APCE must be explainable at the product level:

- What was evaluated?
- Why did the puzzle qualify?
- What version of rules was used?
- How can the result be verified later?

### Seeds And Generation

Generated puzzles must be reproducible.

Required rule data:

- Seed.
- Generation hash.
- Puzzle hash.
- Difficulty profile.
- Rules version.
- Game version.

The product requirement is that a replay can be verified years later against the same rules and generation profile.

### Replay Verification

A valid Maze Arena replay must preserve:

- Puzzle identity.
- Rules version.
- Seed and generation hash.
- Player actions.
- Timing.
- Blocked move events.
- Successful move events.
- Completion state.
- Score inputs.
- Integrity signature.

A replay becomes invalid or under review when:

- Required hashes do not match.
- Timing cannot be verified.
- Action sequence violates the rule set.
- Replay signature is missing or invalid.
- Puzzle cannot be regenerated.
- Match state conflicts with authoritative records.

### Scoring

Scoring should be understandable after the match.

Potential scoring inputs:

- Completion.
- Time.
- Move efficiency.
- Blocked move count when applicable.
- Dependency efficiency.
- Difficulty rating.
- Ranked or tournament modifiers.

The player must understand why their score changed.

### Disconnects

Disconnect rules must be explicit.

The rulebook should define:

- Grace period.
- Reconnect window.
- Whether the timer continues.
- What the opponent sees.
- What spectators see.
- When the match is forfeited.
- How replay and dispute evidence are preserved.

### Draws

Draw conditions must be defined before ranked or tournament launch.

Possible draw causes:

- Both players complete within the same scoring tolerance.
- Both players time out with equivalent progress.
- Match is invalidated by platform failure.

Draw handling must explain:

- Rank impact.
- Wallet impact.
- Tournament impact.
- Replay status.

### Spectator Visibility

Spectators may see:

- Match timer.
- Player progress.
- Completion percentage.
- Replay after completion.
- Public score data.

Spectators should not see hidden information that would give active competitors an unfair advantage.

### Latency Spikes

Latency handling must protect competitive integrity.

Rules should define:

- What is client-side feedback only.
- What is server-authoritative.
- How delayed moves are accepted or rejected.
- What happens during severe latency.
- How disputes are reviewed.

## Future Game Rulebooks

Future games should receive their own sections:

- Memory Arena.
- Logic Arena.
- Pattern Arena.
- Reaction Arena.
- Any future game.

Each should define the same product-level contract before implementation.

## Approval Questions

1. Can a player understand what counts as a valid result?
2. Can support explain why a match was won, lost, blocked, invalid, or disputed?
3. Can replay verification be explained without backend code?
4. Can spectators understand what they are allowed to see?
5. Can future games follow the same rulebook structure?
