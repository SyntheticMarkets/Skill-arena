package realtime

import (
	"context"
	"errors"
	"os"
	"sort"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"skill-arena/internal/config"
	"skill-arena/internal/db"
	"skill-arena/internal/games/maze"
	"skill-arena/internal/games/maze/solver"
	"skill-arena/internal/models"
	"skill-arena/internal/observability"
)

func TestPhase9OneHundredLiveMazeMatches(t *testing.T) {
	databaseURL := os.Getenv("SKILL_ARENA_PHASE9_LOAD_POSTGRES_URL")
	redisURL := os.Getenv("SKILL_ARENA_PHASE9_LOAD_REDIS_URL")
	s3Endpoint := os.Getenv("SKILL_ARENA_PHASE9_LOAD_S3_ENDPOINT")
	s3Bucket := os.Getenv("SKILL_ARENA_PHASE9_LOAD_S3_BUCKET")
	s3AccessKey := os.Getenv("SKILL_ARENA_PHASE9_LOAD_S3_ACCESS_KEY")
	s3SecretKey := os.Getenv("SKILL_ARENA_PHASE9_LOAD_S3_SECRET_KEY")
	if databaseURL == "" || redisURL == "" || s3Endpoint == "" || s3Bucket == "" ||
		s3AccessKey == "" || s3SecretKey == "" {
		t.Skip("Phase 9 load infrastructure is not configured")
	}

	ctx, cancel := context.WithTimeout(t.Context(), 20*time.Minute)
	defer cancel()
	timings := &phase9TimingRecorder{values: map[string][]time.Duration{}}
	ctx = observability.WithTimingRecorder(ctx, timings)
	const matchCount = 100
	databasePoolSize := matchCount
	if configured := os.Getenv("SKILL_ARENA_PHASE9_DATABASE_POOL_SIZE"); configured != "" {
		parsed, parseErr := strconv.Atoi(configured)
		if parseErr != nil || parsed < 1 || parsed > matchCount {
			t.Fatalf("invalid SKILL_ARENA_PHASE9_DATABASE_POOL_SIZE %q", configured)
		}
		databasePoolSize = parsed
	}
	batchWorkers := phase9OptionalInt(t, "SKILL_ARENA_PHASE9_ACTION_BATCH_WORKERS", 0, 16)
	batchSize := phase9OptionalInt(t, "SKILL_ARENA_PHASE9_ACTION_BATCH_SIZE", 0, 128)
	batchWindow := time.Duration(
		phase9OptionalInt(t, "SKILL_ARENA_PHASE9_ACTION_BATCH_WINDOW_US", 0, 10000),
	) * time.Microsecond
	store, err := db.NewWithOptions(ctx, db.Options{
		DatabaseURL: databaseURL, RedisURL: redisURL, Environment: "production",
		DatabaseMaxOpenConns:   databasePoolSize,
		DatabaseMaxIdleConns:   databasePoolSize,
		GameActionBatchWorkers: batchWorkers,
		GameActionBatchSize:    batchSize,
		GameActionBatchWindow:  batchWindow,
		Storage: config.StorageSettings{
			Provider: "s3-compatible", Endpoint: s3Endpoint, Bucket: s3Bucket,
			AccessKey: s3AccessKey, SecretKey: s3SecretKey, Region: "us-east-1",
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = store.Close(context.Background()) })
	postgresDiagnostics, err := startPhase9PostgresDiagnostics(ctx, databaseURL)
	if err != nil {
		t.Fatal(err)
	}
	postgresDiagnosticsStopped := false
	defer func() {
		if !postgresDiagnosticsStopped {
			_, _ = postgresDiagnostics.Stop(context.Background())
		}
	}()
	poolStop := make(chan struct{})
	poolStopped := make(chan struct{})
	var poolMaxInUse atomic.Int64
	go func() {
		defer close(poolStopped)
		ticker := time.NewTicker(2 * time.Millisecond)
		defer ticker.Stop()
		for {
			select {
			case <-poolStop:
				return
			case <-ticker.C:
				if stats, ok := store.DatabaseStats(); ok {
					for {
						previous := poolMaxInUse.Load()
						if int64(stats.InUse) <= previous ||
							poolMaxInUse.CompareAndSwap(previous, int64(stats.InUse)) {
							break
						}
					}
				}
			}
		}
	}()
	defer func() {
		select {
		case <-poolStopped:
		default:
			close(poolStop)
			<-poolStopped
		}
	}()
	service := NewService(store)

	firstPlayers := make([]*models.User, matchCount)
	secondPlayers := make([]*models.User, matchCount)
	for index := range matchCount {
		firstPlayers[index] = createRealtimeUser(t, store, phase9LoadEmail("first", index))
		secondPlayers[index] = createRealtimeUser(t, store, phase9LoadEmail("second", index))
	}
	request := QueueRequest{
		GameID: maze.ModuleID, Mode: "pvp", WalletCategory: "practice",
		Region: "af-south", Jurisdiction: "ZA", LatencyMS: 20,
	}
	matches := make(chan *models.RealtimeMatch, matchCount)
	errs := make(chan error, matchCount)
	preparation := make([]time.Duration, 0, matchCount)
	var preparationMu sync.Mutex
	var wait sync.WaitGroup
	for index := range matchCount {
		wait.Add(1)
		go func(index int) {
			defer wait.Done()
			pairRequest := request
			pairRequest.Region = "phase9-shard-" + time.Unix(0, int64(index)).UTC().Format("150405.000000000")
			started := time.Now()
			if _, match, queueErr := service.Queue(ctx, firstPlayers[index].ID, pairRequest); queueErr != nil || match != nil {
				if queueErr != nil {
					errs <- queueErr
				} else {
					errs <- errors.New("first player unexpectedly matched")
				}
				return
			}
			_, match, queueErr := service.Queue(ctx, secondPlayers[index].ID, pairRequest)
			elapsed := time.Since(started)
			if queueErr != nil {
				errs <- queueErr
				return
			}
			if match == nil {
				return
			}
			preparationMu.Lock()
			preparation = append(preparation, elapsed)
			preparationMu.Unlock()
			matches <- match
		}(index)
	}
	wait.Wait()
	close(matches)
	close(errs)
	for loadErr := range errs {
		t.Fatal(loadErr)
	}
	paired := make([]*models.RealtimeMatch, 0, matchCount)
	seenPlayers := make(map[string]struct{}, matchCount*2)
	for match := range matches {
		if len(match.Participants) != 2 {
			t.Fatalf("match participants=%d", len(match.Participants))
		}
		for _, participant := range match.Participants {
			if _, duplicate := seenPlayers[participant.UserID]; duplicate {
				t.Fatalf("player %s was paired more than once", participant.UserID)
			}
			seenPlayers[participant.UserID] = struct{}{}
		}
		paired = append(paired, match)
	}
	if len(paired) != matchCount || len(seenPlayers) != matchCount*2 {
		t.Fatalf("matches=%d players=%d", len(paired), len(seenPlayers))
	}
	actionLatencies := make([]time.Duration, 0, matchCount*40)
	ordinaryActionLatencies := make([]time.Duration, 0, matchCount*40)
	completionActionLatencies := make([]time.Duration, 0, matchCount)
	reconnectLatencies := make([]time.Duration, 0, matchCount)
	var latencyMu sync.Mutex
	errs = make(chan error, matchCount)
	for _, match := range paired {
		wait.Add(1)
		go func(match *models.RealtimeMatch) {
			defer wait.Done()
			first := match.Participants[0].UserID
			second := match.Participants[1].UserID
			if _, readyErr := service.Ready(ctx, first, match.ID); readyErr != nil {
				errs <- readyErr
				return
			}
			if _, readyErr := service.Ready(ctx, second, match.ID); readyErr != nil {
				errs <- readyErr
				return
			}
			firstState := loadMazeEngineState(t, store, match.ID, first)
			secondState := loadMazeEngineState(t, store, match.ID, second)
			if firstState.PuzzleHash != secondState.PuzzleHash ||
				firstState.ParticipantID == secondState.ParticipantID {
				errs <- errors.New("PvP participants do not share one puzzle with independent state")
				return
			}
			if disconnectErr := service.Disconnect(ctx, first, "phase9-session", "phase9-connection", match.ID); disconnectErr != nil {
				errs <- disconnectErr
				return
			}
			reconnectStarted := time.Now()
			if _, _, reconnectErr := service.Reconnect(ctx, first, match.ID, 0); reconnectErr != nil {
				errs <- reconnectErr
				return
			}
			latencyMu.Lock()
			reconnectLatencies = append(reconnectLatencies, time.Since(reconnectStarted))
			latencyMu.Unlock()

			instance, solverErr := solver.New(solver.Config{Version: 1, MaxArrows: 2048})
			if solverErr != nil {
				errs <- solverErr
				return
			}
			solution, solverErr := instance.Solve(ctx, firstState.Board, false)
			if solverErr != nil {
				errs <- solverErr
				return
			}
			for index, step := range solution.Steps {
				participants := []string{first, second}
				if index == len(solution.Steps)-1 {
					participants = participants[:1]
				}
				for _, participant := range participants {
					action := mazeAction(
						match.ID,
						"phase9-"+participant+"-"+step.ArrowID,
						step.ArrowID, int64(index+1), int64(index),
					)
					actionStarted := time.Now()
					result, actionErr := service.GameAction(ctx, participant, match.ID, action, 20*time.Millisecond)
					elapsed := time.Since(actionStarted)
					if actionErr != nil {
						errs <- actionErr
						return
					}
					if !result.Receipt.Accepted {
						errs <- errors.New("canonical action was rejected")
						return
					}
					latencyMu.Lock()
					actionLatencies = append(actionLatencies, elapsed)
					if result.Completion == nil {
						ordinaryActionLatencies = append(ordinaryActionLatencies, elapsed)
					} else {
						completionActionLatencies = append(completionActionLatencies, elapsed)
					}
					latencyMu.Unlock()
				}
			}
			completed, matchErr := service.Match(ctx, first, match.ID)
			if matchErr != nil || completed.Status != models.MatchCompleted {
				if matchErr != nil {
					errs <- matchErr
				} else {
					errs <- errors.New("match did not complete")
				}
				return
			}
			if replayErr := service.FinalizeReplay(ctx, match.ID); replayErr != nil {
				errs <- replayErr
				return
			}
			replay, replayErr := service.Replay(ctx, first, match.ID)
			if replayErr != nil || replay.Signature == "" || replay.StorageKey == "" {
				if replayErr != nil {
					errs <- replayErr
				} else {
					errs <- errors.New("replay integrity metadata is incomplete")
				}
			}
		}(match)
	}
	wait.Wait()
	close(errs)
	for loadErr := range errs {
		t.Fatal(loadErr)
	}
	close(poolStop)
	<-poolStopped
	postgresStats, err := postgresDiagnostics.Stop(ctx)
	if err != nil {
		t.Fatal(err)
	}
	postgresDiagnosticsStopped = true

	prepP99 := phase9Quantile(preparation, 99)
	actionP95 := phase9Quantile(actionLatencies, 95)
	actionP99 := phase9Quantile(actionLatencies, 99)
	ordinaryActionP95 := phase9Quantile(ordinaryActionLatencies, 95)
	ordinaryActionP99 := phase9Quantile(ordinaryActionLatencies, 99)
	completionActionP95 := phase9Quantile(completionActionLatencies, 95)
	reconnectP95 := phase9Quantile(reconnectLatencies, 95)
	t.Logf(
		"matches=%d actions=%d preparation_p99=%s action_p95=%s action_p99=%s ordinary_action_p95=%s ordinary_action_p99=%s completion_action_p95=%s reconnect_p95=%s",
		len(paired), len(actionLatencies), prepP99, actionP95, actionP99,
		ordinaryActionP95, ordinaryActionP99, completionActionP95, reconnectP95,
	)
	for _, name := range timings.names() {
		t.Logf(
			"component=%s count=%d p50=%s p95=%s p99=%s",
			name, timings.count(name), timings.quantile(name, 50),
			timings.quantile(name, 95), timings.quantile(name, 99),
		)
	}
	if stats, ok := store.DatabaseStats(); ok {
		t.Logf(
			"db_pool max_open=%d max_in_use=%d wait_count=%d wait_duration=%s idle=%d",
			stats.MaxOpenConnections, poolMaxInUse.Load(), stats.WaitCount,
			stats.WaitDuration, stats.Idle,
		)
	}
	for _, line := range postgresDiagnostics.logLines() {
		t.Log(line)
	}
	t.Logf(
		"pg_stats transactions=%d blocks_read=%d blocks_hit=%d temp_files=%d temp_bytes=%d deadlocks=%d block_read_ms=%.3f block_write_ms=%.3f wal_records=%d wal_fpi=%d wal_bytes=%d wal_writes=%d wal_syncs=%d wal_buffers_full=%d wal_write_ms=%.3f wal_sync_ms=%.3f checkpoints=%d checkpoint_write_ms=%.3f checkpoint_sync_ms=%.3f checkpoint_buffers=%d clean_buffers=%d backend_writes=%d backend_fsyncs=%d io_writes=%d io_write_ms=%.3f io_fsyncs=%d io_fsync_ms=%.3f",
		postgresStats.transactions, postgresStats.blocksRead, postgresStats.blocksHit,
		postgresStats.tempFiles, postgresStats.tempBytes, postgresStats.deadlocks,
		postgresStats.blockReadMS, postgresStats.blockWriteMS,
		postgresStats.walRecords, postgresStats.walFPI, postgresStats.walBytes,
		postgresStats.walWrites, postgresStats.walSyncs, postgresStats.walBuffersFull,
		postgresStats.walWriteMS, postgresStats.walSyncMS,
		postgresStats.checkpoints, postgresStats.checkpointWriteMS,
		postgresStats.checkpointSyncMS, postgresStats.checkpointBuffers,
		postgresStats.cleanBuffers, postgresStats.backendWrites,
		postgresStats.backendFsyncs, postgresStats.ioWrites,
		postgresStats.ioWriteMS, postgresStats.ioFsyncs, postgresStats.ioFsyncMS,
	)
	if prepP99 > 5*time.Second {
		t.Fatalf("match preparation p99 %s exceeds 5s", prepP99)
	}
	reportOnly := strings.EqualFold(
		os.Getenv("SKILL_ARENA_PHASE9_REPORT_LATENCY_ONLY"), "true",
	)
	if actionP95 > 50*time.Millisecond || actionP99 > 100*time.Millisecond {
		if !reportOnly {
			t.Fatalf("action latency p95=%s p99=%s exceeds target", actionP95, actionP99)
		}
		t.Logf("REPORT ONLY: action latency p95=%s p99=%s exceeds target", actionP95, actionP99)
	}
	if reconnectP95 > 250*time.Millisecond {
		if !reportOnly {
			t.Fatalf("reconnect p95 %s exceeds 250ms", reconnectP95)
		}
		t.Logf("REPORT ONLY: reconnect p95 %s exceeds 250ms", reconnectP95)
	}
}

func phase9OptionalInt(t *testing.T, name string, minimum, maximum int) int {
	t.Helper()
	value := os.Getenv(name)
	if value == "" {
		return 0
	}
	parsed, err := strconv.Atoi(value)
	if err != nil || parsed < minimum || parsed > maximum {
		t.Fatalf("invalid %s %q", name, value)
	}
	return parsed
}

type phase9TimingRecorder struct {
	mu     sync.Mutex
	values map[string][]time.Duration
}

func (r *phase9TimingRecorder) ObserveTiming(name string, duration time.Duration) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.values[name] = append(r.values[name], duration)
}

func (r *phase9TimingRecorder) names() []string {
	r.mu.Lock()
	defer r.mu.Unlock()
	names := make([]string, 0, len(r.values))
	for name := range r.values {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func (r *phase9TimingRecorder) count(name string) int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return len(r.values[name])
}

func (r *phase9TimingRecorder) quantile(name string, percentile int) time.Duration {
	r.mu.Lock()
	values := append([]time.Duration(nil), r.values[name]...)
	r.mu.Unlock()
	return phase9Quantile(values, percentile)
}

func phase9LoadEmail(group string, index int) string {
	return "phase9-" + group + "-" + time.Unix(0, int64(index)).UTC().Format("150405.000000000") + "@example.com"
}

func phase9Quantile(values []time.Duration, percentile int) time.Duration {
	if len(values) == 0 {
		return 0
	}
	sorted := append([]time.Duration(nil), values...)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i] < sorted[j] })
	index := (len(sorted)*percentile + 99) / 100
	if index > 0 {
		index--
	}
	return sorted[index]
}
