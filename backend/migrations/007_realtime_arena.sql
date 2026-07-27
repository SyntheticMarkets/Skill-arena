CREATE TABLE IF NOT EXISTS realtime_matches (
    id TEXT PRIMARY KEY,
    game_id TEXT NOT NULL,
    game_version TEXT NOT NULL,
    rules_version TEXT NOT NULL,
    protocol_version TEXT NOT NULL,
    replay_version TEXT NOT NULL,
    mode TEXT NOT NULL,
    status TEXT NOT NULL CHECK (status IN ('created','waiting_for_players','ready','starting','live','paused','reconnecting','completed','cancelled','abandoned')),
    region TEXT NOT NULL,
    wallet_category TEXT NOT NULL,
    seed_reference TEXT NOT NULL DEFAULT '',
    state_version BIGINT NOT NULL DEFAULT 1 CHECK (state_version > 0),
    sequence BIGINT NOT NULL DEFAULT 0 CHECK (sequence >= 0),
    created_at TIMESTAMPTZ NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL,
    started_at TIMESTAMPTZ,
    completed_at TIMESTAMPTZ
);

CREATE TABLE IF NOT EXISTS realtime_participants (
    match_id TEXT NOT NULL REFERENCES realtime_matches(id) ON DELETE CASCADE,
    user_id TEXT NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    status TEXT NOT NULL,
    ready BOOLEAN NOT NULL DEFAULT FALSE,
    rating INTEGER NOT NULL DEFAULT 0,
    region TEXT NOT NULL,
    latency_ms INTEGER NOT NULL DEFAULT 0 CHECK (latency_ms >= 0),
    last_sequence BIGINT NOT NULL DEFAULT 0 CHECK (last_sequence >= 0),
    joined_at TIMESTAMPTZ NOT NULL,
    last_seen_at TIMESTAMPTZ NOT NULL,
    left_at TIMESTAMPTZ,
    PRIMARY KEY(match_id,user_id)
);

CREATE TABLE IF NOT EXISTS realtime_queue (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    game_id TEXT NOT NULL,
    mode TEXT NOT NULL,
    wallet_category TEXT NOT NULL,
    region TEXT NOT NULL,
    jurisdiction TEXT NOT NULL,
    rating INTEGER NOT NULL,
    latency_ms INTEGER NOT NULL CHECK (latency_ms >= 0),
    priority INTEGER NOT NULL DEFAULT 0,
    status TEXT NOT NULL CHECK (status IN ('waiting','matched','cancelled','expired')),
    match_id TEXT REFERENCES realtime_matches(id),
    created_at TIMESTAMPTZ NOT NULL,
    expires_at TIMESTAMPTZ NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL
);

CREATE UNIQUE INDEX IF NOT EXISTS realtime_queue_one_waiting_per_user
ON realtime_queue(user_id) WHERE status='waiting';
CREATE INDEX IF NOT EXISTS realtime_queue_matchmaking_idx
ON realtime_queue(game_id,mode,wallet_category,region,status,priority DESC,created_at);
CREATE INDEX IF NOT EXISTS realtime_matches_status_idx ON realtime_matches(status,updated_at);
CREATE INDEX IF NOT EXISTS realtime_participants_user_idx ON realtime_participants(user_id,last_seen_at DESC);

CREATE TABLE IF NOT EXISTS realtime_presence (
    user_id TEXT PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    state TEXT NOT NULL CHECK (state IN ('online','offline','in_match','in_queue','idle','disconnected','reconnecting')),
    session_id TEXT NOT NULL DEFAULT '',
    connection_id TEXT NOT NULL DEFAULT '',
    match_id TEXT NOT NULL DEFAULT '',
    region TEXT NOT NULL DEFAULT '',
    last_heartbeat TIMESTAMPTZ NOT NULL,
    expires_at TIMESTAMPTZ NOT NULL
);
CREATE INDEX IF NOT EXISTS realtime_presence_state_idx ON realtime_presence(state,expires_at);

CREATE TABLE IF NOT EXISTS realtime_events (
    id TEXT PRIMARY KEY,
    match_id TEXT NOT NULL REFERENCES realtime_matches(id) ON DELETE CASCADE,
    user_id TEXT,
    type TEXT NOT NULL,
    sequence BIGINT NOT NULL,
    state_version BIGINT NOT NULL,
    server_time TIMESTAMPTZ NOT NULL,
    payload JSONB NOT NULL DEFAULT '{}'::jsonb,
    previous_hash TEXT NOT NULL DEFAULT '',
    integrity_hash TEXT NOT NULL,
    UNIQUE(match_id,sequence)
);
CREATE INDEX IF NOT EXISTS realtime_events_stream_idx ON realtime_events(match_id,sequence);
CREATE INDEX IF NOT EXISTS realtime_events_type_idx ON realtime_events(type);

CREATE TABLE IF NOT EXISTS realtime_snapshots (
    match_id TEXT NOT NULL REFERENCES realtime_matches(id) ON DELETE CASCADE,
    version BIGINT NOT NULL,
    sequence BIGINT NOT NULL,
    state JSONB NOT NULL,
    checksum TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL,
    PRIMARY KEY(match_id,version)
);

CREATE TABLE IF NOT EXISTS realtime_replays (
    id TEXT PRIMARY KEY,
    match_id TEXT NOT NULL UNIQUE REFERENCES realtime_matches(id) ON DELETE RESTRICT,
    game_id TEXT NOT NULL,
    game_version TEXT NOT NULL,
    rules_version TEXT NOT NULL,
    protocol_version TEXT NOT NULL,
    replay_version TEXT NOT NULL,
    first_sequence BIGINT NOT NULL,
    last_sequence BIGINT NOT NULL,
    event_count INTEGER NOT NULL,
    event_root_hash TEXT NOT NULL,
    signature TEXT NOT NULL,
    storage_key TEXT NOT NULL DEFAULT '',
    status TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL
);
CREATE INDEX IF NOT EXISTS realtime_replays_status_idx ON realtime_replays(status,created_at);
