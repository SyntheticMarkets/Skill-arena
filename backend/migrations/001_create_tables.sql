CREATE TABLE IF NOT EXISTS users (
    id TEXT PRIMARY KEY,
    email TEXT UNIQUE NOT NULL,
    country TEXT NOT NULL DEFAULT '',
    date_of_birth DATE,
    terms_accepted_at TIMESTAMP WITH TIME ZONE,
    username TEXT NOT NULL,
    display_name TEXT NOT NULL,
    hidden_from_public BOOLEAN NOT NULL DEFAULT FALSE,
    password_hash TEXT NOT NULL,
    role TEXT NOT NULL DEFAULT 'player',
    email_verified BOOLEAN NOT NULL DEFAULT FALSE,
    kyc_status TEXT NOT NULL DEFAULT 'unverified',
    created_at TIMESTAMP WITH TIME ZONE NOT NULL
);

CREATE TABLE IF NOT EXISTS auth_sessions (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL REFERENCES users(id),
    refresh_token_hash TEXT NOT NULL UNIQUE,
    user_agent TEXT,
    ip_address TEXT,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL,
    expires_at TIMESTAMP WITH TIME ZONE NOT NULL,
    revoked_at TIMESTAMP WITH TIME ZONE
);

CREATE TABLE IF NOT EXISTS wallets (
    user_id TEXT PRIMARY KEY REFERENCES users(id),
    live_balance NUMERIC(18,2) NOT NULL DEFAULT 0,
    live_locked_balance NUMERIC(18,2) NOT NULL DEFAULT 0,
    demo_balance NUMERIC(18,2) NOT NULL DEFAULT 1000,
    demo_locked_balance NUMERIC(18,2) NOT NULL DEFAULT 0,
    pending_withdrawals NUMERIC(18,2) NOT NULL DEFAULT 0,
    bonus_balance NUMERIC(18,2) NOT NULL DEFAULT 0
);

CREATE TABLE IF NOT EXISTS ledger_entries (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL REFERENCES users(id),
    transaction_type TEXT NOT NULL,
    amount NUMERIC(18,2) NOT NULL,
    balance_before NUMERIC(18,2) NOT NULL,
    balance_after NUMERIC(18,2) NOT NULL,
    currency TEXT NOT NULL,
    reference TEXT,
    metadata JSONB,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL
);

CREATE TABLE IF NOT EXISTS game_sessions (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL REFERENCES users(id),
    game_type TEXT NOT NULL,
    mode TEXT,
    house_tier TEXT,
    calibration BOOLEAN NOT NULL DEFAULT FALSE,
    stake NUMERIC(18,2) NOT NULL,
    reward_rate NUMERIC(8,4),
    difficulty INTEGER,
    difficulty_rating INTEGER,
    complexity_score INTEGER,
    expected_solve_percentiles JSONB,
    difficulty_profile JSONB,
    puzzle_seed TEXT,
    generation_nonce TEXT,
    generation_hash TEXT,
    puzzle_version JSONB,
    game_rules_version TEXT,
    state TEXT,
    outcome TEXT,
    reward NUMERIC(18,2),
    maze_cells JSONB,
    width INTEGER,
    height INTEGER,
    start_x INTEGER,
    start_y INTEGER,
    end_x INTEGER,
    end_y INTEGER,
    moves JSONB,
    lines JSONB,
    clicks JSONB,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL,
    completed_at TIMESTAMP WITH TIME ZONE,
    is_finished BOOLEAN NOT NULL DEFAULT FALSE
);

CREATE TABLE IF NOT EXISTS devices (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL REFERENCES users(id),
    fingerprint TEXT NOT NULL,
    device_name TEXT,
    os TEXT,
    browser TEXT,
    last_seen TIMESTAMP WITH TIME ZONE NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL,
    UNIQUE(user_id, fingerprint)
);

