package realtime

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"skill-arena/internal/arena/core"
	"skill-arena/internal/db"
	"skill-arena/internal/games/testarena"
	"skill-arena/internal/models"
)

func TestRealtimeLifecycleIsGameAgnosticAndAuthoritative(t *testing.T) {
	ctx := context.Background()
	store, err := db.New(ctx, t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = store.Close(context.Background()) })
	if err := store.ArenaRegistry().Register(newPvPTestModule()); err != nil {
		t.Fatal(err)
	}
	first := createRealtimeUser(t, store, "first@example.com")
	second := createRealtimeUser(t, store, "second@example.com")
	outsider := createRealtimeUser(t, store, "outsider@example.com")
	service := NewService(store)
	request := QueueRequest{GameID: testarena.ModuleID, Mode: "pvp", WalletCategory: "practice", Region: "af-south", Jurisdiction: "ZA", LatencyMS: 20}

	entry, match, err := service.Queue(ctx, first.ID, request)
	if err != nil || match != nil || entry.Status != models.QueueWaiting {
		t.Fatalf("first queue: entry=%+v match=%+v err=%v", entry, match, err)
	}
	_, match, err = service.Queue(ctx, second.ID, request)
	if err != nil {
		t.Fatal(err)
	}
	if match == nil || match.GameID != testarena.ModuleID || len(match.Participants) != 2 {
		t.Fatalf("generic match was not paired: %+v", match)
	}
	if _, err := service.Match(ctx, outsider.ID, match.ID); !errors.Is(err, ErrNotParticipant) {
		t.Fatalf("outsider access should fail, got %v", err)
	}
	match, err = service.Ready(ctx, first.ID, match.ID)
	if err != nil || match.Status != models.MatchWaiting {
		t.Fatalf("first ready: %s %v", match.Status, err)
	}
	match, err = service.Ready(ctx, second.ID, match.ID)
	if err != nil || match.Status != models.MatchLive {
		t.Fatalf("second ready did not start server lifecycle: %+v %v", match, err)
	}
	if err := service.Disconnect(ctx, first.ID, "session-1", "connection-1", match.ID); err != nil {
		t.Fatal(err)
	}
	match, err = service.Match(ctx, second.ID, match.ID)
	if err != nil || match.Status != models.MatchReconnecting {
		t.Fatalf("disconnect did not enter recovery: %+v %v", match, err)
	}
	match, events, err := service.Reconnect(ctx, first.ID, match.ID, 0)
	if err != nil || match.Status != models.MatchLive || len(events) < 5 {
		t.Fatalf("reconnect failed: match=%+v events=%d err=%v", match, len(events), err)
	}
	assertEventChain(t, events)
	match, err = service.Leave(ctx, first.ID, match.ID)
	if err != nil || match.Status != models.MatchCompleted {
		t.Fatalf("server forfeit completion failed: %+v %v", match, err)
	}
	if err := service.FinalizeReplay(ctx, match.ID); err != nil {
		t.Fatal(err)
	}
	replay, err := service.Replay(ctx, second.ID, match.ID)
	if err != nil || replay.EventCount == 0 || replay.EventRootHash == "" || replay.Signature == "" || replay.StorageKey == "" {
		t.Fatalf("replay integrity metadata incomplete: %+v %v", replay, err)
	}
}

func TestPracticeSeedsAndStateAreIndependent(t *testing.T) {
	ctx := context.Background()
	store, err := db.New(ctx, t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = store.Close(context.Background()) })
	if err := store.ArenaRegistry().Register(newPvPTestModule()); err != nil {
		t.Fatal(err)
	}
	first := createRealtimeUser(t, store, "practice-1@example.com")
	second := createRealtimeUser(t, store, "practice-2@example.com")
	service := NewService(store)
	request := QueueRequest{GameID: testarena.ModuleID, Mode: "practice", WalletCategory: "practice", Region: "global", LatencyMS: 5}
	_, firstMatch, err := service.Queue(ctx, first.ID, request)
	if err != nil {
		t.Fatal(err)
	}
	_, secondMatch, err := service.Queue(ctx, second.ID, request)
	if err != nil {
		t.Fatal(err)
	}
	if firstMatch.ID == secondMatch.ID || firstMatch.SeedReference == secondMatch.SeedReference {
		t.Fatal("practice matches must have independent match and seed references")
	}
}

