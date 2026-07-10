# Authentication Flow

## Registration

1. User posts email/password.
2. Backend creates user and default wallet/progression.
3. Backend creates one-time email verification token.
4. Email job is queued.
5. User remains unverified until token is consumed.

## Email Verification

1. User opens verification link.
2. Backend validates signed token.
3. Token must be unexpired and unused.
4. User is marked verified.
5. Audit log records action.

## Login

1. User posts credentials.
2. Backend checks lockout state.
3. Backend verifies password.
4. If MFA is enabled, TOTP or recovery code is required.
5. Privileged users without MFA receive enrollment-only token.
6. Backend issues JWT access token and refresh token.
7. Refresh session is stored and mirrored into Redis.

## Refresh

1. Client submits refresh token.
2. Backend validates session hash.
3. Old refresh token is revoked.
4. New refresh token session is created.
5. Redis session state is rotated.

## Logout

1. Client submits refresh token.
2. Backend revokes session.
3. Redis session state is deleted.
4. Audit log records action.

## Password Reset

1. User requests reset.
2. Backend creates expiring one-time reset token.
3. Email job is queued.
4. User submits token and new password confirmation.
5. Backend checks password history.
6. Password hash is updated.
7. Existing sessions are revoked.
8. Audit log records action.

## MFA

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

## JWT Claims

Access token includes:

- `sub`
- `role`
- MFA verification state
- Optional enrollment-only flag
- Expiry

## Rate Limiting

Rate limits protect:

- Login
- Registration
- Verification resend
- Password reset
- MFA confirm
- Match creation
- Replay retrieval
- Withdrawals

Production rate limiting uses Redis; local development falls back to memory.
