# Skill Arena Master Documentation

This is the canonical documentation source for Skill Arena. Product, architecture, gameplay, security, infrastructure, frontend, and operational decisions should be updated in this file from this point forward.

The original individual Markdown files were archived on 2026-07-15 under `docs/backup/individual-markdown-2026-07-15/`. They are retained for historical recovery and should not be edited as active documentation.

Documentation maintenance rules:

- Update this `README.md` as the single source of truth.
- Add new subjects as sections in this file and include them in the contents list.
- Do not edit archived files unless restoring historical material into this document.
- Do not create new standalone project Markdown documents without first changing this documentation policy.

## Contents

- [Vertical Production Roadmap](#vertical-production-roadmap)
- [Product Identity](#product-identity)
- [Design Principles](#design-principles)
- [Competitive Psychology](#competitive-psychology)
- [Platform Language](#platform-language)
- [Notification Guidelines](#notification-guidelines)
- [Onboarding Experience](#onboarding-experience)
- [First Five Minutes](#first-five-minutes)
- [Player Journey](#player-journey)
- [Site Map](#site-map)
- [Low-Fidelity Experience Wireframes](#low-fidelity-experience-wireframes)
- [Design System Plan](#design-system-plan)
- [Game Economy](#game-economy)
- [Game Rules](#game-rules)
- [Arena Core](#arena-core)
- [Arena Hub](#arena-hub)
- [Session Gateway](#session-gateway)
- [Game Protocol](#game-protocol)
- [Authentication Flow](#authentication-flow)
- [Payment Flow](#payment-flow)
- [API Reference](#api-reference)
- [Database Schema](#database-schema)
- [Backend Feature Freeze](#backend-feature-freeze)
- [Production Readiness](#production-readiness)
- [Backup Strategy](#backup-strategy)
- [Implementation Audit](#implementation-audit)
- [Planning Inventory](#planning-inventory)
- [Phase 1 And 3 Requirements](#phase-1-and-3-requirements)
- [Project Overview](#project-overview)

---

## Vertical Production Roadmap

Status: Approved delivery model

Skill Arena is delivered as vertical production slices. Product implementation and foundation hardening proceed together when they belong to the same user journey. A sprint must produce a visible, usable outcome while making the backend, security, API, infrastructure, and tests required by that outcome production-ready.

Do not create frontend placeholders that depend on unfinished backend work. Do not build infrastructure without connecting it to the product outcome that requires it.

### Production Slice Rule

A page or module is complete only when all eight gates pass:

| Gate | Requirement |
|---|---|
| Design | Approved experience and responsive states are implemented. |
| Frontend | The complete user journey works without placeholders or fabricated data. |
| Backend | Every service required by the journey is production-ready. |
| Security | Authentication, authorization, abuse controls, privacy, and financial controls are verified. |
| API | Contracts, validation, errors, idempotency, and versioning are complete and documented. |
| Tests | Unit, integration, end-to-end, failure-path, and relevant load tests pass. |
| Production | Health, observability, deployment, backup, recovery, and dependency behavior are proven. |
| Freeze | The slice is tagged and receives only bug, security, performance, scalability, or integration fixes. |

A passing build alone does not complete a gate. A module cannot be frozen while any dependency required by its real user flow remains simulated, local-only, client-authoritative, or operationally unverified.

### Foundation Work

The following defects remain launch blockers, but they are completed inside the production slice that first depends on them:

- Replace PostgreSQL whole-store snapshots with transactional domain repositories.
- Replace `float64` financial values with integer minor units or an approved fixed-decimal representation.
- Implement production payment providers and signed provider callbacks.
- Implement production email delivery.
- Implement the WebSocket Session Gateway.
- Make PvP state and outcomes server-authoritative.
- Complete replay signing, verification, immutable storage, and reconstruction integrity.
- Implement dependency-aware health, metrics, tracing, alerting, and worker monitoring.
- Implement and prove production backup, restore, and disaster recovery.
- Harden MFA, session revocation, device management, and privileged access.

Difficulty, game rules, financial controls, and integrity requirements must never be weakened to satisfy delivery timelines or performance tests.

### Sprint 1: Landing, Boot, And Authentication

Visible outcome: a visitor can understand Skill Arena, enter the platform, register, verify their identity, authenticate securely, and reach the correct next destination.

Required foundation work:

- Upgrade vulnerable frontend dependencies.
- Remove placeholder and fabricated content from the entry journey.
- Replace snapshot persistence for users, authentication sessions, MFA, devices, and audit records with transactional PostgreSQL repositories, migrations, constraints, and indexes.
- Complete production SMTP or transactional email delivery.
- Complete email verification, password reset, MFA enrollment/challenge, logout, refresh rotation, session revocation, and device management.
- Add frontend and end-to-end coverage for the complete authentication journey.
- Apply security headers, production CORS, rate limits, dependency health checks, and authentication observability.
- Prove backup and restore of the identity and authentication data required to recover this slice.

#### Sprint 1 Production Report

Report date: 2026-07-22

Status: **COMPLETE - FROZEN**

Independent validation completed on 2026-07-22. The implementation, security, API, responsive-design, integration, performance, and release gates pass. Legal content, production SMTP credentials, and local Docker execution are tracked as launch/configuration work rather than missing Sprint 1 implementation.

| Gate | Status | Evidence |
|---|---|---|
| Design | Complete | Every completed page and important state has desktop, 1024px tablet, and Pixel 7 proof under `docs/proof/sprint-1-final-validation/`. No fabricated player, match, prize, or leaderboard statistics are displayed. |
| Frontend | Complete | Boot recovery, landing, Guest Arena, registration, verification, login, password recovery, MFA challenge, privileged MFA enrollment, loading, error, success, mobile, keyboard, and screen-reader states are implemented. |
| Backend | Complete | Authentication uses normalized transactional PostgreSQL repositories. JSON remains a local-development fallback and is rejected as the production identity authority. |
| Security | Complete | Bcrypt, signed purpose-bound one-time tokens, JWT issuer/audience/type validation, refresh-family rotation and replay revocation, strict protected cookies, Origin-based CSRF protection, explicit CORS, lockout, rate limits, encrypted TOTP secrets, hashed recovery codes, current-role authorization, and hash-chained audit events are implemented. |
| API | Complete | Authentication contracts are versioned under `/api/v1`, documented below, and use stable JSON error codes. Browser token material is never returned to JavaScript. |
| Tests | Complete | Go unit/integration suite, real PostgreSQL integration, frontend unit tests, and 12 retry-free desktop/tablet/mobile authentication E2E tests pass. |
| Production | Credentials required | Production configuration rejects local outbox email, insecure cookies, weak secrets, missing Redis, wildcard/non-HTTPS origins, and missing SMTP configuration. Live SMTP delivery requires deployment credentials. |
| Freeze | Complete | Frozen by Git tag `sprint-1-v1.0-freeze`. No Sprint 2 implementation is included. |

##### Delivered Experience

1. The boot screen recovers a protected server session before rendering private navigation.
2. Landing sends every visitor through the game-agnostic Guest Arena before registration.
3. Registration validates email, password policy, ISO country, date of birth, age confirmation, and consent flags.
4. Email verification and password reset use signed, purpose-bound, expiring tokens whose hashes are stored for one-time consumption. Reopening an authentic verification link for an already verified identity returns success without consuming anything again; reset tokens remain strictly one-time.
5. Login rejects unverified or inactive identities, applies timing equalization and lockout, and creates only protected cookie sessions.
6. Refresh rotation is transactional. Reuse of a rotated refresh token revokes the complete token family.
7. Session recovery validates the JWT and its live PostgreSQL session on every request. Logout, session revoke, device revoke, and password reset invalidate server state.
8. Privileged roles receive an enrollment-only session until TOTP is confirmed. Recovery codes are displayed once and stored only as hashes. Privileged MFA cannot be disabled.
9. The authenticated Guest Arena state confirms identity without exposing unfinished Sprint 2 navigation or pretending preview gameplay is live.

##### Verification Results

```text
go version go1.26.5 windows/amd64
gofmt: completed over every backend Go file
go test ./...: PASS
go vet ./...: PASS
go build ./...: PASS
PostgreSQL 17.10 repository integration: PASS
Next.js 16.2.11 production build: PASS (22 static routes)
Sprint 1 ESLint: PASS, zero warnings
TypeScript typecheck: PASS
Vitest: 2 files, 3 tests passed
Playwright: 12 tests passed without retries (desktop Chromium, 1024px tablet Chromium, and Pixel 7)
npm audit --omit=dev: 0 vulnerabilities
govulncheck: 0 called vulnerabilities
```

Coverage baseline:

- Go repository excluding `internal/id`: 34.5% statements. Windows Application Control blocked only the coverage-instrumented `internal/id` executable; the normal `internal/id` and full repository tests passed.
- Authentication HTTP server package: 84.8% statements.
- Database package: 44.0% statements.
- Frontend unit coverage: 24.32% statements, 20.58% branches, 24.65% functions, and 26.16% lines.
- Frontend flow coverage is supplemented by the complete retry-free desktop/mobile E2E journey.

Real PostgreSQL backup/restore proof:

```text
users 1/1
auth_tokens 2/2
auth_sessions 2/2
mfa_settings 0/0
password_history 2/2
login_security 0/0
devices 0/0
audit_logs 9/9
schema_migrations 1/1
audit checksum matched: true
backup size: 25068 bytes
```

##### Final Independent Validation

Validation method: production Next.js build, real Go API, real PostgreSQL 17 integration, development-only email capture, Chromium desktop/tablet/mobile automation, handler integration tests, security failure tests, static review, and repeatable benchmarks. No wallet, Maze gameplay, tournament, treasury, payment-provider, or Sprint 2 implementation was included.

One application defect was discovered: authenticated mobile users could not see the Guest Arena logout action because the public navigation was hidden below 520px. The responsive rule was corrected and the complete mobile MFA/session/logout journey then passed.

###### Design Proof

Each link is a full-page screenshot from the retry-free final run.

| Page or state | Desktop | Tablet 1024px | Mobile Pixel 7 |
|---|---|---|---|
| Boot recovery | [Desktop](docs/proof/sprint-1-final-validation/boot-desktop-chromium.png) | [Tablet](docs/proof/sprint-1-final-validation/boot-tablet-chromium.png) | [Mobile](docs/proof/sprint-1-final-validation/boot-mobile-chromium.png) |
| Landing | [Desktop](docs/proof/sprint-1-final-validation/landing-desktop-chromium.png) | [Tablet](docs/proof/sprint-1-final-validation/landing-tablet-chromium.png) | [Mobile](docs/proof/sprint-1-final-validation/landing-mobile-chromium.png) |
| Guest Arena | [Desktop](docs/proof/sprint-1-final-validation/guest-arena-desktop-chromium.png) | [Tablet](docs/proof/sprint-1-final-validation/guest-arena-tablet-chromium.png) | [Mobile](docs/proof/sprint-1-final-validation/guest-arena-mobile-chromium.png) |
| Registration | [Desktop](docs/proof/sprint-1-final-validation/register-desktop-chromium.png) | [Tablet](docs/proof/sprint-1-final-validation/register-tablet-chromium.png) | [Mobile](docs/proof/sprint-1-final-validation/register-mobile-chromium.png) |
| Verification pending | [Desktop](docs/proof/sprint-1-final-validation/verification-pending-desktop-chromium.png) | [Tablet](docs/proof/sprint-1-final-validation/verification-pending-tablet-chromium.png) | [Mobile](docs/proof/sprint-1-final-validation/verification-pending-mobile-chromium.png) |
| Email verified | [Desktop](docs/proof/sprint-1-final-validation/verify-email-desktop-chromium.png) | [Tablet](docs/proof/sprint-1-final-validation/verify-email-tablet-chromium.png) | [Mobile](docs/proof/sprint-1-final-validation/verify-email-mobile-chromium.png) |
| Login | [Desktop](docs/proof/sprint-1-final-validation/login-desktop-chromium.png) | [Tablet](docs/proof/sprint-1-final-validation/login-tablet-chromium.png) | [Mobile](docs/proof/sprint-1-final-validation/login-mobile-chromium.png) |
| Forgot password | [Desktop](docs/proof/sprint-1-final-validation/forgot-password-desktop-chromium.png) | [Tablet](docs/proof/sprint-1-final-validation/forgot-password-tablet-chromium.png) | [Mobile](docs/proof/sprint-1-final-validation/forgot-password-mobile-chromium.png) |
| Password reset | [Desktop](docs/proof/sprint-1-final-validation/password-reset-desktop-chromium.png) | [Tablet](docs/proof/sprint-1-final-validation/password-reset-tablet-chromium.png) | [Mobile](docs/proof/sprint-1-final-validation/password-reset-mobile-chromium.png) |
| MFA enrollment | [Desktop](docs/proof/sprint-1-final-validation/mfa-enrollment-desktop-chromium.png) | [Tablet](docs/proof/sprint-1-final-validation/mfa-enrollment-tablet-chromium.png) | [Mobile](docs/proof/sprint-1-final-validation/mfa-enrollment-mobile-chromium.png) |
| Recovery codes | [Desktop](docs/proof/sprint-1-final-validation/mfa-recovery-codes-desktop-chromium.png) | [Tablet](docs/proof/sprint-1-final-validation/mfa-recovery-codes-tablet-chromium.png) | [Mobile](docs/proof/sprint-1-final-validation/mfa-recovery-codes-mobile-chromium.png) |
| MFA login | [Desktop](docs/proof/sprint-1-final-validation/mfa-login-desktop-chromium.png) | [Tablet](docs/proof/sprint-1-final-validation/mfa-login-tablet-chromium.png) | [Mobile](docs/proof/sprint-1-final-validation/mfa-login-mobile-chromium.png) |
| Invalid MFA feedback | [Desktop](docs/proof/sprint-1-final-validation/mfa-login-invalid-desktop-chromium.png) | [Tablet](docs/proof/sprint-1-final-validation/mfa-login-invalid-tablet-chromium.png) | [Mobile](docs/proof/sprint-1-final-validation/mfa-login-invalid-mobile-chromium.png) |
| Session recovery | [Desktop](docs/proof/sprint-1-final-validation/session-recovery-desktop-chromium.png) | [Tablet](docs/proof/sprint-1-final-validation/session-recovery-tablet-chromium.png) | [Mobile](docs/proof/sprint-1-final-validation/session-recovery-mobile-chromium.png) |
| Recovery-code login | [Desktop](docs/proof/sprint-1-final-validation/mfa-recovery-login-desktop-chromium.png) | [Tablet](docs/proof/sprint-1-final-validation/mfa-recovery-login-tablet-chromium.png) | [Mobile](docs/proof/sprint-1-final-validation/mfa-recovery-login-mobile-chromium.png) |
| Logged-out Guest Arena | [Desktop](docs/proof/sprint-1-final-validation/logout-desktop-chromium.png) | [Tablet](docs/proof/sprint-1-final-validation/logout-tablet-chromium.png) | [Mobile](docs/proof/sprint-1-final-validation/logout-mobile-chromium.png) |

No placeholder screen, lorem ipsum, fabricated platform count, fake match volume, fake prize pool, TODO control, or mock production service appears in the Sprint 1 surfaces. HTML input `placeholder` attributes are field hints, not unfinished UI. All form interaction in validation uses accessible labels and roles; keyboard-focus styles, semantic status/error regions, reduced-motion handling, and responsive text containment are present.

###### Complete Player Journey

| Step | Expected result | Actual result and evidence |
|---|---|---|
| Landing | Explain Skill Arena and lead to exploration before registration. | Passed; truthful live-state messaging and primary Guest Arena action. [Screenshot](docs/proof/sprint-1-final-validation/landing-desktop-chromium.png) |
| Guest Arena | Show game-agnostic disciplines without fake availability or requiring an account. | Passed; Maze is explicitly preview-only and future games are unreleased. [Screenshot](docs/proof/sprint-1-final-validation/guest-arena-desktop-chromium.png) |
| Register | Validate identity fields, age, consent, and password policy; create an unverified identity. | Passed with `201 verification_required`. [Screenshot](docs/proof/sprint-1/journey-register-desktop-chromium.png) |
| Verify email | Deliver a signed link, consume it once, verify the account, and handle authentic repeat visits idempotently. | Passed; tampered and expired links fail. [Screenshot](docs/proof/sprint-1-final-validation/verify-email-desktop-chromium.png) |
| Login | Reject unverified/invalid identities and create protected cookies for a verified identity. | Passed. [Screenshot](docs/proof/sprint-1/journey-login-desktop-chromium.png) |
| Forgot password | Return enumeration-resistant confirmation and queue recovery email. | Passed with `202` whether or not an identity exists. [Result](docs/proof/sprint-1/journey-forgot-password-result-desktop-chromium.png) |
| Password reset | Validate signed expiry and password confirmation/history, update password, and revoke sessions atomically. | Passed; old session rejected and new password accepted. [Result](docs/proof/sprint-1/journey-password-reset-result-desktop-chromium.png) |
| MFA enrollment | Restrict privileged session, render QR/secret, confirm TOTP, and expose ten recovery codes once. | Passed. [Enrollment](docs/proof/sprint-1-final-validation/mfa-enrollment-desktop-chromium.png), [codes](docs/proof/sprint-1-final-validation/mfa-recovery-codes-desktop-chromium.png) |
| MFA login | Return signed five-minute challenge, reject wrong code, accept current TOTP. | Passed. [Challenge](docs/proof/sprint-1-final-validation/mfa-login-desktop-chromium.png), [invalid state](docs/proof/sprint-1-final-validation/mfa-login-invalid-desktop-chromium.png) |
| Recovery-code login | Accept one stored recovery code exactly once. | Passed; reuse returns `401`. [Screenshot](docs/proof/sprint-1-final-validation/mfa-recovery-login-desktop-chromium.png) |
| Session recovery | Reload with protected cookies and recover the server-validated identity. | Passed across all viewports. [Screenshot](docs/proof/sprint-1-final-validation/session-recovery-desktop-chromium.png) |
| Logout | Revoke current session, clear cookies, and return to guest state. | Passed across all viewports after the mobile logout defect was fixed. [Screenshot](docs/proof/sprint-1-final-validation/logout-mobile-chromium.png) |

###### Sprint 1 Endpoint Evidence

All errors use `{"code":"...","message":"..."}`. Representative endpoint contracts follow; full models are documented in the API Reference section.

| Route | Authentication | Request example | Success example | Principal errors | Integration evidence |
|---|---|---|---|---|---|
| `GET /health` | No | None | `200 {"status":"ready","checks":{...}}` | `503 not_ready` | `TestSprint1PublicEntryAndHealthContracts` |
| `GET /health/live` | No | None | `200 {"status":"alive"}` | None | `TestSprint1PublicEntryAndHealthContracts` |
| `GET /health/ready` | No | None | `200` with identity/email readiness | `503 not_ready` | `TestSprint1PublicEntryAndHealthContracts`, Playwright startup gate |
| `GET /api/v1/config/features` | No | None | `200` capability flags | `429` | `TestSprint1PublicEntryAndHealthContracts`, Guest Arena E2E |
| `GET /api/v1/platform/stats` | No | None | `200` real pre-launch state | `429` | `TestSprint1PublicEntryAndHealthContracts`, Landing E2E |
| `GET /api/v1/platform/puzzle-preview` | No | None | `200 {"lines":[...]}` | `429` | `TestSprint1PublicEntryAndHealthContracts`, Landing/Guest Arena E2E |
| `POST /api/v1/auth/register` | No | email, password, country, date of birth, consents | `201 {"status":"verification_required","email":"..."}` | `400`, `409`, `429`, `503` | `TestAuthenticationLifecycleAndSessionRevocation`, `TestSprint1InvalidExpiryRateLimitAndAuthorizationContracts`, Playwright |
| `POST /api/v1/auth/verify-email` | No | `{"token":"..."}` | `204` | `400 AUTH_TOKEN_EXPIRED/USED/INVALID`, `429` | lifecycle/compliance tests, Playwright |
| `POST /api/v1/auth/resend-verification` | No | `{"email":"..."}` | `202` | `400`, `429`, `503` | `TestSprint1InvalidExpiryRateLimitAndAuthorizationContracts` |
| `POST /api/v1/auth/login` | No | `{"email":"...","password":"..."}` | `200` session or `202` MFA challenge | `401`, `403`, `423`, `429` | lifecycle, MFA, invalid-flow tests, Playwright |
| `POST /api/v1/auth/mfa/challenge` | Signed challenge | challenge plus TOTP or recovery code | `200` and protected cookies | `400`, `401`, `429` | `TestSprint1MFAChallengeAndRecoveryCodeContracts`, Playwright |
| `POST /api/v1/auth/refresh-token` | Refresh cookie | No body | `200` rotated cookies | `401` | lifecycle and PostgreSQL repository integration tests |
| `POST /api/v1/auth/logout` | Access session | No body | `204`, expired cookies | `401`, `403` | `TestSprint1SessionDeviceLogoutAndCSRFContracts`, Playwright |
| `GET /api/v1/auth/session` | Access session | None | `200 {"authenticated":true,"user":...}` | `401`, `403` | lifecycle/session compliance tests, Playwright recovery |
| `GET /api/v1/auth/sessions` | Access session | None | `200 {"sessions":[...]}` | `401`, `500` | `TestSprint1SessionDeviceLogoutAndCSRFContracts` |
| `POST /api/v1/auth/sessions/revoke` | Access session | `{"sessionId":"..."}` | `204` | `400`, `401`, `404` | `TestSprint1SessionDeviceLogoutAndCSRFContracts` |
| `GET /api/v1/auth/devices` | Access session | None | `200 {"devices":[...]}` | `401`, `500` | `TestSprint1SessionDeviceLogoutAndCSRFContracts` |
| `POST /api/v1/auth/devices/revoke` | Access session | `{"deviceId":"..."}` | `204` | `400`, `401`, `404` | `TestSprint1SessionDeviceLogoutAndCSRFContracts` |
| `POST /api/v1/devices/fingerprint` | Access session | fingerprint and optional device metadata | `200` device | `400`, `401`, `500` | `TestSprint1SessionDeviceLogoutAndCSRFContracts` |
| `POST /api/v1/auth/password-reset/request` | No | `{"email":"..."}` | `202` | `400`, `429`, `503` | lifecycle test, Playwright |
| `POST /api/v1/auth/password-reset/confirm` | Signed reset token | token, password, confirmation | `204` | `400 AUTH_TOKEN_EXPIRED/USED/PASSWORD_POLICY`, `429` | lifecycle/expiry tests, PostgreSQL integration, Playwright |
| `POST /api/v1/auth/mfa/setup` | Access or enrollment-only session | No body | `200` secret and `otpauthUrl` | `401`, `409` | privileged/player MFA tests, Playwright |
| `POST /api/v1/auth/mfa/confirm` | Access or enrollment-only session | `{"code":"123456"}` | `200 {"recoveryCodes":[...]}` | `400`, `401`, `429` | privileged/player MFA tests, Playwright |
| `POST /api/v1/auth/mfa/disable` | Player access session | password plus TOTP/recovery proof | `204` | `400`, `401`, `403` for privileged roles | `TestPrivilegedAccountMustEnrollMFA`, `TestSprint1PlayerCanEnableAndDisableMFA` |

###### Security Failure Evidence

| Attempt | Expected | Actual |
|---|---|---|
| Reuse rotated refresh token | Reject and revoke token family | `401`; replacement family session also invalidated |
| Use revoked session/device | Reject access | `401 session is expired or revoked` |
| Cookie-authenticated POST without approved Origin | Reject CSRF attempt | `403 {"code":"FORBIDDEN"...}` |
| Read browser tokens from JavaScript | Tokens unavailable | Access and refresh are `HttpOnly`; both are `Secure` in production and `SameSite=Strict` |
| Exceed login rate | Reject excess attempts | `429 RATE_LIMITED`; parallel browser validation also triggered this protection until serialized |
| Inspect stored password | Never equal plaintext | Bcrypt hash verified with `CompareHashAndPassword` |
| Invalid TOTP | Reject without consuming challenge | `401`; visible error proof captured |
| Reuse recovery code | Reject second use | First login `200`, reuse `401` |
| Expired email token | Reject | `400 AUTH_TOKEN_EXPIRED` |
| Expired password-reset token | Reject | `400 AUTH_TOKEN_EXPIRED` |
| Expired MFA challenge | Reject | `401 AUTH_TOKEN_EXPIRED` |
| Tampered verification token | Reject | `400` |
| Underage or weak-password registration | Reject | `400`; weak password uses `AUTH_PASSWORD_POLICY` |
| Access protected endpoint without session | Reject | `401` across logout, sessions, devices, MFA, and device registration |

###### Performance Evidence

Measurements are local baselines from an Intel i7-8665U on Windows, not internet or production-SLA claims.

| Measurement | Result |
|---|---|
| Landing response end, 15 isolated Chromium contexts | p50 3.5 ms, p95 12.4 ms |
| Landing DOM content loaded | p50 33.2 ms, p95 75.8 ms |
| Landing load event | p50 82.2 ms, p95 104.3 ms |
| Landing primary heading ready, wall clock | p50 169.5 ms, p95 267.3 ms |
| Registration handler, 3 x 20 runs | 87.0-93.1 ms/op |
| Login handler, 3 x 20 runs | 75.7-78.6 ms/op |
| Authenticated session validation | 25.1-52.2 microseconds/op |
| PostgreSQL `GetUserByEmail` | 0.247-0.279 ms/op |
| PostgreSQL `ValidateAuthSession` | 0.442-0.730 ms/op |

Repeatable commands are implemented in `backend/internal/server/auth_benchmark_test.go` and `frontend/test/performance-validation.mjs`.

##### Remaining Known Issues

There are no Critical, High, Medium, or Low issues preventing Sprint 1 freeze.

Non-blocking launch checklist and technical debt:

1. **High - launch legal:** approved Terms of Service, Privacy Policy, and Fair Play text/URLs must be supplied and reviewed before public launch.
2. **High - launch configuration:** SMTP credentials, sender-domain DNS, and a real mailbox must be configured and delivery/bounce behavior tested in staging. SMTP with mandatory TLS is implemented and production rejects local outbox mode.
3. **Medium - environment verification:** Docker is unavailable on this workstation. Dockerfiles and Compose configuration were statically reviewed, while native backend and frontend production builds passed; container execution remains a staging/CI check.
4. **Low - out-of-scope technical debt:** repository-wide frontend lint reports one error and three warnings in pre-existing Admin, Dashboard, and Tournament pages. The isolated Sprint 1 lint and all Sprint 1 test code are clean.
5. **Low - coverage debt:** frontend unit coverage is 24.32% statements. Complete desktop/tablet/mobile integration journeys cover the release-critical behavior, but component-level coverage should increase through future bug and accessibility maintenance.

Freeze recommendation: **APPROVE SPRINT 1.** Do not begin Sprint 2 until separately requested.

##### Files Changed

Canonical documentation and archive consolidation:

- `.gitignore`, `README.md`
- Root documents removed after consolidation: `API_REFERENCE.md`, `ARENA_CORE.md`, `ARENA_HUB.md`, `AUTH_FLOW.md`, `BACKEND_FREEZE.md`, `BACKUP_STRATEGY.md`, `COMPETITIVE_PSYCHOLOGY.md`, `DATABASE_SCHEMA.md`, `DESIGN_PRINCIPLES.md`, `DESIGN_SYSTEM.md`, `FIRST_5_MINUTES.md`, `GAME_ECONOMY.md`, `GAME_PROTOCOL.md`, `GAME_RULES.md`, `IMPLEMENTATION_AUDIT.md`, `NOTIFICATION_GUIDELINES.md`, `PAYMENT_FLOW.md`, `PHASE_1_AND_3_REQUIREMENTS.md`, `PLANNING_INVENTORY.md`, `PLATFORM_LANGUAGE.md`, `PLAYER_JOURNEY.md`, `PRODUCTION_READINESS.md`, `PRODUCT_IDENTITY.md`, `SESSION_GATEWAY.md`, `SITE_MAP.md`, and `WIREFRAMES.md`.
- Archive copies under `docs/backup/individual-markdown-2026-07-15/`: every removed root document above plus `ONBOARDING_EXPERIENCE.md` and the archived root `README.md`.

Backend and deployment:

- `backend/Dockerfile`
- `backend/internal/config/config.go`, `backend/internal/config/config_test.go`
- `backend/internal/db/db.go`, `backend/internal/db/auth_postgres.go`, `backend/internal/db/auth_postgres_integration_test.go`
- `backend/internal/email/sender.go`, `backend/internal/email/sender_test.go`
- `backend/internal/handlers/auth.go`, `backend/internal/handlers/error_map.go`, `backend/internal/handlers/errors.go`, `backend/internal/handlers/middleware.go`
- `backend/internal/models/device.go`, `backend/internal/models/security.go`, `backend/internal/models/user.go`
- `backend/internal/redis/redis.go`, `backend/internal/redis/redis_test.go`
- `backend/internal/server/server.go`, `backend/internal/server/auth_integration_test.go`, `backend/internal/server/auth_compliance_test.go`, `backend/internal/server/auth_benchmark_test.go`
- `backend/internal/workers/manager.go`
- `backend/migrations/001_create_tables.sql`, `backend/migrations/002_auth_identity.sql`
- `docker-compose.yml`

Frontend and tests:

- `frontend/Dockerfile`, `frontend/eslint.config.mjs`, `frontend/next.config.mjs`, `frontend/next-env.d.ts`, `frontend/package.json`, `frontend/package-lock.json`, `frontend/playwright.config.ts`, `frontend/tsconfig.json`, `frontend/vitest.config.ts`
- `frontend/app/layout.tsx`, `frontend/app/page.tsx`, `frontend/app/app-shell.tsx`, `frontend/app/auth-context.tsx`
- `frontend/app/arena/page.tsx`
- `frontend/app/auth/auth-frame.tsx`, `frontend/app/auth/login/page.tsx`, `frontend/app/auth/register/page.tsx`, `frontend/app/auth/register/page.test.tsx`
- `frontend/app/auth/forgot-password/page.tsx`, `frontend/app/auth/reset-password/page.tsx`, `frontend/app/auth/verification-pending/page.tsx`, `frontend/app/auth/verify-email/page.tsx`
- `frontend/app/auth/mfa/page.tsx`, `frontend/app/auth/mfa/setup/page.tsx`
- `frontend/app/lib/api.ts`, `frontend/app/lib/api.test.ts`
- `frontend/e2e/authentication.spec.ts`, `frontend/e2e/final-validation.spec.ts`, `frontend/test/setup.ts`, `frontend/test/performance-validation.mjs`
- `frontend/styles/globals.css`

Proof artifacts:

- `docs/proof/sprint-1/landing-desktop-chromium.png`, `docs/proof/sprint-1/landing-mobile-chromium.png`
- `docs/proof/sprint-1/guest-arena-desktop-chromium.png`, `docs/proof/sprint-1/guest-arena-mobile-chromium.png`
- `docs/proof/sprint-1/verification-pending-desktop-chromium.png`, `docs/proof/sprint-1/verification-pending-mobile-chromium.png`
- `docs/proof/sprint-1/authenticated-desktop-chromium.png`, `docs/proof/sprint-1/authenticated-mobile-chromium.png`
- `docs/proof/sprint-1-final-validation/`: 48 desktop/tablet/mobile screenshots covering every completed Sprint 1 page and important MFA/session state.

### Sprint 2: Arena Hub, Navigation, Profile, And Notifications

Visible outcome: authenticated players enter a game-agnostic Arena Hub that presents identity, progression, available games, notifications, and one clear next action.

Required foundation work:

- Establish normalized PostgreSQL repositories for users, profiles, progression, game metadata, and notifications.
- Complete the game registry and capability-driven Hub API without Maze-specific assumptions.
- Implement durable notification storage and delivery contracts.
- Remove client-estimated statistics and all fake activity.
- Complete profile management, route protection, loading, error, empty, and recovery states.

### Sprint 3: Financial Platform

Visible outcome: players can understand and control their complete financial relationship with Skill Arena through a provider-independent wallet, deposit and withdrawal lifecycles, limits, financial assessment, responsible gaming controls, and transparent status timelines.

Required foundation work:

- Convert all money to integer minor units or an approved fixed-decimal type.
- Implement transactional PostgreSQL wallet, ledger, payment, withdrawal, treasury, and idempotency repositories.
- Complete Player Wallet balances, pending funds, transaction history, statements, limits, verification status, and payment-method presentation.
- Implement Payment Core, a provider registry, and provider-neutral Card, EFT, and Bank Transfer contracts. Future providers, including crypto where legally approved, must not require Wallet redesign.
- Integrate approved live payment providers behind Payment Core; Wallet must never branch on provider identity.
- Verify signed webhooks and make settlement idempotent.
- Implement the withdrawal lifecycle: Requested -> Pending Review -> Approved -> Processing -> Completed, or Rejected.
- Implement the policy decision boundary: request -> policy engine -> Trust Score and rules -> manual review or auto-approval -> Treasury -> provider settlement.
- Initial production policy is 100% manual approval. The player always sees Pending Review and never sees or controls internal approval logic.
- Complete financial assessment, country and age rules, source-of-funds fields where legally required, responsible gaming, cooling-off, self-exclusion, and daily/monthly deposit and withdrawal limits.
- Complete KYC evidence storage, AML, risk, treasury approval/rejection, reconciliation, reserve validation, and immutable audit flows.
- Expose role-protected approval and rejection APIs for the future Admin CRM, but implement no CRM screens in the player application.
- Store statements and exports in production object storage.
- Prove end-to-end cent-level reconciliation and provider failure recovery.

### Sprint 4: Admin CRM

Visible outcome: authorized staff use a separate application, authentication surface, navigation model, permission system, and deployment boundary to manage users, withdrawals, deposits requiring review, KYC, financial assessments, support, tournaments, moderation, treasury, reconciliation, fraud signals, announcements, compliance, and audit records.

Required foundation work:

- Create a separate CRM application. Do not add CRM pages or navigation to the player Next.js application.
- Replace broad role ranking with explicit permissions and separation of duties.
- Complete treasury, fraud, compliance, support, and super-admin workflows.
- Implement tamper-evident audit records and evidence retention.
- Add safe administrative session controls and mandatory MFA reauthentication for sensitive actions.
- Complete production observability, alerts, worker/queue monitoring, and operational runbooks.

### Sprint 5: Session Gateway, Presence, Notifications, And Realtime Events

Visible outcome: players receive authenticated live presence, match, notification, and reconnect events across the platform.

Required foundation work:

- Implement the authenticated WebSocket Session Gateway and versioned event protocol.
- Use Redis for production sessions, presence, queues, atomic rate limits, and ownership-safe distributed locks.
- Implement reconnect, resume, heartbeats, ordering, deduplication, backpressure, and graceful dependency failure.
- Load-test concurrent connections and chaos-test Redis, PostgreSQL, and gateway failure behavior.

### Sprint 6: Maze Arena

Visible outcome: Practice, Ranked, Daily Challenge, and Replay provide deterministic, satisfying, server-authoritative Maze competition.

Required foundation work:

- Make every gameplay action an intent validated by Arena Core; the frontend may not decide collisions or outcomes.
- Complete server-authoritative PvP progress, combo, moves, timing, finish, disconnect, and reconnect state.
- Verify generator, solver, validator, difficulty scorer, and replay behavior against approved reference fixtures.
- Complete replay signatures over seed, puzzle hash, generation hash, difficulty profile, rules version, ordered actions, timing, and outcome.
- Store immutable replay artifacts in object storage and support verification, playback, and disputes.
- Prove deterministic reconstruction and gameplay parity through integration and end-to-end tests.

### Sprint 7: Tournaments, Leaderboards, Seasons, And Rewards

Visible outcome: players can qualify, compete, follow brackets and rankings, receive auditable rewards, and understand seasonal progress.

Required foundation work:

- Complete tournament and season lifecycle state machines.
- Use shared server-generated seeds per match and independent authoritative player state.
- Validate leaderboard calculations and reward eligibility through durable workers.
- Settle entries, prizes, refunds, and rewards through the transactional wallet and treasury flow.
- Add spectator, dispute, replay, recovery, and operational controls required by live competition.

### Sprint Workflow

Each sprint follows:

Planning -> Wireframes -> High-fidelity UX approval -> Implementation -> Security review -> Testing -> Production verification -> Review -> Fixes -> Commit -> Tag -> Freeze

Only the current sprint may be implemented. A sprint does not advance because its page looks finished; it advances only after every production slice gate passes with recorded evidence.

---

## Product Identity

<!-- Archived source: docs/backup/individual-markdown-2026-07-15/PRODUCT_IDENTITY.md -->

Status: Draft for review

Sprint: 1 - Product Identity & Design Foundation

No UI implementation should begin until this document and the companion Sprint 1 documents are reviewed and approved.

### North Star

Every session should leave the player feeling they made meaningful progress.

Progress does not have to mean winning or earning money. Progress can mean:

- Improving skill.
- Increasing Trust Score.
- Climbing a leaderboard.
- Unlocking a challenge.
- Completing a replay.
- Learning a new strategy.
- Becoming more confident before entering ranked play.

Every product decision should support this North Star. If a screen, feature, animation, or piece of copy does not help the player feel progress, it should be simplified, redesigned, or removed.

### Why We Exist

Skill Arena exists to prove that competitive skill, not luck, can create meaningful progression, recognition, and rewards in an environment players trust.

This is the company mission, not a marketing line.

It means Skill Arena must:

- Reward learning and improvement, not only final outcomes.
- Make fairness visible before asking for trust.
- Treat money movement as a trust system, not a checkout flow.
- Make competition feel earned, recorded, and verifiable.
- Give every player a reason to return even after a loss.
- Build an ecosystem where practice, training, ranked matches, tournaments, replays, wallet flows, and trust systems all support one coherent progression loop.

### Brand Personality

Skill Arena should feel:

- Competitive
- Premium
- Intelligent
- Fair
- Trustworthy
- Fast
- Focused
- Rewarding
- High-tech
- Progress-driven

Short definition:

Skill Arena is a premium competitive skill platform where every match feels fair, meaningful, and worth returning to.

### Product Positioning

Skill Arena is not a generic gaming website.

It is a competitive arena for players who want to test skill, build trust, climb rankings, and compete in structured formats. The platform must feel alive, disciplined, and credible enough for real-money play without losing the energy of competition.

### What Skill Arena Is

- A competitive gaming platform.
- A skill-first arena.
- A trust-based real-money competition system.
- A progression engine.
- A replay-verified competitive environment.
- A platform that can support Maze Arena and future games.

### What Skill Arena Is Not

- A casual arcade portal.
- A casino-themed product.
- A CRUD dashboard.
- A financial app with games attached.
- A maze-only application.
- A landing page pretending to be a product.

### Target Audiences

#### Competitive Gamers

Why they join:

- They want ranked competition.
- They want proof of skill.
- They want opponents, leaderboards, and stakes.

What keeps them playing:

- Clear improvement loops.
- Ranked queues.
- Rivalry.
- Tournament opportunities.
- Match summaries that show where they improved.

What makes them trust the platform:

- Transparent ranking logic.
- Replay integrity.
- Anti-cheat signals.
- Fair matchmaking.
- Clear wallet and payout rules.

#### Puzzle Enthusiasts

Why they join:

- They enjoy problem-solving under pressure.
- They want daily challenges and measurable improvement.
- They want games that feel intelligent rather than random.

What keeps them playing:

- New puzzle patterns.
- Better completion times.
- Replay learning.
- Challenge tiers.
- Difficulty progression.

What makes them trust the platform:

- Consistent rules.
- Clear puzzle readability.
- No hidden randomness after match start.
- Replayable outcomes.

#### Casual Players

Why they join:

- They are curious.
- They want a low-risk way to try skill competition.
- They want practice before live competition.

What keeps them playing:

- Daily progress.
- Achievements.
- Practice balance.
- Clear next actions.
- Low-friction tutorials and practice.

What makes them trust the platform:

- Friendly onboarding.
- Clear practice vs live balance distinction.
- No pressure to deposit too early.
- Helpful error states.

#### Esports-Oriented Players

Why they join:

- They want prestige.
- They want tournaments.
- They want public ranking and status.

What keeps them playing:

- Brackets.
- Spectator/replay support.
- Seasonal rankings.
- Competitive identity.
- Recognition.

What makes them trust the platform:

- Public rules.
- Match records.
- Dispute-ready replay evidence.
- Visible tournament lifecycle.

#### Real-Money Competitors

Why they join:

- They want skill-based stakes.
- They want withdrawals and rewards tied to performance.
- They want a platform that takes fairness seriously.

What keeps them playing:

- Trust Score progression.
- Transparent wallet flows.
- Reliable payouts.
- Treasury confidence.
- Ranked and tournament formats.

What makes them trust the platform:

- Strong authentication.
- MFA.
- Audit trails.
- AML and treasury workflows.
- Idempotent financial operations.
- Clear pending/available/locked balances.

### Emotional Journey

| Moment | Desired Emotion | Product Meaning |
|---|---|---|
| Landing | Curious | This is a serious arena worth entering. |
| Guest Arena Hub | Interested | I can understand the ecosystem before committing. |
| Registration | Excited | I am creating a competitor identity. |
| Email verification | Reassured | The platform protects accounts and competition. |
| Player profile | Invested | I am becoming a recognizable competitor. |
| Verification pending | Informed | I can practice while I see exactly what remains locked. |
| Live unlock | Trusted | The platform has clearly opened the next competitive tier. |
| First practice match | Focused | I understand the core challenge. |
| Dashboard | Motivated | I know what to do next. |
| Wallet | Confident | My money and balances are clear. |
| Arena Hub | In control | I can choose the right module without leaving Skill Arena. |
| Queue | Anticipation | A real match is about to happen. |
| Match | Focus | Every move matters. |
| Defeat | Informed | I know what to improve. |
| Victory | Satisfaction | My skill produced a result. |
| Replay | Insight | I can learn from the match. |
| Leaderboard | Ambition | I can climb higher. |
| Tournament | Prestige | This is a bigger stage. |
| Withdrawal | Trust | The platform honors outcomes. |
| Return daily | Momentum | There is always progress waiting. |

### Product Voice

The voice should be:

- Clear.
- Competitive.
- Calm under pressure.
- Direct.
- Trust-building.
- Never childish.
- Never casino-like.
- Never vague about money.

Example tone:

- "Your withdrawal is pending treasury approval."
- "Replay verified. No integrity flags found."
- "Daily calibration complete. Trust Score updated."
- "You lost the match, but improved route efficiency by 8%."

Avoid:

- Hype without information.
- Fake urgency.
- Gambling language.
- Empty motivational copy.
- Placeholder statistics.

### Identity Approval Questions

Before implementation begins, reviewers should answer:

1. Does this identity feel like a competitive gaming platform?
2. Does it feel premium without becoming cold?
3. Does it create trust for real-money play?
4. Does it leave room for future games beyond Maze Arena?
5. Does every major product area connect back to meaningful progress?

---

## Design Principles

<!-- Archived source: docs/backup/individual-markdown-2026-07-15/DESIGN_PRINCIPLES.md -->

Status: Draft for review

Sprint: 1 - Product Identity & Design Foundation

### Primary Principle

Every screen must answer:

What is the next action that gets this player into a match, improves their skill, or advances their competitive progress?

If the answer is unclear, the screen is not ready.

### Core Principles

#### 1. One Primary Action Per Screen

Every page should have one dominant next step.

Examples:

- Landing: enter the Arena.
- Guest Arena Hub: explore or select a protected action.
- Dashboard: continue playing.
- Wallet: choose deposit or withdraw based on context.
- Arena Hub: choose a game module.
- Match Summary: replay, rematch, or queue again.

Secondary actions may exist, but they must not compete visually with the primary action.

#### 1A. Preserve Player Intent

Authentication, email verification, identity checks, and eligibility gates must return the player to the action that started the flow.

Examples:

- A guest who selects Ranked returns to Ranked eligibility after authentication.
- A player who opens a document-required notification returns to the exact verification task.
- A player who completes a wallet requirement returns to the pending deposit or withdrawal.

Do not send every completed flow to a generic dashboard.

#### 2. Progress Must Be Visible

Players should always understand what improved, unlocked, changed, or moved.

Progress signals may include:

- Skill improvement.
- Trust Score movement.
- XP and level.
- League or MMR movement.
- Challenge completion.
- Replay verification.
- Wallet settlement.
- Tournament advancement.

#### 3. Competition Should Feel Alive

The platform should communicate that other players are present and active.

Use:

- Live activity.
- Queue status.
- Recent match outcomes.
- Leaderboard movement.
- Tournament countdowns.
- Rival/opponent context.

Do not use fake statistics. If real data is unavailable, omit the module.

#### 4. Trust Must Be Designed, Not Claimed

Trust is created through clarity and proof.

Trust-building UI includes:

- Clear balance states.
- Pending/settled labels.
- Replay integrity status.
- Audit-style timelines.
- MFA/security prompts.
- Transparent error messages.
- Explicit transaction lifecycle.

Avoid vague labels such as "processing" when a more precise status exists.

#### 5. The Platform Is Game-Agnostic

Maze Arena is the first game, not the platform identity.

Shared UI must support:

- Maze Arena.
- Memory Arena.
- Logic Arena.
- Pattern Arena.
- Reaction Arena.
- Future games.

No global navigation, dashboard, wallet, leaderboard, or tournament UI should assume Maze-specific mechanics.

#### 6. Make Risk Understandable

Because Skill Arena supports live balances, the interface must clearly separate:

- Practice balance.
- Live balance.
- Available balance.
- Locked balance.
- Pending withdrawals.
- Rewards.
- Fees.

Money movement should always show state, reason, and next step.

#### 7. Reward Focus, Not Noise

The arena should feel energetic, but not chaotic.

Motion, effects, and live activity should support:

- Anticipation.
- Feedback.
- Victory.
- Progress.
- Status changes.

They should not distract from gameplay, wallet actions, or security decisions.

#### 8. Fast Feedback Everywhere

Every interaction should visibly respond.

Required states:

- Loading.
- Disabled.
- Hover/focus.
- Success.
- Error.
- Pending.
- Empty.
- Locked.
- Verified.

#### 9. Explain Failure Constructively

Defeat, blocked moves, failed payments, rejected withdrawals, and verification errors should tell the player what happened and what to do next.

Bad:

- "Error."
- "Failed."

Good:

- "Withdrawal rejected: KYC approval required for this amount."
- "Move blocked: dependency still active."
- "Replay flagged: route timing was too fast for verification."

#### 10. Build For Repetition

Players will see core screens many times.

Design should support repeated use:

- Scannable dashboards.
- Compact but clear wallet data.
- Quick queue entry.
- Persistent progress context.
- Minimal friction after first use.

#### 11. Every Screen Must Teach

Skill Arena should constantly help players understand the system.

If a player loses, they should understand why.

If a player wins, they should understand why.

If a player withdraws, they should understand every stage of the process.

Teaching creates trust because players can see cause and effect instead of guessing.

Examples:

- Match summary explains route efficiency, mistakes, timing, and dependency decisions.
- Replay shows what changed the result, not only what happened.
- Wallet timelines explain provider, pending, review, settlement, and ledger stages.
- Ranked screens explain MMR movement, placement, promotion, and demotion.
- Trust Score screens explain which actions increased, limited, or protected access.
- Error states explain the next useful action instead of ending the flow.

### Page-Level Design Tests

Before a page is approved, ask:

1. Why does this page exist?
2. What is the primary action?
3. What progress does this page show?
4. What competitive context does this page create?
5. What trust signal does this page provide?
6. Does this work for future games?
7. Does it avoid fake or placeholder data?
8. Is the page premium, or does it feel like CRUD?
9. What does this screen teach the player?

### Motion Principles

Motion should be:

- Fast.
- Purposeful.
- Directional.
- tied to state changes.

Use motion for:

- Queue transitions.
- Match start.
- Success/failure.
- Balance state changes.
- Rank movement.
- Replay playback.

Avoid motion for:

- Decorative noise.
- Constant looping distractions.
- UI that must be read carefully.

### Accessibility Principles

The platform must be usable under competitive pressure.

Requirements:

- Clear focus states.
- Keyboard navigation.
- Sufficient contrast.
- Motion reduction support.
- Text that does not overlap.
- Error messages that do not rely on color alone.
- Touch targets suitable for mobile.

### Approval Gate

No implementation should begin until these principles are approved.

---

## Competitive Psychology

<!-- Archived source: docs/backup/individual-markdown-2026-07-15/COMPETITIVE_PSYCHOLOGY.md -->

Status: Draft for review

Sprint: 1 - Product Identity & Design Foundation

This document defines the emotional system behind Skill Arena.

It should influence notifications, animations, rewards, achievements, dashboard design, match flows, wallet messaging, tournament moments, and replay education.

### Core Psychology

Skill Arena should create a loop where the player thinks:

```text
I understand what happened.
I know how to improve.
I can try again.
My progress matters.
The platform is fair.
```

The goal is not to make every moment positive.

The goal is to make every moment meaningful.

### Emotional Timeline

| Moment | Desired Feeling | Product Responsibility |
|---|---|---|
| Before queue | Excited | Show opportunity, eligibility, stakes, and progress upside. |
| Queue | Anticipation | Make waiting feel active without using fake data. |
| Match found | Adrenaline | Clearly transition from preparation to competition. |
| Countdown | Focus | Reduce distractions and clarify rules. |
| Match start | Control | Make input, timing, and state readable. |
| Mid-match | Pressure | Show competition without overwhelming the player. |
| Blocked move | Correction | Explain why the action failed. |
| Successful move | Satisfaction | Confirm skillful action quickly. |
| Victory | Achievement | Celebrate skill, progress, rank, reward, or trust movement. |
| Defeat | Motivation | Explain the loss and point to one useful next step. |
| Replay | Insight | Turn the result into learning. |
| Rank movement | Ambition | Show the player where they stand and what is next. |
| Tournament entry | Prestige | Make the stage feel larger than normal play. |
| Wallet deposit | Confidence | Make money movement clear and controlled. |
| Withdrawal | Trust | Show every stage and remove ambiguity. |
| Daily return | Momentum | Remind the player what they can progress today. |

### Victory Psychology

Victory should answer:

- What did I do well?
- What changed because I won?
- What did I unlock or move toward?
- What is the next challenge?

Victory can celebrate:

- Match result.
- Route efficiency.
- Speed improvement.
- Rank movement.
- Trust Score movement.
- Challenge progress.
- Tournament advancement.
- Reward eligibility.

Avoid:

- Empty confetti.
- Casino-like reward language.
- Fake urgency.
- Over-celebrating tiny actions.

### Defeat Psychology

Defeat should never feel like a dead end.

Defeat should answer:

- Why did I lose?
- Was the match fair?
- What is one thing I can improve?
- Should I replay, practice, queue again, or take a daily challenge?

Useful defeat outputs:

- Missed dependency.
- Slower route decision.
- Risky path choice.
- Opponent completed a key stage earlier.
- Replay comparison.
- Suggested training focus.

Avoid:

- "You lost" with no explanation.
- Shame.
- Dark patterns pushing an immediate deposit.
- Hiding replay insight.

### Anticipation Psychology

Queue and matchmaking are not passive waiting screens.

They should create controlled anticipation by showing:

- Selected mode.
- Eligible game.
- Trust/wallet requirements.
- Expected rules.
- Player readiness.
- Match found transition.

The platform must not fabricate live opponent counts or fake queue activity.

### Trust Psychology

Trust is emotional before it is technical.

Players trust Skill Arena when:

- They understand why money is pending, locked, settled, or rejected.
- They understand why a match result was valid.
- They understand why their Trust Score changed.
- They can review replay and wallet history.
- The platform communicates uncertainty honestly.

Trust-breaking patterns:

- Vague processing states.
- Unexplained rank changes.
- Wallet balance jumps without lifecycle.
- Unclear withdrawal delays.
- Hidden rules.

### Return Psychology

Daily return should be driven by momentum, not manipulation.

Reasons to return:

- New daily challenge.
- Practice target.
- Rival movement.
- Tournament countdown.
- House progress.
- Season progress.
- Replay insight to apply.
- Wallet or trust state resolved.

The player should feel:

- "I have something meaningful to do today."

Not:

- "The platform is pressuring me."

### Approval Questions

1. Does the platform make winning meaningful?
2. Does the platform make losing useful?
3. Does queue create anticipation without fake data?
4. Does wallet communication create confidence?
5. Does every return reason connect to real progress?
6. Does the psychology support future games?

---

## Platform Language

<!-- Archived source: docs/backup/individual-markdown-2026-07-15/PLATFORM_LANGUAGE.md -->

Status: Draft for review

Sprint: 1 - Product Identity & Design Foundation

This document defines the words Skill Arena uses across buttons, navigation, notifications, errors, tooltips, emails, wallet states, match summaries, and admin surfaces.

Language is part of the product identity.

### Language Principles

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

### People

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

### Core Actions

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
| Open the platform | Enter the Arena | Get Started |
| View verification state | Trust Status | KYC Status |

### Practice And Live

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

### Onboarding And Eligibility

Preferred terms:

- Secure account.
- Player identity.
- Financial assessment.
- Complete verification.
- Verification pending.
- More information required.
- Live competition unlocked.
- Trust Status.

Do not use `KYC` as the primary player-facing label. It may appear in legal explanations where precision requires it.

Locked states must use this structure:

```text
Capability locked.
Reason or requirement.
Next action.
```

### Wallet Language

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

### Match Language

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

### Error Language

Every error should explain:

- What happened.
- Why it happened when known.
- What the player can do next.

Bad:

- "Error."

Good:

- "Replay verification failed. The match remains under review while integrity checks complete."

### Approval Questions

1. Do the terms sound like Skill Arena?
2. Are player-facing labels consistent?
3. Does language teach instead of merely report?
4. Are financial terms precise enough for a real-money platform?
5. Is internal terminology separated from player-facing terminology?

---

## Notification Guidelines

<!-- Archived source: docs/backup/individual-markdown-2026-07-15/NOTIFICATION_GUIDELINES.md -->

Status: Draft for review

Sprint: 1 - Product Identity & Design Foundation

This document defines how Skill Arena communicates events across toasts, banners, inbox notifications, emails, push notifications, wallet timelines, match summaries, and admin-visible notices.

### Notification Principles

Notifications should:

- Be precise.
- Teach the player what happened.
- Explain the next step.
- Match the seriousness of the event.
- Avoid fake urgency.
- Avoid casino language.
- Never invent activity or statistics.

Every notification should answer:

- What happened?
- Why does it matter?
- What can the player do next?

### Notification Types

#### Success

Use for completed actions.

Example:

- "Deposit settled. Your live balance is now available for competition."

#### Pending

Use when the system has accepted an action but the lifecycle is not complete.

Example:

- "Withdrawal requested. Treasury review is now pending."

#### Warning

Use when the player should pay attention before continuing.

Example:

- "Ranked entry locked. Complete practice calibration to unlock live competition."

#### Error

Use when an action failed and the player needs a recovery path.

Example:

- "Replay verification failed. The match remains under review."

#### Educational

Use when the platform teaches a rule, result, or process.

Example:

- "Move blocked. The upper route is still locked by an active dependency."

#### Competitive

Use when competition state changes.

Example:

- "Match found. Enter Arena to begin countdown."

### Wallet Notification Examples

Preferred:

- "Deposit pending. Provider confirmation has not arrived yet."
- "Deposit settled. Your live balance is now available for competition."
- "Withdrawal under review. Treasury approval is required before settlement."
- "Withdrawal rejected. Complete verification before requesting this amount."
- "Ledger complete. Your transaction history has been updated."

Avoid:

- "Payment successful."
- "Payment failed."
- "Processing."
- "Cashout done."

### Replay Notification Examples

Preferred:

- "Replay verified. No integrity flags found."
- "Replay under review. Timing validation did not complete."
- "Replay invalid. The puzzle hash does not match the match record."
- "Insight ready. Review the dependency that slowed your route."

Avoid:

- "Replay failed."
- "Invalid game."
- "Something went wrong."

### Match Notification Examples

Preferred:

- "Match found. Countdown begins when both competitors are ready."
- "Victory confirmed. Replay verification complete."
- "Defeat confirmed. Review replay insights before entering the next match."
- "Connection interrupted. Reconnect before the grace period ends."

Avoid:

- "You lost."
- "Winner!"
- "Hurry now!"

### Notification Anatomy

Recommended structure:

```text
Status sentence.
Reason or context.
Next action when useful.
```

Example:

```text
Withdrawal under review.
Treasury approval is required for this amount.
You can track each stage in Wallet.
```

### Action Inbox And Deep Links

The notification bell is the player's central action inbox.

Every actionable notification must include one canonical destination. It should open the exact verification task, transaction, tournament lobby, challenge result, or replay insight rather than a generic dashboard.

Verification examples:

- "Identity verified. Review your newly unlocked live eligibility."
- "Address evidence required. Upload a current document to continue verification."
- "Verification pending. Practice remains available while review completes."

Rules:

- Preserve unread state until the notification or destination is viewed.
- Never place sensitive identity or risk details in push notification previews.
- Expired or completed actions must resolve to a current status page, not a broken destination.
- Security and financial notifications require an immutable server-side event reference.

### Email And Push Guidance

Emails may carry more explanation than toasts.

Push notifications should be short and actionable.

Security, wallet, and withdrawal notifications should prioritize clarity over excitement.

Competition notifications may carry more energy, but should remain precise.

### Approval Questions

1. Does each notification explain what happened?
2. Does it teach or guide when needed?
3. Does it avoid vague system language?
4. Does wallet messaging create trust?
5. Does competitive messaging create energy without fake urgency?

---

## Onboarding Experience

<!-- Archived source: docs/backup/individual-markdown-2026-07-15/ONBOARDING_EXPERIENCE.md -->

Status: Draft for product approval

Sprint: Product onboarding redesign

This document defines the complete path from first launch to live competition. It is a product and UX contract only. It does not authorize frontend or backend implementation.

### Why This Exists

Onboarding must prove that Skill Arena is a competitive ecosystem before it asks a player to complete forms or trust it with money.

The journey should make the player think:

- I understand what this platform is.
- I can explore before committing.
- I can improve before risking money.
- I always know what is unlocked, what is locked, and why.
- My identity, matches, and money are handled transparently.

### Experience Sequence

```text
Boot Experience
  -> Arena Landing
  -> Guest Arena Hub
  -> Locked Feature Intent
  -> Login or Registration
  -> Email Verification
  -> Player Profile
  -> Financial Assessment
  -> Identity Verification
  -> Verification Pending
  -> Practice Access
  -> Verification Decision
  -> Live Competition Unlock
```

Registration should be triggered by meaningful player intent. A visitor may explore the Arena Hub first, but must authenticate before progress can be saved or protected features can be used.

### Phase 1: Boot Experience

Purpose:

- Establish Skill Arena as a destination, not a conventional website.
- Communicate precision, competition, and readiness in two to three seconds.

Sequence:

```text
Dark field
  -> Skill Arena mark appears
  -> A small deterministic arrow sequence resolves
  -> "Every Move Matters"
  -> "Arena initializing"
  -> Requested destination loads
```

Rules:

- First visit may show the full sequence once.
- Returning visits use a shortened transition.
- Deep links must preserve their destination after boot.
- Reduced-motion mode replaces movement with short fades.
- Boot must never hide network failure or extend perceived load artificially.
- Maximum target duration is three seconds on a healthy load.

### Phase 2: Arena Landing

Purpose:

- Explain the promise.
- Establish financial and competitive trust.
- Create desire to enter the ecosystem.

Story order:

```text
Experience
  -> Competition
  -> Progress
  -> Fairness
  -> Wallet trust
  -> Community
  -> Enter the Arena
```

Primary message:

`WHERE SKILL BECOMES VALUE.`

Supporting proof should communicate:

- Compete against real players.
- Improve through practice and replay insight.
- Enter verified ranked and tournament competition.
- Track every wallet state clearly.
- Earn recognition and eligible rewards through skill.

Primary action:

- Enter the Arena.

Secondary action:

- Log in.

The landing page must not present fake live activity, player counts, prize totals, or match statistics.

### Phase 3: Guest Arena Hub

Purpose:

- Let the visitor understand the platform before registration.
- Show the relationship between games, competition, progression, wallet, and trust.

Visible areas:

- Available game modules.
- Practice.
- Ranked.
- Tournaments.
- Leaderboards.
- Challenges.
- Wallet.
- Replays.
- Community.
- Support.

Guest behavior:

- Public game information, rules, public leaderboards, and trust explanations may be explored.
- Practice may offer a short untracked sample only if the approved game protocol supports it.
- Progress, personalized practice, ranked, tournaments, wallet, deposits, withdrawals, notifications, and private replays require authentication.
- Locked surfaces remain readable enough to explain their value; they are not decorative blurred boxes with no context.

When a guest selects a protected action, show a focused authentication gate that preserves intent.

Example:

```text
Selected action: Enter Ranked
Requirement: Create or access your competitor account
After authentication: Return to Ranked eligibility
```

### Phase 4: Account Creation

Account creation is a staged journey, not one long form.

#### Step 1: Secure Account

Collect:

- Email.
- Password.
- Country of residence.
- Age confirmation.
- Terms acceptance.
- Privacy acceptance.
- Fair-play acknowledgement.

Primary action:

- Continue to email verification.

#### Step 2: Email Verification

Required states:

- Email sent.
- Resend available.
- Token expired.
- Token already used.
- Account already verified.
- Verification complete.

The pending destination must survive verification so the player continues where they intended to go.

#### Step 3: Player Identity

Collect:

- Nickname.
- Curated avatar.
- Display country where permitted.
- Timezone.
- Language.

Product rules:

- Verified legal identity is never publicly editable.
- Player-facing identity is nickname plus avatar.
- Initial avatars are curated platform assets, not uploaded photographs.
- Nickname policy, moderation, and change limits must be approved before implementation.
- Proposed nickname rule: one free change every 90 days; additional changes require a defined coin or support policy.

#### Step 4: Financial Assessment

Purpose:

- Collect the information required to determine live-wallet eligibility and compliance review.

Candidate fields, subject to legal and provider approval:

- Employment status.
- Income range.
- Expected deposit range.
- Source of funds.
- Tax residency.
- Politically exposed person declaration.

This step must explain why each category is requested. It must not collect fields until legal, privacy, retention, and jurisdiction requirements are approved.

#### Step 5: Identity Verification

Candidate evidence, subject to provider and jurisdiction rules:

- Identity document, passport, or driver's licence.
- Proof of address.
- Additional evidence when review requires it.

Required UX states:

- Not started.
- In progress.
- Submitted.
- Pending review.
- More information required.
- Rejected with reason and recovery action.
- Approved.
- Expired and renewal required.

### Phase 5: Verification Pending

Pending verification must not become a dead end.

Available:

- Practice games.
- Training.
- Daily challenges.
- Public leaderboards.
- Replay learning.
- Profile and security setup.
- Trust-status tracking.

Locked until approved:

- Live wallet activation.
- Deposits where policy requires approval.
- Withdrawals.
- Ranked cash competition.
- Paid tournament entry.
- Cash rewards settlement.

Every lock must display:

- The capability that is locked.
- The exact requirement.
- The current verification state.
- The next useful action.
- Where the player will return after completion.

### Phase 6: Live Competition Unlock

Approval is a meaningful progression event.

Notification:

```text
Identity verified.
Your live wallet and eligible live competitions are now unlocked.
Review your limits before entering your first live match.
```

Primary action:

- Review live eligibility.

The player should see:

- Verification complete.
- Live wallet state.
- Deposit and withdrawal limits.
- Eligible competition types.
- Trust Score effect.
- Security recommendations such as MFA.

### Trust Status

Trust Status is a persistent, player-readable model rather than a single KYC label.

It should cover:

| Area | Example State | Player Explanation |
|---|---|---|
| Email | Verified | Account ownership confirmed. |
| Player profile | Complete | Competitor identity created. |
| Financial assessment | Complete | Eligibility information received. |
| Identity | Pending review | Submitted evidence is being reviewed. |
| Address | More information required | A newer proof of address is required. |
| Live wallet | Locked | Unlocks after required verification completes. |
| Live competition | Locked | Practice remains available while review continues. |

The Trust Status view must never expose internal risk rules, fraud signals, or sensitive reviewer notes.

### Notification Center

The notification bell is the central action inbox for account, competition, wallet, and verification events.

Each notification must include:

- Category.
- State.
- Timestamp.
- Plain-language message.
- One destination.
- Read/unread state.

Deep-link examples:

| Notification | Destination |
|---|---|
| Address document required | Verification evidence step |
| Identity approved | Live eligibility summary |
| Withdrawal approved | Withdrawal timeline |
| Tournament starts soon | Tournament lobby |
| Daily challenge complete | Challenge result |
| Replay insight ready | Replay insight |

Notification links must restore the exact object or task, not open a generic dashboard.

### Arena Navigation

Primary navigation:

- Home.
- Games.
- Challenges.
- Leaderboards.
- Wallet.
- Replays.
- Community.
- Support.

Utility area:

- Notifications.
- Balance summary for authenticated eligible players.
- Help.
- Player avatar.

Public footer or support navigation:

- Contact.
- Terms and Conditions.
- Privacy.
- Responsible play and eligibility information where required.

### Profile Contract

The profile represents competitive identity.

Player-facing identity may show:

- Curated avatar.
- Nickname.
- Level and XP.
- Country where permitted.
- Trust status summary.
- Current rank.
- House.
- Achievements.
- Match history.

Legal identity and compliance evidence must never be exposed in public profile surfaces.

### Current Architecture Gap

The current backend supports a basic KYC state transition: submit, status, and admin approval. It does not currently provide the full product contract described here for:

- Financial assessment fields.
- Evidence upload and storage.
- Document type and expiry tracking.
- More-information and rejection workflows.
- Jurisdiction-specific requirements.
- Granular eligibility locks.
- Notification deep links for verification tasks.

Because the backend is feature frozen, these items must remain documented product requirements until a separately approved compliance-support sprint is authorized. High-fidelity mockups must label them as required contracts, not production-ready behavior.

### Approval Gate

Before implementation, product, legal/compliance, security, backend, and frontend owners must approve:

1. Whether guest practice is available and whether it is tracked.
2. Which actions require email, identity, address, or financial verification.
3. Which financial-assessment fields are legally required by jurisdiction.
4. Which KYC provider and evidence-retention model will be used.
5. The nickname and avatar policy.
6. Notification categories and deep-link destinations.
7. The exact live-competition unlock rules.
8. High-fidelity mockups for every state in this document.

No implementation should begin from this document alone.

---

## First Five Minutes

<!-- Archived source: docs/backup/individual-markdown-2026-07-15/FIRST_5_MINUTES.md -->

Status: Draft for review

Sprint: 1 - Product Identity & Design Foundation

This document defines the first 300 seconds of Skill Arena.

The first five minutes should prove the product promise before asking for money.

### Goal

By the end of five minutes, a new player should feel:

- I understand what Skill Arena is.
- I played something skill-based.
- I learned from my result.
- I made measurable progress.
- I trust the platform enough to continue.

The goal is not to push a deposit in the first five minutes.

The goal is to earn the player's confidence.

### Seconds 0-3: Enter

Player sees a short boot experience before the requested destination loads.

Product responsibility:

- Establish Skill Arena as a premium arena.
- Respect reduced-motion preferences.
- Never delay access beyond the real initialization need.

Success signal:

- The player feels they entered a product, not a static page.

### Minute 0-1: Understand And Explore

Player arrives on Skill Arena.

Product responsibility:

- Explain that this is skill-based competition.
- Show that progress matters.
- Make fairness and replay integrity visible.
- Present one clear action: enter the Arena Hub.
- Allow the player to understand games, progression, competition, and trust before registration.

Desired feeling:

- Curiosity.

Success signal:

- The player understands Skill Arena is an arena for skill, progression, and trust.

### Minute 1-2: Choose Intent And Register

Player creates a competitor account.

Product responsibility:

- Preserve the protected action that caused registration.
- Keep account creation focused.
- Explain age, terms, and fair play clearly.
- Reinforce that the next step is practice, not deposit.

Desired feeling:

- Excitement.

Success signal:

- The player feels they created a competitor identity.

### Minute 2-3: Verify

Player verifies email.

Product responsibility:

- Make verification fast and clear.
- Explain why account verification protects competition.
- Provide resend and expired-link recovery.

Desired feeling:

- Reassurance.

Success signal:

- The player trusts that account security matters.

### Minute 3-4: Create Player Identity And Start Practice

Player completes the first practice match.

Product responsibility:

- Collect nickname, curated avatar, timezone, and language without exposing legal identity.
- Teach the basic rules quickly.
- Keep the first match readable.
- Show blocked and successful actions clearly.
- Avoid financial pressure.

Desired feeling:

- Focus.

Success signal:

- The player understands the core challenge and wants to improve.

### Minute 4-5: Replay And Progress

Player views replay or match insight.

Product responsibility:

- Show one useful learning moment.
- Explain why the player won or lost.
- Show measurable progress.
- Offer the next action: training tip, daily challenge, or another practice match.

Desired feeling:

- Progress.

Success signal:

- The player thinks, "I learned something and can do better next time."

Financial assessment and identity verification may begin during onboarding, but they must not prevent Practice. The five-minute promise is learning and progress, not live-wallet activation.

### First 5 Minutes Approval Questions

1. Can a new player understand the product within the first minute?
2. Does the first play experience happen before any deposit request?
3. Does the first result teach the player something useful?
4. Does the platform show progress within five minutes?
5. Does the next action feel natural rather than forced?

---

## Player Journey

<!-- Archived source: docs/backup/individual-markdown-2026-07-15/PLAYER_JOURNEY.md -->

Status: Draft for review

Sprint: 1 - Product Identity & Design Foundation

### North Star Journey

Every session should leave the player feeling they made meaningful progress.

The player journey must show progress before, during, and after matches.

### Complete Journey

```text
New Visitor
  -> Boot Experience
  -> Arena Landing
  -> Guest Arena Hub
  -> Protected Action Selected
  -> Registration
  -> Email Verification
  -> Player Profile
  -> Financial Assessment
  -> Identity Verification
  -> Verification Pending
  -> First Practice Match
  -> Match Summary
  -> Replay Insight
  -> Tutorial Tips
  -> Daily Challenge
  -> Practice Progress
  -> Trust Increase
  -> Live Competition Unlock
  -> Dashboard Progress
  -> First Deposit
  -> Ranked Queue
  -> Ranked Match
  -> Leaderboard Movement
  -> Tournament Entry
  -> Tournament Result
  -> Withdrawal
  -> Return Daily
```

The complete onboarding state and lock contract is defined in [Onboarding Experience](#onboarding-experience).

### Step Detail

#### 1. New Visitor

Goal:

- Understand what Skill Arena is.
- Believe it is competitive, fair, and worth joining.

Emotion:

- Curious.

Primary action:

- Enter the Arena.

Trust signals:

- Skill-based positioning.
- Replay integrity.
- Secure wallet messaging.
- Real progression promise.

#### 2. Boot, Landing, And Guest Arena Hub

Goal:

- Establish Skill Arena as a destination.
- Let the visitor understand games, progression, competition, wallet trust, and locked live features before registration.

Emotion:

- Curious, then interested.

Primary action:

- Explore the Arena Hub, then select a meaningful protected action.

Trust signals:

- Clear practice and live distinction.
- Public rules and leaderboards.
- Locked capabilities explain their requirements.
- No fake activity or statistics.

#### 3. Registration

Goal:

- Create competitor identity.

Emotion:

- Excited.

Primary action:

- Submit account details.

Trust signals:

- Clear terms.
- Age verification.
- Privacy clarity.

#### 4. Email Verification

Goal:

- Confirm account ownership.

Emotion:

- Reassured.

Primary action:

- Verify email.

Trust signals:

- Clear resend flow.
- Expired token handling.
- Already verified handling.

#### 5. Player Profile, Financial Assessment, And Identity Verification

Goal:

- Create a player-facing competitor identity.
- Collect only approved eligibility and verification information.
- Keep Practice available while live eligibility is reviewed.

Emotion:

- Invested and informed.

Primary action:

- Complete the next required verification step or continue to Practice.

Trust signals:

- Legal identity is separated from public profile identity.
- Every requested field explains why it is needed.
- Verification status, locks, and next actions are explicit.

#### 6. First Practice Match

Goal:

- Experience the core game without financial risk.

Emotion:

- Focused.

Primary action:

- Complete practice match.

Progress signal:

- First completion.
- Route efficiency.
- Replay available.

#### 7. Match Summary

Goal:

- Understand outcome and improvement.

Emotion:

- Informed.

Primary action:

- Queue again or view replay.

Progress signal:

- Skill metric change.
- XP.
- Trust effect.
- Best route comparison.

#### 8. Replay Insight

Goal:

- Learn from the match.

Emotion:

- Insightful.

Primary action:

- Try again.

Trust signals:

- Replay verified.
- Integrity flags visible when relevant.

#### 9. Tutorial Tips

Goal:

- Convert the replay into useful learning.
- Show the player that improvement is visible and achievable.

Emotion:

- Encouraged.

Primary action:

- Apply one tip in another practice or training run.

Progress signal:

- Suggested skill focus.
- Route decision explanation.
- Timing or accuracy improvement target.
- "Try this next" guidance.

Trust signals:

- The platform explains outcomes without blaming the player.
- The player can improve before spending money.

#### 10. Daily Challenge

Goal:

- Give the player a structured reason to play again before live competition.

Emotion:

- Motivated.

Primary action:

- Start daily challenge.

Progress signal:

- Challenge completion.
- Personal best.
- Skill streak.
- Practice reward or trust-safe reward when allowed.

Trust signals:

- Clear rules.
- No financial pressure.
- Replay-backed result when relevant.

#### 11. Practice Progress

Goal:

- Prove the player enjoys the core loop before asking for a first deposit.

Emotion:

- Confident.

Primary action:

- Continue practice or unlock ranked eligibility.

Progress signal:

- Practice level.
- Accuracy trend.
- Completion trend.
- Replay insight history.

Trust signals:

- Practice progress and practice balance are separate from live balance.
- The player understands the game enough to make an informed deposit decision.

#### 12. Trust Increase

Goal:

- Show that fair play unlocks access.

Emotion:

- Rewarded.

Primary action:

- Continue playing or complete verification.

Progress signal:

- Trust Score movement.
- Trust tier.
- New eligibility.

#### 13. Live Competition Unlock

Goal:

- Turn verification approval into a clear progression event.

Emotion:

- Trusted and ready.

Primary action:

- Review live eligibility and limits.

Progress signal:

- Live wallet status.
- Eligible competition types.
- Trust status movement.

#### 14. Dashboard Progress

Goal:

- Give the player a clear next step.

Emotion:

- Motivated.

Primary action:

- Continue playing.

Progress signal:

- Daily progress.
- Season progress.
- Wallet snapshot.
- Leaderboard preview.

#### 15. First Deposit

Goal:

- Fund live play safely.

Emotion:

- Confident.

Primary action:

- Choose payment method.

Trust signals:

- Provider session.
- Pending/settled state.
- Available vs locked balance.
- Fees and limits.

Important product rule:

- The first deposit should come after practice play, replay insight, tutorial guidance, daily challenge exposure, practice progress, and visible trust movement.
- Skill Arena should earn the deposit by proving the product is fair, enjoyable, and understandable.

#### 16. Ranked Queue

Goal:

- Enter competitive play.

Emotion:

- Anticipation.

Primary action:

- Join queue.

Progress signal:

- League/MMR context.
- Estimated match quality.

#### 17. Ranked Match

Goal:

- Compete under pressure.

Emotion:

- Focus.

Primary action:

- Play.

Trust signals:

- Backend-owned opponent progress.
- Replay-ready match.
- Clear rules.

#### 18. Leaderboard Movement

Goal:

- See competitive status change.

Emotion:

- Ambition.

Primary action:

- Queue again.

Progress signal:

- Rank movement.
- Country/global/season position.

#### 19. Tournament Entry

Goal:

- Join a higher-prestige competition.

Emotion:

- Prestige.

Primary action:

- Register.

Trust signals:

- Prize pool.
- Bracket rules.
- Entry requirements.
- Replay dispute support.

#### 20. Withdrawal

Goal:

- Convert earned balance into payout.

Emotion:

- Trust.

Primary action:

- Request withdrawal.

Trust signals:

- Pending state.
- AML/review state.
- Treasury approval.
- Settlement timeline.

#### 21. Return Daily

Goal:

- Continue the progress loop.

Emotion:

- Momentum.

Primary action:

- Start daily challenge or ranked queue.

Progress signal:

- Daily streak.
- New challenges.
- Season movement.

### Journey Risks

Potential failure points:

- Visitor does not understand skill-based competition.
- Registration asks too much too early.
- Guest exploration becomes a dead-end marketing preview instead of teaching the platform.
- Protected actions forget the player's intended destination after authentication.
- Verification pending blocks Practice and causes abandonment.
- KYC locks do not explain the exact requirement or recovery action.
- First deposit is requested before the player has experienced enough value.
- Wallet feels like a generic balance table.
- Dashboard becomes a data dump.
- Arena Hub hides the best next action.
- Losing feels empty.
- Replay is hard to understand.
- Withdrawal status feels vague.

### Journey Approval Questions

1. Does every step make the player feel progress?
2. Does the first practice match happen early enough?
3. Does practice, replay, tutorial guidance, daily challenge, and practice progress earn the first deposit?
4. Does the journey work for non-Maze future games?
5. Does the return loop feel strong enough for daily play?

---

## Site Map

<!-- Archived source: docs/backup/individual-markdown-2026-07-15/SITE_MAP.md -->

Status: Draft for review

Sprint: 1 - Product Identity & Design Foundation

### Product Order

Backend complete.

Frontend/product order:

1. Product Identity & Design Foundation
2. Design System
3. Landing
4. Authentication
5. Dashboard
6. Wallet
7. Arena Hub
8. Maze Arena
9. Challenges
10. Ranked
11. Leaderboards
12. Tournaments
13. Profile
14. Settings
15. Admin
16. Polish
17. Launch

### Product Design Workflow

Every sprint after this foundation should follow:

```text
Product Documents
  -> Wireframes
  -> High-Fidelity UX Mockups
  -> Approval
  -> Implementation
  -> Testing
  -> Review
  -> Fixes
  -> Commit
  -> Tag
```

Implementation should not begin while the product story, wireframes, or high-fidelity UX mockups are still unresolved.

### Primary Site Structure

```text
Landing
  -> Boot Experience
  -> Guest Arena Hub
  -> Public Game Information
  -> Public Leaderboards
  -> Trust And Wallet Explanation
  -> Register
  -> Login
  -> Forgot Password
  -> Terms
  -> Privacy

Authentication
  -> Register
  -> Email Verification
  -> Player Profile
  -> Financial Assessment
  -> Identity Verification
  -> Verification Pending
  -> Live Eligibility Unlock
  -> Login
  -> MFA
  -> Password Reset
  -> Age Verification

Authenticated App Shell
  -> Dashboard
  -> Wallet
  -> Arena Hub
  -> Challenges
  -> Ranked
  -> Leaderboards
  -> Tournaments
  -> Replays
  -> Profile
  -> Settings
  -> Admin
```

Protected guest actions preserve their destination across authentication and verification.

```text
Guest Arena Hub
  -> Select protected action
  -> Authentication gate
  -> Complete required onboarding step
  -> Return to selected action or its eligibility screen
```

### Authenticated Navigation

```text
Dashboard
  -> Continue Playing
  -> Daily Challenge
  -> Arena Hub
  -> Wallet Summary
  -> Leaderboard Preview
  -> Season Progress
  -> Notifications
  -> Recent Games
  -> Achievements

Wallet
  -> Overview
  -> Deposit
  -> Payment Method
  -> Deposit Confirmation
  -> Pending Deposit
  -> Completed Deposit
  -> Withdraw
  -> Pending Withdrawal
  -> Transaction History
  -> Statements
  -> Export

Arena Hub
  -> Overall Profile
  -> Wallet Summary
  -> Overall Progression
  -> Platform Notifications
  -> Game Modules
  -> Maze Arena
  -> Memory Arena
  -> Reaction Arena
  -> Logic Arena
  -> Future Game Cards
  -> Matchmaking
  -> Live Match
  -> Match Summary
  -> Replay

Maze Arena
  -> Maze Home
  -> Practice
  -> Ranked
  -> Tournament
  -> Game Rules
  -> Puzzle Board
  -> Game-Specific Controls
  -> Maze Statistics
  -> Maze Leaderboard
  -> Maze Replay Renderer

Matchmaking
  -> Mode Selection
  -> Eligibility Check
  -> Queue
  -> Opponent Found
  -> Countdown
  -> Live Match
  -> Disconnect/Reconnect

Live Match
  -> Game Renderer
  -> Opponent Progress
  -> Timer
  -> Rules State
  -> Victory
  -> Defeat
  -> Match Summary
  -> Replay

Challenges
  -> House Challenges
  -> Daily
  -> Weekly
  -> Monthly
  -> Rewards

Ranked
  -> Queue
  -> Matchmaking
  -> Placement
  -> League
  -> MMR
  -> Promotion
  -> Demotion
  -> History

Leaderboards
  -> Global
  -> Country
  -> Friends
  -> Season
  -> Weekly
  -> Monthly
  -> Search
  -> Filters

Tournaments
  -> List
  -> Detail
  -> Join
  -> Bracket
  -> Matchmaking
  -> Live Match
  -> Results
  -> Replay

Profile
  -> Competitor Identity
  -> Curated Avatar
  -> Nickname
  -> Stats
  -> History
  -> Achievements
  -> Trust
  -> Badges
  -> Avatar
  -> Customization

Settings
  -> Profile
  -> Security
  -> Wallet
  -> Notifications
  -> Language
  -> Accessibility
  -> Privacy
  -> Verification Status
  -> Financial Assessment
  -> Identity Evidence

Notifications
  -> Account And Security
  -> Verification Tasks
  -> Wallet Timelines
  -> Match And Replay
  -> Challenges And Tournaments
  -> Deep Link To Exact Action

Admin
  -> Dashboard
  -> Users
  -> Wallet
  -> Treasury
  -> Games
  -> Moderation
  -> Support
  -> Reports
  -> Analytics
```

### Player Movement Model

Primary loop:

```text
Dashboard -> Arena Hub -> Game Module -> Matchmaking/Challenge -> Live Match -> Summary -> Replay/Rematch -> Arena Hub
```

Money loop:

```text
Arena Hub -> Wallet -> Deposit -> Arena Hub -> Game Module -> Live Match -> Arena Hub -> Wallet -> Withdraw
```

Competitive loop:

```text
Dashboard -> Ranked -> Matchmaking -> Live Match -> Rank Movement -> Leaderboard -> Queue Again
```

Tournament loop:

```text
Dashboard -> Tournaments -> Join -> Bracket -> Matchmaking -> Live Match -> Results -> Replay -> Next Round
```

Trust loop:

```text
Dashboard -> Verification/Security -> Calibration -> Clean Matches -> Trust Score Increase -> Higher Access
```

### Navigation Priority

Top-level navigation should prioritize:

1. Dashboard
2. Arena Hub
3. Wallet
4. Ranked
5. Tournaments
6. Leaderboards
7. Profile
8. Settings

Admin navigation should only appear for privileged roles.

### Approval Questions

1. Does this site map put the platform before Maze Arena?
2. Does every major path lead toward play, competition, trust, or progress?
3. Are wallet and ranked flows accessible without overwhelming new users?
4. Can future games fit without restructuring global navigation?
5. Are matchmaking and live match flows game-agnostic?

---

## Low-Fidelity Experience Wireframes

<!-- Archived source: docs/backup/individual-markdown-2026-07-15/WIREFRAMES.md -->

Status: Draft for review

Sprint: 1 - Product Identity & Design Foundation

Rules:

- Layout only.
- No colors.
- No typography styling.
- No visual polish.
- No implementation.
- Wireframes should tell the player story before they place interface blocks.

### Wireframe Philosophy

Skill Arena is not a collection of pages.

It is an ecosystem that moves a player from curiosity to confidence, from confidence to competition, and from competition to meaningful progress.

Every wireframe should answer:

- Why does this screen exist?
- What does it teach?
- What emotion should the player feel?
- What is the next action toward play, progress, or trust?
- What proof does the platform provide before asking for more commitment?

### Landing Story

The landing page should not feel like a marketing template. It should feel like entering the edge of an arena.

Narrative order:

```text
------------------------------------------------------------+
| Experience                                                 |
| The player immediately understands this is skill-based     |
| competition, not gambling, not casual arcade browsing.     |
|                                                            |
| Primary action: Enter the Arena                            |
+------------------------------------------------------------+
| Challenge                                                   |
| Show the nature of competition: timed decisions, fair       |
| rules, replay verification, ranked formats, future games.   |
+------------------------------------------------------------+
| Progress                                                    |
| Show how a player grows before spending money: practice,    |
| replay, tips, daily challenge, practice, trust movement.    |
+------------------------------------------------------------+
| Competition                                                 |
| Show the arena is alive: queues, rival movement, ranked     |
| ladders, tournaments, houses, seasons.                      |
+------------------------------------------------------------+
| Trust                                                       |
| Explain account security, replay integrity, wallet states,  |
| treasury review, and withdrawal transparency.               |
+------------------------------------------------------------+
| Community                                                   |
| Show belonging: houses, leaderboards, tournament stages,    |
| public achievement, verified match history.                 |
+------------------------------------------------------------+
| Join                                                        |
| Return to one clear invitation: create competitor account.  |
+------------------------------------------------------------+
```

Primary action:

- Enter the Arena Hub.

Why this exists:

- To prove the platform is worth trying before asking for registration.

What it teaches:

- Skill Arena is a progression ecosystem built on fair competition and trust.

### Boot Experience

```text
+------------------------------------------------------------+
| Skill Arena mark                                           |
| Small deterministic puzzle sequence                        |
| Every Move Matters                                         |
| Arena initializing                                         |
+------------------------------------------------------------+
```

Primary action:

- None. Continue automatically to the requested destination.

Why this exists:

- To make entering Skill Arena feel intentional and premium.

What it teaches:

- Precision and consequence are part of the platform identity.

### Guest Arena Hub Story

```text
+------------------------------------------------------------+
| Platform Preview                                           |
| Games / Practice / Ranked / Tournaments / Leaderboards     |
+------------------------------------------------------------+
| Available Exploration                                      |
| Public game information / rules / public competition       |
+------------------------------------------------------------+
| Protected Capabilities                                     |
| Wallet / ranked / withdrawals / private replay / progress  |
| Each lock names its requirement and value                  |
+------------------------------------------------------------+
| Selected Intent                                            |
| "Enter Ranked" -> secure account gate -> return here       |
+------------------------------------------------------------+
```

Primary action:

- Explore, then choose a meaningful platform action.

Why this exists:

- To teach the ecosystem before asking for registration.

What it teaches:

- Skill Arena is a platform of games, progression, competition, and trust.

### Register Story

Registration should feel like creating a competitor identity, not filling out an account form.

```text
+------------------------------------------------------------+
| Identity                                                    |
| Create competitor account                                  |
| Short message: your first step is practice, not deposit.    |
+------------------------------------------------------------+
| Step 1: Secure Account                                      |
| Email / password / country / age / terms                   |
+------------------------------------------------------------+
| Step 2: Verify Email                                        |
| Sent / resend / expired / complete                          |
+------------------------------------------------------------+
| Step 3: Player Identity                                     |
| Nickname / curated avatar / timezone / language             |
+------------------------------------------------------------+
| Step 4: Financial Assessment                                |
| Approved eligibility questions with explanations            |
+------------------------------------------------------------+
| Step 5: Identity Verification                               |
| Evidence / status / next action                             |
+------------------------------------------------------------+
| Verification Pending                                       |
| Practice available / live capabilities explain their locks  |
+------------------------------------------------------------+
| Primary action: Continue current step                       |
+------------------------------------------------------------+
```

Primary action:

- Create account.

Why this exists:

- To start a trusted competitor profile.

What it teaches:

- Registration is the start of a fair-play journey, not a payment funnel.

### Login Story

Login should feel like returning to progress.

```text
+------------------------------------------------------------+
| Return Context                                              |
| Welcome back                                                |
| Continue your competitive progress                          |
+------------------------------------------------------------+
| Secure Access                                               |
| Email                                                       |
| Password                                                    |
| MFA or recovery code when required                          |
+------------------------------------------------------------+
| Progress Reminder                                           |
| Last played / active challenge / ranked status / wallet     |
| status when available                                       |
+------------------------------------------------------------+
| Primary action: Log In                                      |
| Secondary: Forgot Password                                  |
+------------------------------------------------------------+
```

Primary action:

- Log in.

Why this exists:

- To restore the player to their next meaningful action.

What it teaches:

- Security protects competitive progress.

### Dashboard Story

Dashboard should feel like mission control for the next match.

```text
+------------------------------------------------------------+
| Player State                                                |
| Level / Trust / League / Season / House                     |
| Notification summary                                        |
+------------------------------------------------------------+
| Next Best Action                                            |
| Continue practice, daily challenge, training, ranked queue,  |
| tournament round, replay review, or wallet action.          |
+------------------------------------------------------------+
| Progress Since Last Session                                 |
| Skill improvement                                           |
| Trust movement                                              |
| Challenge progress                                          |
| Rank movement                                               |
+------------------------------------------------------------+
| Competition Pulse                                           |
| Rival updates                                               |
| Leaderboard movement                                        |
| Tournament countdown                                        |
| House progress                                              |
+------------------------------------------------------------+
| Trust And Wallet Snapshot                                   |
| Available / locked / pending / practice                     |
| Security or verification next step                          |
+------------------------------------------------------------+
| Recent Learning                                             |
| Last match summary                                          |
| Replay insight                                              |
| Suggested training focus                                    |
+------------------------------------------------------------+
```

Primary action:

- Continue the most relevant progress path.

Why this exists:

- To make the player want one more meaningful session.

What it teaches:

- What changed, what matters now, and what to do next.

### Wallet Story

Wallet should feel like a banking-grade trust surface, not a balance widget.

```text
+------------------------------------------------------------+
| Wallet Confidence                                           |
| Live balance, practice balance, available, locked, pending  |
| Clear separation between playable funds and funds in motion |
+------------------------------------------------------------+
| Recommended Financial Action                                |
| Deposit, withdraw, verify, review pending item, or export   |
+------------------------------------------------------------+
| Money Movement Timeline                                     |
| Deposit: provider session -> pending -> verified -> settled |
| Withdraw: request -> AML/risk -> treasury -> provider ->    |
| settlement -> ledger complete                               |
+------------------------------------------------------------+
| Payment Methods And Limits                                  |
| Available methods                                           |
| Limits                                                      |
| Verification requirements                                   |
+------------------------------------------------------------+
| Transaction History                                         |
| Filtered, auditable, exportable, statement-ready            |
+------------------------------------------------------------+
| Trust Education                                             |
| Why a balance is locked, pending, rejected, or available    |
+------------------------------------------------------------+
```

Primary action:

- Resolve the next wallet action required for live competition.

Why this exists:

- To make real-money movement understandable and trustworthy.

What it teaches:

- Every cent has a state, reason, and next step.

### Arena Hub Story

Arena Hub should feel like the player's home inside Skill Arena: wallet, profile, progression, notifications, and game modules in one central command space.

```text
+------------------------------------------------------------+
| Recommended Path                                            |
| Best next game/mode based on progression and eligibility    |
+------------------------------------------------------------+
| Mode First                                                  |
| Practice | Ranked | Tournament | Challenge | Training       |
+------------------------------------------------------------+
| Game Selection                                              |
| Maze Arena                                                  |
| Memory Arena                                                |
| Logic Arena                                                 |
| Pattern Arena                                               |
| Reaction Arena                                              |
| Future games                                                |
+------------------------------------------------------------+
| Eligibility And Stakes                                      |
| Trust requirements                                          |
| Wallet requirements                                         |
| Practice/live distinction                                   |
| Tournament requirements                                     |
+------------------------------------------------------------+
| Matchmaking Entry                                           |
| Queue, challenge, train, or spectate depending on mode      |
+------------------------------------------------------------+
| Learning Loop                                               |
| Recent replay insight                                       |
| Suggested skill focus                                       |
| Daily challenge continuation                                |
+------------------------------------------------------------+
```

Primary action:

- Choose the next mode and enter a game-agnostic matchmaking or training flow.

Why this exists:

- To connect player intent with the right competitive path.

What it teaches:

- Skill Arena supports multiple games through shared progression, trust, and competition systems.

### Wireframe Approval Questions

1. Does each wireframe tell a player story rather than simply arrange UI blocks?
2. Does each screen explain why it exists?
3. Does each screen teach the player something useful?
4. Does the pre-deposit journey prove value before asking for money?
5. Does every screen point toward play, improvement, trust, or competition?
6. Is Maze Arena present without becoming the whole platform?

---

## Design System Plan

<!-- Archived source: docs/backup/individual-markdown-2026-07-15/DESIGN_SYSTEM.md -->

Status: Draft for review

Sprint: 1 - Product Identity & Design Foundation

This document is planning only. It does not define final implementation code.

Before any implementation sprint, the product path must be:

```text
Product Documents
  -> Wireframes
  -> High-Fidelity UX Mockups
  -> Approval
  -> Implementation
```

The design system should not be implemented from text descriptions alone. Approved high-fidelity UX mockups must guide the actual visual execution.

### Design System Purpose

The design system must make Skill Arena feel like a premium competitive gaming platform and keep every future page aligned with the North Star:

Every session should leave the player feeling they made meaningful progress.

The system must support:

- Platform pages.
- Wallet and financial flows.
- Arena Hub.
- Maze Arena.
- Future games.
- Admin and treasury tools.

### Visual Direction

The visual language should communicate:

- Arena energy.
- Precision.
- Trust.
- Competition.
- Progress.

It should avoid:

- Casino styling.
- Generic SaaS dashboards.
- Toy-like game UI.
- Overloaded neon effects.
- One-note color palettes.

### Visual Inspirations

These references are not to be copied. They define the feeling Skill Arena should study before choosing colors, typography, spacing, motion, or component style.

#### Chess.com

What to learn:

- Daily return loops.
- Skill improvement as habit.
- Match history and analysis.
- Casual entry with deep competitive mastery.

What not to copy:

- Casual visual softness if it weakens premium arena energy.

#### FACEIT

What to learn:

- Competitive seriousness.
- Queue and matchmaking tension.
- Player status, ranking, and tournament identity.
- Esports credibility.

What not to copy:

- Density that makes onboarding feel intimidating.

#### Riot Client

What to learn:

- Game launcher as destination.
- Strong mode selection.
- Event energy.
- Player identity and account progression.

What not to copy:

- Heavy franchise-specific art direction.

#### Steam

What to learn:

- Library and hub mental model.
- Activity surfaces.
- Community proof.
- Durable account identity.

What not to copy:

- Store-first browsing behavior.

#### Apple Wallet

What to learn:

- Financial clarity.
- Confidence through restraint.
- Transaction readability.
- Strong state hierarchy.

What not to copy:

- Minimalism that strips away competitive emotion.

#### Reference Synthesis

Skill Arena should feel like:

- The competitive seriousness of FACEIT.
- The improvement loop of Chess.com.
- The destination quality of Riot Client.
- The ecosystem depth of Steam.
- The financial trust clarity of Apple Wallet.

It should not become a clone of any one reference.

### Color Strategy

Color roles should be semantic before decorative.

Required roles:

- Background.
- Surface.
- Elevated surface.
- Primary action.
- Secondary action.
- Success.
- Warning.
- Danger.
- Pending.
- Verified.
- Locked.
- Live balance.
- Practice balance.
- Rank/progression.

The palette should support future games without making Maze Arena the visual identity.

### Typography Strategy

Typography must support:

- Fast scanning.
- Competitive emphasis.
- Financial clarity.
- Dense admin data.
- Mobile readability.

Required scales:

- Display.
- Page title.
- Section title.
- Card title.
- Body.
- Caption.
- Data label.
- Numeric emphasis.

### Spacing And Grid

The layout system should support:

- App shell.
- Dashboard grids.
- Wallet banking flows.
- Game cards.
- Leaderboards.
- Tables.
- Match surfaces.
- Admin panels.

Spacing should prioritize clarity and repeat use over decorative whitespace.

### Component Inventory

Planned reusable components:

- App shell.
- Top navigation.
- Side navigation.
- Page header.
- Section header.
- Card.
- Stat tile.
- Progress meter.
- Button.
- Icon button.
- Input.
- Select.
- Checkbox.
- Toggle.
- Tabs.
- Segmented control.
- Table.
- Badge.
- Status pill.
- Alert.
- Toast.
- Dialog.
- Drawer.
- Tooltip.
- Empty state.
- Loading skeleton.
- Error panel.
- Success panel.
- Timeline.
- Stepper.
- Balance display.
- Transaction row.
- Game card.
- Queue status.
- Match summary block.
- Replay status block.

### State Requirements

Every interactive component must define:

- Default.
- Hover.
- Focus.
- Active.
- Disabled.
- Loading.
- Error.
- Success.

Financial components must additionally define:

- Pending.
- Settled.
- Rejected.
- Locked.
- Available.

Game components must additionally define:

- Waiting.
- Queued.
- Active.
- Victory.
- Defeat.
- Verified replay.
- Flagged replay.

### Motion Plan

Motion should be tied to product meaning:

- Match found.
- Countdown.
- Queue state.
- Rank movement.
- Trust Score change.
- Balance settlement.
- Replay playback.
- Victory/defeat.

Motion must support reduced-motion preferences.

### Icon Strategy

Icons should clarify:

- Play.
- Queue.
- Wallet.
- Deposit.
- Withdraw.
- Lock.
- Unlock.
- Shield/security.
- Replay.
- Trophy.
- Rank.
- Settings.
- Alert.
- Verified.

Icons should not replace labels where financial or security clarity is required.

### UI Library Structure

Future implementation target:

```text
frontend/components/ui
  app-shell
  navigation
  buttons
  forms
  feedback
  data-display
  wallet
  game
  replay
  layout
```

No components should be created until this plan is approved.

### Acceptance Criteria For Future Sprint 1 Implementation

When implementation begins after approval:

- Components must be reusable.
- Components must not contain Maze-only assumptions.
- Components must include all required states.
- Components must be documented by usage.
- Components must pass build/tests.
- Design review must confirm platform identity alignment.

### Approval Questions

1. Does this system support the product identity?
2. Does it support wallet trust and game energy at the same time?
3. Does it avoid looking like a generic SaaS CRUD app?
4. Does it support future games?
5. Does it make the next action clear?

---

## Game Economy

<!-- Archived source: docs/backup/individual-markdown-2026-07-15/GAME_ECONOMY.md -->

Status: Draft for review

Sprint: 1 - Product Identity & Design Foundation

This document defines the product relationship between practice play, live play, rewards, trust, deposits, withdrawals, tournament entries, prize pools, house rewards, and season rewards.

It is not an implementation document.

### Economy Purpose

The Skill Arena economy exists to support fair competition and meaningful progression.

It should never feel like gambling, pressure, or arbitrary reward distribution.

Every economic system must answer:

- Why does this exist?
- What behavior does it reward?
- How does the player understand it?
- How is it kept fair?
- How does it support trust?

### Economy Layers

```text
Practice
  -> Learning
  -> Replay insight
  -> Tutorial tips
  -> Daily challenge
  -> Practice progress
  -> Trust movement
  -> First deposit
  -> Live play
  -> Ranked
  -> Tournament entry
  -> Prize pool
  -> Rewards
  -> Withdrawal
  -> Return loop
```

### Practice Economy

Why it exists:

- Let players experience Skill Arena without financial risk.
- Prove the game loop before asking for money.
- Teach rules, replay, and improvement.

What it should reward:

- Completion.
- Learning.
- Clean play.
- Practice consistency.

What it should not do:

- Pretend practice rewards are withdrawable live money.
- Confuse practice balance with live balance.
- Pressure immediate deposit.

### Live Economy

Why it exists:

- Allow skill-based competition with real stakes.

Requirements:

- Live balance must be clearly separate from practice balance.
- Available balance must be separate from locked and pending funds.
- Every live movement must be auditable.
- Every financial operation must have an idempotent lifecycle.
- Players must understand fees, limits, verification, and settlement states.

### Rewards

Rewards should reinforce skill, trust, and return behavior.

Reward types may include:

- Challenge rewards.
- Ranked rewards.
- Tournament winnings.
- House rewards.
- Season rewards.
- Trust-based access unlocks.

Reward rules:

- Rewards must have clear eligibility.
- Rewards must show whether they are practice, locked, pending, or available.
- Rewards must be connected to verified outcomes.
- Rewards must not imitate gambling mechanics.

### Trust Score Relationship

Trust Score should influence access, not feel like a mystery score.

Trust can affect:

- Ranked eligibility.
- Tournament eligibility.
- Withdrawal review intensity.
- Limits.
- Challenge access.
- House progression.

Trust should be increased by:

- Verified fair play.
- Account security completion.
- Clean match history.
- Completed verification.
- Consistent replay-valid outcomes.

Trust should be protected by:

- Suspicious activity review.
- Replay integrity checks.
- AML/risk review.
- Device/session security.

### Deposits

Deposits should happen only after Skill Arena has earned enough player confidence.

Preferred pre-deposit journey:

```text
Practice match
  -> Replay insight
  -> Tutorial tips
  -> Daily challenge
  -> Practice progress
  -> Trust increase
  -> First deposit
```

Deposit product requirements:

- Explain why deposit is needed.
- Show payment method.
- Show provider session.
- Show pending state.
- Show verification and settlement.
- Show when funds become available.

### Withdrawals

Withdrawals are the strongest trust moment in the product.

Withdrawal product requirements:

- Show available balance.
- Show limits and verification requirements.
- Show pending state.
- Show AML/risk review when applicable.
- Show treasury approval.
- Show provider settlement.
- Show ledger completion.
- Explain rejection clearly.

The player should feel:

- "The platform is careful with money."

Not:

- "The platform is hiding my money."

### Tournament Entry And Prize Pools

Tournament economics must be transparent before entry.

Required product clarity:

- Entry requirement.
- Entry fee when applicable.
- Prize pool.
- Reward distribution.
- Bracket rules.
- Replay dispute rules.
- Withdrawal implications.
- Cancel/refund conditions.

Prize pools should feel prestigious, structured, and auditable.

### House Rewards

House rewards should create belonging and long-term motivation.

They should reward:

- Participation.
- Clean competition.
- Improvement.
- Contribution to house objectives.

They should not overpower individual skill competition.

### Season Rewards

Season rewards should create long arcs of progress.

Season economy should explain:

- Season duration.
- Ranking impact.
- Reward eligibility.
- Trust requirements.
- Tie-breakers.
- Claim process.
- Expiry or rollover rules when applicable.

### Economy Approval Questions

1. Does the economy prove value before asking for deposit?
2. Are practice and live balances impossible to confuse?
3. Does every reward have a clear reason?
4. Does Trust Score influence access transparently?
5. Are withdrawals designed as trust moments?
6. Can future games participate without changing the economy model?

---

## Game Rules

<!-- Archived source: docs/backup/individual-markdown-2026-07-15/GAME_RULES.md -->

Status: Draft for review

Sprint: 1 - Product Identity & Design Foundation

This is the product rulebook for Skill Arena games.

It is not backend architecture and it is not frontend design.

Every game should eventually have its own rules section that defines how players compete, how outcomes are verified, and how spectators, replays, latency, scoring, and disputes work.

### Rulebook Purpose

Game rules exist so players, designers, developers, support staff, treasury, and admins all understand the same competitive contract.

Every game rule must answer:

- Why does this rule exist?
- What does the player see?
- What does the opponent see?
- What does the spectator see?
- How is the rule enforced?
- How is the result verified?
- What happens when something goes wrong?

### Shared Game Rules

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

### Maze Arena Rules

Maze Arena is Skill Arena Game 1.

#### Legal Move

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

#### Blocked Move

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

#### Dependency

A dependency is a rule relationship where one route, node, or action must be completed before another becomes valid.

Dependencies create:

- Strategic planning.
- Puzzle readability requirements.
- Difficulty scaling.
- Replay learning moments.

Dependencies must be visible enough for skilled players to reason about them.

#### Difficulty

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

#### APCE

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

#### Seeds And Generation

Generated puzzles must be reproducible.

Required rule data:

- Seed.
- Generation hash.
- Puzzle hash.
- Difficulty profile.
- Rules version.
- Game version.

The product requirement is that a replay can be verified years later against the same rules and generation profile.

#### Replay Verification

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

#### Scoring

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

#### Disconnects

Disconnect rules must be explicit.

The rulebook should define:

- Grace period.
- Reconnect window.
- Whether the timer continues.
- What the opponent sees.
- What spectators see.
- When the match is forfeited.
- How replay and dispute evidence are preserved.

#### Draws

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

#### Spectator Visibility

Spectators may see:

- Match timer.
- Player progress.
- Completion percentage.
- Replay after completion.
- Public score data.

Spectators should not see hidden information that would give active competitors an unfair advantage.

#### Latency Spikes

Latency handling must protect competitive integrity.

Rules should define:

- What is client-side feedback only.
- What is server-authoritative.
- How delayed moves are accepted or rejected.
- What happens during severe latency.
- How disputes are reviewed.

### Future Game Rulebooks

Future games should receive their own sections:

- Memory Arena.
- Logic Arena.
- Pattern Arena.
- Reaction Arena.
- Any future game.

Each should define the same product-level contract before implementation.

### Approval Questions

1. Can a player understand what counts as a valid result?
2. Can support explain why a match was won, lost, blocked, invalid, or disputed?
3. Can replay verification be explained without backend code?
4. Can spectators understand what they are allowed to see?
5. Can future games follow the same rulebook structure?

---

## Arena Core

<!-- Archived source: docs/backup/individual-markdown-2026-07-15/ARENA_CORE.md -->

Skill Arena treats every client as compromised. Clients render state and submit player intent; the backend authenticates, authorizes, validates, applies, scores, signs, and settles.

### Core Rule

Games plug into Arena Core. Games do not directly mutate wallets, leaderboards, progression, trust, tournaments, challenges, or rewards.

### Backend Flow

1. Validate JWT and derive the actor user ID server-side.
2. Load the authoritative server session.
3. Verify session ownership or match participation.
4. Resolve the registered game module.
5. Submit client intent only, such as `click line_12`.
6. Game module validates and applies the action against server state.
7. Arena Core/store settles wallet, XP, trust, replay, challenges, tournaments, and audit.

### Game Module Contract

Backend modules implement `internal/arena/core.GameModule`.

Current module:

- `maze_arena` in `backend/internal/games/maze`
- `test_arena` in `backend/internal/games/testarena` for modularity tests only

Future modules should implement the same contract without calling wallet, payment, leaderboard, tournament, or challenge services directly.

### Manifests And Capabilities

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

### Contexts

Game modules receive one authoritative context object.

`SessionContext` carries session, actor, wallet, season, league, trust, house, tournament, practice, and configuration data.

`ActionContext` carries authenticated actor, session, action stream, sequence number, replay position, latency, and server receive time.

The client cannot override context values. Arena Core builds them from JWT-authenticated state.

### Event Bus

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

### Replay Rule

Replays store seed, rules version, game version, action stream, timing, and server signature. They do not trust client-provided board state or outcome.

### Seed Rules

- Practice: one unique seed per player/session.
- PvP: one shared seed per match, independent board state per player.
- Tournament: one shared seed per bracket match.
- Daily challenge: one shared seed per day.

### Client Must Never Submit

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

### Freeze Rule

Arena Core v1.0 is an extension boundary, not a rewrite target. Future work should add modules and capabilities through the existing interfaces.

---

## Arena Hub

<!-- Archived source: docs/backup/individual-markdown-2026-07-15/ARENA_HUB.md -->

Arena Hub is the authenticated player home for Skill Arena. A player logs into Skill Arena, not into an individual game.

### Sprint 2 Implementation Status

Status: Frozen as `sprint-2-v1.0-freeze` after the final regression audit.

The Arena Hub runtime is a server-backed player command surface. It contains no admin navigation, fabricated statistics, fixed player counts, simulated tournaments, or client-owned progression state.

Implemented player surfaces:

- Dynamic welcome, overall level, XP, league, rating, Trust Score, wallet summary, and unread notification count.
- Server-derived recommended action, daily objectives, competition eligibility, locked-state reasons, recent activity, and resumable activity.
- Capability-driven game directory loaded from the Arena Core registry.
- Read-only wallet summary and recorded ledger history.
- Editable competitor profile with server-side username, display name, country, language, and curated avatar validation.
- Durable notification center with all/unread/read/archived views.
- Support guidance, contact route, durable support-ticket creation, and ticket history.
- Server-backed settings for MFA status, sessions, devices, and owned revocation.
- Honest empty states when no tournament, replay, activity, notification, or transaction exists.

Access rules:

| Identity state | Available |
|---|---|
| Guest | Landing, game catalog and rules, public leaderboard, registration, and authentication |
| Registered and verified | Arena Hub, Practice, profile, notifications, support, wallet status, replay history, settings |
| Live eligible | Ranked/live capabilities only when backend KYC, account, profile, and competition rules approve entry |
| Privileged staff | No player-application admin UI; operations tooling remains a separate future application |

The frontend sends browser requests with protected cookies through `app/lib/api.ts`. Hub state is refreshed after player mutations. It does not poll. Notification creation is written to an append-only `notification_events` stream so the future Session Gateway can deliver updates without changing notification ownership or REST contracts.

Persistence:

- PostgreSQL production tables: `player_profiles`, `progression`, `game_modules`, `player_notifications`, `notification_events`, and `support_tickets`.
- Local development fallback: `arena_hub.json`.
- Profile updates synchronize `users` and `player_profiles` in one PostgreSQL transaction.
- Notification creation writes the notification and delivery event in one PostgreSQL transaction.
- Creating a support ticket emits a durable owned notification backed by the notification event stream.
- Game metadata is synchronized from registered Arena Core manifests; pages do not maintain a second game catalog.

Out of scope for this slice:

- Payment execution, deposits, withdrawals, and payment methods.
- KYC evidence capture and financial assessment.
- WebSocket delivery and presence.
- Authoritative ranked gameplay, tournament entry/brackets, and new game modules.
- Admin CRM.

### Sprint 2 Validation Report

Validation date: 2026-07-23.

| Gate | Status | Evidence |
|---|---|---|
| Design | Pass | Responsive Hub, Profile, Notifications, and Support proof captured for desktop, tablet, and mobile under `docs/proof/sprint-2-arena-hub/`. |
| Frontend | Pass | Dynamic Hub, player navigation, catalog, wallet status, challenges, tournaments, replay history, profile, settings, notifications, support, and honest empty/error/loading states use versioned APIs. |
| Backend | Pass | Normalized Arena Hub repositories, aggregate state, game registry sync, profile persistence, durable notifications/events, and support tickets are implemented. |
| Security | Pass | All private routes require an owned session; notification ownership, profile input, avatar allowlist, support categories, and cross-account denial are covered by integration tests. |
| API | Pass | Public and protected `/api/v1` contracts, examples, access rules, and errors are documented in this README. |
| Tests | Pass | `go test ./...`, PostgreSQL restart integration, Vitest, and the desktop/tablet/mobile Playwright journey pass. |
| Production | Pass for Sprint 2 code | `go vet ./...`, `go build ./...`, ESLint, TypeScript, and the Next.js production build pass. Production still requires deployment configuration and credentials. |
| Freeze | Pass | Final regression audit passed; the release is committed and tagged `sprint-2-v1.0-freeze`. |

Verification results:

- Go full suite: all packages passed; the database package completed in 129.995 seconds and server package in 9.244 seconds.
- PostgreSQL 17: fresh-cluster migration, normalized writes, restart persistence, notification event history, and support/game metadata checks passed.
- Go static/build: `go vet ./...` and `go build ./...` exited successfully.
- Frontend unit tests: 3 files and 4 tests passed.
- Frontend coverage baseline: 25.55% statements overall for the configured Sprint 1 and Hub scope; Dashboard is 72% and API helpers are 73.8%.
- Frontend static/build: ESLint passed with zero warnings, TypeScript passed, and Next.js generated all 23 player routes.
- Browser validation: all 15 Sprint 1 authentication and Sprint 2 Hub tests passed in desktop Chromium, tablet Chromium, and mobile Chromium.
- Browser proof: 12 full-page screenshots cover Dashboard, Profile, Notifications, and Support across all three viewports. Three complete journey videos are retained beside them.

Known deployment configuration:

- Set `SKILL_ARENA_SUPPORT_EMAIL` to the approved production support address.
- Run migration `003_arena_hub.sql` during deployment.
- The Session Gateway will consume `notification_events` in its scheduled slice; the Hub does not poll.

Final regression audit:

- Frozen Sprint 1 authentication UI, context, and E2E source files are unchanged.
- Registration, email verification, login, forgot/reset password, MFA enrollment, MFA login, recovery codes, session recovery, and logout all pass on desktop, tablet, and mobile.
- Seventeen player API routes consumed by the Hub were confirmed registered in `server.go` and documented in this README.
- The player frontend contains no Admin API call, navigation item, CRM component, or `/admin` route.
- Fresh PostgreSQL 17 migration and restart persistence passed again.
- The E2E test server uses elevated test-only login/register limits because all 15 tests share one loopback IP; production rate limits and their backend tests are unchanged.

Freeze decision: **SPRINT 2 APPROVED AND FROZEN.** Sprint 3 remains unimplemented until its planning and approval workflow begins.

### Arena Hub Owns

- Wallet
- Deposits
- Withdrawals
- Profile
- Avatar
- Overall XP
- Overall level
- Trust score
- Notifications
- Friends
- Houses
- Platform challenges
- Shop
- Settings
- Support

Game modules must not show or own wallet, deposits, withdrawals, KYC, treasury, or account security flows.

### Games Are Modules

Games are applications inside Skill Arena.

Current and future modules:

- Maze Arena
- Memory Arena
- Reaction Arena
- Logic Arena
- Chess Arena
- Sudoku Arena

When a player enters Maze Arena, they remain inside Skill Arena. They enter a focused game module with Maze-specific modes, stats, rankings, achievements, and replays.

### Maze Owns

- Maze home
- Practice
- Ranked
- Tournament play
- Maze replay
- Maze statistics
- Maze achievements
- Maze leaderboard

Maze does not own wallet, deposits, withdrawals, or profile security.

### Progression Split

Arena Hub has overall progression:

- overall level
- overall XP
- trust
- house
- season standing

Each game has game-specific progression:

- Maze level
- Maze rank
- Maze league
- Maze personal bests
- Maze achievements

Future games follow the same model.

### Leaderboards

Arena Hub leaderboards:

- overall players
- houses
- overall XP
- overall season ranking

Game leaderboards:

- Maze global
- Maze weekly
- Maze season
- Maze country

### Challenges

Arena challenges:

- play 3 games
- invite a friend
- complete verification

Game challenges:

- finish Maze under 30 seconds
- reach a combo threshold
- solve a target difficulty

### Navigation Principle

Landing Page -> Authentication -> Arena Hub -> Game Module -> Arena Hub

Back from a game returns to Arena Hub, not to the public landing page.

---

## Session Gateway

<!-- Archived source: docs/backup/individual-markdown-2026-07-15/SESSION_GATEWAY.md -->

Session Gateway is the single authenticated live connection for a logged-in client.

The client opens one WebSocket after login:

1. Login with REST.
2. Receive JWT.
3. Open one WebSocket.
4. Authenticate the socket with the JWT.
5. Receive all live events through that connection.

### REST Is For

- login
- register
- deposit
- withdraw
- settings
- profile
- avatar
- KYC
- password reset
- account security

These are request and response flows.

### WebSocket Is For

- matchmaking
- match found
- PvP countdown
- timers
- opponent progress
- replay spectating
- leaderboard movement
- notifications
- challenge progress
- tournament updates
- presence

These are live flows.

### Event Path

Game module -> Arena Core -> Event Bus -> Session Gateway -> Client

The client still sends intent only. The server remains authoritative.

Example:

Client sends:

```json
{
  "type": "game.action",
  "sessionId": "sess_123",
  "action": {
    "actionType": "click",
    "targetId": "line_17",
    "sequence": 4
  }
}
```

Server responds with authoritative state/event:

```json
{
  "type": "action_accepted",
  "scope": "game",
  "scopeId": "sess_123",
  "payload": {
    "accepted": true,
    "progress": 42
  }
}
```

The client must never send wallet changes, rewards, winner, score, trust, XP, or completion state.

### One Connection

Do not create one WebSocket per feature or game. One authenticated Session Gateway connection carries:

- game events
- matchmaking events
- tournament events
- notifications
- challenge progress
- presence
- live leaderboard updates

This keeps reconnect, presence, and authorization simple.

---

## Game Protocol

<!-- Archived source: docs/backup/individual-markdown-2026-07-15/GAME_PROTOCOL.md -->

### Game Registry

Game modules expose:

- Metadata contract
- Renderer contract
- Replay contract
- Tournament contract

Maze Arena is registered as Game #1. Future games should use the same contracts.

### Maze Arena Session

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

### Puzzle Generation

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

### Difficulty

Difficulty profile includes:

- Rating
- Line count
- Dependency depth
- Branching factor
- False-route rate
- Dead-end factor
- Cross dependencies
- Noise factor

### PvP Protocol

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

### Replay Protocol

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

---

## Authentication Flow

<!-- Archived source: docs/backup/individual-markdown-2026-07-15/AUTH_FLOW.md -->

### Registration

1. User submits email, password, ISO country code, date of birth, age confirmation, and required consent flags.
2. Backend enforces the password policy, age requirement, normalized email uniqueness, and allowed account state.
3. The dedicated identity repository creates the user and initial password-history record transactionally.
4. Backend creates a purpose-bound HMAC-signed token with an embedded expiry and stores only its SHA-256 hash.
5. A durable email job is queued for SMTP delivery. Local development writes a private `.eml` outbox artifact; production cannot enable the outbox.
6. User remains unverified and cannot log in until the token is consumed.

### Email Verification

1. User opens verification link.
2. Backend validates the purpose-bound signature and embedded expiry before querying the token repository.
3. The stored token hash must be unexpired and unused for the first successful verification.
4. User is marked verified and the action is audited.
5. Reopening the same authentic link after verification is an idempotent success; a used token cannot verify a different identity or perform another action.

### Login

1. User posts credentials.
2. Backend checks lockout state.
3. Backend verifies password.
4. Unverified, suspended, disabled, or temporarily locked accounts are rejected with a stable API error.
5. If MFA is enabled, backend returns a signed five-minute MFA challenge; TOTP or a one-time recovery code completes login.
6. Privileged users without MFA receive an enrollment-only session that can access only MFA setup, MFA confirmation, session status, and logout.
7. Backend stores the refresh session and sets the access JWT and refresh token in `HttpOnly`, `SameSite=Strict` cookies. Production cookies require `Secure`.
8. Tokens are never returned to browser JavaScript and are never stored in `localStorage`.

### Refresh

1. Browser submits the protected refresh cookie.
2. Backend validates its hash, account state, expiry, and session family inside a serializable transaction.
3. Old refresh token is revoked and a replacement token is issued in a new protected cookie.
4. Reuse of a rotated token revokes the entire refresh family and creates an audit event.
5. The frontend uses one shared in-flight refresh request to prevent concurrent browser requests from rotating the same token twice.

### Logout

1. Browser submits the protected session cookies.
2. Backend revokes the current refresh session and deletes its Redis session state.
3. Access and refresh cookies are expired immediately.
4. Audit log records the action.

### Password Reset

1. User requests reset.
2. Backend creates expiring one-time reset token.
3. Email job is queued.
4. User submits token and new password confirmation.
5. Backend checks password history.
6. Token consumption, bcrypt password update, password-history insertion, session revocation, and hash-chained audit events commit in one serializable PostgreSQL transaction.
7. Reuse, expiry, and malformed/tampered tokens return stable API errors.

### MFA

Supported:

- TOTP
- Recovery codes

Required roles:

- `super_admin`
- `admin`
- `treasury_manager`
- `fraud_analyst`

Safe migration:

- Existing privileged accounts can enroll MFA using an enrollment-only token.
- Enrollment-only tokens cannot access privileged routes.

### JWT Claims

Access token includes:

- `sub`
- `sid`
- `jti`
- `role`
- `typ=access`
- `iss=skill-arena-api`
- `aud=skill-arena-web`
- MFA verification state
- Optional enrollment-only flag
- Issued-at timestamp
- Expiry

### Rate Limiting

Rate limits protect:

- Login
- Registration
- Verification resend
- Password reset
- MFA confirm
- MFA login challenge
- Match creation
- Replay retrieval
- Withdrawals

Production rate limiting uses Redis; local development falls back to memory.

---

## Payment Flow

<!-- Archived source: docs/backup/individual-markdown-2026-07-15/PAYMENT_FLOW.md -->

### Provider Abstraction

Payment providers implement one interface:

- Create deposit session
- Request withdrawal
- Parse webhook
- Validate credentials
- Health check

Configured provider families:

- PayFast
- Ozow
- Card provider
- Bank EFT
- Future crypto provider

Provider-specific logic must not leak into wallet handlers.

### Deposit

Request requirements:

- Authenticated user
- Verified email
- Positive amount
- `Idempotency-Key`
- Provider/method/currency metadata

Lifecycle:

1. Deposit request
2. Provider session
3. Pending
4. Provider webhook/callback
5. Verification
6. Settlement
7. Ledger entry
8. Available live balance
9. Audit log

Invariant: wallet balance is not credited at deposit request time.

### Withdrawal

Request requirements:

- Authenticated user
- Verified email
- KYC when required
- Positive amount
- Available live balance
- Trust-tier limit
- `Idempotency-Key`

Lifecycle:

1. Withdrawal request
2. Pending withdrawal hold
3. AML/risk checks
4. Treasury approval or rejection
5. Provider payout
6. Settlement
7. Ledger withdrawal and fee entries
8. Wallet pending hold released
9. Audit log

Invariant: withdrawal is not final-debited at request time.

### Idempotency

All financial operations require `Idempotency-Key`.

Behavior:

- Same key and same request hash returns the existing operation.
- Same key and different request hash is rejected.
- Keys are recorded in the operation metadata and in production idempotency storage.

### Treasury Reconciliation

Treasury health compares:

- Player reserve
- Player liabilities
- House exposure
- Reserve totals

Financial flow tests verify that wallet balances reconcile with balance-changing ledger entries and that treasury remains solvent.

### AML

Risk inputs:

- Large withdrawal
- Velocity
- Country rules
- Trust tier

High-risk cases create AML review records and may escalate to fraud analyst workflow.

---

## API Reference

<!-- Archived source: docs/backup/individual-markdown-2026-07-15/API_REFERENCE.md -->

Base path: `/api/v1`

Browser authentication uses `HttpOnly`, `SameSite=Strict` access and refresh cookies. Service/native clients may send `Authorization: Bearer <token>`. Browser JavaScript must not persist either token.

Financial POST requests require `Idempotency-Key`.

### Public

| Method | Path | Purpose |
|---|---|---|
| GET | `/health` | Service health |
| GET | `/health/live` | Process liveness |
| GET | `/health/ready` | Identity and email dependency readiness |
| GET | `/api/v1/config/features` | Feature flags |
| GET | `/api/v1/platform/stats` | Public platform stats |
| GET | `/api/v1/platform/puzzle-preview` | Puzzle preview |
| GET | `/api/v1/leaderboard` | Public leaderboard |

### Authentication

| Method | Path | Purpose |
|---|---|---|
| POST | `/api/v1/auth/register` | Create account |
| POST | `/api/v1/auth/login` | Create protected cookie session or return MFA challenge |
| POST | `/api/v1/auth/mfa/challenge` | Complete MFA challenge and create protected session |
| POST | `/api/v1/auth/refresh-token` | Rotate protected refresh cookie |
| POST | `/api/v1/auth/logout` | Revoke current session and clear cookies |
| GET | `/api/v1/auth/session` | Recover current authenticated identity |
| GET | `/api/v1/auth/sessions` | List current and historical sessions |
| POST | `/api/v1/auth/sessions/revoke` | Revoke one owned session |
| GET | `/api/v1/auth/devices` | List registered devices |
| POST | `/api/v1/auth/devices/revoke` | Revoke device and its sessions |
| POST | `/api/v1/devices/fingerprint` | Register or refresh an authenticated device identity |
| POST | `/api/v1/auth/verify-email` | Consume email verification token |
| POST | `/api/v1/auth/resend-verification` | Send another verification link |
| POST | `/api/v1/auth/password-reset/request` | Request reset email |
| POST | `/api/v1/auth/password-reset/confirm` | Confirm reset token and new password |
| POST | `/api/v1/auth/mfa/setup` | Start TOTP setup |
| POST | `/api/v1/auth/mfa/confirm` | Confirm TOTP setup |
| POST | `/api/v1/auth/mfa/disable` | Disable MFA |

#### Authentication Request Contracts

`POST /api/v1/auth/register`

```json
{
  "email": "player@example.com",
  "password": "minimum 12 characters with uppercase, number, and symbol",
  "country": "ZA",
  "dateOfBirth": "1990-01-31",
  "acceptTerms": true,
  "acceptFairPlay": true
}
```

Returns `201 {"status":"verification_required","email":"player@example.com"}` after the verification email job is durably accepted.

`POST /api/v1/auth/login`

```json
{"email":"player@example.com","password":"..."}
```

Returns `200` with non-secret session state and protected cookies, or `202` when MFA is required:

```json
{"mfaRequired":true,"challengeToken":"signed-one-time-token","expiresIn":300}
```

`POST /api/v1/auth/mfa/challenge`

```json
{"challengeToken":"...","code":"123456"}
```

Use `recoveryCode` instead of `code` for one-time recovery. Success returns the same non-secret session body as login and sets protected cookies.

`POST /api/v1/auth/verify-email` accepts `{"token":"..."}`. `POST /api/v1/auth/resend-verification` and `POST /api/v1/auth/password-reset/request` accept `{"email":"player@example.com"}`. Resend and recovery requests use enumeration-resistant `202` responses.

`POST /api/v1/auth/password-reset/confirm`

```json
{"token":"...","password":"...","confirmPassword":"..."}
```

Success returns `204` and invalidates every existing session in the same transaction.

`POST /api/v1/auth/mfa/setup` returns `{"secret":"...","otpauthUrl":"otpauth://..."}` only to an authenticated session. `POST /api/v1/auth/mfa/confirm` accepts `{"code":"123456"}` and returns ten recovery codes once. Recovery codes are stored only as hashes. `POST /api/v1/auth/mfa/disable` requires password plus `code` or `recoveryCode`; privileged roles cannot disable MFA.

`POST /api/v1/auth/sessions/revoke` accepts `{"sessionId":"..."}`. `POST /api/v1/auth/devices/revoke` accepts `{"deviceId":"..."}` and revokes every session associated with the device.

Authentication endpoints return JSON errors in this shape:

```json
{
  "code": "AUTH_EMAIL_UNVERIFIED",
  "message": "verify your email before signing in"
}
```

Important status codes include `400` invalid request/token, `401` invalid credentials/session/MFA proof, `403` unverified or insufficient privilege, `409` identity/MFA conflict, `423` account lockout, `429` rate limit, and `503` identity or email dependency unavailable.

Successful login/session recovery returns non-secret identity state:

```json
{
  "authenticated": true,
  "mfaEnrollmentRequired": false,
  "expiresIn": 900,
  "user": {
    "id": "...",
    "email": "player@example.com",
    "role": "player",
    "emailVerified": true,
    "status": "active"
  }
}
```

### Identity

| Method | Path | Purpose |
|---|---|---|
| POST | `/api/v1/identity/kyc-submit` | Submit KYC |
| GET | `/api/v1/identity/kyc-status` | KYC status |

### Profile And Progression

| Method | Path | Purpose |
|---|---|---|
| GET | `/api/v1/profile` | Current player profile |
| POST | `/api/v1/profile` | Update owned competitor profile |
| GET | `/api/v1/progression` | XP, level, league, trust |
| GET | `/api/v1/achievements` | Player achievements |
| GET | `/api/v1/achievements/catalog` | Static achievement catalog |

### Arena Hub

| Method | Path | Authentication | Purpose |
|---|---|---|---|
| GET | `/api/v1/catalog/games` | Public | Registered game metadata, capabilities, availability, and rules summary |
| GET | `/api/v1/catalog/games/{id}` | Public | One registered game contract |
| GET | `/api/v1/hub` | Player | Aggregate owned Hub state |
| GET | `/api/v1/notifications?status=` | Player | Owned notifications; status may be unread, read, or archived |
| POST | `/api/v1/notifications/read` | Player | Mark one owned notification read |
| POST | `/api/v1/notifications/archive` | Player | Archive one owned notification |
| GET | `/api/v1/support/content` | Public | Support articles and configured contact destination |
| GET | `/api/v1/support/tickets` | Player | Owned support-ticket history |
| POST | `/api/v1/support/tickets` | Player | Create an owned support ticket |

`GET /api/v1/hub` returns server-derived state:

```json
{
  "generatedAt": "2026-07-23T06:00:00Z",
  "profile": {
    "userId": "player-id",
    "username": "competitor",
    "displayName": "Competitor",
    "country": "ZA",
    "language": "en"
  },
  "progression": {
    "xp": 0,
    "level": 1,
    "eloRating": 1200,
    "leagueTier": "Bronze",
    "trustScore": 100
  },
  "wallet": {
    "currency": "USD",
    "availableBalance": 0,
    "pendingDeposits": 0,
    "pendingWithdrawals": 0
  },
  "notifications": {"unread": 0, "total": 0},
  "objectives": [],
  "recommendedAction": {
    "id": "practice",
    "label": "Enter Practice",
    "actionUrl": "/games"
  },
  "recentActivity": [],
  "tournaments": [],
  "challenges": [],
  "games": [],
  "eligibility": {
    "emailVerified": true,
    "profileComplete": false,
    "mfaEnabled": false,
    "walletVisible": true,
    "liveEligible": false,
    "blockers": ["Complete your competitor profile."]
  }
}
```

Profile update request:

```json
{
  "username": "competitor_1",
  "displayName": "Competitor One",
  "avatarUrl": "strategist",
  "country": "ZA",
  "language": "en"
}
```

Notification state request:

```json
{"notificationId":"notification-id"}
```

Support ticket request:

```json
{
  "category": "account",
  "subject": "Account question",
  "message": "The support team needs enough detail to investigate this request."
}
```

Supported ticket categories are `account`, `security`, `gameplay`, `wallet`, and `responsible_gaming`. Player-owned endpoints return `401` without a valid session, `400` for invalid contracts, and `404` when an owned notification does not exist.

### Seasons

| Method | Path | Purpose |
|---|---|---|
| GET | `/api/v1/seasons/current` | Active season |
| GET | `/api/v1/seasons/leaderboard` | Season ranking |

### Wallet

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

### Treasury

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

### Games

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

### Calibration And House

| Method | Path | Purpose |
|---|---|---|
| POST | `/api/v1/calibration/start` | Start daily calibration |
| GET | `/api/v1/calibration/baseline` | Behavioral baseline |
| GET | `/api/v1/house/tiers` | House tiers |
| POST | `/api/v1/house/start` | Start house challenge |

### PvP

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

### Tournaments

| Method | Path | Purpose |
|---|---|---|
| GET | `/api/v1/tournaments` | List tournaments |
| POST | `/api/v1/tournaments/register` | Register |
| POST | `/api/v1/tournaments/submit-match` | Submit tournament match |
| GET | `/api/v1/tournaments/{id}` | Tournament detail |

### Replays

| Method | Path | Purpose |
|---|---|---|
| GET | `/api/v1/replays` | Player replay list |
| GET | `/api/v1/replays/{sessionId}` | Replay detail |
| GET | `/api/v1/admin/replays/{sessionId}` | Admin replay detail |

### Admin

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

---

## Database Schema

<!-- Archived source: docs/backup/individual-markdown-2026-07-15/DATABASE_SCHEMA.md -->

Production database: PostgreSQL.

Development fallback: JSON files under `backend/data/`, ignored from Git.

Migration sources: `backend/migrations/001_create_tables.sql`, `002_auth_normalized.sql`, and `003_arena_hub.sql`.

### Core Tables

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
| `player_profiles` | Public competitor identity and presentation preferences |
| `game_modules` | Arena Core manifest metadata and capability flags |
| `player_notifications` | Durable owned notification state |
| `notification_events` | Append-only notification delivery/event stream |
| `support_tickets` | Durable player support requests |

### Money Tables

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

### JSONB Fields

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

### Indexes

Important indexes:

- `idx_ledger_entries_user_created`
- `idx_game_sessions_user_created`
- `idx_auth_sessions_user`
- `idx_audit_logs_created`
- `idx_pvp_matches_queue`
- `idx_background_jobs_status`
- `idx_financial_idempotency_user_operation`
- `idx_player_profiles_username_lower`
- `idx_progression_rank`
- `idx_game_modules_availability`
- `idx_notifications_user_status_created`
- `idx_notification_events_user_sequence`
- `idx_support_tickets_user_updated`

### Repository Note

At freeze, PostgreSQL is authoritative. Authentication and Arena Hub domains use normalized repositories. Older domains continue through `store_snapshots` plus dedicated financial idempotency tables. This is intentionally transitional. The domain store API isolates callers so each remaining subsystem can later be normalized without changing handlers or business workflows.

---

## Backend Feature Freeze

<!-- Archived source: docs/backup/individual-markdown-2026-07-15/BACKEND_FREEZE.md -->

Status: Backend v1.0 feature freeze

The backend is frozen for business features. Future backend work is limited to bug fixes, security fixes, performance/scalability work, production operations, and frontend integration support.

### Architecture

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

### Freeze Boundary

Production uses PostgreSQL for authoritative persistence. JSON files are development-only fallback state. Users, auth tokens, auth sessions, password history, MFA settings, login security, devices, and hash-chained auth audit records use dedicated normalized PostgreSQL tables and serializable transactions. The remaining pre-freeze domains still use the documented transactional PostgreSQL snapshot boundary.

Important architecture note: the current PostgreSQL persistence is an intermediate production persistence layer. Domain boundaries remain isolated through the store and module contracts so Wallet, Replay, Users, Treasury, Matchmaking, Tournament, and Game subsystems can later migrate to dedicated normalized PostgreSQL repositories without changing REST handlers or business workflows.

### Configuration

Required production environment:

- `SKILL_ARENA_ENV=production`
- `SKILL_ARENA_DATABASE_URL=postgres://...`
- `SKILL_ARENA_REDIS_URL=redis://...`
- `SKILL_ARENA_JWT_SECRET`
- `SKILL_ARENA_PUZZLE_SECRET`
- `SKILL_ARENA_MFA_ENCRYPTION_KEY`
- `SKILL_ARENA_ALLOWED_ORIGINS`
- `SKILL_ARENA_COOKIE_SECURE=true`
- `SKILL_ARENA_PUBLIC_BASE_URL=https://...`
- `SKILL_ARENA_SUPPORT_EMAIL=support@...`
- `SKILL_ARENA_EMAIL_OUTBOX_ONLY=false`
- `SKILL_ARENA_SMTP_HOST`
- `SKILL_ARENA_SMTP_PORT`
- `SKILL_ARENA_EMAIL_FROM`

Provider credentials:

- Email: `SKILL_ARENA_SMTP_HOST`, `SKILL_ARENA_SMTP_USER`, `SKILL_ARENA_SMTP_PASS`
- PayFast: `SKILL_ARENA_PAYFAST_MERCHANT_ID`, `SKILL_ARENA_PAYFAST_PASSPHRASE`
- Ozow: `SKILL_ARENA_OZOW_SITE_CODE`, `SKILL_ARENA_OZOW_PRIVATE_KEY`
- Storage: `SKILL_ARENA_STORAGE_PROVIDER=s3`, `SKILL_ARENA_S3_ENDPOINT`, `SKILL_ARENA_S3_BUCKET`, `SKILL_ARENA_S3_ACCESS_KEY`, `SKILL_ARENA_S3_SECRET_KEY`

### Authentication

Implemented:

- Registration
- Login
- JWT access tokens
- Refresh token rotation
- Refresh-family replay detection and family revocation
- Logout/revoke
- Email verification with signed expiring one-time token
- Password reset with expiring one-time token
- Password history
- Account lockout and suspicious login audit
- TOTP MFA
- Recovery codes
- Privileged role MFA enforcement
- Session and device listing/revocation
- CSRF origin enforcement for cookie-authenticated writes
- Strict production CORS origin validation
- Redis-backed atomic rate limiting with in-memory development fallback

Privileged roles requiring MFA:

- `super_admin`
- `admin`
- `treasury_manager`
- `fraud_analyst`

Existing privileged users can receive an enrollment-only token and complete MFA setup without lockout. Enrollment-only tokens cannot access privileged routes.

### Roles

Role order is defined in `models/user.go`. Administrative actions are enforced through `RequireRole`.

Public leaderboard output hides privileged accounts.

### Wallet And Ledger

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

### Deposit Lifecycle

Deposit flow:

1. Client submits deposit request with `Idempotency-Key`.
2. Backend creates provider session.
3. Session enters provider/pending lifecycle.
4. Provider callback marks pending/verified/settled.
5. Settlement creates ledger entry.
6. Wallet available balance changes only after settlement.
7. Audit log records state transitions.

The backend must never directly credit a wallet at request time.

### Withdrawal Lifecycle

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

### Treasury

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

### AML And Risk

AML review inputs:

- Withdrawal velocity
- Large withdrawal threshold
- Country rules
- Trust tier limits
- Manual escalation target

AML cases are tied to withdrawal IDs and can be approved/rejected as part of the treasury lifecycle.

### Maze Arena

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

### PvP And Matchmaking

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

### Replay

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

### Workers

Workers handle:

- Replay export
- Email outbox
- Leaderboard recalculation
- Tournament reward tasks
- Telemetry aggregation
- Backup scheduling

Redis coordinates queue markers and job claiming.

### Storage

Development:

- Local filesystem object storage.

Production:

- S3-compatible storage.

Used for:

- Replay exports
- Backup snapshots
- Analytics exports
- Evidence/dispute artifacts

### Observability

Implemented primitives:

- Structured JSON logging
- Metrics counters/snapshots
- Health component records
- Worker health
- Queue stats

Production deployment should wire these to the platform monitoring stack.

### Security Model

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

### Infrastructure Dependencies

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

### Verification At Freeze

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

---

## Production Readiness

<!-- Archived source: docs/backup/individual-markdown-2026-07-15/PRODUCTION_READINESS.md -->

### Distributed Locking

The current JSON-backed single-server build does not require distributed locks. Before horizontal scaling, add a lock provider around matchmaking, wallet settlement, tournament payouts, and replay review transitions.

Recommended options:
- PostgreSQL advisory locks when Postgres becomes the primary store.
- Redis locks with short TTLs for matchmaking and live session operations.

Lock keys should be scoped by user, match, tournament, and wallet transaction group.

### Background Job Queue

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

### Secrets Management

Runtime configuration supports environment overrides. Sensitive values must remain in secure environment variables or a secrets manager, not committed config files.

Production target:
- JWT secrets from a secrets manager.
- SMTP credentials from a secrets manager.
- Payment/KYC provider keys from a secrets manager.
- Key rotation runbook.

### Disaster Recovery Drill

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

### Maintenance and Shutdown

Maintenance mode is controlled by environment-backed configuration:
- `SKILL_ARENA_MAINTENANCE_ENABLED`
- `SKILL_ARENA_MAINTENANCE_MESSAGE`
- `SKILL_ARENA_MAINTENANCE_ALLOW_SUPER_ADMINS`

During maintenance, new match creation, PvP queue entry, tournament registration, and house challenge starts are blocked. Existing match submissions continue.

The API now uses graceful shutdown to stop accepting new requests, cancel workers, let active work persist, and close the store cleanly.

### Backend Freeze Boundary

Backend architecture is frozen after this milestone. Allowed backend changes:
- bug fixes
- security fixes
- performance improvements
- new game modules
- additive API versions

Do not redesign APCE, replay format, puzzle engine, matchmaking, trust engine, admin architecture, role hierarchy, or session lifecycle without a new architecture review.

---

## Backup Strategy

<!-- Archived source: docs/backup/individual-markdown-2026-07-15/BACKUP_STRATEGY.md -->

### Current Local Persistence

The current development build uses JSON files under the configured data directory. Until PostgreSQL/object storage are introduced, production-like backups must archive the full data directory as one consistency unit.

### Required Backup Jobs

- Daily platform backup: archive users, wallets, ledger, sessions, progression, devices, audit logs, tournaments, PvP matches, telemetry, review cases, treasury, and metrics.
- Replay backup: copy finished game sessions plus telemetry and review cases to replay backup storage after completion.
- Tournament recovery backup: snapshot tournament, participant, match, submission, wallet-lock, and ledger files before bracket generation, before result transitions, and after payout settlement.

### Recovery Rules

- Restore ledger and wallet files together. Never restore one without the other.
- Restore tournament files and ledger files together for tournament incidents.
- Replay verification requires `puzzleSeed`, `difficultyProfile`, and `puzzleVersion`; backups must retain all three.
- Audit logs are append-only recovery evidence and must be included in every backup set.

### Production Target

- PostgreSQL daily snapshots with point-in-time recovery.
- Object storage lifecycle policy for replay artifacts and telemetry exports.
- Separate encrypted offsite copy for audit logs, replay archives, and tournament recovery snapshots.
- Monthly restore test using a clean environment.

---

## Implementation Audit

<!-- Archived source: docs/backup/individual-markdown-2026-07-15/IMPLEMENTATION_AUDIT.md -->

> Historical record: this section preserves the pre-Sprint audit and must not be used as current implementation status. The Vertical Production Roadmap, Authentication Flow, API Reference, Backend Freeze Reference, and Sprint Production Reports are authoritative. Statements below that describe MFA, PostgreSQL, Redis, or the Sprint 1 frontend as missing are intentionally retained only as audit history.

### Planning Sources Reviewed

The planning folder contains the full platform roadmap across founder governance, Codex rules, Phase 1 through Phase 9 specifications, admin duties, and UI handbooks. [Planning Inventory](#planning-inventory) contains a PDF-by-PDF inventory with phase, part, page count, and text excerpts.

### Current Build Status

Implemented MVP foundations:
- Go API with versioned `/api/v1` routes.
- Next.js frontend with registration, login, dashboard, wallet actions, leaderboard, maze play, progression, and achievements.
- Server-authoritative maze generation and move validation.
- JSON-backed local persistence for users, wallets, ledger entries, devices, sessions, progression, achievements, auth sessions, audit logs, telemetry, review cases, and metrics.
- JWT access tokens, refresh tokens, logout/session revocation, email-verification state, KYC submission state, device fingerprint capture, RBAC scaffolding, admin-only API routes, server-side rate limiting, and CORS for local frontend/API use.
- Expanded SQL migration target covering users, auth sessions, wallets, ledger, game sessions, devices, progression, achievements, and audit logs.
- Replay verification reports generated from stored server-side game sessions, including route validation, shortest-path analysis, efficiency, timing flags, player replay APIs, and admin replay detail access.
- House challenge tiers with server-owned eligibility, stake, payout, difficulty metadata, larger maze generation, player house APIs, dashboard start flow, and audit events.
- Treasury state, reserve coverage calculations, player liability tracking, house exposure, live house challenge reserve gates, public treasury status, admin treasury health, and house risk reports.
- Active season model, 90-day default season, season leaderboard, dashboard Season Center, and achievement catalog API.
- Tournament event model, default daily/weekly/monthly tournaments, registration with wallet locking, participant seed list, treasury checks for live prize pools, tournament APIs, and dashboard Tournament Center.
- Tournament bracket generation, match result reporting, entry-fee settlement, champion prize settlement, admin bracket/result APIs, and tournament match state in player details.
- Admin Operations page for user/KYC review, treasury health, house risk, tournament operations, result reporting, and audit log visibility.
- Daily calibration mode with no wallet/rank/reward impact, behavioral baseline persistence, player baseline API, admin baseline visibility, and dashboard calibration controls.
- PvP queue lifecycle with same-stake matching, locked entry stakes, shared server-generated maze, route submission, winner/refund settlement, platform fee accounting through retained pot, progression updates, player PvP APIs, dashboard queue controls, and PvP board submission.
- Phase 9 sidebar architecture added for Dashboard, Games, Challenges, Tournaments, Leaderboards, Wallet, Replays, Profile, and Settings.
- Game Hub introduced so Maze Arena is a featured game inside the platform instead of the homepage.
- Frontend Maze Arena interaction corrected toward the critical gameplay spec: clickable directional line objects, dependency blocking, red failed-line feedback, hidden/pixelated opponent maze, opponent progress, and estimated lines remaining.
- Adaptive Puzzle Complexity Engine (APCE) added. The existing 1-100 difficulty rating remains as a balancing band, while unlimited `complexityScore` now drives long-term scaling for line count, dependency depth, branch factor, false routes, dependency trees, cross dependencies, noise, dead ends, human solve estimates, and expected solve percentiles for top 1%, top 10%, and average players.
- Live procedural generation metadata added across sessions, PvP matches, tournament matches, and replay reports with saved puzzle seeds, difficulty profiles, lifecycle state, and puzzle/generator/difficulty/game-rules/replay version fields.
- Game session lifecycle formalized with validated CREATED, GENERATING, READY, ACTIVE, COMPLETED, CANCELLED, and EXPIRED transitions.
- Dedicated matchmaking service package added for queue selection, timeout expiration, duplicate active-match detection, self-match rejection, and activation decisions.
- Trust Score engine expanded to include account age, completed matches, replay review status, verification, device consistency, and withdrawal history; trust tier now gates withdrawal limits and existing eligibility checks.
- Anti-bot telemetry collection foundation added for click timing, intervals, mouse/touch movement, device fingerprint, reaction variance, accuracy, and failed/successful click counts.
- Review pipeline added for flagged replays with PENDING_REVIEW, MANUAL_REVIEW, APPROVED, and REJECTED transitions plus admin review-case APIs and audit logs.
- Metrics collection added for puzzle generation, replay reconstruction, matchmaking duration, completion time, failed clicks, and validation failure foundations.
- Durable local background job queue foundation added for replay exports, email sending, backup jobs, leaderboard recalculation, tournament reward payouts, and telemetry aggregation.
- Background worker manager added with replay, email, leaderboard, tournament, telemetry, and backup workers. Workers claim persisted jobs, retry with exponential backoff, persist failure state, support cancellation/requeue actions, write artifacts, and expose queue statistics.
- Global maintenance mode added with environment configuration for enabled state, message, and super-admin bypass. New match, queue, tournament registration, and house challenge entry points are blocked during maintenance while existing submissions continue.
- Admin background job dashboard APIs added for pending/running/completed/failed/cancelled job lists, queue statistics, retry, cancel, and requeue actions.
- Automated backup execution added through scheduled/manual backup jobs, backup records, backup verification, and admin backup history/manual backup endpoints.
- Disaster recovery validation added through the admin restore-validation endpoint and `cmd/recovery`, checking database, replay, configuration, and job queue restore inputs with pass/fail reports.
- Graceful shutdown added for API shutdown, worker cancellation, pending job persistence, metrics persistence, backup history persistence, and HTTP request draining.
- Central runtime configuration service added for difficulty, trust thresholds, withdrawal limits, replay thresholds, anti-bot settings, tournament defaults, house settings, rate limits, cache TTLs, and feature flags with environment overrides.
- Feature flag system added for Maze Arena, Memory Arena, Reaction Arena, Logic Arena, Marketplace, Guilds, and Streaming, plus `/api/v1/config/features`.
- In-memory cache layer added and wired to leaderboard reads, with configuration-ready TTLs for leaderboard/profile/season/config caching.
- Versioned WebSocket event contracts defined for `match_started`, `progress_updated`, `match_finished`, and `notification_created`.
- Central globally unique ID generator added with typed prefixes for sessions, matches, replays, audits, and generic objects.
- Standard API error response helpers added with stable error codes including `AUTH_INVALID_TOKEN`, `MATCH_NOT_FOUND`, `TRUST_TOO_LOW`, `HOUSE_LOCKED`, and `RATE_LIMITED`.
- Immutable configured Super Admins added for `geldenhuysj0106@gmail.com` and `skillarenagame@gmail.com`, with hierarchy support for Super Admin, Admin, Treasury Manager, Fraud Analyst, Support, Moderator, and Player.
- Admin role-management backend APIs added for super-admin role updates, admin suspension, and admin MFA reset audit requests. Super admins cannot be demoted through UI/API.
- System health backend snapshot added for API/database/queue/cache/storage/replay queue/active matches/online players/memory/backup/deployment status.
- System health now includes maintenance state, worker health, queue status, queue statistics, and backup status.
- Production readiness notes added for future distributed locking, secrets management, monthly disaster recovery restore drills, and backend freeze boundaries.
- PvP self-match safeguards hardened in backend queue logic and frontend match filtering.
- PvP response security now redacts opponent board lines, seeds, moves, and click history while preserving opponent progress metadata.
- Tournament match gameplay added with player-specific server-generated arrow-line boards, click replay submission, automatic winner selection after both players submit, bracket advancement, persisted tournament submissions, and player Tournament Center board loading.

### Phase Coverage

- Phase 1: Partially implemented. Auth, wallet, ledger events, maze sessions, leaderboard, progression, achievements, PvP matching/settlement, active season, season leaderboard, tournament registration, replay verification reports, and house challenge tiers exist. Full treasury, advanced governance, and production-grade compliance remain.
- Phase 2: Partially implemented. Public home, auth pages, player dashboard, PvP queue/play controls, tournament center, replay center, and admin operations page exist. Full navigation, dedicated wallet/profile screens, richer replay viewer, and full admin UX remain.
- Phase 3: Partially implemented. Wallet, ledger events, PvP stake/reward ledger flow, audit logs, reserve state, solvency coverage, house exposure, and schema coverage exist locally. Production-grade double-entry accounting, external reconciliation, provider integrations, AML, and financial reports remain.
- Phase 4: Partially implemented. JWT, refresh tokens, revocation, RBAC hierarchy, immutable super-admins, audit logs, device fingerprints, calibration baselines, rate limits, trust tiers, telemetry collection, review cases, system health snapshot, and basic risk signals exist. MFA, fraud engine scoring, SOC workflows, and advanced security monitoring remain.
- Phase 5: Partially implemented monolith foundation. API, Docker scaffolding, durable local job workers, backup execution, recovery validation, maintenance mode, graceful shutdown, and health reporting exist. PostgreSQL, Redis/distributed locks, event bus, object replay storage, CI/CD, observability, and microservice split remain.
- Phase 6: Not implemented. Multi-game SDK, AI personalization, analytics warehouse, marketplace, mobile expansion, and franchise systems remain.
- Phase 7: Partially implemented. Server-authoritative arrow-line generation/validation, APCE unlimited complexity scoring with expected solve percentiles, procedural versioning, replay reconstruction checks, replay integrity flags, trust engine foundation, telemetry collection, house tiers, tier difficulty metadata, adaptive risk recommendation, reserve gates, and calibration baselines exist. Final anti-bot scoring, AI solver, deeper replay intelligence, and economy risk engine remain.
- Phase 8: Partially implemented. Basic Maze Arena gameplay, PvP queue/match submission, replay center, house challenge start flow, Season Center, Tournament Center, MVP tournament brackets/payouts, daily calibration, admin operations page, background job APIs, backup APIs, and system health APIs exist. Richer tournament match gameplay, richer house challenge lifecycle, mobile/offline replay, and deeper admin UX remain.
- Phase 9: Partially implemented. Basic responsive UI exists, including PvP queue controls. Full design system, localization, theme system, app structure, admin UX, replay theater, and richer tournament/PvP ecosystem UI remain.

### Recommended Next Build Order

1. Implement WebSocket transport using the now-defined event contracts for opponent progress, matchmaking updates, tournament updates, and notifications.
2. Add MFA setup/verification/recovery and require MFA for admin actions and high-risk withdrawals.
3. Build the final anti-bot scoring engine using the telemetry now being collected.
4. Replace JSON persistence with PostgreSQL-backed repositories using the expanded migration schema.
5. Freeze backend architecture. Future backend changes should be limited to bug fixes, security fixes, performance improvements, new game modules, and additive API versions.
6. Continue approved Phase 9 UX implementation for landing, games hub, localization, theme system, house challenge UX, seasonal progression, admin job dashboard, and system health dashboard.
7. Replace JSON persistence with PostgreSQL/object storage when preparing for real production traffic; keep the public backend contracts stable while doing so.

### Detailed Outstanding Gap List

This is the current high-signal list of what is still missing from the PDF roadmap and planning inventory:

#### Product and Gameplay
- Dedicated Play/Game Lobby screen with featured Maze Arena, quick play, recommended queues, daily events, and challenge browsing.
- True PvP matchmaking rules beyond the service foundation: ELO bands, league eligibility, casual/ranked separation, rematch/rival logic, private challenges, friend challenges, reconnect UX, and websocket/live opponent progress.
- Backend gameplay model now uses server-authoritative arrow-line dependency validation for sessions, PvP, and tournament matches. Remaining work is richer procedural rule tuning, timeout/disconnect handling, and visual replay reconstruction.
- Tournament gameplay integration still needs richer UX, spectator mode, qualification paths, and dispute handling. Core playable bracket boards, player submissions, automatic winners, and bracket advancement now exist.
- Replay theater UI: visual route playback, speed controls, opponent comparison, suspicious route annotations, admin review workflow, and shareable replay links.
- House challenge lifecycle: dynamic challenge seeds, house-specific procedural rules, challenge history, tier progression screens, profitability tuning, and exploit detection.
- Daily/weekly challenge systems, seasonal objectives, event banners, patch/news center, and reward preview screens.
- Mobile-first web layout and future native mobile shell, including touch-first maze controls and offline replay viewing.

#### Wallet, Treasury, and Finance
- PostgreSQL-backed repository layer replacing local JSON files.
- Production double-entry ledger with debit/credit accounts, immutable transaction groups, system wallets, treasury reserves, and reconciliation reports.
- Payment provider integration for deposits, bank/payout provider integration for withdrawals, webhooks, failed payment handling, refunds, chargebacks, and settlement status.
- AML/KYC provider integration, withdrawal limits, manual review queues, compliance notes, and high-value approval workflows.
- Treasury allocation automation for player reserve, revenue reserve, season fund, championship fund, jackpot fund, and emergency reserve.
- Admin finance reports, downloadable statements, treasury variance reports, and reserve proof/audit views.

#### Security, Anti-Cheat, and Compliance
- MFA setup/verify/recovery codes and MFA enforcement for withdrawals/admin actions.
- Password reset, account lockout, session/device management UI, and suspicious login alerts.
- Anti-bot scoring engine with timing-model analysis, input entropy, replay anomaly scoring, solver detection, and risk queues. Telemetry collection now exists.
- Fraud center with investigation workflows, account restrictions, dispute handling, case notes, and staff role separation. Review-case foundations now exist.
- Terms/privacy/platform constitution acceptance, compliance logs, data retention rules, and admin approval trails for material treasury actions.

#### Platform Architecture
- WebSocket live updates, Redis/distributed locks, event bus, object storage for replay artifacts, and production SQL-backed durable job processing.
- CI/CD, container orchestration, observability dashboards, logs/metrics/tracing, backups, restore tests, and environment separation.
- Service boundaries for auth, wallet, gameplay, tournaments, replay, notifications, admin, and analytics.
- API client layer and frontend module split; current dashboard is still too large and should be decomposed into reusable screens/components.

#### Future Roadmap Phases
- Multi-game SDK and future games: Memory Arena, Logic Arena, Reflex Arena, Pattern Arena, Puzzle Arena.
- AI personalization, smart matchmaking, recommendations, analytics warehouse, and operational intelligence.
- Marketplace/store, cosmetics, premium season pass, sponsored events, clans, friends, rivals, hall of fame, public profiles, trophies, notifications, localization, and regional expansion.

---

## Planning Inventory

<!-- Archived source: docs/backup/individual-markdown-2026-07-15/PLANNING_INVENTORY.md -->

### .

- `Founder_Operations_Handbook_Enterprise_Edition_v1.pdf` (4 pages): Skill Arena Founder Operations Handbook - Enterprise Edition Version 1 Executive Operations Manual Executive Governance The founder is responsible for strategic direction, treasury oversight, regulatory compliance, platform integrity and executive approvals. No material treasury, reward, banking or infrastructure decision should occur without documented approval and audit records.The founder is responsible for strategic direction, treasury oversight, regulatory compliance, platform integrity and executive approvals. No material treasury, reward, banking or infrastructure decision should occur without documented approval and audit records.The founder is responsible for strategic direction, treasury oversight, regulatory compliance, platform integrity and executive approvals. No material treasury, reward, banking or infrastructure decision should occur without documented approval and audit

- `Skill_Arena_Big_Picture_Roadmap_For_Codex.pdf` (2 pages): Skill Arena - Big Picture Roadmap For Codex Executive Overview of the Entire Platform Vision What We Are Building Skill Arena is not a single game. It is a global skill-gaming ecosystem designed to support multiple competitive games under a unified platform. Maze Arena is Game #1. Future games include: - Memory Arena - Logic Arena - Reflex Arena - Pattern Arena - Puzzle Arena All games share: - Wallets - Rankings - Seasons - Tournaments - Legacy System - Hall Of Fame - Security Framework - Treasury & CRM Systems Core Mission Build the world's leading skill-based gaming ecosystem where skill determines outcomes, all gameplay is verifiable, every token is auditable and long-term sustainability is prioritized. Platform Principles - Skill Determines Outcomes - Every Live Match Is Replayable - Every Token Is Auditable - Every Challenge Is Verifiable - No Reward May Exceed Treasury Reserves -

- `Skill_Arena_CTO_Codex_Development_Charter_Enterprise_Edition_v1.pdf` (3 pages): Skill Arena CTO & Codex Development Charter - Enterprise Edition Version 1 Engineering Governance Manual Architecture Principles All critical business logic executes server-side. Services must be modular, scalable and independently deployable.All critical business logic executes server-side. Services must be modular, scalable and independently deployable.All critical business logic executes server-side. Services must be modular, scalable and independently deployable. Zero Trust Client Clients never calculate balances, rewards, rankings, treasury values, trust scores or challenge outcomes.Clients never calculate balances, rewards, rankings, treasury values, trust scores or challenge outcomes.Clients never calculate balances, rewards, rankings, treasury values, trust scores or challenge outcomes. Backend Standards Primary backend services implemented in Golang. AI and analytics services im


### Admin and Coder Duties

- `Admin and Coder Duties\Skill_Arena_Maintenance_and_Operations_Manual.pdf` (1 pages): Skill Arena Maintenance & Operations Manual Daily Review treasury, withdrawals, fraud alerts, server health and support tickets. Weekly Review profitability, challenge performance, infrastructure costs and backups. Monthly Test disaster recovery, review access controls and validate reserves. KPIs Monitor active users, retention, payouts, fraud rate, uptime and treasury coverage.


### codex structure plan

- `codex structure plan\Skill_Arena_Founder_Action_Plan_for_Codex.pdf` (1 pages): Skill Arena Founder Action Plan Accounts Set up cloud provider, payment provider, domain, email and GitHub organization. Legal Prepare Terms, Privacy Policy, Platform Constitution and dispute procedures. Treasury Define reserve rules, payout policies and accounting procedures. Codex Support Provide all phase documents, business rules and decision approvals. Launch Run beta testing, penetration testing and treasury validation before launch.

- `codex structure plan\Skill_Arena_Recommended_Tech_Stack_and_Codex_Rules.pdf` (1 pages): Skill Arena Recommended Technology Stack & Critical Development Rules Recommended Stack Frontend: Next.js + React + TypeScript. Mobile: React Native. Backend: Golang. AI Services: Python. Database: PostgreSQL. Cache: Redis. Storage: S3/MinIO. Infrastructure: Docker + Kubernetes. Critical Rules Client is zero trust. All rewards, rankings, balances, challenge outcomes and treasury calculations are server-side only. Security Use MFA, RBAC, audit logs, service authentication and encrypted communications. Development Order Build APIs, treasury and security first. UI follows stable backend services.


### Phase 1 overview and rules

- `Phase 1 overview and rules\Skill_Arena_Phase_1_Master_Specification.pdf` (3 pages): Skill Arena - Phase 1 Master Specification Version 1.0 - Combined Foundation Blueprint Contents Part 1 - Constitution, Vision, Token Economy & Wallet Architecture Part 2 - Progression, Leagues, ELO & Matchmaking Part 3 - PvP Arena, Replay System & Match Flow Part 4 - House Challenge Engine & Procedural Generation Part 5 - Seasons, Legacy, Achievements & Rewards Part 6 - Tournament System & Championships Part 7 - Treasury, CRM & Financial Reconciliation Part 8 - Security, Anti-Cheat & Platform Protection Part 9 - Sustainability, Governance & Compliance Part 1 - Foundation Constitution principles, server-authoritative architecture, platform vision, token economy, deposits, withdrawals, wallet architecture and security foundations. Part 2 - Progression XP levels, prestige, legacy points, house reputation, league structure, ELO rating, matchmaking and seasonal rankings. Part 3 - PvP Arena Ma


### Phase 1 overview and rules\part 1

- `Phase 1 overview and rules\part 1\Phase_1_Part_1_Business_Game_Design_Foundation.pdf` (1 pages): Skill Arena - Phase 1 Part 1 Business & Game Design Foundation Specification Chapter 1 - Skill Arena Constitution Purpose: Define the permanent principles governing the platform. 1. Skill Determines Outcomes. 2. Every Live Match Is Replayable. 3. Every Token Is Auditable. 4. Every Challenge Is Verifiable. 5. No Reward May Exceed Treasury Reserves. 6. Infinite Progression. 7. Fair Seasonal Competition. 8. Sustainable Growth. 9. Server Authority. 10. Platform Must Outlive Any Single Game. 11. Transparency. 12. Trust Above Everything. Server Authority Rule: The client is trusted only for display, audio, visual effects and user input. The server is trusted for wallets, rewards, rankings, gameplay validation, maze generation, challenge outcomes, replays, treasury operations and all business logic. Chapter 2 - Platform Vision Skill Arena is a global skill-gaming ecosystem. Launch Game: - Maze


### Phase 1 overview and rules\part 2

- `Phase 1 overview and rules\part 2\Phase_1_Part_2_Progression_Leagues_Matchmaking.pdf` (2 pages): Skill Arena - Phase 1 Part 2 Progression, Leagues, ELO & Matchmaking Specification Chapter 5 - Progression Framework The platform uses five progression systems: 1. XP Level 2. Skill Rating (ELO) 3. League Rank 4. House Reputation 5. Legacy Points These systems are independent and serve different purposes. XP Level System XP Levels never reset and have no cap. XP Sources: - Complete Match - PvP Victory - House Challenge Success - Tournament Participation - Seasonal Achievements Players continue progressing indefinitely. Prestige System After reaching milestone levels, players unlock Prestige. Example: Prestige I Prestige II Prestige III Prestige has no upper limit. Prestige is permanent and never resets. Legacy Points Legacy Points represent lifetime contribution to the ecosystem. Legacy Points are earned from: - Seasonal Participation - Tournament Success - House Challenge Success - PvP


### Phase 1 overview and rules\part 3

- `Phase 1 overview and rules\part 3\Phase_1_Part_3_PvP_Arena_Replay_System.pdf` (2 pages): Skill Arena - Phase 1 Part 3 PvP Arena, Match Flow, Replay & Disconnect Specification Chapter 6 - PvP Arena Overview PvP Arena is the core competitive system of Skill Arena. All future games must integrate with the PvP framework. Players compete using the same rules, same challenge seed and same starting conditions. Match Creation A match may be created through: - Ranked Queue - Casual Queue - Friend Challenge - Cross-League Challenge - Tournament Match Server generates Match ID and validates entry requirements. Entry Validation Server validates: - Account Status - Wallet Balance - League Eligibility - Ban Status - Tournament Qualification Client never performs final validation. Match Pot Calculation Example: Player A Entry = 10 Tokens Player B Entry = 10 Tokens Total Pot = 20 Tokens Platform Fee = 10% Platform Revenue = 2 Tokens Winner Receives = 18 Tokens Match Start Process Server: 1.


### Phase 1 overview and rules\part 4

- `Phase 1 overview and rules\part 4\Phase_1_Part_4_House_Challenge_Engine.pdf` (2 pages): Skill Arena - Phase 1 Part 4 House Challenge Engine & Procedural Generation Specification Chapter 7 - House Challenge Overview House Challenges are player-versus-platform competitions. Unlike PvP, House Challenges generate unique content for each player. No two players should receive the same challenge configuration. House Challenge Objectives The House Challenge system must: - Remain Fair - Remain Verifiable - Remain Profitable - Prevent Exploitation - Scale Infinitely Every challenge must be unique and auditable. Challenge Unlock Requirements Access is controlled through: - XP Level - Skill Rating - Match History - House Reputation House tiers unlock progressively. House Tiers Bronze House Silver House Gold House Diamond House Future tiers may be added dynamically. Higher tiers: - Cost More - Reward More - Require Greater Skill Unique Challenge Generation House challenges use: Player I


### Phase 1 overview and rules\part 5

- `Phase 1 overview and rules\part 5\Phase_1_Part_5_Seasons_Legacy_Rewards.pdf` (2 pages): Skill Arena - Phase 1 Part 5 Seasons, Legacy, Achievements & Reward Distribution Specification Chapter 8 - Season System Overview Seasons are the primary long-term engagement system. Every season runs for 90 days. Seasons provide: - Competition - Rankings - Rewards - Achievements - Championships Each season receives a unique identity and theme. Season Structure Season Duration: 90 Days Example: Season 1 Season 2 Season 3 At season end: - Seasonal rankings reset - Seasonal points reset Permanent progression remains. Season Points (SP) Season Points determine seasonal rankings. Sources: - PvP Victories - House Challenge Success - Tournament Participation - Seasonal Objectives - Seasonal Achievements SP resets at season end. Season Rewards Rewards scale with platform growth. Rewards are funded through: - Season Fund - Tournament Revenue Allocation - Platform Revenue Allocation Rewards must


### Phase 1 overview and rules\part 6

- `Phase 1 overview and rules\part 6\Phase_1_Part_6_Tournament_System_Championships.pdf` (2 pages): Skill Arena - Phase 1 Part 6 Tournament System, Championships & Qualification Specification Chapter 9 - Tournament Philosophy Tournaments are designed to create structured competition, increase player engagement and provide prestige-based progression. Tournament rewards must always be treasury-backed and sustainable. Tournament Types 1. Daily Tournaments 2. Weekly Tournaments 3. Monthly Championships 4. Seasonal Championships 5. World Championships Each tier increases prestige, difficulty and reward potential. Daily Tournaments Purpose: Frequent competition. Entry Fee: Default 5 Tokens. Duration: 24 Hours. Rewards scale according to participation. Weekly Tournaments Purpose: Mid-level competition. Entry Fee: Default 25 Tokens. Duration: 7 Days. Higher rewards and prestige than Daily Tournaments. Monthly Championships Purpose: Elite competition. Entry Fee: Default 100 Tokens. Duration: 1


### Phase 1 overview and rules\part 7

- `Phase 1 overview and rules\part 7\Phase_1_Part_7_Treasury_CRM_Financial_Ledger.pdf` (2 pages): Skill Arena - Phase 1 Part 7 Treasury, CRM, Financial Ledger & Reconciliation Specification Chapter 10 - Financial Philosophy Every token must be traceable. Every balance must be reconcilable. Every liability must be backed. The platform must always be able to explain: - Where money came from - Where money went - Who owns it - Which reserve it belongs to Treasury Architecture Separate financial pools: 1. Player Funds Reserve 2. Platform Revenue Reserve 3. Season Fund Reserve 4. Championship Fund Reserve 5. Jackpot Fund Reserve 6. Emergency Reserve Funds may not be mixed without authorization. Player Funds Reserve Represents money belonging to players. Must always remain fully backed. Cannot be used for: - Operating expenses - Marketing - Development costs Player liabilities take priority. Revenue Reserve Platform income generated from: - PvP Fees - Tournament Fees - Premium Passes - Futu


### Phase 1 overview and rules\part 8

- `Phase 1 overview and rules\part 8\Phase_1_Part_8_Security_AntiCheat_Architecture.pdf` (2 pages): Skill Arena - Phase 1 Part 8 Security Architecture, Anti-Cheat & Platform Protection Specification Chapter 11 - Security Philosophy Security is a core platform principle. The platform assumes: - Clients can be modified - Traffic can be inspected - Accounts can be compromised - Attackers will attempt exploitation Security must be designed proactively. Server Authoritative Architecture Client Trust Level = ZERO The client is trusted only for: - Display - Audio - Visual Effects - User Input The server controls: - Wallets - XP - Rankings - Match Results - Treasury - Rewards - Replays - House Challenges - Tournament Logic Client Security Rules The client may never: - Generate Rewards - Generate XP - Calculate Rankings - Create Tokens - Modify Balances - Determine Match Outcomes - Generate Challenge Seeds All business logic remains server-side. Authentication Framework Authentication Component


### Phase 1 overview and rules\part 9

- `Phase 1 overview and rules\part 9\Phase_1_Part_9_Sustainability_Governance_Compliance.pdf` (2 pages): Skill Arena - Phase 1 Part 9 Sustainability, Governance, Compliance & Growth Strategy Specification Chapter 12 - Sustainability Philosophy Skill Arena must be designed to survive long term. The platform shall prioritize: - Sustainability - Fairness - Security - Transparency - Growth Short-term profit must never threaten long-term survival. Revenue Model Primary Revenue Sources: - PvP Platform Fees - Tournament Fees - Premium Season Passes - Cosmetic Sales - Sponsored Events - Future Premium Features Revenue streams should be diversified. Reward Sustainability Rewards must scale with: - Platform Growth - Treasury Size - Active Player Base - Revenue Generation No reward may be promised without funding. Treasury Backing Rules Every reward system must be treasury backed. Examples: - Tournament Rewards - Championship Rewards - Seasonal Rewards - House Challenge Rewards Treasury limits overrid


### Phase 2 UI design

- `Phase 2 UI design\Phase_2_Master_Overview_Product_Architecture.pdf` (2 pages): Skill Arena - Phase 2 Master Overview Product & Platform Design Architecture Purpose of Phase 2 Transform the business architecture from Phase 1 into a complete product architecture defining screens, workflows, navigation and user experiences. Primary Objective Design a scalable platform capable of supporting millions of users, multiple games, global expansion and future mobile applications. Platform Areas Public Platform, Player Platform and Administration Platform. Public Platform Landing pages, registration, login, games showcase, leaderboards, tournaments and marketing content. Player Platform Dashboard, Wallet, Play, House Challenges, Tournaments, Replays, Profile, Achievements and Settings. Administration Platform CRM, Treasury, Security Operations, Compliance Center, Reporting and Support Systems. Document 1 Authentication, Registration, Verification, KYC, Onboarding and User Jour


### Phase 2 UI design\part 1

- `Phase 2 UI design\part 1\Phase_2_Part_1_Navigation_User_Journey_Information_Architecture.pdf` (2 pages): Skill Arena - Phase 2 Part 1 Platform Navigation, User Journey & Information Architecture Purpose This document defines how users move through the Skill Arena ecosystem. The objective is to create a scalable structure capable of supporting multiple games, millions of players and future platform expansion without redesigning navigation. Platform Areas The platform consists of three primary areas: 1. Public Area 2. Player Area 3. Administration Area Public Area Accessible without login. Contains: - Landing Page - About - Games - Tournaments - Leaderboards - Login - Register - Help Center Player Area Accessible after login. Contains: - Dashboard - Play - House Challenges - Tournaments - Leaderboards - Wallet - Profile - Settings - Notifications Administration Area Accessible only to authorized staff. Contains: - CRM - Treasury - User Management - Security Center - Dispute Resolution - Repla


### Phase 2 UI design\part 2

- `Phase 2 UI design\part 2\Phase_2_Document_2_Dashboard_Navigation_Enterprise_Spec.pdf` (2 pages): Skill Arena - Phase 2 Document 2 Dashboard & Navigation System - Enterprise Product Specification Document Purpose Define the primary authenticated player experience, dashboard architecture, navigation structure, widgets, permissions, security controls and user workflows. Business Objectives Provide a central command center for every player. Increase engagement, improve retention and surface all important platform actions within three clicks. Screen ID PLAYER_DASHBOARD_001 User Roles Standard Player, Verified Player, Premium Player, Founder, Tournament Participant, Moderator and Administrator. Desktop Layout Global Header, Left Navigation, Main Content Area, Right Activity Panel and Footer. Mobile Layout Bottom Navigation Bar, Compact Header, Swipeable Widgets and Mobile Notification Center. Global Header Components Logo, Search, Notifications, Wallet Summary, Profile Avatar, Quick Depos


### Phase 2 UI design\part 3

- `Phase 2 UI design\part 3\Phase_2_Document_3_Wallet_Store_Financial_Experience_Enterprise_Spec.pdf` (3 pages): Skill Arena - Phase 2 Document 3 Wallet, Store & Financial Experience - Enterprise Product Specification Document Purpose Define the complete financial user experience including deposits, withdrawals, token purchases, transaction history, wallet management, store purchases and financial security. Business Objectives Provide a trusted, transparent and secure financial experience while maintaining complete treasury reconciliation. Screen Group WALLET_001, DEPOSIT_001, WITHDRAWAL_001, STORE_001, TRANSACTION_HISTORY_001. Wallet Overview Screen Displays Live Wallet Balance, Demo Wallet Balance, Pending Deposits, Pending Withdrawals, Available Balance and Account Status. Live Wallet Component Shows real-money token balance. Values calculated server-side only. Demo Wallet Component Displays practice tokens used for learning and training. Deposit Screen Allows player to purchase tokens using sup


### Phase 2 UI design\part 4

- `Phase 2 UI design\part 4\Phase_2_Document_4_Game_Lobby_Matchmaking_Challenge_Selection_Enterprise_Spec.pdf` (3 pages): Skill Arena - Phase 2 Document 4 Game Lobby, Matchmaking & Challenge Selection - Enterprise Product Specification Document Purpose Define how players discover games, select challenges, enter matchmaking queues and access tournaments, PvP and House Challenge content. Business Objectives Reduce friction between login and gameplay while ensuring fair matchmaking and sustainable competition. Primary Screen Group GAME_LOBBY_001, MATCHMAKING_001, HOUSE_SELECTION_001, TOURNAMENT_ENTRY_001. Game Lobby Overview Central hub where players select a game mode and review available activities. Lobby Sections Featured Game, Quick Play, House Challenges, Tournaments, Daily Events, Recommended Activities and News. Featured Game Area Maze Arena displayed as primary game. Future games added without redesigning the lobby. Quick Play Section One-click access to the player's most appropriate PvP queue based on


### Phase 2 UI design\part 5

- `Phase 2 UI design\part 5\Phase_2_Document_5_Live_Gameplay_PvP_House_Replay_Enterprise_Spec.pdf` (2 pages): Skill Arena - Phase 2 Document 5 Live Gameplay Experience, PvP Arena, House Challenge Interface & Replay Integration Document Purpose Define the live gameplay experience, player interface, PvP arena layout, House Challenge layout, HUD systems, replay integration and post-match workflow. Business Objectives Create a competitive, fair and highly engaging gameplay experience while maintaining server-authoritative validation. Primary Screens LIVE_PVP_001, LIVE_HOUSE_001, MATCH_RESULTS_001, REPLAY_VIEWER_001. Gameplay Philosophy Skill-based gameplay with equal conditions, transparent outcomes and replay verification. PvP Arena Layout Main Maze Area, HUD Bar, Timer, Lives Counter, Progress Indicator, Opponent Progress Panel and Match Status Panel. House Challenge Layout Main Challenge Area, Difficulty Indicator, House Tier Indicator, Lives Counter, Progress Indicator and Dynamic Timer. HUD Com


### Phase 2 UI design\part 6

- `Phase 2 UI design\part 6\Phase_2_Document_6_Tournament_Center_Championships_Spectator_Enterprise_Spec.pdf` (3 pages): Skill Arena - Phase 2 Document 6 Tournament Center, Championships, Qualification System & Spectator Experience Document Purpose Define tournament discovery, registration, qualification, championship participation, brackets, rewards and spectator experiences. Business Objectives Create a competitive ecosystem that drives engagement, retention, prestige and seasonal participation. Primary Screens TOURNAMENT_CENTER_001, TOURNAMENT_DETAILS_001, REGISTRATION_001, BRACKETS_001, CHAMPIONSHIP_001, SPECTATOR_001. Tournament Center Overview Central location for all competitive events, championships and seasonal competitions. Tournament Categories Daily Tournaments, Weekly Tournaments, Monthly Championships, Seasonal Championships and World Championships. Tournament Browser Displays event name, start date, entry fee, qualification requirements, player count and prize pool. Featured Events Section H


### Phase 2 UI design\part 7

- `Phase 2 UI design\part 7\Phase_2_Document_7_Profile_Achievements_Legacy_HallOfFame_Enterprise_Spec.pdf` (3 pages): Skill Arena - Phase 2 Document 7 Profile System, Achievements, Legacy Progression, Hall of Fame & Trophy Cabinet Document Purpose Define player identity, progression display, achievements, legacy tracking, founder status, trophies and public profile systems. Business Objectives Increase player retention, prestige, social recognition and long-term engagement. Primary Screens PROFILE_001, ACHIEVEMENTS_001, LEGACY_001, TROPHY_CABINET_001, HALL_OF_FAME_001. Profile Overview Public and private player profile displaying identity, progression and accomplishments. Profile Components Avatar, Username, Country, League, ELO Rating, XP Level, Legacy Rank, Founder Status and Activity Statistics. Player Statistics Matches Played, Wins, Losses, Win Rate, House Challenges Completed, Tournaments Entered and Lifetime Progression. Avatar System Players may customize profile appearance through unlockable co


### Phase 2 UI design\part 8

- `Phase 2 UI design\part 8\Phase_2_Document_8_CRM_Treasury_Admin_Compliance_Enterprise_Spec.pdf` (3 pages): Skill Arena - Phase 2 Document 8 CRM Portal, Treasury Console, User Administration, Security Operations & Compliance Center Document Purpose Define the internal operational systems used by administrators, finance teams, compliance officers, support staff and security personnel. Business Objectives Provide complete visibility into players, finances, security events, compliance workflows and operational health. Primary Portals CRM_001, TREASURY_001, USER_ADMIN_001, SECURITY_CENTER_001, COMPLIANCE_001, SUPPORT_001. CRM Overview Central management interface for player accounts, activity history, support actions and account administration. CRM Components Player Search, Account Overview, Activity Timeline, Wallet Summary, Match History, Support Notes and Risk Flags. User Administration Create, suspend, restrict, verify and review user accounts according to permission levels. Player Profile Man


### Phase 3 crm

- `Phase 3 crm\Phase_3_Master_Overview_Treasury_Financial_Architecture.pdf` (2 pages): Skill Arena - Phase 3 Master Overview Treasury, Financial Infrastructure, Compliance & Risk Architecture Phase 3 Purpose Define the complete financial backbone of Skill Arena including treasury controls, wallets, ledgers, compliance, risk management and reporting. Primary Objective Ensure every token, reward, reserve and liability is fully auditable, traceable and financially backed. Financial Philosophy Every token must have an owner. Every balance must reconcile. Every liability must be covered by reserves. Document 1 Treasury, Wallet & Ledger Architecture. Defines reserves, wallet structures, double-entry accounting and treasury foundations. Document 2 Database Schema & Financial Data Architecture. Defines tables, relationships, transaction models, ledger records and audit storage. Document 3 Reconciliation Engine, Solvency Monitoring & Treasury Health Framework. Defines reserve valid


### Phase 3 crm\part 1

- `Phase 3 crm\part 1\Phase_3_Document_1_Treasury_Wallet_Ledger_Architecture.pdf` (3 pages): Skill Arena - Phase 3 Document 1 Treasury, Wallet & Ledger Architecture Specification Document Purpose Define the financial backbone of Skill Arena including treasury architecture, wallet systems, token accounting, reserves and ledger design. Business Objective Ensure every token is auditable, every liability is backed and every transaction is traceable. Financial Philosophy Every token must have an owner. Every balance must reconcile. Every liability must be treasury-backed. Core Financial Components Treasury Engine, Wallet Engine, Ledger Engine, Reconciliation Engine, Audit Engine and Compliance Engine. Treasury Structure Player Funds Reserve, Revenue Reserve, Season Reserve, Championship Reserve, Jackpot Reserve and Emergency Reserve. Player Funds Reserve Represents liabilities owed to players. Must remain fully backed and segregated from operating funds. Revenue Reserve Stores platfo


### Phase 3 crm\part 2

- `Phase 3 crm\part 2\Phase_3_Document_2_Database_Schema_Transaction_Models_Treasury_Data_Architecture.pdf` (2 pages): Skill Arena - Phase 3 Document 2 Database Schema, Financial Tables, Transaction Models & Treasury Data Architecture Document Purpose Define the core database structures required to support treasury management, wallets, transactions, accounting, compliance and auditing. Architecture Philosophy Database design must prioritize auditability, scalability, traceability and financial integrity. Primary Database Domains Users, Wallets, Treasury, Transactions, Tournaments, Replays, Compliance, Security and Reporting. Users Table Stores player identity, account status, profile references, verification status and role assignments. Wallets Table Stores wallet identifiers, balances, wallet types, status flags and ownership references. Wallet Types Live Wallet, Demo Wallet, Locked Wallet, Bonus Wallet and System Wallet. Treasury Accounts Table Stores reserve balances including Player Reserve, Revenue


### Phase 3 crm\part 3

- `Phase 3 crm\part 3\Phase_3_Document_3_Reconciliation_Engine_Solvency_Treasury_Health.pdf` (3 pages): Skill Arena - Phase 3 Document 3 Reconciliation Engine, Reserve Calculations, Solvency Monitoring & Treasury Health Framework Document Purpose Define how the platform validates financial accuracy, monitors treasury health and guarantees reserve integrity. Business Objective Ensure every player liability is backed and every financial discrepancy is detected immediately. Reconciliation Philosophy The platform must continuously prove that wallet balances, treasury balances and external financial balances agree. Core Components Reconciliation Engine, Solvency Engine, Treasury Health Engine, Alert Engine and Audit Engine. Reconciliation Scope Player Wallets, Treasury Accounts, Payment Providers, Bank Accounts, Tournament Pools and Reward Pools. Real-Time Reconciliation Critical balances validated continuously against internal records. Daily Reconciliation Comprehensive reconciliation process


### Phase 3 crm\part 4

- `Phase 3 crm\part 4\Phase_3_Document_4_Payment_Providers_AML_Compliance_Architecture.pdf` (3 pages): Skill Arena - Phase 3 Document 4 Payment Provider Integration, Deposits, Withdrawals, AML Controls & Financial Compliance Architecture Document Purpose Define how external payment providers integrate with the platform and how deposits, withdrawals, AML monitoring and compliance controls operate. Business Objective Provide secure global payment processing while maintaining compliance, fraud prevention and treasury integrity. Payment Architecture Payment Gateway Layer, Deposit Engine, Withdrawal Engine, Compliance Engine, AML Engine and Reconciliation Engine. Supported Payment Methods Card Payments, Bank Transfers, Digital Wallets, Regional Payment Providers and future integrations. Provider Integration Model All providers communicate through a centralized payment abstraction layer. Deposit Workflow Player Initiates Deposit → Provider Processing → Confirmation → Ledger Entry → Wallet Credi


### Phase 3 crm\part 5

- `Phase 3 crm\part 5\Phase_3_Document_5_Treasury_Operations_Dashboards_Reporting_Framework.pdf` (2 pages): Skill Arena - Phase 3 Document 5 Treasury Operations Center, Financial Dashboards, Reporting Engine & Executive Monitoring Framework Document Purpose Define the operational command center used to monitor treasury health, financial performance, liabilities, reserves and business metrics. Business Objective Provide executives, finance teams and operators with real-time visibility into the financial condition of the platform. Treasury Operations Center Centralized dashboard for monitoring all treasury activity, reserves, liabilities and financial performance. Primary Dashboard Groups Executive Dashboard, Treasury Dashboard, Liability Dashboard, Revenue Dashboard, Compliance Dashboard and Risk Dashboard. Executive Dashboard High-level overview of platform health, growth, solvency, revenue and treasury status. Key Executive Metrics Active Players, Deposits, Withdrawals, Revenue, Treasury Heal


### Phase 3 crm\part 6

- `Phase 3 crm\part 6\Phase_3_Document_6_Fraud_Risk_Management_Dispute_Resolution_Architecture.pdf` (2 pages): Skill Arena - Phase 3 Document 6 Fraud Detection, Risk Management, Dispute Resolution & Financial Investigation Architecture Document Purpose Define the systems used to detect fraud, manage financial risk, investigate suspicious activity and resolve player disputes. Business Objective Protect players, treasury reserves and platform integrity while maintaining transparency and fairness. Fraud Management Philosophy Every financial event, match result and reward must be verifiable and auditable. Core Systems Fraud Detection Engine, Risk Scoring Engine, Investigation Center, Dispute Resolution Center and Case Management System. Fraud Detection Engine Continuously evaluates player activity, financial transactions and gameplay behaviour. Risk Categories Financial Fraud, Account Abuse, Collusion, Multi-Accounting, Bonus Abuse, Payment Abuse and Match Manipulation. Risk Scoring Framework Each ac


### phase 4 security prevention

- `phase 4 security prevention\Phase_4_Master_Overview_Security_AntiCheat_Infrastructure_Architecture.pdf` (2 pages): Skill Arena - Phase 4 Master Overview Security, Anti-Cheat, Fraud Prevention & Infrastructure Protection Architecture Phase 4 Purpose Define the complete security architecture protecting users, gameplay, infrastructure, treasury systems and platform operations. Primary Objective Ensure the platform remains secure, fair, resilient and resistant to exploitation. Security Philosophy Client Trust Level = Zero. Server Authoritative Systems. Zero Trust Architecture. Continuous Verification. Document 1 Authentication, Session Management & Zero Trust Security Architecture. Document 2 API Security, Service Security, Infrastructure Security & Network Protection Architecture. Document 3 Anti-Cheat Engine, Gameplay Validation, Replay Verification & Match Integrity Architecture. Document 4 Bot Detection, Device Fingerprinting, Multi-Account Detection & Behavioral Security Architecture. Document 5 Sec


### phase 4 security prevention\part 1

- `phase 4 security prevention\part 1\Phase_4_Document_1_Authentication_Session_Zero_Trust_Architecture.pdf` (2 pages): Skill Arena - Phase 4 Document 1 Authentication, Session Management & Zero Trust Security Architecture Document Purpose Define the identity, authentication and session security architecture for the entire Skill Arena ecosystem. Business Objective Ensure only authorized users gain access while preventing account compromise, session hijacking and unauthorized activity. Security Philosophy Client Trust Level = Zero. Every request must be authenticated, validated and authorized. Zero Trust Architecture No user, device, session or service is trusted automatically. Verification occurs continuously. Identity Management Central identity service responsible for registration, authentication, authorization and account lifecycle management. Authentication Methods Email/Password, Multi-Factor Authentication, Device Verification and Future SSO Integrations. Password Security Strong password policies,


### phase 4 security prevention\part 2

- `phase 4 security prevention\part 2\Phase_4_Document_2_API_Service_Infrastructure_Network_Security.pdf` (3 pages): Skill Arena - Phase 4 Document 2 API Security, Service Security, Infrastructure Security & Network Protection Architecture Document Purpose Define how APIs, backend services, servers, networks and cloud infrastructure are protected against unauthorized access and attacks. Business Objective Ensure platform availability, confidentiality, integrity and resilience under normal and hostile conditions. Security Philosophy All services operate under Zero Trust principles. Every request is authenticated, authorized and monitored. API Security Architecture Central API Gateway, Authentication Layer, Authorization Layer, Rate Limiting Engine and Audit Layer. API Authentication JWT validation, service authentication, token verification and request signing. API Authorization Role-based and permission-based access control for every endpoint. Rate Limiting Prevent abuse, scraping, brute-force attacks


### phase 4 security prevention\part 3

- `phase 4 security prevention\part 3\Phase_4_Document_3_AntiCheat_Gameplay_Validation_Replay_Integrity.pdf` (3 pages): Skill Arena - Phase 4 Document 3 Anti-Cheat Engine, Gameplay Validation, Replay Verification & Match Integrity Architecture Document Purpose Define how gameplay is validated, cheating is prevented, replays are verified and match integrity is maintained. Business Objective Ensure every victory, defeat, reward and ranking outcome is earned through legitimate gameplay. Security Philosophy The server is the authority for all gameplay outcomes. The client is only an input and display layer. Anti-Cheat Engine Central system responsible for detecting, preventing and investigating gameplay manipulation. Gameplay Validation Engine Validates every move, timer event, life deduction, completion event and reward calculation server-side. Server Authoritative Design The server determines challenge state, progress, completion and rewards. Client Trust Level Zero. No gameplay result generated by the clie


### phase 4 security prevention\part 4

- `phase 4 security prevention\part 4\Phase_4_Document_4_Bot_Detection_Device_Fingerprinting_Behavioral_Security.pdf` (3 pages): Skill Arena - Phase 4 Document 4 Bot Detection, Device Fingerprinting, Multi-Account Detection & Behavioral Security Architecture Document Purpose Define systems used to identify bots, detect multi-account abuse, analyze player behaviour and strengthen platform security. Business Objective Protect competitive integrity, prevent abuse and ensure all players compete fairly. Security Philosophy Trust behaviour, not claims. Every account, device and gameplay pattern is continuously evaluated. Behavioral Security Engine Central engine responsible for behavioural analysis and anomaly detection. Bot Detection Framework Detect scripted gameplay, automation tools, macros and non-human interactions. Bot Detection Signals Input timing, reaction patterns, completion consistency, navigation behaviour and challenge interactions. Human Verification Models Compare player activity against established hum


### phase 4 security prevention\part 5

- `phase 4 security prevention\part 5\Phase_4_Document_5_SOC_Incident_Response_Threat_Intelligence.pdf` (2 pages): Skill Arena - Phase 4 Document 5 Security Operations Center (SOC), Incident Response, Threat Intelligence & Vulnerability Management Architecture Document Purpose Define the operational security framework used to monitor, detect, investigate and respond to security threats. Business Objective Protect platform availability, user accounts, treasury assets and operational integrity through continuous security operations. Security Operations Center Central command center responsible for security visibility and threat response. SOC Responsibilities Monitoring, Detection, Investigation, Containment, Recovery and Post-Incident Analysis. Security Monitoring Framework Collect and analyze logs, alerts, authentication events, infrastructure events and application activity. Threat Intelligence Platform Aggregate internal and external threat indicators for proactive defense. Threat Intelligence Sourc


### phase 5 tech spec and engineer

- `phase 5 tech spec and engineer\Phase_5_Master_Overview_Technical_Architecture_Blueprint.pdf` (2 pages): Skill Arena - Phase 5 Master Overview Technical Architecture, Platform Engineering & Infrastructure Blueprint Phase 5 Purpose Define how Skill Arena is engineered, deployed, scaled and operated at a technical level. Primary Objective Provide a complete technical blueprint for developers, architects and Codex to build the platform. Architecture Philosophy Microservice-driven, event-driven, cloud-native and server-authoritative. Document 1 System Architecture, Backend Services & Microservices Design. Document 2 API Architecture, Service Communication, Event Bus Design & Real-Time Messaging Framework. Document 3 Game Engine Architecture, Matchmaking Engine, Challenge Generation & Gameplay Processing Framework. Document 4 Database Architecture, Caching Strategy, Replay Storage & High Availability Framework. Document 5 Cloud Infrastructure, Deployment Architecture, CI/CD Pipeline, DevOps & Gl


### phase 5 tech spec and engineer\part 1

- `phase 5 tech spec and engineer\part 1\Phase_5_Document_1_System_Backend_Microservices_Architecture.pdf` (2 pages): Skill Arena - Phase 5 Document 1 System Architecture, Backend Services & Microservices Design Document Purpose Define the high-level technical architecture used to build and operate the Skill Arena platform. Business Objective Provide a scalable, maintainable and fault-tolerant foundation capable of supporting global growth. Architecture Philosophy Service-oriented architecture with clear separation of responsibilities and independently scalable components. Core Architecture Client Applications, API Gateway, Microservices Layer, Event Bus, Databases, Cache Layer and Infrastructure Services. Frontend Layer Web Application, Mobile Applications, Admin Portal and Internal Operations Portal. API Gateway Single secure entry point for clients, authentication, rate limiting and request routing. Microservices Strategy Independent services responsible for gameplay, wallets, tournaments, profiles,


### phase 5 tech spec and engineer\part 2

- `phase 5 tech spec and engineer\part 2\Phase_5_Document_2_API_EventBus_RealTime_Messaging_Architecture.pdf` (2 pages): Skill Arena - Phase 5 Document 2 API Architecture, Service Communication, Event Bus Design & Real-Time Messaging Framework Document Purpose Define how services communicate, how APIs are structured and how real-time events move throughout the platform. Business Objective Enable secure, scalable and reliable communication between users, services and platform components. API Architecture Central API Gateway with versioned APIs, authentication controls and service routing. API Design Principles Consistency, security, scalability, observability and backward compatibility. API Categories Public APIs, Authenticated Player APIs, Administrative APIs and Internal Service APIs. API Gateway Responsibilities Authentication, authorization, rate limiting, routing, logging and monitoring. REST API Layer Primary interface for standard platform operations and data retrieval. Real-Time Communication Layer


### phase 5 tech spec and engineer\part 3

- `phase 5 tech spec and engineer\part 3\Phase_5_Document_3_Game_Engine_Matchmaking_Challenge_Generation_Architecture.pdf` (2 pages): Skill Arena - Phase 5 Document 3 Game Engine Architecture, Matchmaking Engine, Challenge Generation & Gameplay Processing Framework Document Purpose Define the core gameplay architecture responsible for challenge generation, matchmaking, gameplay execution and competitive fairness. Business Objective Deliver infinitely scalable skill-based gameplay with fair matchmaking and server-authoritative processing. Game Engine Philosophy Every challenge must be unique, verifiable, scalable and resistant to exploitation. Core Components Game Engine, Matchmaking Engine, Challenge Generator, Difficulty Engine, Validation Engine and Replay Engine. Game Engine Responsibilities Challenge creation, state management, rule enforcement, progression tracking and outcome validation. Maze Challenge Generator Procedurally generates unique challenge layouts using controlled randomization. Challenge Uniqueness C


### phase 5 tech spec and engineer\part 4

- `phase 5 tech spec and engineer\part 4\Phase_5_Document_4_Database_Caching_Replay_Storage_High_Availability.pdf` (2 pages): Skill Arena - Phase 5 Document 4 Database Architecture, Caching Strategy, Data Storage, Replay Storage & High Availability Framework Document Purpose Define how data is stored, cached, replicated, protected and made highly available across the platform. Business Objective Provide reliable, scalable and fault-tolerant storage for gameplay, financial, replay and operational data. Architecture Philosophy Separate data by domain ownership, maximize resilience and ensure long-term auditability. Database Strategy Domain-driven databases assigned to User, Gameplay, Wallet, Treasury, Tournament, Replay and Security services. Primary Data Domains Identity Data, Gameplay Data, Financial Data, Tournament Data, Replay Data, Compliance Data and Security Data. Database Ownership Each microservice owns its own authoritative datastore. Read/Write Separation Support dedicated read replicas and optimized


### phase 5 tech spec and engineer\part 5

- `phase 5 tech spec and engineer\part 5\Phase_5_Document_5_Cloud_Deployment_CICD_DevOps_Global_Scaling.pdf` (2 pages): Skill Arena - Phase 5 Document 5 Cloud Infrastructure, Deployment Architecture, CI/CD Pipeline, DevOps & Global Scaling Framework Document Purpose Define how the platform is deployed, operated, scaled and maintained in production environments. Business Objective Provide a reliable, scalable and continuously deployable platform capable of global operation. Cloud Strategy Cloud-native architecture designed for elasticity, resilience and automation. Infrastructure Layers Edge Layer, Application Layer, Service Layer, Data Layer and Operations Layer. Deployment Architecture Containerized workloads deployed through orchestrated infrastructure. Environment Strategy Development, Testing, Staging, Pre-Production and Production environments. CI/CD Philosophy Automated build, test, validation and deployment pipelines. Source Control Strategy Version-controlled repositories with branch protections a


### phase 6 platform evolution

- `phase 6 platform evolution\Phase_6_Master_Overview_Platform_Evolution_10_Year_Blueprint.pdf` (2 pages): Skill Arena - Phase 6 Master Overview Platform Evolution, Multi-Game Framework, AI Systems & 10-Year Strategic Growth Blueprint Phase 6 Purpose Define the long-term evolution of Skill Arena beyond the initial launch platform. Primary Objective Create a framework that supports continuous expansion, innovation and sustainable growth over the next decade. Strategic Vision Transform Skill Arena from a single game into a global skill-based competitive ecosystem. Document 1 Multi-Game Framework & Game SDK Architecture. Document 2 AI Systems, Smart Matchmaking, Personalization Engine & Intelligent Platform Services. Document 3 Analytics Platform, Business Intelligence, Data Warehouse & Executive Insights Architecture. Document 4 Marketplace, Creator Ecosystem, Partner Integrations & Future Revenue Architecture. Document 5 Mobile Ecosystem, Global Expansion Strategy, Franchise Model & Long-Term


### phase 6 platform evolution\part 1

- `phase 6 platform evolution\part 1\Phase_6_Document_1_Multi_Game_Framework_Game_SDK_Architecture.pdf` (2 pages): Skill Arena - Phase 6 Document 1 Multi-Game Framework & Game SDK Architecture Document Purpose Define how future games integrate into Skill Arena without requiring major platform redesign. Business Objective Transform Skill Arena from a single-game platform into a scalable multi-game competitive ecosystem. Platform Vision Every future game should inherit wallets, tournaments, matchmaking, leaderboards, replays, security and progression systems automatically. Multi-Game Framework Central framework allowing multiple game types to coexist within the same platform architecture. Game Integration Philosophy Build once, integrate many. Core platform services reused across all games. Game SDK Overview Developer toolkit used to connect new games to platform services. SDK Responsibilities Authentication, Matchmaking, Wallet Integration, Replay Integration, Events and Analytics. Supported Game Type


### phase 6 platform evolution\part 2

- `phase 6 platform evolution\part 2\Phase_6_Document_2_AI_Systems_Personalization_Intelligent_Services.pdf` (2 pages): Skill Arena - Phase 6 Document 2 AI Systems, Smart Matchmaking, Personalization Engine & Intelligent Platform Services Architecture Document Purpose Define the future AI architecture powering personalization, intelligent matchmaking, recommendations and operational intelligence. Business Objective Increase engagement, retention, fairness and operational efficiency using intelligent platform services. AI Vision Use AI to improve player experience while preserving fairness, transparency and competitive integrity. Core AI Components Personalization Engine, Smart Matchmaking Engine, Recommendation Engine, AI Analytics Engine and AI Operations Layer. Personalization Engine Adapts dashboards, recommendations, events and progression opportunities to individual players. Player Profiles Build behavioural models using gameplay, progression, engagement and activity patterns. Recommendation Engine S


### phase 6 platform evolution\part 3

- `phase 6 platform evolution\part 3\Phase_6_Document_3_Analytics_Business_Intelligence_Data_Warehouse.pdf` (2 pages): Skill Arena - Phase 6 Document 3 Analytics Platform, Business Intelligence, Data Warehouse & Executive Insights Architecture Document Purpose Define the data analytics architecture used to measure platform performance, player behaviour, financial health and strategic growth. Business Objective Transform operational data into actionable intelligence for product, finance, marketing and executive teams. Analytics Vision Every major business decision should be supported by measurable platform intelligence. Core Components Analytics Platform, Data Warehouse, Business Intelligence Layer, Reporting Engine and Executive Insights Engine. Data Collection Framework Collect gameplay, financial, operational, tournament, security and engagement data. Event Collection Centralized event ingestion from all platform services. Data Warehouse Central repository optimized for reporting, forecasting and busin


### phase 6 platform evolution\part 4

- `phase 6 platform evolution\part 4\Phase_6_Document_4_Marketplace_Creator_Ecosystem_Revenue_Architecture.pdf` (2 pages): Skill Arena - Phase 6 Document 4 Marketplace, Creator Ecosystem, Partner Integrations & Future Revenue Architecture Document Purpose Define future platform monetization, creator tools, partner integrations and marketplace expansion opportunities. Business Objective Create diversified revenue streams while increasing player engagement and ecosystem growth. Platform Vision Evolve Skill Arena into a broader competitive gaming ecosystem with creators, partners and digital economies. Marketplace Overview Central marketplace for digital goods, cosmetics, seasonal content and future platform products. Marketplace Categories Cosmetics, Avatars, Themes, Profile Items, Seasonal Content, Founder Items and Future Collectibles. Creator Ecosystem Enable approved creators to contribute content, events and future experiences. Creator Profiles Dedicated creator identities with performance metrics and com


### phase 6 platform evolution\part 5

- `phase 6 platform evolution\part 5\Phase_6_Document_5_Mobile_Global_Expansion_Franchise_Roadmap.pdf` (2 pages): Skill Arena - Phase 6 Document 5 Mobile Ecosystem, Global Expansion Strategy, Franchise Model & Long-Term Product Roadmap Architecture Document Purpose Define the long-term growth strategy covering mobile expansion, global operations, franchise opportunities and the future roadmap. Business Objective Position Skill Arena as a globally recognized competitive skill-gaming ecosystem. Long-Term Vision Build a scalable platform capable of expanding into multiple regions, games and business models. Mobile Ecosystem Native mobile applications, tablet experiences and mobile-first engagement strategies. Mobile Features Gameplay, wallets, tournaments, notifications, replays, leaderboards and social features. Global Expansion Strategy Support regional growth through localization, compliance and infrastructure expansion. Localization Framework Languages, currencies, cultural adaptations and regional


### Phase 7 maze generation and anti bot

- `Phase 7 maze generation and anti bot\Phase_7_Master_Overview_Challenge_Intelligence_Blueprint.pdf` (2 pages): Skill Arena - Phase 7 Master Overview Challenge Intelligence, Competitive Integrity, Economic Sustainability & House Protection Blueprint Phase 7 Purpose Define the core intellectual property that powers challenge generation, competitive fairness, anti-bot intelligence, ranking systems and economic sustainability. Primary Objective Ensure Skill Arena remains fair, scalable, profitable and resistant to exploitation. Core Philosophy Skill determines outcomes. Systems verify fairness. Treasury remains protected. Competition remains sustainable. Document 1 Maze Generation, Difficulty Engineering, House Probability & Competitive Challenge Architecture. Document 2 Anti-Bot Intelligence, Behavioral Analysis, Human Verification, Trust Score & House Risk Engine Architecture. Document 3 AI Solver Framework, Human Performance Modeling, Difficulty Calibration & Challenge Balancing Architecture. Docu


### Phase 7 maze generation and anti bot\Part 1 The Maze

- `Phase 7 maze generation and anti bot\Part 1 The Maze\Phase_7_Document_1_Maze_Generation_Difficulty_House_Probability_Architecture.pdf` (3 pages): Skill Arena - Phase 7 Document 1 Maze Generation, Difficulty Engineering, House Probability & Competitive Challenge Architecture Document Purpose Define the core intellectual property responsible for maze generation, difficulty scaling, challenge fairness and house challenge balancing. Business Objective Create an infinitely scalable challenge engine that remains fair for players while maintaining long-term platform sustainability. Core Philosophy No two challenges should be predictably identical. Every maze must be unique, verifiable and server-generated. Server Authority All maze generation, validation, difficulty scoring and completion verification occur server-side. Maze Generation Formula Maze Seed + Difficulty Score + Category + Challenge Rules + Randomization Engine = Generated Challenge. Challenge Metadata Maze ID, Seed, Difficulty Score, Validation Hash, Category, Replay Referen

- `Phase 7 maze generation and anti bot\Part 1 The Maze\Phase_7_Document_1_Revision_House_Win_Rate_Update.pdf` (1 pages): Phase 7 Document 1 - Revision 1 House Win Rate & Adaptive Difficulty Amendment This amendment replaces the fixed house win rate concept with an adaptive probability model. Adaptive House Probability Model The platform shall not hard-code a fixed 65 percent house win rate. Instead, the system shall operate within a target range of approximately 60 to 70 percent house victories, with 65 percent used as the operational target. Actual performance will be monitored continuously. Dynamic Difficulty Adjustment The platform continuously evaluates completion rates, challenge success rates and treasury exposure. If player success rates become too high, challenge difficulty may be increased. If player success rates become too low, challenge difficulty may be reduced. House Fairness Principle House advantage must come from challenge design, timer pressure and difficulty engineering rather than impos


### Phase 7 maze generation and anti bot\Part 2 the anti bot

- `Phase 7 maze generation and anti bot\Part 2 the anti bot\Phase_7_Document_2_AntiBot_Intelligence_TrustScore_HouseRisk_Architecture.pdf` (3 pages): Skill Arena - Phase 7 Document 2 Anti-Bot Intelligence, Behavioral Analysis, Human Verification, Trust Score & House Risk Engine Architecture Document Purpose Define how Skill Arena detects bots, automation, macros, impossible performance and coordinated abuse while protecting competitive integrity. Business Objective Ensure every reward, victory and ranking is earned by genuine human skill. Security Philosophy Trust behaviour, not claims. Every player action is continuously analyzed. Core Components Anti-Bot Engine, Behavioral Analysis Engine, Human Verification Engine, Trust Score Engine and House Risk Engine. Anti-Bot Engine Continuously evaluates gameplay for signs of automation, scripting, macros and external assistance. Bot Detection Signals Reaction time consistency, movement precision, path efficiency, click intervals and completion patterns. Human Behavior Model Humans hesitate,


### Phase 7 maze generation and anti bot\Part 3 the ai solver framework

- `Phase 7 maze generation and anti bot\Part 3 the ai solver framework\Phase_7_Document_3_AI_Solver_Human_Performance_Difficulty_Calibration.pdf` (3 pages): Skill Arena - Phase 7 Document 3 AI Solver Framework, Human Performance Modeling, Difficulty Calibration & Challenge Balancing Architecture Document Purpose Define how Skill Arena predicts challenge difficulty, models human performance and balances house challenges over time. Business Objective Create a mathematically controlled challenge ecosystem that remains fair, scalable and sustainable. Core Philosophy Every challenge should be measurable, predictable and continuously calibrated using real performance data. Core Components AI Solver Engine, Human Performance Model, Difficulty Calibration Engine, Challenge Analytics Engine and House Balance Engine. AI Solver Engine Every generated maze is solved automatically before being released to players. Solver Responsibilities Calculate optimal route, shortest path, decision complexity, trap exposure and theoretical completion time. Solver Met


### Phase 7 maze generation and anti bot\Part 4 Replay AI for evaluation

- `Phase 7 maze generation and anti bot\Part 4 Replay AI for evaluation\Phase_7_Document_4_Replay_Intelligence_Match_Integrity_Architecture.pdf` (3 pages): Skill Arena - Phase 7 Document 4 Replay Intelligence, Match Integrity Analytics, Exploit Detection & Competitive Fairness Architecture Document Purpose Define how replay data is transformed into a competitive intelligence system for fairness, anti-cheat and platform protection. Business Objective Use replay information to continuously improve challenge integrity, detect exploits and protect platform sustainability. Core Philosophy Every match becomes a source of intelligence, not merely a dispute record. Replay Intelligence Engine Central platform responsible for analyzing historical and live replay data. Replay Data Sources PvP matches, House Challenges, Tournaments, Verification Challenges and Special Events. Replay Metadata Player IDs, Maze IDs, Completion Times, Route Paths, Decision Events and Validation Records. Route Analysis Engine Analyze how players move through challenges and


### Phase 7 maze generation and anti bot\Part 5 Competitive Integrity Master Engine

- `Phase 7 maze generation and anti bot\Part 5 Competitive Integrity Master Engine\Phase_7_Document_5_Competitive_Integrity_Reputation_League_Rankings.pdf` (3 pages): Skill Arena - Phase 7 Document 5 Competitive Integrity Master Engine, Reputation System, League Progression & Global Ranking Architecture Document Purpose Define the systems that determine player reputation, rankings, league placement, progression and competitive standing. Business Objective Create a fair and sustainable competitive ecosystem where skill is accurately measured and rewarded. Core Philosophy Rankings should reflect demonstrated skill, consistency, integrity and long-term performance. Competitive Integrity Master Engine Central authority responsible for evaluating player performance and competitive standing. Player Reputation System Every player receives a dynamic reputation score based on conduct, trust and competitive integrity. Reputation Factors Trust Score, Fair Play History, Verification Results, Tournament Conduct and Community Standing. Reputation Impact Influences


### Phase 7 maze generation and anti bot\Part 6 Challenge Economy Risk Management Reward Balancing

- `Phase 7 maze generation and anti bot\Part 6 Challenge Economy Risk Management Reward Balancing\Phase_7_Document_6_Challenge_Economy_Risk_Reward_Sustainability_Architecture.pdf` (3 pages): Skill Arena - Phase 7 Document 6 Challenge Economy, Risk Management, Reward Balancing & Sustainability Architecture Document Purpose Define the economic rules that connect gameplay, rewards, treasury protection and long-term platform sustainability. Business Objective Ensure Skill Arena remains financially sustainable while providing attractive rewards and competitive experiences. Core Philosophy Every reward, payout and incentive must be supported by treasury reserves and sustainable economic models. Challenge Economy Engine Central system responsible for balancing entry fees, rewards, treasury exposure and profitability. Token Economy Integration All gameplay activities integrate with the platform token and wallet ecosystem. PvP Entry Structure Players enter matches using tokens. Platform fees are collected before prize pools are formed. PvP Platform Fee Model Initial design uses a 10


### Phase 8 Game Flow

- `Phase 8 Game Flow\Phase_8_Document_8_Maze_Game_Master_Specification_Implementation_Blueprint.pdf` (2 pages): Skill Arena - Phase 8 Document 8 Complete Maze Game Master Specification & Implementation Blueprint Document Purpose Provide the complete implementation blueprint for Version 1 of the Skill Arena Maze Game. Business Objective Deliver a fully defined product specification that developers can use to build the first production-ready game. Core Philosophy Skill determines outcomes. Every challenge is unique. Every reward is auditable. Every result is verifiable. Player Journey Registration, verification, wallet activation, practice mode, competitive play, progression and rewards. Gameplay Foundation Procedurally generated mazes powered by the Maze Intelligence Engine. Difficulty Framework Infinite difficulty scaling using difficulty scores rather than fixed levels. Daily Calibration System Unique daily calibration challenges used for player modelling and platform intelligence. PvP System Ran


### Phase 8 Game Flow\Part 1 Game play

- `Phase 8 Game Flow\Part 1 Game play\Phase_8_Document_1_Complete_Gameplay_Flow_Architecture.pdf` (3 pages): Skill Arena - Phase 8 Document 1 Complete Gameplay Flow Architecture & Player Journey Specification Document Purpose Define the complete player journey from account registration through competitive progression and long-term engagement. Business Objective Provide developers with a complete blueprint for how players interact with the platform. Player Lifecycle Visitor → Registered User → Verified Player → Competitive Player → League Competitor → Elite Competitor. Step 1 - Landing Page User visits platform and views game information, rankings, rewards and platform benefits. Step 2 - Registration User creates account using email verification and platform onboarding flow. Step 3 - Identity Verification Optional verification becomes progressively required for higher-value rewards and withdrawals. Step 4 - Wallet Activation User receives wallet profile and token account infrastructure. Step 5 -

- `Phase 8 Game Flow\Part 1 Game play\Phase_8_Document_1_Revision_Daily_Skill_Calibration.pdf` (2 pages): Phase 8 Document 1 - Revision 1 Daily Skill Calibration & Player Baseline System Amendment Purpose Introduce a daily skill calibration system that continuously measures player capability and improves platform intelligence. Daily Calibration Maze Every player receives one daily calibration challenge generated specifically for measurement purposes. No Financial Impact Calibration challenges have no entry fees, rewards, losses or ranking consequences. No Competitive Impact Results do not affect league rankings, seasonal standings or reputation scores. AI Calibration Purpose Results are used to improve AI Solver accuracy, human performance models and difficulty calibration. Behavioral Baseline The platform builds a long-term behavioural profile for each player. Account Sharing Detection Major changes in gameplay style, reaction patterns or completion ability trigger investigation signals. Bo


### Phase 8 Game Flow\Part 2 Maze Mechanics

- `Phase 8 Game Flow\Part 2 Maze Mechanics\Phase_8_Document_2_Maze_Mechanics_Gameplay_Rules_Specification.pdf` (3 pages): Skill Arena - Phase 8 Document 2 Complete Maze Mechanics, Controls, Lives, Traps, Vision System, Checkpoints & Gameplay Rules Specification Document Purpose Define exactly how the Maze Game operates from the player's perspective. Business Objective Create a skill-based challenge that is easy to understand but difficult to master. Core Philosophy Player skill, decision making and consistency determine success. Game Objective Navigate from the maze entrance to the maze exit while overcoming obstacles and challenge mechanics. Movement Controls Keyboard, touch and future controller support. Movement validation remains server-authoritative. Movement Rules Players may move only through valid maze paths. Illegal movement is rejected server-side. Maze Structure Every maze contains a start point, end point, valid routes, dead ends and challenge elements. Procedural Generation Every challenge is g


### Phase 8 Game Flow\Part 3 PvP Match Specification

- `Phase 8 Game Flow\Part 3 PvP Match Specification\Phase_8_Document_3_PvP_Match_Specification_Matchmaking_Prize_Distribution.pdf` (3 pages): Skill Arena - Phase 8 Document 3 PvP Match Specification, Match Lifecycle, Matchmaking Rules, Prize Distribution & Replay Architecture Document Purpose Define the complete PvP challenge experience from challenge creation to reward distribution. Business Objective Provide a fair, competitive and scalable player-versus-player ecosystem. Core Philosophy Both players compete on equal challenge conditions. Skill determines the outcome. PvP Match Types Ranked PvP, Casual PvP, Private Challenges and Tournament Matches. Ranked PvP Impacts rankings, reputation, leagues, XP and seasonal progression. Casual PvP No ranking impact. Designed for practice and social competition. Private Challenges Players may invite specific opponents and optionally configure stake values. Tournament Matches Controlled by tournament systems and championship qualification rules. Match Creation Player selects challenge t


### Phase 8 Game Flow\Part 4 House challenge specification

- `Phase 8 Game Flow\Part 4 House challenge specification\Phase_8_Document_4_House_Challenge_Tiers_Payouts_Lifecycle.pdf` (3 pages): Skill Arena - Phase 8 Document 4 House Challenge Specification, House Tiers, Eligibility Rules, Payout Logic & Challenge Lifecycle Architecture Document Purpose Define the complete house challenge system including eligibility, progression, risk controls and reward structures. Business Objective Provide a sustainable challenge environment where players compete directly against the platform. Core Philosophy House challenges must be difficult, fair, verifiable and economically sustainable. House Challenge Types Bronze House, Silver House, Gold House, Platinum House, Elite House and Legend House. Bronze House Tier Entry-level house challenges designed for new competitive players. Silver House Tier Intermediate challenge category with increased difficulty and rewards. Gold House Tier Advanced challenge category requiring demonstrated player skill. Platinum House Tier High-difficulty challenge


### Phase 8 Game Flow\Part 5 UI UX screen

- `Phase 8 Game Flow\Part 5 UI UX screen\Phase_8_Document_5_UI_UX_Screen_Blueprint_Player_Dashboard.pdf` (2 pages): Skill Arena - Phase 8 Document 5 UI/UX Screen Blueprint, Navigation Architecture, Player Dashboard & Gameplay Interface Specification Document Purpose Define every major screen, navigation flow and user interaction within the Skill Arena platform. Business Objective Provide a simple, professional and scalable player experience across all devices. Core Design Philosophy Easy to learn, fast to navigate, competitive by design and mobile-first. Main Navigation Home, Play, House Challenges, Tournaments, Rankings, Wallet, Marketplace, Replays and Profile. Home Dashboard Player overview including rank, league, balance, XP, seasonal progress and quick actions. Quick Actions Play PvP, Challenge House, Deposit, Withdraw, View Rankings and Watch Replays. Player Profile Screen Avatar, statistics, trust score, reputation, achievements and progression history. Wallet Screen Token balances, deposits, w


### Phase 8 Game Flow\Part 6 Mobile app Specification

- `Phase 8 Game Flow\Part 6 Mobile app Specification\Phase_8_Document_6_Mobile_App_Offline_Replay_Cross_Platform_Architecture.pdf` (2 pages): Skill Arena - Phase 8 Document 6 Mobile Application Specification, Notifications, Offline Replay System & Cross-Platform Experience Architecture Document Purpose Define the mobile experience, notification systems and cross-platform behaviour of Skill Arena. Business Objective Deliver a world-class mobile experience that matches the quality and functionality of the web platform. Core Philosophy Play anywhere, compete anywhere, review anywhere. Supported Platforms iOS, Android, Web and future tablet devices. Mobile First Design Interfaces optimized for touch interaction before desktop adaptation. Authentication Experience Secure login, biometric support and multi-factor authentication integration. Mobile Dashboard Quick access to rankings, wallet, challenges, tournaments and profile information. Push Notification Framework Challenge invites, tournament reminders, rewards, security alerts a


### Phase 8 Game Flow\Part 7 admin platform

- `Phase 8 Game Flow\Part 7 admin platform\Phase_8_Document_7_Admin_Moderation_Investigation_Operations_Platform.pdf` (3 pages): Skill Arena - Phase 8 Document 7 Administration Platform, Moderation Console, Investigation Center & Operational Control Specification Document Purpose Define the complete administration and moderation environment used to manage, monitor and protect the platform. Business Objective Provide operational teams with the tools required to maintain fairness, security and platform stability. Core Philosophy Every action must be traceable, auditable and role-controlled. Administration Platform Centralized control center for platform operations, player management and system monitoring. Role-Based Access Control Permissions assigned according to operational responsibility and least-privilege principles. Admin Dashboard High-level view of platform health, treasury status, player activity and risk indicators. Moderation Console Dedicated workspace for reviewing player reports, disputes and enforceme


### Phase 9 UI

- `Phase 9 UI\Phase_9_Document_1_Brand_Identity_and_Design_System.pdf` (2 pages): Phase 9 - Document 1 Brand Identity & Design System Platform Philosophy Skill Arena is a Competitive Human Skill Platform inspired by FACEIT, Chess.com and modern SaaS design. Logo System Shield + SA Monogram + Neural Path Connections. Color System Arena Blue #00D4FF, Arena Purple #7C3AED, Success #22C55E, Warning #F59E0B, Danger #EF4444. Theme System Dark Theme and Light Theme supported from day one. Typography Inter and Space Grotesk. League Colors Bronze, Silver, Gold, Platinum, Diamond, Elite and Legend. Navigation Philosophy FACEIT-inspired left navigation with future expansion support. Localization English, Afrikaans, French, German, Spanish, Portuguese, Italian, Arabic, Chinese, Japanese and Korean. Mobile First Desktop, tablet and mobile support required. Component Standards Reusable, localized, responsive, accessible and theme-aware. Accessibility WCAG AA compliance target. Anim

- `Phase 9 UI\Phase_9_Document_2_Navigation_Architecture_and_Information_Architecture.pdf` (2 pages): Phase 9 - Document 2 Navigation Architecture & Information Architecture Purpose Define the complete structure, navigation and user journeys of the Skill Arena platform. Public Navigation Home, Games, Leaderboards, Tournaments, About, Support, Login and Register. Authenticated Navigation Dashboard, Games, Challenges, Tournaments, Leaderboards, Wallet, Replays, Profile and Settings. Dashboard Structure Player overview, featured games, season progress, challenges, rankings and quick actions. Games Hub Maze Arena, Memory Arena, Logic Arena, Pattern Arena and future games without redesign. Challenges Architecture Daily Calibration, Daily Challenges, Weekly Challenges, Boss Challenges and Seasonal Challenges. PvP Structure Ranked, Casual, Private Challenges and Tournament Play. House Challenges Bronze, Silver, Gold, Platinum, Elite and Legend challenge tiers. Tournament Architecture Open, Seas

- `Phase 9 UI\Phase_9_Document_3_Dashboard_UX_and_Game_Hub_Experience.pdf` (2 pages): Phase 9 - Document 3 Dashboard UX and Game Hub Experience Purpose Define the primary user dashboard and overall game hub experience. Dashboard Philosophy Users must immediately understand where they are, what to do next and how they are progressing. Welcome Banner Displays username, league, rank and season information. Player Overview Cards Wallet Balance, Trust Score, League Points and Current Streak. Continue Playing Automatically presents the user's last active game and fastest route back into gameplay. Featured Games Highlights Maze Arena and future games with quick launch actions. Daily Challenges Daily Calibration, Daily Challenges and Weekly Challenges. Season Center Current season status, rewards preview and progression tracking. Leaderboard Snapshot Top ranked players and quick access to full rankings. Event Banner Boss events, seasonal events and special announcements. News Cen

- `Phase 9 UI\Phase_9_Document_4_Maze_Arena_UX_Gameplay_and_Replay_Experience.pdf` (2 pages): Phase 9 - Document 4 Maze Arena UX, Gameplay and Replay Experience Purpose Define the complete Maze Arena player experience including gameplay, PvP, house challenges and replays. Maze Arena Structure Maze Arena contains Home, Casual, Ranked, House Challenges, Boss Events, Replays, Statistics and Leaderboards. Maze Arena Home Displays current league, rank, statistics, recent matches and quick access to all gameplay modes. Statistics Matches Played, Win Rate, Current Streak, Best Time, Average Time, League Progress and Trust Score. Ranked Mode Primary competitive mode with league progression and rewards. Casual Mode Practice mode with no rank impact. Dual Maze PvP System Players see their own maze fully visible while the opponent maze is blurred or pixelated. Opponent Visibility Only approximate progress, position and estimated moves remaining are shown. Exact route and decisions remain hi

- `Phase 9 UI\Phase_9_Document_5_Wallet_Treasury_and_Reward_Experience.pdf` (2 pages): Phase 9 - Document 5 Wallet, Treasury and Reward Experience Purpose Define the complete financial user experience including balances, rewards, deposits and withdrawals. Core Philosophy The wallet must feel transparent, trustworthy and professional rather than gambling-focused. Wallet Overview Displays available balance, pending rewards, pending withdrawals and lifetime earnings. Wallet Navigation Overview, Deposit, Withdraw, Rewards, Transactions, Treasury Status and Security. Transaction History Every transaction includes date, amount, type, reference and status. Deposit Experience Fast, secure deposits with country-aware payment methods. Deposit Methods Support EFT, Instant EFT, Cards, PayFast, Ozow and future payment providers. Withdrawal Experience Simple withdrawal workflow with verification, review and processing stages. Withdrawal Statuses Pending, Processing, Approved, Completed

- `Phase 9 UI\Phase_9_Document_6_PvP_Tournaments_Clans_and_Competitive_Ecosystem.pdf` (2 pages): Phase 9 - Document 6 PvP, Tournaments, Clans and Competitive Ecosystem Purpose Transform Skill Arena from a game into a long-term competitive ecosystem. Competitive Philosophy Players should feel they belong to a community, league and progression system. PvP Hub Ranked, Casual, Private Challenges, Rivals, Match History and Statistics. Ranked Play Primary competitive mode with leagues, rankings, rewards and progression. Casual Play Practice mode without rank impact. Private Challenges Direct challenges between players with configurable stakes and settings. Rivals System Automatically suggests players of similar rank to create long-term competition. Rival Dashboard Displays wins, losses, head-to-head history and current rivalry streaks. Friends System Online friends, invitations, recent activity and social interaction. Clan System Clan leadership, members, rankings, achievements and progre

- `Phase 9 UI\Phase_9_Document_7_Admin_UX_Fraud_Treasury_Replay_Operations.pdf` (2 pages): Phase 9 - Document 7 Admin UX, Fraud Center, Treasury Center, Replay Theater and Operations Dashboard Purpose Define the enterprise administration platform used to operate Skill Arena. Admin Philosophy The admin platform should feel like enterprise banking and operations software rather than a game panel. Admin Navigation Dashboard, Users, Fraud Center, Treasury Center, Replay Theater, Support, Reports and System Health. Operations Dashboard Provides a real-time overview of users, challenges, treasury health, fraud alerts and withdrawals. Executive Summary Active Players, Challenge Volume, Treasury Coverage, Fraud Alerts and Pending Withdrawals. Operations Feed Real-time events such as withdrawals, fraud alerts, tournaments and system notifications. User Management Player profiles, trust scores, balances, verification status, devices and activity history. Fraud Center Suspicious accounts

- `Phase 9 UI\Phase_9_Document_8_Frontend_Implementation_Handbook.pdf` (2 pages): Phase 9 - Document 8 Frontend Implementation Handbook and Codex Development Blueprint Purpose Provide the complete frontend implementation blueprint for Codex and development teams. Technology Stack Next.js, React, TypeScript, TailwindCSS, Zustand, TanStack Query, i18next and Framer Motion. Folder Structure Components, Layouts, Modules, Services, Hooks, Store, Themes, Locales, Utilities and Assets. Design Tokens Arena Blue, Arena Purple, Success, Warning and Danger token system. Typography Inter as primary font and Space Grotesk as secondary font. Theme System Dark and Light theme support from day one. Localization All text must use translation keys and support multi-language architecture. Component Philosophy Reusable, localized, responsive, accessible and theme-aware components. Layouts App Layout, Authentication Layout and Admin Layout. Global Components Buttons, Cards, Tables, Modals

- `Phase 9 UI\Phase_9_Master_Overview_Skill_Arena_UX_Handbook.pdf` (2 pages): Phase 9 - Master Overview Skill Arena UX, Design System & Frontend Experience Handbook Purpose Consolidated reference handbook covering all approved Phase 9 frontend, UX and design decisions. Platform Philosophy Competitive Human Skill Platform inspired by FACEIT, Chess.com and modern SaaS experiences. Brand Identity Shield + SA Monogram + Neural Path logo system with Arena Blue and Arena Purple color scheme. Design System Dark and Light themes, Inter typography, Space Grotesk headings and reusable component architecture. Localization Multi-language support from day one using translation-driven architecture. Navigation Architecture Public, Authenticated and Admin navigation structures designed for infinite expansion. Dashboard Experience Player progression, challenges, featured games, rankings and season tracking. Maze Arena Experience Ranked, Casual, House Challenges, Boss Events, Repla

---

## Phase 1 And 3 Requirements

<!-- Archived source: docs/backup/individual-markdown-2026-07-15/PHASE_1_AND_3_REQUIREMENTS.md -->

### Executive Summary

This document extracts key requirements from Phase 1 and Phase 3 PDFs, organized by component with core features/APIs, database models, user workflows, and business logic for each system.

---

### PHASE 1: FOUNDATION & GAME DESIGN (Launch: Maze Arena)

### Core Constitutional Principles (Non-Negotiable)
1. **Skill Determines Outcomes** - Pure skill-based competition
2. **Every Live Match Is Replayable** - Full replay verification capability
3. **Every Token Is Auditable** - Complete financial traceability
4. **Every Challenge Is Verifiable** - All results can be validated
5. **Server Authority** - Client trust level = ZERO for business logic
6. **No Reward May Exceed Treasury Reserves** - Financial sustainability
7. **Infinite Progression** - No caps on advancement
8. **Fair Seasonal Competition** - Resets and seasonal mechanics
9. **Sustainability First** - Long-term viability over short-term profit

---

### PHASE 1: COMPONENT SPECIFICATIONS

### 1. USER SYSTEM & AUTHENTICATION

#### Core APIs Needed:
```
POST /auth/register         → Register new user
POST /auth/verify-email     → Verify email
POST /auth/login            → Authenticate user
POST /auth/mfa-setup        → Enable MFA
POST /auth/mfa-verify       → Verify MFA token
POST /auth/refresh-token    → Get new access token
POST /identity/kyc-submit   → Submit KYC verification
GET  /identity/kyc-status   → Check KYC status
POST /devices/fingerprint   → Register device
```

#### Database Models:
| Table | Purpose | Key Fields |
|-------|---------|-----------|
| `users` | User identity | user_id, email, password_hash, kyc_status, verified_date, created_at |
| `user_profiles` | User info | user_id, avatar, country, username, bio |
| `user_sessions` | Active sessions | user_id, session_token, refresh_token, expires_at |
| `user_devices` | Device tracking | user_id, device_fingerprint, device_name, os, browser, last_seen |
| `kyc_records` | Identity verification | user_id, verification_provider, status, document_type, verified_date |

#### User Workflows:
```
1. Registration Flow:
   - Email registration → Email verification → Profile setup →
   - KYC submission → Admin verification → Account activated

2. Login Flow:
   - Email + password → JWT issued → MFA challenge (if enabled) →
   - Session created → Ready for API calls

3. Account Escalation:
   - Basic account → Verified account (high-value withdrawals) →
   - KYC approved (withdraw limits lifted)

4. Security Workflow:
   - Enable MFA → Download recovery codes → Confirm MFA works
```

#### Business Logic:
- Password: Min 12 chars, uppercase, numbers, symbols required
- Email verification required before wallet activation
- KYC mandatory for withdrawals > USD $500
- MFA enforced for withdrawals
- Device fingerprinting prevents account sharing
- Session timeout: 30 days (refresh tokens)

---

### 2. WALLET & TOKEN SYSTEM

#### Core APIs Needed:
```
GET  /wallet/balance          → Get current balance
POST /wallet/deposit          → Initiate deposit
POST /wallet/withdrawal       → Initiate withdrawal
POST /wallet/lock-tokens      → Reserve tokens for match
POST /wallet/unlock-tokens    → Release reserved tokens
GET  /wallet/transactions     → Transaction history
GET  /wallet/available        → Available balance (not locked)
```

#### Database Models:
| Table | Purpose | Key Fields |
|-------|---------|-----------|
| `wallets` | User wallets | wallet_id, user_id, wallet_type, balance, status, created_at |
| `wallet_types` | Wallet categories | LIVE_WALLET, DEMO_WALLET, LOCKED_WALLET, BONUS_WALLET, SYSTEM_WALLET |
| `transactions` | All movements | transaction_id, wallet_id, amount, type, reference_id, status, timestamp |
| `ledger_entries` | Double-entry ledger | entry_id, transaction_id, debit_account, credit_account, amount, balance_after |
| `wallet_audit` | Immutable audit log | audit_id, wallet_id, previous_balance, new_balance, reason, timestamp |

#### User Workflows:
```
Deposit Workflow:
1. Player clicks "Deposit"
2. Select payment method and amount (min USD $10)
3. Redirected to payment provider
4. Payment provider confirms transaction
5. System receives webhook confirmation
6. Ledger entry created (debit: Bank, credit: Live Wallet)
7. Player receives tokens in Live Wallet
8. Available immediately for play

Withdrawal Workflow:
1. Player initiates withdrawal
2. System verifies: account verified, wallet balance, limits
3. AML screening performed
4. Admin manual review (if high value)
5. Withdrawal approved
6. Ledger entry created (debit: Live Wallet, credit: Bank Account)
7. Bank transfer initiated
8. Player receives funds in 1-3 business days

Match Entry Workflow:
1. Player enters match (10 tokens)
2. System locks tokens (debit: Live Wallet, credit: Locked Wallet)
3. Match occurs
4. Match completes - tokens unlocked
5. Ledger entries: Prize credited or fee taken
```

#### Business Logic:
- **Wallet Types:**
  - `LIVE_WALLET`: Real money, can withdraw
  - `DEMO_WALLET`: Practice tokens, no value, can't transfer
  - `LOCKED_WALLET`: Reserved for active matches
  - `BONUS_WALLET`: Promotional tokens, withdrawal restrictions
  - `SYSTEM_WALLET`: Treasury system account

- **Balance Calculation:**
  ```
  Available Balance = Live Wallet - Locked Wallet - Pending Withdrawals
  ```

- **Minimum Deposit:** USD $10 (100 tokens at 1:10 rate)
- **Withdrawal Limits:**
  - Unverified: None (blocked unless KYC passes)
  - Verified: USD $50,000/day, USD $500,000/month
  - Enhanced: No limit

- **Fees:**
  - Deposits: 2.5% (absorbed by platform)
  - Withdrawals: 1% (charged to player)
  - Match entry: Included in PvP fee

---

### 3. PROGRESSION SYSTEMS (5 Independent)

#### Core APIs Needed:
```
GET  /progression/xp         → Get XP level and prestige
GET  /progression/elo        → Get skill rating
GET  /progression/league     → Get league rank
GET  /progression/house      → Get house reputation
GET  /progression/legacy     → Get legacy points
POST /progression/award-xp   → Award XP (internal only)
```

#### Database Models:
| Table | Purpose | Key Fields |
|-------|---------|-----------|
| `progression_xp` | Level system | user_id, current_level, total_xp_earned, prestige_level, never_reset |
| `progression_elo` | Skill rating | user_id, current_rating, matches_played, wins, losses, k_factor |
| `progression_league` | League rank | user_id, league_tier, rank_in_league, promotion_points, season_id |
| `progression_house_reputation` | House challenges | user_id, reputation_score, challenges_completed, win_rate |
| `progression_legacy` | Lifetime tracker | user_id, total_legacy_points, contribution_tier, never_reset |
| `prestige_milestones` | Prestige unlocks | prestige_level, xp_required, rewards_given |

#### Progression Types:

##### 1. XP LEVEL (Infinite, Never Resets)
- **Unlock house tiers:** XP determines available house challenge tiers
- **XP Sources:**
  - Complete PvP match: +10 XP
  - PvP victory: +50 XP
  - House challenge success: +25 XP per tier
  - Tournament participation: +100 XP
  - Seasonal achievements: +500 XP
- **No cap** - progression continues indefinitely
- **Prestige System:**
  - Unlock after XP milestones (e.g., 100k XP = Prestige I)
  - No upper limit on Prestige
  - Permanent badge on profile
  - Cosmetic rewards at each level

##### 2. SKILL RATING (ELO-Based)
- **Match-based rating** reflecting competitive skill
- **Initial rating:** 1200 (for all new players)
- **Formula:** `New_Rating = Old_Rating + K_Factor × (Actual_Result - Expected_Result)`
  - K_Factor = 32 (standard competitive)
  - Actual_Result = 1 (win) or 0 (loss)
  - Expected_Result = calculated from opponent rating
- **Rating floors:** Minimum 1000 (no negative ratings)
- **Used for:** Matchmaking, league placement, tournament eligibility

##### 3. LEAGUE RANK (Seasonal, Resets)
- **League Tiers:** Bronze, Silver, Gold, Platinum, Diamond, Elite, Legend
- **Rank within tier:** 1-100
- **Promotion/Demotion:** Based on season points
- **Resets:** January 1st each season (mid-season optional)
- **Permanent progression:** Achievement for reaching each tier recorded in stats

##### 4. HOUSE REPUTATION (Separate System)
- **Earned from:** House challenge completions only
- **Score increases:** When winning house challenges
- **Unlocks:** Higher house tiers (Bronze → Silver → Gold, etc.)
- **Never impacts:** PvP rankings or seasonal standings
- **Win rate tracking:** Monitor individual player success vs house

##### 5. LEGACY POINTS (Lifetime, Never Resets)
- **Represents:** Lifetime contribution to platform
- **Sources:**
  - Seasonal participation: +10 points
  - Tournament success: +50 per placement
  - House challenge completion: +5 per tier
  - PvP activity: +1 per match
  - Seasonal achievements: +100 per achievement
- **Purpose:** Long-term prestige marker
- **Hall of Fame:** Top 1000 legacy point holders get recognition

#### Business Logic:
```
XP_Level = floor(Total_XP_Earned / 1000)
Prestige_Level = floor(XP_Level / 100)
Available_House_Tiers = tiers where min_xp <= player_xp_level
Matchmaking_Rating = current_elo_rating
```

---

### 4. MATCH & GAMEPLAY SYSTEM

#### Core APIs Needed:
```
POST /matches/create          → Create new match
GET  /matches/{match_id}      → Get match state
POST /matches/{match_id}/move → Submit player move
POST /matches/{match_id}/complete → Mark challenge complete
GET  /matches/{match_id}/replay  → Get replay data
GET  /matches/history         → Player match history
```

#### Database Models:
| Table | Purpose | Key Fields |
|-------|---------|-----------|
| `matches` | Match metadata | match_id, match_type, player_1_id, player_2_id, maze_id, status, created_at, ended_at |
| `match_participants` | Player match data | match_id, user_id, entry_fee, final_result, completion_time, verified |
| `match_pots` | Financial tracking | match_id, total_pot, platform_fee, house_edge, winner_prize, loser_prize |
| `challenge_state` | Live game state | match_id, current_state_json, moves_validated, lives_remaining, completion_percent, last_update |
| `match_movements` | Move audit trail | movement_id, match_id, user_id, move_type, move_data, server_validated, timestamp |
| `replay_data` | Full replay recording | replay_id, match_id, compressed_movements, verification_hash, created_at |

#### Match Types:

##### 1. RANKED QUEUE
- **Impact:** Affects ELO, league rank, seasonal standing
- **Matchmaking:** ELO ± 200 rating points
- **Entry fee:** 10 tokens (configurable)
- **Rewards:** Based on win/loss and ELO difference

##### 2. CASUAL QUEUE
- **Impact:** None (no ranking changes)
- **Entry fee:** 5 tokens (configurable)
- **Purpose:** Practice without penalty
- **Rewards:** Reduced, flat-rate

##### 3. FRIEND CHALLENGE
- **Opponent:** Specific player invite
- **Entry fee:** Configurable (both must agree)
- **Impact:** Custom (can be ranked or casual)
- **Privacy:** Match not shown on leaderboards

##### 4. CROSS-LEAGUE CHALLENGE
- **Opponent:** Different league tier
- **Adjustments:** ELO adjustment factors applied
- **Entry fee:** Variable based on league difference
- **Purpose:** Competitive fun without matchmaking constraints

##### 5. TOURNAMENT MATCH
- **Controlled by:** Tournament system
- **Entry fee:** Paid at tournament entry
- **Impact:** Tournament standing only
- **Rewards:** Tournament prize pool

#### Match Lifecycle:

```
1. CREATION PHASE
   - Player selects queue type and entry amount
   - Server validates:
     * Account status (not banned)
     * Wallet balance ≥ entry fee
     * League eligibility (not above max or below min)
     * KYC status (verified if high value)
     * Device verified (not flagged)
   - If valid: Match created, entry fee LOCKED
   - Matchmaking begins

2. MATCHMAKING PHASE
   - Server finds opponent with similar rating
   - Timeout: 30 seconds (if no match, auto-refund)
   - Once paired: Both players notified
   - Both have 10 seconds to accept/decline

3. CHALLENGE GENERATION PHASE
   - Server generates shared maze seed
   - Both players receive SAME maze
   - Seed stored in match record (immutable)
   - Verification hash created

4. MATCH START PHASE
   - Both players see identical starting maze
   - Timer synchronized across both clients
   - Lives counter initialized
   - Match status: IN_PROGRESS

5. GAMEPLAY PHASE
   - Every move sent to server
   - Server validates move:
     * Is move in valid path?
     * Is player alive?
     * Did movement comply with maze rules?
     * If invalid: Move rejected, client notified
   - Client renders authorized moves only
   - Progress tracked server-side

6. COMPLETION PHASE
   - First player to reach exit wins
   - OR timer expires (current leader wins)
   - OR both players fail (draw or lowest-loss)
   - Server marks match: COMPLETED

7. RESULTS PHASE
   - Server calculates results:
     * Winner determined
     * Match duration calculated
     * ELO changes calculated
     * Prize pool distributed
     * Ledger entries created
   - Players notified immediately
   - Replay data saved and verified
```

#### Match Entry Validation (Server-Side Only):
```python
def validate_entry(player_id, match_type, entry_fee):
    # All checks MUST happen server-side
    account = get_account(player_id)

    # 1. Account Status
    if account.banned or account.suspended:
        return False, "Account restricted"

    # 2. Wallet Balance
    available = get_available_balance(player_id)
    if available < entry_fee:
        return False, "Insufficient balance"

    # 3. League Eligibility
    player_league = get_league_tier(player_id)
    if match_type == "ranked" and not is_eligible_for_ranked(player_league):
        return False, "League restrictions"

    # 4. KYC Status
    if entry_fee > 100 and not account.kyc_verified:
        return False, "KYC required"

    # 5. Tournament Qualification
    if match_type == "tournament":
        if not is_tournament_qualified(player_id):
            return False, "Tournament eligibility not met"

    return True, "Validated"
```

#### Prize Pool Calculation:

```
Example:
  Player A entry = 10 tokens
  Player B entry = 10 tokens

  Total Pot = 20 tokens
  Platform Fee = 10% × 20 = 2 tokens (goes to Platform Revenue Reserve)
  Actual Prize Pool = 20 - 2 = 18 tokens

  If Player A wins:
    Player A receives: 18 tokens
    Player B receives: 0 tokens

  If draw/mutual loss (both fail):
    Split 18 tokens: 9 each
```

#### Business Logic:
- **Platform Fee:** 10% of entry pot
- **ELO Adjustment:** K=32, consider rating difference
- **Replay Storage:** All replays stored for 90 days minimum
- **Dispute Period:** 24 hours for players to dispute

---

### 5. HOUSE CHALLENGE ENGINE

#### Core APIs Needed:
```
GET  /house/tiers            → Get available house tiers
POST /house/challenge        → Generate new house challenge
POST /house/submit           → Submit challenge completion
GET  /house/history          → Get challenge history
GET  /house/analytics        → Get personal vs house stats
```

#### Database Models:
| Table | Purpose | Key Fields |
|-------|---------|-----------|
| `house_tiers` | Tier definitions | tier_id, tier_name, min_xp_level, min_elo, cost_per_attempt, reward_multiplier, win_rate_target |
| `house_challenges` | Challenge instances | challenge_id, player_id, tier_id, maze_id, difficulty_score, seed, created_at, expires_at |
| `house_results` | Outcome tracking | result_id, challenge_id, player_id, status (win/loss), completion_time, payout, treasury_impact |
| `house_analytics` | Player statistics | user_id, challenges_completed, win_rate, total_payout, avg_completion_time |

#### House Tiers (Progressive):

| Tier | Min XP | Min ELO | Entry Fee | Reward | House WR Target |
|------|--------|---------|-----------|--------|-----------------|
| Bronze | 0 | 1200 | 5 | 7.5 | 65% |
| Silver | 10k | 1400 | 15 | 22.5 | 65% |
| Gold | 30k | 1600 | 50 | 75 | 65% |
| Platinum | 75k | 1800 | 150 | 225 | 65% |
| Elite | 150k | 2000 | 500 | 750 | 65% |
| Legend | 300k | 2200 | 1500 | 2250 | 65% |

#### House Challenge Generation:

```
Challenge_Seed = Hash(
    player_id,
    timestamp,
    difficulty_tier,
    server_secret_salt,
    randomization_nonce
)

Challenge_ID = SHA256(Challenge_Seed)

Uniqueness_Check:
    IF Challenge_ID exists in database:
        Regenerate with new nonce
    ELSE:
        Store and use
```

#### House Challenge Unlock Requirements:

```
def can_access_tier(player_id, tier):
    player_xp = get_xp_level(player_id)
    player_elo = get_elo_rating(player_id)
    player_reputation = get_house_reputation(player_id)

    # All requirements must be met
    return (
        player_xp >= tier.min_xp_level AND
        player_elo >= tier.min_elo AND
        player_reputation >= tier.min_reputation
    )
```

#### House Challenge Features:

- **Unique for Each Player:** No two players get identical challenges
- **Deterministic:** Same seed always generates same maze (for verification)
- **Difficulty Calibrated:** Based on player skill model
- **Adaptive:** Win rate monitored, difficulty adjusted dynamically
- **Verifiable:** All results auditable via replay
- **Profitable for Platform:** House edge targets ~65% win rate

#### House Win Rate Model (Adaptive):

```
Adaptive House Probability:
  - Target range: 60-70% house win rate
  - Operational target: 65%
  - Monitor daily completion rates

Dynamic Difficulty Adjustment:
  IF player_success_rate > 70%:
      Increase difficulty (more traps, tighter timer)
  ELSE IF player_success_rate < 60%:
      Decrease difficulty (more time, simpler layouts)
  ELSE:
      Maintain current difficulty

House Fairness Principle:
  - Advantage comes from difficulty design, not impossible conditions
  - Highly skilled players CAN win even highest tiers
  - No "rigged" mechanics, all mathematically verifiable
```

#### Business Logic:
- **Treasury Protection:** Only payout if reserves sufficient
- **Daily Limits:** Player can do unlimited house challenges
- **Reputation Impact:** Winning increases house reputation score
- **XP Rewards:** +25 XP per tier for completion
- **Season Tracking:** House performance affects seasonal achievements

---

### 6. SEASONAL SYSTEM

#### Core APIs Needed:
```
GET  /seasons/current        → Get current season info
GET  /seasons/{season_id}    → Get specific season
GET  /seasons/points         → Get player season points
POST /seasons/claim-reward   → Claim season rewards
GET  /seasons/pass-status    → Get pass info
POST /seasons/pass-purchase  → Buy premium pass
```

#### Database Models:
| Table | Purpose | Key Fields |
|-------|---------|-----------|
| `seasons` | Season metadata | season_id, season_number, start_date, end_date, theme, status, total_fund_allocated |
| `season_points` | Player standings | user_id, season_id, points_earned, rank_in_season, tier_earned |
| `season_achievements` | Seasonal badges | achievement_id, season_id, requirement_type, point_value |
| `season_passes` | Pass purchases | pass_id, user_id, season_id, pass_type (free/premium), purchased_date |
| `season_rewards` | Reward definitions | reward_id, season_id, point_threshold, reward_amount, reward_type |
| `season_rewards_earned` | Claimed rewards | earning_id, user_id, season_id, reward_id, claimed_date |

#### Season Structure:

```
Season Duration: 90 days (configurable)
Example:
  Season 1: Jan 1 - Mar 31
  Season 2: Apr 1 - Jun 30
  Season 3: Jul 1 - Sep 30
  Season 4: Oct 1 - Dec 31

At Season End:
  - Season Points RESET to 0
  - League Rank RESET (if soft reset enabled)
  - XP, Prestige, Legacy Points REMAIN (permanent)
  - Season achievements locked (can't earn more)
  - Rewards distributed
  - New season begins immediately
```

#### Season Points System:

**Sources of Season Points:**
- PvP Victory: +10 points (base, scales with opponent rating)
- House Challenge Success: +5 points (per tier)
- Tournament Participation: +50 points
- Seasonal Objective: +100 points (e.g., "win 10 matches")
- Seasonal Achievement: +500 points (major milestones)

**Season Points Reset:**
```
def end_season(season_id):
    # Reset season-specific data
    UPDATE season_points SET points_earned = 0
    UPDATE season_pass SET status = 'expired'
    UPDATE season_achievements SET earned = False

    # DO NOT RESET
    UPDATE progression_xp  -- Keep XP
    UPDATE progression_legacy  -- Keep legacy
    UPDATE progression_prestige  -- Keep prestige

    # Archive current season
    ARCHIVE season_id
```

#### Season Pass System:

**Free Pass:**
- Available to all players
- Rewards: +50% XP on all activities
- No cost
- All progress visible

**Premium Pass:**
- Cost: TBD (e.g., 50 tokens or USD $5)
- Rewards: +100% XP, +50% Season Points, exclusive cosmetics
- Purchasable anytime
- Refund if not used (first 7 days)

#### Season Rewards:

```
Rewards Funded By:
  1. Season Fund Reserve (primary)
  2. Tournament Revenue Allocation (10% of tournament fees)
  3. Platform Revenue Allocation (5% of platform fees)

Distribution Examples:
  Top 10 players: 1000 tokens each
  Top 50 players: 500 tokens each
  Top 100 players: 250 tokens each
  Top 1000 players: 50 tokens each

Sustainability Rule:
  Total_Rewards_Available = Season_Fund_Budget
  IF rewards_needed > budget:
      Scale rewards down proportionally
      Increase season fund next quarter
```

#### Business Logic:
- **Reward Funding:** Must never exceed Season Fund balance
- **Tier Achievements:** Reaching league tiers = seasonal achievement
- **Pass Progression:** Daily login streaks earn bonus points
- **Leaderboard Visibility:** Top 100 always displayed publicly

---

### 7. TOURNAMENT SYSTEM

#### Core APIs Needed:
```
GET  /tournaments            → List active tournaments
GET  /tournaments/{id}       → Tournament details
POST /tournaments/{id}/enter → Register for tournament
GET  /tournaments/{id}/bracket  → View brackets
GET  /tournaments/{id}/results  → Get final results
POST /tournaments/{id}/claim  → Claim rewards
```

#### Database Models:
| Table | Purpose | Key Fields |
|-------|---------|-----------|
| `tournaments` | Tournament metadata | tournament_id, tournament_type, entry_fee, max_participants, prize_pool, duration, start_date, status |
| `tournament_participants` | Registrations | participant_id, tournament_id, user_id, entry_fee_locked, qualification_met, final_placement |
| `tournament_brackets` | Match structure | bracket_id, tournament_id, round_number, seed_1, seed_2, winner, status |
| `tournament_matches` | Match results | match_id, tournament_id, bracket_id, player_1, player_2, result, match_data |
| `tournament_rewards` | Prize pool | reward_id, tournament_id, placement, reward_amount, reward_status |

#### Tournament Hierarchy:

| Type | Frequency | Entry Fee | Duration | Max Players | Prize Pool | Prestige |
|------|-----------|-----------|----------|-------------|------------|----------|
| Daily | Every 24h | 5 tokens | 24h | Unlimited | Dynamic | Low |
| Weekly | Every 7d | 25 tokens | 7d | 256 | Dynamic | Medium |
| Monthly | Every 30d | 100 tokens | 30d | 512 | Fixed | High |
| Seasonal | Every 90d | 500 tokens | 90d | 1000 | Large | Very High |
| World | Annual | 1000 tokens | 30d | 2000 | Massive | Max |

#### Tournament Qualification:

```
def check_tournament_eligibility(player_id, tournament_id):
    tournament = get_tournament(tournament_id)
    player = get_player(player_id)

    # Base eligibility
    if tournament.qualification_required:
        required_elo = tournament.min_elo
        if player.current_elo < required_elo:
            return False

    # League eligibility
    if tournament.league_restricted:
        if player.league_tier not in tournament.allowed_leagues:
            return False

    # Reputation eligibility
    if tournament.requires_verified:
        if not player.kyc_verified:
            return False

    # Trust score check
    if tournament.min_trust_score:
        if player.trust_score < tournament.min_trust_score:
            return False

    return True
```

#### Tournament Bracket Structure:

```
Single Elimination (Daily/Weekly):
  Round 1: 256 → 128 matches
  Round 2: 128 → 64 matches
  Round 3: 64 → 32 matches
  Round 4: 32 → 16 matches (Quarterfinals)
  Round 5: 16 → 8 matches (Semifinals)
  Round 6: 8 → 4 matches (Semifinals)
  Round 7: 4 → 2 matches (Finals)
  Round 8: 2 → 1 match (Champion)

Prize Distribution (Example Weekly):
  1st place (champion): 75 tokens
  2nd place: 40 tokens
  3rd-4th place: 20 tokens each
  5th-8th place: 10 tokens each
  9th-32nd place: 5 tokens each
```

#### Prize Pool Calculation:

```
Tournament_Entry_Fee = 25 tokens
Max_Participants = 256

Total_Collected = 256 × 25 = 6400 tokens
Platform_Fee = 10% × 6400 = 640 tokens
Prize_Pool = 6400 - 640 = 5760 tokens

Distribution:
  Winners fund = 5760 tokens
  Platform revenue = 640 tokens (to Platform Revenue Reserve)
```

#### Business Logic:
- **All tournaments treasury-backed** (prizes funded before tournament starts)
- **No overpayment:** Scale prizes down if fewer participants
- **Guaranteed payouts:** Winners always receive promised rewards
- **Escalating stakes:** Higher tournaments = higher entry fees and prizes
- **Prestigious display:** Tournament wins displayed prominently on profile

---

### 8. TREASURY & FINANCIAL SYSTEM

#### Core APIs Needed:
```
GET  /treasury/balance          → Get reserve balances
GET  /treasury/liabilities      → Get total liabilities
GET  /treasury/coverage-ratio   → Calculate solvency
GET  /treasury/audit-report     → Get financial audit
POST /treasury/reconcile        → Trigger reconciliation
GET  /treasury/health-score     → Get treasury health
```

#### Database Models:
| Table | Purpose | Key Fields |
|-------|---------|-----------|
| `treasury_accounts` | Reserve accounts | account_id, account_type, balance, min_threshold, status |
| `reserves` | Segregated funds | reserve_id, name, balance, purpose, monthly_target, max_balance |
| `financial_ledger` | Double-entry accounting | entry_id, timestamp, debit_account, credit_account, amount, description, verified_by |
| `reserve_snapshots` | Daily backup | snapshot_id, snapshot_date, reserve_balances_json, calculated_liabilities |

#### Reserve Structure (Segregated Funds):

**1. Player Funds Reserve**
- **Purpose:** Money belonging to players
- **Calculation:** Sum of all player wallet balances
- **Rules:**
  - MUST always equal sum of all player wallets
  - Cannot be used for operations/marketing/development
  - Highest priority claim on platform assets
  - Immobilized (cannot be touched)
- **Target:** 100% coverage ratio (Liabilities ≤ Reserve)

**2. Platform Revenue Reserve**
- **Sources:**
  - PvP platform fees (10% of match pots)
  - Tournament platform fees
  - Premium season pass sales
  - Cosmetic sales (future)
  - Sponsorships (future)
- **Uses:**
  - Operations (server costs, salaries)
  - Development (new features)
  - Marketing (user acquisition)
- **Strategy:** Reinvest 80%, allocate 20% to other reserves

**3. Season Fund Reserve**
- **Purpose:** Season reward payouts
- **Funding:** 5% of Platform Revenue
- **Cycle:** Replenish monthly to ensure sufficient for next season
- **Controls:**
  - Threshold alert if < 1 month funding
  - Manual approval for large disbursements

**4. Championship Fund Reserve**
- **Purpose:** Tournament prize pools
- **Funding:** 10% of Platform Revenue
- **Calculation:**
  - Calculate monthly tournament volume
  - Reserve 3x monthly amount
- **Release:** Locked until tournament begins

**5. Jackpot Fund Reserve**
- **Purpose:** Special promotion prizes, bonus pools
- **Funding:** 5% of Platform Revenue
- **Uses:** Limited-time high-reward challenges
- **Max Payout:** Single event capped at fund size ÷ 10

**6. Emergency Reserve**
- **Purpose:** Contingency buffer for crises
- **Target:** 20% of Platform Revenue Reserve
- **Uses:** System failures, fraud recovery, regulatory fines
- **Release:** CEO + CFO approval required

#### Treasury Architecture:

```
Financial Flow:

INFLOW:
  Player Deposits → Payment Provider → Confirmed →
  Player Funds Reserve + Platform Revenue

MATCH ENTRY:
  Entry Fees → Locked Wallet → Match Completes →
  Winner's Prize from Prize Pool → Ledger Entry

REWARDS:
  Season/Tournament Rewards → Championship Fund →
  Ledger Entry → Player Wallet (LOCKED until withdrawal)

WITHDRAWAL:
  Player Initiates → KYC Check → AML Screen →
  Withdrawal Fee (1%) → Payment Provider →
  Bank Transfer → Player Funds decreased
```

#### Financial Philosophy:

**Core Rules:**
1. **Auditability:** Every token tracked
2. **Reconciliation:** Ledger must match reality daily
3. **Coverage:** Liabilities never exceed reserves
4. **Segregation:** Funds never mixed without approval
5. **Priority:** Player funds > all other claims
6. **Immutability:** Financial records never deleted

#### Solvency Calculations:

```
Total_Player_Liabilities = Sum(all_player_wallet_balances)
Player_Funds_Reserve_Balance = ?
Coverage_Ratio = Player_Funds_Reserve / Total_Player_Liabilities

Target_Ratio = 1.0 (100% coverage)
Alert_Yellow = 0.95 (95% - warning)
Alert_Red = 0.90 (90% - emergency)

Solvency_Status:
  IF Coverage_Ratio >= 1.0: GREEN (Fully solvent)
  IF 0.95 <= Coverage_Ratio < 1.0: YELLOW (Minor deficit)
  IF Coverage_Ratio < 0.95: RED (Major concern)

Emergency Actions (if RED):
  1. Halt all withdrawals > USD $1000
  2. Notify CEO
  3. Audit immediately
  4. Consider freezing platform
  5. Legal/regulatory notification
```

#### Ledger Structure (Double-Entry):

```
Every transaction has two entries (debit ≠ credit):

Example 1: Player deposits USD $100 (10 tokens)
  Entry 1: DEBIT Player Funds Reserve (10 tokens)
  Entry 2: CREDIT Bank Account (USD 100 + 2.5% fee)

Example 2: Player wins 20-token PvP match
  Entry 1: DEBIT Prize Pool Account (20 tokens)
  Entry 2: CREDIT Player Wallet (20 tokens)

Example 3: Platform collects 2-token fee (10% of 20-token pot)
  Entry 1: DEBIT PvP Match Pot (2 tokens)
  Entry 2: CREDIT Platform Revenue Reserve (2 tokens)

All entries immutable once recorded.
```

#### Business Logic:
- **No Negative Balances:** All accounts start at 0, can't go negative
- **Atomic Transactions:** All-or-nothing, no partial transactions
- **Delayed Settlement:** Some transactions settle after 24 hours (fraud check)
- **Monthly Rebalancing:** Move funds between reserves to maintain targets

---

### 9. ANTI-CHEAT & SECURITY SYSTEM

#### Core APIs Needed:
```
POST /security/validate-move     → Validate game move
GET  /security/risk-score        → Get account risk
POST /security/device-check      → Verify device
GET  /security/audit-log         → Get security events
POST /security/report-exploit    → Report suspicious activity
```

#### Database Models:
| Table | Purpose | Key Fields |
|-------|---------|-----------|
| `device_fingerprints` | Device tracking | fingerprint_id, user_id, device_hash, os, browser, last_seen, first_seen |
| `anti_bot_scores` | Bot detection | score_id, user_id, risk_score, last_updated, signals_detected |
| `security_events` | Incident log | event_id, user_id, event_type, severity, timestamp, details |
| `replay_verification` | Replay validation | replay_id, verification_hash, status, verified_date |
| `suspicious_accounts` | Flagged accounts | account_id, user_id, reason, status, investigation_date |

#### Server Authoritative Architecture:

**Client Trust Level = ZERO for:**
- Wallet balance calculations
- XP and reward generation
- Ranking calculations
- Match result determination
- Challenge seed generation
- All business logic

**Client Trusted Only For:**
- Display rendering
- Audio playback
- Visual effects
- User input (coordinates, clicks)

**Server Must Always:**
- Validate every move
- Recalculate balances
- Verify rankings
- Regenerate challenges
- Confirm all results

#### Movement Validation:

```python
def validate_move(match_id, player_id, move_data):
    """
    Validates every player move server-side.
    CLIENT MOVE NEVER TRUSTED.
    """
    match = get_match(match_id)
    challenge = get_challenge(match.maze_id)

    # Check move is valid in maze
    current_position = get_player_position(match_id, player_id)
    new_x, new_y = move_data['coordinates']

    # 1. Is new position in valid path?
    if not is_valid_path(challenge, current_position, (new_x, new_y)):
        return False, "Invalid move - not in maze path"

    # 2. Is player alive?
    if not is_player_alive(match_id, player_id):
        return False, "Player already lost"

    # 3. Is move within game rules?
    if not follows_game_rules(match_id, move_data):
        return False, "Violates game rules"

    # 4. Is timing valid?
    if move_time_invalid(match, move_data):
        return False, "Move timing invalid"

    # If all checks pass, accept move
    update_player_position(match_id, player_id, (new_x, new_y))
    log_move(match_id, player_id, move_data, timestamp=now())
    return True, "Move accepted"
```

#### Anti-Bot Engine:

**Bot Detection Signals:**
- **Reaction Time Consistency:** Humans vary; bots are consistent
  - Humans: 150-500ms reaction time, variable
  - Bots: 10-50ms, highly consistent
  - Alert if: < 100ms for 100+ moves

- **Movement Precision:** Humans make mistakes; bots don't
  - Optimal path efficiency > 95%: Suspicious
  - Alert if: > 10 consecutive perfect moves

- **Completion Consistency:** Humans perform variably
  - Same maze completion time ± 2%: Suspicious
  - Alert if: 50+ identical completion times

- **Input Patterns:** Bots follow patterns
  - Click timing regularity
  - Movement direction patterns
  - Decision point analysis

**Risk Scoring:**

```
Risk_Score = 0-100

Calculation:
  reaction_time_score = analyze_reaction_times()      # 0-25
  movement_precision_score = analyze_precision()      # 0-25
  completion_consistency_score = analyze_consistency()  # 0-25
  input_pattern_score = analyze_patterns()            # 0-25

  Risk_Score = sum(all_scores)

Green (0-30): Normal human behavior
Yellow (31-70): Suspicious patterns detected
Red (71-100): High probability of automation

Actions:
  Green: Allow normal play
  Yellow: Monitor closely, require verification challenge
  Red: Suspend account, flag for investigation
```

#### Device Fingerprinting:

```python
def generate_fingerprint(request):
    """
    Creates unique device identifier from hardware characteristics
    """
    fingerprint_data = {
        'user_agent': request.headers['User-Agent'],
        'screen_resolution': request.body['screen_resolution'],
        'timezone': request.body['timezone'],
        'language': request.body['language'],
        'hardware_concurrency': request.body['cpu_cores'],
        'device_memory': request.body['memory_gb'],
        'gpu_info': request.body['gpu_model'],
        'installed_plugins': request.body['browser_plugins'],
        'canvas_fingerprint': request.body['canvas_hash'],
        'webgl_data': request.body['webgl_vendor'],
    }

    device_hash = SHA256(json.dumps(fingerprint_data))
    return device_hash

def check_account_sharing(user_id, new_fingerprint):
    """
    Detect if account is being used from multiple devices
    """
    known_devices = get_user_devices(user_id)

    if new_fingerprint in known_devices:
        return "Known device"

    # Check if too many new devices in short time
    new_devices_30d = get_new_devices(user_id, days=30)
    if len(new_devices_30d) > 3:
        return "High device churn - possible account sharing"

    # Check if devices in different geographic locations simultaneously
    if impossible_travel_detected(user_id):
        return "Impossible travel - simultaneous locations"

    return "Device added to whitelist"
```

#### Replay Verification:

```python
def verify_replay(replay_id):
    """
    Cryptographically verify replay authenticity
    """
    replay = get_replay(replay_id)
    match = get_match(replay.match_id)

    # 1. Recreate challenge from seed
    expected_challenge = generate_challenge(
        maze_seed=match.maze_seed,
        difficulty=match.difficulty_score
    )

    # 2. Replay all moves
    for frame in replay.frames:
        # Validate frame move against challenge
        if not validate_move_frame(expected_challenge, frame):
            return False, "Frame contains invalid move"

    # 3. Verify hash
    expected_hash = SHA256(serialize_replay_frames(replay.frames))
    if expected_hash != replay.verification_hash:
        return False, "Replay hash mismatch - tampered"

    # 4. Check completion time
    if replay.completion_time != match.completion_time:
        return False, "Time mismatch"

    return True, "Replay verified authentic"
```

#### Security Rules:

**Authentication:**
- Email + Password (12 chars minimum, complex)
- MFA required for withdrawals
- Device fingerprinting
- Session timeout: 30 days

**API Security:**
- JWT token validation on every request
- Rate limiting: 1000 requests/minute per user
- Input sanitization on all fields
- SQL injection prevention (parameterized queries)
- XSS prevention (output encoding)
- CSRF token on state-changing requests

**Infrastructure Security:**
- TLS 1.3+ for all communications
- Encrypted database at rest
- Encrypted backups
- VPN/firewall for internal services
- Zero-trust network architecture

---

### 10. REPLAY & VERIFICATION SYSTEM

#### Core APIs Needed:
```
GET  /replays/{replay_id}       → Get replay data
POST /replays/{replay_id}/verify → Verify replay authenticity
GET  /replays/history           → Get user replay list
POST /replays/{replay_id}/report → Report replay issue
```

#### Database Models:
| Table | Purpose | Key Fields |
|-------|---------|-----------|
| `replays` | Replay records | replay_id, match_id, recording_data, verification_hash, created_at, expires_at |
| `replay_frames` | Movement data | frame_id, replay_id, frame_number, position_x, position_y, move_type, timestamp |
| `replay_metadata` | Replay info | replay_id, duration_seconds, player_1_name, player_2_name, final_result, winner_id |
| `replay_disputes` | Challenges | dispute_id, replay_id, complainant_id, reason, status, resolution |

#### Replay Recording:

**What Gets Recorded:**
- Player 1 position every frame
- Player 2 position every frame
- Lives remaining (both players)
- Completion percentage
- Timer state
- Events (traps, deaths, completion)
- Final result and timing

**Storage:**
- Compressed movement data (not video)
- Deterministic: Same input always generates same output
- Size: ~10KB per match typically
- Retention: 90 days minimum, archivable after

**Compression:**
- Frame-to-frame delta encoding (only position changes)
- Timestamps encoded efficiently
- No video file (just player coordinates)
- Playable back from seed + frame data

#### Verification Features:

**Deterministic Replay:**
- Start with maze seed
- Feed recorded moves
- Regenerate exact same challenge state
- Verify final result matches

**Cryptographic Verification:**
- Replay hash created at completion
- Hash includes all move data
- Tampering detected immediately
- Certificate chain verifies server authority

**Dispute Resolution:**
- Player challenges match outcome
- Admin can replay and verify
- Evidence stored permanently
- Reversal possible only if fraud detected

#### Replay Use Cases:

1. **Player Verification:** "Did I actually lose this match?"
2. **Spectator Viewing:** Tournament spectators watch live replays
3. **Leaderboard Verification:** Verify top scores are legitimate
4. **Cheat Detection:** Identify impossible patterns
5. **Dispute Resolution:** Settle player complaints
6. **Platform Analytics:** Analyze gameplay patterns

---

### 11. LEADERBOARDS & RANKINGS

#### Core APIs Needed:
```
GET  /leaderboards/global     → Top 1000 globally
GET  /leaderboards/country/{code} → Country rankings
GET  /leaderboards/league/{tier}  → League-specific
GET  /leaderboards/season/{id}    → Seasonal standings
GET  /player/{id}/rank        → Get specific player rank
```

#### Database Models:
| Table | Purpose | Key Fields |
|-------|---------|-----------|
| `leaderboard_snapshots` | Cached rankings | snapshot_id, snapshot_date, board_type, rank_data_json |
| `player_rankings` | Current position | user_id, global_rank, country_rank, league_rank, elo_rating, last_updated |
| `historic_rankings` | Historical tracking | ranking_id, user_id, date, rank_position, elo_rating |

#### Leaderboard Types:

| Type | Frequency | Criteria | Visibility |
|------|-----------|----------|-----------|
| Global | Real-time | ELO rating | Top 1000 public |
| Country | Daily | ELO + Country | Top 100 per country |
| League | Daily | League tier + points | All players in league |
| Seasonal | Daily | Season points | Reset each season |
| Tournament | Real-time | Tournament placement | During tournament |

#### Ranking Calculation:

```python
def calculate_global_rankings():
    """
    Update global leaderboard daily
    """
    players = get_all_active_players()

    rankings = []
    for player in players:
        elo = get_current_elo(player.id)
        country = player.country

        rankings.append({
            'user_id': player.id,
            'elo_rating': elo,
            'country': country,
            'matches_played': get_match_count(player.id),
            'win_rate': calculate_win_rate(player.id),
        })

    # Sort by ELO descending
    rankings.sort(key=lambda x: x['elo_rating'], reverse=True)

    # Assign ranks
    for rank, player_data in enumerate(rankings, start=1):
        player_data['global_rank'] = rank

        if rank <= 1000:  # Only top 1000 shown
            player_data['public_visible'] = True

    # Store snapshot
    store_leaderboard_snapshot(rankings)

    return rankings
```

#### Private Profile Data:

- **Shown on Profile:** ELO, rank, wins, losses, win rate
- **Hidden from Public:** Exact withdrawal times, payment methods, full transaction history
- **Admin Only:** Risk scores, KYC status, security flags

---

### 12. ACHIEVEMENTS & LEGACY SYSTEM

#### Core APIs Needed:
```
GET  /achievements             → Get all achievements
GET  /achievements/{id}        → Get specific achievement
GET  /player/achievements      → Get player achievements
GET  /player/legacy            → Get legacy info
POST /achievements/{id}/claim  → Claim reward
```

#### Database Models:
| Table | Purpose | Key Fields |
|-------|---------|-----------|
| `achievements` | Achievement definitions | achievement_id, name, description, requirement_type, reward_xp, icon_url |
| `user_achievements` | Player progress | user_achievement_id, user_id, achievement_id, earned_date, progress_percent |
| `legacy_ranks` | Legacy progression | user_id, legacy_points, rank_tier, achievements_earned, hall_of_fame |

#### Achievement Categories:

**Gameplay Achievements:**
- Win 10 PvP matches
- Win 25 PvP matches
- Achieve 50-match win streak
- Complete all house tiers
- Win tournament

**Progression Achievements:**
- Reach XP level 100
- Reach Prestige I
- Reach Diamond league
- Earn 1000 legacy points
- Complete seasonal challenge

**Social Achievements:**
- Invite 5 friends
- Create clan
- Participate in clan tournament
- Reach 100 followers

**Special Achievements:**
- Founder status (before launch)
- Early adopter (first 10k players)
- Platform ambassador
- Bug bounty contributor

#### Legacy System:

```
Legacy Tiers (Based on Legacy Points):
  Bronze: 0-100 points
  Silver: 101-500 points
  Gold: 501-1500 points
  Platinum: 1501-3000 points
  Diamond: 3001-5000 points
  Legend: 5001+ points

Hall of Fame:
  Top 1000 legacy point holders
  Lifetime recognition
  Special badge on profile
  Featured on platform

Legacy Points Earned From:
  Season participation: +10 points
  Tournament success: +50 points (per placement)
  House completion: +5 points (per tier)
  PvP activity: +1 point (per match)
  Seasonal achievement: +100 points
  Founder status: +500 points (one-time)
```

---

### PHASE 3: FINANCIAL INFRASTRUCTURE & COMPLIANCE

### 3. CORE COMPONENTS

#### 1. ADVANCED TREASURY & DOUBLE-ENTRY LEDGER

#### Core APIs Needed:
```
GET  /treasury/accounts         → Get all reserve accounts
GET  /treasury/balance          → Get current balance
POST /treasury/reconcile        → Start reconciliation
GET  /treasury/reconcile-status → Check reconciliation progress
GET  /treasury/audit-report     → Generate financial audit
GET  /treasury/solvency         → Calculate solvency ratio
```

#### Database Models:
| Table | Purpose | Key Fields |
|-------|---------|-----------|
| `treasury_accounts` | Reserve accounts | account_id, account_name, account_type, balance, min_threshold, max_capacity, status |
| `double_entry_ledger` | All transactions | entry_id, transaction_id, debit_account_id, credit_account_id, amount, description, approved_by, timestamp |
| `reserve_snapshots` | Daily backups | snapshot_id, snapshot_date, all_reserves_json, total_liabilities, coverage_ratio |
| `reconciliation_logs` | Audit trail | log_id, reconciliation_date, discrepancies_found, resolution, status |

#### Financial Double-Entry Pattern:

```
Every transaction creates TWO immutable ledger entries:
  Entry 1: DEBIT (fund decreases)
  Entry 2: CREDIT (fund increases)
  Sum(Debits) = Sum(Credits) → Accounting Equation Always Balanced

Example: Player deposits USD $100
  DEBIT:  Bank Account                +100 USD
  CREDIT: Player Funds Reserve        -100 USD
  (converted to tokens at 1:10 rate)

Example: Player wins 20-token PvP
  DEBIT:  Match Prize Pool Account    +20 tokens
  CREDIT: Player Wallet               -20 tokens

All entries:
  - Immutable (never changed after creation)
  - Timestamped (UTC with milliseconds)
  - Approved (at least one signature)
  - Auditable (full chain preserved)
```

#### Reserve Reconciliation Workflow:

```
1. COLLECTION PHASE
   - Fetch all player wallet balances
   - Sum = Expected Player Funds Reserve
   - Fetch all reserve account balances
   - Get external payment provider balances
   - Get bank account balance

2. VALIDATION PHASE
   - Player Funds Reserve ≥ Sum(Player Wallets)?
   - All reserve accounts positive?
   - No orphaned transactions?
   - All ledger entries balanced?

3. COMPARISON PHASE
   - Internal records vs external statements
   - Payment provider confirmations
   - Bank statement matching
   - Discrepancy analysis

4. ADJUSTMENT PHASE
   - Identify root cause of any discrepancies
   - Create adjustment journal entries if needed
   - Document all changes
   - Obtain approval signatures

5. REPORTING PHASE
   - Generate reconciliation report
   - Archive for audit
   - Alert if issues found
   - Update executive dashboard

Frequency:
  Real-time: Critical accounts (player funds, bank)
  Daily: Comprehensive reconciliation
  Weekly: Deep financial analysis
  Monthly: Executive reporting
```

#### Solvency Monitoring:

```python
def calculate_solvency_ratio():
    """
    Track platform ability to cover all player liabilities
    """
    total_player_wallets = sum_all_player_balances()
    player_funds_reserve = get_reserve_balance('PLAYER_FUNDS')

    # Coverage ratio = how many times over can reserves cover liabilities
    coverage_ratio = player_funds_reserve / total_player_wallets

    # Treasury health score (0-100)
    if coverage_ratio >= 1.10:
        health_score = 100
        status = "EXCELLENT"
    elif coverage_ratio >= 1.05:
        health_score = 90
        status = "GOOD"
    elif coverage_ratio >= 1.0:
        health_score = 75
        status = "ADEQUATE"
    elif coverage_ratio >= 0.98:
        health_score = 50
        status = "CONCERNING"
    else:
        health_score = 0
        status = "CRITICAL"

    # Alert thresholds
    if coverage_ratio < 0.95:
        send_alert_to_cfo("SOLVENCY_RED", coverage_ratio)
    elif coverage_ratio < 1.0:
        send_alert_to_cfo("SOLVENCY_YELLOW", coverage_ratio)

    return {
        'coverage_ratio': coverage_ratio,
        'health_score': health_score,
        'status': status,
        'total_liabilities': total_player_wallets,
        'available_reserves': player_funds_reserve,
    }
```

---

#### 2. PAYMENT PROVIDER INTEGRATION

#### Core APIs Needed:
```
POST /payments/deposit          → Initiate deposit
POST /payments/withdrawal       → Initiate withdrawal
GET  /payments/status/{id}      → Check payment status
GET  /payments/methods          → Get available payment methods
POST /payments/webhook          → Receive provider updates
GET  /payments/reconcile        → Reconcile with provider
```

#### Database Models:
| Table | Purpose | Key Fields |
|-------|---------|-----------|
| `payment_providers` | Provider configs | provider_id, provider_name, api_endpoint, status, supported_regions |
| `deposits` | Deposit records | deposit_id, user_id, amount_local_currency, amount_tokens, provider, status, provider_tx_id |
| `withdrawals` | Withdrawal records | withdrawal_id, user_id, amount_tokens, amount_local_currency, provider, bank_account_id, status |
| `payment_audit` | Transaction log | audit_id, payment_id, timestamp, old_status, new_status, provider_response |

#### Deposit Processing Flow:

```
1. USER INITIATES
   - Select payment method
   - Enter amount (min USD $10)
   - Enter payment details
   - Submit

2. SERVER PROCESSES
   - Validate amount > minimum
   - Check payment method available in region
   - Create deposit record (status: PENDING)
   - Generate payment session with provider
   - Redirect user to provider

3. PAYMENT PROVIDER PROCESSES
   - Collect payment
   - Perform fraud checks
   - Confirm or decline

4. WEBHOOK CONFIRMATION
   - Provider sends webhook: CONFIRMED
   - Server validates webhook signature
   - Create ledger entries:
     * DEBIT: Bank Account
     * CREDIT: Player Funds Reserve
   - Credit player wallet (status: COMPLETED)
   - Emit notification: "Deposit Received"

5. PLAYER SEES
   - Balance updated immediately
   - Can use tokens for matches
   - Transaction in history
```

#### Withdrawal Processing Flow:

```
1. USER INITIATES
   - Enter withdrawal amount
   - Confirm bank account
   - Submit

2. VERIFICATION PHASE
   - Check account verified (KYC)
   - Check wallet balance ≥ amount
   - Check within daily/monthly limits
   - Lock tokens (debit Live, credit Pending)
   - Status: VERIFICATION_PENDING

3. AML SCREENING
   - Screen against OFAC list
   - Check for suspicious patterns
   - Review transaction history
   - If suspicious: Flag for manual review
   - Status: AML_SCREENING_COMPLETE

4. ADMIN REVIEW (if high value or flagged)
   - Manual verification
   - Document review
   - Approve or deny
   - If approved: Status: APPROVED

5. PAYMENT PROVIDER INITIATION
   - Create withdrawal instruction
   - Send to bank/payment provider
   - Track provider transaction ID
   - Status: PROCESSING

6. SETTLEMENT
   - 1-3 business days
   - Provider confirms
   - Create ledger entries:
     * DEBIT: Player Funds Reserve
     * CREDIT: Bank Account
   - Unlock tokens
   - Player notified: "Withdrawal Complete"
   - Status: COMPLETED

Possible outcomes:
  - APPROVED → Processing → COMPLETED
  - DENIED → Tokens unlocked, returned to wallet
  - FAILED → Provider error, retry or refund
```

#### Supported Payment Methods:

| Provider | Regions | Processing Time | Fees |
|----------|---------|-----------------|------|
| Stripe (Cards) | Global | Instant | 2.9% + $0.30 |
| EFT | South Africa | 1-2 days | 2% |
| Instant EFT | South Africa | Minutes | 3% |
| PayFast | South Africa, Africa | Minutes | 2.5% |
| Ozow | South Africa | Minutes | 1% |

---

#### 3. AML & COMPLIANCE SYSTEM

#### Core APIs Needed:
```
POST /compliance/aml-screen     → Screen transaction for AML
GET  /compliance/kyc-status     → Get KYC verification status
POST /compliance/kyc-submit     → Submit KYC documents
GET  /compliance/risk-score     → Get account risk level
GET  /compliance/report         → Generate compliance report
```

#### Database Models:
| Table | Purpose | Key Fields |
|-------|---------|-----------|
| `aml_screenings` | AML checks | screening_id, user_id, amount, direction (deposit/withdrawal), status, provider_response, screening_date |
| `kyc_records` | KYC verification | kyc_id, user_id, verification_status, document_type, verification_date, verified_by, expiration_date |
| `compliance_events` | Compliance incidents | event_id, user_id, event_type, severity, timestamp, description, resolved |

#### KYC (Know Your Customer) Process:

```
Tier 1: Basic Registration (No KYC)
  - Create account
  - Email verification
  - Play with demo wallet
  - Limit: Can't deposit or withdraw

Tier 2: Email Verified
  - Verify email
  - Can deposit up to USD $1000/month
  - Can withdraw up to USD $500/month
  - Auto-screened deposits

Tier 3: Basic KYC (Identity Verified)
  - Submit government ID
  - Manual verification by provider
  - Can deposit USD $50,000/month
  - Can withdraw USD $10,000/month
  - Enhanced AML screening

Tier 4: Enhanced KYC (Full Verification)
  - Submit proof of address
  - Source of funds verification
  - Manual senior review
  - Can deposit/withdraw unlimited
  - Real-time monitoring

Enhanced Verification Triggers:
  - Withdrawal > USD $50,000
  - Deposit > USD $100,000
  - Rapid deposit/withdrawal pattern
  - Flagged for suspicious activity
```

#### AML Screening Rules:

```python
def aml_screen_transaction(user_id, amount, direction):
    """
    Screen transaction against AML rules
    """
    user = get_user(user_id)

    # Check OFAC list (US sanctions)
    if is_ofac_listed(user.name, user.country):
        return BLOCKED, "OFAC listing detected"

    # Check transaction velocity
    recent_transactions = get_transactions_24h(user_id)
    if sum(recent_transactions) > threshold_for_tier(user.kyc_tier):
        return FLAGGED, "Velocity threshold exceeded"

    # Check unusual patterns
    if is_unusual_pattern(user_id, amount, direction):
        return FLAGGED, "Unusual transaction pattern"

    # Check if account is young with large transaction
    account_age_days = (now() - user.created_at).days
    if account_age_days < 30 and amount > 1000:
        return FLAGGED, "Large transaction on young account"

    # Check for structuring (rapid small transactions)
    if detect_structuring(user_id):
        return FLAGGED, "Possible structuring detected"

    # If all checks pass
    return ALLOWED, "Transaction approved"
```

#### Compliance Reporting:

```
Daily Compliance Report:
  - Total deposits processed
  - Total withdrawals processed
  - Flagged transactions count
  - Blocked transactions count
  - KYC verifications completed
  - Suspicious activity investigations

Monthly Regulatory Filing:
  - Transaction report to authorities (if required)
  - Sanctions list hits
  - Customer due diligence updates
  - Incident summary
```

---

#### 4. FRAUD DETECTION & RISK MANAGEMENT

#### Core APIs Needed:
```
GET  /fraud/risk-score/{user_id}  → Get account risk
POST /fraud/report-suspicious      → Report suspicious activity
GET  /fraud/cases                  → List fraud investigations
POST /fraud/cases/{id}/resolve     → Resolve case
GET  /disputes/open               → List open disputes
```

#### Database Models:
| Table | Purpose | Key Fields |
|-------|---------|-----------|
| `risk_scores` | Player risk assessment | risk_id, user_id, financial_risk, behavioral_risk, account_risk, overall_score, updated_at |
| `fraud_cases` | Investigation records | case_id, case_type, user_id, amount, status, created_at, resolved_at, resolution |
| `dispute_records` | Complaint tracking | dispute_id, complainant_id, respondent_id, reason, evidence, status, ruling |

#### Risk Categories:

**Financial Fraud:**
- Chargebacks on deposits
- Stolen payment methods
- Money laundering patterns
- Unusual fund flows

**Account Abuse:**
- Account sharing
- Multiple accounts per player
- Credential sharing
- Unauthorized access

**Collusion:**
- Players coordinating wins
- Intentional losses
- Match fixing
- Reward sharing schemes

**Match Manipulation:**
- Obvious throws
- Impossible performance
- Bot usage
- Exploit abuse

**Bonus Abuse:**
- Sign-up bonus grinding
- Fake accounts for bonuses
- Welcome offer fraud
- Referral exploitation

**Payment Abuse:**
- Deposit and rapid withdrawal
- Chargeback after winning
- Multiple failed attempts
- Card testing

#### Dynamic Risk Scoring:

```python
def calculate_risk_score(user_id):
    """
    Continuously evaluate account risk
    Scale: 0-100 (higher = riskier)
    """
    user = get_user(user_id)

    # Financial Risk (0-25)
    financial_score = 0
    deposits_last_30d = get_deposits_total(user_id, days=30)
    withdrawals_last_30d = get_withdrawals_total(user_id, days=30)

    if deposits_last_30d == 0:
        financial_score += 10  # No deposits yet
    elif deposits_last_30d > 50000:
        financial_score += 15  # Unusually high

    if withdrawals_last_30d / deposits_last_30d > 0.8:
        financial_score += 10  # Rapid withdrawal pattern

    chargebacks = count_chargebacks(user_id)
    financial_score += min(chargebacks * 5, 25)

    # Behavioral Risk (0-25)
    behavioral_score = 0

    win_rate = get_win_rate(user_id)
    if win_rate > 0.95:
        behavioral_score += 15  # Suspiciously high

    # Check for bot behavior
    bot_signals = count_bot_signals(user_id)
    behavioral_score += min(bot_signals * 3, 25)

    # Account Risk (0-25)
    account_score = 0

    if user.kyc_status == "UNVERIFIED":
        account_score += 10

    device_count = count_unique_devices(user_id)
    if device_count > 5:
        account_score += 15  # Multiple devices

    location_diversity = check_location_diversity(user_id)
    if location_diversity > 3:
        account_score += 10  # Impossible travel

    # Overall Score
    overall_score = financial_score + behavioral_score + account_score

    # Risk Level
    if overall_score < 20:
        risk_level = "GREEN"
    elif overall_score < 50:
        risk_level = "YELLOW"
    else:
        risk_level = "RED"

    return {
        'overall_score': overall_score,
        'risk_level': risk_level,
        'financial_score': financial_score,
        'behavioral_score': behavioral_score,
        'account_score': account_score,
    }
```

#### Dispute Resolution Workflow:

```
1. COMPLAINT FILED
   - Player submits dispute
   - Provide evidence/reason
   - Case created (status: OPEN)

2. REVIEW PHASE
   - Examine replay data
   - Check match logs
   - Verify server calculations
   - Review player history

3. INVESTIGATION
   - If fraud suspected: Deeper investigation
   - Interview if needed
   - External evidence gathering
   - Policy application

4. DECISION
   - Uphold original result OR
   - Reverse (refund entry fee) OR
   - Compensation (partial refund)

5. COMMUNICATION
   - Notify both parties
   - Document reasoning
   - Update case status: RESOLVED

6. APPEAL
   - Option to appeal within 7 days
   - Different reviewer
   - Final decision final

Case Statuses:
  OPEN → UNDER_REVIEW → DECIDED → RESOLVED
  At any point can be ESCALATED
```

---

#### 5. RECONCILIATION & SOLVENCY ENGINE

#### Core APIs Needed:
```
POST /reconciliation/start       → Start reconciliation process
GET  /reconciliation/status      → Check progress
GET  /reconciliation/report      → Get results
POST /reconciliation/adjust      → Create adjustment entry
GET  /solvency/metrics           → Get solvency status
```

#### Reconciliation Workflow:

```
Step 1: BALANCE COLLECTION (Automated, Real-time)
  - Fetch all player wallet balances → Sum
  - Fetch all reserve account balances
  - Fetch payment provider balances
  - Fetch bank statement (if available)
  - Fetch locked tokens in matches

Step 2: INTERNAL VALIDATION (Automated, Daily)
  - Player wallets ≥ 0 all
  - No negative balances
  - Sum of wallet types = total
  - No orphaned transactions

Step 3: LEDGER VERIFICATION (Automated, Daily)
  - Verify accounting equation: Debits = Credits
  - Check no duplicate entries
  - Verify all amounts non-negative
  - Check timestamps in order

Step 4: PROVIDER RECONCILIATION (Automated, Daily)
  - Compare internal deposit records vs provider
  - Match transaction IDs
  - Verify amounts
  - Identify discrepancies

Step 5: SOLVENCY CHECK (Automated, Real-time)
  - Calculate coverage ratio
  - Compare to minimums
  - Alert if concerning

Step 6: ADJUSTMENT PHASE (Manual, if needed)
  - Identify discrepancies requiring adjustment
  - Create journal entries to correct
  - Document reasons
  - Require approval signatures

Step 7: REPORTING (Automated, Daily)
  - Generate reconciliation report
  - Archive for audit
  - Alert executives if issues
  - Update dashboards
```

#### Treasury Health Scoring:

```
Coverage Ratio = Total Reserves / Total Liabilities

Health Score Calculation:
  IF Coverage_Ratio >= 1.20: Health_Score = 100 (EXCELLENT)
  IF Coverage_Ratio >= 1.10: Health_Score = 90 (VERY GOOD)
  IF Coverage_Ratio >= 1.05: Health_Score = 80 (GOOD)
  IF Coverage_Ratio >= 1.00: Health_Score = 70 (ADEQUATE)
  IF Coverage_Ratio >= 0.98: Health_Score = 50 (CONCERNING)
  IF Coverage_Ratio >= 0.95: Health_Score = 30 (WARNING)
  IF Coverage_Ratio < 0.95: Health_Score = 0 (CRITICAL)

Alert Levels:
  Coverage >= 1.0: GREEN (All liabilities covered)
  0.95 <= Coverage < 1.0: YELLOW (Minor concern, review plan)
  Coverage < 0.95: RED (Emergency, halt withdrawals, notify CEO)
```

---

#### 6. REPORTING & ANALYTICS INFRASTRUCTURE

#### Core APIs Needed:
```
GET  /reports/executive         → Get executive summary
GET  /reports/treasury          → Get treasury report
GET  /reports/compliance        → Get compliance report
GET  /reports/risk              → Get risk report
GET  /dashboards/treasury       → Real-time treasury dashboard
GET  /dashboards/executive      → Real-time executive dashboard
```

#### Database Models:
| Table | Purpose | Key Fields |
|-------|---------|-----------|
| `report_definitions` | Report templates | report_id, report_name, frequency, recipients, template_json |
| `generated_reports` | Report instances | generated_id, report_id, generation_date, data_json, status |
| `dashboard_metrics` | Cached metrics | metric_id, metric_type, value, calculated_at |

#### Executive Dashboard (Real-Time):

**Key Metrics:**
- Active Players (today, week, month)
- New Players (today, week, month)
- Deposits (USD amount, token amount)
- Withdrawals (USD amount, token amount)
- Platform Revenue (fees collected)
- Treasury Health Score (0-100)
- Reserve Coverage Ratio
- Solvency Status (Green/Yellow/Red)
- Fraud Alert Count
- System Uptime %

**Visualizations:**
- Revenue trend (30-day chart)
- Player growth curve
- Deposit vs Withdrawal comparison
- Reserve balance trend
- Active match volume
- Regional breakdown

---

#### 7. DATABASE SCHEMA OVERVIEW (Phase 3)

**Primary Domains:**

| Domain | Purpose | Tables |
|--------|---------|--------|
| Users | Identity & profile | users, user_profiles, user_sessions, devices |
| Wallets | Token management | wallets, transactions, ledger_entries |
| Treasury | Financial reserves | treasury_accounts, reserves, reserve_snapshots |
| Transactions | All payments | deposits, withdrawals, settlement_records |
| Compliance | KYC & AML | kyc_records, aml_screenings, compliance_events |
| Fraud | Risk & investigation | risk_scores, fraud_cases, disputes |
| Reconciliation | Audit & verification | reconciliation_logs, journal_adjustments, audit_logs |
| Reporting | Analytics | report_definitions, generated_reports, dashboard_metrics |

---

#### 8. AUDIT & COMPLIANCE LOGGING

#### Core APIs Needed:
```
GET  /audit-logs              → Query audit trail
POST /audit/archive           → Archive old records
GET  /compliance/certifications → Get compliance proof
```

#### Database Models:
| Table | Purpose | Key Fields |
|-------|---------|-----------|
| `audit_logs` | Complete trail | log_id, user_id, action, resource_type, before_state, after_state, timestamp, approved_by |
| `compliance_archive` | Record preservation | archive_id, record_type, data_hash, timestamp, retention_until |

#### Audit Logging Requirements:

**What Gets Logged:**
- All financial transactions
- All player withdrawals
- All admin actions
- All system changes
- All security events
- All KYC updates
- All dispute resolutions

**Immutability:**
- Audit logs cannot be deleted
- Logs cannot be modified
- Access logged
- Export trail recorded

**Retention:**
- Financial records: 7 years (regulatory)
- Audit logs: 5 years minimum
- Compliance records: 10 years
- Dispute records: 3 years post-resolution

---

### PHASE 3 SUMMARY: FINANCIAL INFRASTRUCTURE COMPONENTS

1. ✅ Advanced Treasury & Double-Entry Ledger
2. ✅ Payment Provider Integration (Deposits/Withdrawals)
3. ✅ AML & Compliance System (KYC, screening)
4. ✅ Fraud Detection & Risk Management (Risk scoring, disputes)
5. ✅ Reconciliation & Solvency Engine (Daily reconciliation, health monitoring)
6. ✅ Payment Reconciliation (Provider statement matching)
7. ✅ Reporting & Analytics Infrastructure (Dashboards, reports)
8. ✅ Audit & Compliance Logging (Immutable trails, retention)

---

### IMPLEMENTATION PRIORITY

### Phase 1 Build Order (Critical Path):
1. User System & Authentication (foundation)
2. Wallet & Token System (enables payments)
3. Treasury & Financial System (tracks money)
4. Match & Gameplay System (core feature)
5. Anti-Cheat & Security (prevents cheating)
6. Progression Systems (keeps players engaged)
7. House Challenge Engine (secondary gameplay)
8. Seasonal System (long-term engagement)
9. Replay & Verification System (dispute resolution)
10. Tournament System (competitive path)
11. Leaderboards & Rankings (social motivation)
12. Achievements & Legacy (rewards)

### Phase 3 Build Order (Financial):
1. Payment Provider Integration (payments work first)
2. Advanced Ledger (track all money)
3. Reconciliation Engine (verify daily)
4. AML & Compliance (legal requirement)
5. Fraud Detection (protect treasury)
6. Audit Logging (compliance trail)
7. Reporting & Analytics (visibility)

---

### CRITICAL ARCHITECTURAL DECISIONS

### Server Authority Rules
- **NEVER** trust client for: rewards, XP, rankings, match results, balance calculations
- **ALWAYS** validate server-side
- **Client** is input and display only

### Financial Integrity Rules
- **Double-entry accounting:** Every transaction has debit + credit
- **Segregated reserves:** Funds never mixed
- **Coverage rule:** Player liabilities ≤ reserves always
- **Immutability:** Financial records never deleted
- **Auditability:** Complete trail maintained

### Data Model Principles
- Event sourcing for financial events
- Immutable audit logs
- Never delete records (archive if needed)
- Timestamp everything (UTC)
- Cryptographic verification where possible

---

## Project Overview

<!-- Archived source: docs/backup/individual-markdown-2026-07-15/README.md -->

Skill Arena is a competitive gaming platform backend and frontend workspace.

### Backend

The backend is a Go service located in `backend/`.

Common verification commands:

```powershell
go test ./...
go vet ./...
go build ./...
gofmt -w .
```

### Frontend

The frontend is a Next.js application located in `frontend/`.

### Runtime Data

Development JSON data is kept under `backend/data/`. Production deployments should use the configured production services for database, cache, storage, payments, email and observability.
