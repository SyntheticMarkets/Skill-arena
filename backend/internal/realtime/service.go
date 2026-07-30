package realtime

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"skill-arena/internal/db"
	"skill-arena/internal/games/interfaces"
	gamesession "skill-arena/internal/games/session"
	"skill-arena/internal/id"
	"skill-arena/internal/models"
	"skill-arena/internal/observability"
)

var (
	ErrNotParticipant = errors.New("user is not a match participant")
	ErrAlreadyQueued  = errors.New("user already has an active queue entry")
	ErrUnsupported    = errors.New("game does not support the requested mode")
)

type QueueRequest struct {
	GameID         string `json:"gameId"`
	Mode           string `json:"mode"`
	WalletCategory string `json:"walletCategory"`
	Region         string `json:"region"`
	Jurisdiction   string `json:"jurisdiction"`
	LatencyMS      int    `json:"latencyMs"`
}

type Service struct {
	store           *db.Store
	queueTTL        time.Duration
	presenceTTL     time.Duration
	reconnectWindow time.Duration
	maxRatingGap    int
	maxLatencyMS    int
	games           *gamesession.Service
}

func NewService(store *db.Store) *Service {
	games, _ := gamesession.New(store)
	service := &Service{
		store: store, queueTTL: 2 * time.Minute, presenceTTL: 45 * time.Second,
		reconnectWindow: 30 * time.Second, maxRatingGap: 250, maxLatencyMS: 500,
		games: games,
	}
	if settings := store.RuntimeSettings(); settings != nil {
		realtime := settings.Realtime
		if realtime.QueueTTL > 0 {
			service.queueTTL = realtime.QueueTTL
		}
		if realtime.PresenceTTL > 0 {
			service.presenceTTL = realtime.PresenceTTL
		}
		if realtime.ReconnectWindow > 0 {
			service.reconnectWindow = realtime.ReconnectWindow
		}
		if realtime.MaxRatingGap >= 0 {
			service.maxRatingGap = realtime.MaxRatingGap
		}
		if realtime.MaxLatencyMS > 0 {
			service.maxLatencyMS = realtime.MaxLatencyMS
		}
	}
	return service
}

