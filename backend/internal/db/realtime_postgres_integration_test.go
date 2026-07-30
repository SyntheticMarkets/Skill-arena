package db

import (
	"context"
	"encoding/json"
	"os"
	"testing"
	"time"

	"skill-arena/internal/config"
	"skill-arena/internal/models"
)

func TestPostgresRealtimeRepository(t *testing.T) {
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
	if _, err := store.pg.ExecContext(ctx, `TRUNCATE realtime_replays,realtime_snapshots,realtime_events,realtime_presence,realtime_queue,realtime_participants,realtime_matches,users CASCADE`); err != nil {
		t.Fatal(err)
	}
	user := models.NewUser("realtime-pg-user", "realtime-pg@example.com", "hash")
	if err := store.CreateUser(ctx, user); err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC()
	match := models.RealtimeMatch{
		ID: "realtime-pg-match", GameID: "test", GameVersion: "1", RulesVersion: "1",
		ProtocolVersion: "1", ReplayVersion: "1", Mode: "practice", Status: models.MatchCreated,
		Region: "global", WalletCategory: "practice", StateVersion: 1, CreatedAt: now, UpdatedAt: now,
	}
	participant := models.RealtimeParticipant{MatchID: match.ID, UserID: user.ID, Status: "joined", Region: "global", JoinedAt: now, LastSeenAt: now}
	created, err := store.CreateRealtimeMatch(ctx, match, participant)
	if err != nil || len(created.Participants) != 1 {
		t.Fatalf("create match=%+v err=%v", created, err)
	}
	event, err := store.AppendRealtimeEvent(ctx, match.ID, user.ID, "match_created", json.RawMessage(`{"mode":"practice"}`))
	if err != nil || event.Sequence != 1 || event.IntegrityHash == "" {
		t.Fatalf("event=%+v err=%v", event, err)
	}
	state := json.RawMessage(`{"status":"created"}`)
	if err := store.SaveRealtimeSnapshot(ctx, models.RealtimeSnapshot{MatchID: match.ID, Version: 1, Sequence: 1, State: state, Checksum: "checksum", CreatedAt: now}); err != nil {
		t.Fatal(err)
	}
	snapshot, err := store.LatestRealtimeSnapshot(ctx, match.ID)
	if err != nil || snapshot.Checksum != "checksum" {
		t.Fatalf("snapshot=%+v err=%v", snapshot, err)
	}
	entry := models.RealtimeQueueEntry{ID: "queue-pg", UserID: user.ID, GameID: "test", Mode: "pvp", WalletCategory: "practice", Region: "global", Jurisdiction: "ZA", Status: models.QueueWaiting, CreatedAt: now, UpdatedAt: now, ExpiresAt: now.Add(time.Minute)}
	if err := store.UpsertRealtimeQueue(ctx, entry); err != nil {
		t.Fatal(err)
	}
	waiting, err := store.WaitingRealtimeQueue(ctx, "test", "pvp", "practice", "global", now)
	if err != nil || len(waiting) != 1 {
		t.Fatalf("waiting=%+v err=%v", waiting, err)
	}
	if err := store.SavePresence(ctx, models.PresenceRecord{UserID: user.ID, State: models.PresenceInQueue, LastHeartbeat: now, ExpiresAt: now.Add(time.Minute)}); err != nil {
		t.Fatal(err)
	}
	metrics, err := store.RealtimeMetrics(ctx)
	if err != nil || metrics.OnlinePlayers != 1 || metrics.QueuedPlayers != 1 || metrics.MatchesCreated != 1 {
		t.Fatalf("metrics=%+v err=%v", metrics, err)
	}
	transition := *created
	transition.Status = models.MatchWaiting
	saved, transitionEvent, err := store.TransitionRealtimeMatch(
		ctx, transition, created.StateVersion, user.ID, "match_waiting",
		json.RawMessage(`{"reason":"integration"}`),
	)
	if err != nil || saved.StateVersion != created.StateVersion+1 ||
		saved.Sequence != event.Sequence+1 || transitionEvent.Sequence != saved.Sequence {
		t.Fatalf("transition match=%+v event=%+v err=%v", saved, transitionEvent, err)
	}
	latest, err := store.LatestRealtimeSnapshot(ctx, match.ID)
	if err != nil || latest.Version != saved.StateVersion ||
		latest.Sequence != saved.Sequence || latest.Checksum == "" {
		t.Fatalf("transition snapshot=%+v err=%v", latest, err)
	}
}
