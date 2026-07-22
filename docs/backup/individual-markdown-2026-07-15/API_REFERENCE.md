# Skill Arena API Reference

Base path: `/api/v1`

Authentication: Bearer JWT in `Authorization: Bearer <token>` for protected routes.

Financial POST requests require `Idempotency-Key`.

## Public

| Method | Path | Purpose |
|---|---|---|
| GET | `/health` | Service health |
| GET | `/api/v1/config/features` | Feature flags |
| GET | `/api/v1/platform/stats` | Public platform stats |
| GET | `/api/v1/platform/puzzle-preview` | Puzzle preview |
| GET | `/api/v1/leaderboard` | Public leaderboard |

## Authentication

| Method | Path | Purpose |
|---|---|---|
| POST | `/api/v1/auth/register` | Create account |
| POST | `/api/v1/auth/login` | Login, returns JWT/refresh token or MFA challenge |
| POST | `/api/v1/auth/refresh-token` | Rotate refresh token |
| POST | `/api/v1/auth/logout` | Revoke refresh token |
| POST | `/api/v1/auth/verify-email` | Consume email verification token |
| POST | `/api/v1/auth/resend-verification` | Send another verification link |
| POST | `/api/v1/auth/password-reset/request` | Request reset email |
| POST | `/api/v1/auth/password-reset/confirm` | Confirm reset token and new password |
| POST | `/api/v1/auth/mfa/setup` | Start TOTP setup |
| POST | `/api/v1/auth/mfa/confirm` | Confirm TOTP setup |
| POST | `/api/v1/auth/mfa/disable` | Disable MFA |

## Identity

| Method | Path | Purpose |
|---|---|---|
| POST | `/api/v1/identity/kyc-submit` | Submit KYC |
| GET | `/api/v1/identity/kyc-status` | KYC status |

## Profile And Progression

| Method | Path | Purpose |
|---|---|---|
| GET | `/api/v1/profile` | Current player profile |
| GET | `/api/v1/progression` | XP, level, league, trust |
| GET | `/api/v1/achievements` | Player achievements |
| GET | `/api/v1/achievements/catalog` | Static achievement catalog |

## Seasons

| Method | Path | Purpose |
|---|---|---|
| GET | `/api/v1/seasons/current` | Active season |
| GET | `/api/v1/seasons/leaderboard` | Season ranking |

## Wallet

| Method | Path | Purpose |
|---|---|---|
| GET | `/api/v1/wallet` | Wallet summary |
| GET | `/api/v1/wallet/transactions` | Ledger history |
| GET | `/api/v1/wallet/balance` | Live/demo balances |
| GET | `/api/v1/wallet/available` | Available balances |
| POST | `/api/v1/wallet/deposit` | Create provider deposit session |
| POST | `/api/v1/wallet/withdraw` | Request withdrawal |
| POST | `/api/v1/wallet/lock-tokens` | Lock wallet funds |
| POST | `/api/v1/wallet/unlock-tokens` | Unlock wallet funds |

Deposit request:

```json
{
  "amount": 100,
  "currency": "USD",
  "provider": "payfast",
  "method": "card",
  "country": "ZA",
  "reference": "client-reference"
}
```

Withdrawal request uses the same body shape and requires email verification and KYC when thresholds require it.

## Treasury

| Method | Path | Role | Purpose |
|---|---|---|---|
| GET | `/api/v1/treasury/status` | player | Public treasury status |
| GET | `/api/v1/admin/treasury/health` | admin | Treasury health |
| POST | `/api/v1/admin/treasury/withdrawals/approve` | treasury_manager | Approve withdrawal |
| POST | `/api/v1/admin/treasury/withdrawals/reject` | treasury_manager | Reject withdrawal |
| POST | `/api/v1/admin/treasury/withdrawals/settle` | treasury_manager | Settle withdrawal |

Treasury action body:

```json
{
  "withdrawalId": "obj_x",
  "providerRef": "provider-reference",
  "reason": "optional rejection reason"
}
```

## Games