func (s *Service) Queue(ctx context.Context, userID string, request QueueRequest) (*models.RealtimeQueueEntry, *models.RealtimeMatch, error) {
	request.GameID = strings.TrimSpace(request.GameID)
	request.Mode = strings.ToLower(strings.TrimSpace(request.Mode))
	request.WalletCategory = strings.ToLower(strings.TrimSpace(request.WalletCategory))
	request.Region = strings.ToLower(strings.TrimSpace(request.Region))
	request.Jurisdiction = strings.ToUpper(strings.TrimSpace(request.Jurisdiction))
	if request.WalletCategory == "" {
		request.WalletCategory = "practice"
	}
	if request.WalletCategory != "practice" && request.WalletCategory != "live" {
		return nil, nil, errors.New("wallet category is invalid")
	}
	if request.Region == "" {
		request.Region = "global"
	}
	if request.LatencyMS < 0 || request.LatencyMS > s.maxLatencyMS {
		return nil, nil, errors.New("latency is outside the accepted range")
	}
	user, err := s.store.GetUserByID(ctx, userID)
	if err != nil {
		return nil, nil, err
	}
	country := strings.ToUpper(strings.TrimSpace(user.Country))
	if country == "" {
		country = request.Jurisdiction
	}
	if country == "" || (request.Jurisdiction != "" && request.Jurisdiction != country) {
		return nil, nil, errors.New("jurisdiction does not match the verified player profile")
	}
	request.Jurisdiction = country
	if request.WalletCategory == "live" {
		assessment, assessmentErr := s.store.GetFinancialAssessment(ctx, userID)
		if assessmentErr != nil {
			return nil, nil, assessmentErr
		}
		if assessment.Status != models.AssessmentStatusComplete || assessment.ResponsibleStatus != "active" {
			return nil, nil, errors.New("player is not eligible for live competition")
		}
	}
	if !s.supportsMode(request.GameID, request.Mode) {
		return nil, nil, ErrUnsupported
	}
	if existing, err := s.store.RealtimeQueueForUser(ctx, userID); err == nil && existing.Status == models.QueueWaiting && existing.ExpiresAt.After(time.Now().UTC()) {
		return existing, nil, ErrAlreadyQueued
	}
	progression, err := s.store.GetProgressionByUserID(ctx, userID)
	if err != nil {
		return nil, nil, err
	}
	now := time.Now().UTC()
	entry := models.RealtimeQueueEntry{
		ID: id.New("que"), UserID: userID, GameID: request.GameID, Mode: request.Mode,
		WalletCategory: request.WalletCategory, Region: request.Region, Jurisdiction: request.Jurisdiction,
		Rating: progression.EloRating, LatencyMS: request.LatencyMS, Priority: s.serverPriority(progression),
		Status: models.QueueWaiting, CreatedAt: now, ExpiresAt: now.Add(s.queueTTL), UpdatedAt: now,
	}
	if request.Mode == "practice" {
		match, err := s.createMatch(ctx, entry, nil)
		if err != nil {
			return nil, nil, err
		}
		entry.Status, entry.MatchID, entry.UpdatedAt = models.QueueMatched, match.ID, time.Now().UTC()
		if err := s.store.UpsertRealtimeQueue(ctx, entry); err != nil {
			return nil, nil, err
		}
		_ = s.setPresence(ctx, userID, models.PresenceInMatch, "", "", match.ID, request.Region)
		return &entry, match, nil
	}
	if err := s.store.UpsertRealtimeQueue(ctx, entry); err != nil {
		return nil, nil, err
	}
	_ = s.setPresence(ctx, userID, models.PresenceInQueue, "", "", "", request.Region)

	lockKey := "realtime:matchmaking:" + entry.GameID + ":" + entry.Mode + ":" + entry.WalletCategory + ":" + entry.Region
	lockToken, err := s.acquireMatchmakingLock(ctx, lockKey)
	if err != nil {
		return &entry, nil, err
	}
	defer s.store.Redis().Unlock(context.Background(), lockKey, lockToken)
	current, err := s.store.RealtimeQueueForUser(ctx, userID)
	if err != nil {
		return &entry, nil, err
	}
	if current.ID != entry.ID || current.Status != models.QueueWaiting {
		if current.MatchID == "" {
			return current, nil, nil
		}
		match, matchErr := s.store.GetRealtimeMatch(ctx, current.MatchID)
		return current, match, matchErr
	}
	candidates, err := s.store.WaitingRealtimeQueue(ctx, entry.GameID, entry.Mode, entry.WalletCategory, entry.Region, now)
	if err != nil {
		return &entry, nil, err
	}
	var opponent *models.RealtimeQueueEntry
	for i := range candidates {
		candidate := candidates[i]
		if candidate.UserID != userID && abs(candidate.Rating-entry.Rating) <= s.maxRatingGap && compatibleJurisdiction(entry, candidate) {
			opponent = &candidate
			break
		}
	}
	if opponent == nil {
		return &entry, nil, nil
	}
	match, err := s.createMatch(ctx, *opponent, &entry)
	if err != nil {
		return &entry, nil, err
	}
	entry.Status, entry.MatchID, entry.UpdatedAt = models.QueueMatched, match.ID, time.Now().UTC()
	opponent.Status, opponent.MatchID, opponent.UpdatedAt = models.QueueMatched, match.ID, entry.UpdatedAt
	if err := s.store.UpsertRealtimeQueue(ctx, *opponent); err != nil {
		return &entry, nil, err
	}
	if err := s.store.UpsertRealtimeQueue(ctx, entry); err != nil {
		return &entry, nil, err
	}
	_ = s.setPresence(ctx, userID, models.PresenceInMatch, "", "", match.ID, request.Region)
	_ = s.setPresence(ctx, opponent.UserID, models.PresenceInMatch, "", "", match.ID, opponent.Region)
	return &entry, match, nil
}

