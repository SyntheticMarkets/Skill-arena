package realtime

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"

	"skill-arena/internal/config"
	"skill-arena/internal/db"
	"skill-arena/internal/games/interfaces"
	"skill-arena/internal/games/maze"
	"skill-arena/internal/games/maze/engine"
	"skill-arena/internal/games/maze/generator"
	mazereplay "skill-arena/internal/games/maze/replay"
	"skill-arena/internal/games/maze/solver"
	gamesession "skill-arena/internal/games/session"
	"skill-arena/internal/models"

	"github.com/gorilla/websocket"
)

func TestMazePracticeLifecycleIsAuthoritativeIdempotentAndReplayable(t *testing.T) {
	ctx := context.Background()
	store, err := db.New(ctx, t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = store.Close(context.Background()) })
	player := createRealtimeUser(t, store, "maze-practice@example.com")
	service := NewService(store)
	_, match, err := service.Queue(ctx, player.ID, QueueRequest{
		GameID: maze.ModuleID, Mode: "practice", WalletCategory: "practice",
		Region: "global", Jurisdiction: "ZA", LatencyMS: 8,
	})
	if err != nil {
		t.Fatal(err)
	}
	match, err = service.Ready(ctx, player.ID, match.ID)
	if err != nil || match.Status != models.MatchLive {
		t.Fatalf("practice start: match=%+v err=%v", match, err)
	}
	initial := loadMazeEngineState(t, store, match.ID, player.ID)
	solverInstance, err := solver.New(solver.Config{Version: 1, MaxArrows: 2048})
	if err != nil {
		t.Fatal(err)
	}
	solution, err := solverInstance.Solve(ctx, initial.Board, false)
	if err != nil {
		t.Fatal(err)
	}
	blockedID := blockedArrow(t, initial)
	sequence := int64(1)
	blocked := mazeAction(match.ID, "blocked-action", blockedID, sequence, 0)
	blockedResult, err := service.GameAction(ctx, player.ID, match.ID, blocked, 8*time.Millisecond)
	if err != nil {
		t.Fatal(err)
	}
	if blockedResult.Receipt.Accepted || blockedResult.Receipt.ResultCode != engine.CodeActionBlocked ||
		blockedResult.Receipt.StateVersionAfter != 0 {
		t.Fatalf("blocked receipt = %+v", blockedResult.Receipt)
	}
	duplicate, err := service.GameAction(ctx, player.ID, match.ID, blocked, 8*time.Millisecond)
	if err != nil || !duplicate.Duplicate ||
		duplicate.Receipt.ReceiptHash != blockedResult.Receipt.ReceiptHash {
		t.Fatalf("idempotent retry = %+v err=%v", duplicate, err)
	}
	gap := mazeAction(match.ID, "gap", solution.Steps[0].ArrowID, 3, 0)
	if _, err := service.GameAction(ctx, player.ID, match.ID, gap, 8*time.Millisecond); !errors.Is(err, gamesession.ErrActionSequence) {
		t.Fatalf("sequence gap error = %v", err)
	}
	for index, step := range solution.Steps {
		sequence++
		action := mazeAction(
			match.ID, "accepted-"+step.ArrowID, step.ArrowID,
			sequence, int64(index),
		)
		result, actionErr := service.GameAction(
			ctx, player.ID, match.ID, action, 8*time.Millisecond,
		)
		if actionErr != nil {
			t.Fatalf("apply %s: %v", step.ArrowID, actionErr)
		}
		if !result.Receipt.Accepted {
			t.Fatalf("canonical action %s was blocked", step.ArrowID)
		}
	}
	finalMatch, err := service.Match(ctx, player.ID, match.ID)
	if err != nil || finalMatch.Status != models.MatchCompleted {
		t.Fatalf("practice completion = %+v err=%v", finalMatch, err)
	}
	finalState := loadMazeEngineState(t, store, match.ID, player.ID)
	if finalState.Status != engine.StatusCompleted ||
		len(finalState.RemovedIDs) != len(finalState.Board.Arrows) {
		t.Fatalf("final state = %+v", finalState)
	}
	syncResult, err := service.GameSync(ctx, player.ID, match.ID)
	if err != nil || syncResult.StateVersion != int64(finalState.Version) ||
		syncResult.Snapshot.Checksum == "" {
		t.Fatalf("terminal sync = %+v err=%v", syncResult, err)
	}
	verifyLiveReplayParity(t, store, service, finalMatch, initial, finalState)
	if err := service.FinalizeReplay(ctx, finalMatch.ID); err != nil {
		t.Fatal(err)
	}
	persisted, err := store.GetRealtimeReplay(ctx, finalMatch.ID)
	if err != nil || persisted.Status != mazereplay.StatusVerified ||
		persisted.ReplayVersion != "v2" || persisted.StorageKey == "" {
		t.Fatalf("persisted replay=%+v err=%v", persisted, err)
	}
	repository, err := mazereplay.NewObjectRepository(
		store.GamesObjectStore(), "replays/maze",
	)
	if err != nil {
		t.Fatal(err)
	}
	artifact, err := repository.Load(ctx, persisted.ID)
	if err != nil {
		t.Fatal(err)
	}
	report, err := replayServiceForTest(t, store, service).Verify(ctx, artifact)
	if err != nil || !report.Verified {
		t.Fatalf("persisted replay report=%+v err=%v", report, err)
	}
}

