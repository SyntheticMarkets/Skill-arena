# Skill Arena Backup Strategy

## Current Local Persistence

The current development build uses JSON files under the configured data directory. Until PostgreSQL/object storage are introduced, production-like backups must archive the full data directory as one consistency unit.

## Required Backup Jobs

- Daily platform backup: archive users, wallets, ledger, sessions, progression, devices, audit logs, tournaments, PvP matches, telemetry, review cases, treasury, and metrics.
- Replay backup: copy finished game sessions plus telemetry and review cases to replay backup storage after completion.
- Tournament recovery backup: snapshot tournament, participant, match, submission, wallet-lock, and ledger files before bracket generation, before result transitions, and after payout settlement.

## Recovery Rules

- Restore ledger and wallet files together. Never restore one without the other.
- Restore tournament files and ledger files together for tournament incidents.
- Replay verification requires `puzzleSeed`, `difficultyProfile`, and `puzzleVersion`; backups must retain all three.
- Audit logs are append-only recovery evidence and must be included in every backup set.

## Production Target

- PostgreSQL daily snapshots with point-in-time recovery.
- Object storage lifecycle policy for replay artifacts and telemetry exports.
- Separate encrypted offsite copy for audit logs, replay archives, and tournament recovery snapshots.
- Monthly restore test using a clean environment.