func (s *Service) acquireMatchmakingLock(ctx context.Context, key string) (string, error) {
	for {
		token, locked, err := s.store.Redis().Lock(ctx, key, 5*time.Second)
		if err != nil {
			return "", err
		}
		if locked {
			return token, nil
		}
		timer := time.NewTimer(10 * time.Millisecond)
		select {
		case <-ctx.Done():
			timer.Stop()
			return "", ctx.Err()
		case <-timer.C:
		}
	}
}

func (s *Service) createMatch(ctx context.Context, first models.RealtimeQueueEntry, second *models.RealtimeQueueEntry) (*models.RealtimeMatch, error) {
	module, err := s.store.ArenaRegistry().Get(first.GameID)
	if err != nil {
		return nil, err
	}
	meta := module.Manifest()
	now := time.Now().UTC()
	match := models.RealtimeMatch{
		ID: id.Match(), GameID: meta.ID, GameVersion: meta.Versions.Game, RulesVersion: meta.Versions.Rules,
		ProtocolVersion: meta.Versions.Protocol, ReplayVersion: meta.Versions.Replay, Mode: first.Mode,
		Status: models.MatchCreated, Region: first.Region, WalletCategory: first.WalletCategory,
		SeedReference: id.New("seed"), StateVersion: 1, CreatedAt: now, UpdatedAt: now,
	}
	p1 := queueParticipant(match.ID, first, now)
	created, err := s.store.CreateRealtimeMatch(ctx, match, p1)
	if err != nil {
		return nil, err
	}
	if second != nil {
		p2 := queueParticipant(match.ID, *second, now)
		if err := s.store.SaveRealtimeParticipant(ctx, p2); err != nil {
			return nil, err
		}
	}
	target := models.MatchReady
	if second != nil {
		target = models.MatchWaiting
	}
	created, err = s.transition(ctx, created, target, "", "match_created", map[string]any{"mode": first.Mode, "gameId": first.GameID})
	if err != nil {
		return nil, err
	}
	return s.store.GetRealtimeMatch(ctx, created.ID)
}

func (s *Service) Ready(ctx context.Context, userID, matchID string) (*models.RealtimeMatch, error) {
	match, participant, err := s.participant(ctx, userID, matchID)
	if err != nil {
		return nil, err
	}
	if terminal(match.Status) {
		return nil, ErrInvalidTransition
	}
	participant.Ready = true
	participant.Status = "ready"
	participant.LastSeenAt = time.Now().UTC()
	if err := s.store.SaveRealtimeParticipant(ctx, *participant); err != nil {
		return nil, err
	}
	match, err = s.store.GetRealtimeMatch(ctx, matchID)
	if err != nil {
		return nil, err
	}
	allReady := len(match.Participants) > 0
	for _, p := range match.Participants {
		allReady = allReady && p.Ready && p.LeftAt == nil
	}
	if !allReady {
		return match, nil
	}
	if match.Status == models.MatchWaiting {
		match, err = s.transition(ctx, match, models.MatchReady, userID, "players_ready", nil)
		if err != nil {
			return nil, err
		}
	}
	if s.games != nil {
		if err := s.games.PrepareMatch(ctx, match); err != nil &&
			!errors.Is(err, gamesession.ErrRuntimeUnavailable) {
			return nil, err
		}
	}
	match, err = s.transition(ctx, match, models.MatchStarting, "", "match_starting", nil)
	if err != nil {
		return nil, err
	}
	now := time.Now().UTC()
	match.StartedAt = &now
	match, err = s.transition(ctx, match, models.MatchLive, "", "match_started", map[string]any{"serverTime": now})
	return match, err
}