func TestMazeCachedActionStateCannotBypassMatchStatus(t *testing.T) {
	ctx := context.Background()
	store, err := db.New(ctx, t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = store.Close(context.Background()) })
	player := createRealtimeUser(t, store, "maze-cache-authority@example.com")
	service := NewService(store)
	_, match, err := service.Queue(ctx, player.ID, QueueRequest{
		GameID: maze.ModuleID, Mode: "practice", WalletCategory: "practice",
		Region: "global", Jurisdiction: "ZA", LatencyMS: 8,
	})
	if err != nil {
		t.Fatal(err)
	}
	if match, err = service.Ready(ctx, player.ID, match.ID); err != nil {
		t.Fatal(err)
	}
	initial := loadMazeEngineState(t, store, match.ID, player.ID)
	blocked := mazeAction(match.ID, "cache-prime", blockedArrow(t, initial), 1, 0)
	if _, err := service.GameAction(ctx, player.ID, match.ID, blocked, 8*time.Millisecond); err != nil {
		t.Fatal(err)
	}
	if err := service.Disconnect(ctx, player.ID, "session", "connection", match.ID); err != nil {
		t.Fatal(err)
	}
	solverInstance, err := solver.New(solver.Config{Version: 1, MaxArrows: 2048})
	if err != nil {
		t.Fatal(err)
	}
	solution, err := solverInstance.Solve(ctx, initial.Board, false)
	if err != nil {
		t.Fatal(err)
	}
	action := mazeAction(match.ID, "while-reconnecting", solution.Steps[0].ArrowID, 2, 0)
	if _, err := service.GameAction(
		ctx, player.ID, match.ID, action, 8*time.Millisecond,
	); !errors.Is(err, gamesession.ErrActionConflict) {
		t.Fatalf("action while reconnecting error = %v", err)
	}
}

