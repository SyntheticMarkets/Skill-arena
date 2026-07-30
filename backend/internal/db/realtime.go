package db

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"time"

	"skill-arena/internal/arena/registry"
	gamesregistry "skill-arena/internal/games/registry"
	"skill-arena/internal/id"
	"skill-arena/internal/models"
	"skill-arena/internal/observability"
	"skill-arena/migrations"
)

var ErrRealtimeConflict = errors.New("realtime state conflict")

func (s *Store) initPostgresRealtime(ctx context.Context) error {
	return s.applyFinancialMigration(ctx, "007_realtime_arena", migrations.RealtimeArena)
}

func (s *Store) ArenaRegistry() *registry.Registry {
	return s.arenaRegistry
}

func (s *Store) GamesRegistry() *gamesregistry.Registry {
	return s.gamesRegistry
}

func (s *Store) CreateRealtimeMatch(ctx context.Context, match models.RealtimeMatch, participant models.RealtimeParticipant) (*models.RealtimeMatch, error) {
	if match.ID == "" || participant.UserID == "" {
		return nil, errors.New("match and participant identifiers are required")
	}
	if s.pg != nil {
		tx, err := s.pg.BeginTx(ctx, nil)
		if err != nil {
			return nil, err
		}
		defer tx.Rollback()
		_, err = tx.ExecContext(ctx, `INSERT INTO realtime_matches
(id,game_id,game_version,rules_version,protocol_version,replay_version,mode,status,region,wallet_category,seed_reference,state_version,sequence,created_at,updated_at)
VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15)`,
			match.ID, match.GameID, match.GameVersion, match.RulesVersion, match.ProtocolVersion, match.ReplayVersion,
			match.Mode, match.Status, match.Region, match.WalletCategory, match.SeedReference, match.StateVersion,
			match.Sequence, match.CreatedAt, match.UpdatedAt)
		if err != nil {
			return nil, err
		}
		if err := insertRealtimeParticipant(ctx, tx, participant); err != nil {
			return nil, err
		}
		if err := tx.Commit(); err != nil {
			return nil, err
		}
		return s.GetRealtimeMatch(ctx, match.ID)
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	copyMatch := match
	copyParticipant := participant
	copyMatch.Participants = []models.RealtimeParticipant{copyParticipant}
	s.realtimeMatches[match.ID] = &copyMatch
	return cloneRealtimeMatch(&copyMatch), nil
}

func insertRealtimeParticipant(ctx context.Context, tx *sql.Tx, p models.RealtimeParticipant) error {
	_, err := tx.ExecContext(ctx, `INSERT INTO realtime_participants
(match_id,user_id,status,ready,rating,region,latency_ms,last_sequence,joined_at,last_seen_at,left_at)
VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)
ON CONFLICT(match_id,user_id) DO UPDATE SET status=EXCLUDED.status,ready=EXCLUDED.ready,latency_ms=EXCLUDED.latency_ms,last_seen_at=EXCLUDED.last_seen_at,left_at=EXCLUDED.left_at`,
		p.MatchID, p.UserID, p.Status, p.Ready, p.Rating, p.Region, p.LatencyMS, p.LastSequence, p.JoinedAt, p.LastSeenAt, p.LeftAt)
	return err
}

func (s *Store) GetRealtimeMatch(ctx context.Context, matchID string) (*models.RealtimeMatch, error) {
	if s.pg != nil {
		var m models.RealtimeMatch
		rows, err := s.pg.QueryContext(ctx, `SELECT
m.id,m.game_id,m.game_version,m.rules_version,m.protocol_version,m.replay_version,
m.mode,m.status,m.region,m.wallet_category,m.seed_reference,m.state_version,m.sequence,
m.created_at,m.updated_at,m.started_at,m.completed_at,
p.match_id,p.user_id,p.status,p.ready,p.rating,p.region,p.latency_ms,p.last_sequence,
p.joined_at,p.last_seen_at,p.left_at
FROM realtime_matches m
JOIN realtime_participants p ON p.match_id=m.id
WHERE m.id=$1
ORDER BY p.joined_at`, matchID)
		if err != nil {
			return nil, err
		}
		defer rows.Close()
		found := false
		for rows.Next() {
			var p models.RealtimeParticipant
			if err := rows.Scan(
				&m.ID, &m.GameID, &m.GameVersion, &m.RulesVersion, &m.ProtocolVersion,
				&m.ReplayVersion, &m.Mode, &m.Status, &m.Region, &m.WalletCategory,
				&m.SeedReference, &m.StateVersion, &m.Sequence, &m.CreatedAt, &m.UpdatedAt,
				&m.StartedAt, &m.CompletedAt, &p.MatchID, &p.UserID, &p.Status, &p.Ready,
				&p.Rating, &p.Region, &p.LatencyMS, &p.LastSequence, &p.JoinedAt,
				&p.LastSeenAt, &p.LeftAt,
			); err != nil {
				return nil, err
			}
			found = true
			m.Participants = append(m.Participants, p)
		}
		if err := rows.Err(); err != nil {
			return nil, err
		}
		if !found {
			return nil, sql.ErrNoRows
		}
		return &m, nil
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	match := s.realtimeMatches[matchID]
	if match == nil {
		return nil, sql.ErrNoRows
	}
	return cloneRealtimeMatch(match), nil
}

func (s *Store) RealtimeMatchCountForUser(ctx context.Context, userID string) (int, error) {
	if s.pg != nil {
		var count int
		err := s.pg.QueryRowContext(ctx, `SELECT count(*) FROM realtime_participants WHERE user_id=$1`, userID).Scan(&count)
		return count, err
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	count := 0
	for _, match := range s.realtimeMatches {
		for _, participant := range match.Participants {
			if participant.UserID == userID {
				count++
				break
			}
		}
	}
	return count, nil
}

func (s *Store) SaveRealtimeMatch(ctx context.Context, match models.RealtimeMatch, expectedVersion int64) (*models.RealtimeMatch, error) {
	match.UpdatedAt = time.Now().UTC()
	match.StateVersion = expectedVersion + 1
	if s.pg != nil {
		result, err := s.pg.ExecContext(ctx, `UPDATE realtime_matches SET status=$1,state_version=$2,updated_at=$3,started_at=$4,completed_at=$5 WHERE id=$6 AND state_version=$7`,
			match.Status, match.StateVersion, match.UpdatedAt, match.StartedAt, match.CompletedAt, match.ID, expectedVersion)
		if err != nil {
			return nil, err
		}
		count, _ := result.RowsAffected()
		if count != 1 {
			return nil, ErrRealtimeConflict
		}
		return cloneRealtimeMatch(&match), nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	current := s.realtimeMatches[match.ID]
	if current == nil {
		return nil, sql.ErrNoRows
	}
	if current.StateVersion != expectedVersion {
		return nil, ErrRealtimeConflict
	}
	match.Participants = append([]models.RealtimeParticipant(nil), current.Participants...)
	s.realtimeMatches[match.ID] = cloneRealtimeMatch(&match)
	return cloneRealtimeMatch(&match), nil
}

func (s *Store) TransitionRealtimeMatch(
	ctx context.Context,
	match models.RealtimeMatch,
	expectedVersion int64,
	userID, eventType string,
	payload json.RawMessage,
) (*models.RealtimeMatch, *models.RealtimeEvent, error) {
	if len(payload) == 0 {
		payload = json.RawMessage(`{}`)
	}
	if s.pg != nil {
		started := time.Now()
		conn, err := s.pg.Conn(ctx)
		observability.ObserveTiming(ctx, "db.realtime_transition.acquire", started)
		if err != nil {
			return nil, nil, err
		}
		defer conn.Close()
		var sequence int64
		var previous string
		started = time.Now()
		err = conn.QueryRowContext(ctx, `SELECT m.sequence,
COALESCE((SELECT integrity_hash FROM realtime_events
          WHERE match_id=m.id ORDER BY sequence DESC LIMIT 1),'')
FROM realtime_matches m WHERE m.id=$1 AND m.state_version=$2`,
			match.ID, expectedVersion).Scan(&sequence, &previous)
		observability.ObserveTiming(ctx, "db.realtime_transition.read", started)
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil, ErrRealtimeConflict
		}
		if err != nil {
			return nil, nil, err
		}
		match.StateVersion = expectedVersion + 1
		event := s.newRealtimeEvent(
			match.ID, userID, eventType, sequence+1, match.StateVersion, payload, previous,
		)
		match.Sequence = event.Sequence
		match.UpdatedAt = event.ServerTime
		state, err := json.Marshal(match)
		if err != nil {
			return nil, nil, err
		}
		checksum := sha256.Sum256(state)
		started = time.Now()
		result, err := conn.ExecContext(ctx, `WITH transition_guard AS MATERIALIZED (
    SELECT 1
    FROM realtime_matches AS guarded_match
    WHERE guarded_match.id=$17
      AND guarded_match.state_version=$18
      AND guarded_match.sequence=$25
      AND COALESCE((
          SELECT integrity_hash FROM realtime_events
          WHERE match_id=guarded_match.id
          ORDER BY sequence DESC LIMIT 1
      ),'')=$26
    FOR UPDATE OF guarded_match
), event_insert AS (
    INSERT INTO realtime_events(
        id,match_id,user_id,type,sequence,state_version,server_time,payload,
        previous_hash,integrity_hash
    )
    SELECT $1,$2,NULLIF($3,''),$4,$5,$6,$7,$8,$9,$10
    FROM transition_guard
    RETURNING 1
), match_update AS (
    UPDATE realtime_matches SET
        status=$11,state_version=$12,sequence=$13,updated_at=$14,
        started_at=$15,completed_at=$16
    WHERE id=$17 AND state_version=$18
      AND EXISTS (SELECT 1 FROM event_insert)
    RETURNING 1
)
INSERT INTO realtime_snapshots(match_id,version,sequence,state,checksum,created_at)
SELECT $19,$20,$21,$22,$23,$24
FROM match_update
ON CONFLICT(match_id,version) DO NOTHING`,
			event.ID, event.MatchID, event.UserID, event.Type, event.Sequence,
			event.StateVersion, event.ServerTime, event.Payload, event.PreviousHash,
			event.IntegrityHash, match.Status, match.StateVersion, match.Sequence,
			match.UpdatedAt, match.StartedAt, match.CompletedAt, match.ID,
			expectedVersion, match.ID, match.StateVersion, match.Sequence, state,
			hex.EncodeToString(checksum[:]), time.Now().UTC(), sequence, previous,
		)
		observability.ObserveTiming(ctx, "db.realtime_transition.persist", started)
		if err != nil {
			return nil, nil, err
		}
		affected, _ := result.RowsAffected()
		if affected != 1 {
			return nil, nil, ErrRealtimeConflict
		}
		return cloneRealtimeMatch(&match), &event, nil
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	current := s.realtimeMatches[match.ID]
	if current == nil {
		return nil, nil, sql.ErrNoRows
	}
	if current.StateVersion != expectedVersion {
		return nil, nil, ErrRealtimeConflict
	}
	match.StateVersion = expectedVersion + 1
	match.Participants = append([]models.RealtimeParticipant(nil), current.Participants...)
	events := s.realtimeEvents[match.ID]
	previous := ""
	if len(events) > 0 {
		previous = events[len(events)-1].IntegrityHash
	}
	event := s.newRealtimeEvent(
		match.ID, userID, eventType, current.Sequence+1, match.StateVersion,
		payload, previous,
	)
	match.Sequence = event.Sequence
	match.UpdatedAt = event.ServerTime
	s.realtimeMatches[match.ID] = cloneRealtimeMatch(&match)
	s.realtimeEvents[match.ID] = append(events, event)
	state, err := json.Marshal(match)
	if err != nil {
		return nil, nil, err
	}
	checksum := sha256.Sum256(state)
	s.realtimeSnapshots[match.ID] = append(
		s.realtimeSnapshots[match.ID],
		models.RealtimeSnapshot{
			MatchID: match.ID, Version: match.StateVersion, Sequence: match.Sequence,
			State: state, Checksum: hex.EncodeToString(checksum[:]),
			CreatedAt: time.Now().UTC(),
		},
	)
	return cloneRealtimeMatch(&match), &event, nil
}

func (s *Store) SaveRealtimeParticipant(ctx context.Context, p models.RealtimeParticipant) error {
	if s.pg != nil {
		_, err := s.pg.ExecContext(ctx, `INSERT INTO realtime_participants
(match_id,user_id,status,ready,rating,region,latency_ms,last_sequence,joined_at,last_seen_at,left_at)
VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)
ON CONFLICT(match_id,user_id) DO UPDATE SET
status=EXCLUDED.status,ready=EXCLUDED.ready,latency_ms=EXCLUDED.latency_ms,
last_sequence=EXCLUDED.last_sequence,last_seen_at=EXCLUDED.last_seen_at,
left_at=EXCLUDED.left_at`,
			p.MatchID, p.UserID, p.Status, p.Ready, p.Rating, p.Region, p.LatencyMS,
			p.LastSequence, p.JoinedAt, p.LastSeenAt, p.LeftAt,
		)
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	match := s.realtimeMatches[p.MatchID]
	if match == nil {
		return sql.ErrNoRows
	}
	for i := range match.Participants {
		if match.Participants[i].UserID == p.UserID {
			match.Participants[i] = p
			return nil
		}
	}
	match.Participants = append(match.Participants, p)
	return nil
}

func (s *Store) UpsertRealtimeQueue(ctx context.Context, entry models.RealtimeQueueEntry) error {
	if s.pg != nil {
		_, err := s.pg.ExecContext(ctx, `INSERT INTO realtime_queue
(id,user_id,game_id,mode,wallet_category,region,jurisdiction,rating,latency_ms,priority,status,match_id,created_at,expires_at,updated_at)
VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,NULLIF($12,''),$13,$14,$15)
ON CONFLICT(id) DO UPDATE SET status=EXCLUDED.status,match_id=EXCLUDED.match_id,updated_at=EXCLUDED.updated_at`,
			entry.ID, entry.UserID, entry.GameID, entry.Mode, entry.WalletCategory, entry.Region, entry.Jurisdiction,
			entry.Rating, entry.LatencyMS, entry.Priority, entry.Status, entry.MatchID, entry.CreatedAt, entry.ExpiresAt, entry.UpdatedAt)
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	copyEntry := entry
	s.realtimeQueue[entry.ID] = &copyEntry
	return nil
}

func (s *Store) WaitingRealtimeQueue(ctx context.Context, gameID, mode, wallet, region string, now time.Time) ([]models.RealtimeQueueEntry, error) {
	items := []models.RealtimeQueueEntry{}
	if s.pg != nil {
		rows, err := s.pg.QueryContext(ctx, `SELECT id,user_id,game_id,mode,wallet_category,region,jurisdiction,rating,latency_ms,priority,status,COALESCE(match_id,''),created_at,expires_at,updated_at FROM realtime_queue WHERE game_id=$1 AND mode=$2 AND wallet_category=$3 AND region=$4 AND status='waiting' AND expires_at>$5 ORDER BY priority DESC,created_at`, gameID, mode, wallet, region, now)
		if err != nil {
			return nil, err
		}
		defer rows.Close()
		for rows.Next() {
			var e models.RealtimeQueueEntry
			if err := rows.Scan(&e.ID, &e.UserID, &e.GameID, &e.Mode, &e.WalletCategory, &e.Region, &e.Jurisdiction, &e.Rating, &e.LatencyMS, &e.Priority, &e.Status, &e.MatchID, &e.CreatedAt, &e.ExpiresAt, &e.UpdatedAt); err != nil {
				return nil, err
			}
			items = append(items, e)
		}
		return items, rows.Err()
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, e := range s.realtimeQueue {
		if e.GameID == gameID && e.Mode == mode && e.WalletCategory == wallet && e.Region == region && e.Status == models.QueueWaiting && e.ExpiresAt.After(now) {
			items = append(items, *e)
		}
	}
	sort.Slice(items, func(i, j int) bool {
		if items[i].Priority == items[j].Priority {
			return items[i].CreatedAt.Before(items[j].CreatedAt)
		}
		return items[i].Priority > items[j].Priority
	})
	return items, nil
}

func (s *Store) RealtimeQueueForUser(ctx context.Context, userID string) (*models.RealtimeQueueEntry, error) {
	if s.pg != nil {
		var e models.RealtimeQueueEntry
		err := s.pg.QueryRowContext(ctx, `SELECT id,user_id,game_id,mode,wallet_category,region,jurisdiction,rating,latency_ms,priority,status,COALESCE(match_id,''),created_at,expires_at,updated_at FROM realtime_queue WHERE user_id=$1 ORDER BY created_at DESC LIMIT 1`, userID).
			Scan(&e.ID, &e.UserID, &e.GameID, &e.Mode, &e.WalletCategory, &e.Region, &e.Jurisdiction, &e.Rating, &e.LatencyMS, &e.Priority, &e.Status, &e.MatchID, &e.CreatedAt, &e.ExpiresAt, &e.UpdatedAt)
		return &e, err
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	var found *models.RealtimeQueueEntry
	for _, e := range s.realtimeQueue {
		if e.UserID == userID && (found == nil || e.CreatedAt.After(found.CreatedAt)) {
			copyEntry := *e
			found = &copyEntry
		}
	}
	if found == nil {
		return nil, sql.ErrNoRows
	}
	return found, nil
}

func (s *Store) SavePresence(ctx context.Context, p models.PresenceRecord) error {
	if s.pg != nil {
		_, err := s.pg.ExecContext(ctx, `INSERT INTO realtime_presence(user_id,state,session_id,connection_id,match_id,region,last_heartbeat,expires_at)
VALUES($1,$2,$3,$4,$5,$6,$7,$8) ON CONFLICT(user_id) DO UPDATE SET state=EXCLUDED.state,session_id=EXCLUDED.session_id,connection_id=EXCLUDED.connection_id,match_id=EXCLUDED.match_id,region=EXCLUDED.region,last_heartbeat=EXCLUDED.last_heartbeat,expires_at=EXCLUDED.expires_at`,
			p.UserID, p.State, p.SessionID, p.ConnectionID, p.MatchID, p.Region, p.LastHeartbeat, p.ExpiresAt)
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	copyPresence := p
	s.realtimePresence[p.UserID] = &copyPresence
	return nil
}

func (s *Store) GetPresence(ctx context.Context, userID string) (*models.PresenceRecord, error) {
	if s.pg != nil {
		var p models.PresenceRecord
		err := s.pg.QueryRowContext(ctx, `SELECT user_id,state,session_id,connection_id,match_id,region,last_heartbeat,expires_at FROM realtime_presence WHERE user_id=$1`, userID).
			Scan(&p.UserID, &p.State, &p.SessionID, &p.ConnectionID, &p.MatchID, &p.Region, &p.LastHeartbeat, &p.ExpiresAt)
		return &p, err
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	p := s.realtimePresence[userID]
	if p == nil {
		return nil, sql.ErrNoRows
	}
	copyPresence := *p
	return &copyPresence, nil
}

func (s *Store) AppendRealtimeEvent(ctx context.Context, matchID, userID, eventType string, payload json.RawMessage) (*models.RealtimeEvent, error) {
	if len(payload) == 0 {
		payload = json.RawMessage(`{}`)
	}
	if s.pg != nil {
		tx, err := s.pg.BeginTx(ctx, nil)
		if err != nil {
			return nil, err
		}
		defer tx.Rollback()
		var sequence, version int64
		if err := tx.QueryRowContext(ctx, `UPDATE realtime_matches SET sequence=sequence+1,updated_at=$2 WHERE id=$1 RETURNING sequence,state_version`, matchID, time.Now().UTC()).Scan(&sequence, &version); err != nil {
			return nil, err
		}
		var previous string
		_ = tx.QueryRowContext(ctx, `SELECT integrity_hash FROM realtime_events WHERE match_id=$1 ORDER BY sequence DESC LIMIT 1`, matchID).Scan(&previous)
		event := s.newRealtimeEvent(matchID, userID, eventType, sequence, version, payload, previous)
		_, err = tx.ExecContext(ctx, `INSERT INTO realtime_events(id,match_id,user_id,type,sequence,state_version,server_time,payload,previous_hash,integrity_hash) VALUES($1,$2,NULLIF($3,''),$4,$5,$6,$7,$8,$9,$10)`,
			event.ID, event.MatchID, event.UserID, event.Type, event.Sequence, event.StateVersion, event.ServerTime, event.Payload, event.PreviousHash, event.IntegrityHash)
		if err != nil {
			return nil, err
		}
		if err := tx.Commit(); err != nil {
			return nil, err
		}
		return &event, nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	match := s.realtimeMatches[matchID]
	if match == nil {
		return nil, sql.ErrNoRows
	}
	match.Sequence++
	previous := ""
	events := s.realtimeEvents[matchID]
	if len(events) > 0 {
		previous = events[len(events)-1].IntegrityHash
	}
	event := s.newRealtimeEvent(matchID, userID, eventType, match.Sequence, match.StateVersion, payload, previous)
	s.realtimeEvents[matchID] = append(events, event)
	return &event, nil
}

func (s *Store) newRealtimeEvent(matchID, userID, eventType string, sequence, version int64, payload json.RawMessage, previous string) models.RealtimeEvent {
	event := models.RealtimeEvent{ID: id.New("evt"), MatchID: matchID, UserID: userID, Type: eventType, Sequence: sequence, StateVersion: version, ServerTime: time.Now().UTC(), Payload: payload, PreviousHash: previous}
	key := []byte("development-realtime-integrity-key")
	if s.settings != nil && s.settings.Security.PuzzleSecret != "" {
		key = []byte(s.settings.Security.PuzzleSecret)
	}
	mac := hmac.New(sha256.New, key)
	_, _ = fmt.Fprintf(mac, "%s|%s|%s|%d|%d|%s|%s", event.MatchID, event.UserID, event.Type, event.Sequence, event.StateVersion, previous, string(payload))
	event.IntegrityHash = hex.EncodeToString(mac.Sum(nil))
	return event
}

func (s *Store) RealtimeEventsAfter(ctx context.Context, matchID string, after int64, limit int) ([]models.RealtimeEvent, error) {
	if limit <= 0 || limit > 500 {
		limit = 200
	}
	if s.pg != nil {
		rows, err := s.pg.QueryContext(ctx, `SELECT id,match_id,COALESCE(user_id,''),type,sequence,state_version,server_time,payload,previous_hash,integrity_hash FROM realtime_events WHERE match_id=$1 AND sequence>$2 ORDER BY sequence LIMIT $3`, matchID, after, limit)
		if err != nil {
			return nil, err
		}
		defer rows.Close()
		var events []models.RealtimeEvent
		for rows.Next() {
			var e models.RealtimeEvent
			if err := rows.Scan(&e.ID, &e.MatchID, &e.UserID, &e.Type, &e.Sequence, &e.StateVersion, &e.ServerTime, &e.Payload, &e.PreviousHash, &e.IntegrityHash); err != nil {
				return nil, err
			}
			events = append(events, e)
		}
		return events, rows.Err()
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	var events []models.RealtimeEvent
	for _, e := range s.realtimeEvents[matchID] {
		if e.Sequence > after && len(events) < limit {
			events = append(events, e)
		}
	}
	return events, nil
}

func (s *Store) SaveRealtimeReplay(ctx context.Context, replay models.RealtimeReplay) error {
	if s.pg != nil {
		_, err := s.pg.ExecContext(ctx, `INSERT INTO realtime_replays(id,match_id,game_id,game_version,rules_version,protocol_version,replay_version,first_sequence,last_sequence,event_count,event_root_hash,signature,storage_key,status,created_at)
VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15)
ON CONFLICT(match_id) DO UPDATE SET event_count=EXCLUDED.event_count,last_sequence=EXCLUDED.last_sequence,event_root_hash=EXCLUDED.event_root_hash,signature=EXCLUDED.signature,storage_key=EXCLUDED.storage_key,status=EXCLUDED.status`,
			replay.ID, replay.MatchID, replay.GameID, replay.GameVersion, replay.RulesVersion, replay.ProtocolVersion, replay.ReplayVersion, replay.FirstSequence, replay.LastSequence, replay.EventCount, replay.EventRootHash, replay.Signature, replay.StorageKey, replay.Status, replay.CreatedAt)
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	copyReplay := replay
	s.realtimeReplays[replay.MatchID] = &copyReplay
	return nil
}

func (s *Store) SaveRealtimeSnapshot(ctx context.Context, snapshot models.RealtimeSnapshot) error {
	if s.pg != nil {
		_, err := s.pg.ExecContext(ctx, `INSERT INTO realtime_snapshots(match_id,version,sequence,state,checksum,created_at)
VALUES($1,$2,$3,$4,$5,$6) ON CONFLICT(match_id,version) DO NOTHING`,
			snapshot.MatchID, snapshot.Version, snapshot.Sequence, snapshot.State, snapshot.Checksum, snapshot.CreatedAt)
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.realtimeSnapshots[snapshot.MatchID] = append(s.realtimeSnapshots[snapshot.MatchID], snapshot)
	return nil
}

func (s *Store) LatestRealtimeSnapshot(ctx context.Context, matchID string) (*models.RealtimeSnapshot, error) {
	if s.pg != nil {
		var snapshot models.RealtimeSnapshot
		err := s.pg.QueryRowContext(ctx, `SELECT match_id,version,sequence,state,checksum,created_at FROM realtime_snapshots WHERE match_id=$1 ORDER BY version DESC LIMIT 1`, matchID).
			Scan(&snapshot.MatchID, &snapshot.Version, &snapshot.Sequence, &snapshot.State, &snapshot.Checksum, &snapshot.CreatedAt)
		return &snapshot, err
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	snapshots := s.realtimeSnapshots[matchID]
	if len(snapshots) == 0 {
		return nil, sql.ErrNoRows
	}
	copySnapshot := snapshots[len(snapshots)-1]
	return &copySnapshot, nil
}

func (s *Store) GetRealtimeReplay(ctx context.Context, matchID string) (*models.RealtimeReplay, error) {
	if s.pg != nil {
		var r models.RealtimeReplay
		err := s.pg.QueryRowContext(ctx, `SELECT id,match_id,game_id,game_version,rules_version,protocol_version,replay_version,first_sequence,last_sequence,event_count,event_root_hash,signature,storage_key,status,created_at FROM realtime_replays WHERE match_id=$1`, matchID).
			Scan(&r.ID, &r.MatchID, &r.GameID, &r.GameVersion, &r.RulesVersion, &r.ProtocolVersion, &r.ReplayVersion, &r.FirstSequence, &r.LastSequence, &r.EventCount, &r.EventRootHash, &r.Signature, &r.StorageKey, &r.Status, &r.CreatedAt)
		return &r, err
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	r := s.realtimeReplays[matchID]
	if r == nil {
		return nil, sql.ErrNoRows
	}
	copyReplay := *r
	return &copyReplay, nil
}

func (s *Store) RealtimeMetrics(ctx context.Context) (models.RealtimeMetrics, error) {
	m := models.RealtimeMetrics{CheckedAt: time.Now().UTC()}
	if s.pg != nil {
		err := s.pg.QueryRowContext(ctx, `SELECT
(SELECT count(*) FROM realtime_presence WHERE connection_id<>'' AND expires_at>now()),
(SELECT count(*) FROM realtime_presence WHERE state <> 'offline' AND expires_at>now()),
(SELECT count(*) FROM realtime_queue WHERE status='waiting' AND expires_at>now()),
(SELECT count(*) FROM realtime_matches WHERE status IN ('ready','starting','live','paused','reconnecting')),
(SELECT count(*) FROM realtime_events WHERE type='participant_reconnected'),
(SELECT count(*) FROM realtime_matches),
(SELECT count(*) FROM realtime_events WHERE type='match_error'),
(SELECT count(*) FROM realtime_replays WHERE status='pending'),
COALESCE((SELECT avg(latency_ms) FROM realtime_participants WHERE left_at IS NULL),0),
COALESCE((SELECT EXTRACT(EPOCH FROM now()-min(created_at))::bigint FROM realtime_queue WHERE status='waiting' AND expires_at>now()),0)`).
			Scan(&m.Connections, &m.OnlinePlayers, &m.QueuedPlayers, &m.ActiveMatches, &m.Reconnects,
				&m.MatchesCreated, &m.MatchErrors, &m.ReplayBacklog, &m.GatewayLatencyMS, &m.OldestQueueSecond)
		return m, err
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	now := time.Now().UTC()
	for _, p := range s.realtimePresence {
		if p.State != models.PresenceOffline && p.ExpiresAt.After(now) {
			m.OnlinePlayers++
			if p.ConnectionID != "" {
				m.Connections++
			}
		}
	}
	for _, q := range s.realtimeQueue {
		if q.Status == models.QueueWaiting && q.ExpiresAt.After(now) {
			m.QueuedPlayers++
		}
	}
	for _, match := range s.realtimeMatches {
		m.MatchesCreated++
		switch match.Status {
		case models.MatchReady, models.MatchStarting, models.MatchLive, models.MatchPaused, models.MatchReconnecting:
			m.ActiveMatches++
		}
	}
	var latencyTotal, latencyCount int
	for _, match := range s.realtimeMatches {
		for _, participant := range match.Participants {
			if participant.LeftAt == nil {
				latencyTotal += participant.LatencyMS
				latencyCount++
			}
		}
	}
	if latencyCount > 0 {
		m.GatewayLatencyMS = float64(latencyTotal) / float64(latencyCount)
	}
	for _, events := range s.realtimeEvents {
		for _, event := range events {
			if event.Type == "participant_reconnected" {
				m.Reconnects++
			}
			if event.Type == "match_error" {
				m.MatchErrors++
			}
		}
	}
	return m, nil
}

func (s *Store) RunRealtimeMaintenance(ctx context.Context, now time.Time) (map[string]int64, error) {
	counts := map[string]int64{"queuesExpired": 0, "presenceExpired": 0, "matchesAbandoned": 0}
	reconnectWindow := 30 * time.Second
	if s.settings != nil && s.settings.Realtime.ReconnectWindow > 0 {
		reconnectWindow = s.settings.Realtime.ReconnectWindow
	}
	if s.pg != nil {
		tx, err := s.pg.BeginTx(ctx, nil)
		if err != nil {
			return nil, err
		}
		defer tx.Rollback()
		result, err := tx.ExecContext(ctx, `UPDATE realtime_queue SET status='expired',updated_at=$1 WHERE status='waiting' AND expires_at<=$1`, now)
		if err != nil {
			return nil, err
		}
		counts["queuesExpired"], _ = result.RowsAffected()
		result, err = tx.ExecContext(ctx, `UPDATE realtime_presence SET state='offline',connection_id='',expires_at=$1 WHERE state<>'offline' AND expires_at<=$1`, now)
		if err != nil {
			return nil, err
		}
		counts["presenceExpired"], _ = result.RowsAffected()
		result, err = tx.ExecContext(ctx, `UPDATE realtime_matches SET status='abandoned',state_version=state_version+1,completed_at=$1,updated_at=$1 WHERE status='reconnecting' AND updated_at<=$2`, now, now.Add(-reconnectWindow))
		if err != nil {
			return nil, err
		}
		counts["matchesAbandoned"], _ = result.RowsAffected()
		return counts, tx.Commit()
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, q := range s.realtimeQueue {
		if q.Status == models.QueueWaiting && !q.ExpiresAt.After(now) {
			q.Status, q.UpdatedAt = models.QueueExpired, now
			counts["queuesExpired"]++
		}
	}
	for _, p := range s.realtimePresence {
		if p.State != models.PresenceOffline && !p.ExpiresAt.After(now) {
			p.State, p.ConnectionID, p.ExpiresAt = models.PresenceOffline, "", now
			counts["presenceExpired"]++
		}
	}
	for _, match := range s.realtimeMatches {
		if match.Status == models.MatchReconnecting && !match.UpdatedAt.After(now.Add(-reconnectWindow)) {
			match.Status, match.UpdatedAt = models.MatchAbandoned, now
			match.StateVersion++
			match.CompletedAt = &now
			counts["matchesAbandoned"]++
		}
	}
	return counts, nil
}

func cloneRealtimeMatch(match *models.RealtimeMatch) *models.RealtimeMatch {
	if match == nil {
		return nil
	}
	copyMatch := *match
	copyMatch.Participants = append([]models.RealtimeParticipant(nil), match.Participants...)
	return &copyMatch
}
