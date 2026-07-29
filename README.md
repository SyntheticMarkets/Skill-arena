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
- [Sprint 6 Phase 1: Stray Arrows Architecture Review](#sprint-6-phase-1-stray-arrows-architecture-review)
- [Sprint 6 Phase 2: Maze Arena Architecture And Games Platform](#sprint-6-phase-2-maze-arena-architecture-and-games-platform)
- [Sprint 6 Phase 3: Puzzle Generator And Solver Design](#sprint-6-phase-3-puzzle-generator-and-solver-design)
- [Sprint 6 Phase 4: Implementation Blueprint](#sprint-6-phase-4-implementation-blueprint)
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

### Release 1.0 Architecture

Status: **Sprints 1-5 complete and frozen. Sprint 6 Implementation Phases 1 through 3 are complete and validated; Phase 4 has not started.**

Release 1.0 is organized as independently owned product domains. A frozen domain may receive bug fixes, security fixes, performance work, scalability work, or integration support, but its business contract may not be silently redesigned by a later sprint.

```text
Skill Arena Release 1.0
|
+-- Sprint 1: Identity & Security             [FROZEN]
|   `-- tag: sprint-1-v1.0-freeze
|
+-- Sprint 2: Player Platform / Arena Hub     [FROZEN]
|   `-- tag: sprint-2-v1.0-freeze
|
+-- Sprint 3: Financial Platform              [FROZEN]
|   `-- tag: sprint-3-v1.0-freeze
|
+-- Sprint 4: Admin CRM                       [COMPLETE - FROZEN]
|
+-- Sprint 5: Realtime Arena                  [COMPLETE - FROZEN]
|   `-- tag: sprint-5-v1.0-freeze
|
+-- Sprint 6: Maze Arena                      [IN PROGRESS - PHASES 1-3 COMPLETE]
|
+-- Sprint 7: Competition Platform            [PLANNED]
|   `-- Seasons, tournaments, leaderboards, and rewards
|
`-- Sprint 8: Production Launch               [PLANNED]
```

| Sprint | Domain | Release responsibility | Status |
|---|---|---|---|
| 1 | Identity & Security | Registration, verification, authentication, MFA, sessions, devices, and account recovery | Frozen |
| 2 | Player Platform | Arena Hub, navigation, player profile, notifications, support entry, and game discovery | Frozen |
| 3 | Financial Platform | Wallet, ledger, deposits, withdrawals, limits, assessments, responsible gaming, Payment Core, and Treasury contracts | Frozen |
| 4 | Admin CRM | Separate staff identity, permissions, operations, compliance, finance, support, audit, and monitoring application | Complete - frozen at `sprint-4-v1.0-freeze` |
| 5 | Realtime Arena | Authenticated gateway, presence, live events, reconnect, ordering, and distributed coordination | Complete - frozen at `sprint-5-v1.0-freeze` |
| 6 | Maze Arena | Deterministic puzzle pipeline, authoritative gameplay, PvP, replay, and game-specific presentation | Planned |
| 7 | Competition Platform | Tournament, season, leaderboard, reward, spectator, dispute, and competition settlement lifecycles | Planned |
| 8 | Production Launch | Provider certification, jurisdiction approval, deployment, disaster recovery, load, chaos, security, and launch-candidate verification | Planned |

#### Domain Boundaries

```text
Player Platform ----\
                     \
Admin CRM ------------> Versioned API and event contracts
                       |
                       v
                Platform Domains
             / Identity & Security
            /  Financial Platform
           /   Realtime Arena
          /    Game and Competition Platform
         v
PostgreSQL | Redis | Object Storage | Email | Payment Providers
```

- The Player Platform and Admin CRM are separate applications, security surfaces, navigation systems, and deployment units.
- Identity & Security is the authority for users, staff identities, sessions, MFA, devices, and revocation.
- The Financial Platform is the authority for money, Payment Core routing, ledger state, Treasury state, financial policy, and settlement.
- Admin CRM may operate frozen domains only through explicit, permission-protected APIs. It may not edit wallets directly or bypass financial state machines.
- Realtime Arena owns authenticated live transport and presence. Games consume its contracts instead of creating private transport layers.
- Arena Core owns game-agnostic sessions, actions, capabilities, versions, and replay contracts. Maze Arena remains Game Module 1.
- The Competition Platform consumes authoritative game outcomes and Financial Platform settlement contracts; it does not calculate or credit money independently.
- PostgreSQL stores authoritative transactional state, Redis stores ephemeral distributed state, and S3-compatible object storage stores durable growing artifacts.
- External providers remain behind domain interfaces. Player and CRM clients must not branch on payment, identity, email, or storage provider identity.

#### Release Rules

1. Only the current sprint may add business functionality.
2. Planning, wireframes, and high-fidelity UX approval precede implementation.
3. Every administrative action must be authenticated, permission-checked, attributable, and immutable in audit history.
4. Every financial transition must remain idempotent, transactional, provider-neutral, and reconcilable in integer minor units.
5. Every live game outcome must remain server-authoritative, deterministic, replayable, and versioned.
6. A sprint freezes only after its design, frontend, backend, security, API, tests, production, and evidence gates pass.
7. Sprint 8 may approve Release 1.0 only after all deployment tasks, external approvals, load tests, chaos tests, backup restore, disaster recovery, and launch-candidate checks pass.

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

Implementation status: **COMPLETE AND APPROVED**. The approved freeze is identified by the annotated tag `sprint-3-v1.0-freeze`. Sprint 4 subsequently completed and is frozen independently.

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

#### Sprint 3 Validation Report

Validation date: 2026-07-24.

Sprint 3 is approved for feature freeze. Payment-provider onboarding is a post-freeze deployment milestone. Sprint 4 subsequently completed without changing the frozen Financial Platform contract.

| Gate | Status | Evidence |
|---|---|---|
| Design | Pass | Wallet Overview, Deposit, Withdraw, Activity, Limits, and Assessment form one responsive player workspace. No approval, treasury, fraud, or KYC-review control exists in the player application. |
| Frontend | Pass | All money and lifecycle state comes from `/api/v1/financial`; loading, error, empty, policy-gated, evidence-upload, export, and responsible-gaming states are implemented. |
| Backend | Pass | Normalized PostgreSQL financial repositories, integer minor units, provider-neutral Payment Core, serializable settlement, manual withdrawal approval, provider balance checks, reserve gates, evidence, artifacts, and immutable journals are implemented. |
| Security | Pass | Financial writes require authentication, CSRF protection, verified email, completed eligibility, jurisdiction policy, limits, and idempotency. Adapter-owned signatures are verified before parsing; callback replay and contract mismatches fail closed. Provider routing and internal policy data are not serialized to players. |
| API | Pass | Player, generic provider callback, artifact, evidence, future-CRM transition, payout-destination, and reconciliation routes are versioned and documented below. |
| Tests | Pass | Provider contract, routing, health failover, lifecycle, HTTP integration, PostgreSQL, object-storage, frontend, and responsive regression suites pass. |
| Production | Platform complete; selected adapter pending | Production startup requires an explicitly active, configured provider adapter plus SMTP, PostgreSQL, Redis, and S3-compatible credentials. Provider selection is a deployment and commercial decision. |
| Freeze | Approved | The provider-independent Financial Platform passed final regression validation and is frozen at `sprint-3-v1.0-freeze`. |

Verification results:

- `go test ./... -count=1`: all packages passed; the final database package completed in 121.986 seconds and the server package in 8.331 seconds.
- Fresh disposable PostgreSQL 17 on an isolated local port: authentication, Arena Hub, migrations `004_financial_platform.sql` and `005_financial_completion.sql`, exact provider-neutral deposit transitions, settlement, evidence, payout destination, reserve audit, exact minor-unit balance, and journal verification passed. The cluster was stopped and removed after validation.
- `go vet ./...` and `go build ./...`: passed.
- Go coverage: 33.3% repository-wide; database 39.3%, Payment Core 63.1%, server 95.4%, and storage 63.4%.
- Real S3-compatible lifecycle: a checksum-verified MinIO server accepted signed bucket health, PUT, GET, integrity comparison, and DELETE operations.
- Vitest: 4 files and 6 tests passed; repository-wide frontend statement coverage is 25.78%.
- ESLint: passed with zero warnings. TypeScript: passed. Next.js 16.2.11 production build: passed with 23 generated routes.
- Playwright: all 18 Sprint 1, Sprint 2, and Sprint 3 tests passed across desktop, tablet, and mobile in a serialized 2.9-minute run.
- `npm audit --omit=dev --audit-level=high`: zero vulnerabilities.
- Docker is not installed on this workstation, so `docker compose config` and container startup remain environment validation gaps. Native production builds, fresh PostgreSQL, and real MinIO protocol validation passed.

Implemented production contracts:

- `BIGINT` minor-unit wallets, deposits, withdrawals, journal entries, limits, and reconciliations.
- Atomic deposit pending reserve, settlement, wallet credit, lifetime total, transition history, and journal append.
- Atomic withdrawal reserve, approval/rejection/processing/completion state machine, reserve return on rejection/failure, and final journal debit.
- Capability-driven Payment Core with full deposit, callback, signature, status-query, refund, payout, balance, reconciliation, health, country, currency, and idempotency contracts.
- Country, currency, method, availability, priority, cost, preferred-provider, and preflight-failover routing. The selected adapter remains internal and never changes the player API.
- Stripe remains a reference adapter for Checkout and Connect flows. No simulated PayFast, Ozow, Peach, Xsolla, Flutterwave, Card, EFT, or crypto adapter is registered in production.
- Provider/reference/amount/currency matching, duplicate protection, idempotent network retry, and financial notifications are generic Payment Core behavior.
- Provider-authenticated balance retrieval before deposit settlement, withdrawal processing, and reconciliation.
- Reserve checks stop settlement when provider funds cannot cover the requested movement and player liabilities; each check receives a SHA-256 identity and an S3-backed immutable audit artifact.
- Configurable jurisdiction policy for currency, age, methods, source-of-funds requirement, and daily/monthly limits.
- Player-controlled limit reductions, cooling-off, and self-exclusion; increases require compliance review.
- S3-compatible evidence, statement, financial export, treasury audit, and provider audit storage with ownership checks and SHA-256 verification; local filesystem remains development-only.
- AML review inputs record verification, high-value, daily velocity, risk classification, identity evidence, and source-of-funds evidence while the launch policy remains 100% manual approval.
- A provider-neutral compliance contract defines identity verification callbacks and AML screening integration without exposing review functions to players.
- Separate role-protected future-CRM APIs with no Admin component, route, or control in the player application.

#### Deployment Tasks (Post-Freeze)

Remaining deployment and external approval work:

| Priority | Evidence gap | Required completion |
|---|---|---|
| Deployment | Selected production adapter | After commercial approval, implement or activate the selected provider adapter and complete its sandbox certification. This is the remaining provider deployment task, not Payment Core architecture work. |
| High | Launch configuration | Supply selected-provider, SMTP, PostgreSQL, Redis, and S3-compatible secrets through the deployment secret manager and verify `/health/ready` in staging. |
| High | Jurisdiction approval | Obtain legal/compliance approval for the ZA launch rules, identity checks, sanctions/PEP provider, source-of-funds thresholds, retention, responsible-gaming limits, and supported payment methods. This is an external policy approval, not unfinished application code. |
| Medium | Container validation | Exercise the pinned Compose stack in CI or staging because Docker is unavailable on this workstation. |

Freeze decision: **SPRINT 3 APPROVED.** The provider-independent Financial Platform is complete. Implementing and certifying the commercially selected production adapter remains a deployment task. Sprint 4 subsequently consumed this contract through permission-protected APIs.

Completion-phase file inventory:

- Contract and deployment: `README.md`, `docker-compose.yml`, `frontend/Dockerfile`, `frontend/next.config.mjs`.
- Configuration: `backend/internal/config/config.go`, `backend/internal/config/config_test.go`.
- Financial persistence: `backend/internal/db/db.go`, `backend/internal/db/hub_postgres.go`, `backend/internal/db/financial_postgres.go`, `backend/internal/db/financial_completion.go`.
- Financial models and API: `backend/internal/models/financial.go`, `backend/internal/handlers/financial.go`, `backend/internal/server/server.go`.
- Provider and compliance boundaries: `backend/internal/payments/provider.go`, `backend/internal/payments/registry.go`, `backend/internal/payments/stripe.go`, `backend/internal/compliance/provider.go`.
- Object storage: `backend/internal/storage/storage.go`.
- Migrations: `backend/migrations/004_financial_platform.sql`, `backend/migrations/005_financial_completion.sql`, `backend/migrations/embed.go`.
- Backend verification: `backend/internal/db/financial_test.go`, `backend/internal/db/financial_postgres_integration_test.go`, `backend/internal/payments/provider_test.go`, `backend/internal/payments/stripe_test.go`, `backend/internal/server/financial_compliance_test.go`, `backend/internal/server/stripe_financial_integration_test.go`, `backend/internal/storage/storage_test.go`, `backend/internal/storage/s3_integration_test.go`.
- Player frontend: `frontend/app/lib/api.ts`, `frontend/app/wallet/page.tsx`, `frontend/styles/globals.css`.
- Frontend verification: `frontend/app/wallet/page.test.tsx`, `frontend/e2e/financial-platform.spec.ts`, `docs/proof/sprint-3-financial-platform/`.

#### Provider Integration Guide

This guide is the permanent onboarding contract for Xsolla, Peach Payments, Ozow, PayFast, Flutterwave, or any future provider. Adding an adapter must not change Wallet, Treasury, Ledger, statements, notifications, policy rules, payment states, or player APIs.

##### Adapter Boundary

A new adapter belongs in `backend/internal/payments`. It implements `payments.Provider` and owns all provider-specific:

- Credentials and configuration validation.
- HTTP clients, URLs, headers, request models, and response models.
- Callback signature verification and event-envelope parsing.
- Payment, refund, payout, balance, reconciliation, and health API calls.
- Provider reference values and payout-destination validation.
- Translation from provider statuses into Payment Core statuses.

Provider SDK types must not appear outside the adapter. The adapter must not import Wallet, Treasury, Ledger, notification, policy, or HTTP handler packages.

##### Required Interface

Every production adapter implements:

1. `Descriptor`
2. `ValidateConfiguration`
3. `CreateDepositSession`
4. `VerifySignature`
5. `ParseCallback`
6. `QueryPaymentStatus`
7. `Refund`
8. `ValidatePayoutDestination`
9. `CreatePayout`
10. `QueryPayoutStatus`
11. `Balance`
12. `Reconcile`
13. `Health`

The descriptor declares capabilities, methods, countries, currencies, priority, and cost. Production routing excludes adapters that do not declare the complete capability set required for the requested operation.

##### Registration And Configuration

1. Add the adapter constructor in `backend/internal/payments`.
2. Register the configured adapter in `RegistryFromSettings`.
3. Keep credentials in adapter-specific environment variables supplied by the deployment secret manager.
4. Add the provider ID to `SKILL_ARENA_PAYMENT_ACTIVE_PROVIDERS`.
5. Configure the default with `SKILL_ARENA_PAYMENT_DEFAULT_PROVIDER`.
6. Configure countries, currencies, methods, priority, and cost through `SKILL_ARENA_PAYMENT_ROUTES`.
7. Verify `ValidateConfiguration("production")` fails closed for missing, sandbox, malformed, or unsafe production configuration.

Multiple adapters may be active simultaneously. Registration may add an adapter, but must not alter Payment Core selection, Wallet handlers, Treasury handlers, financial states, or frontend code.

##### Routing

Payment Core filters adapters by:

- Operation capability.
- Country.
- Currency.
- Payment method.
- Configuration status.
- Health.

Eligible adapters are ordered by:

- Explicit business preference.
- Routing priority.
- Estimated variable and fixed cost.
- Stable provider ID ordering.

Failover occurs only during health preflight, before an external financial operation is attempted. After the first attempt, the operation remains pinned to that adapter and idempotency key. Payment Core must never switch providers after an ambiguous response.

##### Callback Contract

Callbacks use `/api/v1/payments/webhooks/{providerId}`.

The adapter must:

- Verify the signature against the exact raw request body before parsing.
- Enforce timestamp or replay-window validation when the provider supports it.
- Return a stable provider event ID.
- Return a signature fingerprint without exposing the signature.
- Normalize the resource as deposit, payout, or ignored.
- Normalize status to pending, succeeded, failed, expired, or unknown.
- Return provider reference, internal resource ID, amount in minor units, and ISO currency.

Payment Core and the financial store enforce event idempotency, resource ownership, provider/reference matching, amount/currency matching, legal transitions, reserve checks, settlement, audit, and notification behavior.

##### Idempotency

Every deposit, refund, and payout requires an idempotency key. The adapter must transmit it using the provider's supported mechanism.

- Same key and same request represents the same operation.
- Same key with different request data must fail.
- Network retries reuse the same provider and key.
- A timeout or unknown response must not trigger cross-provider failover.
- Duplicate callbacks must return success without applying settlement twice.

##### Required Tests

Each adapter must include:

- Interface and descriptor contract tests.
- Configuration validation tests for development, sandbox, and production.
- Deposit session request and response tests.
- Valid, invalid, malformed, and expired signature tests.
- Pending, successful, failed, expired, and ignored callback tests.
- Duplicate callback and callback mismatch tests.
- Payment status and payout status tests.
- Refund tests.
- Payout-destination and payout tests.
- Balance and reconciliation tests.
- Idempotent retry tests.
- Provider timeout, HTTP error, malformed response, and outage tests.
- Routing eligibility and health-failover tests.
- An end-to-end sandbox deposit and withdrawal lifecycle test.

##### Production Certification

An adapter may be enabled in production only after:

- Commercial and jurisdiction approval.
- Production credentials are stored in the secret manager.
- Allowed countries, currencies, methods, costs, and limits are approved.
- Sandbox deposit, callback, settlement, refund, payout, and reconciliation pass.
- Duplicate, delayed, reordered, invalid, and replayed callbacks fail safely.
- Timeout and outage behavior is verified without duplicate money movement.
- Treasury confirms provider balances and reserves.
- Finance confirms cent-level ledger reconciliation.
- Compliance approves KYC, AML, evidence, retention, and dispute requirements.
- Security reviews credentials, signatures, TLS, callback exposure, logs, and redaction.
- Staging readiness and health checks pass.
- The adapter is enabled through configuration without frontend or Payment Core changes.

### Sprint 4: Admin CRM

Status: **COMPLETE AND FROZEN** at annotated tag `sprint-4-v1.0-freeze`.

Objective: build the operational and compliance control center as a standalone application. The player platform must contain no administrative functionality.

Visible outcome: authorized staff use a separate application, authentication surface, navigation model, permission system, API boundary, and deployment boundary to manage users, financial operations, KYC/AML, support, compliance, monitoring, notifications, and immutable audit records.

Required foundation work:

- Create a separate CRM application. Do not add CRM pages or navigation to the player Next.js application.
- Give the CRM independent authentication, authorization, navigation, API gateway, and deployment configuration.
- Replace broad role ranking with explicit permissions and separation of duties for Super Administrator, Compliance, Finance, Support, Operations, and Read Only roles.
- Enforce MFA, session timeout, device/IP logging, revocation, rate limiting, CSRF protection, and audit attribution for every administrator.
- Build an operational dashboard from real APIs for players, finance, games, support, compliance, and system health.
- Build searchable user management for profile, verification, risk, limits, devices, sessions, match history, and financial history.
- Permit lock, unlock, forced logout, MFA reset, suspension, and internal notes. Never permit direct wallet editing.
- Build financial operations for deposits, withdrawals, statements, Treasury, provider balances, reserve checks, and reconciliation through the frozen Payment Core.
- Implement the manual withdrawal review workflow with mandatory internal reasons and generic player notifications.
- Build KYC/AML review queues for evidence, provider responses, risk indicators, approval, rejection, information requests, and escalation.
- Build compliance controls for jurisdictions, assessments, limits, cooling-off, self-exclusion, and restrictions without code changes.
- Build Support CRM for assignment, replies, escalation, closure, attachments, priority, and internal-only notes.
- Build an immutable Audit Center recording administrator, timestamp, IP, device, previous value, new value, reason, and affected resource.
- Build administrative announcements and security, maintenance, and compliance notices through the existing notification service.
- Build read-only monitoring for API, PostgreSQL, Redis, object storage, providers, queues, workers, email, and alerts. Do not add restart controls.
- Keep every administrative API authenticated, permission-checked, validated, rate-limited, audited, and inaccessible through player endpoints.
- Make the CRM responsive, accessible, enterprise-focused, dark-mode capable, and entirely API-driven with no placeholder data.

#### Sprint 4 Implemented Architecture

The Admin CRM is a standalone Next.js application under `admin-crm/`. It is not a route, layout, or navigation branch inside the player frontend. Production deployment exposes the CRM separately on port `3100` and proxies versioned requests to the backend through its same-origin `/gateway` boundary.

The backend issues CRM-only access and refresh cookies with a dedicated JWT audience. Player tokens and player cookies are rejected by CRM middleware. Administrator sessions enforce mandatory MFA enrollment, explicit permission claims, idle expiry, refresh rotation, revocation, request rate limits, origin checks, and complete actor attribution.

Roles use an explicit permission matrix rather than inherited numeric ranking:

| Role | Operational scope |
|---|---|
| Super Administrator | All CRM permissions and administrator role management |
| Operations | Dashboard, users, finance read, Treasury read, KYC read, support, audit, notices, and monitoring |
| Finance / Treasury | Financial review, withdrawal decisions, Treasury, reconciliation, audit, and monitoring |
| Compliance / Fraud | Finance read, KYC decisions, compliance policies, audit, and monitoring |
| Support | User read and support case management only |
| Read Only | Read-only operational, finance, compliance, support, audit, and monitoring access |

The CRM exposes no direct wallet balance mutation. Financial decisions use the frozen Financial Platform state machines, Payment Core, reserve validation, ledger, and reconciliation contracts.

#### Sprint 4 API

All routes are versioned under `/api/v1/admin-crm`. Except for login and MFA challenge, every route requires a valid CRM access token, active administrator session, completed MFA, and the route-specific permission.

| Route | Methods | Purpose |
|---|---|---|
| `/auth/login` | `POST` | Administrator credential verification and MFA enrollment/challenge start |
| `/auth/mfa/challenge` | `POST` | Complete TOTP or recovery-code authentication |
| `/auth/mfa/setup` | `POST` | Create an encrypted TOTP enrollment secret |
| `/auth/mfa/confirm` | `POST` | Confirm TOTP and issue one-time recovery codes |
| `/auth/session` | `GET` | Return the current administrator identity and permissions |
| `/auth/refresh` | `POST` | Rotate the CRM refresh token and recover the session |
| `/auth/logout` | `POST` | Revoke the current session and clear CRM cookies |
| `/dashboard` | `GET` | Authoritative player, finance, game, support, compliance, and dependency summary |
| `/users` | `GET` | Search and filter player records |
| `/users/{id}` | `GET` | Full identity, security, progression, wallet, match, and compliance record |
| `/users/{id}/status` | `POST` | Suspend, disable, lock, or reactivate an account with a reason |
| `/users/{id}/force-logout` | `POST` | Revoke all player sessions |
| `/users/{id}/notes` | `GET`, `POST` | Read or append internal-only notes |
| `/users/{id}/restrictions` | `GET`, `POST`, `PATCH` | Read, apply, expire, or lift typed restrictions |
| `/users/{id}/role` | `POST` | Super-administrator role assignment |
| `/users/{id}/mfa/reset` | `POST` | Super-administrator MFA reset and forced logout |
| `/finance` | `GET` | Deposits, withdrawals, providers, reconciliation, and reserve evidence |
| `/finance/withdrawals/{id}/decision` | `POST` | Manual approval or rejection with mandatory reason |
| `/compliance/cases` | `GET` | KYC, AML, financial assessment, evidence, provider-response, and review queue |
| `/compliance/evidence/{id}` | `GET` | Authorized evidence retrieval from object storage |
| `/compliance/decisions` | `POST` | Approve, reject, request information, or escalate a review |
| `/compliance/jurisdictions` | `GET`, `PUT` | Runtime country, age, source-of-funds, and financial-limit policy |
| `/support/tickets` | `GET` | Search the support work queue |
| `/support/tickets/{id}` | `PATCH` | Assign, prioritize, reply, escalate, annotate, or close a ticket |
| `/support/attachments/{id}` | `GET` | Authorized support attachment retrieval |
| `/audit` | `GET` | Filter immutable administrator and platform audit records |
| `/announcements` | `GET`, `POST` | Read and send audited operational notices |
| `/monitoring` | `GET` | Read-only API, database, Redis, storage, queue, worker, email, and provider health |

#### Sprint 4 Persistence And Security

Migration `006_admin_crm.sql` adds administrator roles, internal notes, support messages and attachments, account and responsible-gaming restrictions, jurisdiction policies, immutable KYC/AML provider-response summaries, announcements, audit context fields, constraints, and operational indexes. PostgreSQL is authoritative in production. Support attachments and compliance evidence use the configured object-storage interface. Provider adapters record normalized identity, age, address, sanctions, PEP, AML, and source-of-funds outcomes through the provider-neutral repository boundary; raw sensitive documents remain in object storage.

Audit entries form a cryptographic hash chain. Hash timestamps are normalized to PostgreSQL microsecond precision so records can be reproduced after persistence. User-restriction updates verify both restriction ID and owning user ID before mutation. Cooling-off and self-exclusion require explicit expiry, block financial operations and competition entry, and remain visible in the player record. CRM mutation handlers fail visibly if their required audit entry cannot be recorded.

The CRM container runs the Next.js standalone production bundle as a non-root user. Docker Compose gives the application an independent service, port, backend dependency, and health check. No administrator screens or navigation were added to the player application.

#### Sprint 4 Proof

Responsive proof is generated by the real end-to-end administrator journey under `docs/proof/sprint-4-admin-crm/`. The suite registers and verifies configured administrator accounts, performs CRM login, enrolls MFA, stores recovery codes, loads every operational module from live APIs, signs out, and repeats at desktop, tablet, and mobile viewports.

#### Sprint 4 Production Validation

Validation date: 2026-07-25.

- Backend: `gofmt` completed; `go test ./...`, `go vet ./...`, and `go build ./...` passed.
- PostgreSQL: `TestPostgresAdminCRMRepository` passed against a fresh PostgreSQL 17 database after applying every migration. The test verifies role persistence, object-stored support evidence, jurisdiction policy, provider-response JSONB and array round trips, and audit-chain integrity.
- Admin CRM: Vitest passed 2 files and 5 tests; ESLint passed with zero warnings; TypeScript passed; the Next.js 16.2.11 production build generated all 13 routes.
- Admin end to end: Playwright passed the complete administrator login, MFA enrollment, recovery-code retention, operational-module, and logout journey on desktop, tablet, and mobile. Result: 3 passed.
- Frozen-domain regression: player Vitest passed 4 files and 6 tests; ESLint, TypeScript, and the 23-route production build passed. Playwright passed 18 Sprint 1-3 journeys across desktop, tablet, and mobile.
- Dependencies: production `npm audit` reported zero vulnerabilities for both the player platform and Admin CRM.
- Repository integrity: 483 tracked and pending project files were scanned; no zero-byte files, NUL-corrupted text, invalid UTF-8, or duplicate migration prefixes were found. Builds verified imports and model references.
- Isolation: no player admin route, navigation entry, approval control, Treasury dashboard, fraud tool, KYC review, or compliance operation exists in the player application.
- Responsive proof: 30 current screenshots are stored under `docs/proof/sprint-4-admin-crm/`.

Docker was unavailable in the validation environment. The standalone non-root image, independent Compose service, health check, and deployment boundary were reviewed statically; exercising the image in the target container platform remains deployment-environment verification, not missing application code.

Freeze decision: **SPRINT 4 APPROVED.** The separate Admin CRM, its secured APIs, PostgreSQL persistence, object-storage evidence access, operations modules, permissions, audit controls, responsive UI, and Sprint 1-3 regressions are complete. Sprint 5 has not begun.

### Sprint 5: Session Gateway, Presence, Notifications, And Realtime Events

Visible outcome: players receive authenticated live presence, match, notification, and reconnect events across the platform.

Implementation status: complete and frozen. Sprint 5 does not implement Maze rules or any other game-specific action.

#### Realtime Architecture

```text
Authenticated player
        |
        +-- REST: queue and lifecycle intents
        |
        `-- WebSocket: session negotiation, heartbeat, sync, and events
                         |
                         v
                  Realtime Arena
              / lifecycle authority
             /  matchmaking policy
            /   presence and recovery
           /    ordered event stream
          v
PostgreSQL authoritative state | Redis coordination | Object storage replays
          |
          v
Arena Core game registry and versioned capability contracts
```

- Clients submit intent only. They cannot submit match results, authoritative timestamps, state versions, queue priority, opponents, rewards, or wallet effects.
- Match states are `created`, `waiting_for_players`, `ready`, `starting`, `live`, `paused`, `reconnecting`, then `completed`, `cancelled`, or `abandoned`.
- Queue matching is game, mode, wallet category, region, jurisdiction, rating, latency, priority, and restriction aware. Priority is calculated by the server.
- Practice creates an independent match and seed reference. PvP creates one match with a shared server seed reference and independent participant state.
- Each game is accepted only through Arena Core metadata, version, mode, and capability declarations. Realtime Arena never imports Maze rules.
- PostgreSQL migration `007_realtime_arena.sql` normalizes matches, participants, queues, presence, events, snapshots, and replay manifests.
- Every event receives an atomic per-match sequence and chained HMAC integrity hash. Every state transition records a checksummed snapshot.
- Reconnect returns the current authoritative snapshot plus all events after the client's acknowledged sequence. Clients deduplicate by sequence.
- Terminal sessions enqueue durable replay persistence. Replay manifests include game, rules, protocol, replay versions, event range, root hash, signature, and object-storage key.
- Redis coordinates matchmaking locks and connection throttles. Gateways hold no authoritative match state, so clients may reconnect to another node.
- The realtime worker expires queues and presence, abandons expired reconnect windows, and persists replay artifacts. Admin CRM exposes read-only operational metrics.

#### Realtime API v1

All routes require a valid, non-revoked player session. Competition restrictions are evaluated before queue or match access.

| Method | Route | Purpose |
|---|---|---|
| `POST` | `/api/v1/realtime/queue` | Enter Practice, PvP, or Tournament matchmaking |
| `GET` | `/api/v1/realtime/queue` | Read the player's latest queue state |
| `DELETE` | `/api/v1/realtime/queue` | Cancel an active queue entry |
| `GET` | `/api/v1/realtime/matches/{matchId}` | Read an owned authoritative match snapshot |
| `POST` | `/api/v1/realtime/matches/{matchId}/ready` | Signal participant readiness |
| `POST` | `/api/v1/realtime/matches/{matchId}/leave` | Leave; the server decides cancellation, abandonment, or forfeit completion |
| `POST` | `/api/v1/realtime/matches/{matchId}/reconnect` | Recover snapshot and ordered events after a sequence |
| `POST` | `/api/v1/realtime/matches/{matchId}/heartbeat` | Refresh presence and receive server time |
| `GET` | `/api/v1/realtime/events/{matchId}?after={sequence}` | Read an owned ordered event delta |
| `GET` | `/api/v1/realtime/replays/{matchId}` | Read signed replay metadata for an owned terminal match |
| `GET` | `/api/v1/realtime/gateway` | Upgrade to the authenticated WebSocket protocol |

Gateway client messages are limited to `heartbeat`, `subscribe`, `reconnect`, `ready`, `leave`, and `ack`. Unknown messages and client-authored state are rejected. Approved origins, connection rate limits, read limits, ping/pong deadlines, write deadlines, and bounded client reconnection apply.

#### Realtime Configuration

Production configuration is environment-only:

- `SKILL_ARENA_REALTIME_QUEUE_TTL_SECONDS`
- `SKILL_ARENA_REALTIME_PRESENCE_TTL_SECONDS`
- `SKILL_ARENA_REALTIME_RECONNECT_SECONDS`
- `SKILL_ARENA_REALTIME_MAX_RATING_GAP`
- `SKILL_ARENA_REALTIME_MAX_LATENCY_MS`
- `SKILL_ARENA_REALTIME_MAX_MESSAGE_BYTES`
- `SKILL_ARENA_REALTIME_CONNECTIONS_PER_MINUTE`

#### Sprint 5 Production Validation

Validation date: 2026-07-27.

- Backend: all 16 test-bearing packages passed. `go vet ./...` and `go build ./...` passed after repository-wide `gofmt`.
- Realtime core: seven focused tests passed, covering origin rejection, duplicate active-connection rejection, 100 concurrent authenticated gateway negotiations, non-Maze lifecycle execution, independent Practice seeds, event-chain integrity, replay signing, reconnect, and 100-player concurrent matchmaking without duplicate pairing.
- PostgreSQL: `TestPostgresRealtimeRepository` passed against an isolated fresh PostgreSQL 17 cluster after applying every migration through `007_realtime_arena.sql`. Match, participant, queue, presence, event, snapshot, and metrics persistence were verified.
- Distributed coordination: Redis locks now use random ownership tokens and compare-and-delete release semantics. Non-owners cannot release a lock. Matchmaking rechecks authoritative queue state after acquiring the lock.
- Browser client: Vitest passed 5 files and 9 tests. The reusable realtime client has 68.23% statement and 70.66% line coverage. TypeScript, ESLint, and the 23-route Next.js 16.2.11 production build passed.
- Admin CRM: Vitest passed 2 files and 5 tests. Coverage measured 56.75% statements and 61.76% lines. TypeScript, ESLint, and the 13-route production build passed. Realtime monitoring remains read-only.
- End to end: the complete Sprint 1-5 Playwright regression passed 21 journeys with zero failure contexts and 21 videos. The final strengthened realtime queue, start, reconnect, and terminal-state journey passed again on desktop, tablet, and mobile.
- Performance baseline: Practice lifecycle benchmark measured `4,344,815 ns/op`, `42,474 B/op`, and 130 allocations per operation on an Intel i7-8665U. The final 100-connection gateway test completed in 3.34 seconds; the 100-player no-double-pair matchmaking test completed in 3.54 seconds.
- Security: cookie-authenticated gateway access, JWT session revocation, approved-origin enforcement, competition restrictions, input allowlists, 8 KiB configurable messages, connection throttling, single active player connection, heartbeat deadlines, server time, sequence deduplication, ownership checks, HMAC event chains, signed replay manifests, and client-state rejection were verified.
- Dependencies: production `npm audit` reported zero vulnerabilities for the Player Platform and Admin CRM.
- Repository integrity: 500 tracked and pending files were scanned. There were no zero-byte files; all 267 text files had no NUL bytes or invalid UTF-8; migration prefixes and model type declarations were unique; `git diff --check` passed.
- Evidence: current desktop, tablet, and mobile screenshots and videos are stored under `docs/proof/sprint-5-realtime-arena/`.

Docker was unavailable in the validation environment. Compose wiring, health checks, production environment variables, PostgreSQL, Redis, object storage, workers, API, Player Platform, and Admin CRM dependencies were reviewed statically. Target-orchestrator deployment and network chaos remain Release 1.0 launch verification, not missing Sprint 5 application code.

Freeze decision: **SPRINT 5 APPROVED.** Realtime Arena is frozen at `sprint-5-v1.0-freeze`. Future changes are limited to bug fixes, security fixes, performance, scalability, and integration support. Maze Arena production implementation remains Sprint 6 and has not begun.

### Sprint 6: Maze Arena

Visible outcome: Practice, Ranked, Daily Challenge, and Replay provide deterministic, satisfying, server-authoritative Maze competition.

Required foundation work:

- Make every gameplay action an intent validated by Arena Core; the frontend may not decide collisions or outcomes.
- Complete server-authoritative PvP progress, combo, moves, timing, finish, disconnect, and reconnect state.
- Verify generator, solver, validator, difficulty scorer, and replay behavior against approved reference fixtures.
- Complete replay signatures over seed, puzzle hash, generation hash, difficulty profile, rules version, ordered actions, timing, and outcome.
- Store immutable replay artifacts in object storage and support verification, playback, and disputes.
- Prove deterministic reconstruction and gameplay parity through integration and end-to-end tests.

#### Sprint 6 Phase 1: Stray Arrows Architecture Review

Status: **COMPLETE - DOCUMENTATION REVIEW ONLY**

Approval: **APPROVED**

Decision: **SPRINT 6 IMPLEMENTATION HAS NOT STARTED.**

This review compares the supplied `sample app/stray-arrows` project with the frozen Skill Arena architecture. It does not approve the reference code for production use, redesign Arena Core, or authorize Phase 2.

##### Review Scope And Evidence

The review covered:

- The standalone root and `www` game bundles.
- `index.html`, `handcrafted-levels.js`, `levels-baked.js`, and the service worker.
- The generator, baker, handcrafted validator, rendering, input, animation, sound, persistence, PWA, Android, and iOS structure.
- The current Skill Arena game registry, Maze module, Puzzle Service, Realtime Arena, replay models, frontend Maze preview, and README product contract.
- The frozen Sprint 1 through Sprint 5 ownership boundaries.

Verified evidence:

- The reference contains 141 files including generated native wrappers and assets.
- Root and `www` copies of the six shipped web artifacts are byte-identical.
- All five handcrafted tutorial boards pass geometry and solvability validation.
- The baked catalogue contains 500 distinct serialized boards.
- The baked catalogue contains no overlapping cells, invalid directions, or non-contiguous arrow paths under the supplied rules.
- Eight baked boards are unsolvable: levels 18, 20, 21, 22, 23, 298, 407, and 415.
- The eight deadlocked boards leave between 15 and 29 arrows stuck.
- The baker knowingly falls back to its first candidate when all candidates fail, even when that candidate is deadlocked.
- The current reference has no authoritative server, action journal, replay contract, integrity signature, or dispute evidence.

##### Executive Conclusion

The supplied game is a strong **gameplay and presentation reference**, but it is not a production Maze module and must not be copied into Skill Arena.

The most valuable reusable ideas are:

- Cell-based arrow geometry.
- Directional escape-ray collision rules.
- Monotonic dependency clearing.
- Seeded procedural generation concepts.
- Handcrafted teaching boards.
- Camera, zoom, pan, hit testing, rendering, sound, haptics, and motion concepts.

The following cannot be reused as production authority:

- Client-side generation and validation.
- Public and predictable seed derivation.
- Fixed baked levels.
- Local timers, scores, lives, rewards, coins, progression, and completion.
- `localStorage` persistence.
- Client-owned daily and weekly challenge dates.
- Standalone ads, achievements, menus, PWA ownership, and native wrappers.

The current Skill Arena Maze implementation is also not yet reference-compatible. It has useful server-side foundations, but it combines two different games:

- A legacy grid maze with a moving start/end player path.
- An arrow-line puzzle using routed points and dependency metadata.

Sprint 6 must establish one approved Maze Arena ruleset. The supplied reference and the current product direction both identify that ruleset as the arrow escape game.

##### Frozen Ownership Boundary

Realtime Arena remains responsible for:

- Authentication and authorization.
- Matchmaking and queue ownership.
- Match lifecycle.
- Presence and connection ownership.
- WebSocket transport.
- Heartbeats, disconnect, reconnect, and ordered synchronization.
- Generic match events and snapshots.
- Generic replay persistence and signing infrastructure.
- Server time, sequence enforcement, security, monitoring, and scaling.

Maze Arena is responsible only for:

- Arrow-puzzle generation.
- Puzzle solving and validation.
- Collision and move rules.
- Maze-specific authoritative board state.
- Difficulty analysis and scoring inputs.
- Progress and completion calculation.
- Maze-specific action and replay payloads.
- Renderer data and Maze presentation.

Maze Arena must not create its own authentication, matchmaking, WebSocket, presence, session, reconnect, generic replay storage, wallet, reward, or tournament infrastructure.

##### Actual Reference Rules

The supplied reference implements these mechanics:

1. An arrow is an ordered list of occupied grid cells from tail to head.
2. Each arrow has one immutable direction: right, up, left, or down.
3. A tap selects an arrow by proximity to any cell in its body.
4. The escape ray starts one cell beyond the arrow head and continues in the arrow direction to the board edge.
5. Any occupied cell belonging to another live arrow blocks that ray.
6. A clear arrow is removed from logical state immediately and its exit animation begins.
7. A blocked arrow remains in logical state.
8. Removing an arrow can make other arrows clear because all collision checks use the current live-arrow set.
9. Completion occurs only when no live arrows remain.
10. The reference does not automatically remove newly unblocked arrows. The player must tap each one.

This is collision-derived dependency logic. A separate declared `DependsOn` list is not the rule; dependencies are a consequence of geometry and current occupancy.

The reference's current blocked feedback is a short horizontal shake and sound. It does **not** slide the arrow to the blocker and return. The desired Skill Arena blocked-move animation therefore exceeds the supplied reference and must be specified separately in the implementation contract.

The successful animation clips the rendered arrow along its existing polyline and then extends it beyond the head. Logical removal happens before the 1.2 second animation completes. Skill Arena may retain the visual concept, but server state and client presentation state must remain explicitly separate.

##### KEEP

The following should remain as reference behavior or design input:

| Area | Keep | Reason |
|---|---|---|
| Core rule | An arrow can leave only when its directional escape ray is clear | This is the actual puzzle |
| Direction | Fixed direction per arrow | Prevents rotation or client-selected movement |
| Geometry | Ordered tail-to-head orthogonal cells | Deterministic and readable |
| Dependencies | Recalculate from live geometry after every accepted action | Creates the puzzle chain |
| Tutorial | Five small handcrafted teaching boards | All five validate and teach the rule progressively |
| Generator concept | Seeded generation followed by a solver | Suitable starting principle |
| Pattern bias | Braid, spiral, mosaic, piton, diagonal, rings, maze, and rays as generator inputs | Adds visual variety without changing rules |
| Camera | Pinch zoom, wheel zoom, transform inversion, and bounded pan | Mature interaction foundation |
| Input | Tap-versus-pan threshold and world-coordinate hit testing | Appropriate for touch play |
| Rendering | Continuous rounded paths, proportional arrowheads, cached animation paths | Better than debug line graphics |
| Feedback | Exit animation, blocked feedback, sound, haptics, reduced motion | Teaches outcomes without modal interruptions |
| Performance ideas | Delta-time animation, cached background, limited hot-path allocation | Useful renderer guidance |
| Mobile lessons | Safe-area handling and crisp canvas transforms | Useful for responsive implementation |

KEEP means preserve the behavior or lesson, not copy the standalone source into production.

##### MODIFY

| Reference area | Required Skill Arena change |
|---|---|
| Generator | Port into a deterministic, versioned, server-only Maze generator |
| Randomness | Replace public Park-Miller seeds and level-number formulas with server-derived seed material |
| Puzzle pipeline | Enforce Generate -> Solve -> Validate -> Score -> Hash -> Persist metadata -> Deliver |
| Retry behavior | Reject every failed candidate; never return an unsolvable fallback |
| Solver | Return an independently verified solution and dependency analysis |
| Validator | Validate geometry, occupancy, direction/head alignment, collisions, solvability, metadata, and hashes |
| Difficulty | Accept an authoritative Difficulty Profile instead of deriving gameplay from a fixed level number |
| Collision | Use one canonical server implementation shared by generation validation, live action validation, solver, and replay |
| State | Store independent authoritative player board state for each participant |
| Actions | Accept one ordered intent at a time with match, target, sequence, and server receipt time |
| Timing | Use server match time; client time is telemetry only |
| Progress | Derive cleared count, total count, combo, blocked actions, and completion on the server |
| Scoring | Derive score from approved rules and authoritative events |
| Replay | Record seed reference, versions, profile, ordered actions, server timing, results, and integrity data |
| Rendering | Convert game state into a versioned renderer payload; do not expose mutable authority |
| Correct animation | Keep the arrow visible until its presentation has completely exited |
| Blocked animation | Add travel toward the nearest blocker, impact, recoil, and return without changing authoritative state |
| Accessibility | Add semantic controls or an equivalent accessible interaction model around the canvas |
| Tests | Extract mechanics from the monolithic page into deterministic fixtures and cross-language parity tests |

##### REMOVE

The following must not enter the Skill Arena Maze module:

- The 500-board baked production catalogue.
- Fixed post-tutorial levels as puzzle identities.
- The public `level * 7919 + 31337` seed formula.
- Client-side daily and weekly seed calculation from `Date.now()`.
- Local best times, stars, hearts, coins, rewards, achievements, and progression authority.
- Hint, continue, and skip purchases tied to standalone local coins.
- AdMob reward and advertising logic.
- `localStorage` as game, reward, challenge, or replay authority.
- URL controls that alter production puzzle generation or unlock progression.
- Service-worker ownership inside the Maze module.
- Standalone menu, settings, statistics, update prompt, store listing, PWA, Android shell, and iOS shell.
- Standalone app wallet/economy assumptions.
- The legacy grid-walking maze as the production Maze Arena ruleset.
- Any client API that submits score, completion, winner, seed, difficulty, reward, or board state.

The reference project remains in `sample app/stray-arrows` as a review fixture. It is not a production dependency.

##### MOVE Or Retain In Platform Services

| Concern found in reference | Skill Arena owner |
|---|---|
| Login identity | Sprint 1 Identity and Security |
| Player profile and progression | Sprint 2 Arena Hub |
| Coins, stakes, deposits, withdrawals, and rewards | Sprint 3 Financial Platform |
| Review, fraud, support, and disputes | Sprint 4 Admin CRM |
| Queue, match, presence, network, reconnect, and session clock | Sprint 5 Realtime Arena |
| Generic replay storage, signing, and delivery | Realtime Arena and object storage |
| Tournament scheduling and prize settlement | Competition Platform in Sprint 7 |
| Notifications | Arena Hub notification service |
| Game-specific puzzle and rules | Sprint 6 Maze module |
| Game-specific rendering | Sprint 6 Maze renderer |

Nothing from the standalone game should replace a frozen platform service.

##### Current Skill Arena Compatibility Findings

| Area | Current state | Phase 1 conclusion |
|---|---|---|
| Game registry | `maze_arena` and `test_arena` implement the shared module contract | KEEP |
| Manifest and versions | Maze declares game, rules, replay, protocol, modes, and capabilities | KEEP and extend only through version bumps |
| Puzzle seed derivation | HMAC derivation includes purpose, match, player/shared identity, nonce, profile, and versions | KEEP |
| Practice uniqueness | Non-shared generation includes player identity and a random nonce | Correct foundation |
| PvP fairness | Existing legacy PvP generation assigns the same seed, hash, and cloned board to both players | Correct foundation |
| Tournament fairness | Existing tournament generation assigns one shared puzzle per match | Correct foundation |
| Generation locking | Practice, initial PvP, and initial bracket generation release the store lock before CPU generation | Correct foundation |
| Next tournament round | `advanceTournamentLocked` performs puzzle generation while holding the global store lock | INCORRECT; must be corrected as a generic performance defect before that path is used |
| Puzzle repository | Dedicated Puzzle Service defaults to an in-memory repository | Not production-ready for Sprint 6 |
| Pipeline separation | Generator also produces a solution; validator replays it; difficulty score is profile input rather than measured output | MODIFY |
| Generator fallback | Current Skill Arena generator can silently return an escape-only fallback after failed candidates | INCORRECT for authoritative difficulty |
| Collision model | Server has geometric ray intersection, but generation also creates declared dependency metadata | Requires one canonical rule source |
| Maze module | Supports both line clicks and legacy grid moves and accepts action batches | Must be narrowed to the approved arrow game |
| Action result | `core.ActionResult` exposes Maze-specific moves, lines, and clicks | Existing game-agnostic boundary leak |
| Session model | Generic `GameSession` contains Maze cells, start/end coordinates, lines, and clicks | Existing game-agnostic boundary leak |
| Realtime gateway | Supports lifecycle, ready, leave, subscribe, reconnect, heartbeat, event polling, and sequence acknowledgement | KEEP |
| Realtime actions | Gateway has no generic game-action message or module dispatch path | Missing integration required by every interactive game |
| Replay infrastructure | Signed generic realtime event artifacts exist | KEEP |
| Maze replay | Current Maze replay can persist full board data and uses client action timestamps in places | Does not satisfy seed-and-actions-only reconstruction |
| Frontend | Maze previews are imported directly by landing, guest arena, and game pages | Maze presentation currently leaks outside a module boundary |

The generic action-envelope gap benefits every future realtime game. If Sprint 6 needs an additive gateway extension for authenticated, sequenced game intent, that may qualify as frozen-platform integration support. It must remain game-agnostic, backward-compatible, and independently tested. Maze-specific message fields must not be added to the gateway contract.

Maze-specific fields already present in generic models are recorded as architectural debt. Phase 1 does not authorize removing or breaking those frozen contracts. Phase 2 must determine whether Maze can use them safely or whether a backward-compatible generic state/event envelope is required.

##### Puzzle Generation And Uniqueness Contract

The production contract is:

```text
Server seed source
  -> Generate candidate
  -> Solve independently
  -> Validate geometry and rules
  -> Analyze actual difficulty
  -> Calculate puzzle and generation hashes
  -> Enforce uniqueness policy
  -> Persist immutable metadata
  -> Assign to match/session
  -> Deliver renderer state
```

Mode rules:

| Mode | Seed and puzzle rule |
|---|---|
| Tutorial | Approved handcrafted boards 1 through 5 may be reused as teaching fixtures and must be versioned |
| Practice | Unique seed per player session; no intentional board reuse |
| Training | Unique seed per training session unless an explicit lesson fixture is approved |
| PvP or Ranked | One server-generated seed and identical puzzle metadata for both opponents; independent board states |
| Daily Challenge | One approved shared seed for the challenge period |
| Tournament | One shared seed per match; different matches receive different seeds |
| Replay | Regenerate from immutable versioned seed metadata and replay authoritative actions |

Uniqueness requires a database-enforced identity, not a probabilistic promise. At minimum, the generation hash or puzzle hash must be unique for modes that prohibit reuse. A seed collision, generation-hash collision, duplicate board hash, failed validation, or unavailable approved candidate must fail closed and request another candidate.

The backend may deliver identical renderer state to both PvP players, but each participant must receive a separate mutable state record. One player's accepted action must never mutate the opponent's board.

##### Deterministic Replay Assessment

Reference game result: **NOT REPLAY READY**

The reference could become deterministic at the pure mechanic level because its seeded generator and cell collision rules are deterministic for a fixed runtime implementation. It cannot currently produce a verifiable replay because it does not record:

- Generator version.
- Rules version.
- Difficulty profile.
- Puzzle hash.
- Generation hash.
- Ordered action IDs.
- Server sequence numbers.
- Server receipt times.
- Accepted and rejected outcomes.
- Match clock state.
- Completion and score inputs.
- Integrity signature.

The production replay should store a compact authoritative event stream, not trust a saved client board. It must retain enough immutable version information to run the historical generator and rules implementation years later.

Each action event must distinguish:

- Intent received.
- Accepted escape.
- Rejected blocked action.
- Rejected duplicate or unknown target.
- Rejected stale or out-of-order sequence.
- Rejected action after finish, timeout, leave, or forfeit.

Renderer-only animation events may be derived from authoritative outcomes. They must not be used to decide game state.

##### Server Authority

The server must own:

- Seed selection and uniqueness.
- Difficulty profile.
- Generator, solver, validator, and difficulty analysis versions.
- Puzzle identity, generation hash, and puzzle hash.
- Participant board state.
- Arrow occupancy and removal state.
- Collision and nearest-blocker calculation.
- Action ordering and deduplication.
- Accepted and rejected move outcomes.
- Match timer, timeout, disconnect, reconnect, and forfeit.
- Combo, progress, completion, score, winner, draw, and result.
- Replay events, integrity signature, storage, and dispute evidence.
- Eligibility, rewards, progression, and settlement events.

The client must send only intent such as:

```json
{
  "type": "game.action",
  "matchId": "mat_...",
  "sequence": 17,
  "action": {
    "type": "arrow.click",
    "targetId": "arrow_23"
  }
}
```

This is a conceptual Phase 1 example, not an approved API contract.

##### Client Responsibilities

The client may own:

- Rendering the server-provided immutable puzzle geometry.
- Camera, zoom, pan, resize, and coordinate transforms.
- Hit testing that resolves a pointer position to an arrow intent.
- Optimistic press feedback that cannot commit game state.
- Animating server-accepted escapes.
- Animating server-rejected blocked impacts and returns.
- Sound, haptics, reduced-motion presentation, and accessibility.
- Displaying server-owned time, progress, combo, opponent progress, and result.
- Requesting sync after a sequence gap or reconnect.

The client must roll back or resynchronize presentation when an optimistic interaction disagrees with the authoritative response.

##### Security And Exploit Review

The supplied reference is vulnerable by design as an offline casual game:

| Exploit | Reference exposure | Required production control |
|---|---|---|
| State editing | All arrows and state live in browser memory | Server-owned participant state |
| Save editing | Progress, coins, lives, stars, and results use `localStorage` | Authenticated durable repositories |
| Timer editing | Timer increments from client animation delta | Server match clock |
| Score/reward editing | Completion directly credits local coins and achievements | Platform event settlement |
| Seed prediction | Public level formula and Park-Miller RNG | Secret server derivation plus random nonce |
| Fixed-solution lookup | 500 baked boards ship to every client | Unique server-generated live puzzles |
| Daily challenge manipulation | Client date controls seed and completion key | Server challenge schedule |
| Action fabrication | No authenticated action stream | JWT session, participant check, sequence, idempotency |
| Packet replay | No nonce or action deduplication | Match-scoped monotonic sequence and dedupe |
| Replay editing | No signed replay exists | Signed immutable authoritative events |
| Completion fabrication | Client marks arrows dead and decides completion | Server validator and finish state |
| Bot solving | Full board and deterministic rules are locally inspectable | Behavioral telemetry, rate plausibility, replay review; never rely on obscurity |
| Memory automation | Public `_ae` helpers expose generation and board state | Production bundle exposes renderer data only |
| Difficulty override | URL flags alter level and fresh/dot modes | Server-selected profile; reject client overrides |
| Offline stale build | Service worker caches game assets | Platform-controlled version negotiation |
| Multi-action speedup | Current Skill Arena Maze module accepts batches | One ordered action intent or strictly bounded protocol |
| Client timestamps | Current Maze paths preserve client action times | Treat as untrusted telemetry only |

No anti-cheat design can prevent a player from seeing the puzzle that must be rendered. Competitive integrity must come from server authority, unique puzzles, authenticated ordering, timing analysis, deterministic replay, anomaly detection, and review evidence.

##### Rendering, Camera, Animation, And Mobile Findings

The reference provides useful quality targets:

- A fixed logical 390 by 844 canvas with device-pixel-aware presentation.
- Grid-centered zoom from 1.0 to 2.5.
- Pinch zoom and pan with inverse hit testing.
- One-finger pan only after zoom and movement threshold.
- Rounded arrow shafts and joins.
- Directional arrowheads derived from the final path segment.
- Staggered board reveal.
- Exit easing, sound, haptics, confetti, and reduced-motion handling.
- Safe-area-aware top controls.

Required changes:

- The Skill Arena renderer must be a Maze module, not a whole application.
- Layout must respond to platform navigation and match HUD rather than assume a portrait standalone canvas.
- Desktop mouse pan should be supported intentionally, not only wheel zoom.
- Canvas interaction needs an accessible alternative and keyboard/focus contract.
- Arrow colors must remain legible and must not encode hidden authoritative information.
- Blocked animation must use the server-provided blocker/collision result.
- Correct-move presentation must wait for server acceptance.
- PvP opponent state must expose only approved progress, not hidden board actions during the live match.

##### Difficulty Assessment

The reference's difficulty is mainly a function of level number, board dimensions, path length, cluster count, pattern bias, arrow count, and at most one blocker during generation. This is not sufficient as Skill Arena's authoritative Difficulty Profile.

The production analyzer must measure actual output, including:

- Arrow count and occupied-cell count.
- Board width, height, density, and readable scale.
- Dependency edge count.
- Dependency depth.
- Branching and number of simultaneously open choices.
- Cross-dependency count.
- Longest dependency chain.
- Path-length distribution.
- Direction distribution.
- Visual overlap and routing complexity.
- Expected solve-time percentiles.
- Minimum required successful actions.
- Difficulty confidence and rejection reason.

Difficulty must come from player progression, league, season, mode, practice profile, or tournament rules. It must never be reduced because of server load, timeout avoidance, or lock contention.

The current reference baker's scoring is useful research, but the output proves that candidate scoring cannot substitute for a hard validator.

##### Phase 1 Classification Summary

| Review area | Classification | Reason |
|---|---|---|
| Gameplay concept | KEEP | Strong deterministic monotonic puzzle rule |
| Cell geometry | KEEP | Clear canonical collision representation |
| Handcrafted tutorial 1-5 | KEEP | Validated teaching fixtures |
| Baked level catalogue | REMOVE | Repeated public content and eight deadlocks |
| Procedural generator | MODIFY | Useful heuristics, wrong runtime ownership and failure policy |
| Solver | MODIFY | Useful monotonic solver, must be independent/versioned/authoritative |
| Difficulty | MODIFY | Level-driven rather than measured profile output |
| Collision validation | MODIFY | Correct concept, must become one server canonical implementation |
| Camera and input | KEEP/MODIFY | Strong interaction work, must fit platform renderer/accessibility |
| Rendering | KEEP/MODIFY | Good concepts, not reusable as a monolithic app |
| Successful animation | KEEP/MODIFY | Good feedback; presentation must follow server acceptance |
| Blocked animation | MODIFY | Reference shake is below approved impact/return behavior |
| Local state and economy | REMOVE | Direct integrity violation |
| Replay | MISSING | No action journal or integrity model |
| Server authority | MISSING in reference | Everything is client-owned |
| Current Arena registry | KEEP | Proven modular foundation |
| Current Realtime lifecycle | KEEP | Correct frozen owner |
| Generic realtime action dispatch | MISSING | Required generic integration point |
| Current Maze rules parity | INCORRECT | Legacy grid maze plus non-equivalent dependency-line model |
| Production Puzzle repository | MISSING | Memory-only dedicated repository |

##### Phase 1 Decision And Stop Gate

Phase 1 is complete as an engineering review.

Sprint 6 remains **NOT STARTED** because no production implementation was authorized.

Phase 2 may design the Maze module only after this report is approved. Phase 2 must resolve:

1. The canonical cell-based arrow geometry and collision contract.
2. The module-owned authoritative state envelope.
3. The generic, backward-compatible Realtime Arena action envelope.
4. The production puzzle repository and uniqueness constraints.
5. The exact versioned generator/solver/validator/analyzer boundaries.
6. The renderer payload and hidden-information policy.
7. Replay reconstruction and historical version retention.
8. Migration or retirement of the legacy grid maze and current line prototype.

No React, CSS, gameplay, API, database migration, or backend implementation was added during Phase 1.

#### Sprint 6 Phase 2: Maze Arena Architecture And Games Platform

Status: **APPROVED**

Implementation status: **SPRINT 6 NOT STARTED**

Phase 1 is approved. Phase 2 defines the Games Platform and Maze Arena plugin architecture only. It does not authorize gameplay implementation, database migrations, API changes, frontend work, or modifications to frozen Sprint 1 through Sprint 5 code.

##### Architectural Decision

Skill Arena has two distinct layers:

```text
Frozen Platform Services
  Identity + Arena Hub + Financial Platform + Admin CRM + Realtime Arena
                                  |
                                  v
                         Games Platform Registry
                                  |
                 +----------------+----------------+
                 |                |                |
                 v                v                v
             Maze Arena       Future Game     Future Game
```

The frozen platform owns players, money, operations, transport, sessions, matchmaking, presence, event storage, and infrastructure.

The Games Platform owns the contract through which a registered game receives authoritative contexts and returns deterministic game transitions.

Each game owns only its rules, state, actions, outcomes, replay codec, and renderer data.

Maze Arena is Game 1. It is not a special case in Realtime Arena.

##### Design Principles

1. Realtime Arena knows a match's `gameId`, not the game's rules.
2. Realtime Arena resolves a game through the registry.
3. A game receives authenticated, server-built context and generic action envelopes.
4. A game returns deterministic state transitions and domain events.
5. A game never calls wallet, payment, progression, tournament, trust, notification, or admin repositories directly.
6. Platform services consume emitted events and apply platform policy.
7. Client state is never authoritative.
8. Every stored match identifies the exact game, rules, protocol, replay, generator, solver, validator, difficulty, canonical encoding, and renderer versions used.
9. Versioned historical behavior is immutable.
10. A future game must be addable without Maze-specific or game-specific changes to Realtime Arena.

##### Target Folder Structure

The logical production structure is:

```text
backend/
  internal/
    arena/                         # Frozen Sprint 5 platform contracts
      core/
      events/
      registry/
      sdk/
      security/
    realtime/                      # Frozen generic transport and lifecycle
    games/                         # Games Platform
      registry/
        catalog.go                 # Game factories and version resolution
        bootstrap.go               # Composition-root registration
        compatibility.go           # Version compatibility checks
      interfaces/
        module.go                  # Runtime game contract
        context.go                 # Match/action/viewer contexts
        action.go                  # Generic action/result envelopes
        state.go                   # Opaque game state and transitions
        snapshot.go                # Player/spectator snapshot contracts
        replay.go                  # Game replay codec contract
        renderer.go                # Renderer payload contract
        versions.go                # Complete version tuple
      shared/
        canonical.go               # Canonical serialization and hashing
        errors.go                  # Stable generic game errors
        testkit/                   # Contract tests for every module
      maze/
        module.json
        module.go                  # Registry adapter only
        domain/
          arrow.go
          board.go
          state.go
          action.go
          outcome.go
        puzzle/
          service.go
          generator.go
          solver.go
          validator.go
          analyzer.go
          profile.go
          seed.go
          hashes.go
          repository.go
          tutorial.go
        replay/
          codec.go
          verifier.go
        renderer/
          payload.go
        fixtures/
        tests/

frontend/
  app/
    games/
      registry/
      interfaces/
      shared/
      maze/
        renderer/
        camera/
        animation/
        audio/
        accessibility/
```

This is a target design, not an instruction to move frozen files during Phase 2.

`backend/internal/arena` remains the frozen platform boundary. `backend/internal/games/interfaces` will be an additive game-runtime contract with a compatibility adapter to the existing `arena/core.GameModule`; it must not become a competing transport or session system.

`shared` is limited to game-neutral codecs, errors, and contract-test utilities. Maze geometry, puzzle algorithms, collision rules, and scoring must remain under `games/maze`.

Go modules are registered explicitly at the application composition root. Skill Arena will not use unsafe runtime-loaded Go plugins or scan arbitrary executable code from the filesystem. "Loaded through the registry" means Realtime Arena resolves a registered factory by `gameId` and version instead of importing or switching on Maze.

##### Registry Architecture

The registry stores factories and immutable descriptors, not active match state.

Each registration declares:

- Game ID.
- Game version.
- Rules version.
- Protocol version.
- Replay version.
- Renderer schema version.
- Supported modes.
- Capability flags.
- Minimum and maximum players.
- Compatibility tuple.
- Module factory.
- Manifest hash.

Registry behavior:

1. Application bootstrap registers approved modules.
2. Registry validates every manifest and version tuple.
3. Duplicate game ID plus game version registrations fail startup.
4. A match pins the exact registered version when it is created.
5. Reconnect and replay resolve the pinned version, never `latest`.
6. A retired version may reject new matches while remaining available for historical replay.
7. Missing historical versions place a replay under integrity review; the system must not silently substitute another version.

Realtime Arena receives the registry as a dependency. It may call `Resolve(gameId, gameVersion)` but must not import `games/maze`.

Adding Sudoku, Chess, Memory, or another game requires:

- A new module folder.
- A valid manifest.
- An approved registry registration.
- Passing the generic module contract suite.
- No Maze or Realtime Arena changes.

##### Generic Runtime Game Interface

The existing frozen `arena/core.GameModule` remains supported. Phase 3 may add an adapter to the following conceptual runtime contract without removing or changing existing public methods:

```text
RuntimeGame
  Descriptor()
  InitializeMatch(context, match request) -> initial match state
  InitializeParticipant(context, initial match state) -> participant state
  GenerateState(context, generation request) -> generated state reference
  ValidateAction(context, participant state, action) -> validation result
  ApplyAction(context, participant state, validated action) -> transition
  Snapshot(context, state, viewer) -> renderer snapshot
  Completion(context, state) -> completion result
  DetermineWinner(context, match states) -> outcome
  SerializeReplay(context, replay source) -> game replay metadata
  RestoreReplay(context, replay metadata, events) -> reconstructed state
  Cleanup(context, match reference) -> cleanup instructions
```

The interface uses opaque, versioned game payloads. Generic platform packages must not add `MazeCells`, `ArrowLine`, `ArrowClick`, or other game-specific fields.

Required generic data types:

| Contract | Purpose |
|---|---|
| `MatchContext` | Authenticated match identity, mode, participants, versions, server clock, region, and approved configuration |
| `ParticipantContext` | Authenticated participant, trust/eligibility summary, reconnect state, and viewer role |
| `ActionEnvelope` | Action ID, match ID, kind, opaque payload, client sequence, expected state version, and telemetry time |
| `ActionContext` | Server-derived actor, participant, server receipt time, latency, current sequence, and current state version |
| `GameState` | Opaque versioned authoritative state owned by the game module |
| `Transition` | Accepted/rejected result, next state, events, progress, completion, and snapshot invalidation |
| `ViewerContext` | Player, opponent, spectator, replay viewer, support reviewer, or integrity verifier |
| `RendererSnapshot` | Versioned client-safe presentation data |
| `CompletionResult` | Complete, incomplete, timeout, forfeit, invalid, or review state |
| `MatchOutcome` | Winner, loser, draw, cancelled, invalidated, or unresolved |

Interface rules:

- `ValidateAction` must not mutate state.
- `ApplyAction` receives only a validated action and returns a new state/version.
- A rejected action does not advance game state but is still journaled.
- `Snapshot` must enforce viewer visibility and may not expose private opponent state.
- `DetermineWinner` uses only authoritative participant states and server time.
- `Cleanup` returns instructions; games do not delete platform records directly.
- Game methods must honor context cancellation and bounded deadlines.
- Methods must be deterministic for the same state, action, versions, and authoritative time inputs.

##### Maze Plugin Responsibilities

Maze Arena implements:

- Cell-based arrow-board generation.
- Maze seed interpretation.
- Maze Difficulty Profile interpretation.
- Geometry validation.
- Solver and measured difficulty analysis.
- Arrow-click validation.
- Collision and nearest-blocker calculation.
- Participant board transitions.
- Maze progress and combo calculation.
- Maze completion and Maze scoring inputs.
- Maze-specific replay event encoding and verification.
- Maze renderer snapshots.

Maze Arena does not implement:

- Authentication.
- Matchmaking.
- Queueing.
- WebSocket negotiation.
- Connection ownership.
- Presence.
- Heartbeats.
- Reconnect windows.
- Generic event persistence.
- Generic snapshots or replay object storage.
- Wallet locking or settlement.
- Rewards, XP, trust, houses, leaderboards, seasons, or tournaments.
- Admin review screens.

Maze emits events such as `maze.action.accepted`, `maze.action.blocked`, `maze.completed`, and `maze.score.inputs.ready`. Platform consumers decide progression, rewards, notifications, ranking, and settlement.

##### Production Puzzle Service

The Puzzle Service is a Maze-owned domain service behind game-neutral storage and hashing primitives.

Pipeline:

```text
Generation request
  -> Resolve immutable version tuple
  -> Create cryptographically secure seed
  -> Claim generation nonce
  -> Generate candidate outside global locks
  -> Solve independently
  -> Validate geometry and solution
  -> Analyze measured difficulty
  -> Compare measured output with requested profile
  -> Canonically serialize hash inputs
  -> Calculate puzzle, generation, and validation hashes
  -> Claim uniqueness
  -> Persist immutable metadata
  -> Assign to match/session
  -> Return renderer-safe generated state
```

Required inputs:

- Game ID and version.
- Rules version.
- Mode and purpose.
- Match or session identity.
- Participant identity only when the mode is not shared.
- Requested Difficulty Profile.
- Generator, solver, validator, analyzer, and canonical encoding versions.
- Server-generated cryptographic seed or seed reference.
- Server-generated nonce.

Required outputs:

- Puzzle ID.
- Puzzle hash.
- Generation hash.
- Validation hash.
- Seed reference.
- Requested Difficulty Profile ID.
- Measured Difficulty Analysis ID.
- Minimum successful actions.
- Expected solve-time distribution.
- Version tuple.
- Validation result.
- Renderer state.

Generation, solving, validation, analysis, hashing, and replay reconstruction are CPU work. They must execute outside global store/database locks. Database transactions should cover only reservations, uniqueness claims, assignments, and state updates.

The service fails closed:

- An unsolvable candidate is rejected.
- A difficulty mismatch is rejected.
- A duplicate puzzle hash in a one-use mode is rejected.
- A geometry or version mismatch is rejected.
- A timeout does not reduce complexity or substitute an easier fallback.
- An unavailable generator version does not fall back to `latest`.
- The escape-only fallback in the current prototype is not permitted in production.

##### Generator Architecture

Maze generation uses integer grid coordinates and immutable ordered arrow geometry.

Canonical arrow:

```text
Arrow
  ID
  Ordered cells: tail -> head
  Direction: right | up | left | down
```

Generator stages:

1. Resolve requested board bounds and Difficulty Profile.
2. Create pattern and cluster inputs from the profile.
3. Construct candidate arrows in reverse dependency-safe order or through another approved deterministic strategy.
4. Enforce cell uniqueness and orthogonal continuity.
5. Enforce final-segment and arrow-direction alignment.
6. Derive collision dependencies from geometry.
7. Solve the candidate independently.
8. Analyze actual difficulty.
9. Accept only when all hard constraints and tolerance bands pass.

Pattern bias may include braid, spiral, mosaic, piton, diagonal, rings, maze rows, rays, or future versioned patterns. A pattern changes generation preference, not collision rules.

The production generator does not expose:

- A permanent post-tutorial level catalogue.
- Public seed formulas.
- Client-selectable generation flags.
- A client-selected difficulty.
- A fallback board that bypasses validation.

##### Tutorial And Progression Rule

Tutorial boards 1 through 5 may remain handcrafted because they are teaching fixtures, not competitive live puzzles.

Tutorial requirements:

- Each board has an immutable fixture ID.
- Geometry, rules, expected action sequence, and renderer version are versioned.
- Fixtures pass the same validator used for generated puzzles.
- Tutorial results do not masquerade as unique competitive puzzles.

After the five tutorial fixtures, every Practice, House, Ranked, PvP, and Tournament board is generated.

The frontend may display a progression label such as "Level 15." Internally, that label maps to a Difficulty Profile request. There is no canonical production `Level 15` board.

##### Permanent Puzzle Uniqueness Rules

Puzzle uniqueness is enforced by persisted claims and canonical hashes.

| Mode | Generation and reuse policy |
|---|---|
| Tutorial | Approved fixture reuse is allowed |
| Practice | Fresh puzzle for every participant session |
| Training | Fresh unless the product explicitly selects a versioned lesson fixture |
| House Challenge | Fresh puzzle for every challenge attempt |
| PvP/Ranked | One fresh puzzle for the match; identical initial puzzle for both participants |
| Tournament | One fresh puzzle per tournament match; identical initial puzzle for that match's participants |
| Daily Challenge | One versioned shared puzzle for the approved daily challenge window |
| Replay | Reconstruct the existing puzzle; never create a new uniqueness claim |

PvP flow:

```text
Match created
  -> One puzzle generated and validated
  -> One puzzle assignment linked to the match
  -> Player A gets participant state A
  -> Player B gets participant state B
  -> Both states reference identical immutable puzzle metadata
  -> Actions mutate only the actor's participant state
```

One-use means the puzzle hash cannot be assigned to a second one-use match or session. A repeated seed is also rejected for one-use generation. Tutorial and shared scheduled challenges use explicit reuse policies and cannot accidentally enter one-use queues.

Puzzle uniqueness is not delegated to random probability. It is enforced by a database uniqueness claim over the canonical puzzle hash and by mode-aware assignment constraints.

Seed policy:

- Generate seed material with a cryptographically secure random source.
- Keep active-match seed material server-side or encrypted at rest.
- Send renderer geometry, puzzle identity, and required public metadata to clients.
- Do not expose a seed during a live match if it enables local solution generation.
- Include the replay seed or approved seed reference after the match according to replay visibility policy.
- Store a seed hash for uniqueness and auditing.

##### Difficulty Profile Architecture

Difficulty has two records:

1. **Requested Difficulty Profile**: the authoritative generation target.
2. **Measured Difficulty Analysis**: what the generated board actually contains.

They must never be treated as the same value.

Requested Difficulty Profile:

| Field | Meaning |
|---|---|
| Profile schema version | Immutable interpretation of all fields |
| Complexity score | Unbounded progression input |
| Rating band | Player-facing or matchmaking calibration band |
| Line count | Target arrow-count range |
| Dependency depth | Target longest dependency-chain range |
| Branching | Target available-choice and dependency-branch ranges |
| False routes | Target tempting-but-currently-blocked choice ratio; never fake graphics or false rules |
| Density | Target occupied-cell ratio |
| Pattern bias | Weighted versioned pattern preferences |
| Expected solve time | Target percentile distribution |
| Visual complexity | Readability constraints, route length, turns, crowding, and minimum on-screen scale |
| Direction balance | Allowed directional distribution |
| Path-length profile | Short, medium, and long-arrow target ranges |
| Source | Practice, league, season, house, ranked, or tournament policy |

Measured Difficulty Analysis:

- Actual arrow and occupied-cell counts.
- Actual density.
- Dependency edge count.
- Longest dependency depth.
- Branching distribution.
- Initial and per-wave open choices.
- Cross-dependency count.
- Blocked-choice ratio.
- Path-length and turn distributions.
- Direction distribution.
- Visual crowding/readability score.
- Minimum successful actions.
- Solver work factor.
- Expected top 1%, top 10%, median, and average solve times.
- Analyzer confidence.
- Acceptance/rejection reasons.

Candidate acceptance requires every hard safety rule plus configured tolerance between requested and measured values. Difficulty cannot change because of server load, lock contention, queue length, or generation timeout.

##### Authoritative Collision Model

Decision: **Use integer cell-occupancy collision as the only authoritative Maze rule.**

The current prototype's declared `DependsOn` values may be retained temporarily as diagnostics or generation hints, but they are not authority.

Canonical collision:

1. Read the arrow's immutable head cell and direction.
2. Step one integer cell at a time toward the board edge.
3. Ignore removed arrows.
4. The first occupied cell belonging to another live arrow is the nearest blocker.
5. If a blocker exists, reject the action and leave game state unchanged.
6. If no blocker exists, mark the arrow removed in the actor's state.
7. Recalculate derived availability from the new live occupancy.
8. Complete only when every arrow is removed.

Why this model is authoritative:

- It matches the approved reference mechanic.
- Integer geometry avoids floating-point intersection ambiguity.
- Generation, solver, validator, action handling, replay, and rendering can share one rules definition.
- The nearest blocker and distance are deterministic, enabling instructional blocked animation.
- Dependencies cannot drift away from geometry.
- It supports canonical hashing and cross-language fixtures.

Derived dependency graph:

- May be calculated for solver, difficulty, hints, analytics, and replay explanation.
- Must always be derived from geometry plus live state.
- Must never override collision results.
- Must be invalidated or recalculated after state changes.

Visual stroke thickness, glow, shadows, and anti-aliasing do not change collision. Collision uses occupied logical cells only.

##### Generic Action Protocol

Realtime Arena transports generic game actions. It does not understand Maze payloads.

Conceptual client envelope:

```json
{
  "type": "game.action",
  "matchId": "mat_...",
  "actionId": "act_...",
  "sequence": 17,
  "expectedStateVersion": 12,
  "clientSentAt": "2026-07-28T12:00:00Z",
  "action": {
    "kind": "arrow.click",
    "payload": {
      "arrowId": "arrow_23"
    }
  }
}
```

Only the Maze module interprets `arrow.click` and `arrowId`. The client does not send direction, collision result, blocker, completion, combo, score, winner, seed, difficulty, or resulting state.

Generic dispatch:

```text
Authenticated gateway
  -> Validate envelope size and protocol
  -> Resolve participant and match
  -> Enforce action ID idempotency
  -> Enforce participant sequence and expected state version
  -> Resolve registered game module
  -> Build server ActionContext
  -> Validate game action
  -> Apply transition under match/participant concurrency control
  -> Persist action receipt, state, and hash-chained events atomically
  -> Emit client-safe result and snapshots
```

Conceptual result:

```json
{
  "type": "game.action.result",
  "matchId": "mat_...",
  "actionId": "act_...",
  "sequence": 104,
  "stateVersion": 13,
  "accepted": false,
  "code": "ACTION_BLOCKED",
  "events": [],
  "presentation": {}
}
```

For a blocked Maze action, `presentation` may include a client-safe blocker reference and collision distance so the renderer can animate impact and return. That metadata explains an authoritative result; it does not grant authority to the renderer.

Protocol controls:

- One globally unique action ID per submitted intent.
- Unique participant client sequence within a match.
- Monotonic server event sequence.
- Optimistic state-version check.
- Duplicate action returns the original result.
- Sequence gap requests synchronization.
- Stale state returns a stable conflict code and snapshot reference.
- Actions after completion, timeout, leave, or forfeit are rejected.
- Client timestamps are telemetry only.
- Payloads are size-limited and schema-validated by the selected game version.
- Per-player and per-match action rates are enforced server-side.

Adding `game.action` to the frozen gateway is permitted only in a later approved implementation phase as generic, backward-compatible integration support. No Maze fields may be added to the gateway's generic message type.

##### State And Concurrency Model

Every match has immutable shared puzzle metadata and separate mutable participant states.

Maze participant state contains conceptually:

- Match and participant references.
- Puzzle ID and puzzle hash.
- State schema version.
- State version.
- Removed-arrow bitset or canonical removed IDs.
- Successful action count.
- Blocked action count.
- Current combo and maximum combo.
- Completion percentage.
- Completion status.
- Started and completed server timestamps.
- Last accepted participant sequence.
- State checksum.

State transition rules:

- Apply actions under a per-match/per-participant distributed lock or an atomic compare-and-swap on state version.
- Never hold a global store lock during generation, solving, validation, analysis, or replay reconstruction.
- Persist participant state update, action receipt, and realtime events in one database transaction.
- Opponent state is independent and cannot be mutated by the actor's transaction.
- Reconnect reconstructs from the latest checksum-verified snapshot plus later events.
- A hash or state-version mismatch fails closed and requests recovery/review.

##### Replay Architecture

The generic Realtime Arena continues to own event persistence, object storage, root hashing, signatures, and replay delivery.

Maze owns the interpretation and deterministic reconstruction of Maze events.

Canonical Maze replay metadata:

- Replay ID and match ID.
- Game, rules, protocol, replay, renderer, and canonical encoding versions.
- Generator, solver, validator, analyzer, and Difficulty Profile schema versions.
- Puzzle ID.
- Seed or approved encrypted seed reference.
- Seed hash.
- Requested Difficulty Profile hash.
- Measured Difficulty Analysis hash.
- Generation hash.
- Puzzle hash.
- Validation hash.
- Ordered authoritative action event range.
- Server timing.
- Participant completion states.
- Winner/draw/invalid outcome.
- Event root hash.
- Replay hash.
- Platform signature and signing key ID.

The replay does not store a full maze as canonical state when deterministic reconstruction is possible.

Replay verification:

```text
Resolve exact historical versions
  -> Load seed and profile
  -> Regenerate candidate
  -> Recalculate generation and puzzle hashes
  -> Re-run validator and compare validation hash
  -> Start independent participant states
  -> Apply ordered authoritative events
  -> Compare state checksums and completion
  -> Recalculate outcome and replay hash
  -> Verify event root and platform signature
```

Optional generated geometry may exist as a temporary cache or dispute artifact, but it is never replay authority and must be checksum-linked to the canonical puzzle hash.

Historical generator artifacts and canonical encoders must be retained for the platform's replay-retention period. A generator version may be disabled for new matches without being deleted.

##### Hash Architecture

All hashes use a versioned canonical serialization. Raw language-specific JSON map ordering is not an approved canonical format.

| Hash | Canonical inputs |
|---|---|
| Seed hash | Secret seed bytes plus seed-format version |
| Difficulty Profile hash | Canonical requested profile plus profile schema version |
| Generation hash | Game ID, generator version, seed hash, profile hash, generation parameters, and canonical encoding version |
| Puzzle hash | Canonical immutable board geometry plus rules version |
| Validation hash | Puzzle hash, solver/validator versions, canonical solution hash, dependency analysis, and validation result |
| State checksum | Puzzle hash, participant state schema/version, removed IDs, progress, and last event sequence |
| Event integrity hash | Previous event hash, authoritative event envelope, server sequence, and state version |
| Replay hash | Version tuple, puzzle/validation hashes, participant event roots, timing, and outcome |

HMAC or digital signatures protect authenticity. Plain hashes provide identity and integrity comparison but are not signatures.

##### Database Design

Phase 3 may add normalized PostgreSQL structures through a new additive migration. Phase 2 defines the logical model only.

Existing frozen tables remain:

- `game_modules`.
- `realtime_matches`.
- `realtime_participants`.
- `realtime_events`.
- `realtime_snapshots`.
- `realtime_replays`.

Proposed logical tables:

###### `game_module_versions`

- `game_id`.
- `game_version`.
- `rules_version`.
- `protocol_version`.
- `replay_version`.
- `renderer_version`.
- `manifest_hash`.
- `artifact_digest`.
- `status`: active, replay_only, retired, revoked.
- `new_match_allowed`.
- `released_at`.
- `retired_at`.

Primary identity: game ID plus complete version tuple.

###### `game_generator_versions`

- `game_id`.
- `generator_version`.
- `solver_version`.
- `validator_version`.
- `analyzer_version`.
- `difficulty_profile_version`.
- `canonical_encoding_version`.
- `artifact_digest`.
- `determinism_fixture_hash`.
- `status`.
- `released_at`.
- `retired_at`.

Versions are immutable after use.

###### `game_difficulty_profiles`

- `id`.
- `game_id`.
- `schema_version`.
- `source`.
- Canonical profile data.
- `profile_hash`.
- `created_at`.

Profile hashes are unique per game and schema version.

###### `game_difficulty_analyses`

- `id`.
- `puzzle_id`.
- `analyzer_version`.
- Measured analysis data.
- `analysis_hash`.
- `accepted`.
- Rejection reasons.
- `created_at`.

###### `game_puzzles`

- `id`.
- `game_id`.
- `mode`.
- `seed_ciphertext` or secret-manager reference.
- `seed_hash`.
- `generation_nonce`.
- `generation_hash`.
- `puzzle_hash`.
- `validation_hash`.
- `difficulty_profile_id`.
- `difficulty_analysis_id`.
- Complete immutable version references.
- `solution_hash`.
- `minimum_actions`.
- `status`: generating, validated, rejected, assigned, consumed, retired.
- `created_at`.
- `validated_at`.

Canonical geometry is regenerated from versions, seed, and profile. It is not required as an authoritative database column.

###### `game_puzzle_uniqueness_claims`

- `puzzle_hash`.
- `seed_hash`.
- `puzzle_id`.
- `reuse_policy`.
- `first_scope_type`.
- `first_scope_id`.
- `claimed_at`.

One-use puzzle hashes and seeds receive exclusive claims. Approved reusable tutorial/daily fixtures use explicit non-one-use policies.

###### `game_puzzle_assignments`

- `id`.
- `puzzle_id`.
- `scope_type`: tutorial, session, match, house_attempt, daily_challenge.
- `scope_id`.
- `mode`.
- `reuse_policy`.
- `assigned_at`.
- `consumed_at`.

A PvP match has one assignment and two participant states.

###### `game_participant_states`

- `match_id`.
- `user_id`.
- `game_id`.
- `state_schema_version`.
- `state_version`.
- Opaque canonical state data.
- `state_checksum`.
- `last_client_sequence`.
- `last_server_sequence`.
- `status`.
- `updated_at`.

Primary identity: match ID plus user ID.

###### `game_action_receipts`

- `action_id`.
- `match_id`.
- `user_id`.
- `client_sequence`.
- `expected_state_version`.
- `action_kind`.
- Canonical action payload.
- `accepted`.
- Stable result code.
- `state_version_before`.
- `state_version_after`.
- First and last emitted realtime event sequences.
- `server_received_at`.
- `processed_at`.
- `receipt_hash`.

Unique constraints:

- Action ID.
- Match ID plus user ID plus client sequence.

###### `game_replay_metadata`

One-to-one companion to `realtime_replays`:

- `replay_id`.
- `puzzle_id`.
- Generator, solver, validator, analyzer, difficulty, renderer, and canonical encoding versions.
- Seed reference and seed hash.
- Difficulty Profile and analysis hashes.
- Generation, puzzle, validation, and replay hashes.
- Winner/draw/invalid outcome.
- Signing key ID.
- Verification status and timestamp.

No separate Maze event store is introduced. Ordered game events remain in `realtime_events`.

Database invariants:

- Puzzle versions and hashes are immutable after validation.
- One-use uniqueness claims are exclusive.
- One puzzle assignment per PvP/tournament match.
- Separate participant states per player.
- Action idempotency and participant sequence uniqueness.
- State version increases monotonically.
- Replays reference completed or explicitly invalidated matches.
- Historical version rows cannot be deleted while referenced.
- Foreign keys use restrictive deletion for integrity evidence.
- Seed plaintext never appears in logs, API errors, analytics, or client-visible active-match payloads.

##### Version Compatibility

The complete compatibility tuple is:

```text
game version
rules version
protocol version
replay version
renderer schema version
generator version
solver version
validator version
analyzer version
difficulty profile schema version
state schema version
canonical encoding version
```

Compatibility rules:

- New matches use one approved complete tuple.
- Match state, actions, snapshots, and replay metadata pin that tuple.
- Versions are append-only; behavior is not changed in place.
- Patch versions may be declared compatible only through tested manifest metadata.
- Rules or canonical encoding changes require explicit new versions.
- Replays always resolve exact versions.
- Renderer clients declare supported renderer schemas.
- If a client cannot render the pinned schema, the server rejects entry or uses an explicitly versioned approved transformer.
- No silent fallback, implicit upgrade, or `latest` substitution.
- Revoked versions remain available to integrity staff where legally and operationally safe, but cannot start new matches.

Required compatibility tests:

- Same seed/profile/version produces byte-identical canonical geometry.
- Server and renderer fixtures agree on arrow IDs, cells, and directions.
- Historical replay reconstructs to the same final checksums.
- Old action payloads remain readable by their pinned protocol version.
- Unsupported combinations fail startup or match creation.

##### Rendering Boundary

Client-side only:

- Canvas/WebGL/DOM rendering.
- Camera, zoom, pan, resize, and coordinate transforms.
- Pointer, touch, keyboard, and accessible input mapping to an arrow ID.
- Press, hover, focus, and pending-response states.
- Correct-exit animation after server acceptance.
- Blocked impact and return animation after server rejection.
- Particle effects, sound, haptics, glow, highlighting, and reduced motion.
- Interpolation between authoritative snapshots.
- Display of server-provided clock, progress, combo, status, and result.

Server-authoritative:

- Puzzle generation and identity.
- Direction and geometry.
- Live occupancy.
- Collision and blocker.
- Action acceptance.
- Removed state.
- Progress and combo.
- Timer and timeout.
- Completion and score inputs.
- Winner, draw, forfeit, and invalidation.
- Replay and integrity.

Renderer snapshot views:

| Viewer | Allowed data |
|---|---|
| Player | Own complete renderable board, own progress, approved public opponent progress |
| Opponent | Never receives the other player's hidden action history or mutable board state |
| Spectator | Approved delayed/public state only |
| Replay viewer | Post-match state according to replay visibility rules |
| Support reviewer | Required dispute evidence under RBAC |
| Integrity verifier | Full authoritative reconstruction inputs under privileged service identity |

Animation state is presentation state. It may lag authoritative state briefly, but it cannot affect validation. If a sync conflicts with local animation, the renderer cancels or reconciles the animation and displays the server snapshot.

##### Error Contract

Generic stable result codes:

- `GAME_NOT_REGISTERED`.
- `GAME_VERSION_UNAVAILABLE`.
- `GAME_ACTION_UNSUPPORTED`.
- `ACTION_DUPLICATE`.
- `ACTION_OUT_OF_ORDER`.
- `STATE_VERSION_CONFLICT`.
- `MATCH_NOT_LIVE`.
- `MATCH_ALREADY_COMPLETE`.
- `PARTICIPANT_NOT_FOUND`.
- `GAME_STATE_INVALID`.
- `GAME_PROCESSING_UNAVAILABLE`.

Maze-specific stable result codes:

- `ARROW_NOT_FOUND`.
- `ARROW_ALREADY_REMOVED`.
- `ACTION_BLOCKED`.
- `PUZZLE_STATE_INVALID`.

Errors do not reveal hidden opponent state, secret seeds, solution order, internal stack traces, or sensitive integrity data.

##### Observability And Operational Boundaries

Games emit structured metrics through platform observability:

- Generation attempts, acceptance rate, and duration.
- Solver, validator, analyzer, and replay reconstruction duration.
- Rejection reasons.
- Duplicate puzzle candidates.
- Difficulty requested-versus-measured variance.
- Actions accepted, blocked, duplicate, stale, and out of order.
- State-version conflicts.
- Replay verification success and failure.
- Per-version error and latency rates.

Games do not create their own logging, tracing, alerting, worker, or queue infrastructure. They use the frozen platform facilities and include game/version/match correlation fields.

##### Phase 2 Decisions

| Decision | Approved architecture |
|---|---|
| Games Platform | Registry-driven plugin layer above frozen Arena/Realtime services |
| Folder structure | `games/registry`, `games/interfaces`, `games/shared`, and one folder per game |
| Generic game interface | Opaque state, generic actions, deterministic transitions, viewer-safe snapshots, replay codec |
| Maze rules | Integer cell-occupancy arrow escape |
| Dependencies | Derived from geometry; never separate authority |
| Puzzle generation | Server-only, versioned pipeline outside global locks |
| Fixed levels | Tutorial 1-5 only; no baked production catalogue |
| Practice | Fresh one-use puzzle per participant session |
| House | Fresh one-use puzzle per attempt |
| PvP/Ranked | One fresh shared puzzle per match, independent player states |
| Tournament | One fresh shared puzzle per tournament match |
| Difficulty | Requested profile plus independently measured analysis |
| Replay | Exact-version seed reconstruction plus authoritative events and signatures |
| Storage | Normalized metadata, uniqueness claims, participant states, action receipts, existing realtime event stream |
| Networking | Generic Realtime Arena action envelope only |
| Rendering | Client presentation only; no gameplay authority |
| Versioning | Complete immutable compatibility tuple; no silent fallback |

##### Phase 2 Definition Of Done

Documentation deliverables are complete:

- Games Platform architecture.
- Registry and folder structure.
- Generic runtime game interface.
- Maze plugin ownership.
- Production Puzzle Service design.
- Generator architecture.
- Tutorial and fixed-level policy.
- Permanent uniqueness rules.
- Requested and measured difficulty architecture.
- Authoritative collision decision.
- Generic action protocol.
- State and concurrency model.
- Replay and hash architecture.
- PostgreSQL logical model.
- Version compatibility rules.
- Rendering boundary.
- Error and observability contracts.

No production gameplay code, migration, API, React component, CSS, or frozen Sprint 1 through Sprint 5 implementation was changed during Phase 2.

Phase 2 is **APPROVED**.

Sprint 6 implementation remains **NOT STARTED**.

Phase 3 may define the Puzzle Generator and Solver. Production implementation remains prohibited until the design phases are approved.

#### Sprint 6 Phase 3: Puzzle Generator And Solver Design

Status: **APPROVED**

Implementation status: **SPRINT 6 NOT STARTED**

Phase 2 is approved. Phase 3 finalizes the production Maze puzzle generator, solver, validator, difficulty calibration, hashing, uniqueness, and performance design. It does not implement these systems.

##### Permanent Puzzle Generation Contract

These rules are permanent platform requirements:

1. Tutorial puzzles 1 through 5 may be handcrafted and versioned.
2. Every non-tutorial puzzle is generated by the server.
3. Every live PvP or Ranked match receives exactly one newly generated puzzle.
4. Both participants in that match receive identical immutable puzzle geometry and metadata.
5. Each participant receives independent mutable board state.
6. Practice receives a newly generated puzzle for every session.
7. House Challenge receives a newly generated puzzle for every attempt.
8. Tournament puzzles are generated specifically for each tournament match.
9. Daily Challenge may intentionally share one approved puzzle during its challenge window.
10. Generation uses an immutable Generator Version, requested Difficulty Profile, and cryptographically secure seed.
11. The database prevents a one-use puzzle from being intentionally assigned again.
12. Replays reconstruct puzzles from exact generator metadata whenever deterministic reconstruction is possible.
13. Full layouts are not canonical replay storage.
14. No client may choose or submit the live seed, Difficulty Profile, puzzle, solution, or generation result.
15. No invalid, duplicate, deadlocked, or out-of-profile puzzle may reach a player.

##### Generator Objectives

The production generator must:

- Produce deterministic output for identical versioned inputs.
- Produce a fresh one-use puzzle for modes that require uniqueness.
- Generate readable orthogonal arrow geometry.
- Create dependencies through physical collision.
- Meet an authoritative requested Difficulty Profile within approved tolerances.
- Produce enough branching and variation to avoid a fixed memorized solution.
- Pass an independent solver and validator.
- Avoid runtime, map-order, floating-point, clock, thread-scheduling, or platform-dependent output.
- Support exact reconstruction years later.
- Fail closed when it cannot satisfy the contract.

The generator does not decide player progression, matchmaking, rewards, or competition outcomes.

##### Complete Generation Pipeline

```text
Authoritative generation request
        |
        v
Resolve complete immutable version tuple
        |
        v
Load requested Difficulty Profile
        |
        v
Generate secure seed material
        |
        v
Derive deterministic generator stream
        |
        v
Select pattern inputs
        |
        v
Generate bounded candidate batch
        |
        v
Structural geometry validation
        |
        v
Collision and dependency derivation
        |
        v
Independent solver validation
        |
        v
Gameplay simulation validation
        |
        v
Measured difficulty analysis
        |
        v
Requested-versus-measured verification
        |
        v
Deterministic candidate selection
        |
        v
Canonical puzzle and validation hashes
        |
        v
Transactional database uniqueness claim
        |
        v
Immutable metadata persistence and assignment
        |
        v
Production-ready renderer state
```

No candidate is delivered before the uniqueness claim and assignment transaction succeeds.

##### Generation Request

The authoritative request contains:

- Request ID and idempotency key.
- Game ID.
- Match, session, challenge, or tournament scope.
- Mode and reuse policy.
- Complete version tuple.
- Requested Difficulty Profile ID and hash.
- Participant ID only for non-shared generation.
- Server-owned generation policy.
- Deadline and priority.

The request does not accept client-provided:

- Seed.
- Pattern.
- Difficulty values.
- Candidate count.
- Puzzle geometry.
- Solution.
- Reuse policy.
- Generator version override.

An internal integrity or developer tool may request explicit test fixtures only under privileged non-production policy and auditable configuration.

##### Seed Strategy

Seed creation uses two layers:

1. A 256-bit random generation secret from the operating system cryptographic random source.
2. A domain-separated deterministic seed derived from that secret and immutable generation context.

Conceptual derivation:

```text
random material: CSPRNG(32 bytes)

effective seed:
  HMAC-SHA-256(
    generation key,
    seed format version
    + random material
    + game ID
    + scope type
    + scope ID
    + generator version
    + Difficulty Profile hash
    + generation nonce
  )
```

The production design may use HKDF instead of direct HMAC if the implementation contract specifies the exact algorithm and vectors. The algorithm cannot change without a Seed Format Version change.

Seed requirements:

- At least 256 bits of cryptographically secure random material.
- Domain separation between Practice, House, PvP, Ranked, Tournament, Daily, and replay fixtures.
- No derivation from player ID, level, date, match number, or public data alone.
- No use of `math/rand`, JavaScript `Math.random`, Park-Miller, or another runtime-dependent pseudo-random generator.
- No live seed in application logs, traces, analytics, error messages, or client payloads.
- Encrypted seed material or secret-manager reference at rest.
- Separate non-secret seed hash for uniqueness and auditing.
- Exact seed format and derivation version stored with the puzzle.

Mode-specific seed rules:

| Mode | Seed scope |
|---|---|
| Tutorial | No generated seed required; fixture version is authoritative |
| Practice | New random material for each player session |
| Training | New random material unless using an approved lesson fixture |
| House | New random material for each attempt |
| PvP/Ranked | One new random seed scoped to the match |
| Tournament | One new random seed scoped to each tournament match |
| Daily | One new random seed scoped to the challenge window |
| Replay | Decrypt/load the original seed; never generate a replacement |

Both PvP participants reference the same puzzle and seed metadata. Their mutable states remain separate.

##### Deterministic Random Stream

The generator must not depend on a language standard-library RNG whose output may change between versions.

Decision: use a specified HMAC-SHA-256 counter stream for deterministic generation.

Conceptual stream:

```text
block[n] = HMAC-SHA-256(effective_seed, stream_version + domain + uint64_be(n))
```

Domains separate:

- Pattern selection.
- Board dimensions.
- Cluster anchors.
- Arrow placement.
- Direction choice.
- Path growth.
- Candidate derivation.
- Tie breaking.

Random integer selection uses rejection sampling to avoid modulo bias. Generator scoring uses integers or explicitly scaled fixed-point values. Generator authority does not use platform-dependent floating-point comparisons.

Required published test vectors:

- Effective seed.
- Domain.
- Counter.
- Expected output block.
- Expected bounded integer samples.
- Expected final puzzle hash.

These vectors must pass in every supported implementation language.

##### Generator Versioning

The Generator Version identifies behavior, not merely a release label.

The complete generator identity includes:

- Generator algorithm version.
- Seed format version.
- Deterministic stream version.
- Pattern catalogue version.
- Pattern selection version.
- Geometry schema version.
- Candidate scoring version.
- Canonical encoding version.
- Default constraint-policy version.

Versioning rules:

- Output-changing behavior requires a new Generator Version.
- Pattern weight changes require a new Pattern Selection or Generator Version.
- Bug fixes that change generated geometry require a new version.
- Historical versions are immutable.
- A version can become `replay_only` but cannot be deleted while referenced.
- Every version stores an artifact digest and determinism fixture hash.
- New matches never select an unapproved or replay-only version.
- Replay resolves the exact historical version and refuses silent substitution.

Version qualification requires:

- Determinism fixtures.
- Cross-platform hash parity.
- Solver and validator compatibility.
- Difficulty calibration corpus.
- Performance profile.
- Security review.
- Approved activation record.

##### Candidate Generation

One seed may derive a bounded set of candidate streams:

```text
candidate_seed[i] =
  HMAC-SHA-256(effective_seed, "candidate" + uint64_be(i))
```

Candidate indexes are stable and begin at zero.

Generation may run candidate indexes in parallel, but selection cannot depend on which worker finishes first.

Deterministic selection:

1. Generate a configured fixed candidate batch for the requested profile.
2. Reject candidates that fail any hard validation.
3. Rank accepted candidates using an immutable integer scoring tuple:
   - requested-versus-measured difficulty distance;
   - readability compliance;
   - dependency and branching target distance;
   - pattern-quality score;
   - candidate index as final tie breaker.
4. Select the same winning candidate regardless of worker scheduling.
5. Persist the selected candidate index.

If no candidate passes:

- Mark the generation attempt failed.
- Record non-sensitive failure reasons and metrics.
- Do not return an easier or escape-only board.
- Do not reduce the requested Difficulty Profile.
- Do not reuse a previous puzzle.
- Retry with new cryptographic seed material under bounded orchestration policy.
- Fail the match/session safely when its deadline is exhausted.

##### Board And Arrow Geometry

Canonical board:

- Positive integer width and height.
- Integer row/column coordinates.
- Stable coordinate origin and direction encoding.
- Versioned geometry schema.

Canonical arrow:

- Stable ID derived from canonical placement order.
- Non-empty ordered cells from tail to head.
- Four-direction enum: right, up, left, down.
- Orthogonally adjacent consecutive cells.
- No repeated cell within the arrow.
- No cell shared with another arrow.
- Final body segment aligned with the declared direction.
- Head within the board.
- Escape ray evaluated from one cell beyond the head.

Structural rejection conditions:

- Empty board or arrow.
- Out-of-bounds cell.
- Duplicate occupied cell.
- Self-intersection through a repeated cell.
- Diagonal or disconnected path.
- Direction outside the versioned enum.
- Head/direction mismatch.
- Arrow ID collision.
- Board above approved readability or memory bounds.
- Canonical encoding failure.

Visual stroke width, glow, gradients, shadows, and animation do not alter logical geometry.

##### Pattern System

Patterns influence candidate generation but never change rules.

Initial pattern families:

- Braid.
- Spiral.
- Maze rows.
- Rings.
- Mosaic.
- Piton.
- Diagonal weave.
- Rays.

Each pattern definition contains:

- Pattern ID and version.
- Eligible difficulty range.
- Minimum board dimensions.
- Weighted spatial attractors.
- Direction and path-length biases.
- Density and cluster preferences.
- Readability limits.
- Compatibility with other secondary pattern influences.

Pattern selection:

1. Filter patterns by requested profile and board constraints.
2. Calculate integer weights from the immutable Pattern Selection Version.
3. Select with the pattern-domain deterministic random stream.
4. Optionally select a bounded secondary influence when the profile permits.
5. Store selected pattern IDs and versions in generation metadata.

Anti-predictability rules:

- No public level-to-pattern rotation.
- No fixed seven-level or chapter sequence.
- No public seed formula.
- Pattern does not determine a reusable layout.
- Seed variation changes anchors, routes, dependencies, density, and arrow identity.
- Pattern metadata need not be exposed during an active match.

Patterns are not player-visible "levels." The player may see a neutral post-match analysis label if product design later approves it.

##### Dependency Creation

Dependencies are created only through physical geometry.

For arrow `A`, every other live arrow occupying a cell on `A`'s directional escape ray is a blocker that must be removed before `A` can escape.

Dependency graph:

- Node: arrow ID.
- Directed edge `A -> B`: arrow A requires arrow B to be removed.
- Multiple occupied cells from B on A's ray create one edge.
- Graph is derived after geometry generation.
- Graph is never stored as independent authority.
- Graph may be persisted as hashed analysis evidence or regenerated from geometry.

Generation targets:

- At least one initially open arrow.
- Dependency depth within requested tolerance.
- Branching within requested tolerance.
- Cross dependencies within requested tolerance.
- No dependency cycle.
- No isolated competitive arrow unless the Difficulty Profile explicitly permits it.

An isolated arrow has zero incoming and zero outgoing dependency degree. Low-level tutorial fixtures may intentionally include isolated arrows to teach basic movement. Generated competitive boards reject them because they add actions without strategic dependency.

##### Authoritative Solver

The solver is independent of generator placement logic. It receives canonical board geometry and rules version only.

The solver must not trust:

- Generator placement order.
- Generator-provided solution.
- Declared dependency metadata.
- Client state.
- Baked solution data.

Solver stages:

1. Re-run structural geometry validation.
2. Build an integer occupancy index.
3. Calculate every arrow's complete blocker set from its escape ray.
4. Build the dependency graph.
5. Detect cycles.
6. Run deterministic topological removal simulation.
7. Apply each selected removal through the same canonical collision function used by live gameplay.
8. Verify all arrows are removed.
9. Return canonical solution, solution-shape classification, dependency analysis, and solver hash inputs.

Canonical solution tie break:

- At each step, calculate all currently open arrows.
- Select the smallest canonical arrow ID.
- Record the selected arrow.
- Continue until complete or deadlocked.

The canonical solution is for validation, replay checks, automated tests, and analysis. It is not the only sequence a player must use unless the graph itself has a unique ordering.

##### Multiple Solutions Policy

Decision: **Multiple valid completion orders are intentionally allowed for generated competitive puzzles.**

Reasoning:

- Branching is an approved Difficulty Profile dimension.
- Multiple open choices create strategy and replay learning.
- Both PvP players receive the same choices, preserving fairness.
- Forcing one global sequence would make branching impossible and increase memorization.
- In this monotonic removal game, a valid removal only removes blockers and cannot create a new blocker.

The solver classifies solution shape:

- `unique`: exactly one valid topological completion order.
- `multiple`: more than one valid topological completion order.
- `unsolvable`: no complete order.

A dependency DAG has a unique topological ordering only when exactly one node is available at every Kahn solver step. If two or more nodes are available at any step, multiple valid orders exist.

Profile policy:

- Tutorial may require a unique or tightly guided order.
- Early Practice may allow low branching.
- Ranked, advanced Practice, House, and Tournament may intentionally require multiple choices.
- The measured branching profile must match the requested range.

The platform does not need to enumerate an exponential number of solutions. It needs to prove complete solvability and classify unique versus multiple ordering.

##### Solver Guarantees

A puzzle passes solver validation only when:

- At least one arrow exists.
- At least one initial legal action exists.
- Every arrow is reachable by a complete removal order.
- The dependency graph is acyclic.
- Canonical simulation removes every arrow.
- Every simulated action agrees with the live collision function.
- No generated competitive arrow is isolated unless permitted.
- No impossible or ambiguous geometry exists.
- Minimum successful actions equals the number of arrows.
- Unique/multiple solution classification is known.
- Dependency and branching metrics are complete.
- Solver output is deterministic.

No solver timeout is interpreted as solvable. A timeout or resource limit is a validation failure.

##### Collision Validation

One pure versioned collision function is shared conceptually by:

- Generator validation.
- Dependency derivation.
- Solver simulation.
- Live action validation.
- Replay reconstruction.
- Dispute verification.

Collision input:

- Immutable board geometry.
- Removed-arrow state.
- Target arrow ID.
- Rules version.

Collision output:

- `clear` or `blocked`.
- Nearest blocker ID when blocked.
- Collision cell.
- Distance in cells from the head.
- Escape distance to complete board exit when clear.

Collision invariants:

- Only cells in the forward directional ray count.
- Removed arrows do not block.
- The target arrow does not block itself.
- Visual dimensions do not affect collision.
- Same input produces byte-identical canonical output.
- Rejected blocked action does not mutate state.
- Accepted action removes only the target arrow.

##### Gameplay Simulation Validator

After solver success, a separate validator replays the canonical solution through the production rule contract.

It verifies:

- Every action is accepted in order.
- State version increments correctly.
- Removed-arrow state is monotonic.
- Derived progress reaches exactly 100%.
- Completion occurs only after the final arrow.
- No action removes more than one arrow.
- No hidden generator shortcut bypasses collision.
- Canonical final state checksum matches solver expectations.

This guards against a solver that proves a graph property but disagrees with live action semantics.

##### Difficulty Calibration

The generator receives a requested Difficulty Profile. The analyzer measures the completed candidate independently.

Measured features:

- Board width and height.
- Arrow count.
- Occupied-cell count and density.
- Minimum successful actions.
- Initially open arrow count.
- Open-choice count at every solver wave.
- Dependency edge count.
- Longest dependency depth.
- Branching distribution.
- Cross-dependency count.
- Isolated-arrow count.
- Blocked-choice ratio.
- Arrow path-length percentiles.
- Turn-count percentiles.
- Direction distribution.
- Spatial cluster distribution.
- Nearest-route spacing.
- Visual crowding.
- Minimum rendered cell size at target viewports.
- Canonical solver operations.
- Expected solve-time percentiles.

Difficulty acceptance has two levels:

Hard constraints:

- Solvable.
- No cycles.
- No prohibited isolated arrows.
- Geometry valid.
- Density and visual scale within accessibility/readability bounds.
- Required arrow count and dependency depth safety bounds.

Tolerance constraints:

- Complexity score distance.
- Line count.
- Branching.
- False-route/blocked-choice ratio.
- Density.
- Pattern bias result.
- Expected solve time.
- Visual complexity.

The candidate score uses integer normalized distances. Weights belong to the immutable Difficulty Analyzer Version.

No candidate is accepted solely because the generator intended it to be difficult. Measured analysis is authoritative.

##### Expected Solve Time Calibration

Expected solve time is initially estimated from a versioned model using:

- Successful action count.
- Number and distribution of open choices.
- Dependency depth.
- Blocked-choice exposure.
- Path readability.
- Visual complexity.
- Board navigation burden.
- Historical anonymous performance cohorts when enough production data exists.

Before production telemetry is available:

- Use solver-derived features and controlled human calibration sessions.
- Mark model confidence explicitly.
- Use conservative broad percentile ranges.

After sufficient verified data:

- Recalibrate in a new Analyzer Version.
- Never rewrite historical puzzle analysis.
- Exclude suspicious, assisted, disconnected, or invalid sessions.
- Keep player Trust Score out of the puzzle's measured intrinsic difficulty.

##### Hashing And Canonicalization

Canonical board encoding includes:

- Geometry schema version.
- Board width and height.
- Arrows sorted by canonical ID.
- Each arrow's direction.
- Each arrow's ordered tail-to-head cell list.
- Rules version.

It excludes:

- Removed state.
- Player identity.
- Match result.
- Animation.
- Color.
- Sound.
- Client timestamps.

Hashes:

| Hash | Purpose |
|---|---|
| Seed hash | Audit and duplicate-seed detection without exposing seed |
| Difficulty Profile hash | Identity of requested generation target |
| Generation hash | Identity of generator inputs and selected candidate |
| Puzzle hash | Identity of immutable canonical board and rules |
| Solver hash | Identity of canonical solution and dependency analysis |
| Validation hash | Identity of all required validation results |
| Replay genesis hash | Binds immutable puzzle metadata before actions exist |
| Final replay hash | Binds replay genesis, ordered authoritative events, timing, and outcome |

Generation hash inputs:

- Game ID.
- Complete generator identity.
- Seed hash.
- Difficulty Profile hash.
- Selected pattern metadata.
- Candidate index.
- Canonical encoding version.

Validation hash inputs:

- Puzzle hash.
- Solver, validator, analyzer, and rules versions.
- Solver hash.
- Measured analysis hash.
- Gameplay simulation final checksum.
- Acceptance decision.

The final Replay Hash cannot exist until the match ends. The Puzzle Service produces the Replay Genesis Hash; Realtime Replay infrastructure produces the final Replay Hash.

##### Database Uniqueness Flow

```text
Begin short transaction
  -> Insert immutable puzzle metadata
  -> Claim seed hash under one-use policy
  -> Claim puzzle hash under one-use policy
  -> Insert scope assignment
  -> Commit
```

If a seed or puzzle hash uniqueness constraint conflicts:

- Roll back.
- Mark the generation attempt as duplicate.
- Generate new cryptographic seed material.
- Never assign the existing puzzle to the new one-use scope.

Shared PvP behavior:

- One puzzle row.
- One puzzle assignment for the match.
- Two participant state rows.

Tournament behavior:

- One puzzle assignment for each tournament match.
- Different tournament matches receive different one-use claims.

Daily behavior:

- One explicit reusable challenge-window assignment.
- All participants reference that approved assignment.
- It cannot be reassigned as a live one-use puzzle.

Replay reconstruction reads existing metadata and does not attempt a new uniqueness claim.

##### Production Validation Pipeline

Every generated candidate passes these gates in order:

| Gate | Verification | Failure result |
|---|---|---|
| Version | Complete approved tuple exists | Reject |
| Seed | Format, length, scope, and derivation valid | Reject |
| Structure | Board and arrow schema valid | Reject |
| Geometry | Bounds, continuity, occupancy, head direction valid | Reject |
| Collision | Every ray and nearest blocker deterministic | Reject |
| Dependency | Graph derivation complete and acyclic | Reject |
| Solver | Complete canonical solution exists | Reject |
| Isolation | No prohibited degree-zero arrow | Reject |
| Simulation | Canonical actions reproduce live rules | Reject |
| Difficulty | Hard constraints and tolerances pass | Reject |
| Determinism | Canonical encoding and expected hashes stable | Reject |
| Uniqueness | Seed and puzzle claims available | Retry new seed |
| Persistence | Metadata and assignment transaction commits | Fail closed |

There is no warning-only path for a production puzzle.

##### Performance Design

The existing Sprint 5 Practice lifecycle benchmark measured approximately 4.34 ms per operation, but it uses the current prototype generator and is not a production Generator/Solver certification.

Phase 3 planning estimates for the complete production pipeline on a modern server CPU:

| Profile | Expected candidate pipeline | Initial service target |
|---|---:|---:|
| Tutorial fixture validation | Under 10 ms | P99 under 50 ms |
| Standard Practice/PvP | 25-250 ms | P50 under 100 ms, P95 under 500 ms, P99 under 1.5 s |
| Advanced Ranked/House | 100-750 ms | P50 under 300 ms, P95 under 1.5 s, P99 under 3 s |
| Elite Tournament | 250 ms-2 s | P50 under 750 ms, P95 under 3 s, P99 under 5 s |
| Replay reconstruction | 10-250 ms | P95 under 500 ms for standard profiles |

These are design targets, not measured production claims. Phase 3 implementation must add dedicated generator, solver, validator, analyzer, hash, uniqueness, and reconstruction benchmarks before freeze.

Hard generation deadline:

- Configured by profile class.
- Never used to lower difficulty.
- Standard initial ceiling: 5 seconds.
- Elite/Tournament initial ceiling: 10 seconds.
- Deadline exhaustion fails generation safely.
- Final ceilings require load-test evidence.

##### Concurrency And Scalability

Generator workers are stateless with respect to active player state.

Concurrency rules:

- No global store lock during generation, solving, validation, analysis, hashing, or reconstruction.
- Candidate indexes may run concurrently in bounded worker pools.
- Worker completion order never affects selected output.
- CPU concurrency is bounded per process.
- Jobs have context deadline, cancellation, priority, and idempotency key.
- Database transactions remain short.
- Database uniqueness constraints are final authority.
- Redis may coordinate reservations and backpressure but cannot replace PostgreSQL uniqueness.
- Queue depth and oldest-job age are monitored.
- Generation capacity scales horizontally by adding workers.

Priority order:

1. Active Tournament or matched live competition awaiting a puzzle.
2. Ranked/PvP match preparation.
3. Practice and House requests.
4. Pool replenishment.
5. Offline calibration and corpus generation.

The platform should prepare a puzzle before declaring a match live. A player does not enter an active timer while generation is incomplete.

##### Caching Strategy

Allowed caches:

- Immutable version manifests and artifacts.
- Difficulty Profile records by profile hash.
- Pattern definitions by version.
- Validated, unassigned one-use puzzle pools.
- Reconstructed immutable geometry keyed by puzzle hash.
- Replay reconstruction results keyed by replay hash.

Rules for pre-generated pools:

- Every pooled puzzle already passed all validation.
- Every puzzle remains unassigned and one-use.
- Claiming it is an atomic database operation.
- Pool selection does not reuse a puzzle.
- Pools are partitioned by complete version tuple and Difficulty Profile class.
- Expired pools may be retired, never silently reassigned across incompatible profiles.
- A cached puzzle is not public and is not sent before assignment.

Disallowed caches:

- Reusable live puzzle catalogue.
- Client-side generated boards.
- Cached authoritative participant state without durable version checks.
- Cache entries lacking puzzle hash and version tuple.
- A fallback cache that changes difficulty.

Cache loss affects latency, not integrity. PostgreSQL metadata and object/version artifacts remain authoritative.

##### Failure Handling

| Failure | Required behavior |
|---|---|
| CSPRNG unavailable | Fail generation and alert; never use weak randomness |
| Version unavailable | Reject request; no fallback to latest |
| Candidate deadlock | Reject candidate |
| All candidates fail | Retry with new seed under bounded policy |
| Difficulty mismatch | Reject candidate; never relabel |
| Duplicate seed/puzzle | Roll back claim and generate fresh seed |
| Worker timeout | Cancel work, preserve metrics, fail safely |
| Worker crash | Idempotent job retry on another worker |
| PostgreSQL unavailable | Do not assign or deliver puzzle |
| Redis unavailable | Use approved degraded queue path only if PostgreSQL uniqueness remains enforceable |
| Object storage unavailable | Generation may continue only if replay/version retention requirements remain safely queued; live policy decided before implementation |
| Hash mismatch | Quarantine puzzle and create integrity alert |
| Replay reconstruction mismatch | Mark replay under review; never rewrite history |
| Match cancelled during generation | Discard or return unassigned candidate to an approved private pool |

Financial matches must emit platform cancellation/refund instructions if generation fails after funds are locked. The Maze module does not perform the refund itself.

##### Generator Security

Security requirements:

- Seed-encryption key comes from the approved secret manager.
- Separate keys for seed derivation, seed encryption, replay signing, and event integrity.
- Key IDs are stored; key material is not.
- Active seed material is least-privilege restricted.
- Generation workers cannot access wallet or admin data.
- Puzzle metadata APIs redact seed and canonical solution during active matches.
- No endpoint returns solver output to active players.
- Generation requests are internal authenticated service calls.
- Candidate failure logs contain codes, not board dumps or secrets.
- Test fixtures never use production keys.
- Bot resistance does not rely on hiding the rules.

##### Generator And Solver Test Strategy

Required test layers:

Unit:

- Deterministic random stream vectors.
- Geometry rules.
- Collision and nearest blocker.
- Dependency graph.
- Unique/multiple solution classification.
- Canonical encoding and every hash.
- Difficulty feature extraction.
- Candidate ranking.

Property:

- Same input always gives same output.
- Every accepted puzzle is solvable.
- Accepted state transitions are monotonic.
- Removing an arrow never creates a blocker.
- No occupied-cell overlap.
- Every arrow direction aligns with its head segment.
- Puzzle hash changes when canonical geometry changes.
- Renderer-only metadata never changes puzzle hash.

Fuzz:

- Malformed geometry.
- Extreme board sizes.
- Duplicate IDs and cells.
- Invalid directions.
- Corrupted replay metadata.
- State/version mismatch.
- Canonical decoder inputs.

Integration:

- Generate -> solve -> validate -> analyze -> hash -> claim -> assign.
- Concurrent duplicate uniqueness claims.
- Shared PvP puzzle with independent states.
- Practice uniqueness across large sample.
- Tournament per-match uniqueness.
- Replay regeneration across stored versions.
- PostgreSQL rollback and retry.
- Worker cancellation and recovery.

Parity:

- Reference handcrafted tutorial fixtures.
- Approved reference collision fixtures.
- Backend, renderer, and replay verifier agree on IDs, cells, directions, blockers, and hashes.

Load:

- Concurrent generation by profile class.
- Pool claim contention.
- Replay reconstruction bursts.
- Worker restart and queue recovery.
- Database uniqueness conflicts.

Long-run corpus:

- Generate at least 100,000 candidates across approved profile bands before freeze.
- Zero accepted deadlocks.
- Zero accepted structural violations.
- Zero duplicate one-use assignments.
- Difficulty acceptance and rejection distributions documented.
- Pattern distribution and readability outliers reviewed.

##### Qualification And Release Gates

A Generator Version cannot become active until:

- All deterministic vectors pass.
- Solver and validator independently pass every accepted corpus puzzle.
- Reference rule fixtures pass.
- 100,000-candidate qualification corpus has zero invalid accepted puzzles.
- Requested-versus-measured calibration is documented.
- Performance targets are measured.
- Cross-platform canonical hashes match.
- PostgreSQL uniqueness races are tested.
- Replay reconstruction succeeds from stored metadata.
- Security and secrets review passes.
- Observability dashboards and alerts exist.
- Rollback to the prior approved version is proven.

Activation:

- Register version as inactive.
- Run qualification.
- Approve for selected modes/profile bands.
- Enable through server configuration.
- Monitor acceptance, latency, failures, and integrity.
- Expand gradually.

Rollback:

- Stop assigning the affected version to new matches.
- Preserve it for replay and disputes.
- Continue existing matches unless integrity policy requires cancellation.
- Never mutate historical puzzle metadata.

##### Phase 3 Decisions

| Decision | Approved design |
|---|---|
| Tutorial | Five versioned handcrafted fixtures allowed |
| Non-tutorial puzzles | Server-generated only |
| Seed | 256-bit CSPRNG material plus domain-separated HMAC derivation |
| Deterministic RNG | Versioned HMAC-SHA-256 counter stream |
| Candidate generation | Bounded deterministic batch; parallel execution allowed |
| Candidate selection | Immutable integer score tuple, never fastest-worker wins |
| Patterns | Versioned weighted generator inputs, not levels |
| Geometry | Integer orthogonal tail-to-head cells |
| Dependencies | Derived from physical escape-ray collision |
| Collision | One pure canonical function shared by all rule consumers |
| Solver | Independent graph plus live-rule simulation |
| Multiple solutions | Intentionally allowed and measured |
| Deadlocks | Hard rejection |
| Isolated arrows | Rejected in generated competition unless profile explicitly permits |
| Difficulty | Measured output must match requested profile |
| Hashes | Versioned canonical seed, generation, puzzle, solver, validation, and replay hashes |
| Uniqueness | PostgreSQL claims over seed and puzzle hash |
| Caching | Private validated unassigned one-use pools allowed |
| Performance | Bounded stateless workers outside global locks |
| Failure | Fail closed; never lower difficulty or return fallback board |
| Replay | Exact historical regeneration plus authoritative actions |

##### Phase 3 Definition Of Done

Documentation deliverables are complete:

- Permanent puzzle generation rules.
- Complete production generation pipeline.
- Secure seed and deterministic random-stream strategy.
- Generator versioning.
- Deterministic candidate generation and selection.
- Pattern system.
- Geometry and dependency creation.
- Independent authoritative solver.
- Intentional multiple-solution policy.
- Collision and gameplay simulation validation.
- Requested-versus-measured difficulty calibration.
- Canonical hashing and replay genesis.
- Database uniqueness flow.
- Performance targets.
- Concurrency, scaling, caching, and failure handling.
- Security requirements.
- Test, qualification, activation, and rollback gates.

No production gameplay code, migration, API, worker, React component, CSS, or frozen Sprint 1 through Sprint 5 implementation was changed during Phase 3.

Phase 3 is **APPROVED**.

Sprint 6 implementation remains **NOT STARTED**.

Phase 4 may convert the approved architecture into an implementation blueprint. Production gameplay implementation remains prohibited until Phase 4 is approved.

#### Sprint 6 Phase 4: Implementation Blueprint

Status: **APPROVED**

Implementation status: **IMPLEMENTATION PHASE 1 VALIDATED - PHASE 2 NOT STARTED**

Phase 3 is approved. Phase 4 is the implementation blueprint for Maze Arena as the first Games Platform consumer. It converts approved decisions into package ownership, implementation stages, contracts, migration specifications, tests, acceptance targets, and freeze evidence. It does not authorize production code, migrations, API changes, frontend components, or modifications to frozen Sprint 1 through Sprint 5.

Sprint 6 governance status:

| Area | Status |
|---|---|
| Architecture | Complete |
| Design | Complete |
| Implementation Blueprint | Complete |
| Governance | Complete |
| Implementation | Phases 1 and 2 complete and validated; Phase 3 not started |

##### Blueprint Authority

Implementation must follow this order of authority:

1. Frozen Sprint 1 through Sprint 5 public contracts.
2. Approved Sprint 6 Phase 2 Games Platform architecture.
3. Approved Sprint 6 Phase 3 Generator and Solver design.
4. This Phase 4 implementation blueprint.
5. Implementation details that do not contradict the documents above.

When implementation exposes an ambiguity, development stops in the affected phase and the README is amended and approved before code proceeds. A developer may not silently reinterpret collision, uniqueness, difficulty, replay, authority, or version compatibility rules.

##### Architecture Protection Rule

This is a permanent Skill Arena engineering rule.

No Sprint 6 implementation may modify Sprint 1, Sprint 2, Sprint 3, Sprint 4, or Sprint 5 business logic.

An exception is permitted only when **all** of the following are proven before the change is made:

1. The change is platform-generic.
2. The change benefits future games, not only Maze Arena.
3. The change is fully backward compatible.
4. Existing public contracts remain unchanged.
5. Regression tests for every affected frozen sprint pass.
6. The change and its rationale are documented in this README.

Additional controls:

- A Maze-specific requirement is never sufficient justification to change a frozen platform domain.
- Additive implementation behind an existing interface is preferred over altering the interface.
- A generic extension must not introduce Maze fields, Maze branches, Maze terminology, or Maze storage into frozen platform packages.
- Before implementation, the proposed exception must identify the frozen files affected, public contracts reviewed, future-game benefit, compatibility strategy, regression suite, rollback plan, and approving decision.
- After implementation, the phase validation report must include the exact frozen files changed and evidence for every condition above.
- If any condition is false, uncertain, or unproven, the frozen sprint remains unchanged and the behavior belongs inside the Games Platform or Maze module.
- Emergency bug or security fixes follow the frozen sprint maintenance policy and remain separate from feature implementation.

This rule applies during Sprint 6 and remains the default protection model for every future game.

##### Final Project Structure

The production implementation will use the following logical layout:

```text
backend/
  internal/
    arena/                              # Frozen Sprint 5 platform boundary
    realtime/                           # Frozen transport, sessions, lifecycle, storage
    games/
      registry/
        catalog.go                      # Immutable descriptors and factory lookup
        bootstrap.go                    # Explicit composition-root registration
        compatibility.go                # Version tuple compatibility
        manifest.go                     # Manifest parsing and validation
      interfaces/
        module.go                       # Generic runtime game interface
        context.go                      # Match, participant, action, viewer contexts
        action.go                       # Generic action/result envelopes
        state.go                        # Opaque state and transition contracts
        snapshot.go                     # Viewer-safe snapshot contract
        replay.go                       # Generic replay codec contract
        renderer.go                     # Versioned renderer payload contract
        versions.go                     # Complete compatibility tuple
      shared/
        canonical.go                    # Game-neutral canonical encoding
        errors.go                       # Stable platform game error codes
        ids.go                          # Validated game-domain identifiers
        hashes.go                       # Game-neutral hash primitives
        testkit/
          contract.go                   # Required module conformance suite
          fixtures.go                   # Game-neutral contract fixtures
      maze/
        module.json                     # Immutable module descriptor
        module.go                       # RuntimeGame adapter and composition
        engine/
          state.go                      # Authoritative participant state
          action.go                     # Maze action schema
          collision.go                  # Sole cell-occupancy collision authority
          transition.go                 # Validated immutable state transitions
          progress.go                   # Progress and combo inputs
          completion.go                 # Completion and score inputs
          snapshot.go                   # Viewer-safe state projection
        generator/
          service.go                    # Pipeline orchestration
          request.go                    # Internal generation request
          seed.go                       # CSPRNG and domain-separated derivation
          random.go                     # Versioned deterministic random stream
          patterns.go                   # Versioned pattern selection
          geometry.go                   # Candidate geometry construction
          dependency.go                 # Geometry-derived dependency graph
          candidate.go                  # Deterministic candidate ranking
          difficulty.go                 # Requested Difficulty Profile handling
          metadata.go                   # Immutable puzzle metadata
          hashes.go                     # Generation and puzzle hash inputs
          repository.go                 # Puzzle metadata and assignment port
        solver/
          solver.go                     # Independent authoritative solver
          graph.go                      # Graph analysis and topological behavior
          simulation.go                 # Live-rule solution verification
          solution.go                   # Canonical solution representation
        validator/
          validator.go                  # Ordered production validation gates
          geometry.go                   # Structural and geometric checks
          collision.go                  # Collision consistency checks
          dependency.go                 # Dependency consistency checks
          difficulty.go                 # Measured difficulty acceptance
          determinism.go                # Reproducibility and hash checks
          report.go                     # Canonical validation result
        replay/
          codec.go                      # Maze event payload codec
          genesis.go                    # Replay genesis metadata
          reconstruct.go                # Exact-version reconstruction
          verifier.go                   # State, hash, outcome verification
        renderer/
          schema.go                     # Client-safe renderer schema
          projection.go                 # Authoritative state to renderer payload
          visibility.go                 # Player/spectator/reviewer visibility
        api/
          schema.go                     # Maze payload schemas only
          errors.go                     # Mapping Maze errors to generic codes
          manifest.go                   # Public game metadata projection
        shared/
          arrow.go                      # Maze-only immutable arrow type
          board.go                      # Maze-only canonical board type
          cell.go                       # Integer cell and direction primitives
          versions.go                   # Maze version constants
          encoding.go                   # Maze canonical encoding
        tests/
          fixtures/                     # Approved tutorial and collision vectors
          corpus/                       # Qualification corpus metadata
          testdata/                     # Non-secret malformed inputs
    persistence/
      postgres/
        games/                           # Implementations of game repository ports

frontend/
  app/
    games/
      registry/
        catalog.ts                      # Renderer registration by game/version
      interfaces/
        renderer.ts                     # Generic game renderer contract
        events.ts                       # Generic client event envelopes
      shared/
        transport.ts                    # Existing Realtime client adapter
        state.ts                        # Generic connection/sync state
      maze/
        renderer/
          MazeRenderer.tsx              # Renderer composition
          MazeBoard.tsx                 # Board presentation
          MazeArrow.tsx                 # Accessible arrow presentation
        camera/
          useMazeCamera.ts              # Pan, zoom, fit, responsive framing
        animation/
          transitions.ts                # Accepted/blocked presentation
          timing.ts                     # Versioned renderer timing constants
        audio/
          cues.ts                       # Optional accessible sound cues
        accessibility/
          controls.tsx                  # Keyboard and assistive interaction
          announcements.ts              # Non-visual action feedback
        protocol/
          schemas.ts                    # Runtime validation of renderer payloads
          actions.ts                    # Maze action intent creation
        tests/
          fixtures/
          renderer.spec.tsx
          accessibility.spec.tsx
          protocol.spec.ts
```

The physical implementation may combine very small files when that improves clarity, but package ownership and dependency direction may not change.

##### Folder Ownership

| Folder | Owns | Must not own |
|---|---|---|
| `games/registry` | Approved module registration, descriptors, factories, compatibility resolution | Match state, Maze rules, transport |
| `games/interfaces` | Generic runtime contracts and opaque envelopes | Any Maze field or behavior |
| `games/shared` | Canonical game-neutral primitives and conformance tests | Puzzle generation or game-specific rules |
| `games/maze/engine` | Maze actions, collision, state transitions, progress, completion | Networking, persistence, matchmaking, rewards |
| `games/maze/generator` | Seeds, deterministic generation, profiles, patterns, metadata, repository port | Live participant state or WebSockets |
| `games/maze/solver` | Independent solvability and solution classification | Candidate generation decisions |
| `games/maze/validator` | Production acceptance gates and measured difficulty verification | Mutating accepted puzzles |
| `games/maze/replay` | Maze event codec, regeneration, deterministic verification | Generic event storage or signing infrastructure |
| `games/maze/renderer` | Client-safe payload projection and visibility | Client components or gameplay authority |
| `games/maze/api` | Maze-specific payload validation and stable error mapping | HTTP server, auth, sessions, transport |
| `games/maze/shared` | Maze-only immutable value types and encoding | Platform-wide utilities |
| `games/maze/tests` | Maze fixtures, corpus metadata, contract evidence | Production fallback data |
| `persistence/postgres/games` | PostgreSQL implementations of repository ports | Domain policy |
| `frontend/app/games/maze` | Rendering, interaction intent, camera, animation, audio, accessibility | Generation, collision decisions, scoring, completion |

Dependency direction:

```text
Realtime Arena -> Games interfaces <- Maze module
                                  |
                                  v
                            Maze domain packages
                                  |
                                  v
                         Repository/service ports
                                  |
                                  v
                      PostgreSQL/Redis/S3 adapters
```

Maze domain packages may depend on `games/interfaces`, `games/shared`, and Maze-owned packages. They may not import wallet, treasury, authentication handlers, matchmaking internals, notification delivery, tournament repositories, or Admin CRM packages.

##### Module Manifest Blueprint

`games/maze/module.json` is version controlled, validated at startup, and represented by an immutable database registration:

```json
{
  "id": "maze",
  "name": "Maze Arena",
  "gameVersion": "1.0.0",
  "rulesVersion": 1,
  "protocolVersion": 1,
  "replayVersion": 1,
  "rendererVersion": 1,
  "stateSchemaVersion": 1,
  "supports": {
    "practice": true,
    "pvp": true,
    "ranked": true,
    "houseChallenge": true,
    "tournament": true,
    "dailyChallenge": true,
    "replay": true,
    "spectator": true,
    "teams": false,
    "ai": false
  },
  "players": {
    "minimum": 1,
    "maximum": 2
  }
}
```

The manifest contains no secrets, environment routing, active Generator Version, financial policy, or mutable feature flags. Runtime policy selects approved versions and modes through server configuration and database status.

##### Implementation Phases And Approval Gates

Sprint 6 is divided into nine small, sequential implementation phases. Only the current phase may be implemented. Future-phase code, placeholder scaffolding, speculative APIs, and incomplete production paths are prohibited.

Every phase follows:

```text
Approved phase scope
  -> Implementation
  -> Format and build
  -> Focused tests
  -> Applicable Sprint 1-5 regressions
  -> Material documentation reconciliation
  -> Validation report
  -> Review
  -> Explicit approval
  -> Next phase
```

A successful build does not authorize the next phase. Work stops after the validation report until the current phase is reviewed.

###### Implementation Phase 1: Games Platform

Deliver:

- Games Registry.
- Generic Game Interface.
- Module manifest validation and explicit module loading.
- Dependency injection at the application composition root.
- Compatibility and historical version resolution.
- Shared canonical primitives and stable game errors.
- A test-only module proving a second game can register without platform changes.
- Registration and generic contract tests.

Must not implement:

- Maze generation, solving, collision, gameplay, replay, API payloads, Realtime action dispatch, or frontend.

Gate:

- Existing Sprint 5 modules compile and behave unchanged.
- Duplicate, invalid, or incompatible registrations fail startup.
- Historical resolution never falls back to latest.
- The test-only game registers and passes the generic suite without Realtime or Maze knowledge.
- Dependency direction and Architecture Protection Rule checks pass.

###### Implementation Phase 2: Puzzle Service

Deliver:

- Puzzle Service interface and orchestration boundary.
- Generator Version registration.
- Cryptographically secure seed generation and encryption/reference handling.
- Versioned deterministic random stream.
- Puzzle metadata, Difficulty Profile metadata, and canonical hash primitives.
- Repository ports.
- Additive PostgreSQL migrations and repository implementations for version, profile, puzzle, analysis, uniqueness, and assignment metadata required by the service.
- Worker boundary, cancellation, deadlines, idempotency, short transactions, and atomic assignment.

Must not implement:

- Production pattern generation, authoritative solver behavior, live Maze actions, replay finalization, Realtime dispatch, or frontend.

Gate:

- Seed and random-stream vectors are deterministic where required.
- Secret seed material never reaches logs, traces, errors, or client payloads.
- Generator CPU work is structurally outside global locks and database transactions.
- Concurrent uniqueness claims produce one winner.
- Rollback leaves no partial metadata, claim, or assignment.
- PostgreSQL is authoritative in production; in-memory adapters remain test/local only.

###### Implementation Phase 3: Generator

Deliver:

- Versioned pattern selection.
- Deterministic geometry and physical dependency generation.
- Bounded candidate batches and deterministic candidate ranking.
- Production validation pipeline orchestration.
- Measured difficulty feature calculation and requested-profile comparison.
- Generation, puzzle, profile, analysis, solution, and validation hashing inputs.
- Stable rejection reasons and generation observability.

Must not implement:

- Final solver engine shortcuts inside generation.
- Client puzzle generation.
- Realtime gameplay or frontend rendering.

Gate:

- Identical complete inputs produce byte-identical canonical candidates and hashes.
- Worker completion order cannot affect selection.
- Server load never reduces complexity or changes requested difficulty.
- Every candidate reaches all required validation gates; no warning-only acceptance exists.
- Generator tests prove structural bounds, geometry invariants, pattern behavior, cancellation, and deterministic ranking.

###### Implementation Phase 4: Solver

Deliver:

- Independent authoritative graph solver.
- Live-rule simulation.
- Deadlock detection.
- Isolated-arrow policy enforcement.
- Unique/multiple valid completion classification.
- Canonical solution and minimum-action calculation.
- Difficulty verification independent from generator intent.
- Puzzle uniqueness verification against persisted seed and puzzle claims.

Must not implement:

- Generator-specific candidate construction in solver packages.
- Alternative collision rules.
- Gameplay transport or frontend.

Gate:

- All approved tutorial and collision fixtures pass.
- Every known deadlock, malformed board, and impossible collision is rejected.
- Canonical solution reproduces the Maze engine's approved collision rules.
- Requested and measured difficulty mismatch is rejected.
- Accepted puzzles are solvable and uniqueness claims are valid.
- Solver remains independently testable from generator selection logic.

###### Implementation Phase 5: Replay

Deliver:

- Replay genesis metadata.
- Maze replay event codec.
- Generation, validation, event-root, state, outcome, and final replay hashes.
- Exact-version puzzle reconstruction.
- Ordered action replay.
- Replay signature integration through frozen Realtime infrastructure.
- Replay verification and under-review failure path.

Must not implement:

- A separate Maze event store, object-storage service, signature system, or public replay API.
- Full-board canonical replay storage when deterministic reconstruction is possible.

Gate:

- Exact metadata reconstructs identical puzzle bytes and hash.
- Ordered events reconstruct participant state and authoritative outcome.
- Seed, action, timing, version, hash, or signature corruption fails closed.
- Historical version resolution works without fallback.
- Replay failure enters review and never mutates historical evidence.

###### Implementation Phase 6: Maze Engine

Deliver:

- Sole integer cell-occupancy collision authority.
- Maze action schema and validation.
- Immutable participant state transitions.
- Blocked and successful move results.
- Progress and combo tracking.
- Completion detection.
- Maze scoring inputs.
- Authoritative server timing inputs.
- Viewer-safe renderer projection.
- Handcrafted tutorial fixtures 1 through 5.

Must not implement:

- Networking, authentication, matchmaking, sessions, presence, reconnect, Wallet settlement, progression rewards, tournament logic, or client authority.

Gate:

- Every arrow moves only in its fixed direction.
- Blocked actions preserve state and return authoritative collision presentation.
- Only arrows that completely exit are removed.
- State versions advance only on accepted transitions.
- Progress, completion, scoring inputs, and timing are server-derived.
- The engine passes unit, property, fuzz, race, and approved reference collision tests without a frontend.

###### Implementation Phase 7: Realtime Integration

Deliver:

- Read-only game catalog projection.
- Generic `game.action` and `game.sync.request` dispatch through the frozen gateway.
- Maze registration through the Games Registry.
- Existing Realtime queue, ready, match, leave, heartbeat, reconnect, event, and replay routes carrying generic game payloads.
- Practice and tutorial lifecycle.
- PvP and Ranked shared-puzzle assignment with independent participant states.
- Progress synchronization, completion, winner, timeout, disconnect, forfeit, reconnect, and replay-ready events.
- House, Daily, and Tournament assignment policy adapters without owning those platform lifecycles.

Must not implement:

- Maze-specific WebSockets, queues, sessions, presence, replay storage, matchmaking, or settlement.
- Admin CRM controls.

Gate:

- Realtime packages contain no Maze import, field, switch, or rule.
- Duplicate action returns the original receipt.
- State conflict and sequence-gap recovery are deterministic.
- PvP players receive identical puzzle and validation hashes with independent states.
- Practice and House assignments are fresh and one-use.
- Daily reuse is explicitly window-scoped.
- Tournament matches receive separate claims while participants in one match share a puzzle.
- Authentication, rate limits, presence, reconnect, lifecycle, and settlement remain owned by frozen services.

###### Implementation Phase 8: Frontend

Deliver:

- Generic renderer registry.
- Maze renderer and board.
- Responsive camera, fit, pan, and zoom.
- Accepted move acceleration, glide, exit, and removal after leaving the board.
- Blocked move approach, impact, shake/bounce, and return.
- Audio and effects with preferences.
- Loading, preparation, ready, reconnecting, stale state, completion, defeat, victory, replay, and failure states.
- Desktop, tablet, mobile, keyboard, screen-reader, reduced-motion, and reduced-audio support.
- Protocol runtime validation and frontend tests.

Must not implement:

- Puzzle generation, collision decisions, scoring, completion, winner, seed selection, difficulty selection, or replay verification on the client.

Gate:

- Client sends intent only.
- Renderer follows authoritative snapshots and presentation events.
- Desktop, tablet, and mobile evidence passes.
- Chromium, Firefox, and WebKit critical journeys pass.
- Accessibility, reduced-motion, protocol, reconnect, and error-state tests pass.
- No placeholder UI, mock production data, or sample-app production dependency remains.

###### Implementation Phase 9: Production Hardening

Deliver:

- At least 100,000-candidate qualification corpus.
- Load, stress, reconnect-storm, and soak tests.
- Security and zero-trust action testing.
- Replay certification across supported build targets.
- Performance profiling and tuning without changing difficulty.
- Failure testing for PostgreSQL, Redis, object storage, workers, generation, and network loss.
- Observability dashboards, alerts, operational runbooks, rollback proof, and full Sprint 1 through Sprint 5 regression evidence.
- Final Sprint 6 production and freeze report.

Must not implement:

- New game rules, modes, platform features, frontend pages, or architecture redesign.

Gate:

- Every Sprint 6 Definition of Done and Freeze Criteria item below is demonstrated.
- All measurable performance and capacity targets pass on documented reference infrastructure.
- No unresolved Critical or High defect remains.
- Any accepted Medium or Low defect has an owner, impact analysis, mitigation, and explicit release decision.
- Security, replay, uniqueness, authority, failure recovery, and frozen-sprint regression evidence pass.

##### Mandatory Implementation Phase Validation Report

Every implementation phase ends with a short, permanent validation record in this README. The report must contain:

| Field | Required evidence |
|---|---|
| Phase | Phase number, name, scope, start date, and validation date |
| Commit | Commit SHA under review; no freeze tag until Sprint 6 completion |
| Files changed | Complete categorized file list, including generated migrations or artifacts |
| APIs | Every API/event added, extended, or confirmed unchanged |
| Database | Migrations, indexes, constraints, repository changes, and rollback result |
| Tests added | Test files and behaviors introduced |
| Tests passed | Exact commands, counts, coverage, environment, and relevant output |
| Build verification | Formatting, Go build/vet/race, TypeScript, lint, frontend build, and applicable tools |
| Performance impact | Before/after measurements or `not applicable` with evidence |
| Security impact | Authority, authentication, authorization, secrets, rate limit, integrity, dependency, and threat review |
| Frozen sprint impact | Files touched, contracts reviewed, regressions run, and Architecture Protection Rule result |
| Documentation | Material documentation updated, or explicit confirmation that no architecture, public contract, operational behavior, or developer workflow changed |
| Remaining work | Work explicitly deferred to later approved phases |
| Risks discovered | Severity, impact, owner, mitigation, and decision |
| Recommendation | `APPROVE IMPLEMENTATION PHASE N` or `DO NOT APPROVE IMPLEMENTATION PHASE N` |

Report rules:

- Use actual command output and measured evidence; do not write "tests pass" without identifying the tests.
- `Not applicable` requires a reason.
- A remaining item that belongs to the current phase prevents approval.
- Future-phase work is listed but not implemented.
- Any unexpected frozen-sprint change automatically requires Architecture Protection Rule evidence.
- The validation report remains mandatory even when no other project documentation requires an update.
- The next implementation phase remains **NOT STARTED** until explicit approval is given.
- A phase approval is not a Sprint 6 freeze and does not create a freeze tag.

##### Release Engineering Rule

Every implementation phase must satisfy all applicable requirements below before it can be approved.

Functional:

- The approved phase scope is fully implemented.
- Every acceptance criterion is met.
- No future-phase feature or placeholder implementation is included.
- The implementation does not contradict the canonical product and architecture contracts.

Build:

- Backend formatting and builds pass.
- Frontend formatting and builds pass when the phase affects frontend code; otherwise the existing frontend production build must remain green.
- No new compiler, linter, vet, runtime, or deprecation warning is introduced.
- Production dependency and vulnerability audits are clean, or any external advisory has an explicit documented release decision.

Testing:

- Unit tests pass.
- Integration tests pass.
- Existing applicable regression tests pass.
- Tests are added for every new behavior, error, boundary, and security control.
- Race, fuzz, property, browser, or end-to-end suites run when required by the phase contract.

Regression:

- All previously frozen Sprint 1 through Sprint 5 regression suites pass.
- No public API or event contract changes unless the change was explicitly documented and approved before implementation.
- No architectural boundary is violated. Maze logic may not enter Realtime Arena, Arena Hub, Identity, Financial Platform, Admin CRM, or another frozen platform domain.
- Any permitted generic frozen-sprint extension satisfies and records every Architecture Protection Rule condition.

Performance:

- No regression against documented targets is accepted silently.
- Relevant latency is measured.
- Memory and allocation impact are measured for CPU-intensive, stateful, realtime, replay, or high-volume paths.
- `Not applicable` requires a documented reason.

Security:

- No client-authoritative gameplay is introduced.
- Replay and event integrity remain preserved.
- Authentication and authorization are verified at every affected boundary.
- Inputs, outputs, errors, secrets, dependencies, and abuse controls are reviewed.
- The Architecture Protection Rule passes.

Documentation and evidence:

- The phase validation report includes a summary.
- Files changed are listed.
- Public contracts changed or confirmed unchanged are listed.
- Database changes or confirmation of no database change are listed.
- Test evidence includes exact commands and results.
- Performance evidence includes measurements or a justified `not applicable`.
- Security evidence identifies the controls reviewed.
- Known limitations, deferred later-phase work, and discovered risks are explicit.
- Project documentation is updated whenever the phase changes architecture, public contracts, operational behavior, or developer workflows.
- Purely internal refactoring or test additions do not require unrelated product-document rewrites.
- When no material documentation update is required, the validation report says so and explains why.

Freeze decision:

- Every phase ends with exactly one recommendation: `APPROVED` or `CHANGES REQUIRED`.
- `APPROVED` is permitted only when no current-phase requirement remains incomplete.
- `CHANGES REQUIRED` lists the exact corrections needed.
- Terms such as partial, mostly complete, nearly ready, or conditionally approved are not valid phase decisions.
- The next phase may not begin until the current phase is explicitly approved by the product owner.

##### Implementation Phase 1 Validation Report

Phase: **Implementation Phase 1 - Games Platform**

Scope: Games Registry, generic Game Interface, manifest-backed registration, exact version resolution, dependency injection, Sprint 5 compatibility, and associated tests.

Implementation date: **2026-07-28**

Validation date: **2026-07-28**

Implementation commit: `c36a74705b03780f4af6e2c74625604ec7c8e248`

Freeze tag: none. A phase approval is not a Sprint 6 freeze.

###### Summary

Implementation Phase 1 establishes the generic Games Platform without adding Maze gameplay.

Delivered:

- Versioned generic game descriptors and capability contracts.
- Generic runtime interfaces for match initialization, participant initialization, generation, actions, transitions, snapshots, completion, outcomes, replay, and cleanup.
- Immutable, concurrency-safe factory registry.
- Exact historical version resolution with no fallback to latest.
- One active new-match version per game.
- Manifest and compatibility tuple validation.
- Domain-separated, length-prefixed canonical hashing primitive.
- Explicit production bootstrap containing Maze only.
- Backward-compatible adapter to the frozen Sprint 5 `arena/registry.Registry`.
- Dependency injection through the API composition root and store options.
- Test-only second module inside registry tests.
- Factory panic containment and descriptor mismatch rejection.

Not delivered because it belongs to later phases:

- Puzzle Service.
- Generator, solver, validator, and Difficulty Analyzer.
- Maze collision or action engine.
- Replay implementation.
- Generic Realtime game-action dispatch.
- Frontend renderer.
- Database migrations.

###### Files Changed

Canonical documentation:

- `README.md`.

Production composition and backward-compatible injection:

- `backend/cmd/api/main.go`.
- `backend/internal/db/db.go`.
- `backend/internal/db/realtime.go`.

Games Platform interfaces:

- `backend/internal/games/interfaces/action.go`.
- `backend/internal/games/interfaces/context.go`.
- `backend/internal/games/interfaces/module.go`.
- `backend/internal/games/interfaces/renderer.go`.
- `backend/internal/games/interfaces/replay.go`.
- `backend/internal/games/interfaces/snapshot.go`.
- `backend/internal/games/interfaces/state.go`.
- `backend/internal/games/interfaces/versions.go`.

Games Registry:

- `backend/internal/games/registry/bootstrap.go`.
- `backend/internal/games/registry/legacy.go`.
- `backend/internal/games/registry/manifest.go`.
- `backend/internal/games/registry/registry.go`.
- `backend/internal/games/registry/registry_test.go`.

Shared primitives and contract test support:

- `backend/internal/games/shared/errors.go`.
- `backend/internal/games/shared/hashes.go`.
- `backend/internal/games/shared/hashes_test.go`.
- `backend/internal/games/shared/testkit/contract.go`.

Store integration test:

- `backend/internal/db/games_registry_test.go`.

Frozen Sprint 5 test compatibility correction:

- `frontend/app/lib/realtime.test.ts`.

###### Public Contracts

Public REST APIs: **unchanged**.

Realtime WebSocket protocol: **unchanged**.

Database schema: **unchanged**.

Player and Admin CRM frontend contracts: **unchanged**.

Existing Sprint 5 `Store.ArenaRegistry()` contract: **unchanged**.

Additive internal contracts:

- `interfaces.Module`.
- `interfaces.RuntimeGame`.
- Complete `interfaces.Versions`.
- `interfaces.Descriptor`.
- Generic contexts, actions, states, transitions, snapshots, replay metadata, outcomes, and cleanup instructions.
- `games/registry.Registry`.
- `db.Options.GamesRegistry`.
- `Store.GamesRegistry()`.

The current Maze module is loaded through a compatibility registration. It has not been converted into the new runtime gameplay implementation.

###### Database

Migrations added: **none**.

Tables, indexes, constraints, and foreign keys changed: **none**.

Persistence behavior changed: **none**.

PostgreSQL repository work remains reserved for Implementation Phase 2.

###### Tests Added

Eight new top-level tests plus nine invalid-descriptor subtests cover:

- Active and historical version registration.
- Exact version resolution.
- No fallback to latest.
- Missing game versus missing version errors.
- Duplicate registration.
- Multiple active-version rejection.
- Factory descriptor mismatch.
- Factory panic containment.
- Invalid IDs, versions, status, new-match status, player ranges, renderer keys, hashes, and duplicate modes.
- Descriptor immutability.
- One hundred concurrent registry reads.
- Production bootstrap contents.
- Test-module exclusion from production.
- Sprint 5 Arena registry compatibility.
- Canonical hash determinism, domain separation, and field-boundary safety.
- Store dependency injection while retaining the frozen `ArenaRegistry()` contract.

###### Backend Verification

Environment:

```text
go version go1.26.5 windows/amd64
```

Formatting:

```text
gofmt -w <all changed Go files>
gofmt -l <all changed Go files>
```

Result: passed; no changed Go file remained unformatted.

Full backend tests:

```text
go test -count=1 ./...
```

Result:

- Exit code `0`.
- Eighteen test-bearing backend packages passed.
- All 33 root packages compiled.
- `internal/db` passed in `141.538s`.
- `internal/realtime` passed in `12.958s`.
- New `internal/games/registry`, `internal/games/shared`, and store-injection tests passed.

Static and build verification:

```text
go vet ./...
go build ./...
go mod verify
```

Result:

- `go vet`: exit code `0`.
- `go build`: exit code `0`.
- Module verification: `all modules verified`.

Race verification:

```text
CGO_ENABLED=1 go test -race -count=1 ./internal/games/...
CGO_ENABLED=1 go test -race -count=1 ./internal/db -run '^TestStoreUsesInjectedGamesRegistryWithoutChangingArenaContract$'
```

Result:

- Games packages passed.
- Store injection passed in `4.950s`.
- Portable GCC `16.1.0` was used outside the repository.
- The downloaded w64devkit `v2.9.0` archive matched its published SHA-256 `bff1d13fc2718eebd93548cf37f8d0332d925458d5e99506cff8f46eb5a9de5a`.

Coverage:

```text
go test -count=1 -cover ./internal/games/registry ./internal/games/shared
```

Result:

- Games Registry: `83.0%` statements.
- Shared canonical hashing: `100.0%` statements.

###### Frontend And Frozen-Sprint Regression Verification

Player Platform:

```text
npm run lint
npm run typecheck
npm test -- --run
npm run build
npm audit --omit=dev
```

Result:

- ESLint passed with zero warnings.
- TypeScript passed.
- Vitest: 5 files and 9 tests passed.
- Next.js `16.2.11` production build compiled 23 routes.
- Production dependency audit found 0 vulnerabilities.

Admin CRM:

```text
npm run lint
npm run typecheck
npm test -- --run
npm run build
npm audit --omit=dev
```

Result:

- ESLint passed with zero warnings.
- TypeScript passed.
- Vitest: 2 files and 5 tests passed.
- Next.js `16.2.11` production build compiled 13 routes.
- Production dependency audit found 0 vulnerabilities.

The Player Platform typecheck initially exposed a pre-existing use of `Array.prototype.at` in `frontend/app/lib/realtime.test.ts`, which was outside the configured TypeScript library target. It was replaced with equivalent indexed access. This changed test syntax only; Realtime production behavior and public contracts are unchanged.

###### Performance Evidence

Command:

```text
go test -run '^$' -bench BenchmarkRegistryResolve -benchmem -count=3 ./internal/games/registry
```

Environment:

- Windows AMD64.
- Intel Core i7-8665U at 1.90 GHz.

Results:

```text
49321 ns/op   1008 B/op   6 allocs/op
11943 ns/op   1008 B/op   6 allocs/op
 7708 ns/op   1008 B/op   6 allocs/op
```

The first process-warmup sample was slower. Warm samples resolved an exact module version in approximately `7.7-11.9 us`. Registry resolution is not on the per-action gameplay path. No documented Phase 1 latency target regressed.

###### Security Evidence

Verified:

- Module IDs, complete versions, status, player ranges, modes, renderer keys, and manifest hashes are validated.
- Duplicate game/version registrations fail.
- More than one active new-match version for a game fails.
- Historical resolution is exact and never substitutes latest.
- Revoked and unavailable versions have distinct fail-closed errors.
- Factory panics become startup/resolution errors.
- Factory descriptors must exactly match immutable registered descriptors.
- Descriptor slices are defensively copied.
- Registry reads are mutex-protected and passed race detection.
- Production bootstrap registers Maze only.
- The second game used for modularity proof exists only in test code.
- No seed, gameplay authority, client state, wallet behavior, authentication behavior, or replay signature logic was introduced.
- No hardcoded secret, placeholder, TODO, or sample implementation exists in the new Games Platform packages.

Supply-chain verification:

```text
govulncheck -show verbose ./...
```

Result:

- Zero reachable vulnerabilities.
- Zero vulnerabilities in imported packages.
- One module-only advisory, `GO-2026-5932`, applies to the unmaintained `golang.org/x/crypto/openpgp` package.
- Skill Arena does not import or call `openpgp`; no vulnerable symbol is reachable.
- The advisory remains tracked as dependency hygiene and is not a Phase 1 exploitable path.

###### Architecture Protection Rule

Frozen files changed:

- `backend/cmd/api/main.go`.
- `backend/internal/db/db.go`.
- `backend/internal/db/realtime.go`.
- `frontend/app/lib/realtime.test.ts`.

Rule evidence:

1. Platform-generic: registry injection and compatibility benefit every future game.
2. Future-game benefit: production composition can add another module without a Maze branch in Store or Realtime.
3. Backward compatibility: `Store.ArenaRegistry()` remains available and all Sprint 5 tests pass.
4. Public contracts unchanged: no HTTP, WebSocket, model, database, auth, financial, or CRM contract changed.
5. Regression tests: full Go, Player Platform, and Admin CRM gates pass.
6. Rationale documented: this report records every frozen file and reason.

No frozen Sprint 1 through Sprint 5 business rule was changed.

Rollback:

- Revert implementation commit `c36a74705b03780f4af6e2c74625604ec7c8e248`.
- No database migration, persisted state transformation, or public protocol rollback is required.

###### Remaining Work

At the Phase 1 validation point, Implementation Phase 2 was **NOT STARTED** and owned:

- Puzzle Service.
- Generator Version persistence.
- Secure generation seed lifecycle.
- Puzzle metadata persistence.
- PostgreSQL puzzle repositories and migrations.

At the Phase 1 validation point, Implementation Phases 3 through 9 were also **NOT STARTED**.

###### Known Limitations And Risks

- `RuntimeGame` is a contract only in Phase 1. Maze runtime implementation intentionally belongs to later phases.
- The current legacy Maze module remains available through the Sprint 5 compatibility adapter until its approved replacement phase.
- The Games Registry is process-local immutable configuration. Database-backed version records belong to Phase 2.
- Registry benchmark variance includes Windows process and CPU warmup; later load testing must run on the Release 1.0 reference environment.
- The unreachable `openpgp` module advisory remains tracked until the parent dependency can remove it without destabilizing frozen security code.

No limitation above represents incomplete Implementation Phase 1 scope.

###### Phase Decision

**APPROVED**

The product owner subsequently authorized Implementation Phase 2; its validation report follows.

##### Implementation Phase 2 Validation Report

Phase: **Implementation Phase 2 - Puzzle Service**

Scope: Puzzle Service orchestration, generator lifecycle boundary, complete generator-version identity, cryptographically secure seed lifecycle, deterministic random stream, Difficulty Profile and puzzle metadata, repository ports, normalized PostgreSQL persistence, atomic uniqueness, assignment, tests, and production configuration.

Implementation date: **2026-07-28**

Validation date: **2026-07-28**

Implementation commit: `ac267e0d189a3911cc854a4301c1380dfab3243e`

Freeze tag: none. A phase approval is not a Sprint 6 freeze.

###### Summary

Implementation Phase 2 establishes the production Puzzle Service foundation without implementing a puzzle-generation algorithm or Maze gameplay.

Delivered:

- Maze-owned Puzzle Service behind repository ports.
- Exact immutable generator identity covering generator, seed format, random stream, patterns, geometry, scoring, policy, solver, validator, analyzer, difficulty schema, and canonical encoding versions.
- Generator-version lifecycle: `qualification`, `active`, `replay_only`, `retired`, and `revoked`.
- Integer/fixed-point Difficulty Profile metadata without floating-point authority.
- 256-bit cryptographic entropy and domain-separated HMAC-SHA-256 seed derivation.
- AES-256-GCM seed encryption at rest with authenticated metadata.
- Separate derivation and encryption keys with production startup validation.
- Non-secret seed hashes and opaque seed references for uniqueness and audit.
- Versioned HMAC-SHA-256 counter stream with rejection sampling and a fixed test vector.
- Idempotent puzzle preparation through a hashed request key.
- CPU processor boundary that runs outside repository transactions.
- Atomic analysis, uniqueness claim, assignment, and puzzle finalization.
- Concurrent retry handling that returns the original committed assignment.
- PostgreSQL production adapter and memory-only local/test adapter.
- Additive migration `008_games_puzzle_service.sql`.
- Dependency injection through `db.Options` and production store composition.

Not delivered because it belongs to later phases:

- Pattern selection or geometry generation.
- Dependency graph construction.
- Solver, deadlock detection, or solution uniqueness.
- Measured difficulty calculation or calibration.
- Replay generation, reconstruction, or verification.
- Maze action validation, scoring, or completion.
- Realtime gameplay dispatch, PvP synchronization, or client rendering.

###### Files Changed

Configuration and composition:

- `backend/internal/config/config.go`.
- `backend/internal/config/config_test.go`.
- `backend/internal/db/db.go`.
- `docker-compose.yml`.

Puzzle Service:

- `backend/internal/games/maze/generator/hash.go`.
- `backend/internal/games/maze/generator/memory_repository.go`.
- `backend/internal/games/maze/generator/pipeline.go`.
- `backend/internal/games/maze/generator/repository.go`.
- `backend/internal/games/maze/generator/seed.go`.
- `backend/internal/games/maze/generator/service.go`.
- `backend/internal/games/maze/generator/stream.go`.
- `backend/internal/games/maze/generator/types.go`.

PostgreSQL:

- `backend/internal/persistence/postgres/games/puzzle_repository.go`.
- `backend/migrations/008_games_puzzle_service.sql`.
- `backend/migrations/embed.go`.

Tests:

- `backend/internal/games/maze/generator/service_test.go`.
- `backend/internal/db/games_puzzle_postgres_integration_test.go`.

Canonical documentation:

- `README.md`.

###### Public Contracts

Public REST APIs: **unchanged**.

Realtime WebSocket protocol: **unchanged**.

Player Platform and Admin CRM contracts: **unchanged**.

Frozen Sprint 1 through Sprint 5 business behavior: **unchanged**.

Additive internal contracts:

- `generator.Repository`.
- `generator.Service`.
- `generator.Processor`.
- `generator.VersionKey` and `generator.GeneratorVersion`.
- `generator.DifficultyProfile` and `generator.DifficultyAnalysis`.
- `generator.PuzzleMetadata`, `generator.Finalization`, and `generator.Assignment`.
- `generator.SeedVault`, `generator.SeedScope`, and `generator.RandomStream`.
- `db.Options.PuzzleRepository`.
- `Store.GamesPuzzleService()`.

No route, event, or client payload exposes seed ciphertext, nonce, or plaintext seed material.

###### Database

Migration:

```text
008_games_puzzle_service
```

Tables:

- `game_generator_versions`.
- `game_difficulty_profiles`.
- `game_puzzles`.
- `game_difficulty_analyses`.
- `game_puzzle_uniqueness_claims`.
- `game_puzzle_assignments`.

Controls:

- Positive immutable version components.
- Bounded lifecycle and reuse-policy values.
- Canonical SHA-256 digest constraints.
- No plaintext seed column.
- Encrypted seed ciphertext and nonce requirements.
- Unique request and seed references.
- Restrictive foreign keys.
- One accepted analysis per puzzle/analyzer version.
- Partial unique indexes for one-use seed and puzzle hashes.
- One assignment per scope.
- Transactional migration checksum validation.
- Serializable finalization transaction.

PostgreSQL is selected whenever the store uses a PostgreSQL URL. The memory repository is used only when PostgreSQL is disabled for local development or injected by tests.

###### Tests Added

Coverage includes:

- Version and Difficulty Profile validation.
- Secure seed creation, encryption, recovery, and authenticated-data binding.
- Ciphertext tampering rejection.
- Scope/domain separation.
- No secret fields in JSON metadata.
- Exact deterministic random-stream vector.
- Random-stream chunk independence and domain separation.
- Concurrent one-use puzzle-hash claims with one winner.
- Rollback with no partial metadata.
- Cancellation and replay-only version rejection.
- Preparation idempotency.
- Sequential and concurrent work idempotency.
- Processor execution outside repository transactions.
- PostgreSQL migration, persistence, sealed-seed handling, atomic claims, and rollback record counts.

###### Backend Verification

Environment:

```text
go version go1.26.5 windows/amd64
PostgreSQL 17.10 isolated validation cluster
```

Formatting, static analysis, build, and modules:

```text
gofmt -w <changed Go files>
go vet ./...
go build ./...
go mod verify
```

Result:

- All changed Go files are formatted.
- `go vet`: exit code `0`.
- `go build`: exit code `0`.
- Module verification: `all modules verified`.

Full backend regression:

```text
go test ./... -count=1
```

Result:

- Exit code `0`.
- All backend packages compiled.
- `internal/db` passed in `142.428s`.
- `internal/realtime` passed in `16.363s`.
- New Puzzle Service tests passed.

PostgreSQL integration:

Each integration suite ran against a freshly recreated isolated PostgreSQL 17 database:

```text
TestPostgresAuthenticationRepository                         PASS
TestPostgresArenaHubRepository                               PASS
TestPostgresFinancialRepository                              PASS
TestPostgresAdminCRMRepository                               PASS
TestPostgresRealtimeRepository                               PASS
TestPostgresGamesPuzzleServicePersistenceAndAtomicClaims     PASS
```

The new Puzzle Service PostgreSQL test passed in `0.60s` without race instrumentation and `5.669s` with race instrumentation.

Race verification:

```text
CGO_ENABLED=1 go test -race ./internal/games/maze/generator -count=5
CGO_ENABLED=1 go test -race ./internal/db -run '^TestPostgresGamesPuzzleServicePersistenceAndAtomicClaims$'
```

Result: passed with no race report.

An attempted package-wide `go test -race ./internal/db` exceeded Go's `10m` test timeout while the legacy `TestConcurrentLaunchLoadPaths` was still executing hundreds of CPU-heavy prototype puzzle generations. It produced no race report before timeout. The same test passes in the normal full regression. Phase 2 changed neither that legacy generator nor the load test.

Coverage:

```text
go test ./internal/games/maze/generator -cover -count=1
```

Result: `71.1%` statement coverage.

###### Frontend And Frozen-Sprint Regression Verification

Player Platform:

```text
npm test
npm run lint
npm run typecheck
npm run build
npm run test:e2e
npm audit --audit-level=high --omit=dev
```

Result:

- Vitest: 5 files and 9 tests passed.
- ESLint passed with zero warnings.
- TypeScript passed.
- Next.js `16.2.11` production build compiled 23 routes.
- Playwright: 21 tests passed across desktop, tablet, and mobile in `3.9m`.
- Production dependency audit found 0 vulnerabilities.

Admin CRM:

```text
npm test
npm run lint
npm run typecheck
npm run build
npm run test:e2e
npm audit --audit-level=high --omit=dev
```

Result:

- Vitest: 2 files and 5 tests passed.
- ESLint passed with zero warnings.
- TypeScript passed.
- Next.js `16.2.11` production build compiled 13 routes.
- Playwright: 3 tests passed across desktop, tablet, and mobile in `1.0m`.
- Production dependency audit found 0 vulnerabilities.

No Player Platform or Admin CRM source file changed in Phase 2.

###### Performance Evidence

Command:

```text
go test ./internal/games/maze/generator -run '^$' -bench BenchmarkServicePrepare -benchmem -count=5
```

Environment:

- Windows AMD64.
- Intel Core i7-8665U at 1.90 GHz.
- In-memory repository, isolating service and cryptographic overhead.

Results:

```text
19751 ns/op   8147 B/op   239 allocs/op
20182 ns/op   8213 B/op   239 allocs/op
20046 ns/op   8214 B/op   239 allocs/op
21076 ns/op   8181 B/op   239 allocs/op
19956 ns/op   8213 B/op   239 allocs/op
```

Puzzle preparation averages approximately `20 us` before PostgreSQL/network latency. Production generation and solver latency remain Phase 3 and Phase 4 acceptance work because no algorithm exists in Phase 2.

###### Security Evidence

Verified:

- Seed entropy comes from `crypto/rand`.
- Seed derivation is HMAC-SHA-256 with length-prefixed domain and scope fields.
- Practice/House/Training seeds require participant scope.
- PvP and tournament seeds can remain match-scoped and shared.
- Seeds are encrypted with AES-256-GCM before persistence.
- Encryption authenticated data binds puzzle, game, mode, profile, and exact generator version.
- Derivation and encryption keys must be separate and at least 32 characters in production.
- Development-key markers fail production startup validation.
- Seed ciphertext and nonce are excluded from JSON.
- Seed material is never formatted into errors or logs.
- Deterministic streams are versioned and domain-separated.
- Bounded integer selection uses rejection sampling.
- Exact versions never fall back to latest.
- Replay-only, retired, or revoked versions cannot create a new puzzle.
- Idempotency keys are hashed before persistence.
- One-use seed and puzzle hashes are enforced by PostgreSQL unique indexes.
- Finalization uses a serializable short transaction.
- Processor CPU work occurs before the finalization transaction.
- No client-authoritative gameplay, route, renderer, solver, wallet, or reward behavior was introduced.
- New source contains no TODO, placeholder, fake provider, sample implementation, `math/rand`, or hardcoded production secret.

###### Architecture Protection Rule

Frozen implementation files changed:

- `backend/internal/config/config.go`.
- `backend/internal/config/config_test.go`.
- `backend/internal/db/db.go`.
- `backend/migrations/embed.go`.
- `docker-compose.yml`.

Rule evidence:

1. Platform-generic: secure puzzle keys, repository injection, migration startup, and service composition support every future deterministic game.
2. Future-game benefit: the store depends on a repository port and does not contain Maze generation or action rules.
3. Backward compatibility: existing Store, REST, Realtime, player, and admin contracts remain unchanged.
4. Public contracts unchanged: no HTTP route, WebSocket event, authentication, financial, CRM, or frontend contract changed.
5. Regression tests: full Go, all isolated PostgreSQL repositories, Player Platform, Admin CRM, and browser suites pass.
6. Rationale documented: production requires normalized PostgreSQL metadata and distinct encryption material; local development retains the injected memory adapter.

No frozen Sprint 1 through Sprint 5 business rule was changed.

Rollback:

- Revert implementation commit `ac267e0d189a3911cc854a4301c1380dfab3243e`.
- Migration `008` is additive. Production rollback leaves its unreferenced tables in place unless an approved database rollback procedure removes them.
- No existing table, row, API, event, or client contract requires data transformation.

###### Known Limitations And Risks

- The Phase 2 `Processor` is a production orchestration contract; no generator, solver, validator, analyzer, or gameplay implementation exists yet.
- Historical seed decryption requires retention of the configured encryption key. Key rotation and key-ring/secret-manager resolution must be completed before a key is rotated; no production key has been rotated in this phase.
- All PostgreSQL integration tests pass on fresh isolated databases. Running every PostgreSQL integration test against one shared database exposes an existing harness collision because each frozen test migrates the same local legacy snapshot before its own cleanup.
- The Admin CRM PostgreSQL audit-chain test failed once because of existing timestamp normalization behavior and passed on immediate isolated retry. No Admin CRM code changed.
- Package-wide database race instrumentation exceeds the existing ten-minute test timeout in the legacy high-complexity puzzle load test. Targeted Phase 2 race tests pass.
- Production pattern generation, solver latency, difficulty calibration, and replay determinism remain explicitly deferred to their approved phases.

No limitation above represents missing Implementation Phase 2 scope.

###### Remaining Work

Implementation Phase 3 remains **NOT STARTED** and owns:

- Versioned pattern selection.
- Deterministic geometry and physical dependency generation.
- Bounded candidate generation and deterministic ranking.
- Production validation pipeline orchestration.
- Measured difficulty feature calculation.
- Canonical generation, puzzle, validation, solution, and analysis hash inputs.

Implementation Phases 4 through 9 remain **NOT STARTED**.

###### Phase Decision

**APPROVED**

Implementation Phase 3 must not begin until the product owner explicitly approves this validation report.

##### Implementation Phase 3 Validation Report

Phase: **Implementation Phase 3 - Puzzle Generator**

Scope: deterministic candidate generation, versioned pattern selection, integer geometry, physically derived dependency graphs, structural validation, measured difficulty calibration, canonical hashing, deterministic qualification, and Puzzle Service integration.

Implementation date: **2026-07-29**

Validation date: **2026-07-29**

Implementation commit: `6182eaac3b4fd155b844e5d5f682b9e496ee5cfa`

Freeze tag: none. A phase approval is not a Sprint 6 freeze.

###### Summary

Implementation Phase 3 delivers the production generator pipeline authorized by the approved blueprint without implementing the Phase 4 solver or any Maze gameplay.

Delivered:

- Integer cell geometry with canonical arrow identifiers, ordered paths, fixed directions, bounds, overlap, path-continuity, and arrowhead validation.
- Versioned catalogue containing braid, spiral, maze rows, rings, mosaic, piton, diagonal weave, and rays pattern inputs.
- Deterministic weighted pattern selection with optional Difficulty Profile bias.
- Domain-separated random streams for dimensions, line count, pattern selection, and candidate placement.
- Fixed, bounded candidate batches with bounded placement attempts and concurrency.
- Candidate result slots indexed independently of goroutine completion order.
- Reverse dependency-safe placement that constructs physical escape-ray relationships.
- Dependency graphs derived exclusively from board occupancy rather than trusted generator declarations.
- Structural graph validation for complete node coverage, valid references, duplicate edges, cycles, an initial open arrow, and competitive isolated arrows.
- Integer-only measured difficulty covering line count, occupancy, density, dependency depth, branching, cross dependencies, blocked choices, path length, directional diversity, visual complexity, complexity score, and expected solve time.
- Exact Difficulty Profile calibration with stable rejection codes and no fallback to an easier profile.
- Canonical binary board and graph encodings.
- Canonical Difficulty Profile hashing independent of its stored hash field.
- Domain-separated generation, puzzle, analysis, validation, solution, and final-state hash inputs.
- Mandatory independent-verifier contract. Missing, rejecting, or malformed verifier output fails closed.
- Deterministic candidate ranking and selection.
- Ordered candidate observations suitable for metrics without exposing seed material.
- End-to-end qualification through the existing Puzzle Service and atomic assignment repository boundary.

Not delivered because it belongs to later phases:

- Solver search, deadlock analysis, or solution uniqueness proof.
- Replay generation, reconstruction, signing, or verification.
- Maze action validation, movement, scoring, timing, or completion.
- Realtime action dispatch, PvP synchronization, practice delivery, or reconnection.
- Client rendering, animation, audio, effects, or accessibility.

###### Files Changed

Generator implementation:

- `backend/internal/games/maze/generator/analysis.go`.
- `backend/internal/games/maze/generator/candidate.go`.
- `backend/internal/games/maze/generator/canonical.go`.
- `backend/internal/games/maze/generator/dependencies.go`.
- `backend/internal/games/maze/generator/geometry.go`.
- `backend/internal/games/maze/generator/patterns.go`.
- `backend/internal/games/maze/generator/processor.go`.
- `backend/internal/games/maze/generator/pipeline.go`.

Tests:

- `backend/internal/games/maze/generator/generator_test.go`.

Canonical documentation:

- `README.md`.

###### Public Contracts

Public REST APIs: **unchanged**.

Realtime WebSocket protocol: **unchanged**.

Player Platform and Admin CRM contracts: **unchanged**.

Frozen Sprint 1 through Sprint 5 business behavior: **unchanged**.

Additive internal contracts:

- `generator.Board`, `generator.Arrow`, `generator.Cell`, and `generator.Direction`.
- `generator.Pattern` and version-one pattern selection.
- `generator.DependencyGraph`.
- `generator.MeasuredDifficulty`.
- `generator.GenerationConfig`, `generator.GenerationReport`, and `generator.QualifiedCandidate`.
- `generator.IndependentVerifier` and `generator.Verification`.
- `generator.Observer` and `generator.CandidateObservation`.
- `generator.ProductionProcessor`.

The existing internal `generator.ProcessingInput` now receives the exact persisted `DifficultyProfile` loaded by the Puzzle Service. This is backward-compatible inside the unexported application boundary and introduces no HTTP, event, or client payload change.

###### Database

Database migrations: **none**.

Schema changes: **none**.

Phase 3 uses the normalized Puzzle Service tables and atomic uniqueness/finalization transaction delivered and validated in Phase 2. The generation pipeline executes outside that transaction; only accepted metadata is finalized through the repository.

###### Tests Added

Coverage includes:

- Valid and malformed integer geometry.
- Overlap, disconnected path, direction, and canonical identifier rejection.
- Physical escape-ray dependency derivation.
- Open-arrow behavior and cycle rejection.
- Deterministic approved pattern selection.
- Reproducible generation across one and four workers.
- Fixed canonical board fixture hash.
- Fixed candidate-batch observations in candidate-index order.
- Stable rejection codes for every rejected candidate.
- Missing-verifier rejection.
- Profile-hash drift rejection.
- Malformed verifier output rejection.
- Context cancellation.
- Seed-safe aggregate failure reports.
- Puzzle Service preparation, qualification, encrypted-seed retention, atomic assignment, and idempotent retry.
- A 32-seed structural corpus proving accepted graphs are re-derived from physical board geometry.

The independent verifier used by tests exists only in `_test.go`. It validates integration of the required contract without introducing a production solver before Phase 4.

###### Backend Verification

Environment:

```text
go version go1.26.5 windows/amd64
Windows AMD64
Intel Core i7-8665U at 1.90 GHz
```

Formatting, static analysis, and build:

```text
gofmt -w internal/games/maze/generator
go vet ./...
go build ./...
```

Result:

- All changed Go files are formatted.
- `go vet`: exit code `0`.
- `go build`: exit code `0`.

Full backend regression:

```text
go test ./... -count=1
```

Result:

- Exit code `0`.
- All backend packages compiled.
- `internal/db` passed in `140.525s`.
- `internal/realtime` passed in `15.365s`.
- `internal/server` passed in `14.269s`.
- `internal/payments` passed in `3.214s`.
- `internal/games/maze/generator` passed in `3.707s`.

Race verification:

```text
CGO_ENABLED=1 go test -race ./internal/games/maze/generator -count=1
```

Result: passed in `5.370s` with no race report.

Coverage:

```text
go test ./internal/games/maze/generator -coverprofile=phase3-cover.out -count=1
```

Result: `79.1%` statement coverage.

###### Frontend And Frozen-Sprint Regression Verification

Player Platform:

```text
npm run lint
npm run typecheck
npm test
npm run build
npm run test:e2e
npm audit --audit-level=high --omit=dev
```

Result:

- ESLint passed with zero warnings.
- TypeScript passed.
- Vitest: 5 files and 9 tests passed.
- Next.js `16.2.11` production build compiled 23 routes.
- Playwright: 21 tests passed across desktop, tablet, and mobile in `3.7m`.
- Production dependency audit found 0 vulnerabilities.

Admin CRM:

```text
npm run lint
npm run typecheck
npm test
npm run build
npm run test:e2e
npm audit --audit-level=high --omit=dev
```

Result:

- ESLint passed with zero warnings.
- TypeScript passed.
- Vitest: 2 files and 5 tests passed.
- Next.js `16.2.11` production build compiled 13 routes.
- Playwright: 3 tests passed across desktop, tablet, and mobile in `58.0s`.
- Production dependency audit found 0 vulnerabilities.

No Player Platform or Admin CRM source file changed in Phase 3. Playwright proof images regenerated during validation were restored to their committed versions after the tests.

###### Performance Evidence

Command:

```text
go test ./internal/games/maze/generator -run '^$' -bench BenchmarkProductionGenerator -benchmem -count=3
```

The benchmark executes a complete fixed batch of 16 candidates, structural validation, measured difficulty, independent test verification, canonical encoding, hashing, and deterministic selection.

Results:

```text
4743258 ns/op   3534694 B/op   53939 allocs/op
4759409 ns/op   3554015 B/op   54238 allocs/op
4833116 ns/op   3541177 B/op   54056 allocs/op
```

Observed average generation/qualification latency is approximately `4.78 ms`, comfortably below the approved standard-generation target of `100 ms` on this development machine. Solver-search cost is intentionally absent and becomes Phase 4 performance work.

###### Security Evidence

Verified:

- Generation uses only the sealed 256-bit effective seed prepared by the Puzzle Service.
- Random decisions use versioned, domain-separated cryptographic streams from Phase 2.
- Generation authority uses integer and fixed-point arithmetic only.
- Candidate generation is deterministic regardless of worker scheduling.
- Candidate counts and placement attempts are bounded.
- Context cancellation is checked before and during generation.
- Dependency authority is independently re-derived from board occupancy.
- Difficulty Profiles are hash-bound and exact; mutated profiles and silent fallback are rejected.
- The independent verifier is mandatory and fail-closed.
- Invalid solution hashes, final checksums, classifications, and minimum-action counts are rejected.
- Canonical encodings include geometry/rules versions and sorted identifiers.
- Failure reports expose stable aggregate codes and do not expose seed material.
- CPU generation occurs outside repository transactions and global store locks.
- Accepted puzzle uniqueness remains protected by the Phase 2 atomic PostgreSQL claim.
- No client-authoritative gameplay, network, replay, wallet, reward, or admin behavior was introduced.
- New production source contains no TODO, FIXME, mock, dummy, sample implementation, floating-point gameplay authority, `math/rand`, or hardcoded secret.

###### Architecture Protection Rule

Frozen Sprint 1 through Sprint 5 implementation files changed: **none**.

Boundary verification:

1. All implementation is owned by `backend/internal/games/maze/generator`.
2. Realtime Arena contains no Maze-specific action, state, networking, or matchmaking logic.
3. Arena Hub, Identity, Financial Platform, Admin CRM, and Realtime business contracts are unchanged.
4. No public API or event contract changed.
5. Complete frozen backend and browser regression suites pass.
6. No database migration or infrastructure configuration changed.

Rollback:

- Revert implementation commit `6182eaac3b4fd155b844e5d5f682b9e496ee5cfa`.
- No schema, data, route, event, or client rollback is required.

###### Known Limitations And Risks

- The production processor intentionally cannot accept a puzzle without an independent verifier. Phase 4 must provide the authoritative solver before the generator is composed into a live game flow.
- Structural acyclicity is not treated as proof of gameplay solvability or solution uniqueness. Those remain Phase 4 responsibilities.
- Pattern definitions currently influence deterministic placement, dependency targeting, and direction distribution. Their measured production quality must be calibrated with solver-qualified corpora in Phase 4 and production hardening in Phase 9.
- The benchmark excludes PostgreSQL/network latency and Phase 4 solver-search cost.
- The generator currently allocates approximately `3.54 MB` per 16-candidate batch. This is within current latency targets but should be profiled under Phase 9 concurrent generation load.
- No new PostgreSQL migration was required. Phase 2 persistence and atomic uniqueness behavior remain covered by the passing full `internal/db` regression suite.

No limitation above represents missing Implementation Phase 3 scope.

###### Remaining Work

Implementation Phase 4 remains **NOT STARTED** and owns:

- Authoritative solver search.
- Deadlock detection.
- Completion-path verification.
- Unique-versus-multiple solution classification.
- Minimum-action proof.
- Solver fixture and determinism hashes.
- Solver latency and failure-path validation.

Implementation Phases 5 through 9 remain **NOT STARTED**.

###### Phase Decision

**APPROVED**

Implementation Phase 4 must not begin until the product owner explicitly reviews and approves this validation report.

##### Public REST API Contracts

All routes are additive under `/api/v1`. They use the frozen browser session/cookie, CSRF, CORS, authorization, rate-limit, request-ID, and stable error-envelope controls. Maze does not introduce separate authentication.

Common error envelope:

```json
{
  "code": "GAME_SESSION_NOT_FOUND",
  "message": "The game session could not be found.",
  "requestId": "req_01..."
}
```

No active-session response exposes seed material, canonical solution, hidden dependencies, opponent private action history, solver output, or internal integrity evidence.

###### `GET /api/v1/games`

Purpose: return the approved player-visible game catalog from the Games Registry.

Authentication: optional. Eligibility fields require an authenticated player.

Request: no body.

Response:

```json
{
  "games": [
    {
      "id": "maze",
      "name": "Maze Arena",
      "description": "Clear the board by resolving every path.",
      "gameVersion": "1.0.0",
      "rendererVersion": 1,
      "capabilities": {
        "practice": true,
        "pvp": true,
        "ranked": true,
        "replay": true,
        "spectator": true
      },
      "playerRange": {"minimum": 1, "maximum": 2},
      "availability": "available"
    }
  ]
}
```

Errors:

- `GAME_CATALOG_UNAVAILABLE` (`503`).
- `RATE_LIMITED` (`429`).

Validation: only active player-visible versions are returned; internal artifact digests and retired versions are omitted.

###### `GET /api/v1/games/{gameId}`

Purpose: return one approved game descriptor and available modes.

Authentication: optional.

Response:

```json
{
  "game": {
    "id": "maze",
    "name": "Maze Arena",
    "gameVersion": "1.0.0",
    "rendererVersion": 1,
    "modes": ["tutorial", "practice", "pvp", "ranked", "house_challenge", "daily_challenge", "tournament"],
    "availability": "available"
  }
}
```

Errors:

- `GAME_NOT_FOUND` (`404`).
- `GAME_VERSION_UNAVAILABLE` (`409`).
- `RATE_LIMITED` (`429`).

Validation: `gameId` uses the registry ID format; only an approved player-visible version and modes are returned.

Maze enters gameplay through the frozen Sprint 5 Realtime API. Phase 4 adds game fields and opaque module payloads to those existing routes; it does not create a competing session, queue, snapshot, leave, reconnect, event, gateway, or replay API.

###### `POST /api/v1/realtime/queue`

Purpose: enter Practice, PvP, Ranked, or an approved platform-directed match flow through existing matchmaking.

Authentication: required; verified, active player session.

Headers:

- `Content-Type: application/json`.
- `X-CSRF-Token` for browser mutation.
- `Idempotency-Key` required.

Request:

```json
{
  "gameId": "maze",
  "mode": "practice",
  "modeReference": null,
  "clientCapabilities": {
    "rendererVersion": 1,
    "reducedMotion": false
  }
}
```

Response:

```json
{
  "queueId": "que_01...",
  "gameId": "maze",
  "mode": "practice",
  "status": "preparing",
  "matchId": "mat_01...",
  "gatewayPath": "/api/v1/realtime/gateway"
}
```

Errors:

- `AUTH_REQUIRED` (`401`).
- `EMAIL_VERIFICATION_REQUIRED` (`403`).
- `GAME_NOT_FOUND` (`404`).
- `GAME_MODE_UNSUPPORTED` (`422`).
- `GAME_MODE_INELIGIBLE` (`403`).
- `GAME_VERSION_UNAVAILABLE` (`409`).
- `PUZZLE_GENERATION_UNAVAILABLE` (`503`).
- `IDEMPOTENCY_KEY_REQUIRED` (`400`).
- `IDEMPOTENCY_CONFLICT` (`409`).
- `RATE_LIMITED` (`429`).

Validation:

- The server resolves all game, generator, rules, profile, and match versions.
- The server resolves Difficulty Profile from mode and player/platform policy.
- The client cannot submit seed, level internals, profile, board, opponent, priority, stake outcome, or authoritative state.
- Repeating an idempotency key with the same request returns the original queue/match result.
- Tournament, House, and Daily entry continues through their owning platform lifecycle before it delegates the approved match request to Realtime Arena.

###### `GET /api/v1/realtime/queue`

Purpose: recover the player's latest authoritative queue or preparation state.

Authentication: required.

Request: no body.

Response:

```json
{
  "queueId": "que_01...",
  "gameId": "maze",
  "mode": "practice",
  "status": "preparing",
  "matchId": "mat_01...",
  "gatewayPath": "/api/v1/realtime/gateway"
}
```

Errors:

- `QUEUE_NOT_FOUND` (`404`).
- `GAME_VERSION_UNAVAILABLE` (`409`).
- `RATE_LIMITED` (`429`).

Validation: the queue is resolved from the authenticated player; no player ID query parameter is accepted.

###### `DELETE /api/v1/realtime/queue`

Purpose: cancel an active queue entry before authoritative match start.

Authentication: required.

Headers: `X-CSRF-Token` and `Idempotency-Key`.

Request: no body.

Response:

```json
{
  "queueId": "que_01...",
  "status": "cancelled"
}
```

Errors:

- `QUEUE_NOT_FOUND` (`404`).
- `MATCH_ALREADY_STARTED` (`409`).
- `IDEMPOTENCY_KEY_REQUIRED` (`400`).
- `RATE_LIMITED` (`429`).

Validation: only the authenticated player's active queue entry may be cancelled; duplicate cancellation returns the original terminal result.

###### `GET /api/v1/realtime/matches/{matchId}`

Purpose: recover owned match status and a full viewer-safe game snapshot.

Authentication: required; owning participant or an explicitly authorized viewer through the applicable spectator/reviewer policy.

Response:

```json
{
  "matchId": "mat_01...",
  "gameId": "maze",
  "mode": "practice",
  "status": "ready",
  "gameVersion": "1.0.0",
  "rendererVersion": 1,
  "stateVersion": 12,
  "serverSequence": 104,
  "snapshot": {
    "schemaVersion": 1,
    "board": {},
    "progress": {}
  },
  "checksum": "sha256:...",
  "gatewayPath": "/api/v1/realtime/gateway"
}
```

Errors:

- `MATCH_FORBIDDEN` (`403`).
- `MATCH_NOT_FOUND` (`404`).
- `MATCH_EXPIRED` (`410`).
- `SNAPSHOT_NOT_READY` (`409`).
- `STATE_INTEGRITY_FAILURE` (`503`).

Validation: ownership and viewer role are resolved server-side; `snapshot` remains opaque to the generic handler and is validated against the pinned renderer schema.

###### `POST /api/v1/realtime/matches/{matchId}/ready`

Purpose: declare renderer readiness after the authoritative puzzle is assigned and the initial snapshot is accepted.

Authentication: required; owning participant.

Request:

```json
{
  "rendererVersion": 1,
  "snapshotChecksum": "sha256:..."
}
```

Response:

```json
{
  "matchId": "mat_01...",
  "participantStatus": "ready",
  "matchStatus": "ready",
  "startsAt": null
}
```

Errors:

- `MATCH_FORBIDDEN` (`403`).
- `MATCH_NOT_READY` (`409`).
- `RENDERER_VERSION_UNSUPPORTED` (`409`).
- `SNAPSHOT_CHECKSUM_MISMATCH` (`409`).

Validation: renderer version must be compatible with the pinned match version and the checksum must match the delivered viewer snapshot. The client cannot start the match clock; Realtime Arena starts according to authoritative readiness and lifecycle policy.

###### `POST /api/v1/realtime/matches/{matchId}/leave`

Purpose: request leave or forfeit through the existing authoritative match lifecycle.

Authentication: required; owning participant.

Headers: `X-CSRF-Token` and `Idempotency-Key`.

Response:

```json
{
  "matchId": "mat_01...",
  "status": "forfeited"
}
```

For Practice, status may be `abandoned`. Existing platform rules decide competitive forfeit and settlement consequences.

Errors:

- `MATCH_FORBIDDEN` (`403`).
- `MATCH_NOT_FOUND` (`404`).
- `MATCH_ALREADY_COMPLETE` (`409`).
- `IDEMPOTENCY_KEY_REQUIRED` (`400`).

Validation: only the authenticated participant may leave; repeated requests return the persisted terminal result. Competitive status and consequences are server-derived.

###### `POST /api/v1/realtime/matches/{matchId}/reconnect`

Purpose: recover from the last acknowledged server sequence and state version.

Authentication: required; owning participant.

Request:

```json
{
  "lastServerSequence": 100,
  "lastStateVersion": 11,
  "lastSnapshotChecksum": "sha256:..."
}
```

Response:

```json
{
  "matchId": "mat_01...",
  "status": "live",
  "stateVersion": 12,
  "serverSequence": 104,
  "snapshot": {},
  "events": [],
  "checksum": "sha256:..."
}
```

Errors:

- `MATCH_FORBIDDEN` (`403`).
- `MATCH_NOT_FOUND` (`404`).
- `RECONNECT_WINDOW_EXPIRED` (`410`).
- `STATE_INTEGRITY_FAILURE` (`503`).

Validation: sequence and version must be non-negative and cannot exceed authoritative values. A missing or mismatched checksum forces a full viewer-safe snapshot.

###### `POST /api/v1/realtime/matches/{matchId}/heartbeat`

Purpose: preserve existing connection presence and return authoritative server time.

Authentication: required; owning participant.

Request:

```json
{
  "lastServerSequence": 104
}
```

Response:

```json
{
  "matchId": "mat_01...",
  "presence": "online",
  "serverTime": "2026-07-28T12:00:02Z",
  "serverSequence": 104
}
```

Errors:

- `MATCH_FORBIDDEN` (`403`).
- `MATCH_NOT_FOUND` (`404`).
- `HEARTBEAT_RATE_LIMITED` (`429`).

Validation: sequence is non-negative and membership is server-resolved. Maze adds no heartbeat fields or timing authority.

###### `GET /api/v1/realtime/events/{matchId}?after={sequence}`

Purpose: return an authorized ordered event delta using existing Realtime persistence.

Authentication: required; owning participant or approved viewer.

Request: no body. `after` is a required non-negative sequence. The existing bounded server limit applies.

Response:

```json
{
  "matchId": "mat_01...",
  "events": [],
  "nextSequence": null,
  "eventRoot": "sha256:..."
}
```

Errors:

- `MATCH_FORBIDDEN` (`403`).
- `MATCH_NOT_FOUND` (`404`).
- `EVENT_SEQUENCE_INVALID` (`422`).
- `EVENT_INTEGRITY_FAILURE` (`503`).

Validation: event visibility is filtered by viewer role and pinned renderer schema. A sequence beyond the current committed sequence is rejected.

###### `GET /api/v1/realtime/replays/{matchId}`

Purpose: return signed replay metadata and authorized renderer-safe replay events for an owned terminal match through existing replay infrastructure.

Authentication: required unless future platform policy explicitly publishes the replay.

Response:

```json
{
  "replayId": "rpl_01...",
  "matchId": "mat_01...",
  "gameId": "maze",
  "gameVersion": "1.0.0",
  "replayVersion": 1,
  "rendererVersion": 1,
  "verificationStatus": "verified",
  "outcome": "completed",
  "eventCount": 84,
  "events": [],
  "eventRoot": "sha256:...",
  "startedAt": "2026-07-28T12:00:00Z",
  "completedAt": "2026-07-28T12:02:04Z"
}
```

Errors:

- `REPLAY_NOT_FOUND` (`404`).
- `REPLAY_FORBIDDEN` (`403`).
- `REPLAY_NOT_READY` (`409`).
- `REPLAY_UNDER_REVIEW` (`409`).
- `REPLAY_VERSION_UNAVAILABLE` (`503`).
- `REPLAY_INTEGRITY_FAILURE` (`503`).

Validation: `matchId` must identify a terminal match visible to the authenticated viewer. Seed references, canonical solution, signing internals, and opponent-private data are never returned.

###### `GET /api/v1/realtime/gateway`

Purpose: upgrade to the existing authenticated WebSocket protocol.

Authentication: required through the frozen cookie/session flow and approved origin.

Request: HTTP WebSocket upgrade with no JSON body.

Response: `101 Switching Protocols`, followed by the frozen connection acknowledgement and generic event protocol.

Errors:

- `AUTH_REQUIRED` (`401`).
- `SESSION_REVOKED` (`401`).
- `ORIGIN_NOT_ALLOWED` (`403`).
- `REALTIME_CONNECTION_EXISTS` (`409`).
- `REALTIME_CONNECTION_RATE_LIMITED` (`429`).
- `REALTIME_UNAVAILABLE` (`503`).

Validation: session, origin, one-active-connection policy, message-size limit, heartbeat deadlines, and connection throttles remain owned by Sprint 5. Maze adds only generic `game.action` and `game.sync.request` message kinds described below. It does not add a second gateway or Maze-specific socket.

##### Internal Service Contracts

Puzzle generation is not a public API. Realtime Arena and approved platform mode coordinators call a typed internal `PuzzleService`:

```text
Generate(context, GenerationRequest) -> ValidatedPuzzle
Claim(context, PuzzleReference, AssignmentScope) -> PuzzleAssignment
Load(context, PuzzleID) -> PuzzleMetadata
Regenerate(context, PuzzleMetadata) -> CanonicalPuzzle
Verify(context, PuzzleMetadata) -> VerificationReport
```

Rules:

- `Generate` performs CPU work outside a database transaction.
- `Claim` performs the short uniqueness and assignment transaction.
- A generated candidate is not deliverable until `Claim` commits.
- Internal requests carry authenticated service identity and correlation IDs.
- No Admin CRM endpoint can author or alter a puzzle, seed, solution, or validation result.

##### Realtime Event Contracts

The frozen gateway retains authentication, connection ownership, heartbeats, resume, sequence, and generic event persistence. Phase 4 specifies additive generic events. Every envelope contains:

```json
{
  "type": "game.event",
  "eventId": "evt_01...",
  "matchId": "mat_01...",
  "gameId": "maze",
  "serverSequence": 104,
  "stateVersion": 13,
  "occurredAt": "2026-07-28T12:00:01.250Z",
  "kind": "game.action.accepted",
  "payload": {}
}
```

Generic gateway fields do not contain Maze geometry. The selected module owns and schema-validates opaque `payload`.

###### Client-To-Server Messages

| Type | Purpose | Required controls |
|---|---|---|
| `game.action` | Submit one player intent | Auth, membership, action ID, client sequence, expected state version, size/schema limit, rate limit |
| `game.sync.request` | Request authoritative recovery snapshot | Auth, viewer permission, last server sequence, last state version |
| `match.leave` | Leave or forfeit | Auth, membership, idempotency, platform lifecycle rules |

Maze action payload:

```json
{
  "type": "game.action",
  "matchId": "mat_01...",
  "actionId": "act_01...",
  "clientSequence": 17,
  "expectedStateVersion": 12,
  "action": {
    "kind": "arrow.click",
    "payload": {"arrowId": "arrow_23"}
  }
}
```

The server ignores client time for ordering and never accepts client direction, collision, blocker, progress, score, completion, winner, board, seed, or resulting state.

###### Server-To-Client Events

| Event kind | Audience | Required payload |
|---|---|---|
| `match.started` | Participants and authorized spectators | Match/game/version identifiers, mode, authoritative start/deadline, participant-safe metadata |
| `game.puzzle.ready` | Participant/viewer-specific | Renderer version, immutable client-safe board, puzzle hash commitment, initial state version |
| `game.action.accepted` | Actor; sanitized progress may reach opponent/spectator | Action ID, resulting state version, authoritative presentation transition |
| `game.action.rejected` | Actor only unless integrity policy requires review | Action ID, stable code, unchanged state version, blocked presentation when applicable |
| `game.progress.updated` | Participant-safe audiences | Participant reference, completion percent, combo/progress fields allowed by mode policy |
| `game.snapshot` | Requesting viewer | Renderer version, state/server sequence, viewer-safe snapshot, checksum |
| `game.sync.required` | Affected connection | Stable reason, expected sequence/state, snapshot instruction |
| `match.completed` | Participants and authorized spectators | Server outcome, completion reason, final timing, replay pending status |
| `replay.ready` | Replay-authorized users | Replay ID, verification status, retrieval reference |
| `match.invalidated` | Authorized participants/reviewers | Stable reason category, review state; no sensitive anti-cheat evidence |

Ordering and delivery:

- Events use monotonic per-match server sequence.
- State-changing events commit atomically with state and action receipt.
- Delivery is at least once; clients deduplicate by event ID and sequence.
- Duplicate actions return the original persisted result.
- A sequence gap or checksum mismatch requires `game.sync.request`.
- No event emitted before transaction commit is authoritative.
- Opponent payloads are filtered by mode and viewer role.
- `replay.ready` is emitted only after replay persistence and initial integrity verification.

Stable action result codes:

- `ACTION_ACCEPTED`.
- `ACTION_BLOCKED`.
- `ACTION_INVALID`.
- `ACTION_DUPLICATE`.
- `ACTION_SEQUENCE_GAP`.
- `ACTION_STATE_CONFLICT`.
- `ACTION_RATE_LIMITED`.
- `MATCH_NOT_READY`.
- `MATCH_COMPLETE`.
- `MATCH_FORFEITED`.
- `GAME_VERSION_UNAVAILABLE`.
- `STATE_INTEGRITY_FAILURE`.

##### Database Migration Blueprint

Sprint 6 implementation uses additive, ordered PostgreSQL migrations. No existing frozen table or column is renamed, dropped, or reinterpreted.

Proposed migration sequence:

1. `game_module_versions`.
2. `game_generator_versions`.
3. `game_difficulty_profiles`.
4. `game_puzzles`.
5. `game_difficulty_analyses`.
6. `game_puzzle_uniqueness_claims`.
7. `game_puzzle_assignments`.
8. `game_participant_states`.
9. `game_action_receipts`.
10. `game_replay_metadata`.
11. Foreign keys from new tables to existing Realtime records.
12. Approved tutorial fixtures and version registration as controlled seed data.

Migrations are transactional where PostgreSQL permits, forward-only in production, reversible in an empty/non-production verification database, and validated from both a clean database and the current frozen schema.

###### `game_module_versions`

Primary key:

- Complete identity across `game_id`, `game_version`, `rules_version`, `protocol_version`, `replay_version`, and `renderer_version`.

Constraints:

- Non-empty normalized game ID.
- Semantic game version.
- Positive integer protocol/schema versions.
- Allowed status: `active`, `replay_only`, `retired`, `revoked`.
- `new_match_allowed` is false unless status is `active`.
- Manifest hash and artifact digest are fixed-length canonical digests.
- Referenced versions cannot be deleted.

Indexes:

- Active version lookup by game ID.
- Historical compatibility tuple lookup.
- Manifest hash uniqueness.

###### `game_generator_versions`

Primary key:

- Game ID plus generator, solver, validator, analyzer, Difficulty Profile schema, and canonical encoding versions.

Constraints:

- Every version component is positive and immutable after first puzzle use.
- Allowed status: `qualification`, `active`, `replay_only`, `retired`, `revoked`.
- Determinism fixture hash and artifact digest are required before activation.

Indexes:

- Active tuple by game ID and profile schema.
- Artifact digest.
- Status and release time.

###### `game_difficulty_profiles`

Primary key: immutable profile ID.

Foreign key: game/module version identity using restrictive deletion.

Constraints:

- Canonical profile data validates against its schema version.
- Profile hash is unique per game and schema version.
- Source is one of `practice`, `ranked`, `house`, `daily`, `tournament`, `tutorial`, or approved internal calibration.
- Created profiles are immutable.

Indexes:

- Game, schema version, source.
- Profile hash.

###### `game_puzzles`

Primary key: puzzle ID.

Foreign keys:

- Complete generator version tuple.
- Requested Difficulty Profile.
- Optional accepted difficulty analysis, added after analysis persistence.

Constraints:

- Seed ciphertext/reference is required; plaintext seed is prohibited.
- Fixed-length seed, generation, puzzle, validation, and solution hashes.
- Status transitions: `generating` -> `validated` -> `assigned` -> `consumed`; rejection and retirement are terminal for assignment.
- Validated status requires all immutable hashes, minimum actions, validation timestamp, and accepted analysis.
- Immutable generation inputs and hashes after validation.

Indexes:

- Status, game ID, mode, and created time for private pool claims.
- Puzzle hash.
- Seed hash.
- Generator tuple.
- Difficulty Profile.

###### `game_difficulty_analyses`

Primary key: analysis ID.

Foreign key: puzzle ID with restrictive deletion.

Constraints:

- One authoritative accepted analysis per puzzle/analyzer version.
- Canonical measured fields and analysis hash are immutable.
- Rejection reasons use bounded stable codes.

Indexes:

- Puzzle and analyzer version.
- Accepted/profile classification.
- Analysis hash.

###### `game_puzzle_uniqueness_claims`

Primary key: claim ID or puzzle ID according to repository implementation.

Foreign key: puzzle ID with restrictive deletion.

Constraints:

- Unique `seed_hash` for one-use policies.
- Unique `puzzle_hash` for one-use policies.
- Allowed reuse policy: `one_use`, `tutorial_fixture`, `daily_window`.
- One-use claims have exactly one first scope.
- Tutorial/daily reuse requires an explicit immutable scope policy.

Indexes:

- Unique partial indexes for one-use seed hash and puzzle hash.
- Scope type plus scope ID.
- Claim time for integrity review.

###### `game_puzzle_assignments`

Primary key: assignment ID.

Foreign keys:

- Puzzle ID.
- Existing match/session/challenge reference according to scope type.

Constraints:

- One assignment per PvP or tournament match.
- One fresh assignment per Practice or House attempt.
- Assignment reuse policy equals the puzzle claim policy.
- Assignment cannot precede puzzle validation.
- Consumed time cannot precede assignment time.

Indexes:

- Unique scope type plus scope ID where policy requires one assignment.
- Puzzle ID.
- Mode and assignment time.

###### `game_participant_states`

Primary key: match ID plus user ID.

Foreign keys:

- Existing Realtime match.
- Existing participant/user.
- Puzzle assignment.

Constraints:

- Opaque canonical state validates against pinned state schema version.
- State version and server sequence are non-negative and monotonic through compare-and-swap.
- State checksum is required.
- Allowed status: `ready`, `active`, `completed`, `forfeited`, `timed_out`, `invalid`, `under_review`.

Indexes:

- Match and status.
- User and updated time.
- Puzzle assignment.

###### `game_action_receipts`

Primary key: action ID.

Foreign keys: match and participant state.

Constraints:

- Unique match ID, user ID, and client sequence.
- State version after equals state version before for rejection.
- Accepted actions advance state according to transition contract.
- Result code, canonical payload hash, receipt hash, and processing times are required.
- Receipt is immutable after transaction commit.

Indexes:

- Match and server event sequence range.
- Participant and client sequence.
- Processed time for operations review.

###### `game_replay_metadata`

Primary/foreign key: replay ID referencing existing `realtime_replays`.

Foreign keys: puzzle, complete module version, and complete generator version.

Constraints:

- One row per Realtime replay.
- Final replay hash and signing key ID required for `verified`.
- Verification status: `pending`, `verified`, `failed`, `under_review`.
- Version, seed reference, hashes, outcome, and signing evidence are immutable after verification.

Indexes:

- Match/replay lookup through existing replay relationship.
- Puzzle ID.
- Verification status and time.
- Replay hash.

##### Transaction Boundaries

Generation transaction:

```text
Create generation job metadata
  -> Commit
  -> Generate/Solve/Validate/Analyze/Hash outside lock and transaction
  -> Begin short transaction
  -> Insert puzzle metadata
  -> Insert accepted analysis
  -> Claim seed and puzzle uniqueness
  -> Create assignment
  -> Commit
```

Action transaction:

```text
Resolve participant state
  -> Lock participant row or compare state version
  -> Detect duplicate action
  -> Validate and apply through pinned module
  -> Insert immutable action receipt
  -> Update participant state
  -> Append hash-chained Realtime events
  -> Commit
  -> Deliver committed result
```

Completion transaction:

```text
Persist final participant state
  -> Determine authoritative match outcome
  -> Append completion event
  -> Mark match complete
  -> Queue replay finalization
  -> Commit
```

Puzzle CPU work, replay reconstruction, rendering projection, and network delivery never occur while holding a global store lock.

##### Testing Strategy And Pass Criteria

###### Unit Tests

Required:

- Registry registration, duplicate rejection, compatibility, retirement, and historical resolution.
- Canonical encoding and every hash.
- Seed derivation and deterministic random vectors.
- Every arrow direction, boundary, collision ray, and nearest blocker.
- Immutable state transitions.
- Pattern selection and candidate ranking.
- Dependency derivation.
- Solver unique/multiple/deadlock classification.
- Validator gate ordering and failure codes.
- Difficulty features and tolerance boundaries.
- Replay codec and reconstruction.
- Renderer visibility filtering.
- API and event schema validation.

Pass criteria:

- Zero failures.
- Race-enabled Go tests pass for stateful packages.
- Critical authority packages (`engine`, `generator`, `solver`, `validator`, `replay`) achieve at least 95% statement coverage and 90% branch-equivalent decision coverage measured by the approved Go coverage tooling and review.
- Every stable error code has a test.

###### Property And Fuzz Tests

Required:

- Identical versioned input always produces identical bytes and hashes.
- Every accepted puzzle is solvable through live engine rules.
- State versions and event sequences never decrease.
- Removing an arrow never introduces a blocker.
- Canonical encode/decode round trips.
- Malformed geometry, actions, metadata, events, and replay payloads fail closed.

Pass criteria:

- Property suites run at least 10,000 generated cases per approved profile band in CI qualification jobs.
- Go fuzz seed corpus is checked in without secrets.
- Release qualification completes a minimum 30-minute fuzz run per parser/decoder target with no crash, hang, race, or accepted invalid state.

###### Integration Tests

Required:

- Registry -> module -> generator -> solver -> validator -> PostgreSQL claim -> assignment.
- Concurrent uniqueness claims.
- Participant action transaction with receipt and Realtime event.
- Duplicate action idempotency.
- Sequence-gap and stale-state recovery.
- Reconnect from snapshot plus later events.
- Shared PvP assignment with independent state.
- Practice and House uniqueness.
- Daily controlled reuse.
- Tournament per-match uniqueness.
- Replay persistence, object storage delivery, and verification.
- PostgreSQL rollback, Redis degradation, worker retry, and object-storage failure.

Pass criteria:

- Tests run against real PostgreSQL and Redis-compatible services in CI.
- S3-compatible integration uses an isolated real service such as MinIO, not a mocked SDK.
- No partial assignment, state, receipt, or replay remains after an injected transaction failure.
- All retries are idempotent.

###### End-To-End Tests

Required player journeys:

1. Authenticated player opens Game Hub, starts Practice, receives a generated puzzle, acts, reconnects, completes, and watches the verified replay.
2. Two authenticated players queue, receive one shared puzzle, act independently, disconnect/reconnect, complete, receive one authoritative outcome, and access replay.
3. Tutorial player completes fixtures 1 through 5 in order.
4. House participant receives a one-use puzzle through existing eligibility.
5. Daily participants share only the approved daily assignment.
6. Tournament match participants share one puzzle while a second match receives another.
7. Spectator receives only permitted data.

Pass criteria:

- Chromium desktop, tablet, and mobile pass.
- Critical renderer and interaction checks also pass in Firefox and WebKit.
- Screenshots/video evidence shows loading, ready, accepted, blocked, reconnecting, completed, replay, error, and reduced-motion states.
- No client request can forge score, completion, winner, state, seed, or difficulty.

###### Replay Determinism Tests

Required:

- Regenerate from exact metadata on every supported OS/architecture build target.
- Apply authoritative ordered actions.
- Compare puzzle, validation, state, event-root, outcome, and replay hashes.
- Verify signature and key ID.
- Corrupt each protected input independently.

Pass criteria:

- 100% equality for approved golden vectors.
- 100% of qualification corpus sample replays reconstruct.
- Every corruption case fails verification with no historical mutation.

###### Load And Soak Tests

Required:

- Concurrent puzzle generation by profile class.
- 100 simultaneous match preparations.
- 100 simultaneous live PvP matches with representative action rates.
- Practice bursts.
- Puzzle-pool claim contention.
- Replay reconstruction bursts.
- Reconnect storms.
- Worker restart and queue recovery.
- PostgreSQL uniqueness conflicts and Redis unavailability.
- Minimum two-hour steady-state soak and an extended pre-launch soak on target infrastructure.

Pass criteria:

- No duplicate assignment, lost accepted action, divergent participant state, unsigned replay, goroutine leak, unbounded queue growth, or integrity failure.
- Error rate below 0.1% excluding intentional validation rejections and injected failures.
- All accepted requests satisfy final service-level targets below.
- Recovery returns queues and active sessions to a consistent state.

###### Security Tests

Required:

- Unauthorized session, match, snapshot, spectator, and replay access.
- Cross-player action submission.
- Action replay, duplicate sequence, stale state, oversized payload, malformed schema, and rate-limit abuse.
- Seed/solution leakage review across responses, logs, traces, analytics, object metadata, and client bundles.
- Manifest/version tampering.
- Hash and signature corruption.
- Service identity and repository privilege enforcement.

Pass criteria:

- All unauthorized and malformed flows fail closed with stable non-sensitive errors.
- No secret seed, canonical solution, signing material, or opponent-private state is disclosed.
- Static analysis, dependency scanning, secret scanning, and race detection pass.

##### Measurable Performance Targets

Targets are measured on the approved Release 1.0 reference infrastructure with production-like PostgreSQL, Redis, S3-compatible storage, TLS, representative telemetry, and release builds. Measurements report P50, P95, P99, throughput, errors, CPU, memory, queue wait, and database time separately.

| Operation | Acceptance target |
|---|---|
| Tutorial fixture load and validation | P99 under 50 ms |
| Standard Practice/PvP generation pipeline | P50 under 100 ms, P95 under 500 ms, P99 under 1.5 s |
| Advanced Ranked/House generation pipeline | P50 under 300 ms, P95 under 1.5 s, P99 under 3 s |
| Elite Tournament generation pipeline | P50 under 750 ms, P95 under 3 s, P99 under 5 s |
| Standard solver only | P95 under 150 ms, P99 under 500 ms |
| Advanced solver only | P95 under 750 ms, P99 under 1.5 s |
| Replay reconstruction, standard profile | P95 under 500 ms, P99 under 1 s |
| Replay reconstruction, elite profile | P95 under 2 s, P99 under 5 s |
| Accepted action processing, excluding client network | P95 under 50 ms, P99 under 100 ms |
| Blocked action processing | P95 under 30 ms, P99 under 75 ms |
| Snapshot recovery from current durable state | P95 under 250 ms, P99 under 750 ms |
| Prepared-pool puzzle claim | P95 under 100 ms, P99 under 250 ms |
| Standard match preparation without pool | P95 under 2 s, P99 under 5 s |
| Match start after both players ready and puzzle prepared | P95 under 250 ms, P99 under 500 ms |
| Replay-ready after match completion | P95 under 2 s, P99 under 5 s |

Capacity acceptance:

- At least 100 concurrent standard puzzle-generation jobs without violating P99 or integrity targets on the declared reference deployment.
- At least 100 concurrent live PvP matches at representative action rate with no lost or duplicated authoritative action.
- Horizontal worker scaling produces documented throughput improvement without changing output selection.
- Queue depth remains bounded and oldest high-priority live-match job remains within its match-preparation target.

Hard deadlines remain:

- Standard generation: 5 seconds.
- Elite/Tournament generation: 10 seconds.
- Deadline failure never lowers difficulty, changes rules, or returns an unvalidated puzzle.

Final thresholds may be tightened after measurement. They may not be weakened merely to pass testing without documented product and architecture approval.

##### Sprint 6 Definition Of Done

Sprint 6 Maze Arena is complete only when all statements are true:

Architecture and modularity:

- Maze is registered through the generic Games Platform.
- Realtime Arena contains no Maze-specific import, payload, switch, storage, or rule.
- A test-only second module passes the same contract without Realtime changes.
- Complete immutable version tuples are pinned for matches and replays.

Generation and integrity:

- Tutorial 1 through 5 are the only handcrafted production fixtures.
- Every non-tutorial puzzle is server-generated.
- Generator output is deterministic for exact versioned input.
- Every puzzle passes solver, validator, measured difficulty, hashing, and uniqueness before delivery.
- Database constraints prevent prohibited seed or puzzle reuse.
- No global store lock surrounds generation, solving, validation, analysis, hashing, or replay reconstruction.

Mode behavior:

- Practice and House receive fresh one-use puzzles.
- PvP and Ranked participants receive one identical shared puzzle and independent state.
- Each Tournament match receives its own puzzle.
- Daily Challenge reuse is explicit and limited to its approved window.

Gameplay authority:

- The server alone validates actions, collision, progress, combo, completion, timing, and winner.
- The client sends only authorized intent.
- Blocked arrows remain on the board and authoritative presentation explains the collision.
- Only arrows that fully escape are removed.
- Disconnect, timeout, forfeit, and reconnect use frozen Realtime lifecycle rules.

Replay:

- Replays regenerate from exact historical metadata.
- Ordered authoritative actions reproduce state and outcome.
- Puzzle, validation, event-root, state, outcome, and replay hashes verify.
- Platform signature verifies.
- Corruption or unavailable history fails closed into review.

Product quality:

- Renderer is responsive and accessible.
- Accepted and blocked actions are visually understandable.
- Loading, generation, reconnect, stale state, completion, replay, reduced-motion, and failure states are complete.
- No placeholder UI, fake statistics, mock production service, baked post-tutorial catalogue, or client-generated puzzle exists.

Verification:

- Unit, property, fuzz, integration, E2E, replay, security, regression, load, and soak tests pass.
- The 100,000-candidate qualification corpus has zero accepted invalid puzzle, deadlock, structural violation, or duplicate one-use assignment.
- Performance targets are met on documented reference infrastructure.
- Builds, formatting, linting, static analysis, race detection, dependency scanning, and secret scanning pass.
- README contracts and implementation match.

##### Sprint 6 Freeze Criteria

Before a freeze recommendation, the final Sprint 6 report must provide:

1. Commit SHA and complete changed-file inventory.
2. Exact module manifest and compatibility tuple.
3. Database migration list, clean-install result, frozen-schema upgrade result, constraint tests, and rollback evidence.
4. API and Realtime contract tests with request, response, event, and stable error evidence.
5. Generator qualification report covering at least 100,000 candidates.
6. Solver correctness and malformed/deadlock rejection evidence.
7. Uniqueness concurrency evidence for Practice, House, PvP, Ranked, Daily, and Tournament policies.
8. Deterministic replay report across supported build targets.
9. PvP proof showing one puzzle hash and two independent state histories.
10. Desktop, tablet, mobile, Firefox, WebKit, accessibility, and reduced-motion evidence.
11. Performance benchmark, load, stress, reconnect-storm, and soak reports.
12. Security review covering zero-trust actions, seed secrecy, authorization, rate limits, hashes, signatures, dependencies, and secrets.
13. Failure-path evidence for PostgreSQL, Redis, object storage, worker, connection, and generation failure.
14. Observability evidence for generation, validation rejection, queue, match, action, replay, integrity, and version metrics.
15. Full Sprint 1 through Sprint 5 regression results.
16. Confirmation that no Admin CRM, Wallet, Treasury, matchmaking, auth, or Realtime business rule was duplicated inside Maze.
17. List of every unresolved defect categorized Critical, High, Medium, or Low.
18. Explicit freeze recommendation.

Freeze is prohibited when:

- Any Critical or High defect remains.
- Any authority exists on the client.
- Any generated production puzzle can bypass validation.
- Puzzle reuse violates the permanent policy.
- Replay reconstruction or signature verification is incomplete.
- PvP puzzle equality or participant-state independence is not proven.
- Performance or load evidence is missing.
- Frozen Sprint 1 through Sprint 5 regress.
- Documentation and implementation differ.

After approval and freeze:

- Create a dedicated Sprint 6 freeze commit and annotated tag only after all evidence passes.
- Maze changes become limited to bug fixes, security fixes, performance/scalability improvements, approved renderer polish, and backward-compatible integration support.
- Rules, generator, solver, validator, protocol, replay, renderer, and state changes require new immutable versions rather than mutation of historical behavior.

##### Phase 4 Definition Of Done

Documentation deliverables are complete:

- Final Games Platform and Maze folder structure.
- Ownership and dependency boundaries.
- Permanent Architecture Protection Rule for frozen Sprint 1 through Sprint 5 business logic.
- Module manifest blueprint.
- Nine implementation phases with mandatory build, test, validation, report, review, and approval gates.
- Mandatory per-phase validation report and evidence schema.
- Public REST API requests, responses, validation, and errors.
- Internal Puzzle Service contract.
- Generic Realtime client/server event contracts.
- Additive migration sequence, tables, indexes, constraints, and foreign keys.
- Transaction boundaries.
- Unit, property, fuzz, integration, E2E, replay, load, soak, and security strategy.
- Measurable latency and capacity acceptance targets.
- Sprint 6 Definition of Done.
- Complete freeze evidence and prohibition criteria.

No production gameplay code, migration, API, worker, React component, CSS, or frozen Sprint 1 through Sprint 5 implementation was changed during Phase 4.

Phase 4 is **APPROVED**.

Sprint 6 Implementation Phase 1 validation decision is **APPROVED**.

Sprint 6 Implementation Phase 2 validation decision is **APPROVED**.

Implementation Phases 3 through 9 remain **NOT STARTED** and require explicit product-owner authorization.

### Sprint 7: Tournaments, Leaderboards, Seasons, And Rewards

Visible outcome: players can qualify, compete, follow brackets and rankings, receive auditable rewards, and understand seasonal progress.

Required foundation work:

- Complete tournament and season lifecycle state machines.
- Use shared server-generated seeds per match and independent authoritative player state.
- Validate leaderboard calculations and reward eligibility through durable workers.
- Settle entries, prizes, refunds, and rewards through the transactional wallet and treasury flow.
- Add spectator, dispute, replay, recovery, and operational controls required by live competition.

### Sprint 8: Production Launch

Visible outcome: the approved Release 1.0 candidate operates from production infrastructure with certified external services, proven recovery, measured capacity, and no unresolved launch blockers.

Required launch work:

- Certify and activate the commercially selected payment provider adapters without changing the frozen Financial Platform contract.
- Complete jurisdiction, legal, KYC/AML, responsible-gaming, privacy, retention, terms, and fair-play approvals.
- Supply production secrets through the approved secret manager and verify that no credentials exist in source, images, logs, fixtures, or client bundles.
- Deploy independently scalable Player Platform, Admin CRM, API, workers, Realtime Arena, PostgreSQL, Redis, and S3-compatible storage units.
- Prove production email, payment callbacks, object storage, queues, workers, observability, alerting, backup, restore, and disaster recovery.
- Run representative and peak load tests for authentication, CRM operations, wallet settlement, live matches, replay delivery, matchmaking, leaderboards, and tournaments.
- Run dependency and network chaos tests for PostgreSQL, Redis, object storage, email, payment providers, workers, and realtime gateways.
- Complete penetration testing, dependency review, privilege review, financial reconciliation, replay-integrity review, accessibility review, and launch-candidate regression.
- Produce the Release 1.0 runbooks, ownership matrix, incident procedures, rollback plan, support escalation paths, and final launch recommendation.

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
| Deposit: requested -> pending provider -> pending verification -> completed |
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

Freeze decision: **SPRINT 2 APPROVED AND FROZEN.** Sprint 3 and Sprint 4 subsequently completed under their own independent freeze tags.

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
6. Backend derives one shared puzzle seed for the match and creates independent player board states.
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
- Verify callback signature
- Parse callback into provider-neutral events
- Query payment status
- Refund
- Create payout
- Query payout status
- Read provider balance
- Reconcile
- Health check
- Enforce idempotency
- Declare supported currencies and countries

Configured provider families:

- PayFast
- Ozow
- Card
- Bank EFT / bank transfer
- Peach Payments
- Flutterwave
- PayPal
- Xsolla
- Future crypto provider

`Payment Core` owns the provider registry and selects among active adapters by country, currency, method, availability, priority, cost, business preference, and preflight failover. Multiple adapters may be active simultaneously. The player API accepts `card`, `eft`, or `bank_transfer`; it never accepts or returns a provider name. Provider-specific logic, callback headers, objects, and terminology stay inside adapters.

Adapters are disabled until explicitly active and fully configured. Production startup requires at least one active adapter and an active default. No simulated provider is enabled in production; deterministic contract adapters exist only in tests.

### Deposit

Request requirements:

- Authenticated user
- Verified email
- Positive integer `amountMinor`
- `Idempotency-Key`
- Method and ISO-4217 currency
- Completed financial assessment
- Active responsible-gaming status
- Jurisdiction policy and limit eligibility

Lifecycle:

1. `requested`
2. `pending_provider`
3. `pending_verification`
4. `completed`, or `failed`, or `expired`

The signed, replay-protected provider callback validates provider, reference, amount, and currency before settlement. The same provider event is accepted once. Wallet available balance is credited in the same serializable transaction that releases pending funds, records the transition, and appends the hash-chained journal entry.

Provider callbacks persist every lifecycle transition. A successful signed callback moves the deposit to `pending_verification`; reserve validation and the atomic ledger settlement then move it to `completed`. Duplicate callbacks are ignored. A callback whose processing failed may be retried safely with the same provider event ID.

### Withdrawal

Request requirements:

- Authenticated user
- Verified email
- KYC when required
- Positive integer `amountMinor`
- Available live balance
- Trust-tier limit
- `Idempotency-Key`

Lifecycle:

Player-visible lifecycle:

1. `requested`
2. `pending_review`
3. `approved`
4. `processing`
5. `completed`, `rejected`, or `failed`

Initial policy is always manual review. A request atomically moves available funds into a pending withdrawal reserve. Rejection or provider failure returns that reserve exactly once. Completion releases the reserve and appends the immutable debit journal entry. Players never receive approval controls or internal risk details.

### Idempotency

All financial operations require `Idempotency-Key`.

Behavior:

- Same key and same request hash returns the existing operation.
- Same key and different request hash is rejected.
- Keys are stored against normalized deposit or withdrawal records with a request hash and a unique `(user_id, idempotency_key)` constraint.

### Treasury Reconciliation

Normalized Treasury reconciliation compares:

- Provider-reported minor-unit balance
- Hash-chained financial journal balance
- Exact minor-unit variance
- Immutable reconciliation hash

Reconciliation, assessment decisions, and withdrawal transitions are role-protected APIs intended for the separate Admin CRM. No CRM screen or approval action exists in the player application.

### Jurisdiction And Responsible Gaming

`SKILL_ARENA_FINANCIAL_POLICIES` may provide a JSON country-policy map. Each country defines:

- Currency and minimum age
- Enabled payment methods
- Whether source of funds is required
- Daily and monthly deposit limits in minor units
- Daily and monthly withdrawal limits in minor units

South Africa (`ZA`, `ZAR`) is the default policy. Player limit reductions, cooling-off, and self-exclusion take effect immediately. Limit increases require a future CRM compliance decision.

---

## API Reference

<!-- Archived source: docs/backup/individual-markdown-2026-07-15/API_REFERENCE.md -->

Base path: `/api/v1`

Browser authentication uses `HttpOnly`, `SameSite=Strict` access and refresh cookies. Service/native clients may send `Authorization: Bearer <token>`. Browser JavaScript must not persist either token.

Deposit and withdrawal POST requests require a unique 16-128 character `Idempotency-Key`.

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

### Financial Platform

| Method | Path | Purpose |
|---|---|---|
| GET | `/api/v1/financial/overview` | Wallet, eligibility, methods, limits, and active lifecycles |
| GET | `/api/v1/financial/transactions` | Immutable financial journal |
| POST | `/api/v1/financial/transactions/export` | Store a complete CSV journal export in object storage |
| GET | `/api/v1/financial/statements?from=&to=` | Generate a journal-backed statement |
| POST | `/api/v1/financial/statements/export` | Store the latest monthly CSV statement in object storage |
| GET | `/api/v1/financial/artifacts/{artifactId}` | Download an owned artifact with SHA-256 integrity header |
| GET, POST | `/api/v1/financial/evidence` | List or upload owned KYC/AML evidence |
| GET, PUT | `/api/v1/financial/assessment` | Read or submit player financial assessment |
| GET, PUT | `/api/v1/financial/limits` | Read or lower limits; set cooling-off/self-exclusion |
| GET, POST | `/api/v1/financial/deposits` | List deposits or create a provider session |
| GET, POST | `/api/v1/financial/withdrawals` | List or request withdrawals |

Deposit request:

```json
{
  "amountMinor": 10000,
  "currency": "ZAR",
  "method": "card"
}
```

Withdrawal uses the same body shape. The backend selects the provider. Success returns `202`; an identical idempotent replay returns `200` with `Idempotent-Replayed: true`; reuse with a different request returns `409`.

Assessment request:

```json
{
  "country": "ZA",
  "occupation": "employed",
  "sourceOfFunds": "salary"
}
```

Limit request values are integer minor units. Reductions are immediate; increases return `409 LIMIT_INCREASE_REQUIRES_REVIEW`.

Legacy `/api/v1/wallet` read endpoints remain temporarily available for frozen Hub compatibility. Legacy client-controlled deposit, withdrawal, lock, and unlock routes are no longer registered.

### Payment Callback

| Method | Path | Authentication | Purpose |
|---|---|---|---|
| POST | `/api/v1/payments/webhooks/{providerId}` | Adapter signature | Verify and process a provider-neutral payment event |

Each adapter owns its callback headers, signature scheme, envelope parsing, and status normalization. The Stripe reference adapter currently handles:

- `checkout.session.completed`
- `checkout.session.async_payment_succeeded`
- `checkout.session.async_payment_failed`
- `checkout.session.expired`
- `transfer.created`
- `transfer.reversed`
- `transfer.failed`
- `payout.created`
- `payout.paid`
- `payout.failed`
- `payout.canceled`

Payment Core passes the unmodified raw body and all headers to the selected adapter. Unknown providers return `404`, invalid or expired signatures return `401`, resource/amount/currency mismatches return `409`, and duplicate provider event IDs return `200 duplicate_ignored`.

### Treasury

| Method | Path | Role | Purpose |
|---|---|---|---|
| GET | `/api/v1/admin-crm/finance` | finance.read | Review deposits, withdrawals, provider health, reconciliations, and reserve checks |
| POST | `/api/v1/admin-crm/finance/withdrawals/{id}/decision` | withdrawals.review | Approve or reject a pending withdrawal through Payment Core |
| GET | `/api/v1/admin-crm/compliance/cases` | kyc.read | Review financial assessments, KYC/AML evidence, and provider responses |
| POST | `/api/v1/admin-crm/compliance/decisions` | kyc.decide | Record an assessment decision and notify the player |

Withdrawal decision:

```json
{
  "decision": "approve",
  "reason": "Manual review complete"
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

### Administrator API Isolation

The legacy player-audience `/api/v1/admin/*` routes are retired and return `404`. All reachable administrator operations are listed in the Sprint 4 API section and use the `/api/v1/admin-crm/*` namespace, CRM-only cookies and JWT audience, mandatory MFA, active privileged sessions, explicit permissions, and audit attribution.

---

## Database Schema

<!-- Archived source: docs/backup/individual-markdown-2026-07-15/DATABASE_SCHEMA.md -->

Production database: PostgreSQL.

Development fallback: JSON files under `backend/data/`, ignored from Git.

Migration sources: `backend/migrations/001_create_tables.sql`, `002_auth_identity.sql`, `003_arena_hub.sql`, and `004_financial_platform.sql`. Applied checksums are recorded in `schema_migrations`.

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
| `financial_wallets` | Integer minor-unit player balances and pending reserves |
| `financial_assessments` | Jurisdiction, source-of-funds, risk, and responsible-gaming status |
| `financial_limits` | Minor-unit limits, cooling-off, and self-exclusion |
| `financial_deposits` | Provider-neutral deposit state machine |
| `financial_withdrawals` | Policy-controlled withdrawal state machine |
| `financial_journal` | Append-only hash-chained settled money journal |
| `financial_transitions` | Auditable deposit and withdrawal state transitions |
| `payment_webhook_events` | Signed callback replay protection and outcome |
| `treasury_accounts` | Currency-specific treasury account balances |
| `treasury_reconciliations` | Immutable provider-to-journal reconciliation evidence |
| `financial_evidence` | KYC/AML evidence metadata and object-storage integrity |
| `financial_artifacts` | Statements, exports, and audit artifact metadata |
| `financial_payout_destinations` | Verified provider destinations used after manual approval |
| `treasury_reserve_checks` | Immutable provider balance, liability, and settlement decisions |

### Money Tables

`financial_wallets` is authoritative for the Financial Platform. Every amount is a signed 64-bit integer in ISO currency minor units. Constraints prevent negative available, pending, locked, and lifetime balances.

`financial_journal` records only settled balance-changing operations. Every entry includes sequence, previous hash, entry hash, reference, and post-entry balance. Deposit settlement and withdrawal completion append journal entries inside the same serializable transaction as the wallet update.

`wallets`, `ledger_entries`, and `financial_idempotency` are legacy compatibility tables for backend domains not yet migrated to the Sprint 3 Financial Platform. Their mutating player routes are not registered.

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
- `idx_financial_deposits_user_created`
- `idx_financial_withdrawals_review`
- `idx_financial_journal_user_sequence`
- `idx_financial_journal_reference`
- `idx_financial_transitions_resource`
- `idx_payment_webhooks_resource`
- `idx_financial_evidence_user_created`
- `idx_financial_artifacts_user_created`
- `idx_treasury_reserve_provider_created`

### Repository Note

PostgreSQL is authoritative in production. Authentication, Arena Hub, and Financial Platform domains use normalized repositories. Financial operations never use `store_snapshots`; the development runtime uses isolated in-memory financial repositories. Older game and competition domains remain transitional until their own production slices migrate.

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
- `SKILL_ARENA_PUZZLE_ENCRYPTION_KEY`
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
- Payment routing: `SKILL_ARENA_PAYMENT_ACTIVE_PROVIDERS`, `SKILL_ARENA_PAYMENT_DEFAULT_PROVIDER`, `SKILL_ARENA_PAYMENT_ROUTES`
- Stripe reference adapter: `SKILL_ARENA_STRIPE_SECRET_KEY`, `SKILL_ARENA_STRIPE_WEBHOOK_SECRET`, `SKILL_ARENA_STRIPE_MODE`, `SKILL_ARENA_STRIPE_API_BASE`
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
4. Verified provider callback moves the deposit to pending verification.
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
6. Backend derives one shared puzzle seed for the match and creates independent player board states.
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