func TestMazeConcurrentActionRetriesCommitOneTransition(t *testing.T) {
	ctx := context.Background()
	store, err := db.New(ctx, t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = store.Close(context.Background()) })
	player := createRealtimeUser(t, store, "maze-idempotency@example.com")
	service := NewService(store)
	_, match, err := service.Queue(ctx, player.ID, QueueRequest{
		GameID: maze.ModuleID, Mode: "practice", WalletCategory: "practice",
		Region: "global", Jurisdiction: "ZA", LatencyMS: 8,
	})
	if err != nil {
		t.Fatal(err)
	}
	match, err = service.Ready(ctx, player.ID, match.ID)
	if err != nil {
		t.Fatal(err)
	}
	state := loadMazeEngineState(t, store, match.ID, player.ID)
	solverInstance, _ := solver.New(solver.Config{Version: 1, MaxArrows: 2048})
	solution, err := solverInstance.Solve(ctx, state.Board, false)
	if err != nil {
		t.Fatal(err)
	}
	action := mazeAction(
		match.ID, "concurrent-action", solution.Steps[0].ArrowID, 1, 0,
	)

	const requests = 24
	results := make(chan gamesession.ActionResult, requests)
	errs := make(chan error, requests)
	var wait sync.WaitGroup
	for range requests {
		wait.Add(1)
		go func() {
			defer wait.Done()
			result, actionErr := service.GameAction(
				ctx, player.ID, match.ID, action, 8*time.Millisecond,
			)
			results <- result
			errs <- actionErr
		}()
	}
	wait.Wait()
	close(results)
	close(errs)

	for actionErr := range errs {
		if actionErr != nil {
			t.Errorf("concurrent retry: %v", actionErr)
		}
	}
	var receiptHash string
	originals := 0
	for result := range results {
		if !result.Receipt.Accepted {
			t.Fatalf("result=%+v", result)
		}
		if receiptHash == "" {
			receiptHash = result.Receipt.ReceiptHash
		} else if result.Receipt.ReceiptHash != receiptHash {
			t.Fatalf(
				"receipt hash changed: got=%s want=%s",
				result.Receipt.ReceiptHash, receiptHash,
			)
		}
		if !result.Duplicate {
			originals++
		}
	}
	if originals != 1 {
		t.Fatalf("original transitions=%d want=1", originals)
	}
	final := loadMazeEngineState(t, store, match.ID, player.ID)
	if final.Version != 1 || final.SuccessfulActions != 1 {
		t.Fatalf("state advanced more than once: %+v", final)
	}
	events, err := store.RealtimeEventsAfter(ctx, match.ID, 0, 100)
	if err != nil {
		t.Fatal(err)
	}
	processed := 0
	for _, event := range events {
		if event.Type != "game.action.processed" {
			continue
		}
		var payload struct {
			ActionID string `json:"actionId"`
		}
		if err := json.Unmarshal(event.Payload, &payload); err != nil {
			t.Fatal(err)
		}
		if payload.ActionID == action.ActionID {
			processed++
		}
	}
	if processed != 1 {
		t.Fatalf("processed events=%d want=1", processed)
	}
}

func TestMazePvPUsesSharedPuzzleAndIndependentParticipantState(t *testing.T) {
	ctx := context.Background()
	store, err := db.New(ctx, t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = store.Close(context.Background()) })
	first := createRealtimeUser(t, store, "maze-pvp-first@example.com")
	second := createRealtimeUser(t, store, "maze-pvp-second@example.com")
	service := NewService(store)
	request := QueueRequest{
		GameID: maze.ModuleID, Mode: "pvp", WalletCategory: "practice",
		Region: "af-south", Jurisdiction: "ZA", LatencyMS: 12,
	}
	if _, match, err := service.Queue(ctx, first.ID, request); err != nil || match != nil {
		t.Fatalf("first queue match=%+v err=%v", match, err)
	}
	_, match, err := service.Queue(ctx, second.ID, request)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.Ready(ctx, first.ID, match.ID); err != nil {
		t.Fatal(err)
	}
	match, err = service.Ready(ctx, second.ID, match.ID)
	if err != nil || match.Status != models.MatchLive {
		t.Fatalf("PvP start = %+v err=%v", match, err)
	}
	firstState := loadMazeEngineState(t, store, match.ID, first.ID)
	secondState := loadMazeEngineState(t, store, match.ID, second.ID)
	if firstState.PuzzleID != secondState.PuzzleID ||
		firstState.PuzzleHash != secondState.PuzzleHash ||
		firstState.BoardHash != secondState.BoardHash ||
		!reflect.DeepEqual(firstState.Board, secondState.Board) {
		t.Fatal("PvP participants did not receive identical immutable puzzle data")
	}
	if firstState.ParticipantID == secondState.ParticipantID ||
		firstState.Checksum == secondState.Checksum {
		t.Fatal("PvP participant states are not independently identified")
	}
	solverInstance, _ := solver.New(solver.Config{Version: 1, MaxArrows: 2048})
	solution, err := solverInstance.Solve(ctx, firstState.Board, false)
	if err != nil {
		t.Fatal(err)
	}
	_, err = service.GameAction(ctx, first.ID, match.ID, mazeAction(
		match.ID, "first-move", solution.Steps[0].ArrowID, 1, 0,
	), 12*time.Millisecond)
	if err != nil {
		t.Fatal(err)
	}
	firstAfter := loadMazeEngineState(t, store, match.ID, first.ID)
	secondAfter := loadMazeEngineState(t, store, match.ID, second.ID)
	if firstAfter.Version != 1 || secondAfter.Version != 0 ||
		len(secondAfter.RemovedIDs) != 0 {
		t.Fatalf("participant state leaked: first=%+v second=%+v", firstAfter, secondAfter)
	}
	if err := service.Disconnect(ctx, first.ID, "session-a", "connection-a", match.ID); err != nil {
		t.Fatal(err)
	}
	reconnected, _, err := service.Reconnect(ctx, first.ID, match.ID, 0)
	if err != nil || reconnected.Status != models.MatchLive {
		t.Fatalf("reconnect = %+v err=%v", reconnected, err)
	}
	syncResult, err := service.GameSync(ctx, first.ID, match.ID)
	if err != nil || syncResult.StateVersion != 1 ||
		syncResult.LastClientSequence != 1 {
		t.Fatalf("recovered state = %+v err=%v", syncResult, err)
	}
	if _, err := service.Leave(ctx, first.ID, match.ID); err != nil {
		t.Fatal(err)
	}
	forfeited, err := store.GetGameParticipantState(ctx, match.ID, first.ID)
	if err != nil || forfeited.Status != "forfeited" {
		t.Fatalf("forfeit state=%+v err=%v", forfeited, err)
	}
}

