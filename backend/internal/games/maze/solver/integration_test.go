package solver

import (
	"context"
	"testing"
	"time"

	"skill-arena/internal/games/maze/generator"
)

const (
	integrationDerivationKey = "phase-four-integration-derivation-key-material"
	integrationEncryptionKey = "phase-four-integration-encryption-key-material"
)

func TestPuzzleServiceQualifiesGeneratedCorpusWithProductionSolver(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, 7, 29, 14, 0, 0, 0, time.UTC)
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
	if err := service.RegisterVersion(ctx, version); err != nil {
		t.Fatal(err)
	}
	if err := service.RegisterDifficultyProfile(ctx, profile); err != nil {
		t.Fatal(err)
	}
	processor, err := NewQualificationProcessor(
		generator.DefaultGenerationConfig(),
		Config{Version: version.Key.SolverVersion, MaxArrows: 256}, nil,
	)
	if err != nil {
		t.Fatal(err)
	}

	const corpusSize = 32
	seenAssignments := make(map[string]struct{}, corpusSize)
	for index := range corpusSize {
		scopeID := "solver-corpus-" + decimal(index)
		assignment, err := service.Execute(ctx, generator.WorkRequest{
			Prepare: generator.PrepareRequest{
				Mode: "practice", ScopeType: "practice_session", ScopeID: scopeID,
				ParticipantID: "qualification-player", DifficultyID: profile.ID,
				Version: version.Key, IdempotencyKey: scopeID,
			},
			AssignmentMode: "practice", AssignmentType: "practice_session",
			AssignmentID: scopeID, ReusePolicy: generator.ReuseOneUse,
		}, processor)
		if err != nil {
			t.Fatalf("candidate %d: %v", index, err)
		}
		if _, duplicate := seenAssignments[assignment.ID]; duplicate {
			t.Fatalf("candidate %d reused assignment %q", index, assignment.ID)
		}
		seenAssignments[assignment.ID] = struct{}{}
		metadata, err := service.LoadMetadata(ctx, assignment.PuzzleID)
		if err != nil {
			t.Fatal(err)
		}
		if metadata.Status != generator.PuzzleAssigned ||
			!generator.ValidHash(metadata.SolutionHash) ||
			!generator.ValidHash(metadata.ValidationHash) {
			t.Fatalf("candidate %d metadata = %+v", index, metadata)
		}
	}
}

func solverIntegrationVersion(now time.Time) generator.GeneratorVersion {
	key := generator.VersionKey{
		GameID: generator.GameID, GeneratorVersion: 1,
		SeedFormatVersion: 1, RandomStreamVersion: 1,
		PatternCatalogueVersion: 1, PatternSelectionVersion: 1,
		GeometrySchemaVersion: 1, CandidateScoringVersion: 1,
		ConstraintPolicyVersion: 1, SolverVersion: 1,
		ValidatorVersion: 1, AnalyzerVersion: 1,
		DifficultySchemaVersion: 1, CanonicalEncodingVersion: 1,
	}
	return generator.GeneratorVersion{
		Key: key, Status: generator.VersionActive, NewMatchAllowed: true,
		ArtifactDigest:         generator.HashFields("phase-four:artifact", "v1"),
		DeterminismFixtureHash: generator.HashFields("phase-four:fixture", "v1"),
		ReleasedAt:             now, CreatedAt: now,
	}
}

func solverIntegrationProfile(t testing.TB, now time.Time) generator.DifficultyProfile {
	t.Helper()
	profile := generator.DifficultyProfile{
		ID: "phase-four-solver-qualification-v1", GameID: generator.GameID,
		SchemaVersion: 1, Source: "practice",
		ComplexityMin: 0, ComplexityMax: 1_000_000,
		LineCountMin: 8, LineCountMax: 12,
		DependencyDepthMin: 1, DependencyDepthMax: 12,
		BranchingMin: 0, BranchingMax: 12,
		FalseRoutesMin: 0, FalseRoutesMax: 24,
		DensityMinBPS: 1, DensityMaxBPS: 10_000,
		PatternBias:            "balanced",
		ExpectedSolveTimeMinMS: 0, ExpectedSolveTimeMaxMS: 1_000_000,
		VisualComplexityMin: 0, VisualComplexityMax: 100,
		CreatedAt: now,
	}
	hash, err := generator.CanonicalProfileHash(profile)
	if err != nil {
		t.Fatal(err)
	}
	profile.ProfileHash = hash
	return profile
}

func decimal(value int) string {
	if value == 0 {
		return "0"
	}
	buffer := [20]byte{}
	position := len(buffer)
	for value > 0 {
		position--
		buffer[position] = byte('0' + value%10)
		value /= 10
	}
	return string(buffer[position:])
}
