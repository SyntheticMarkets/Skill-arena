# Database Schema

Production database: PostgreSQL.

Development fallback: JSON files under `backend/data/`, ignored from Git.

Migration source: `backend/migrations/001_create_tables.sql`.

## Core Tables

| Table | Purpose |
|---|---|
| `users` | Accounts, roles, KYC status, verification state |
| `auth_sessions` | Refresh token sessions |
| `wallets` | Live/demo balances, locks, pending withdrawals |
| `ledger_entries` | Immutable wallet/treasury ledger events |
| `game_sessions` | Maze Arena sessions, puzzle metadata, moves/clicks |
| `devices` | Device fingerprints |
| `progression` | XP, level, ELO, league, trust score |
| `achievements` | Unlocked achievements |
| `seasons` | Season lifecycle |
| `tournaments` | Tournament definitions |
| `tournament_participants` | Registered users |
| `tournament_matches` | Bracket matches and seeds |
| `tournament_submissions` | Player tournament submissions |
| `pvp_matches` | PvP match state |
| `pvp_submissions` | PvP move submissions |
| `behavioral_baselines` | Player calibration profile |
| `gameplay_telemetry` | Security telemetry |
| `review_cases` | Fraud/replay/manual review |
| `metrics_snapshots` | Metrics payloads |
| `background_jobs` | Worker queue |
| `backup_records` | Backup metadata |
| `audit_logs` | Security and business audit trail |
| `treasury_state` | Treasury reserve state |
| `store_snapshots` | Intermediate production persistence snapshot |
| `financial_idempotency` | Idempotency records for money movement |

## Money Tables

`wallets` stores current balance state.

`ledger_entries` records balance-changing and lock/unlock operations:

- `deposit`
- `withdraw`
- `fee`
- `lock`
- `unlock`
- `stake`
- `reward`
- `loss`

`financial_idempotency` prevents duplicate deposit/withdrawal creation:

- `idempotency_key`
- `user_id`
- `operation`
- `resource_type`
- `resource_id`
- `request_hash`
- `created_at`

## JSONB Fields

Several tables use JSONB for structured game or metadata payloads:

- Puzzle difficulty profile
- Puzzle version
- Maze cells
- Line puzzle data
- Moves and clicks
- Telemetry arrays
- Review flags
- Job payloads
- Audit metadata

## Indexes

Important indexes:

- `idx_ledger_entries_user_created`
- `idx_game_sessions_user_created`
- `idx_auth_sessions_user`
- `idx_audit_logs_created`
- `idx_pvp_matches_queue`
- `idx_background_jobs_status`
- `idx_financial_idempotency_user_operation`

## Repository Note

At freeze, PostgreSQL is authoritative through `store_snapshots` plus financial idempotency tables. This is intentionally transitional. The domain store API isolates callers so each subsystem can later be normalized into dedicated repositories.
