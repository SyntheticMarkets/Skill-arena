package realtime

import (
	"context"
	"errors"
	"os"
	"sort"
	"sync"
	"testing"
	"time"

	"skill-arena/internal/config"
	"skill-arena/internal/db"
	"skill-arena/internal/games/maze"
	"skill-arena/internal/games/maze/solver"
	"skill-arena/internal/models"
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
	store, err := db.NewWithOptions(ctx, db.Options{
		DatabaseURL: databaseURL, RedisURL: redisURL, Environment: "production",
		Storage: config.StorageSettings{
			Provider: "s3-compatible", Endpoint: s3Endpoint, Bucket: s3Bucket,
			AccessKey: s3AccessKey, SecretKey: s3SecretKey, Region: "us-east-1",
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = store.Close(context.Background()) })
	service := NewService(store)

	const matchCount = 100
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

	prepP99 := phase9Quantile(preparation, 99)
	actionP95 := phase9Quantile(actionLatencies, 95)
	actionP99 := phase9Quantile(actionLatencies, 99)
	reconnectP95 := phase9Quantile(reconnectLatencies, 95)
	t.Logf(
		"matches=%d actions=%d preparation_p99=%s action_p95=%s action_p99=%s reconnect_p95=%s",
		len(paired), len(actionLatencies), prepP99, actionP95, actionP99, reconnectP95,
	)
	if prepP99 > 5*time.Second {
		t.Fatalf("match preparation p99 %s exceeds 5s", prepP99)
	}
	if actionP95 > 50*time.Millisecond || actionP99 > 100*time.Millisecond {
		t.Fatalf("action latency p95=%s p99=%s exceeds target", actionP95, actionP99)
	}
	if reconnectP95 > 750*time.Millisecond {
		t.Fatalf("reconnect p95 %s exceeds 750ms", reconnectP95)
	}
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
