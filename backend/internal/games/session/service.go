package session

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"skill-arena/internal/db"
	"skill-arena/internal/games/interfaces"
	"skill-arena/internal/games/shared"
	"skill-arena/internal/id"
	"skill-arena/internal/models"
	"skill-arena/internal/observability"
)

var (
	ErrRuntimeUnavailable = errors.New("game runtime is unavailable")
	ErrActionConflict     = errors.New("game action conflicts with authoritative state")
	ErrActionSequence     = errors.New("game action sequence has a gap")
	ErrDuplicateMismatch  = errors.New("duplicate game action does not match its original request")
)

type Service struct {
	store       *db.Store
	actionCache sync.Map
	streamHeads sync.Map
	now         func() time.Time
}

type gameStreamHead struct {
	Sequence      int64
	IntegrityHash string
}

type ActionResult struct {
	Receipt    models.GameActionReceipt     `json:"receipt"`
	Snapshot   interfaces.RendererSnapshot  `json:"snapshot"`
	Events     []models.RealtimeEvent       `json:"events"`
	Completion *interfaces.CompletionResult `json:"completion,omitempty"`
	Outcome    *interfaces.MatchOutcome     `json:"outcome,omitempty"`
	Duplicate  bool                         `json:"duplicate"`
	Match      *models.RealtimeMatch        `json:"-"`
}

type SyncResult struct {
	Snapshot           interfaces.RendererSnapshot `json:"snapshot"`
	StateVersion       int64                       `json:"stateVersion"`
	LastClientSequence int64                       `json:"lastClientSequence"`
	LastServerSequence int64                       `json:"lastServerSequence"`
}

type ExpiredMatch struct {
	MatchID string
	Outcome interfaces.MatchOutcome
}

func New(store *db.Store) (*Service, error) {
	if store == nil || store.GamesRegistry() == nil {
		return nil, errors.New("game session dependencies are unavailable")
	}
	return &Service{store: store, now: time.Now}, nil
}

func (s *Service) PrepareMatch(ctx context.Context, match *models.RealtimeMatch) error {
	if match == nil || len(match.Participants) == 0 {
		return errors.New("realtime match and participants are required")
	}
	if _, err := s.store.GetGameParticipantState(
		ctx, match.ID, match.Participants[0].UserID,
	); err == nil {
		return nil
	} else if !errors.Is(err, sql.ErrNoRows) {
		return err
	}
	runtime, descriptor, err := s.resolve(match)
	if err != nil {
		return err
	}
	participantIDs := make([]string, len(match.Participants))
	for index, participant := range match.Participants {
		participantIDs[index] = participant.UserID
	}
	matchContext := runtimeMatchContext(match, descriptor, participantIDs, s.now().UTC())
	generated, err := runtime.GenerateState(ctx, matchContext, interfaces.GenerationRequest{
		Mode: match.Mode, SeedReference: match.SeedReference,
	})
	if err != nil {
		return fmt.Errorf("generate authoritative game state: %w", err)
	}
	sharedState, err := runtime.InitializeMatch(ctx, matchContext, interfaces.MatchRequest{
		Mode: match.Mode, Configuration: generated.Metadata,
	})
	if err != nil {
		return fmt.Errorf("initialize authoritative match: %w", err)
	}
	now := s.now().UTC()
	states := make([]models.GameParticipantState, 0, len(match.Participants))
	for _, participant := range match.Participants {
		gameState, stateErr := runtime.InitializeParticipant(ctx, interfaces.ParticipantContext{
			MatchID: match.ID, ParticipantID: participant.UserID,
			UserID: participant.UserID, ViewerRole: "player",
		}, sharedState)
		if stateErr != nil {
			return fmt.Errorf("initialize participant %s: %w", participant.UserID, stateErr)
		}
		encoded, stateErr := json.Marshal(gameState)
		if stateErr != nil {
			return stateErr
		}
		states = append(states, models.GameParticipantState{
			MatchID: match.ID, UserID: participant.UserID, GameID: match.GameID,
			PuzzleID: generated.Reference, StateSchema: gameState.SchemaVersion,
			StateVersion: gameState.Version, State: encoded,
			StateChecksum: gameState.Checksum, Status: "active", UpdatedAt: now,
		})
	}
	payload, err := json.Marshal(map[string]any{
		"gameId": match.GameID, "gameVersion": match.GameVersion,
		"puzzleReference":     generated.Reference,
		"sharedStateChecksum": sharedState.Checksum,
		"participantCount":    len(states),
	})
	if err != nil {
		return err
	}
	events, err := s.store.CreateGameParticipantStates(ctx, states, models.GameEventDraft{
		Type: "game.puzzle.ready", Payload: payload,
	})
	if errors.Is(err, db.ErrRealtimeConflict) {
		_, loadErr := s.store.GetGameParticipantState(ctx, match.ID, match.Participants[0].UserID)
		return loadErr
	}
	if err == nil && len(events) > 0 {
		head := events[len(events)-1]
		s.rememberStream(db.GameActionContext{
			Match: *match, StreamSequence: head.Sequence, StreamHash: head.IntegrityHash,
		})
		for _, state := range states {
			s.actionCache.Store(
				gameActionCacheKey(match.ID, state.UserID),
				db.GameActionContext{
					Match: *match, State: state,
					StreamSequence: head.Sequence, StreamHash: head.IntegrityHash,
				},
			)
		}
	}
	return err
}

