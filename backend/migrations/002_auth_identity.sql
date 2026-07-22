ALTER TABLE users ADD COLUMN IF NOT EXISTS status TEXT NOT NULL DEFAULT 'active';
ALTER TABLE users ADD COLUMN IF NOT EXISTS updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW();
ALTER TABLE users ADD COLUMN IF NOT EXISTS country TEXT NOT NULL DEFAULT '';
ALTER TABLE users ADD COLUMN IF NOT EXISTS date_of_birth DATE;
ALTER TABLE users ADD COLUMN IF NOT EXISTS terms_accepted_at TIMESTAMP WITH TIME ZONE;

ALTER TABLE auth_sessions ADD COLUMN IF NOT EXISTS rotated_at TIMESTAMP WITH TIME ZONE;
ALTER TABLE auth_sessions ADD COLUMN IF NOT EXISTS mfa_verified BOOLEAN NOT NULL DEFAULT FALSE;
ALTER TABLE auth_sessions ADD COLUMN IF NOT EXISTS enrollment_only BOOLEAN NOT NULL DEFAULT FALSE;
ALTER TABLE auth_sessions ADD COLUMN IF NOT EXISTS device_id TEXT;
ALTER TABLE auth_sessions ADD COLUMN IF NOT EXISTS family_id TEXT;

ALTER TABLE devices ADD COLUMN IF NOT EXISTS revoked_at TIMESTAMP WITH TIME ZONE;

ALTER TABLE audit_logs ADD COLUMN IF NOT EXISTS previous_hash TEXT;
ALTER TABLE audit_logs ADD COLUMN IF NOT EXISTS entry_hash TEXT;

CREATE TABLE IF NOT EXISTS auth_tokens (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    purpose TEXT NOT NULL,
    token_hash TEXT NOT NULL UNIQUE,
    expires_at TIMESTAMP WITH TIME ZONE NOT NULL,
    used_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL,
    created_ip TEXT,
    used_ip TEXT
);

CREATE TABLE IF NOT EXISTS mfa_settings (
    user_id TEXT PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    enabled BOOLEAN NOT NULL DEFAULT FALSE,
    totp_secret_ciphertext TEXT,
    recovery_code_hashes TEXT[] NOT NULL DEFAULT '{}',
    confirmed_at TIMESTAMP WITH TIME ZONE,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL
);

CREATE TABLE IF NOT EXISTS password_history (
    id BIGSERIAL PRIMARY KEY,
    user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    password_hash TEXT NOT NULL,
    password_stamp TEXT,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL
);

CREATE TABLE IF NOT EXISTS login_security (
    user_id TEXT PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    failed_count INTEGER NOT NULL DEFAULT 0,
    locked_until TIMESTAMP WITH TIME ZONE,
    last_failed_at TIMESTAMP WITH TIME ZONE,
    last_success_at TIMESTAMP WITH TIME ZONE,
    last_ip_address TEXT,
    last_user_agent TEXT,
    suspicious_flag TEXT,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_users_email_lower ON users (LOWER(email));
CREATE INDEX IF NOT EXISTS idx_auth_tokens_lookup ON auth_tokens (token_hash, purpose);
CREATE INDEX IF NOT EXISTS idx_auth_tokens_user_active ON auth_tokens (user_id, purpose, expires_at) WHERE used_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_auth_sessions_active ON auth_sessions (user_id, expires_at) WHERE revoked_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_auth_sessions_family ON auth_sessions (family_id) WHERE family_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_password_history_user_created ON password_history (user_id, created_at DESC);
CREATE UNIQUE INDEX IF NOT EXISTS idx_audit_logs_entry_hash ON audit_logs (entry_hash) WHERE entry_hash IS NOT NULL;

DO $$ BEGIN
    ALTER TABLE users ADD CONSTRAINT users_status_check CHECK (status IN ('active', 'suspended', 'disabled'));
EXCEPTION WHEN duplicate_object THEN NULL; END $$;
DO $$ BEGIN
    ALTER TABLE users ADD CONSTRAINT users_role_check CHECK (role IN ('player', 'admin', 'super_admin', 'treasury_manager', 'fraud_analyst', 'support', 'moderator'));
EXCEPTION WHEN duplicate_object THEN NULL; END $$;
DO $$ BEGIN
    ALTER TABLE users ADD CONSTRAINT users_country_check CHECK (country = '' OR country ~ '^[A-Z]{2}$');
EXCEPTION WHEN duplicate_object THEN NULL; END $$;