func TestMazePracticeAssignmentsAreFreshUnderConcurrency(t *testing.T) {
	ctx := context.Background()
	store, err := db.New(ctx, t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = store.Close(context.Background()) })
	service := NewService(store)
	users := make([]*models.User, 8)
	for index := range users {
		users[index] = createRealtimeUser(
			t, store, "maze-fresh-"+string(rune('a'+index))+"@example.com",
		)
	}
	puzzles := make(chan string, len(users))
	errs := make(chan error, len(users))
	var wait sync.WaitGroup
	for _, user := range users {
		wait.Add(1)
		go func(userID string) {
			defer wait.Done()
			_, match, queueErr := service.Queue(ctx, userID, QueueRequest{
				GameID: maze.ModuleID, Mode: "practice", WalletCategory: "practice",
				Region: "global", Jurisdiction: "ZA", LatencyMS: 5,
			})
			if queueErr != nil {
				errs <- queueErr
				return
			}
			match, readyErr := service.Ready(ctx, userID, match.ID)
			if readyErr != nil {
				errs <- readyErr
				return
			}
			state, stateErr := store.GetGameParticipantState(ctx, match.ID, userID)
			if stateErr != nil {
				errs <- stateErr
				return
			}
			puzzles <- state.PuzzleID
		}(user.ID)
	}
	wait.Wait()
	close(errs)
	close(puzzles)
	for err := range errs {
		t.Error(err)
	}
	seen := map[string]bool{}
	for puzzleID := range puzzles {
		if seen[puzzleID] {
			t.Fatalf("practice puzzle was intentionally reused: %s", puzzleID)
		}
		seen[puzzleID] = true
	}
	if len(seen) != len(users) {
		t.Fatalf("fresh puzzles=%d users=%d", len(seen), len(users))
	}
}