func (s *Service) GameAction(
	ctx context.Context,
	userID, matchID string,
	envelope interfaces.ActionEnvelope,
	latency time.Duration,
) (gamesession.ActionResult, error) {
	if s.games == nil {
		return gamesession.ActionResult{}, gamesession.ErrRuntimeUnavailable
	}
	result, err := s.games.SubmitAction(ctx, matchID, userID, envelope, latency)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return gamesession.ActionResult{}, ErrNotParticipant
		}
		return gamesession.ActionResult{}, err
	}
	if result.Outcome != nil && result.Outcome.Status == "complete" &&
		result.Match != nil && !terminal(result.Match.Status) {
		match, loadErr := s.store.GetRealtimeMatch(ctx, matchID)
		if loadErr != nil {
			return gamesession.ActionResult{}, loadErr
		}
		completed, transitionErr := s.transition(
			ctx, match, models.MatchCompleted, userID, "game.match.completed", result.Outcome,
		)
		if transitionErr != nil && !errors.Is(transitionErr, ErrInvalidTransition) {
			return gamesession.ActionResult{}, transitionErr
		}
		if transitionErr == nil {
			_, err = s.store.EnqueueJob(
				ctx, models.JobRealtimeReplayPersist,
				map[string]string{"matchId": completed.ID}, time.Now().UTC(),
			)
			if err != nil {
				return gamesession.ActionResult{}, err
			}
		}
	}
	return result, nil
}

func (s *Service) GameSync(
	ctx context.Context,
	userID, matchID string,
) (gamesession.SyncResult, error) {
	match, _, err := s.participant(ctx, userID, matchID)
	if err != nil {
		return gamesession.SyncResult{}, err
	}
	if s.games == nil {
		return gamesession.SyncResult{}, gamesession.ErrRuntimeUnavailable
	}
	return s.games.Sync(ctx, match, userID, "player")
}

func (s *Service) ExpireDueGameMatches(ctx context.Context, now time.Time) (int, error) {
	if s.games == nil {
		return 0, nil
	}
	expired, err := s.games.ExpireDue(ctx, now.UTC())
	if err != nil {
		return 0, err
	}
	completed := 0
	for _, item := range expired {
		match, err := s.store.GetRealtimeMatch(ctx, item.MatchID)
		if err != nil {
			return completed, err
		}
		if terminal(match.Status) {
			continue
		}
		match, err = s.transition(
			ctx, match, models.MatchCompleted, "", "game.match.timed_out", item.Outcome,
		)
		if err != nil {
			return completed, err
		}
		if _, err := s.store.EnqueueJob(
			ctx, models.JobRealtimeReplayPersist,
			map[string]string{"matchId": match.ID}, now.UTC(),
		); err != nil {
			return completed, err
		}
		completed++
	}
	return completed, nil
}

func (s *Service) Leave(ctx context.Context, userID, matchID string) (*models.RealtimeMatch, error) {
	match, participant, err := s.participant(ctx, userID, matchID)
	if err != nil {
		return nil, err
	}
	now := time.Now().UTC()
	participant.Status, participant.LeftAt, participant.LastSeenAt = "left", &now, now
	if err := s.store.SaveRealtimeParticipant(ctx, *participant); err != nil {
		return nil, err
	}
	if s.games != nil {
		if err := s.games.Forfeit(ctx, match.ID, userID, now); err != nil {
			return nil, err
		}
	}
	_ = s.setPresence(ctx, userID, models.PresenceOnline, "", "", "", participant.Region)
	if terminal(match.Status) {
		return s.store.GetRealtimeMatch(ctx, matchID)
	}
	target := models.MatchCancelled
	if match.Status == models.MatchLive || match.Status == models.MatchPaused || match.Status == models.MatchReconnecting {
		if len(match.Participants) > 1 {
			target = models.MatchCompleted
		} else {
			target = models.MatchAbandoned
		}
	}
	match.CompletedAt = &now
	match, err = s.transition(ctx, match, target, userID, "participant_left", map[string]any{"termination": "forfeit"})
	if err == nil {
		_, err = s.store.EnqueueJob(ctx, models.JobRealtimeReplayPersist, map[string]string{"matchId": match.ID}, time.Now().UTC())
	}
	return match, err
}

