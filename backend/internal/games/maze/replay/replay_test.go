package replay

import (
	"context"
	"errors"
	"reflect"
	"sync"
	"testing"
	"time"

	"skill-arena/internal/games/interfaces"
	"skill-arena/internal/games/maze/generator"
	"skill-arena/internal/games/maze/solver"
	"skill-arena/internal/storage"
)

const (
	replayTestDerivationKey = "phase-five-test-derivation-key-material-0001"
	replayTestEncryptionKey = "phase-five-test-encryption-key-material-0002"
)

type fixture struct {
	service   *Service
	puzzles   *generator.Service
	processor *generator.ProductionProcessor
	solver    *solver.Solver
	puzzleID  string
	qualified generator.QualifiedCandidate
	genesis   Genesis
}

type testIntegrity struct{}

func (testIntegrity) SignReplayIntegrity(
	_ context.Context,
	request interfaces.ReplayIntegrityRequest,
) (interfaces.ReplayIntegrityProof, error) {
	return interfaces.ReplayIntegrityProof{
		Algorithm: "test-sha256-v1", KeyID: "test-key-v1",
		Signature: generator.HashFields(
			"test:replay-proof", request.MatchID, request.GameID,
			request.ReplayHash, request.EventRootHash, intString(request.EventCount),
		),
	}, nil
}

func (testIntegrity) VerifyReplayIntegrity(
	ctx context.Context,
	request interfaces.ReplayIntegrityRequest,
	proof interfaces.ReplayIntegrityProof,
) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	expected, _ := (testIntegrity{}).SignReplayIntegrity(ctx, request)
	if proof != expected {
		return errors.New("test replay proof is invalid")
	}
	return nil
}

func newFixture(t testing.TB) fixture {
	return newFixtureWithReplayVersion(t, ReplayVersionLegacy)
}

func newFixtureWithReplayVersion(t testing.TB, replayVersion int) fixture {
	t.Helper()
	now := time.Date(2026, 7, 29, 10, 0, 0, 0, time.UTC)
	repository := generator.NewMemoryRepository()
	vault, err := generator.NewSeedVault(replayTestDerivationKey, replayTestEncryptionKey)
	if err != nil {
		t.Fatal(err)
	}
	puzzles, err := generator.NewService(repository, vault)
	if err != nil {
		t.Fatal(err)
	}
	version := generator.GeneratorVersion{
		Key: generator.VersionKey{
			GameID: generator.GameID, GeneratorVersion: 1,
			SeedFormatVersion: 1, RandomStreamVersion: 1,
			PatternCatalogueVersion: 1, PatternSelectionVersion: 1,
			GeometrySchemaVersion: 1, CandidateScoringVersion: 1,
			ConstraintPolicyVersion: 1, SolverVersion: 1, ValidatorVersion: 1,
			AnalyzerVersion: 1, DifficultySchemaVersion: 1,
			CanonicalEncodingVersion: 1,
		},
		Status: "active", NewMatchAllowed: true,
		ArtifactDigest:         generator.HashFields("test:artifact", "phase-five"),
		DeterminismFixtureHash: generator.HashFields("test:fixture", "phase-five"),
		ReleasedAt:             now, CreatedAt: now,
	}
	profile := generator.DifficultyProfile{
		ID: "phase-five-practice-v1", GameID: generator.GameID,
		SchemaVersion: 1, Source: "practice",
		ComplexityMin: 0, ComplexityMax: 1_000_000,
		LineCountMin: 8, LineCountMax: 12,
		DependencyDepthMin: 1, DependencyDepthMax: 12,
		BranchingMin: 0, BranchingMax: 12,
		FalseRoutesMin: 0, FalseRoutesMax: 24,
		DensityMinBPS: 1, DensityMaxBPS: 10_000,
		PatternBias: "balanced", ExpectedSolveTimeMinMS: 0,
		ExpectedSolveTimeMaxMS: 1_000_000,
		VisualComplexityMin:    0, VisualComplexityMax: 100, CreatedAt: now,
	}
	profile.ProfileHash, err = generator.CanonicalProfileHash(profile)
	if err != nil {
		t.Fatal(err)
	}
	if err := puzzles.RegisterVersion(context.Background(), version); err != nil {
		t.Fatal(err)
	}
	if err := puzzles.RegisterDifficultyProfile(context.Background(), profile); err != nil {
		t.Fatal(err)
	}
	solverInstance, err := solver.New(solver.Config{Version: 1, MaxArrows: 256})
	if err != nil {
		t.Fatal(err)
	}
	processor, err := generator.NewProductionProcessor(
		generator.DefaultGenerationConfig(), solverInstance, nil,
	)
	if err != nil {
		t.Fatal(err)
	}
	assignment, err := puzzles.Execute(context.Background(), generator.WorkRequest{
		Prepare: generator.PrepareRequest{
			Mode: "practice", ScopeType: "practice_session", ScopeID: "phase-five-session",
			ParticipantID: "player-1", DifficultyID: profile.ID, Version: version.Key,
			IdempotencyKey: "phase-five-prepare",
		},
		AssignmentMode: "practice", AssignmentType: "practice_session",
		AssignmentID: "phase-five-session", ReusePolicy: "one_use",
	}, processor)
	if err != nil {
		t.Fatal(err)
	}
	input, err := puzzles.LoadReconstructionInput(context.Background(), assignment.PuzzleID)
	if err != nil {
		t.Fatal(err)
	}
	qualified, err := processor.Generate(context.Background(), input.ProcessingInput)
	if err != nil {
		t.Fatal(err)
	}
	service, err := NewService(puzzles, processor, solverInstance, testIntegrity{})
	if err != nil {
		t.Fatal(err)
	}
	versions := Versions{
		GameVersion: "1.0.0", ProtocolVersion: 1, ReplayVersion: replayVersion,
		RendererVersion: 1, StateSchemaVersion: 1, Generator: version.Key,
	}
	genesis, err := service.BuildGenesis(context.Background(), GenesisRequest{
		ReplayID: "replay-phase-five", MatchID: "match-phase-five",
		PuzzleID: assignment.PuzzleID, Versions: versions,
		CreatedAtUnixMS: now.UnixMilli(),
	})
	if err != nil {
		t.Fatal(err)
	}
	return fixture{
		service: service, puzzles: puzzles, processor: processor, solver: solverInstance,
		puzzleID: assignment.PuzzleID, qualified: qualified, genesis: genesis,
	}
}