func TestGatewayDispatchesGenericMazeActionAndSync(t *testing.T) {
	ctx := context.Background()
	store, err := db.New(ctx, t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = store.Close(context.Background()) })
	player := createRealtimeUser(t, store, "maze-gateway@example.com")
	settings := config.LoadRuntimeSettings()
	settings.CORS.AllowedOrigins = []string{"http://localhost:3000"}
	store.ConfigureRuntime(settings)
	service := NewService(store)
	_, match, err := service.Queue(ctx, player.ID, QueueRequest{
		GameID: maze.ModuleID, Mode: "practice", WalletCategory: "practice",
		Region: "global", Jurisdiction: "ZA", LatencyMS: 7,
	})
	if err != nil {
		t.Fatal(err)
	}
	match, err = service.Ready(ctx, player.ID, match.ID)
	if err != nil {
		t.Fatal(err)
	}
	state := loadMazeEngineState(t, store, match.ID, player.ID)
	solverInstance, _ := solver.New(solver.Config{Version: 1, MaxArrows: 2048})
	solution, err := solverInstance.Solve(ctx, state.Board, false)
	if err != nil {
		t.Fatal(err)
	}
	gateway := NewGateway(
		store, service, &config.Config{Settings: settings},
	)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gateway.ServeAuthenticated(w, r, player.ID, "session-maze")
	}))
	defer server.Close()
	header := http.Header{"Origin": []string{"http://localhost:3000"}}
	conn, _, err := websocket.DefaultDialer.Dial(
		"ws"+strings.TrimPrefix(server.URL, "http"), header,
	)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	var response map[string]any
	if err := conn.ReadJSON(&response); err != nil || response["type"] != "session.negotiated" {
		t.Fatalf("negotiation=%+v err=%v", response, err)
	}
	payload, _ := json.Marshal(engine.ArrowClick{ArrowID: solution.Steps[0].ArrowID})
	if err := conn.WriteJSON(map[string]any{
		"type": "game.action", "matchId": match.ID, "actionId": "gateway-action",
		"kind": engine.ActionArrowClick, "payload": json.RawMessage(payload),
		"clientSequence": 1, "expectedStateVersion": 0, "latencyMs": 7,
	}); err != nil {
		t.Fatal(err)
	}
	if err := conn.ReadJSON(&response); err != nil || response["type"] != "game.action.receipt" {
		t.Fatalf("action receipt=%+v err=%v", response, err)
	}
	if err := conn.WriteJSON(map[string]any{
		"type": "game.sync.request", "matchId": match.ID,
	}); err != nil {
		t.Fatal(err)
	}
	if err := conn.ReadJSON(&response); err != nil || response["type"] != "game.state.sync" {
		t.Fatalf("game sync=%+v err=%v", response, err)
	}
}

func TestMazeDeadlineIsAppliedByServerMaintenance(t *testing.T) {
	ctx := context.Background()
	store, err := db.New(ctx, t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = store.Close(context.Background()) })
	player := createRealtimeUser(t, store, "maze-timeout@example.com")
	service := NewService(store)
	_, match, err := service.Queue(ctx, player.ID, QueueRequest{
		GameID: maze.ModuleID, Mode: "practice", WalletCategory: "practice",
		Region: "global", Jurisdiction: "ZA", LatencyMS: 5,
	})
	if err != nil {
		t.Fatal(err)
	}
	match, err = service.Ready(ctx, player.ID, match.ID)
	if err != nil {
		t.Fatal(err)
	}
	state := loadMazeEngineState(t, store, match.ID, player.ID)
	expired, err := service.ExpireDueGameMatches(
		ctx, time.UnixMilli(state.DeadlineAtMS+1),
	)
	if err != nil || expired != 1 {
		t.Fatalf("expired=%d err=%v", expired, err)
	}
	final := loadMazeEngineState(t, store, match.ID, player.ID)
	if final.Status != engine.StatusTimedOut ||
		final.CompletedAtMS != state.DeadlineAtMS+1 {
		t.Fatalf("timed-out state=%+v", final)
	}
	match, err = service.Match(ctx, player.ID, match.ID)
	if err != nil || match.Status != models.MatchCompleted {
		t.Fatalf("timed-out match=%+v err=%v", match, err)
	}
}