func (s *Service) ObserveMatchTransition(
	match *models.RealtimeMatch,
	event models.RealtimeEvent,
) {
	if match == nil || event.MatchID != match.ID {
		return
	}
	s.rememberStream(db.GameActionContext{
		Match: *match, StreamSequence: event.Sequence, StreamHash: event.IntegrityHash,
	})
	for _, participant := range match.Participants {
		key := gameActionCacheKey(match.ID, participant.UserID)
		cached, ok := s.actionCache.Load(key)
		if !ok {
			continue
		}
		actionContext, valid := cached.(db.GameActionContext)
		if !valid {
			continue
		}
		actionContext.Match = *match
		actionContext.StreamSequence = event.Sequence
		actionContext.StreamHash = event.IntegrityHash
		s.actionCache.Store(key, actionContext)
	}
	if terminalMatchStatus(match.Status) {
		s.streamHeads.Delete(match.ID)
	}
}

func (s *Service) SubmitAction(
	ctx context.Context,
	matchID string,
	userID string,
	envelope interfaces.ActionEnvelope,
	latency time.Duration,
) (ActionResult, error) {
	totalStarted := time.Now()
	defer observability.ObserveTiming(ctx, "game_action.total", totalStarted)

	stageStarted := time.Now()
	if envelope.MatchID != matchID || strings.TrimSpace(userID) == "" {
		return ActionResult{}, ErrActionConflict
	}
	payloadHash := shared.HashFields(
		"skill-arena/game-action-request/v1", envelope.ActionID, envelope.MatchID,
		userID, envelope.Kind, string(envelope.Payload),
		fmt.Sprint(envelope.ClientSequence), fmt.Sprint(envelope.ExpectedStateVersion),
	)
	observability.ObserveTiming(ctx, "game_action.request_validation", stageStarted)

	stageStarted = time.Now()
	actionContext, err := s.gameActionContext(ctx, matchID, userID, envelope)
	observability.ObserveTiming(ctx, "game_action.load", stageStarted)
	if err != nil {
		return ActionResult{}, err
	}
	match := &actionContext.Match
	if match.Status != models.MatchLive {
		return ActionResult{}, ErrActionConflict
	}
	if actionContext.Receipt != nil {
		return s.duplicateReceipt(
			ctx, match, userID, envelope, payloadHash, actionContext.Receipt,
		)
	}
	record := &actionContext.State
	if envelope.ClientSequence != record.LastClientSequence+1 {
		return ActionResult{}, ErrActionSequence
	}
	if envelope.ExpectedStateVersion != record.StateVersion {
		return ActionResult{}, ErrActionConflict
	}
	stageStarted = time.Now()
	runtime, descriptor, err := s.resolve(match)
	if err != nil {
		return ActionResult{}, err
	}
	observability.ObserveTiming(ctx, "game_action.resolve", stageStarted)

	var current interfaces.GameState
	computeStarted := time.Now()
	stageStarted = time.Now()
	if err := json.Unmarshal(record.State, &current); err != nil {
		return ActionResult{}, err
	}
	observability.ObserveTiming(ctx, "game_action.decode_state", stageStarted)

	receivedAt := s.now().UTC()
	runtimeActionContext := interfaces.ActionContext{
		MatchID: match.ID, ParticipantID: userID, UserID: userID,
		ServerReceivedAt: receivedAt, Latency: latency,
		CurrentSequence:     envelope.ClientSequence,
		CurrentStateVersion: record.StateVersion,
	}
	stageStarted = time.Now()
	validated, err := runtime.ValidateAction(ctx, runtimeActionContext, current, envelope)
	if err != nil {
		return ActionResult{}, err
	}
	observability.ObserveTiming(ctx, "game_action.validate", stageStarted)

	stageStarted = time.Now()
	transition, err := runtime.ApplyAction(ctx, runtimeActionContext, current, validated)
	if err != nil {
		return ActionResult{}, err
	}
	observability.ObserveTiming(ctx, "game_action.apply", stageStarted)

	stageStarted = time.Now()
	transitionBytes, err := json.Marshal(transition)
	if err != nil {
		return ActionResult{}, err
	}
	nextStateBytes, err := json.Marshal(transition.NextState)
	if err != nil {
		return ActionResult{}, err
	}
	observability.ObserveTiming(ctx, "game_action.serialize_state", stageStarted)

	stageStarted = time.Now()
	status := "active"
	if transition.Completion != nil {
		switch transition.Completion.Status {
		case "complete":
			status = "completed"
		case "timeout":
			status = "timed_out"
		}
	}
	next := *record
	next.StateSchema = transition.NextState.SchemaVersion
	next.StateVersion = transition.NextState.Version
	next.State = nextStateBytes
	next.StateChecksum = transition.NextState.Checksum
	next.LastClientSequence = envelope.ClientSequence
	next.Status = status
	next.UpdatedAt = receivedAt
	drafts, err := actionEvents(envelope, transition, receivedAt)
	if err != nil {
		return ActionResult{}, err
	}
	receipt := models.GameActionReceipt{
		ActionID: envelope.ActionID, MatchID: match.ID, UserID: userID,
		ClientSequence:       envelope.ClientSequence,
		ExpectedStateVersion: envelope.ExpectedStateVersion,
		ActionKind:           envelope.Kind, ActionPayloadHash: payloadHash,
		Accepted: transition.Accepted, ResultCode: transition.Code,
		StateVersionBefore: record.StateVersion,
		StateVersionAfter:  transition.NextState.Version,
		Transition:         transitionBytes, ServerReceivedAt: receivedAt, ProcessedAt: s.now().UTC(),
	}
	receipt.ReceiptHash = shared.HashFields(
		"skill-arena/game-action-receipt/v1", receipt.ActionID, receipt.MatchID,
		receipt.UserID, receipt.ActionPayloadHash, receipt.ResultCode,
		fmt.Sprint(receipt.StateVersionBefore), fmt.Sprint(receipt.StateVersionAfter),
		string(receipt.Transition), receipt.ServerReceivedAt.Format(time.RFC3339Nano),
	)
	observability.ObserveTiming(ctx, "game_action.prepare_commit", stageStarted)
	observability.ObserveTiming(ctx, "game_action.compute", computeStarted)
	stageStarted = time.Now()
	var committed *models.GameActionReceipt
	var events []models.RealtimeEvent
	for attempt := 0; attempt < 3; attempt++ {
		committed, events, err = s.store.CommitGameAction(
			ctx, *record, next, receipt, actionContext.StreamSequence,
			actionContext.StreamHash, drafts,
		)
		if !errors.Is(err, db.ErrRealtimeConflict) {
			break
		}
		s.actionCache.Delete(gameActionCacheKey(matchID, userID))
		refreshed, refreshErr := s.store.GetGameActionContext(
			ctx, matchID, userID, envelope.ActionID, envelope.ClientSequence,
		)
		if refreshErr != nil {
			err = refreshErr
			break
		}
		if refreshed.Receipt != nil {
			observability.ObserveTiming(ctx, "game_action.commit", stageStarted)
			return s.duplicateReceipt(
				ctx, &refreshed.Match, userID, envelope, payloadHash, refreshed.Receipt,
			)
		}
		if refreshed.Match.Status != models.MatchLive ||
			refreshed.State.StateVersion != record.StateVersion ||
			refreshed.State.LastClientSequence != record.LastClientSequence {
			err = db.ErrRealtimeConflict
			break
		}
		actionContext = refreshed
		match = &actionContext.Match
	}
	observability.ObserveTiming(ctx, "game_action.commit", stageStarted)
	if errors.Is(err, db.ErrRealtimeConflict) {
		return ActionResult{}, ErrActionConflict
	}
	if err != nil {
		return ActionResult{}, err
	}
	postCommitStarted := time.Now()
	match.Sequence = events[len(events)-1].Sequence
	match.UpdatedAt = next.UpdatedAt
	stageStarted = time.Now()
	s.rememberStream(
		db.GameActionContext{
			Match: *match, StreamSequence: match.Sequence,
			StreamHash: events[len(events)-1].IntegrityHash,
		},
	)
	s.actionCache.Store(
		gameActionCacheKey(matchID, userID),
		db.GameActionContext{
			Match: *match, State: next,
			StreamSequence: match.Sequence,
			StreamHash:     events[len(events)-1].IntegrityHash,
		},
	)
	observability.ObserveTiming(ctx, "game_action.publish_cache", stageStarted)

	stageStarted = time.Now()
	snapshot, err := runtime.Snapshot(ctx, interfaces.ViewerContext{
		MatchID: match.ID, UserID: userID, ParticipantID: userID, Role: "player",
	}, transition.NextState)
	if err != nil {
		return ActionResult{}, err
	}
	observability.ObserveTiming(ctx, "game_action.snapshot", stageStarted)

	result := ActionResult{
		Receipt: *committed, Snapshot: snapshot, Events: events,
		Completion: transition.Completion, Match: match,
	}
	if transition.Completion != nil {
		stageStarted = time.Now()
		for _, participant := range match.Participants {
			s.actionCache.Delete(gameActionCacheKey(matchID, participant.UserID))
		}
		outcome, outcomeErr := s.determineOutcome(ctx, runtime, descriptor, match)
		if outcomeErr != nil {
			return ActionResult{}, outcomeErr
		}
		result.Outcome = &outcome
		observability.ObserveTiming(ctx, "game_action.completion", stageStarted)
	}
	observability.ObserveTiming(ctx, "game_action.post_commit", postCommitStarted)
	return result, nil
}