func TestConcurrentQueueDoesNotPairPlayerTwice(t *testing.T) {
	ctx := context.Background()
	store, err := db.New(ctx, t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = store.Close(context.Background()) })
	if err := store.ArenaRegistry().Register(newPvPTestModule()); err != nil {
		t.Fatal(err)
	}
	service := NewService(store)
	request := QueueRequest{GameID: testarena.ModuleID, Mode: "pvp", WalletCategory: "practice", Region: "af-south", Jurisdiction: "ZA", LatencyMS: 10}
	users := make([]*models.User, 100)
	for i := range users {
		users[i] = createRealtimeUser(t, store, fmt.Sprintf("load-%03d@example.com", i))
	}
	var wg sync.WaitGroup
	errs := make(chan error, len(users))
	for _, user := range users {
		wg.Add(1)
		go func(userID string) {
			defer wg.Done()
			_, _, err := service.Queue(ctx, userID, request)
			if err != nil && !errors.Is(err, ErrAlreadyQueued) {
				errs <- err
			}
		}(user.ID)
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		t.Error(err)
	}
	for _, user := range users {
		entry, err := service.QueueStatus(ctx, user.ID)
		if err != nil {
			t.Fatal(err)
		}
		if entry.Status != models.QueueWaiting && entry.Status != models.QueueMatched {
			t.Fatalf("unexpected queue state %s", entry.Status)
		}
		count, err := store.RealtimeMatchCountForUser(ctx, user.ID)
		if err != nil {
			t.Fatal(err)
		}
		if count > 1 {
			t.Fatalf("player %s was paired into %d matches", user.ID, count)
		}
	}
}

func BenchmarkRealtimePracticeLifecycle(b *testing.B) {
	ctx := context.Background()
	store, err := db.New(ctx, b.TempDir())
	if err != nil {
		b.Fatal(err)
	}
	defer store.Close(context.Background())
	if err := store.ArenaRegistry().Register(newPvPTestModule()); err != nil {
		b.Fatal(err)
	}
	service := NewService(store)
	users := make([]*models.User, b.N)
	for i := range users {
		users[i] = models.NewUser("", "bench-"+time.Now().Add(time.Duration(i)).Format("150405.000000000")+"@example.com", "hash")
		if err := store.CreateUser(ctx, users[i]); err != nil {
			b.Fatal(err)
		}
	}
	request := QueueRequest{GameID: testarena.ModuleID, Mode: "practice", WalletCategory: "practice", Region: "global", LatencyMS: 5}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, _, err := service.Queue(ctx, users[i].ID, request); err != nil {
			b.Fatal(err)
		}
	}
}

func createRealtimeUser(t testing.TB, store *db.Store, email string) *models.User {
	t.Helper()
	user := models.NewUser("", email, "hash")
	user.EmailVerified = true
	user.Country = "ZA"
	if err := store.CreateUser(context.Background(), user); err != nil {
		t.Fatal(err)
	}
	return user
}

func assertEventChain(t *testing.T, events []models.RealtimeEvent) {
	t.Helper()
	for i := range events {
		if events[i].Sequence != int64(i+1) || events[i].IntegrityHash == "" {
			t.Fatalf("invalid event sequence/hash at %d: %+v", i, events[i])
		}
		if i > 0 && events[i].PreviousHash != events[i-1].IntegrityHash {
			t.Fatalf("event chain broken at %d", i)
		}
	}
}

type pvpTestModule struct {
	core.GameModule
}

func newPvPTestModule() core.GameModule {
	return pvpTestModule{GameModule: testarena.New()}
}

func (m pvpTestModule) Capabilities() core.CapabilityFlags {
	capabilities := m.GameModule.Capabilities()
	capabilities.PvP = true
	return capabilities
}

func (m pvpTestModule) Manifest() core.Manifest {
	manifest := m.GameModule.Manifest()
	manifest.MaximumPlayers = 2
	manifest.Modes = append(manifest.Modes, "pvp")
	manifest.Capabilities = m.Capabilities()
	return manifest
}

func (m pvpTestModule) Metadata() core.Metadata {
	metadata := m.GameModule.Metadata()
	metadata.MaximumPlayers = 2
	metadata.Modes = append(metadata.Modes, "pvp")
	metadata.Capabilities = m.Capabilities()
	return metadata
}
