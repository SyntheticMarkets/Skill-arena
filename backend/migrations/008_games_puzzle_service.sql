CREATE TABLE IF NOT EXISTS game_generator_versions (
    version_id TEXT PRIMARY KEY,
    game_id TEXT NOT NULL CHECK (game_id = lower(game_id) AND length(game_id) BETWEEN 1 AND 80),
    generator_version INTEGER NOT NULL CHECK (generator_version > 0),
    seed_format_version INTEGER NOT NULL CHECK (seed_format_version > 0),
    random_stream_version INTEGER NOT NULL CHECK (random_stream_version > 0),
    pattern_catalogue_version INTEGER NOT NULL CHECK (pattern_catalogue_version > 0),
    pattern_selection_version INTEGER NOT NULL CHECK (pattern_selection_version > 0),
    geometry_schema_version INTEGER NOT NULL CHECK (geometry_schema_version > 0),
    candidate_scoring_version INTEGER NOT NULL CHECK (candidate_scoring_version > 0),
    constraint_policy_version INTEGER NOT NULL CHECK (constraint_policy_version > 0),
    solver_version INTEGER NOT NULL CHECK (solver_version > 0),
    validator_version INTEGER NOT NULL CHECK (validator_version > 0),
    analyzer_version INTEGER NOT NULL CHECK (analyzer_version > 0),
    difficulty_schema_version INTEGER NOT NULL CHECK (difficulty_schema_version > 0),
    canonical_encoding_version INTEGER NOT NULL CHECK (canonical_encoding_version > 0),
    status TEXT NOT NULL CHECK (status IN ('qualification','active','replay_only','retired','revoked')),
    new_match_allowed BOOLEAN NOT NULL DEFAULT FALSE,
    artifact_digest TEXT NOT NULL CHECK (artifact_digest ~ '^sha256:[0-9a-f]{64}$'),
    determinism_fixture_hash TEXT NOT NULL CHECK (determinism_fixture_hash ~ '^sha256:[0-9a-f]{64}$'),
    record JSONB NOT NULL,
    released_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL,
    CHECK (NOT new_match_allowed OR status = 'active'),
    UNIQUE (
        game_id, generator_version, seed_format_version, random_stream_version,
        pattern_catalogue_version, pattern_selection_version, geometry_schema_version,
        candidate_scoring_version, constraint_policy_version, solver_version,
        validator_version, analyzer_version, difficulty_schema_version,
        canonical_encoding_version
    )
);

CREATE INDEX IF NOT EXISTS idx_game_generator_versions_active
    ON game_generator_versions(game_id, difficulty_schema_version, released_at DESC)
    WHERE status = 'active' AND new_match_allowed;
CREATE INDEX IF NOT EXISTS idx_game_generator_versions_artifact
    ON game_generator_versions(artifact_digest);
CREATE INDEX IF NOT EXISTS idx_game_generator_versions_status_release
    ON game_generator_versions(status, released_at DESC);

CREATE TABLE IF NOT EXISTS game_difficulty_profiles (
    profile_id TEXT PRIMARY KEY,
    game_id TEXT NOT NULL CHECK (game_id = lower(game_id) AND length(game_id) BETWEEN 1 AND 80),
    schema_version INTEGER NOT NULL CHECK (schema_version > 0),
    source TEXT NOT NULL CHECK (source IN ('practice','ranked','house','daily','tournament','tutorial','calibration')),
    profile_hash TEXT NOT NULL CHECK (profile_hash ~ '^sha256:[0-9a-f]{64}$'),
    record JSONB NOT NULL,
    created_at TIMESTAMPTZ NOT NULL,
    UNIQUE (game_id, schema_version, profile_hash)
);

CREATE INDEX IF NOT EXISTS idx_game_difficulty_profiles_source
    ON game_difficulty_profiles(game_id, schema_version, source);