func (s *Service) gameActionContext(
	ctx context.Context,
	matchID, userID string,
	envelope interfaces.ActionEnvelope,
) (db.GameActionContext, error) {
	key := gameActionCacheKey(matchID, userID)
	if cached, ok := s.actionCache.Load(key); ok {
		actionContext, valid := cached.(db.GameActionContext)
		if valid &&
			actionContext.Match.Status == models.MatchLive &&
			envelope.ClientSequence == actionContext.State.LastClientSequence+1 &&
			envelope.ExpectedStateVersion == actionContext.State.StateVersion {
			s.applyCachedStream(&actionContext)
			return actionContext, nil
		}
	}
	actionContext, err := s.store.GetGameActionContext(
		ctx, matchID, userID, envelope.ActionID, envelope.ClientSequence,
	)
	if err != nil {
		return db.GameActionContext{}, err
	}
	if actionContext.Receipt == nil && actionContext.Match.Status == models.MatchLive {
		s.rememberStream(actionContext)
		s.actionCache.Store(key, actionContext)
	}
	return actionContext, nil
}

func gameActionCacheKey(matchID, userID string) string {
	return matchID + "\x1f" + userID
}

func (s *Service) applyCachedStream(actionContext *db.GameActionContext) {
	cached, ok := s.streamHeads.Load(actionContext.Match.ID)
	if !ok {
		return
	}
	head, valid := cached.(gameStreamHead)
	if !valid || head.Sequence <= actionContext.StreamSequence {
		return
	}
	actionContext.Match.Sequence = head.Sequence
	actionContext.StreamSequence = head.Sequence
	actionContext.StreamHash = head.IntegrityHash
}