func (s *Service) Disconnect(ctx context.Context, userID, sessionID, connectionID, matchID string) error {
	if err := s.setPresence(ctx, userID, models.PresenceDisconnected, sessionID, connectionID, matchID, ""); err != nil {
		return err
	}
	if matchID == "" {
		return nil
	}
	match, _, err := s.participant(ctx, userID, matchID)
	if err != nil || terminal(match.Status) {
		return err
	}
	if match.Status == models.MatchLive || match.Status == models.MatchPaused {
		_, err = s.transition(ctx, match, models.MatchReconnecting, userID, "participant_disconnected", map[string]any{"reconnectWindowSeconds": int(s.reconnectWindow.Seconds())})
	}
	return err
}

func (s *Service) Reconnect(ctx context.Context, userID, matchID string, afterSequence int64) (*models.RealtimeMatch, []models.RealtimeEvent, error) {
	stageStarted := time.Now()
	match, participant, err := s.participant(ctx, userID, matchID)
	observability.ObserveTiming(ctx, "reconnect.participant", stageStarted)
	if err != nil {
		return nil, nil, err
	}
	if terminal(match.Status) {
		events, listErr := s.store.RealtimeEventsAfter(ctx, matchID, afterSequence, 500)
		return match, events, listErr
	}
	participant.Status, participant.LastSeenAt = "connected", time.Now().UTC()
	participant.LastSequence = afterSequence
	stageStarted = time.Now()
	if err := s.store.SaveRealtimeParticipant(ctx, *participant); err != nil {
		return nil, nil, err
	}
	observability.ObserveTiming(ctx, "reconnect.save_participant", stageStarted)
	if match.Status == models.MatchReconnecting || match.Status == models.MatchPaused {
		stageStarted = time.Now()
		match, err = s.transition(ctx, match, models.MatchLive, userID, "participant_reconnected", nil)
		observability.ObserveTiming(ctx, "reconnect.transition", stageStarted)
		if err != nil {
			return nil, nil, err
		}
	}
	stageStarted = time.Now()
	_ = s.setPresence(ctx, userID, models.PresenceInMatch, "", "", matchID, participant.Region)
	observability.ObserveTiming(ctx, "reconnect.presence", stageStarted)
	stageStarted = time.Now()
	events, err := s.store.RealtimeEventsAfter(ctx, matchID, afterSequence, 500)
	observability.ObserveTiming(ctx, "reconnect.events", stageStarted)
	return match, events, err
}

func (s *Service) Heartbeat(ctx context.Context, userID, sessionID, connectionID, matchID, region string, latencyMS int) (time.Time, error) {
	if latencyMS < 0 || latencyMS > s.maxLatencyMS {
		return time.Time{}, errors.New("invalid latency")
	}
	state := models.PresenceOnline
	if matchID != "" {
		if _, _, err := s.participant(ctx, userID, matchID); err != nil {
			return time.Time{}, err
		}
		state = models.PresenceInMatch
	}
	now := time.Now().UTC()
	err := s.setPresence(ctx, userID, state, sessionID, connectionID, matchID, region)
	return now, err
}

func (s *Service) CancelQueue(ctx context.Context, userID string) (*models.RealtimeQueueEntry, error) {
	entry, err := s.store.RealtimeQueueForUser(ctx, userID)
	if err != nil {
		return nil, err
	}
	if entry.Status != models.QueueWaiting {
		return nil, db.ErrRealtimeConflict
	}
	entry.Status, entry.UpdatedAt = models.QueueCancelled, time.Now().UTC()
	if err := s.store.UpsertRealtimeQueue(ctx, *entry); err != nil {
		return nil, err
	}
	_ = s.setPresence(ctx, userID, models.PresenceOnline, "", "", "", entry.Region)
	return entry, nil
}