CREATE TABLE IF NOT EXISTS progression (
    user_id TEXT PRIMARY KEY REFERENCES users(id),
    xp INTEGER NOT NULL DEFAULT 0,
    level INTEGER NOT NULL DEFAULT 1,
    prestige INTEGER NOT NULL DEFAULT 0,
    elo_rating INTEGER NOT NULL DEFAULT 1200,
    league_tier TEXT NOT NULL DEFAULT 'Bronze',
    season_points INTEGER NOT NULL DEFAULT 0,
    legacy_points INTEGER NOT NULL DEFAULT 0,
    house_reputation INTEGER NOT NULL DEFAULT 0,
    matches_played INTEGER NOT NULL DEFAULT 0,
    wins INTEGER NOT NULL DEFAULT 0,
    losses INTEGER NOT NULL DEFAULT 0,
    current_streak INTEGER NOT NULL DEFAULT 0,
    best_moves INTEGER,
    trust_score NUMERIC(5,2) NOT NULL DEFAULT 100,
    trust_tier TEXT NOT NULL DEFAULT 'trusted',
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL
);

CREATE TABLE IF NOT EXISTS achievements (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL REFERENCES users(id),
    code TEXT NOT NULL,
    title TEXT NOT NULL,
    description TEXT NOT NULL,
    unlocked_at TIMESTAMP WITH TIME ZONE NOT NULL,
    UNIQUE(user_id, code)
);

CREATE TABLE IF NOT EXISTS seasons (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    theme TEXT NOT NULL,
    starts_at TIMESTAMP WITH TIME ZONE NOT NULL,
    ends_at TIMESTAMP WITH TIME ZONE NOT NULL,
    is_active BOOLEAN NOT NULL DEFAULT FALSE,
    reward_pool NUMERIC(18,2) NOT NULL DEFAULT 0,
    description TEXT
);

CREATE TABLE IF NOT EXISTS tournaments (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    type TEXT NOT NULL,
    status TEXT NOT NULL,
    entry_fee NUMERIC(18,2) NOT NULL,
    wallet_type TEXT NOT NULL,
    prize_pool NUMERIC(18,2) NOT NULL,
    minimum_level INTEGER NOT NULL,
    minimum_trust NUMERIC(5,2) NOT NULL,
    max_participants INTEGER NOT NULL,
    starts_at TIMESTAMP WITH TIME ZONE NOT NULL,
    ends_at TIMESTAMP WITH TIME ZONE NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL,
    description TEXT
);

CREATE TABLE IF NOT EXISTS tournament_participants (
    id TEXT PRIMARY KEY,
    tournament_id TEXT NOT NULL REFERENCES tournaments(id),
    user_id TEXT NOT NULL REFERENCES users(id),
    seed INTEGER NOT NULL,
    status TEXT NOT NULL,
    registered_at TIMESTAMP WITH TIME ZONE NOT NULL,
    UNIQUE(tournament_id, user_id)
);

CREATE TABLE IF NOT EXISTS tournament_matches (
    id TEXT PRIMARY KEY,
    tournament_id TEXT NOT NULL REFERENCES tournaments(id),
    round INTEGER NOT NULL,
    match_number INTEGER NOT NULL,
    player_a_id TEXT REFERENCES users(id),
    player_b_id TEXT REFERENCES users(id),
    winner_id TEXT REFERENCES users(id),
    status TEXT NOT NULL,
    prize_settled BOOLEAN NOT NULL DEFAULT FALSE,
    difficulty_rating INTEGER,
    complexity_score INTEGER,
    expected_solve_percentiles JSONB,
    difficulty_profile JSONB,
    puzzle_version JSONB,
    player_a_seed TEXT,
    player_b_seed TEXT,
    player_a_nonce TEXT,
    player_b_nonce TEXT,
    player_a_generation_hash TEXT,
    player_b_generation_hash TEXT,
    player_a_lines JSONB,
    player_b_lines JSONB,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL,
    completed_at TIMESTAMP WITH TIME ZONE
);

CREATE TABLE IF NOT EXISTS tournament_submissions (
    id TEXT PRIMARY KEY,
    tournament_id TEXT NOT NULL REFERENCES tournaments(id),
    match_id TEXT NOT NULL REFERENCES tournament_matches(id),
    user_id TEXT NOT NULL REFERENCES users(id),
    clicks JSONB,
    is_complete BOOLEAN NOT NULL,
    move_count INTEGER NOT NULL,
    duration_seconds NUMERIC(10,4) NOT NULL DEFAULT 0,
    submitted_at TIMESTAMP WITH TIME ZONE NOT NULL,
    UNIQUE(match_id, user_id)
);