func (s *Service) rememberStream(actionContext db.GameActionContext) {
	next := gameStreamHead{
		Sequence: actionContext.StreamSequence, IntegrityHash: actionContext.StreamHash,
	}
	for {
		cached, ok := s.streamHeads.Load(actionContext.Match.ID)
		if !ok {
			if _, loaded := s.streamHeads.LoadOrStore(actionContext.Match.ID, next); !loaded {
				return
			}
			continue
		}
		head, valid := cached.(gameStreamHead)
		if valid && head.Sequence >= next.Sequence {
			return
		}
		if s.streamHeads.CompareAndSwap(actionContext.Match.ID, cached, next) {
			return
		}
	}
}

func terminalMatchStatus(status string) bool {
	switch status {
	case models.MatchCompleted, models.MatchCancelled, models.MatchAbandoned:
		return true
	default:
		return false
	}
}

func (s *Service) Sync(
	ctx context.Context,
	match *models.RealtimeMatch,
	userID, role string,
) (SyncResult, error) {
	if match == nil {
		return SyncResult{}, ErrActionConflict
	}
	record, err := s.store.GetGameParticipantState(ctx, match.ID, userID)
	if err != nil {
		return SyncResult{}, err
	}
	runtime, _, err := s.resolve(match)
	if err != nil {
		return SyncResult{}, err
	}
	var state interfaces.GameState
	if err := json.Unmarshal(record.State, &state); err != nil {
		return SyncResult{}, err
	}
	if role == "" {
		role = "player"
	}
	snapshot, err := runtime.Snapshot(ctx, interfaces.ViewerContext{
		MatchID: match.ID, UserID: userID, ParticipantID: userID, Role: role,
	}, state)
	if err != nil {
		return SyncResult{}, err
	}
	return SyncResult{
		Snapshot: snapshot, StateVersion: record.StateVersion,
		LastClientSequence: record.LastClientSequence,
		LastServerSequence: record.LastServerSequence,
	}, nil
}