func (f fixture) completedArtifact(t testing.TB) Artifact {
	t.Helper()
	solution, err := f.solver.Solve(context.Background(), f.qualified.Candidate.Board, false)
	if err != nil {
		t.Fatal(err)
	}
	drafts := make([]EventDraft, len(solution.Steps))
	for index, step := range solution.Steps {
		drafts[index] = EventDraft{
			Sequence: uint64(index + 1), ParticipantID: "player-1",
			OffsetMS: int64((index + 1) * 25), Kind: EventArrowAccepted,
			ArrowID: step.ArrowID, Code: "ACTION_ACCEPTED",
		}
	}
	artifact, err := f.service.Seal(context.Background(), SealRequest{
		Genesis: f.genesis, ParticipantIDs: []string{"player-1"}, Events: drafts,
		Outcome:         Outcome{Status: "completed", WinnerIDs: []string{"player-1"}},
		StartedAtUnixMS: f.genesis.CreatedAtUnixMS,
		EndedAtUnixMS:   f.genesis.CreatedAtUnixMS + int64(len(drafts)*25),
	})
	if err != nil {
		t.Fatal(err)
	}
	return artifact
}

func TestReplayReconstructionIsDeterministicAndTamperEvident(t *testing.T) {
	fixture := newFixture(t)
	artifact := fixture.completedArtifact(t)
	if artifact.Genesis.Versions.ReplayVersion != ReplayVersionLegacy ||
		artifact.Participants[0].CurrentCombo != 0 ||
		artifact.Participants[0].MaximumCombo != 0 {
		t.Fatal("legacy replay contract was silently reinterpreted")
	}
	for range 3 {
		report, err := fixture.service.Verify(t.Context(), artifact)
		if err != nil || !report.Verified || report.Status != StatusVerified {
			t.Fatalf("verification report=%+v error=%v", report, err)
		}
	}
	encoded, err := Marshal(artifact)
	if err != nil {
		t.Fatal(err)
	}
	decoded, err := Unmarshal(encoded)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(decoded, artifact) {
		t.Fatal("replay serialization changed the signed artifact")
	}

	tampered := artifact
	tampered.Events = append([]Event(nil), artifact.Events...)
	tampered.Events[0].ArrowID = "tampered"
	assertUnderReview(t, fixture.service, tampered)
	tampered = artifact
	tampered.EventRootHash = generator.HashFields("tampered", "events")
	assertUnderReview(t, fixture.service, tampered)
	tampered = artifact
	tampered.Outcome.WinnerIDs = []string{"unknown"}
	assertUnderReview(t, fixture.service, tampered)
	tampered = artifact
	tampered.ReplayHash = generator.HashFields("tampered", "replay")
	assertUnderReview(t, fixture.service, tampered)
	tampered = artifact
	tampered.Proof.Signature = generator.HashFields("tampered", "proof")
	assertUnderReview(t, fixture.service, tampered)
}