CREATE TABLE IF NOT EXISTS pvp_matches (
    id TEXT PRIMARY KEY,
    player_a_id TEXT NOT NULL REFERENCES users(id),
    player_b_id TEXT REFERENCES users(id),
    queue_type TEXT NOT NULL,
    wallet_type TEXT NOT NULL,
    stake NUMERIC(18,2) NOT NULL,
    prize_pool NUMERIC(18,2) NOT NULL DEFAULT 0,
    platform_fee NUMERIC(18,2) NOT NULL DEFAULT 0,
    status TEXT NOT NULL,
    winner_id TEXT REFERENCES users(id),
    difficulty_rating INTEGER,
    complexity_score INTEGER,
    expected_solve_percentiles JSONB,
    difficulty_profile JSONB,
    puzzle_version JSONB,
    player_a_seed TEXT,
    player_b_seed TEXT,
    player_a_nonce TEXT,
    player_b_nonce TEXT,
    player_a_generation_hash TEXT,
    player_b_generation_hash TEXT,
    maze_cells JSONB,
    width INTEGER,
    height INTEGER,
    start_x INTEGER,
    start_y INTEGER,
    end_x INTEGER,
    end_y INTEGER,
    player_a_lines JSONB,
    player_b_lines JSONB,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL,
    started_at TIMESTAMP WITH TIME ZONE,
    completed_at TIMESTAMP WITH TIME ZONE
);

CREATE TABLE IF NOT EXISTS pvp_submissions (
    id TEXT PRIMARY KEY,
    match_id TEXT NOT NULL REFERENCES pvp_matches(id),
    user_id TEXT NOT NULL REFERENCES users(id),
    moves JSONB NOT NULL,
    clicks JSONB,
    is_valid_route BOOLEAN NOT NULL,
    move_count INTEGER NOT NULL,
    duration_seconds NUMERIC(10,4) NOT NULL DEFAULT 0,
    submitted_at TIMESTAMP WITH TIME ZONE NOT NULL,
    UNIQUE(match_id, user_id)
);

CREATE TABLE IF NOT EXISTS behavioral_baselines (
    user_id TEXT PRIMARY KEY REFERENCES users(id),
    calibration_runs INTEGER NOT NULL DEFAULT 0,
    average_efficiency NUMERIC(8,4) NOT NULL DEFAULT 0,
    average_move_seconds NUMERIC(10,4) NOT NULL DEFAULT 0,
    best_move_count INTEGER,
    last_session_id TEXT,
    last_run_at TIMESTAMP WITH TIME ZONE,
    risk_signal TEXT NOT NULL DEFAULT 'insufficient_data'
);

CREATE TABLE IF NOT EXISTS gameplay_telemetry (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL REFERENCES users(id),
    scope TEXT NOT NULL,
    scope_id TEXT NOT NULL,
    click_timestamps JSONB,
    click_intervals_ms JSONB,
    mouse_movement JSONB,
    touch_movement JSONB,
    device_fingerprint TEXT,
    user_agent TEXT,
    reaction_variance_ms NUMERIC(12,4),
    accuracy NUMERIC(8,4),
    failed_clicks INTEGER NOT NULL DEFAULT 0,
    successful_clicks INTEGER NOT NULL DEFAULT 0,
    privacy_classification TEXT NOT NULL,
    collected_at TIMESTAMP WITH TIME ZONE NOT NULL
);

CREATE TABLE IF NOT EXISTS review_cases (
    id TEXT PRIMARY KEY,
    scope TEXT NOT NULL,
    scope_id TEXT NOT NULL,
    user_id TEXT NOT NULL REFERENCES users(id),
    status TEXT NOT NULL,
    reason TEXT NOT NULL,
    flags JSONB,
    reviewer_id TEXT REFERENCES users(id),
    decision TEXT,
    metadata JSONB,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL,
    UNIQUE(scope, scope_id)
);

CREATE TABLE IF NOT EXISTS metrics_snapshots (
    id TEXT PRIMARY KEY,
    payload JSONB NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL
);