func (s *Service) ExpireDue(
	ctx context.Context,
	serverTime time.Time,
) ([]ExpiredMatch, error) {
	records, err := s.store.ListActiveGameParticipantStates(ctx)
	if err != nil {
		return nil, err
	}
	affected := map[string]bool{}
	for _, candidate := range records {
		match, err := s.store.GetRealtimeMatch(ctx, candidate.MatchID)
		if err != nil {
			return nil, err
		}
		if match.Status != models.MatchLive && match.Status != models.MatchPaused &&
			match.Status != models.MatchReconnecting {
			continue
		}
		runtime, descriptor, err := s.resolve(match)
		if err != nil {
			return nil, err
		}
		deadlineRuntime, ok := runtime.(interfaces.DeadlineRuntime)
		if !ok {
			continue
		}
		lockKey := "game:action:" + match.ID
		token, locked, err := s.store.Redis().Lock(ctx, lockKey, 5*time.Second)
		if err != nil {
			return nil, err
		}
		if !locked {
			continue
		}
		current, loadErr := s.store.GetGameParticipantState(
			ctx, candidate.MatchID, candidate.UserID,
		)
		if loadErr != nil {
			_ = s.store.Redis().Unlock(context.Background(), lockKey, token)
			return nil, loadErr
		}
		var state interfaces.GameState
		if err := json.Unmarshal(current.State, &state); err != nil {
			_ = s.store.Redis().Unlock(context.Background(), lockKey, token)
			return nil, err
		}
		participantIDs := make([]string, len(match.Participants))
		for index, participant := range match.Participants {
			participantIDs[index] = participant.UserID
		}
		transition, expireErr := deadlineRuntime.Expire(
			ctx, runtimeMatchContext(match, descriptor, participantIDs, serverTime),
			state, serverTime,
		)
		if expireErr != nil {
			_ = s.store.Redis().Unlock(context.Background(), lockKey, token)
			return nil, expireErr
		}
		if len(transition.NextState.Payload) != 0 {
			transitionBytes, marshalErr := json.Marshal(transition)
			if marshalErr != nil {
				_ = s.store.Redis().Unlock(context.Background(), lockKey, token)
				return nil, marshalErr
			}
			nextStateBytes, marshalErr := json.Marshal(transition.NextState)
			if marshalErr != nil {
				_ = s.store.Redis().Unlock(context.Background(), lockKey, token)
				return nil, marshalErr
			}
			next := *current
			next.StateSchema = transition.NextState.SchemaVersion
			next.StateVersion = transition.NextState.Version
			next.State = nextStateBytes
			next.StateChecksum = transition.NextState.Checksum
			next.Status = "timed_out"
			next.UpdatedAt = serverTime.UTC()
			payload, marshalErr := json.Marshal(map[string]any{
				"participantId": current.UserID, "code": transition.Code,
				"transition": json.RawMessage(transitionBytes),
			})
			if marshalErr != nil {
				_ = s.store.Redis().Unlock(context.Background(), lockKey, token)
				return nil, marshalErr
			}
			drafts := []models.GameEventDraft{{
				Type: "game.participant.timed_out", Payload: payload,
			}}
			for _, event := range transition.Events {
				drafts = append(drafts, models.GameEventDraft{
					Type: event.Kind, Payload: event.Payload,
				})
			}
			if _, commitErr := s.store.CommitGameSystemTransition(
				ctx, *current, next, drafts,
			); commitErr != nil && !errors.Is(commitErr, db.ErrRealtimeConflict) {
				_ = s.store.Redis().Unlock(context.Background(), lockKey, token)
				return nil, commitErr
			}
			affected[match.ID] = true
		}
		_ = s.store.Redis().Unlock(context.Background(), lockKey, token)
	}
	results := make([]ExpiredMatch, 0, len(affected))
	for matchID := range affected {
		match, err := s.store.GetRealtimeMatch(ctx, matchID)
		if err != nil {
			return nil, err
		}
		runtime, descriptor, err := s.resolve(match)
		if err != nil {
			return nil, err
		}
		outcome, err := s.determineOutcome(ctx, runtime, descriptor, match)
		if err != nil {
			return nil, err
		}
		if outcome.Status != "incomplete" {
			results = append(results, ExpiredMatch{MatchID: matchID, Outcome: outcome})
		}
	}
	return results, nil
}

