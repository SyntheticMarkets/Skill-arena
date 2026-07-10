# Platform Language

Status: Draft for review

Sprint: 1 - Product Identity & Design Foundation

This document defines the words Skill Arena uses across buttons, navigation, notifications, errors, tooltips, emails, wallet states, match summaries, and admin surfaces.

Language is part of the product identity.

## Language Principles

Skill Arena language should be:

- Clear.
- Competitive.
- Trustworthy.
- Precise.
- Calm under pressure.
- Educational.

Avoid:

- Casino language.
- Fake urgency.
- Childish phrasing.
- Vague errors.
- Generic CRUD labels when a product-specific term is clearer.

## People

Preferred terms:

- Player.
- Competitor.
- Opponent.
- Spectator.
- Admin.
- Treasury reviewer.
- Support reviewer.

Use `player` for general product language.

Use `competitor` when emphasizing identity, ranked play, tournaments, or arena participation.

Avoid:

- User, except in admin or technical contexts.
- Member, unless a future membership product exists.
- Customer, except support or compliance contexts.

## Core Actions

Preferred player-facing language:

| Intent | Preferred Label | Avoid |
|---|---|---|
| Start non-live play | Practice | Demo |
| Learn a skill | Training | Tutorial when it sounds passive |
| Enter competitive queue | Find Match | Submit, Start Process |
| Join ranked queue | Enter Ranked | Queue Up when tone feels casual |
| Start match | Enter Arena | Play Now when context needs more weight |
| Repeat match | Rematch | Try Again when ranked stakes apply |
| Watch replay | Review Replay | Watch Recording |
| Learn from match | View Insights | See Details |
| Deposit funds | Deposit | Add Money when precision is needed |
| Withdraw funds | Withdraw | Cash Out |
| Verify identity | Complete Verification | KYC when player-facing |

## Practice And Live

Player-facing terms:

- Practice.
- Training.
- Daily Challenge.
- Skill Calibration.
- Live Competition.
- Ranked.
- Tournament.

Internal/backend terms may still use `demo` where already established, but the player-facing product should prefer `Practice` or `Training`.

Use:

- "Practice balance."
- "Live balance."
- "Practice match."
- "Live competition."

Avoid:

- "Demo account."
- "Demo match" in player-facing UI.
- "Real money mode" when `Live Competition` is clearer.

## Wallet Language

Preferred terms:

- Available.
- Locked.
- Pending.
- Settled.
- Rejected.
- Under review.
- Treasury approval.
- Provider confirmation.
- Ledger complete.
- Statement.

Wallet language must explain state and next step.

Example:

- "Deposit settled. Your live balance is now available for competition."

Avoid:

- "Payment successful" without balance context.
- "Processing" without stage.
- "Cash out."
- "Funds disappeared."

## Match Language

Preferred terms:

- Match found.
- Enter Arena.
- Move accepted.
- Move blocked.
- Dependency active.
- Replay verified.
- Match under review.
- Result confirmed.
- Victory.
- Defeat.

Defeat language should be educational, not final.

Example:

- "Defeat confirmed. Your replay shows two blocked moves on unresolved dependencies."

## Error Language

Every error should explain:

- What happened.
- Why it happened when known.
- What the player can do next.

Bad:

- "Error."

Good:

- "Replay verification failed. The match remains under review while integrity checks complete."

## Approval Questions

1. Do the terms sound like Skill Arena?
2. Are player-facing labels consistent?
3. Does language teach instead of merely report?
4. Are financial terms precise enough for a real-money platform?
5. Is internal terminology separated from player-facing terminology?
