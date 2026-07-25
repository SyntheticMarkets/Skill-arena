ALTER TABLE users DROP CONSTRAINT IF EXISTS users_role_check;
ALTER TABLE users ADD CONSTRAINT users_role_check CHECK (
    role IN (
        'player', 'admin', 'super_admin', 'treasury_manager', 'fraud_analyst',
        'support', 'moderator', 'compliance', 'finance', 'operations', 'read_only'
    )
);

CREATE TABLE IF NOT EXISTS crm_internal_notes (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    author_id TEXT NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    body TEXT NOT NULL CHECK (char_length(body) BETWEEN 1 AND 4000),
    created_at TIMESTAMPTZ NOT NULL
);

ALTER TABLE support_tickets ADD COLUMN IF NOT EXISTS priority TEXT NOT NULL DEFAULT 'normal';
ALTER TABLE support_tickets ADD COLUMN IF NOT EXISTS assigned_to TEXT REFERENCES users(id) ON DELETE SET NULL;
ALTER TABLE support_tickets ADD COLUMN IF NOT EXISTS escalated BOOLEAN NOT NULL DEFAULT FALSE;
ALTER TABLE support_tickets ADD COLUMN IF NOT EXISTS first_response_at TIMESTAMPTZ;
ALTER TABLE support_tickets DROP CONSTRAINT IF EXISTS support_tickets_status_check;
ALTER TABLE support_tickets ADD CONSTRAINT support_tickets_status_check
    CHECK (status IN ('open', 'received', 'in_progress', 'waiting_player', 'escalated', 'closed'));
ALTER TABLE support_tickets ADD CONSTRAINT support_tickets_priority_check
    CHECK (priority IN ('low', 'normal', 'high', 'urgent'));

CREATE TABLE IF NOT EXISTS crm_support_messages (
    id TEXT PRIMARY KEY,
    ticket_id TEXT NOT NULL REFERENCES support_tickets(id) ON DELETE CASCADE,
    author_id TEXT NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    body TEXT NOT NULL CHECK (char_length(body) BETWEEN 1 AND 8000),
    internal BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL
);

CREATE TABLE IF NOT EXISTS crm_support_attachments (
    id TEXT PRIMARY KEY,
    ticket_id TEXT NOT NULL REFERENCES support_tickets(id) ON DELETE CASCADE,
    user_id TEXT NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    object_key TEXT NOT NULL UNIQUE,
    file_name TEXT NOT NULL CHECK (char_length(file_name) BETWEEN 1 AND 255),
    content_type TEXT NOT NULL,
    size_bytes BIGINT NOT NULL CHECK (size_bytes BETWEEN 1 AND 10485760),
    sha256 CHAR(64) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL
);

CREATE TABLE IF NOT EXISTS crm_restrictions (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    restriction_type TEXT NOT NULL CHECK (
        restriction_type IN (
            'account', 'deposit', 'withdrawal', 'competition', 'communication',
            'cooling_off', 'self_exclusion'
        )
    ),
    reason TEXT NOT NULL CHECK (char_length(reason) BETWEEN 4 AND 1000),
    status TEXT NOT NULL CHECK (status IN ('active', 'lifted', 'expired')),
    expires_at TIMESTAMPTZ,
    created_by TEXT NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    created_at TIMESTAMPTZ NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL
);

CREATE TABLE IF NOT EXISTS crm_jurisdiction_policies (
    country CHAR(2) PRIMARY KEY,
    currency CHAR(3) NOT NULL,
    minimum_age INTEGER NOT NULL CHECK (minimum_age BETWEEN 18 AND 99),
    deposit_enabled BOOLEAN NOT NULL,
    withdrawal_enabled BOOLEAN NOT NULL,
    source_of_funds_required BOOLEAN NOT NULL,
    daily_deposit_minor BIGINT NOT NULL CHECK (daily_deposit_minor >= 0),
    monthly_deposit_minor BIGINT NOT NULL CHECK (monthly_deposit_minor >= 0),
    daily_withdrawal_minor BIGINT NOT NULL CHECK (daily_withdrawal_minor >= 0),
    monthly_withdrawal_minor BIGINT NOT NULL CHECK (monthly_withdrawal_minor >= 0),
    updated_by TEXT NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    updated_at TIMESTAMPTZ NOT NULL
);

CREATE TABLE IF NOT EXISTS crm_announcements (
    id TEXT PRIMARY KEY,
    category TEXT NOT NULL CHECK (category IN ('announcement', 'maintenance', 'security', 'compliance')),
    title TEXT NOT NULL CHECK (char_length(title) BETWEEN 4 AND 120),
    message TEXT NOT NULL CHECK (char_length(message) BETWEEN 10 AND 4000),
    audience TEXT NOT NULL CHECK (audience IN ('all', 'verified', 'restricted', 'country')),
    status TEXT NOT NULL CHECK (status IN ('draft', 'sent', 'cancelled')),
    created_by TEXT NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    created_at TIMESTAMPTZ NOT NULL,
    sent_at TIMESTAMPTZ
);

CREATE TABLE IF NOT EXISTS crm_compliance_provider_responses (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    provider TEXT NOT NULL CHECK (char_length(provider) BETWEEN 2 AND 80),
    provider_reference TEXT NOT NULL CHECK (char_length(provider_reference) BETWEEN 2 AND 255),
    check_type TEXT NOT NULL CHECK (check_type IN ('identity', 'address', 'age', 'sanctions', 'pep', 'aml', 'source_of_funds')),
    status TEXT NOT NULL CHECK (status IN ('pending', 'clear', 'review', 'rejected', 'error')),
    risk_signals TEXT[] NOT NULL DEFAULT '{}',
    metadata JSONB NOT NULL DEFAULT '{}',
    received_at TIMESTAMPTZ NOT NULL,
    UNIQUE(provider, provider_reference, check_type)
);

CREATE INDEX IF NOT EXISTS idx_crm_notes_user_created ON crm_internal_notes(user_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_crm_support_status_priority ON support_tickets(status, priority, updated_at DESC);
CREATE INDEX IF NOT EXISTS idx_crm_support_messages_ticket ON crm_support_messages(ticket_id, created_at);
CREATE INDEX IF NOT EXISTS idx_crm_support_attachments_ticket ON crm_support_attachments(ticket_id, created_at);
CREATE INDEX IF NOT EXISTS idx_crm_restrictions_user_status ON crm_restrictions(user_id, status, updated_at DESC);
CREATE INDEX IF NOT EXISTS idx_crm_announcements_status_created ON crm_announcements(status, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_crm_provider_responses_user_received ON crm_compliance_provider_responses(user_id, received_at DESC);

ALTER TABLE audit_logs ADD COLUMN IF NOT EXISTS device TEXT;
ALTER TABLE audit_logs ADD COLUMN IF NOT EXISTS reason TEXT;
ALTER TABLE audit_logs ADD COLUMN IF NOT EXISTS resource_type TEXT;
ALTER TABLE audit_logs ADD COLUMN IF NOT EXISTS previous_value JSONB;
ALTER TABLE audit_logs ADD COLUMN IF NOT EXISTS new_value JSONB;