func (s *Service) Forfeit(
	ctx context.Context,
	matchID, userID string,
	serverTime time.Time,
) error {
	record, err := s.store.GetGameParticipantState(ctx, matchID, userID)
	if errors.Is(err, sql.ErrNoRows) {
		return nil
	}
	if err != nil || record.Status != "active" {
		return err
	}
	next := *record
	next.Status = "forfeited"
	next.UpdatedAt = serverTime.UTC()
	payload, err := json.Marshal(map[string]any{
		"participantId": userID, "reason": "participant_left",
		"stateVersion": record.StateVersion, "stateChecksum": record.StateChecksum,
	})
	if err != nil {
		return err
	}
	_, err = s.store.CommitGameSystemTransition(
		ctx, *record, next,
		[]models.GameEventDraft{{Type: "game.participant.forfeited", Payload: payload}},
	)
	return err
}

func (s *Service) FinalizeReplay(
	ctx context.Context,
	match *models.RealtimeMatch,
	integrity interfaces.ReplayIntegrityService,
) (interfaces.FinalizedReplay, error) {
	if match == nil || match.StartedAt == nil || match.CompletedAt == nil {
		return interfaces.FinalizedReplay{}, errors.New("terminal match timing is incomplete")
	}
	runtime, descriptor, err := s.resolve(match)
	if err != nil {
		return interfaces.FinalizedReplay{}, err
	}
	finalizer, ok := runtime.(interfaces.AuthoritativeReplayRuntime)
	if !ok {
		return interfaces.FinalizedReplay{}, ErrRuntimeUnavailable
	}
	records, err := s.store.ListGameParticipantStates(ctx, match.ID)
	if err != nil {
		return interfaces.FinalizedReplay{}, err
	}
	states := make([]interfaces.GameState, 0, len(records))
	participantIDs := make([]string, 0, len(records))
	for _, record := range records {
		var state interfaces.GameState
		if err := json.Unmarshal(record.State, &state); err != nil {
			return interfaces.FinalizedReplay{}, err
		}
		states = append(states, state)
		participantIDs = append(participantIDs, record.UserID)
	}
	outcome, err := runtime.DetermineWinner(
		ctx, runtimeMatchContext(match, descriptor, participantIDs, match.CompletedAt.UTC()),
		states,
	)
	if err != nil {
		return interfaces.FinalizedReplay{}, err
	}
	if outcome.Status == "incomplete" {
		outcome = terminalOutcome(match, records)
	}
	var realtimeEvents []models.RealtimeEvent
	var after int64
	for {
		page, err := s.store.RealtimeEventsAfter(ctx, match.ID, after, 500)
		if err != nil {
			return interfaces.FinalizedReplay{}, err
		}
		realtimeEvents = append(realtimeEvents, page...)
		if len(page) < 500 {
			break
		}
		after = page[len(page)-1].Sequence
	}
	events := make([]interfaces.ReplayEvent, len(realtimeEvents))
	for index, event := range realtimeEvents {
		events[index] = interfaces.ReplayEvent{
			Sequence: event.Sequence, StateVersion: event.StateVersion,
			OccurredAt: event.ServerTime, ParticipantID: event.UserID,
			Kind: event.Type, Payload: append(json.RawMessage(nil), event.Payload...),
		}
	}
	return finalizer.FinalizeAuthoritativeReplay(
		ctx,
		runtimeMatchContext(match, descriptor, participantIDs, match.CompletedAt.UTC()),
		interfaces.ReplaySource{
			ReplayID: id.Replay(), MatchID: match.ID, States: states, Events: events,
			Outcome: outcome, StartedAtUnixMS: match.StartedAt.UnixMilli(),
			EndedAtUnixMS: match.CompletedAt.UnixMilli(),
		},
		integrity,
	)
}

