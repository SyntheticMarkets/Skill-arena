# Skill Arena Backend Freeze

Status: Backend v1.0 feature freeze

The backend is frozen for business features. Future backend work is limited to bug fixes, security fixes, performance/scalability work, production operations, and frontend integration support.

## Architecture

The API is a Go HTTP service under `backend/`. The runtime entrypoint is `cmd/api/main.go`; recovery validation is in `cmd/recovery/main.go`.

Primary packages:

- `internal/server`: route registration, health endpoint, CORS, API version surface.
- `internal/handlers`: REST handlers and middleware.
- `internal/db`: domain store and business workflows.
- `internal/models`: shared request/response/domain models.
- `internal/game`: Maze Arena generation, versioning, seed derivation, puzzle validation, and game registry.
- `internal/matchmaking`: PvP matching rules.
- `internal/payments`: provider abstraction.
- `internal/redis`: Redis client and local memory fallback.
- `internal/storage`: local and S3-compatible object storage.
- `internal/workers`: background jobs, replay/export/backup/recovery workflows.
- `internal/observability`: structured logging, metrics, and health primitives.

## Freeze Boundary

Production uses PostgreSQL for authoritative persistence. JSON files are development-only fallback state.

Important architecture note: the current PostgreSQL persistence is an intermediate production persistence layer. Domain boundaries remain isolated through the store and module contracts so Wallet, Replay, Users, Treasury, Matchmaking, Tournament, and Game subsystems can later migrate to dedicated normalized PostgreSQL repositories without changing REST handlers or business workflows.

## Configuration

Required production environment:

- `SKILL_ARENA_ENV=production`
- `SKILL_ARENA_DATABASE_URL=postgres://...`
- `SKILL_ARENA_REDIS_URL=redis://...`
- `SKILL_ARENA_JWT_SECRET`
- `SKILL_ARENA_PUZZLE_SECRET`
- `SKILL_ARENA_MFA_ENCRYPTION_KEY`
- `SKILL_ARENA_ALLOWED_ORIGINS`

Provider credentials:

- Email: `SKILL_ARENA_SMTP_HOST`, `SKILL_ARENA_SMTP_USER`, `SKILL_ARENA_SMTP_PASS`
- PayFast: `SKILL_ARENA_PAYFAST_MERCHANT_ID`, `SKILL_ARENA_PAYFAST_PASSPHRASE`
- Ozow: `SKILL_ARENA_OZOW_SITE_CODE`, `SKILL_ARENA_OZOW_PRIVATE_KEY`
- Storage: `SKILL_ARENA_STORAGE_PROVIDER=s3`, `SKILL_ARENA_S3_ENDPOINT`, `SKILL_ARENA_S3_BUCKET`, `SKILL_ARENA_S3_ACCESS_KEY`, `SKILL_ARENA_S3_SECRET_KEY`

## Authentication

Implemented:

- Registration
- Login
- JWT access tokens
- Refresh token rotation
- Logout/revoke
- Email verification with signed expiring one-time token
- Password reset with expiring one-time token
- Password history
- Account lockout and suspicious login audit
- TOTP MFA
- Recovery codes
- Privileged role MFA enforcement

Privileged roles requiring MFA:

- `super_admin`
- `admin`
- `treasury_manager`
- `fraud_analyst`

Existing privileged users can receive an enrollment-only token and complete MFA setup without lockout. Enrollment-only tokens cannot access privileged routes.

## Roles

Role order is defined in `models/user.go`. Administrative actions are enforced through `RequireRole`.

Public leaderboard output hides privileged accounts.

## Wallet And Ledger

Wallet fields:

- Live balance
- Live locked balance
- Demo balance
- Demo locked balance
- Pending withdrawals
- Bonus balance

Ledger transaction types:

- `deposit`
- `withdraw`
- `fee`
- `lock`
- `unlock`
- `stake`
- `reward`
- `loss`

Financial operations require an `Idempotency-Key` and request hash. Repeated requests with the same key return the original operation. Reusing the key with different request data is rejected.

## Deposit Lifecycle

Deposit flow:

1. Client submits deposit request with `Idempotency-Key`.
2. Backend creates provider session.
3. Session enters provider/pending lifecycle.
4. Provider callback marks pending/verified/settled.
5. Settlement creates ledger entry.
6. Wallet available balance changes only after settlement.
7. Audit log records state transitions.

The backend must never directly credit a wallet at request time.

