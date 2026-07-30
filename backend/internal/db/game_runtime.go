package db

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/lib/pq"

	"skill-arena/internal/models"
)

func gameStateKey(matchID, userID string) string {
	return matchID + "\x1f" + userID
}

type GameActionContext struct {
	Match          models.RealtimeMatch
	State          models.GameParticipantState
	Receipt        *models.GameActionReceipt
	StreamSequence int64
	StreamHash     string
}

func (s *Store) CreateGameParticipantStates(
	ctx context.Context,
	states []models.GameParticipantState,
	event models.GameEventDraft,
) ([]models.RealtimeEvent, error) {
	if len(states) == 0 || event.Type == "" {
		return nil, errors.New("participant states and initial event are required")
	}
	if s.pg != nil {
		tx, err := s.pg.BeginTx(ctx, nil)
		if err != nil {
			return nil, err
		}
		defer tx.Rollback()
		for _, state := range states {
			if err := insertGameParticipantState(ctx, tx, state); err != nil {
				return nil, err
			}
		}
		events, err := s.appendGameEventsPostgres(ctx, tx, states[0].MatchID, "", 0, []models.GameEventDraft{event})
		if err != nil {
			return nil, err
		}
		if err := tx.Commit(); err != nil {
			return nil, err
		}
		return events, nil
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	for _, state := range states {
		key := gameStateKey(state.MatchID, state.UserID)
		if _, exists := s.gameParticipantStates[key]; exists {
			return nil, ErrRealtimeConflict
		}
	}
	for _, state := range states {
		copyState := state
		copyState.State = append(json.RawMessage(nil), state.State...)
		s.gameParticipantStates[gameStateKey(state.MatchID, state.UserID)] = &copyState
	}
	events, err := s.appendGameEventsMemory(states[0].MatchID, "", 0, []models.GameEventDraft{event})
	if err != nil {
		return nil, err
	}
	return events, nil
}

func insertGameParticipantState(ctx context.Context, tx *sql.Tx, state models.GameParticipantState) error {
	_, err := tx.ExecContext(ctx, `INSERT INTO game_participant_states(
match_id,user_id,game_id,puzzle_id,state_schema_version,state_version,state,state_checksum,
last_client_sequence,last_server_sequence,status,updated_at)
VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12)`,
		state.MatchID, state.UserID, state.GameID, state.PuzzleID, state.StateSchema,
		state.StateVersion, state.State, state.StateChecksum, state.LastClientSequence,
		state.LastServerSequence, state.Status, state.UpdatedAt)
	return err
}

func (s *Store) GetGameParticipantState(ctx context.Context, matchID, userID string) (*models.GameParticipantState, error) {
	if s.pg != nil {
		var state models.GameParticipantState
		err := s.pg.QueryRowContext(ctx, `SELECT match_id,user_id,game_id,puzzle_id,
state_schema_version,state_version,state,state_checksum,last_client_sequence,
last_server_sequence,status,updated_at FROM game_participant_states
WHERE match_id=$1 AND user_id=$2`, matchID, userID).Scan(
			&state.MatchID, &state.UserID, &state.GameID, &state.PuzzleID,
			&state.StateSchema, &state.StateVersion, &state.State, &state.StateChecksum,
			&state.LastClientSequence, &state.LastServerSequence, &state.Status, &state.UpdatedAt,
		)
		return &state, err
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	state := s.gameParticipantStates[gameStateKey(matchID, userID)]
	if state == nil {
		return nil, sql.ErrNoRows
	}
	copyState := *state
	copyState.State = append(json.RawMessage(nil), state.State...)
	return &copyState, nil
}

func (s *Store) ListGameParticipantStates(ctx context.Context, matchID string) ([]models.GameParticipantState, error) {
	if s.pg != nil {
		rows, err := s.pg.QueryContext(ctx, `SELECT match_id,user_id,game_id,puzzle_id,
state_schema_version,state_version,state,state_checksum,last_client_sequence,
last_server_sequence,status,updated_at FROM game_participant_states
WHERE match_id=$1 ORDER BY user_id`, matchID)
		if err != nil {
			return nil, err
		}
		defer rows.Close()
		var states []models.GameParticipantState
		for rows.Next() {
			var state models.GameParticipantState
			if err := rows.Scan(
				&state.MatchID, &state.UserID, &state.GameID, &state.PuzzleID,
				&state.StateSchema, &state.StateVersion, &state.State, &state.StateChecksum,
				&state.LastClientSequence, &state.LastServerSequence, &state.Status, &state.UpdatedAt,
			); err != nil {
				return nil, err
			}
			states = append(states, state)
		}
		return states, rows.Err()
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	states := make([]models.GameParticipantState, 0)
	for _, state := range s.gameParticipantStates {
		if state.MatchID == matchID {
			copyState := *state
			copyState.State = append(json.RawMessage(nil), state.State...)
			states = append(states, copyState)
		}
	}
	return states, nil
}

func (s *Store) ListActiveGameParticipantStates(ctx context.Context) ([]models.GameParticipantState, error) {
	if s.pg != nil {
		rows, err := s.pg.QueryContext(ctx, `SELECT match_id,user_id,game_id,puzzle_id,
state_schema_version,state_version,state,state_checksum,last_client_sequence,
last_server_sequence,status,updated_at FROM game_participant_states
WHERE status='active' ORDER BY match_id,user_id`)
		if err != nil {
			return nil, err
		}
		defer rows.Close()
		var states []models.GameParticipantState
		for rows.Next() {
			var state models.GameParticipantState
			if err := rows.Scan(
				&state.MatchID, &state.UserID, &state.GameID, &state.PuzzleID,
				&state.StateSchema, &state.StateVersion, &state.State, &state.StateChecksum,
				&state.LastClientSequence, &state.LastServerSequence, &state.Status, &state.UpdatedAt,
			); err != nil {
				return nil, err
			}
			states = append(states, state)
		}
		return states, rows.Err()
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	states := make([]models.GameParticipantState, 0)
	for _, state := range s.gameParticipantStates {
		if state.Status == "active" {
			copyState := *state
			copyState.State = append(json.RawMessage(nil), state.State...)
			states = append(states, copyState)
		}
	}
	return states, nil
}

func (s *Store) GetGameActionReceipt(
	ctx context.Context,
	matchID, userID, actionID string,
	clientSequence int64,
) (*models.GameActionReceipt, error) {
	if s.pg != nil {
		var receipt models.GameActionReceipt
		err := s.pg.QueryRowContext(ctx, `SELECT action_id,match_id,user_id,client_sequence,
expected_state_version,action_kind,action_payload_hash,accepted,result_code,
state_version_before,state_version_after,first_event_sequence,last_event_sequence,
transition,receipt_hash,server_received_at,processed_at FROM game_action_receipts
WHERE (action_id=$1) OR (match_id=$2 AND user_id=$3 AND client_sequence=$4)
ORDER BY CASE WHEN action_id=$1 THEN 0 ELSE 1 END LIMIT 1`,
			actionID, matchID, userID, clientSequence).Scan(
			&receipt.ActionID, &receipt.MatchID, &receipt.UserID, &receipt.ClientSequence,
			&receipt.ExpectedStateVersion, &receipt.ActionKind, &receipt.ActionPayloadHash,
			&receipt.Accepted, &receipt.ResultCode, &receipt.StateVersionBefore,
			&receipt.StateVersionAfter, &receipt.FirstEventSequence, &receipt.LastEventSequence,
			&receipt.Transition, &receipt.ReceiptHash, &receipt.ServerReceivedAt, &receipt.ProcessedAt,
		)
		return &receipt, err
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, receipt := range s.gameActionReceipts {
		if receipt.ActionID == actionID ||
			(receipt.MatchID == matchID && receipt.UserID == userID &&
				receipt.ClientSequence == clientSequence) {
			copyReceipt := *receipt
			copyReceipt.Transition = append(json.RawMessage(nil), receipt.Transition...)
			return &copyReceipt, nil
		}
	}
	return nil, sql.ErrNoRows
}

func (s *Store) GetGameActionContext(
	ctx context.Context,
	matchID, userID, actionID string,
	clientSequence int64,
) (GameActionContext, error) {
	if s.pg != nil {
		var result GameActionContext
		var receiptJSON []byte
		err := s.pg.QueryRowContext(ctx, `SELECT
m.id,m.game_id,m.game_version,m.rules_version,m.protocol_version,m.replay_version,
m.mode,m.status,m.region,m.wallet_category,m.seed_reference,m.state_version,m.sequence,
m.created_at,m.updated_at,m.started_at,m.completed_at,
s.match_id,s.user_id,s.game_id,s.puzzle_id,s.state_schema_version,s.state_version,
s.state,s.state_checksum,s.last_client_sequence,s.last_server_sequence,s.status,s.updated_at,
COALESCE((SELECT integrity_hash FROM realtime_events
          WHERE match_id=m.id ORDER BY sequence DESC LIMIT 1),''),
COALESCE((
    SELECT jsonb_build_object(
        'actionId',r.action_id,'matchId',r.match_id,'userId',r.user_id,
        'clientSequence',r.client_sequence,
        'expectedStateVersion',r.expected_state_version,
        'actionKind',r.action_kind,'actionPayloadHash',r.action_payload_hash,
        'accepted',r.accepted,'resultCode',r.result_code,
        'stateVersionBefore',r.state_version_before,
        'stateVersionAfter',r.state_version_after,
        'firstEventSequence',r.first_event_sequence,
        'lastEventSequence',r.last_event_sequence,
        'transition',r.transition,'receiptHash',r.receipt_hash,
        'serverReceivedAt',r.server_received_at,'processedAt',r.processed_at
    )
    FROM game_action_receipts r
    WHERE r.action_id=$3
       OR (r.match_id=$1 AND r.user_id=$2 AND r.client_sequence=$4)
    ORDER BY CASE WHEN r.action_id=$3 THEN 0 ELSE 1 END
    LIMIT 1
),'null'::jsonb)
FROM game_participant_states s
JOIN realtime_matches m ON m.id=s.match_id
JOIN realtime_participants p ON p.match_id=m.id AND p.user_id=s.user_id
WHERE s.match_id=$1 AND s.user_id=$2`,
			matchID, userID, actionID, clientSequence,
		).Scan(
			&result.Match.ID, &result.Match.GameID, &result.Match.GameVersion,
			&result.Match.RulesVersion, &result.Match.ProtocolVersion,
			&result.Match.ReplayVersion, &result.Match.Mode, &result.Match.Status,
			&result.Match.Region, &result.Match.WalletCategory,
			&result.Match.SeedReference, &result.Match.StateVersion,
			&result.Match.Sequence, &result.Match.CreatedAt, &result.Match.UpdatedAt,
			&result.Match.StartedAt, &result.Match.CompletedAt,
			&result.State.MatchID, &result.State.UserID, &result.State.GameID,
			&result.State.PuzzleID, &result.State.StateSchema, &result.State.StateVersion,
			&result.State.State, &result.State.StateChecksum,
			&result.State.LastClientSequence, &result.State.LastServerSequence,
			&result.State.Status, &result.State.UpdatedAt, &result.StreamHash, &receiptJSON,
		)
		if err != nil {
			return GameActionContext{}, err
		}
		result.StreamSequence = result.Match.Sequence
		if string(receiptJSON) != "null" {
			var receipt models.GameActionReceipt
			if err := json.Unmarshal(receiptJSON, &receipt); err != nil {
				return GameActionContext{}, err
			}
			result.Receipt = &receipt
		}
		return result, nil
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	state := s.gameParticipantStates[gameStateKey(matchID, userID)]
	if state == nil {
		return GameActionContext{}, sql.ErrNoRows
	}
	result := GameActionContext{State: *state}
	result.State.State = append(json.RawMessage(nil), state.State...)
	if match := s.realtimeMatches[matchID]; match != nil {
		result.Match = *cloneRealtimeMatch(match)
		result.StreamSequence = match.Sequence
	}
	if events := s.realtimeEvents[matchID]; len(events) > 0 {
		result.StreamHash = events[len(events)-1].IntegrityHash
	}
	for _, receipt := range s.gameActionReceipts {
		if receipt.ActionID == actionID ||
			(receipt.MatchID == matchID && receipt.UserID == userID &&
				receipt.ClientSequence == clientSequence) {
			copyReceipt := *receipt
			copyReceipt.Transition = append(json.RawMessage(nil), receipt.Transition...)
			result.Receipt = &copyReceipt
			break
		}
	}
	return result, nil
}

func (s *Store) CommitGameAction(
	ctx context.Context,
	expected models.GameParticipantState,
	next models.GameParticipantState,
	receipt models.GameActionReceipt,
	expectedStreamSequence int64,
	expectedStreamHash string,
	drafts []models.GameEventDraft,
) (*models.GameActionReceipt, []models.RealtimeEvent, error) {
	if len(drafts) == 0 {
		return nil, nil, errors.New("game action must emit at least one event")
	}
	if s.pg != nil {
		return s.commitGameActionPostgres(
			ctx, expected, next, receipt, expectedStreamSequence, expectedStreamHash, drafts,
		)
	}
	return s.commitGameActionMemory(
		expected, next, receipt, expectedStreamSequence, expectedStreamHash, drafts,
	)
}

func (s *Store) CommitGameSystemTransition(
	ctx context.Context,
	expected models.GameParticipantState,
	next models.GameParticipantState,
	drafts []models.GameEventDraft,
) ([]models.RealtimeEvent, error) {
	if len(drafts) == 0 {
		return nil, errors.New("game system transition must emit at least one event")
	}
	if s.pg != nil {
		tx, err := s.pg.BeginTx(ctx, nil)
		if err != nil {
			return nil, err
		}
		defer tx.Rollback()
		var version, clientSequence int64
		err = tx.QueryRowContext(ctx, `SELECT state_version,last_client_sequence
FROM game_participant_states WHERE match_id=$1 AND user_id=$2 FOR UPDATE`,
			expected.MatchID, expected.UserID).Scan(&version, &clientSequence)
		if err != nil {
			return nil, err
		}
		if version != expected.StateVersion || clientSequence != expected.LastClientSequence {
			return nil, ErrRealtimeConflict
		}
		events, err := s.appendGameEventsPostgres(
			ctx, tx, expected.MatchID, expected.UserID, next.StateVersion, drafts,
		)
		if err != nil {
			return nil, err
		}
		next.LastServerSequence = events[len(events)-1].Sequence
		result, err := tx.ExecContext(ctx, `UPDATE game_participant_states SET
state_schema_version=$1,state_version=$2,state=$3,state_checksum=$4,
last_server_sequence=$5,status=$6,updated_at=$7
WHERE match_id=$8 AND user_id=$9 AND state_version=$10 AND last_client_sequence=$11`,
			next.StateSchema, next.StateVersion, next.State, next.StateChecksum,
			next.LastServerSequence, next.Status, next.UpdatedAt, next.MatchID,
			next.UserID, expected.StateVersion, expected.LastClientSequence)
		if err != nil {
			return nil, err
		}
		affected, _ := result.RowsAffected()
		if affected != 1 {
			return nil, ErrRealtimeConflict
		}
		if err := tx.Commit(); err != nil {
			return nil, err
		}
		return events, nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	key := gameStateKey(expected.MatchID, expected.UserID)
	current := s.gameParticipantStates[key]
	if current == nil {
		return nil, sql.ErrNoRows
	}
	if current.StateVersion != expected.StateVersion ||
		current.LastClientSequence != expected.LastClientSequence {
		return nil, ErrRealtimeConflict
	}
	events, err := s.appendGameEventsMemory(
		expected.MatchID, expected.UserID, next.StateVersion, drafts,
	)
	if err != nil {
		return nil, err
	}
	next.LastServerSequence = events[len(events)-1].Sequence
	copyState := next
	copyState.State = append(json.RawMessage(nil), next.State...)
	s.gameParticipantStates[key] = &copyState
	return events, nil
}

func (s *Store) commitGameActionPostgres(
	ctx context.Context,
	expected, next models.GameParticipantState,
	receipt models.GameActionReceipt,
	expectedStreamSequence int64,
	expectedStreamHash string,
	drafts []models.GameEventDraft,
) (*models.GameActionReceipt, []models.RealtimeEvent, error) {
	sequence := expectedStreamSequence
	previous := expectedStreamHash
	events := make([]models.RealtimeEvent, 0, len(drafts))
	for _, draft := range drafts {
		sequence++
		event := s.newRealtimeEvent(
			expected.MatchID, expected.UserID, draft.Type, sequence,
			next.StateVersion, draft.Payload, previous,
		)
		events = append(events, event)
		previous = event.IntegrityHash
	}
	next.LastServerSequence = events[len(events)-1].Sequence
	receipt.FirstEventSequence = events[0].Sequence
	receipt.LastEventSequence = events[len(events)-1].Sequence
	tx, err := s.pg.BeginTx(ctx, nil)
	if err != nil {
		return nil, nil, err
	}
	defer tx.Rollback()
	if err := insertGameEvents(ctx, tx, events); err != nil {
		if isPostgresUniqueViolation(err) {
			return nil, nil, ErrRealtimeConflict
		}
		return nil, nil, err
	}
	result, err := tx.ExecContext(ctx, `WITH match_update AS (
    UPDATE realtime_matches SET sequence=$1,updated_at=$2
    WHERE id=$3 AND sequence=$4
      AND COALESCE((SELECT integrity_hash FROM realtime_events
                    WHERE match_id=$3 AND sequence<=$4
                    ORDER BY sequence DESC LIMIT 1),'')=$5
    RETURNING id
), state_update AS (
    UPDATE game_participant_states SET
    state_schema_version=$6,state_version=$7,state=$8,state_checksum=$9,
    last_client_sequence=$10,last_server_sequence=$11,status=$12,updated_at=$13
    WHERE match_id=$14 AND user_id=$15 AND state_version=$16
      AND last_client_sequence=$17
      AND EXISTS (SELECT 1 FROM match_update)
    RETURNING 1
)
INSERT INTO game_action_receipts(
action_id,match_id,user_id,client_sequence,expected_state_version,action_kind,
action_payload_hash,accepted,result_code,state_version_before,state_version_after,
first_event_sequence,last_event_sequence,transition,receipt_hash,server_received_at,processed_at)
SELECT $18,$19,$20,$21,$22,$23,$24,$25,$26,$27,$28,$29,$30,$31,$32,$33,$34
FROM state_update`,
		sequence, time.Now().UTC(), expected.MatchID, expectedStreamSequence,
		expectedStreamHash, next.StateSchema, next.StateVersion, next.State,
		next.StateChecksum, next.LastClientSequence, next.LastServerSequence,
		next.Status, next.UpdatedAt, next.MatchID, next.UserID,
		expected.StateVersion, expected.LastClientSequence,
		receipt.ActionID, receipt.MatchID, receipt.UserID, receipt.ClientSequence,
		receipt.ExpectedStateVersion, receipt.ActionKind, receipt.ActionPayloadHash,
		receipt.Accepted, receipt.ResultCode, receipt.StateVersionBefore,
		receipt.StateVersionAfter, receipt.FirstEventSequence, receipt.LastEventSequence,
		receipt.Transition, receipt.ReceiptHash, receipt.ServerReceivedAt,
		receipt.ProcessedAt,
	)
	if err != nil {
		if isPostgresUniqueViolation(err) {
			return nil, nil, ErrRealtimeConflict
		}
		return nil, nil, err
	}
	affected, _ := result.RowsAffected()
	if affected != 1 {
		return nil, nil, ErrRealtimeConflict
	}
	if err := tx.Commit(); err != nil {
		return nil, nil, err
	}
	return &receipt, events, nil
}

func isPostgresUniqueViolation(err error) bool {
	var pqErr *pq.Error
	var pgxErr *pgconn.PgError
	return (errors.As(err, &pqErr) && pqErr.Code == "23505") ||
		(errors.As(err, &pgxErr) && pgxErr.Code == "23505")
}

func insertGameEvents(
	ctx context.Context,
	tx *sql.Tx,
	events []models.RealtimeEvent,
) error {
	const fields = 10
	values := make([]string, len(events))
	args := make([]any, 0, len(events)*fields)
	for index, event := range events {
		base := index*fields + 1
		placeholders := make([]string, fields)
		for field := range fields {
			placeholders[field] = fmt.Sprintf("$%d", base+field)
		}
		placeholders[2] = "NULLIF(" + placeholders[2] + ",'')"
		values[index] = "(" + strings.Join(placeholders, ",") + ")"
		args = append(args,
			event.ID, event.MatchID, event.UserID, event.Type, event.Sequence,
			event.StateVersion, event.ServerTime, event.Payload, event.PreviousHash,
			event.IntegrityHash,
		)
	}
	_, err := tx.ExecContext(ctx, `INSERT INTO realtime_events(
id,match_id,user_id,type,sequence,state_version,server_time,payload,previous_hash,integrity_hash)
VALUES `+strings.Join(values, ","), args...)
	return err
}

func insertGameActionReceipt(ctx context.Context, tx *sql.Tx, receipt models.GameActionReceipt) error {
	_, err := tx.ExecContext(ctx, `INSERT INTO game_action_receipts(
action_id,match_id,user_id,client_sequence,expected_state_version,action_kind,
action_payload_hash,accepted,result_code,state_version_before,state_version_after,
first_event_sequence,last_event_sequence,transition,receipt_hash,server_received_at,processed_at)
VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17)`,
		receipt.ActionID, receipt.MatchID, receipt.UserID, receipt.ClientSequence,
		receipt.ExpectedStateVersion, receipt.ActionKind, receipt.ActionPayloadHash,
		receipt.Accepted, receipt.ResultCode, receipt.StateVersionBefore,
		receipt.StateVersionAfter, receipt.FirstEventSequence, receipt.LastEventSequence,
		receipt.Transition, receipt.ReceiptHash, receipt.ServerReceivedAt, receipt.ProcessedAt)
	return err
}

func (s *Store) commitGameActionMemory(
	expected, next models.GameParticipantState,
	receipt models.GameActionReceipt,
	expectedStreamSequence int64,
	expectedStreamHash string,
	drafts []models.GameEventDraft,
) (*models.GameActionReceipt, []models.RealtimeEvent, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	key := gameStateKey(expected.MatchID, expected.UserID)
	current := s.gameParticipantStates[key]
	if current == nil {
		return nil, nil, sql.ErrNoRows
	}
	if current.StateVersion != expected.StateVersion ||
		current.LastClientSequence != expected.LastClientSequence {
		return nil, nil, ErrRealtimeConflict
	}
	match := s.realtimeMatches[expected.MatchID]
	if match == nil || match.Sequence != expectedStreamSequence {
		return nil, nil, ErrRealtimeConflict
	}
	events := s.realtimeEvents[expected.MatchID]
	currentHash := ""
	if len(events) > 0 {
		currentHash = events[len(events)-1].IntegrityHash
	}
	if currentHash != expectedStreamHash {
		return nil, nil, ErrRealtimeConflict
	}
	for _, existing := range s.gameActionReceipts {
		if existing.ActionID == receipt.ActionID ||
			(existing.MatchID == receipt.MatchID && existing.UserID == receipt.UserID &&
				existing.ClientSequence == receipt.ClientSequence) {
			return nil, nil, ErrRealtimeConflict
		}
	}
	events, err := s.appendGameEventsMemory(
		expected.MatchID, expected.UserID, next.StateVersion, drafts,
	)
	if err != nil {
		return nil, nil, err
	}
	next.LastServerSequence = events[len(events)-1].Sequence
	copyState := next
	copyState.State = append(json.RawMessage(nil), next.State...)
	s.gameParticipantStates[key] = &copyState
	receipt.FirstEventSequence = events[0].Sequence
	receipt.LastEventSequence = events[len(events)-1].Sequence
	copyReceipt := receipt
	copyReceipt.Transition = append(json.RawMessage(nil), receipt.Transition...)
	s.gameActionReceipts[receipt.ActionID] = &copyReceipt
	return &copyReceipt, events, nil
}

func (s *Store) appendGameEventsPostgres(
	ctx context.Context,
	tx *sql.Tx,
	matchID, userID string,
	stateVersion int64,
	drafts []models.GameEventDraft,
) ([]models.RealtimeEvent, error) {
	var sequence int64
	if err := tx.QueryRowContext(ctx, `SELECT sequence FROM realtime_matches
WHERE id=$1 FOR UPDATE`, matchID).Scan(&sequence); err != nil {
		return nil, err
	}
	var previous string
	_ = tx.QueryRowContext(ctx, `SELECT integrity_hash FROM realtime_events
WHERE match_id=$1 ORDER BY sequence DESC LIMIT 1`, matchID).Scan(&previous)
	events := make([]models.RealtimeEvent, 0, len(drafts))
	for _, draft := range drafts {
		sequence++
		event := s.newRealtimeEvent(
			matchID, userID, draft.Type, sequence, stateVersion, draft.Payload, previous,
		)
		_, err := tx.ExecContext(ctx, `INSERT INTO realtime_events(
id,match_id,user_id,type,sequence,state_version,server_time,payload,previous_hash,integrity_hash)
VALUES($1,$2,NULLIF($3,''),$4,$5,$6,$7,$8,$9,$10)`,
			event.ID, event.MatchID, event.UserID, event.Type, event.Sequence,
			event.StateVersion, event.ServerTime, event.Payload, event.PreviousHash,
			event.IntegrityHash)
		if err != nil {
			return nil, err
		}
		events = append(events, event)
		previous = event.IntegrityHash
	}
	_, err := tx.ExecContext(ctx, `UPDATE realtime_matches SET sequence=$1,updated_at=$2
WHERE id=$3`, sequence, time.Now().UTC(), matchID)
	return events, err
}

func (s *Store) appendGameEventsMemory(
	matchID, userID string,
	stateVersion int64,
	drafts []models.GameEventDraft,
) ([]models.RealtimeEvent, error) {
	match := s.realtimeMatches[matchID]
	if match == nil {
		return nil, sql.ErrNoRows
	}
	events := s.realtimeEvents[matchID]
	previous := ""
	if len(events) > 0 {
		previous = events[len(events)-1].IntegrityHash
	}
	appended := make([]models.RealtimeEvent, 0, len(drafts))
	for _, draft := range drafts {
		match.Sequence++
		event := s.newRealtimeEvent(
			matchID, userID, draft.Type, match.Sequence, stateVersion, draft.Payload, previous,
		)
		events = append(events, event)
		appended = append(appended, event)
		previous = event.IntegrityHash
	}
	s.realtimeEvents[matchID] = events
	return appended, nil
}