func (s *Service) QueueStatus(ctx context.Context, userID string) (*models.RealtimeQueueEntry, error) {
	return s.store.RealtimeQueueForUser(ctx, userID)
}

func (s *Service) Match(ctx context.Context, userID, matchID string) (*models.RealtimeMatch, error) {
	match, _, err := s.participant(ctx, userID, matchID)
	return match, err
}

func (s *Service) Events(ctx context.Context, userID, matchID string, after int64) ([]models.RealtimeEvent, error) {
	if _, _, err := s.participant(ctx, userID, matchID); err != nil {
		return nil, err
	}
	return s.store.RealtimeEventsAfter(ctx, matchID, after, 500)
}

func (s *Service) Replay(ctx context.Context, userID, matchID string) (*models.RealtimeReplay, error) {
	if _, _, err := s.participant(ctx, userID, matchID); err != nil {
		return nil, err
	}
	return s.store.GetRealtimeReplay(ctx, matchID)
}

func (s *Service) FinalizeReplay(ctx context.Context, matchID string) error {
	match, err := s.store.GetRealtimeMatch(ctx, matchID)
	if err != nil {
		return err
	}
	if !terminal(match.Status) {
		return ErrInvalidTransition
	}
	if s.games != nil {
		finalized, finalizeErr := s.games.FinalizeReplay(ctx, match, s)
		if finalizeErr == nil {
			replay := models.RealtimeReplay{
				ID: finalized.ReplayID, MatchID: match.ID, GameID: match.GameID,
				GameVersion: match.GameVersion, RulesVersion: match.RulesVersion,
				ProtocolVersion: match.ProtocolVersion, ReplayVersion: match.ReplayVersion,
				EventCount: finalized.EventCount, EventRootHash: finalized.EventRootHash,
				Signature: finalized.Proof.Signature, StorageKey: finalized.StorageKey,
				Status: finalized.Status, CreatedAt: time.Now().UTC(),
			}
			if err := s.store.SaveRealtimeReplay(ctx, replay); err != nil {
				return err
			}
			payload, marshalErr := json.Marshal(map[string]any{
				"replayId": replay.ID, "replayHash": finalized.ReplayHash,
				"eventRootHash": replay.EventRootHash, "status": replay.Status,
				"keyId": finalized.Proof.KeyID,
			})
			if marshalErr != nil {
				return marshalErr
			}
			_, err = s.store.AppendRealtimeEvent(
				ctx, match.ID, "", "game.replay.ready", payload,
			)
			return err
		}
		if !errors.Is(finalizeErr, gamesession.ErrRuntimeUnavailable) {
			return finalizeErr
		}
	}
	var events []models.RealtimeEvent
	var after int64
	for {
		page, listErr := s.store.RealtimeEventsAfter(ctx, matchID, after, 500)
		if listErr != nil {
			return listErr
		}
		events = append(events, page...)
		if len(page) < 500 {
			break
		}
		after = page[len(page)-1].Sequence
	}
	root := ""
	if len(events) > 0 {
		root = events[len(events)-1].IntegrityHash
	}
	key := []byte("development-replay-signing-key")
	if settings := s.store.RuntimeSettings(); settings != nil && settings.Security.PuzzleSecret != "" {
		key = []byte(settings.Security.PuzzleSecret)
	}
	mac := hmac.New(sha256.New, key)
	_, _ = fmt.Fprintf(mac, "%s|%s|%s|%d", match.ID, match.GameID, root, len(events))
	replay := models.RealtimeReplay{
		ID: id.Replay(), MatchID: match.ID, GameID: match.GameID, GameVersion: match.GameVersion,
		RulesVersion: match.RulesVersion, ProtocolVersion: match.ProtocolVersion, ReplayVersion: match.ReplayVersion,
		EventCount: len(events), EventRootHash: root, Signature: hex.EncodeToString(mac.Sum(nil)), Status: "ready", CreatedAt: time.Now().UTC(),
	}
	if len(events) > 0 {
		replay.FirstSequence, replay.LastSequence = events[0].Sequence, events[len(events)-1].Sequence
	}
	data, err := json.Marshal(map[string]any{"replay": replay, "events": events})
	if err != nil {
		return err
	}
	replay.StorageKey = "replays/realtime/" + replay.ID + ".json"
	if err := s.store.ObjectStore().Put(ctx, replay.StorageKey, data, "application/json"); err != nil {
		return err
	}
	return s.store.SaveRealtimeReplay(ctx, replay)
}