CREATE TABLE IF NOT EXISTS background_jobs (
    id TEXT PRIMARY KEY,
    type TEXT NOT NULL,
    status TEXT NOT NULL,
    payload JSONB,
    attempts INTEGER NOT NULL DEFAULT 0,
    max_attempts INTEGER NOT NULL DEFAULT 3,
    run_after TIMESTAMP WITH TIME ZONE NOT NULL,
    started_at TIMESTAMP WITH TIME ZONE,
    completed_at TIMESTAMP WITH TIME ZONE,
    last_error TEXT,
    worker TEXT,
    result_artifact TEXT,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL
);

CREATE TABLE IF NOT EXISTS backup_records (
    id TEXT PRIMARY KEY,
    type TEXT NOT NULL,
    status TEXT NOT NULL,
    path TEXT NOT NULL,
    verified BOOLEAN NOT NULL DEFAULT FALSE,
    size_bytes BIGINT NOT NULL DEFAULT 0,
    started_at TIMESTAMP WITH TIME ZONE NOT NULL,
    finished_at TIMESTAMP WITH TIME ZONE NOT NULL,
    error TEXT
);

CREATE TABLE IF NOT EXISTS audit_logs (
    id TEXT PRIMARY KEY,
    actor_id TEXT REFERENCES users(id),
    action TEXT NOT NULL,
    target_id TEXT,
    metadata JSONB,
    ip_address TEXT,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL
);

CREATE TABLE IF NOT EXISTS treasury_state (
    id TEXT PRIMARY KEY DEFAULT 'default',
    player_reserve NUMERIC(18,2) NOT NULL,
    revenue_reserve NUMERIC(18,2) NOT NULL,
    season_reserve NUMERIC(18,2) NOT NULL,
    championship_reserve NUMERIC(18,2) NOT NULL,
    jackpot_reserve NUMERIC(18,2) NOT NULL,
    emergency_reserve NUMERIC(18,2) NOT NULL,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL
);

CREATE TABLE IF NOT EXISTS store_snapshots (
    name TEXT PRIMARY KEY,
    payload JSONB NOT NULL,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL
);

CREATE TABLE IF NOT EXISTS financial_idempotency (
    idempotency_key TEXT PRIMARY KEY,
    user_id TEXT NOT NULL,
    operation TEXT NOT NULL,
    resource_type TEXT NOT NULL,
    resource_id TEXT NOT NULL,
    request_hash TEXT NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_ledger_entries_user_created ON ledger_entries(user_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_game_sessions_user_created ON game_sessions(user_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_auth_sessions_user ON auth_sessions(user_id);
CREATE INDEX IF NOT EXISTS idx_audit_logs_created ON audit_logs(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_tournament_participants_tournament ON tournament_participants(tournament_id, seed);
CREATE INDEX IF NOT EXISTS idx_tournament_matches_tournament ON tournament_matches(tournament_id, round, match_number);
CREATE INDEX IF NOT EXISTS idx_tournament_submissions_match ON tournament_submissions(match_id, submitted_at);
CREATE INDEX IF NOT EXISTS idx_pvp_matches_player_a ON pvp_matches(player_a_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_pvp_matches_player_b ON pvp_matches(player_b_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_pvp_matches_queue ON pvp_matches(status, queue_type, wallet_type, stake);
CREATE INDEX IF NOT EXISTS idx_gameplay_telemetry_scope ON gameplay_telemetry(scope, scope_id, collected_at);
CREATE INDEX IF NOT EXISTS idx_review_cases_status ON review_cases(status, updated_at);
CREATE INDEX IF NOT EXISTS idx_background_jobs_status ON background_jobs(status, run_after);
CREATE INDEX IF NOT EXISTS idx_backup_records_started ON backup_records(started_at DESC);
CREATE INDEX IF NOT EXISTS idx_pvp_submissions_match ON pvp_submissions(match_id, submitted_at);
CREATE INDEX IF NOT EXISTS idx_store_snapshots_updated ON store_snapshots(updated_at DESC);
CREATE INDEX IF NOT EXISTS idx_financial_idempotency_user_operation ON financial_idempotency(user_id, operation, created_at DESC);
