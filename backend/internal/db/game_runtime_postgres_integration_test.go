package db

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"sync"
	"testing"
	"time"

	"skill-arena/internal/config"
	"skill-arena/internal/models"
)

func TestPostgresGameRuntimeAtomicStateReceiptAndEvents(t *testing.T) {
	databaseURL := os.Getenv("SKILL_ARENA_TEST_POSTGRES_URL")
	if databaseURL == "" {
		t.Skip("SKILL_ARENA_TEST_POSTGRES_URL is not configured")
	}
	ctx := context.Background()
	store, err := NewWithOptions(ctx, Options{
		DatabaseURL: databaseURL, Environment: "development",
		Storage: config.StorageSettings{LocalRoot: t.TempDir()},
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = store.Close(context.Background()) })
	if _, err := store.pg.ExecContext(ctx, `TRUNCATE
game_action_receipts,game_participant_states,realtime_replays,realtime_snapshots,
realtime_events,realtime_presence,realtime_queue,realtime_participants,
realtime_matches,users CASCADE`); err != nil {
		t.Fatal(err)
	}
	user := models.NewUser("runtime-pg-user", "runtime-pg@example.com", "hash")
	if err := store.CreateUser(ctx, user); err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC().Truncate(time.Microsecond)
	match := models.RealtimeMatch{
		ID: "runtime-pg-match", GameID: "test", GameVersion: "1",
		RulesVersion: "1", ProtocolVersion: "1", ReplayVersion: "1",
		Mode: "practice", Status: models.MatchLive, Region: "global",
		WalletCategory: "practice", StateVersion: 1, CreatedAt: now, UpdatedAt: now,
	}
	participant := models.RealtimeParticipant{
		MatchID: match.ID, UserID: user.ID, Status: "ready", Ready: true,
		Region: "global", JoinedAt: now, LastSeenAt: now,
	}
	if _, err := store.CreateRealtimeMatch(ctx, match, participant); err != nil {
		t.Fatal(err)
	}
	initial := models.GameParticipantState{
		MatchID: match.ID, UserID: user.ID, GameID: "test", PuzzleID: "puzzle-1",
		StateSchema: "1", StateVersion: 0, State: json.RawMessage(`{"version":0}`),
		StateChecksum: "checksum-0", Status: "active", UpdatedAt: now,
	}
	if _, err := store.CreateGameParticipantStates(ctx, []models.GameParticipantState{initial}, models.GameEventDraft{
		Type: "game.puzzle.ready", Payload: json.RawMessage(`{"ready":true}`),
	}); err != nil {
		t.Fatal(err)
	}
	next := initial
	next.StateVersion = 1
	next.State = json.RawMessage(`{"version":1}`)
	next.StateChecksum = "checksum-1"
	next.LastClientSequence = 1
	next.UpdatedAt = now.Add(time.Millisecond)
	receipt := models.GameActionReceipt{
		ActionID: "action-1", MatchID: match.ID, UserID: user.ID,
		ClientSequence: 1, ActionKind: "test.action",
		ActionPayloadHash: "payload-hash", Accepted: true, ResultCode: "accepted",
		StateVersionAfter: 1, Transition: json.RawMessage(`{"accepted":true}`),
		ReceiptHash: "receipt-hash", ServerReceivedAt: now, ProcessedAt: now,
	}
	committed, events, err := store.CommitGameAction(
		ctx, initial, next, receipt,
		[]models.GameEventDraft{{Type: "game.action.processed", Payload: json.RawMessage(`{"accepted":true}`)}},
	)
	if err != nil || committed.FirstEventSequence == 0 ||
		len(events) != 1 || events[0].IntegrityHash == "" {
		t.Fatalf("commit=%+v events=%+v err=%v", committed, events, err)
	}
	loaded, err := store.GetGameParticipantState(ctx, match.ID, user.ID)
	if err != nil || loaded.StateVersion != 1 || loaded.LastClientSequence != 1 ||
		loaded.LastServerSequence != events[0].Sequence {
		t.Fatalf("loaded=%+v err=%v", loaded, err)
	}
	original, err := store.GetGameActionReceipt(ctx, match.ID, user.ID, "action-1", 1)
	if err != nil || original.ReceiptHash != receipt.ReceiptHash {
		t.Fatalf("receipt=%+v err=%v", original, err)
	}

	expected := *loaded
	results := make(chan error, 2)
	var wait sync.WaitGroup
	for index := 0; index < 2; index++ {
		wait.Add(1)
		go func(index int) {
			defer wait.Done()
			candidate := expected
			candidate.StateVersion = 2
			candidate.State = json.RawMessage(`{"version":2}`)
			candidate.StateChecksum = "checksum-2"
			candidate.LastClientSequence = 2
			action := models.GameActionReceipt{
				ActionID: "action-concurrent-" + string(rune('a'+index)),
				MatchID:  match.ID, UserID: user.ID, ClientSequence: 2,
				ExpectedStateVersion: 1, ActionKind: "test.action",
				ActionPayloadHash: "payload-concurrent", Accepted: true,
				ResultCode: "accepted", StateVersionBefore: 1, StateVersionAfter: 2,
				Transition:  json.RawMessage(`{"accepted":true}`),
				ReceiptHash: "receipt-concurrent", ServerReceivedAt: now, ProcessedAt: now,
			}
			_, _, commitErr := store.CommitGameAction(
				ctx, expected, candidate, action,
				[]models.GameEventDraft{{Type: "game.action.processed", Payload: json.RawMessage(`{}`)}},
			)
			results <- commitErr
		}(index)
	}
	wait.Wait()
	close(results)
	successes, conflicts := 0, 0
	for result := range results {
		if result == nil {
			successes++
		} else if errors.Is(result, ErrRealtimeConflict) {
			conflicts++
		} else {
			t.Fatalf("unexpected commit error: %v", result)
		}
	}
	if successes != 1 || conflicts != 1 {
		t.Fatalf("successes=%d conflicts=%d", successes, conflicts)
	}
}
