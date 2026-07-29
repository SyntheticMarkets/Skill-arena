CREATE TABLE IF NOT EXISTS game_participant_states (
    match_id TEXT NOT NULL REFERENCES realtime_matches(id) ON DELETE RESTRICT,
    user_id TEXT NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    game_id TEXT NOT NULL,
    puzzle_id TEXT NOT NULL,
    state_schema_version TEXT NOT NULL,
    state_version BIGINT NOT NULL CHECK (state_version >= 0),
    state JSONB NOT NULL,
    state_checksum TEXT NOT NULL,
    last_client_sequence BIGINT NOT NULL DEFAULT 0 CHECK (last_client_sequence >= 0),
    last_server_sequence BIGINT NOT NULL DEFAULT 0 CHECK (last_server_sequence >= 0),
    status TEXT NOT NULL CHECK (status IN (
        'ready','active','completed','forfeited','timed_out','invalid','under_review'
    )),
    updated_at TIMESTAMPTZ NOT NULL,
    PRIMARY KEY (match_id, user_id)
);

CREATE INDEX IF NOT EXISTS idx_game_participant_states_match_status
    ON game_participant_states(match_id, status);
CREATE INDEX IF NOT EXISTS idx_game_participant_states_user_updated
    ON game_participant_states(user_id, updated_at DESC);
CREATE INDEX IF NOT EXISTS idx_game_participant_states_puzzle
    ON game_participant_states(puzzle_id);

CREATE TABLE IF NOT EXISTS game_action_receipts (
    action_id TEXT PRIMARY KEY,
    match_id TEXT NOT NULL,
    user_id TEXT NOT NULL,
    client_sequence BIGINT NOT NULL CHECK (client_sequence > 0),
    expected_state_version BIGINT NOT NULL CHECK (expected_state_version >= 0),
    action_kind TEXT NOT NULL,
    action_payload_hash TEXT NOT NULL,
    accepted BOOLEAN NOT NULL,
    result_code TEXT NOT NULL,
    state_version_before BIGINT NOT NULL CHECK (state_version_before >= 0),
    state_version_after BIGINT NOT NULL CHECK (state_version_after >= 0),
    CHECK (
        (accepted AND state_version_after = state_version_before + 1)
        OR (NOT accepted AND state_version_after = state_version_before)
    ),
    first_event_sequence BIGINT NOT NULL CHECK (first_event_sequence > 0),
    last_event_sequence BIGINT NOT NULL CHECK (last_event_sequence >= first_event_sequence),
    transition JSONB NOT NULL,
    receipt_hash TEXT NOT NULL,
    server_received_at TIMESTAMPTZ NOT NULL,
    processed_at TIMESTAMPTZ NOT NULL,
    FOREIGN KEY (match_id, user_id)
        REFERENCES game_participant_states(match_id, user_id) ON DELETE RESTRICT,
    UNIQUE (match_id, user_id, client_sequence)
);

CREATE INDEX IF NOT EXISTS idx_game_action_receipts_match_events
    ON game_action_receipts(match_id, first_event_sequence, last_event_sequence);
CREATE INDEX IF NOT EXISTS idx_game_action_receipts_participant_sequence
    ON game_action_receipts(match_id, user_id, client_sequence);
CREATE INDEX IF NOT EXISTS idx_game_action_receipts_processed
    ON game_action_receipts(processed_at DESC);