## Withdrawal Lifecycle

Withdrawal flow:

1. Client submits withdrawal request with `Idempotency-Key`.
2. Backend validates KYC, trust limits, available live balance, and AML rules.
3. Amount plus fee moves to pending withdrawal hold.
4. Risk/AML review may open.
5. Treasury approves or rejects.
6. Provider settlement occurs.
7. Ledger records withdrawal and fee.
8. Wallet available balance and pending withdrawals reconcile.
9. Audit log records state transitions.

The backend must never debit as final at request time.

## Treasury

Treasury tracks:

- Player reserve
- Revenue reserve
- Season reserve
- Championship reserve
- Jackpot reserve
- Emergency reserve

Health reports include:

- Player liabilities
- House exposure
- Solvency state
- Reserve coverage

Treasury actions are audited.

## AML And Risk

AML review inputs:

- Withdrawal velocity
- Large withdrawal threshold
- Country rules
- Trust tier limits
- Manual escalation target

AML cases are tied to withdrawal IDs and can be approved/rejected as part of the treasury lifecycle.

## Maze Arena

Maze Arena is Game #1 in the game registry.

Core mechanics:

- Versioned puzzle generation
- Difficulty profiles
- Deterministic HMAC seed derivation
- Line puzzle dependency chains
- Cross dependencies
- Dead ends and false routes
- Click validation
- Replay reconstruction

The platform uses a game registry and contracts for metadata, renderer, replay, and tournament support so future games can be added without changing platform routes.

## PvP And Matchmaking

PvP flow:

1. Player joins queue.
2. Trust and wallet eligibility are checked.
3. Stake is locked.
4. Redis lock coordinates queue matching.
5. Compatible waiting match activates.
6. Backend derives per-player puzzle seeds.
7. Backend owns current progress updates.
8. Submission validates moves/clicks.
9. Settlement unlocks/consumes stakes and credits reward.

PvP state includes progress, combo, moves remaining, completion percent, finish state, disconnect/reconnect-compatible match detail, and replay metadata.

## Replay

Replay reports include:

- Session ID
- User ID
- Game type and mode
- Difficulty profile
- Puzzle seed
- Generation nonce
- Generation hash
- Puzzle/game/replay version
- Lines/clicks/moves
- Playback events
- Integrity status
- Flags
- HMAC replay signature

Replay verification regenerates puzzle data from seed, profile, and version metadata.

## Workers

Workers handle:

- Replay export
- Email outbox
- Leaderboard recalculation
- Tournament reward tasks
- Telemetry aggregation
- Backup scheduling

Redis coordinates queue markers and job claiming.

## Storage

Development:

- Local filesystem object storage.

Production:

- S3-compatible storage.

Used for:

- Replay exports
- Backup snapshots
- Analytics exports
- Evidence/dispute artifacts

## Observability

Implemented primitives:

- Structured JSON logging
- Metrics counters/snapshots
- Health component records
- Worker health
- Queue stats

Production deployment should wire these to the platform monitoring stack.

## Security Model

Security controls:

- Password hashing
- JWT signing
- Refresh token rotation
- Session revocation
- Device registration
- MFA for privileged users
- Rate limiting
- CORS allowed-origin enforcement
- Financial idempotency
- Audit logging
- Replay signatures
- Puzzle generation HMAC
- Production dependency fail-fast checks

Secrets must come from environment variables or secret manager injection. Do not hardcode secrets in source.

## Infrastructure Dependencies

Production dependencies:

- PostgreSQL
- Redis
- S3-compatible object storage
- SMTP/email provider
- Payment providers
- Secret manager/environment injection
- Monitoring/log aggregation

Local development fallbacks:

- JSON data directory
- In-memory Redis-compatible client
- Local object storage
- Email outbox artifacts

## Verification At Freeze

Commands run:

```powershell
gofmt -l .
go test ./...
go vet ./...
go build ./...
```

Additional freeze evidence:

- Source integrity scan: no zero-byte files, NUL bytes, or invalid UTF-8 in project text files.
- Secret-like literal scan: no hardcoded production secret assignments found.
- Load-path test: 100 auth sessions, 100 deposits, 100 replay requests, 100 PvP joins, 100 leaderboard reads.
- Backup/restore test: backup, delete source data, restore, verify user and wallet.
- Replay longevity test: regenerate seed, puzzle, generation hash, rules version, and replay signature.
