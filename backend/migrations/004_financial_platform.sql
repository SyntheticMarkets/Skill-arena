CREATE TABLE IF NOT EXISTS financial_wallets (
    user_id TEXT PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    currency CHAR(3) NOT NULL,
    available_minor BIGINT NOT NULL DEFAULT 0 CHECK (available_minor >= 0),
    pending_deposit_minor BIGINT NOT NULL DEFAULT 0 CHECK (pending_deposit_minor >= 0),
    pending_withdrawal_minor BIGINT NOT NULL DEFAULT 0 CHECK (pending_withdrawal_minor >= 0),
    locked_minor BIGINT NOT NULL DEFAULT 0 CHECK (locked_minor >= 0),
    lifetime_deposit_minor BIGINT NOT NULL DEFAULT 0 CHECK (lifetime_deposit_minor >= 0),
    lifetime_withdrawal_minor BIGINT NOT NULL DEFAULT 0 CHECK (lifetime_withdrawal_minor >= 0),
    version BIGINT NOT NULL DEFAULT 1,
    updated_at TIMESTAMPTZ NOT NULL
);

CREATE TABLE IF NOT EXISTS financial_assessments (
    user_id TEXT PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    status TEXT NOT NULL CHECK (status IN ('not_started','submitted','in_review','complete','restricted')),
    country CHAR(2) NOT NULL DEFAULT '',
    occupation TEXT NOT NULL DEFAULT '',
    source_of_funds TEXT NOT NULL DEFAULT '',
    risk_classification TEXT NOT NULL DEFAULT 'unassessed',
    verification_status TEXT NOT NULL DEFAULT 'unverified',
    responsible_status TEXT NOT NULL DEFAULT 'active',
    submitted_at TIMESTAMPTZ,
    updated_at TIMESTAMPTZ NOT NULL
);

CREATE TABLE IF NOT EXISTS financial_limits (
    user_id TEXT PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    currency CHAR(3) NOT NULL,
    daily_deposit_minor BIGINT NOT NULL CHECK (daily_deposit_minor >= 0),
    monthly_deposit_minor BIGINT NOT NULL CHECK (monthly_deposit_minor >= 0),
    daily_withdrawal_minor BIGINT NOT NULL CHECK (daily_withdrawal_minor >= 0),
    monthly_withdrawal_minor BIGINT NOT NULL CHECK (monthly_withdrawal_minor >= 0),
    cooling_off_until TIMESTAMPTZ,
    self_excluded_until TIMESTAMPTZ,
    updated_at TIMESTAMPTZ NOT NULL
);

CREATE TABLE IF NOT EXISTS financial_deposits (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    amount_minor BIGINT NOT NULL CHECK (amount_minor > 0),
    currency CHAR(3) NOT NULL,
    method TEXT NOT NULL,
    provider TEXT NOT NULL,
    provider_reference TEXT,
    checkout_url TEXT,
    status TEXT NOT NULL CHECK (status IN ('requested','pending_provider','pending_verification','completed','failed','expired')),
    idempotency_key TEXT NOT NULL,
    request_hash TEXT NOT NULL,
    metadata JSONB NOT NULL DEFAULT '{}',
    requested_at TIMESTAMPTZ NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL,
    completed_at TIMESTAMPTZ,
    UNIQUE(user_id, idempotency_key)
);

CREATE TABLE IF NOT EXISTS financial_withdrawals (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    amount_minor BIGINT NOT NULL CHECK (amount_minor > 0),
    fee_minor BIGINT NOT NULL DEFAULT 0 CHECK (fee_minor >= 0),
    currency CHAR(3) NOT NULL,
    method TEXT NOT NULL,
    provider TEXT NOT NULL,
    provider_reference TEXT,
    status TEXT NOT NULL CHECK (status IN ('requested','pending_review','approved','processing','completed','rejected','failed')),
    policy_decision TEXT NOT NULL,
    policy_reasons JSONB NOT NULL DEFAULT '[]',
    idempotency_key TEXT NOT NULL,
    request_hash TEXT NOT NULL,
    metadata JSONB NOT NULL DEFAULT '{}',
    requested_at TIMESTAMPTZ NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL,
    completed_at TIMESTAMPTZ,
    UNIQUE(user_id, idempotency_key)
);