| Method | Path | Purpose |
|---|---|---|
| POST | `/api/v1/games/start` | Start Maze Arena session |
| POST | `/api/v1/games/finish` | Submit moves/clicks |
| GET | `/api/v1/games/history` | Session history |
| GET | `/api/v1/games/{sessionId}` | Session detail |

Start game body:

```json
{
  "gameType": "demo",
  "mode": "maze",
  "stake": 10,
  "difficulty": 1
}
```

Finish body:

```json
{
  "sessionId": "ses_x",
  "moves": [
    { "direction": "line-0" }
  ]
}
```

## Calibration And House

| Method | Path | Purpose |
|---|---|---|
| POST | `/api/v1/calibration/start` | Start daily calibration |
| GET | `/api/v1/calibration/baseline` | Behavioral baseline |
| GET | `/api/v1/house/tiers` | House tiers |
| POST | `/api/v1/house/start` | Start house challenge |

## PvP

| Method | Path | Purpose |
|---|---|---|
| POST | `/api/v1/pvp/join` | Join PvP queue |
| POST | `/api/v1/pvp/progress` | Update authoritative progress |
| POST | `/api/v1/pvp/submit` | Submit final PvP moves |
| GET | `/api/v1/pvp/matches` | Player PvP matches |
| GET | `/api/v1/pvp/matches/{matchId}` | PvP match detail |

Join body:

```json
{
  "queueType": "standard",
  "walletType": "demo",
  "stake": 10
}
```

Progress body:

```json
{
  "matchId": "mat_x",
  "currentProgress": 50,
  "currentCombo": 4,
  "movesRemaining": 12,
  "completionPercent": 67,
  "finished": false
}
```

## Tournaments

| Method | Path | Purpose |
|---|---|---|
| GET | `/api/v1/tournaments` | List tournaments |
| POST | `/api/v1/tournaments/register` | Register |
| POST | `/api/v1/tournaments/submit-match` | Submit tournament match |
| GET | `/api/v1/tournaments/{id}` | Tournament detail |

## Replays

| Method | Path | Purpose |
|---|---|---|
| GET | `/api/v1/replays` | Player replay list |
| GET | `/api/v1/replays/{sessionId}` | Replay detail |
| GET | `/api/v1/admin/replays/{sessionId}` | Admin replay detail |

## Admin

| Method | Path | Role | Purpose |
|---|---|---|---|
| GET | `/api/v1/admin/users` | admin | User list |
| POST | `/api/v1/admin/roles/update` | super_admin | Update role |
| POST | `/api/v1/admin/roles/suspend` | super_admin | Suspend admin |
| POST | `/api/v1/admin/mfa/reset` | super_admin | Reset privileged MFA |
| GET | `/api/v1/admin/audit-logs` | admin | Audit logs |
| POST | `/api/v1/admin/kyc/approve` | admin | Approve KYC |
| GET | `/api/v1/admin/review-cases` | admin | Review cases |
| POST | `/api/v1/admin/review-cases/transition` | admin | Transition review case |
| GET | `/api/v1/admin/metrics` | admin | Metrics |
| GET | `/api/v1/admin/system-health` | admin | System health |
| GET | `/api/v1/admin/jobs` | admin | Jobs |
| GET | `/api/v1/admin/jobs/stats` | admin | Job queue stats |
| POST | `/api/v1/admin/jobs/retry` | admin | Retry job |
| POST | `/api/v1/admin/jobs/cancel` | admin | Cancel job |
| POST | `/api/v1/admin/jobs/requeue` | admin | Requeue job |
| GET | `/api/v1/admin/backups` | admin | Backup records |
| POST | `/api/v1/admin/backups/restore` | super_admin | Validate backup |
| GET | `/api/v1/admin/house-risk/{tier}` | admin | House risk |
| GET | `/api/v1/admin/baselines` | admin | Behavioral baselines |
| POST | `/api/v1/admin/tournaments/bracket` | admin | Generate bracket |
| POST | `/api/v1/admin/tournaments/result` | admin | Report tournament result |
