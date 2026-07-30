DROP INDEX IF EXISTS realtime_events_stream_idx;
DROP INDEX IF EXISTS idx_game_action_receipts_participant_sequence;

DROP INDEX IF EXISTS idx_game_participant_states_user_updated;
CREATE INDEX IF NOT EXISTS idx_game_participant_states_user
    ON game_participant_states(user_id);