func terminalOutcome(
	match *models.RealtimeMatch,
	records []models.GameParticipantState,
) interfaces.MatchOutcome {
	if match.Status == models.MatchCancelled || match.Status == models.MatchAbandoned {
		return interfaces.MatchOutcome{Status: "canceled", Reason: "match_terminated"}
	}
	forfeited := map[string]bool{}
	for _, record := range records {
		if record.Status == "forfeited" {
			forfeited[record.UserID] = true
		}
	}
	if len(forfeited) > 0 {
		winners := make([]string, 0, len(records)-len(forfeited))
		losers := make([]string, 0, len(forfeited))
		for _, record := range records {
			if forfeited[record.UserID] {
				losers = append(losers, record.UserID)
			} else {
				winners = append(winners, record.UserID)
			}
		}
		if len(winners) > 0 {
			return interfaces.MatchOutcome{
				Status: "complete", WinnerIDs: winners,
				LoserIDs: losers, Reason: "participant_forfeit",
			}
		}
	}
	return interfaces.MatchOutcome{Status: "canceled", Reason: "no_verified_winner"}
}

func (s *Service) duplicate(
	ctx context.Context,
	match *models.RealtimeMatch,
	userID string,
	envelope interfaces.ActionEnvelope,
	payloadHash string,
) (ActionResult, error) {
	receipt, err := s.store.GetGameActionReceipt(
		ctx, match.ID, userID, envelope.ActionID, envelope.ClientSequence,
	)
	if err != nil {
		return ActionResult{}, err
	}
	return s.duplicateReceipt(ctx, match, userID, envelope, payloadHash, receipt)
}

