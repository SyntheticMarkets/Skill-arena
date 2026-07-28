package db

import (
	"context"
	"errors"
	"os"
	"sync"
	"testing"
	"time"

	"skill-arena/internal/config"
	mazegenerator "skill-arena/internal/games/maze/generator"
)

func TestPostgresGamesPuzzleServicePersistenceAndAtomicClaims(t *testing.T) {
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
	if _, err := store.pg.ExecContext(ctx, `
TRUNCATE game_puzzle_assignments,game_puzzle_uniqueness_claims,
 game_difficulty_analyses,game_puzzles,game_difficulty_profiles,
 game_generator_versions CASCADE`); err != nil {
		t.Fatal(err)
	}
	service := store.GamesPuzzleService()
	if service == nil {
		t.Fatal("PostgreSQL Puzzle Service is not configured")
	}
	now := time.Now().UTC().Truncate(time.Microsecond)
	version := integrationVersion(now)
	profile := integrationProfile(now)
	if err := service.RegisterVersion(ctx, version); err != nil {
		t.Fatal(err)
	}
	if err := service.RegisterDifficultyProfile(ctx, profile); err != nil {
		t.Fatal(err)
	}
	prepared := make([]mazegenerator.PreparedPuzzle, 2)
	for index := range prepared {
		prepared[index], err = service.Prepare(ctx, mazegenerator.PrepareRequest{
			Mode: "pvp", ScopeType: "match", ScopeID: "generation-" + string(rune('a'+index)),
			DifficultyID: profile.ID, Version: version.Key,
			IdempotencyKey: "integration-prepare-" + string(rune('a'+index)),
		})
		if err != nil {
			t.Fatal(err)
		}
	}
	commonPuzzleHash := mazegenerator.HashFields("integration:puzzle", "same")
	results := make(chan error, 2)
	var wg sync.WaitGroup
	for index := range prepared {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()
			puzzleID := prepared[index].Metadata.ID
			_, claimErr := service.FinalizeAndAssign(ctx, integrationFinalization(
				puzzleID, "match-"+string(rune('a'+index)), commonPuzzleHash, now,
			))
			results <- claimErr
		}(index)
	}
	wg.Wait()
	close(results)
	successes, duplicates := 0, 0
	for claimErr := range results {
		if claimErr == nil {
			successes++
		} else if errors.Is(claimErr, mazegenerator.ErrDuplicatePuzzle) {
			duplicates++
		} else {
			t.Fatalf("unexpected claim error: %v", claimErr)
		}
	}
	if successes != 1 || duplicates != 1 {
		t.Fatalf("successes=%d duplicates=%d", successes, duplicates)
	}
	var claims, assignments, analyses int
	if err := store.pg.QueryRowContext(ctx, `SELECT count(*) FROM game_puzzle_uniqueness_claims`).Scan(&claims); err != nil {
		t.Fatal(err)
	}
	if err := store.pg.QueryRowContext(ctx, `SELECT count(*) FROM game_puzzle_assignments`).Scan(&assignments); err != nil {
		t.Fatal(err)
	}
	if err := store.pg.QueryRowContext(ctx, `SELECT count(*) FROM game_difficulty_analyses`).Scan(&analyses); err != nil {
		t.Fatal(err)
	}
	if claims != 1 || assignments != 1 || analyses != 1 {
		t.Fatalf("atomic records: claims=%d assignments=%d analyses=%d", claims, assignments, analyses)
	}
	loaded, err := service.LoadMetadata(ctx, prepared[0].Metadata.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(loaded.SeedCiphertext) != 0 || len(loaded.SeedNonce) != 0 {
		t.Fatal("loaded public metadata exposed sealed seed material")
	}
}

func integrationVersion(now time.Time) mazegenerator.GeneratorVersion {
	key := mazegenerator.VersionKey{
		GameID: mazegenerator.GameID, GeneratorVersion: 1, SeedFormatVersion: 1,
		RandomStreamVersion: 1, PatternCatalogueVersion: 1, PatternSelectionVersion: 1,
		GeometrySchemaVersion: 1, CandidateScoringVersion: 1, ConstraintPolicyVersion: 1,
		SolverVersion: 1, ValidatorVersion: 1, AnalyzerVersion: 1,
		DifficultySchemaVersion: 1, CanonicalEncodingVersion: 1,
	}
	return mazegenerator.GeneratorVersion{
		Key: key, Status: mazegenerator.VersionActive, NewMatchAllowed: true,
		ArtifactDigest:         mazegenerator.HashFields("integration:artifact", "v1"),
		DeterminismFixtureHash: mazegenerator.HashFields("integration:fixture", "v1"),
		ReleasedAt:             now, CreatedAt: now,
	}
}

func integrationProfile(now time.Time) mazegenerator.DifficultyProfile {
	profile := mazegenerator.DifficultyProfile{
		ID: "integration-profile-v1", GameID: mazegenerator.GameID, SchemaVersion: 1,
		Source: "ranked", ComplexityMin: 100, ComplexityMax: 200,
		LineCountMin: 10, LineCountMax: 20, DependencyDepthMin: 2,
		DependencyDepthMax: 6, BranchingMin: 1, BranchingMax: 4,
		FalseRoutesMin: 0, FalseRoutesMax: 2, DensityMinBPS: 2000,
		DensityMaxBPS: 7000, PatternBias: "balanced",
		ExpectedSolveTimeMinMS: 30_000, ExpectedSolveTimeMaxMS: 120_000,
		VisualComplexityMin: 1, VisualComplexityMax: 5, CreatedAt: now,
	}
	profile.ProfileHash = mazegenerator.HashFields("integration:profile", profile.ID)
	return profile
}

func integrationFinalization(puzzleID, scopeID, puzzleHash string, now time.Time) mazegenerator.Finalization {
	return mazegenerator.Finalization{
		PuzzleID:       puzzleID,
		GenerationHash: mazegenerator.HashFields("integration:generation", puzzleID),
		PuzzleHash:     puzzleHash,
		ValidationHash: mazegenerator.HashFields("integration:validation", puzzleID),
		SolutionHash:   mazegenerator.HashFields("integration:solution", puzzleID),
		MinimumActions: 10,
		Analysis: mazegenerator.DifficultyAnalysis{
			ID: "analysis-" + puzzleID, PuzzleID: puzzleID, AnalyzerVersion: 1,
			Accepted: true, Classification: "matched", MeasuredFields: []byte(`{"score":100}`),
			AnalysisHash: mazegenerator.HashFields("integration:analysis", puzzleID),
			CreatedAt:    now,
		},
		Assignment: mazegenerator.Assignment{
			ID: "assignment-" + puzzleID, PuzzleID: puzzleID, Mode: "pvp",
			ScopeType: "match", ScopeID: scopeID,
			ReusePolicy: mazegenerator.ReuseOneUse, AssignedAt: now,
		},
	}
}