func BenchmarkMazeRealtimeIdempotentAction(b *testing.B) {
	ctx := context.Background()
	store, err := db.New(ctx, b.TempDir())
	if err != nil {
		b.Fatal(err)
	}
	b.Cleanup(func() { _ = store.Close(context.Background()) })
	player := createRealtimeUser(b, store, "maze-benchmark-action@example.com")
	service := NewService(store)
	_, match, err := service.Queue(ctx, player.ID, QueueRequest{
		GameID: maze.ModuleID, Mode: "practice", WalletCategory: "practice",
		Region: "global", Jurisdiction: "ZA", LatencyMS: 5,
	})
	if err != nil {
		b.Fatal(err)
	}
	match, err = service.Ready(ctx, player.ID, match.ID)
	if err != nil {
		b.Fatal(err)
	}
	state := loadMazeEngineState(b, store, match.ID, player.ID)
	solverInstance, _ := solver.New(solver.Config{Version: 1, MaxArrows: 2048})
	solution, err := solverInstance.Solve(ctx, state.Board, false)
	if err != nil {
		b.Fatal(err)
	}
	action := mazeAction(match.ID, "benchmark-action", solution.Steps[0].ArrowID, 1, 0)
	if _, err := service.GameAction(
		ctx, player.ID, match.ID, action, 5*time.Millisecond,
	); err != nil {
		b.Fatal(err)
	}
	b.ResetTimer()
	b.ReportAllocs()
	for range b.N {
		if _, err := service.GameAction(
			ctx, player.ID, match.ID, action, 5*time.Millisecond,
		); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkMazeRealtimeStateSync(b *testing.B) {
	ctx := context.Background()
	store, err := db.New(ctx, b.TempDir())
	if err != nil {
		b.Fatal(err)
	}
	b.Cleanup(func() { _ = store.Close(context.Background()) })
	player := createRealtimeUser(b, store, "maze-benchmark-sync@example.com")
	service := NewService(store)
	_, match, err := service.Queue(ctx, player.ID, QueueRequest{
		GameID: maze.ModuleID, Mode: "practice", WalletCategory: "practice",
		Region: "global", Jurisdiction: "ZA", LatencyMS: 5,
	})
	if err != nil {
		b.Fatal(err)
	}
	match, err = service.Ready(ctx, player.ID, match.ID)
	if err != nil {
		b.Fatal(err)
	}
	b.ResetTimer()
	b.ReportAllocs()
	for range b.N {
		if _, err := service.GameSync(ctx, player.ID, match.ID); err != nil {
			b.Fatal(err)
		}
	}
}

func mazeAction(
	matchID, actionID, arrowID string,
	sequence, stateVersion int64,
) interfaces.ActionEnvelope {
	payload, _ := json.Marshal(engine.ArrowClick{ArrowID: arrowID})
	return interfaces.ActionEnvelope{
		ActionID: actionID, MatchID: matchID, Kind: engine.ActionArrowClick,
		Payload: payload, ClientSequence: sequence,
		ExpectedStateVersion: stateVersion,
	}
}

func loadMazeEngineState(
	t testing.TB,
	store *db.Store,
	matchID, userID string,
) engine.State {
	t.Helper()
	record, err := store.GetGameParticipantState(context.Background(), matchID, userID)
	if err != nil {
		t.Fatal(err)
	}
	var generic interfaces.GameState
	if err := json.Unmarshal(record.State, &generic); err != nil {
		t.Fatal(err)
	}
	state, err := engine.DecodeState(generic)
	if err != nil {
		t.Fatal(err)
	}
	return state
}

func blockedArrow(t testing.TB, state engine.State) string {
	t.Helper()
	model, err := engine.NewCollisionModel(state.Board)
	if err != nil {
		t.Fatal(err)
	}
	for _, arrow := range state.Board.Arrows {
		collision, collisionErr := model.Evaluate(nil, arrow.ID)
		if collisionErr != nil {
			t.Fatal(collisionErr)
		}
		if !collision.Clear {
			return arrow.ID
		}
	}
	t.Fatal("generated puzzle has no initially blocked arrow")
	return ""
}

func verifyLiveReplayParity(
	t testing.TB,
	store *db.Store,
	integrity *Service,
	match *models.RealtimeMatch,
	initial, final engine.State,
) {
	t.Helper()
	ctx := context.Background()
	solverInstance, err := solver.New(solver.Config{Version: 1, MaxArrows: 2048})
	if err != nil {
		t.Fatal(err)
	}
	processor, err := generator.NewProductionProcessor(
		generator.DefaultGenerationConfig(), solverInstance, nil,
	)
	if err != nil {
		t.Fatal(err)
	}
	replayService, err := mazereplay.NewService(
		store.GamesPuzzleService(), processor, solverInstance, integrity,
	)
	if err != nil {
		t.Fatal(err)
	}
	input, err := store.GamesPuzzleService().LoadReconstructionInput(ctx, initial.PuzzleID)
	if err != nil {
		t.Fatal(err)
	}
	genesis, err := replayService.BuildGenesis(ctx, mazereplay.GenesisRequest{
		ReplayID: "runtime-" + match.ID, MatchID: match.ID, PuzzleID: initial.PuzzleID,
		Versions: mazereplay.Versions{
			GameVersion: match.GameVersion, ProtocolVersion: 1,
			ReplayVersion:      mazereplay.ReplayVersionEngine,
			RendererVersion:    engine.RendererVersion,
			StateSchemaVersion: engine.StateSchemaVersion,
			Generator:          input.Metadata.Version,
		},
		CreatedAtUnixMS: initial.StartedAtMS,
	})
	if err != nil {
		t.Fatal(err)
	}
	events, err := store.RealtimeEventsAfter(ctx, match.ID, 0, 500)
	if err != nil {
		t.Fatal(err)
	}
	drafts := make([]mazereplay.EventDraft, 0)
	for _, event := range events {
		if event.Type != "game.action.processed" {
			continue
		}
		var payload struct {
			Payload struct {
				ArrowID string `json:"arrowId"`
			} `json:"payload"`
			Accepted     bool   `json:"accepted"`
			Code         string `json:"code"`
			OccurredAtMS int64  `json:"occurredAtMs"`
		}
		if err := json.Unmarshal(event.Payload, &payload); err != nil {
			t.Fatal(err)
		}
		kind := mazereplay.EventArrowBlocked
		if payload.Accepted {
			kind = mazereplay.EventArrowAccepted
		}
		drafts = append(drafts, mazereplay.EventDraft{
			Sequence: uint64(len(drafts) + 1), ParticipantID: final.ParticipantID,
			OffsetMS: payload.OccurredAtMS - initial.StartedAtMS,
			Kind:     kind, ArrowID: payload.Payload.ArrowID, Code: payload.Code,
		})
	}
	artifact, err := replayService.Seal(ctx, mazereplay.SealRequest{
		Genesis: genesis, ParticipantIDs: []string{final.ParticipantID},
		Events: drafts,
		Outcome: mazereplay.Outcome{
			Status: "completed", WinnerIDs: []string{final.ParticipantID},
		},
		StartedAtUnixMS: initial.StartedAtMS, EndedAtUnixMS: final.CompletedAtMS,
	})
	if err != nil {
		t.Fatal(err)
	}
	report, err := replayService.Verify(ctx, artifact)
	if err != nil || !report.Verified || len(report.Participants) != 1 {
		t.Fatalf("replay report=%+v err=%v", report, err)
	}
	projection := engine.ReplayProjection(final, uint64(len(drafts)))
	if report.Participants[0].StateChecksum != projection.StateChecksum ||
		report.Participants[0].SuccessfulActions != final.SuccessfulActions ||
		report.Participants[0].BlockedActions != final.BlockedActions ||
		report.Participants[0].MaximumCombo != final.MaximumCombo {
		t.Fatalf("live/replay mismatch: replay=%+v live=%+v", report.Participants[0], projection)
	}
}

func replayServiceForTest(
	t testing.TB,
	store *db.Store,
	integrity *Service,
) *mazereplay.Service {
	t.Helper()
	solverInstance, err := solver.New(solver.Config{Version: 1, MaxArrows: 2048})
	if err != nil {
		t.Fatal(err)
	}
	processor, err := generator.NewProductionProcessor(
		generator.DefaultGenerationConfig(), solverInstance, nil,
	)
	if err != nil {
		t.Fatal(err)
	}
	service, err := mazereplay.NewService(
		store.GamesPuzzleService(), processor, solverInstance, integrity,
	)
	if err != nil {
		t.Fatal(err)
	}
	return service
}
