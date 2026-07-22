# Payment Flow

## Provider Abstraction

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

## Deposit

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

## Withdrawal

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

## Idempotency

All financial operations require `Idempotency-Key`.

Behavior:

- Same key and same request hash returns the existing operation.
- Same key and different request hash is rejected.
- Keys are recorded in the operation metadata and in production idempotency storage.

## Treasury Reconciliation

Treasury health compares:

- Player reserve
- Player liabilities
- House exposure
- Reserve totals

Financial flow tests verify that wallet balances reconcile with balance-changing ledger entries and that treasury remains solvent.

## AML

Risk inputs:

- Large withdrawal
- Velocity
- Country rules
- Trust tier

High-risk cases create AML review records and may escalate to fraud analyst workflow.