func (s *Service) duplicateReceipt(
	ctx context.Context,
	match *models.RealtimeMatch,
	userID string,
	envelope interfaces.ActionEnvelope,
	payloadHash string,
	receipt *models.GameActionReceipt,
) (ActionResult, error) {
	if receipt.MatchID != match.ID || receipt.UserID != userID ||
		receipt.ActionID != envelope.ActionID ||
		receipt.ClientSequence != envelope.ClientSequence ||
		receipt.ActionPayloadHash != payloadHash {
		return ActionResult{}, ErrDuplicateMismatch
	}
	var transition interfaces.Transition
	if err := json.Unmarshal(receipt.Transition, &transition); err != nil {
		return ActionResult{}, err
	}
	runtime, _, err := s.resolve(match)
	if err != nil {
		return ActionResult{}, err
	}
	snapshot, err := runtime.Snapshot(ctx, interfaces.ViewerContext{
		MatchID: match.ID, UserID: userID, ParticipantID: userID, Role: "player",
	}, transition.NextState)
	if err != nil {
		return ActionResult{}, err
	}
	return ActionResult{
		Receipt: *receipt, Snapshot: snapshot,
		Completion: transition.Completion, Duplicate: true,
	}, nil
}

func (s *Service) determineOutcome(
	ctx context.Context,
	runtime interfaces.RuntimeGame,
	descriptor interfaces.Descriptor,
	match *models.RealtimeMatch,
) (interfaces.MatchOutcome, error) {
	records, err := s.store.ListGameParticipantStates(ctx, match.ID)
	if err != nil {
		return interfaces.MatchOutcome{}, err
	}
	states := make([]interfaces.GameState, 0, len(records))
	participants := make([]string, 0, len(records))
	for _, record := range records {
		var state interfaces.GameState
		if err := json.Unmarshal(record.State, &state); err != nil {
			return interfaces.MatchOutcome{}, err
		}
		states = append(states, state)
		participants = append(participants, record.UserID)
	}
	return runtime.DetermineWinner(
		ctx, runtimeMatchContext(match, descriptor, participants, s.now().UTC()), states,
	)
}

func (s *Service) resolve(
	match *models.RealtimeMatch,
) (interfaces.RuntimeGame, interfaces.Descriptor, error) {
	module, err := s.store.GamesRegistry().Resolve(match.GameID, match.GameVersion)
	if err != nil {
		return nil, interfaces.Descriptor{}, fmt.Errorf("%w: %v", ErrRuntimeUnavailable, err)
	}
	runtime, ok := module.(interfaces.RuntimeGame)
	if !ok {
		return nil, interfaces.Descriptor{}, ErrRuntimeUnavailable
	}
	return runtime, module.Descriptor(), nil
}

func runtimeMatchContext(
	match *models.RealtimeMatch,
	descriptor interfaces.Descriptor,
	participants []string,
	serverTime time.Time,
) interfaces.MatchContext {
	return interfaces.MatchContext{
		MatchID: match.ID, GameID: match.GameID, Mode: match.Mode,
		Region: match.Region, ParticipantIDs: append([]string(nil), participants...),
		Versions: descriptor.Versions, ServerTime: serverTime,
	}
}

func actionEvents(
	envelope interfaces.ActionEnvelope,
	transition interfaces.Transition,
	receivedAt time.Time,
) ([]models.GameEventDraft, error) {
	payload, err := json.Marshal(map[string]any{
		"actionId": envelope.ActionID, "kind": envelope.Kind,
		"payload":              json.RawMessage(envelope.Payload),
		"clientSequence":       envelope.ClientSequence,
		"expectedStateVersion": envelope.ExpectedStateVersion,
		"accepted":             transition.Accepted, "code": transition.Code,
		"stateVersion":  transition.NextState.Version,
		"stateChecksum": transition.NextState.Checksum,
		"occurredAtMs":  receivedAt.UnixMilli(),
		"progress":      json.RawMessage(transition.Progress),
		"presentation":  json.RawMessage(transition.Presentation),
	})
	if err != nil {
		return nil, err
	}
	drafts := []models.GameEventDraft{{
		Type: "game.action.processed", Payload: payload,
	}}
	for _, event := range transition.Events {
		drafts = append(drafts, models.GameEventDraft{
			Type: event.Kind, Payload: append(json.RawMessage(nil), event.Payload...),
		})
	}
	return drafts, nil
}