CREATE TABLE IF NOT EXISTS game_puzzles (
    puzzle_id TEXT PRIMARY KEY,
    game_id TEXT NOT NULL CHECK (game_id = lower(game_id) AND length(game_id) BETWEEN 1 AND 80),
    mode TEXT NOT NULL CHECK (length(mode) BETWEEN 1 AND 40),
    status TEXT NOT NULL CHECK (status IN ('preparing','validated','rejected','assigned','consumed','retired')),
    generator_version_id TEXT NOT NULL REFERENCES game_generator_versions(version_id) ON DELETE RESTRICT,
    difficulty_profile_id TEXT NOT NULL REFERENCES game_difficulty_profiles(profile_id) ON DELETE RESTRICT,
    difficulty_profile_hash TEXT NOT NULL CHECK (difficulty_profile_hash ~ '^sha256:[0-9a-f]{64}$'),
    request_hash TEXT NOT NULL UNIQUE CHECK (request_hash ~ '^sha256:[0-9a-f]{64}$'),
    seed_reference TEXT NOT NULL UNIQUE,
    seed_key_id TEXT NOT NULL,
    seed_hash TEXT NOT NULL CHECK (seed_hash ~ '^sha256:[0-9a-f]{64}$'),
    seed_ciphertext BYTEA NOT NULL CHECK (octet_length(seed_ciphertext) > 32),
    seed_nonce BYTEA NOT NULL CHECK (octet_length(seed_nonce) >= 12),
    generation_hash TEXT CHECK (generation_hash IS NULL OR generation_hash ~ '^sha256:[0-9a-f]{64}$'),
    puzzle_hash TEXT CHECK (puzzle_hash IS NULL OR puzzle_hash ~ '^sha256:[0-9a-f]{64}$'),
    validation_hash TEXT CHECK (validation_hash IS NULL OR validation_hash ~ '^sha256:[0-9a-f]{64}$'),
    solution_hash TEXT CHECK (solution_hash IS NULL OR solution_hash ~ '^sha256:[0-9a-f]{64}$'),
    accepted_analysis_id TEXT,
    minimum_actions INTEGER CHECK (minimum_actions IS NULL OR minimum_actions > 0),
    created_at TIMESTAMPTZ NOT NULL,
    validated_at TIMESTAMPTZ,
    assigned_at TIMESTAMPTZ,
    CHECK (
        status IN ('preparing','rejected','retired') OR
        (generation_hash IS NOT NULL AND puzzle_hash IS NOT NULL AND validation_hash IS NOT NULL
         AND solution_hash IS NOT NULL AND accepted_analysis_id IS NOT NULL
         AND minimum_actions IS NOT NULL AND validated_at IS NOT NULL)
    )
);

CREATE INDEX IF NOT EXISTS idx_game_puzzles_pool
    ON game_puzzles(status, game_id, mode, created_at);
CREATE INDEX IF NOT EXISTS idx_game_puzzles_puzzle_hash
    ON game_puzzles(puzzle_hash) WHERE puzzle_hash IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_game_puzzles_seed_hash
    ON game_puzzles(seed_hash);
CREATE INDEX IF NOT EXISTS idx_game_puzzles_generator
    ON game_puzzles(generator_version_id);
CREATE INDEX IF NOT EXISTS idx_game_puzzles_difficulty
    ON game_puzzles(difficulty_profile_id);

CREATE TABLE IF NOT EXISTS game_difficulty_analyses (
    analysis_id TEXT PRIMARY KEY,
    puzzle_id TEXT NOT NULL REFERENCES game_puzzles(puzzle_id) ON DELETE RESTRICT,
    analyzer_version INTEGER NOT NULL CHECK (analyzer_version > 0),
    accepted BOOLEAN NOT NULL,
    classification TEXT NOT NULL CHECK (length(classification) BETWEEN 1 AND 80),
    measured_fields JSONB NOT NULL,
    analysis_hash TEXT NOT NULL CHECK (analysis_hash ~ '^sha256:[0-9a-f]{64}$'),
    rejection_code TEXT CHECK (rejection_code IS NULL OR rejection_code ~ '^[a-z0-9_]{1,64}$'),
    created_at TIMESTAMPTZ NOT NULL,
    UNIQUE (puzzle_id, analyzer_version, analysis_hash)
);

