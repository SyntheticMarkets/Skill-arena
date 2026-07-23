CREATE TABLE IF NOT EXISTS player_profiles (
    user_id TEXT PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    username TEXT NOT NULL,
    display_name TEXT NOT NULL,
    avatar_url TEXT NOT NULL DEFAULT '',
    country TEXT NOT NULL DEFAULT '',
    language TEXT NOT NULL DEFAULT 'en',
    created_at TIMESTAMP WITH TIME ZONE NOT NULL,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL
);

CREATE TABLE IF NOT EXISTS game_modules (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    description TEXT NOT NULL,
    category TEXT NOT NULL,
    version TEXT NOT NULL,
    renderer_key TEXT NOT NULL,
    modes JSONB NOT NULL,
    average_time_seconds INTEGER NOT NULL,
    capabilities JSONB NOT NULL,
    rules_summary JSONB NOT NULL,
    availability TEXT NOT NULL,
    availability_reason TEXT NOT NULL DEFAULT '',
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL
);

CREATE TABLE IF NOT EXISTS player_notifications (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    category TEXT NOT NULL,
    title TEXT NOT NULL,
    message TEXT NOT NULL,
    status TEXT NOT NULL CHECK (status IN ('unread', 'read', 'archived')),
    action_url TEXT NOT NULL DEFAULT '',
    metadata JSONB NOT NULL DEFAULT '{}',
    created_at TIMESTAMP WITH TIME ZONE NOT NULL,
    read_at TIMESTAMP WITH TIME ZONE,
    archived_at TIMESTAMP WITH TIME ZONE
);

CREATE TABLE IF NOT EXISTS notification_events (
    sequence BIGSERIAL PRIMARY KEY,
    notification_id TEXT NOT NULL REFERENCES player_notifications(id) ON DELETE CASCADE,
    user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    event_type TEXT NOT NULL,
    payload JSONB NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL
);

CREATE TABLE IF NOT EXISTS support_tickets (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    category TEXT NOT NULL,
    subject TEXT NOT NULL,
    message TEXT NOT NULL,
    status TEXT NOT NULL CHECK (status IN ('open', 'received', 'closed')),
    created_at TIMESTAMP WITH TIME ZONE NOT NULL,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_player_profiles_username_lower ON player_profiles (LOWER(username));
CREATE INDEX IF NOT EXISTS idx_progression_rank ON progression (elo_rating DESC, xp DESC);
CREATE INDEX IF NOT EXISTS idx_game_modules_availability ON game_modules (availability, name);
CREATE INDEX IF NOT EXISTS idx_notifications_user_status_created ON player_notifications (user_id, status, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_notification_events_user_sequence ON notification_events (user_id, sequence);
CREATE INDEX IF NOT EXISTS idx_support_tickets_user_updated ON support_tickets (user_id, updated_at DESC);
