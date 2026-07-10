# Skill Arena Production Readiness Notes

## Distributed Locking

The current JSON-backed single-server build does not require distributed locks. Before horizontal scaling, add a lock provider around matchmaking, wallet settlement, tournament payouts, and replay review transitions.

Recommended options:
- PostgreSQL advisory locks when Postgres becomes the primary store.
- Redis locks with short TTLs for matchmaking and live session operations.

Lock keys should be scoped by user, match, tournament, and wallet transaction group.

## Background Job Queue

The backend now has a durable local job queue and worker foundation for:
- replay exports
- email sending
- backup jobs
- leaderboard recalculation
- tournament reward payouts
- telemetry aggregation

Current implementation:
- Worker manager starts replay, email, leaderboard, tournament, telemetry, and backup workers.
- Jobs support persisted status, retries, exponential backoff, cancellation, requeue, worker assignment, timing, and artifacts.
- Admin APIs expose job lists, queue statistics, retry, cancel, and requeue operations.

Future scale target:
- Move jobs to PostgreSQL, Redis Streams, or a dedicated queue.
- Add dead-letter queues and distributed worker leasing.
- Keep request handlers enqueue-only for slow work.

## Secrets Management

Runtime configuration supports environment overrides. Sensitive values must remain in secure environment variables or a secrets manager, not committed config files.

Production target:
- JWT secrets from a secrets manager.
- SMTP credentials from a secrets manager.
- Payment/KYC provider keys from a secrets manager.
- Key rotation runbook.

## Disaster Recovery Drill

Backups are not complete until restore is tested. The backend now includes scheduled/manual backup execution and a recovery validation command.

Required drill:
- Restore the latest daily backup into a clean environment.
- Verify login, wallet balances, ledger totals, replay reconstruction, tournament state, and audit logs.
- Record restore duration, failed steps, and corrective actions.
- Repeat monthly before production traffic grows.

Command:

```bash
go run ./cmd/recovery -backup ./backups/<backup-directory> -report ./recovery-report.json
```

## Maintenance and Shutdown

Maintenance mode is controlled by environment-backed configuration:
- `SKILL_ARENA_MAINTENANCE_ENABLED`
- `SKILL_ARENA_MAINTENANCE_MESSAGE`
- `SKILL_ARENA_MAINTENANCE_ALLOW_SUPER_ADMINS`

During maintenance, new match creation, PvP queue entry, tournament registration, and house challenge starts are blocked. Existing match submissions continue.

The API now uses graceful shutdown to stop accepting new requests, cancel workers, let active work persist, and close the store cleanly.

## Backend Freeze Boundary

Backend architecture is frozen after this milestone. Allowed backend changes:
- bug fixes
- security fixes
- performance improvements
- new game modules
- additive API versions

Do not redesign APCE, replay format, puzzle engine, matchmaking, trust engine, admin architecture, role hierarchy, or session lifecycle without a new architecture review.