CREATE TABLE IF NOT EXISTS financial_journal (
    sequence BIGSERIAL PRIMARY KEY,
    id TEXT NOT NULL UNIQUE,
    user_id TEXT NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    account TEXT NOT NULL,
    direction TEXT NOT NULL CHECK (direction IN ('credit','debit')),
    amount_minor BIGINT NOT NULL CHECK (amount_minor > 0),
    currency CHAR(3) NOT NULL,
    balance_after_minor BIGINT NOT NULL CHECK (balance_after_minor >= 0),
    reference_type TEXT NOT NULL,
    reference_id TEXT NOT NULL,
    description TEXT NOT NULL,
    previous_hash TEXT NOT NULL,
    entry_hash TEXT NOT NULL UNIQUE,
    metadata JSONB NOT NULL DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL
);

CREATE TABLE IF NOT EXISTS financial_transitions (
    sequence BIGSERIAL PRIMARY KEY,
    resource_type TEXT NOT NULL CHECK (resource_type IN ('deposit','withdrawal')),
    resource_id TEXT NOT NULL,
    from_status TEXT NOT NULL,
    to_status TEXT NOT NULL,
    actor_type TEXT NOT NULL,
    actor_id TEXT,
    reason TEXT,
    provider_event_id TEXT,
    metadata JSONB NOT NULL DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL,
    UNIQUE(resource_type, resource_id, to_status, provider_event_id)
);

CREATE TABLE IF NOT EXISTS payment_webhook_events (
    id TEXT PRIMARY KEY,
    provider TEXT NOT NULL,
    provider_event_id TEXT NOT NULL,
    signature_hash TEXT NOT NULL,
    payload_hash TEXT NOT NULL,
    resource_type TEXT NOT NULL,
    resource_id TEXT NOT NULL,
    outcome TEXT NOT NULL,
    received_at TIMESTAMPTZ NOT NULL,
    processed_at TIMESTAMPTZ,
    UNIQUE(provider, provider_event_id)
);

CREATE TABLE IF NOT EXISTS treasury_accounts (
    account TEXT NOT NULL,
    currency CHAR(3) NOT NULL,
    balance_minor BIGINT NOT NULL DEFAULT 0,
    version BIGINT NOT NULL DEFAULT 1,
    updated_at TIMESTAMPTZ NOT NULL,
    PRIMARY KEY(account, currency)
);

CREATE TABLE IF NOT EXISTS treasury_reconciliations (
    id TEXT PRIMARY KEY,
    currency CHAR(3) NOT NULL,
    provider TEXT NOT NULL,
    provider_balance_minor BIGINT NOT NULL,
    journal_balance_minor BIGINT NOT NULL,
    difference_minor BIGINT NOT NULL,
    status TEXT NOT NULL CHECK (status IN ('balanced','variance','failed')),
    immutable_hash TEXT NOT NULL UNIQUE,
    created_at TIMESTAMPTZ NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_financial_deposits_user_created ON financial_deposits(user_id, requested_at DESC);
CREATE INDEX IF NOT EXISTS idx_financial_deposits_provider_ref ON financial_deposits(provider, provider_reference);
CREATE INDEX IF NOT EXISTS idx_financial_withdrawals_user_created ON financial_withdrawals(user_id, requested_at DESC);
CREATE INDEX IF NOT EXISTS idx_financial_withdrawals_review ON financial_withdrawals(status, requested_at);
CREATE INDEX IF NOT EXISTS idx_financial_journal_user_sequence ON financial_journal(user_id, sequence DESC);
CREATE INDEX IF NOT EXISTS idx_financial_journal_reference ON financial_journal(reference_type, reference_id);
CREATE INDEX IF NOT EXISTS idx_financial_transitions_resource ON financial_transitions(resource_type, resource_id, sequence);
CREATE INDEX IF NOT EXISTS idx_payment_webhooks_resource ON payment_webhook_events(resource_type, resource_id);