CREATE UNIQUE INDEX IF NOT EXISTS uq_game_difficulty_analyses_accepted
    ON game_difficulty_analyses(puzzle_id, analyzer_version) WHERE accepted;
CREATE INDEX IF NOT EXISTS idx_game_difficulty_analyses_classification
    ON game_difficulty_analyses(accepted, classification);
CREATE INDEX IF NOT EXISTS idx_game_difficulty_analyses_hash
    ON game_difficulty_analyses(analysis_hash);

ALTER TABLE game_puzzles
    ADD CONSTRAINT fk_game_puzzles_accepted_analysis
    FOREIGN KEY (accepted_analysis_id) REFERENCES game_difficulty_analyses(analysis_id)
    DEFERRABLE INITIALLY DEFERRED;

CREATE TABLE IF NOT EXISTS game_puzzle_uniqueness_claims (
    claim_id TEXT PRIMARY KEY,
    puzzle_id TEXT NOT NULL UNIQUE REFERENCES game_puzzles(puzzle_id) ON DELETE RESTRICT,
    seed_hash TEXT NOT NULL CHECK (seed_hash ~ '^sha256:[0-9a-f]{64}$'),
    puzzle_hash TEXT NOT NULL CHECK (puzzle_hash ~ '^sha256:[0-9a-f]{64}$'),
    reuse_policy TEXT NOT NULL CHECK (reuse_policy IN ('one_use','tutorial_fixture','daily_window')),
    scope_type TEXT NOT NULL CHECK (length(scope_type) BETWEEN 1 AND 40),
    scope_id TEXT NOT NULL CHECK (length(scope_id) BETWEEN 1 AND 160),
    claimed_at TIMESTAMPTZ NOT NULL
);

CREATE UNIQUE INDEX IF NOT EXISTS uq_game_puzzle_claims_one_use_seed
    ON game_puzzle_uniqueness_claims(seed_hash) WHERE reuse_policy = 'one_use';
CREATE UNIQUE INDEX IF NOT EXISTS uq_game_puzzle_claims_one_use_hash
    ON game_puzzle_uniqueness_claims(puzzle_hash) WHERE reuse_policy = 'one_use';
CREATE INDEX IF NOT EXISTS idx_game_puzzle_claims_scope
    ON game_puzzle_uniqueness_claims(scope_type, scope_id);
CREATE INDEX IF NOT EXISTS idx_game_puzzle_claims_time
    ON game_puzzle_uniqueness_claims(claimed_at DESC);

CREATE TABLE IF NOT EXISTS game_puzzle_assignments (
    assignment_id TEXT PRIMARY KEY,
    puzzle_id TEXT NOT NULL REFERENCES game_puzzles(puzzle_id) ON DELETE RESTRICT,
    mode TEXT NOT NULL CHECK (length(mode) BETWEEN 1 AND 40),
    scope_type TEXT NOT NULL CHECK (length(scope_type) BETWEEN 1 AND 40),
    scope_id TEXT NOT NULL CHECK (length(scope_id) BETWEEN 1 AND 160),
    reuse_policy TEXT NOT NULL CHECK (reuse_policy IN ('one_use','tutorial_fixture','daily_window')),
    assigned_at TIMESTAMPTZ NOT NULL,
    consumed_at TIMESTAMPTZ,
    CHECK (consumed_at IS NULL OR consumed_at >= assigned_at),
    UNIQUE (scope_type, scope_id)
);

CREATE INDEX IF NOT EXISTS idx_game_puzzle_assignments_puzzle
    ON game_puzzle_assignments(puzzle_id);
CREATE INDEX IF NOT EXISTS idx_game_puzzle_assignments_mode_time
    ON game_puzzle_assignments(mode, assigned_at DESC);