func TestReplaySupportsNoEventCanceledArtifactAndConcurrentVerification(t *testing.T) {
	fixture := newFixture(t)
	artifact, err := fixture.service.Seal(t.Context(), SealRequest{
		Genesis: fixture.genesis, ParticipantIDs: []string{"player-1"},
		Outcome:         Outcome{Status: "canceled", Reason: "participant disconnected"},
		StartedAtUnixMS: fixture.genesis.CreatedAtUnixMS,
		EndedAtUnixMS:   fixture.genesis.CreatedAtUnixMS + 100,
	})
	if err != nil {
		t.Fatal(err)
	}
	const workers = 32
	var wait sync.WaitGroup
	errs := make(chan error, workers)
	for range workers {
		wait.Add(1)
		go func() {
			defer wait.Done()
			report, verifyErr := fixture.service.Verify(context.Background(), artifact)
			if verifyErr != nil {
				errs <- verifyErr
			} else if !report.Verified {
				errs <- errors.New("concurrent replay verification failed")
			}
		}()
	}
	wait.Wait()
	close(errs)
	for err := range errs {
		t.Error(err)
	}
}

func TestReplayObjectRepositoryRoundTrip(t *testing.T) {
	fixture := newFixture(t)
	artifact := fixture.completedArtifact(t)
	repository, err := NewObjectRepository(storage.LocalStore{Root: t.TempDir()}, "")
	if err != nil {
		t.Fatal(err)
	}
	if err := repository.Save(t.Context(), artifact); err != nil {
		t.Fatal(err)
	}
	loaded, err := repository.Load(t.Context(), artifact.Genesis.ReplayID)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(loaded, artifact) {
		t.Fatal("object storage replay round trip changed artifact")
	}
	if _, err := repository.Load(t.Context(), "../escape"); err == nil {
		t.Fatal("unsafe replay object key was accepted")
	}
}

func TestReplayRejectsVersionMismatchInvalidActionsAndUnknownJSON(t *testing.T) {
	fixture := newFixture(t)
	request := GenesisRequest{
		ReplayID: "wrong-version", MatchID: "match", PuzzleID: fixture.puzzleID,
		Versions: fixture.genesis.Versions, CreatedAtUnixMS: fixture.genesis.CreatedAtUnixMS,
	}
	request.Versions.Generator.SolverVersion++
	if _, err := fixture.service.BuildGenesis(t.Context(), request); err == nil {
		t.Fatal("mismatched generator version was accepted")
	}
	request.Versions = fixture.genesis.Versions
	request.Versions.ReplayVersion = 3
	if _, err := fixture.service.BuildGenesis(t.Context(), request); err == nil {
		t.Fatal("unsupported replay version was accepted")
	}
	if _, err := fixture.service.Seal(t.Context(), SealRequest{
		Genesis: fixture.genesis, ParticipantIDs: []string{"player-1"},
		Events: []EventDraft{{
			Sequence: 1, ParticipantID: "player-1", Kind: EventArrowAccepted,
			Code: "ACTION_ACCEPTED", ArrowID: "unknown",
		}},
		Outcome:         Outcome{Status: "canceled"},
		StartedAtUnixMS: fixture.genesis.CreatedAtUnixMS,
		EndedAtUnixMS:   fixture.genesis.CreatedAtUnixMS + 1,
	}); !errors.Is(err, solver.ErrActionMismatch) {
		t.Fatalf("invalid replay action error = %v", err)
	}
	artifact := fixture.completedArtifact(t)
	encoded, _ := Marshal(artifact)
	encoded = append(encoded[:len(encoded)-1], []byte(`,"unknown":true}`)...)
	if _, err := Unmarshal(encoded); err == nil {
		t.Fatal("unknown replay JSON field was accepted")
	}
}

func BenchmarkReplayVerification(b *testing.B) {
	fixture := newFixture(b)
	artifact := fixture.completedArtifact(b)
	b.ResetTimer()
	for range b.N {
		report, err := fixture.service.Verify(context.Background(), artifact)
		if err != nil || !report.Verified {
			b.Fatal(err)
		}
	}
}

func FuzzReplayDecoder(f *testing.F) {
	f.Add([]byte(`{}`))
	f.Add([]byte(`null`))
	f.Add([]byte(`{"genesis":`))
	f.Fuzz(func(t *testing.T, data []byte) {
		_, _ = Unmarshal(data)
	})
}

func assertUnderReview(t testing.TB, service *Service, artifact Artifact) {
	t.Helper()
	report, err := service.Verify(context.Background(), artifact)
	if err != nil {
		t.Fatal(err)
	}
	if report.Verified || report.Status != StatusUnderReview || len(report.Issues) == 0 {
		t.Fatalf("tampered replay report = %+v", report)
	}
}

func intString(value int) string {
	if value == 0 {
		return "0"
	}
	var buffer [20]byte
	index := len(buffer)
	for value > 0 {
		index--
		buffer[index] = byte('0' + value%10)
		value /= 10
	}
	return string(buffer[index:])
}