func (s *Service) transition(ctx context.Context, match *models.RealtimeMatch, target, userID, eventType string, payload any) (*models.RealtimeMatch, error) {
	if err := ValidateTransition(match.Status, target); err != nil {
		return nil, err
	}
	match.Status = target
	if terminal(target) && match.CompletedAt == nil {
		now := time.Now().UTC()
		match.CompletedAt = &now
	}
	raw, _ := json.Marshal(payload)
	saved, event, err := s.store.TransitionRealtimeMatch(
		ctx, *match, match.StateVersion, userID, eventType, raw,
	)
	if err != nil {
		return nil, err
	}
	if s.games != nil {
		s.games.ObserveMatchTransition(saved, *event)
	}
	return saved, nil
}

func (s *Service) participant(ctx context.Context, userID, matchID string) (*models.RealtimeMatch, *models.RealtimeParticipant, error) {
	match, err := s.store.GetRealtimeMatch(ctx, matchID)
	if err != nil {
		return nil, nil, err
	}
	for i := range match.Participants {
		if match.Participants[i].UserID == userID {
			return match, &match.Participants[i], nil
		}
	}
	return nil, nil, ErrNotParticipant
}

func (s *Service) setPresence(ctx context.Context, userID, state, sessionID, connectionID, matchID, region string) error {
	now := time.Now().UTC()
	return s.store.SavePresence(ctx, models.PresenceRecord{UserID: userID, State: state, SessionID: sessionID, ConnectionID: connectionID, MatchID: matchID, Region: region, LastHeartbeat: now, ExpiresAt: now.Add(s.presenceTTL)})
}

func (s *Service) serverPriority(p *models.Progression) int {
	if p == nil {
		return 0
	}
	return min(p.MatchesPlayed/100, 10)
}

func queueParticipant(matchID string, e models.RealtimeQueueEntry, now time.Time) models.RealtimeParticipant {
	return models.RealtimeParticipant{MatchID: matchID, UserID: e.UserID, Status: "joined", Rating: e.Rating, Region: e.Region, LatencyMS: e.LatencyMS, JoinedAt: now, LastSeenAt: now}
}

func compatibleJurisdiction(a, b models.RealtimeQueueEntry) bool {
	if a.WalletCategory == "practice" {
		return true
	}
	return a.Jurisdiction != "" && a.Jurisdiction == b.Jurisdiction
}

func (s *Service) supportsMode(gameID, mode string) bool {
	if registry := s.store.GamesRegistry(); registry != nil {
		if module, err := registry.ResolveForNewMatch(gameID); err == nil {
			for _, allowed := range module.Descriptor().Modes {
				if strings.EqualFold(strings.TrimSpace(allowed), mode) {
					return true
				}
			}
			return false
		}
	}
	module, err := s.store.ArenaRegistry().Get(gameID)
	if err != nil {
		return false
	}
	caps := module.Capabilities()
	switch mode {
	case "practice", "tutorial":
		return caps.Practice
	case "pvp", "ranked":
		return caps.PvP
	case "tournament":
		return caps.Tournament
	default:
		return false
	}
}

func abs(value int) int {
	if value < 0 {
		return -value
	}
	return value
}

var _ = sql.ErrNoRows
