package solver

import (
	"context"
	"os"
	"sort"
	"strconv"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"skill-arena/internal/games/maze/generator"
)

type qualificationObserver struct {
	attempted atomic.Int64
	accepted  atomic.Int64
	mu        sync.Mutex
	rejected  map[string]int
}

func (o *qualificationObserver) ObserveCandidate(_ context.Context, observation generator.CandidateObservation) {
	o.attempted.Add(1)
	if observation.Accepted {
		o.accepted.Add(1)
		return
	}
	o.mu.Lock()
	o.rejected[observation.RejectionCode]++
	o.mu.Unlock()
}

func TestPhase9LargeQualificationCorpus(t *testing.T) {
	rawSize := os.Getenv("SKILL_ARENA_PHASE9_CORPUS_SIZE")
	if rawSize == "" {
		t.Skip("SKILL_ARENA_PHASE9_CORPUS_SIZE is not configured")
	}
	corpusSize, err := strconv.Atoi(rawSize)
	if err != nil || corpusSize < 100_000 {
		t.Fatalf("SKILL_ARENA_PHASE9_CORPUS_SIZE must be at least 100000, got %q", rawSize)
	}

	ctx := t.Context()
	now := time.Date(2026, 7, 30, 0, 0, 0, 0, time.UTC)
	repository := generator.NewMemoryRepository()
	vault, err := generator.NewSeedVault(integrationDerivationKey, integrationEncryptionKey)
	if err != nil {
		t.Fatal(err)
	}
	service, err := generator.NewService(repository, vault)
	if err != nil {
		t.Fatal(err)
	}
	version := solverIntegrationVersion(now)
	profile := solverIntegrationProfile(t, now)
	profile.ID = "phase-nine-standard-qualification-v1"
	profile.LineCountMin = 12
	profile.LineCountMax = 20
	profile.ProfileHash, err = generator.CanonicalProfileHash(profile)
	if err != nil {
		t.Fatal(err)
	}
	if err := service.RegisterVersion(ctx, version); err != nil {
		t.Fatal(err)
	}
	if err := service.RegisterDifficultyProfile(ctx, profile); err != nil {
		t.Fatal(err)
	}
	observer := &qualificationObserver{rejected: map[string]int{}}
	config := generator.DefaultGenerationConfig()
	processor, err := NewQualificationProcessor(
		config, Config{Version: version.Key.SolverVersion, MaxArrows: 256}, observer,
	)
	if err != nil {
		t.Fatal(err)
	}

	executions := (corpusSize + config.CandidateBatch - 1) / config.CandidateBatch
	durations := make([]time.Duration, 0, executions)
	seenPuzzles := make(map[string]struct{}, executions)
	for index := range executions {
		scopeID := "phase9-qualification-" + decimal(index)
		started := time.Now()
		assignment, executeErr := service.Execute(ctx, generator.WorkRequest{
			Prepare: generator.PrepareRequest{
				Mode: "practice", ScopeType: "practice_session", ScopeID: scopeID,
				ParticipantID: "phase9-qualification-player", DifficultyID: profile.ID,
				Version: version.Key, IdempotencyKey: scopeID,
			},
			AssignmentMode: "practice", AssignmentType: "practice_session",
			AssignmentID: scopeID, ReusePolicy: generator.ReuseOneUse,
		}, processor)
		durations = append(durations, time.Since(started))
		if executeErr != nil {
			t.Fatalf("batch %d: %v", index, executeErr)
		}
		metadata, loadErr := service.LoadMetadata(ctx, assignment.PuzzleID)
		if loadErr != nil {
			t.Fatal(loadErr)
		}
		if _, duplicate := seenPuzzles[metadata.PuzzleHash]; duplicate {
			t.Fatalf("batch %d reused puzzle hash %s", index, metadata.PuzzleHash)
		}
		seenPuzzles[metadata.PuzzleHash] = struct{}{}
		if metadata.Status != generator.PuzzleAssigned ||
			!generator.ValidHash(metadata.PuzzleHash) ||
			!generator.ValidHash(metadata.ValidationHash) ||
			!generator.ValidHash(metadata.SolutionHash) {
			t.Fatalf("batch %d metadata failed qualification: %+v", index, metadata)
		}
	}

	if got := observer.attempted.Load(); got < int64(corpusSize) {
		t.Fatalf("attempted=%d, want at least %d", got, corpusSize)
	}
	if observer.accepted.Load() < int64(executions) {
		t.Fatalf("accepted=%d, need at least one accepted candidate per %d batches", observer.accepted.Load(), executions)
	}
	sort.Slice(durations, func(i, j int) bool { return durations[i] < durations[j] })
	quantile := func(percentile int) time.Duration {
		index := (len(durations)*percentile + 99) / 100
		if index > 0 {
			index--
		}
		return durations[index]
	}
	t.Logf(
		"qualification candidates=%d accepted=%d selected=%d p50=%s p95=%s p99=%s rejections=%v",
		observer.attempted.Load(), observer.accepted.Load(), len(seenPuzzles),
		quantile(50), quantile(95), quantile(99), observer.rejected,
	)
}
