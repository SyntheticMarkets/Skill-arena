CREATE TABLE IF NOT EXISTS financial_evidence (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    evidence_type TEXT NOT NULL,
    object_key TEXT NOT NULL UNIQUE,
    content_type TEXT NOT NULL,
    size_bytes BIGINT NOT NULL CHECK (size_bytes > 0),
    sha256 TEXT NOT NULL,
    status TEXT NOT NULL CHECK (status IN ('received','verified','rejected','expired')),
    created_at TIMESTAMPTZ NOT NULL
);

CREATE TABLE IF NOT EXISTS financial_artifacts (
    id TEXT PRIMARY KEY,
    user_id TEXT REFERENCES users(id) ON DELETE RESTRICT,
    artifact_type TEXT NOT NULL,
    object_key TEXT NOT NULL UNIQUE,
    content_type TEXT NOT NULL,
    size_bytes BIGINT NOT NULL CHECK (size_bytes > 0),
    sha256 TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL
);

CREATE TABLE IF NOT EXISTS financial_payout_destinations (
    user_id TEXT NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    provider TEXT NOT NULL,
    provider_account_id TEXT NOT NULL,
    status TEXT NOT NULL CHECK (status IN ('pending','verified','disabled')),
    evidence_id TEXT REFERENCES financial_evidence(id) ON DELETE RESTRICT,
    updated_at TIMESTAMPTZ NOT NULL,
    PRIMARY KEY(user_id, provider),
    UNIQUE(provider, provider_account_id)
);

CREATE TABLE IF NOT EXISTS treasury_reserve_checks (
    id TEXT PRIMARY KEY,
    provider TEXT NOT NULL,
    currency CHAR(3) NOT NULL,
    provider_available_minor BIGINT NOT NULL,
    provider_pending_minor BIGINT NOT NULL,
    liability_minor BIGINT NOT NULL CHECK (liability_minor >= 0),
    requested_minor BIGINT NOT NULL CHECK (requested_minor >= 0),
    purpose TEXT NOT NULL CHECK (purpose IN ('deposit_settlement','withdrawal_processing','reconciliation')),
    passed BOOLEAN NOT NULL,
    immutable_hash TEXT NOT NULL UNIQUE,
    artifact_key TEXT NOT NULL UNIQUE,
    created_at TIMESTAMPTZ NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_financial_evidence_user_created ON financial_evidence(user_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_financial_artifacts_user_created ON financial_artifacts(user_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_treasury_reserve_provider_created ON treasury_reserve_checks(provider, currency, created_at DESC);
