package generator

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

const (
	testDerivationKey = "phase-two-test-derivation-key-material-0001"
	testEncryptionKey = "phase-two-test-encryption-key-material-0002"
)

func testVersion(now time.Time) GeneratorVersion {
	key := VersionKey{
		GameID: GameID, GeneratorVersion: 1, SeedFormatVersion: SeedFormatVersion,
		RandomStreamVersion: RandomStreamVersion, PatternCatalogueVersion: 1,
		PatternSelectionVersion: 1, GeometrySchemaVersion: 1,
		CandidateScoringVersion: 1, ConstraintPolicyVersion: 1,
		SolverVersion: 1, ValidatorVersion: 1, AnalyzerVersion: 1,
		DifficultySchemaVersion: 1, CanonicalEncodingVersion: 1,
	}
	return GeneratorVersion{
		Key: key, Status: VersionActive, NewMatchAllowed: true,
		ArtifactDigest:         HashFields("test:artifact", "v1"),
		DeterminismFixtureHash: HashFields("test:fixture", "v1"),
		ReleasedAt:             now, CreatedAt: now,
	}
}

func testProfile(now time.Time) DifficultyProfile {
	profile := DifficultyProfile{
		ID: "practice-standard-v1", GameID: GameID, SchemaVersion: 1, Source: "practice",
		ComplexityMin: 100, ComplexityMax: 200, LineCountMin: 10, LineCountMax: 20,
		DependencyDepthMin: 3, DependencyDepthMax: 7, BranchingMin: 1, BranchingMax: 4,
		FalseRoutesMin: 0, FalseRoutesMax: 3, DensityMinBPS: 2500, DensityMaxBPS: 6500,
		PatternBias: "balanced", ExpectedSolveTimeMinMS: 30_000,
		ExpectedSolveTimeMaxMS: 120_000, VisualComplexityMin: 1, VisualComplexityMax: 5,
		CreatedAt: now,
	}
	profile.ProfileHash = HashFields("test:profile", profile.ID)
	return profile
}

func TestServicePreparePersistsOnlySealedSeedAndCanRecoverIt(t *testing.T) {
	now := time.Date(2026, 7, 28, 10, 0, 0, 0, time.UTC)
	repository := NewMemoryRepository()
	vault, err := newSeedVault(testDerivationKey, testEncryptionKey, bytes.NewReader(bytes.Repeat([]byte{0x41}, 128)))
	if err != nil {
		t.Fatal(err)
	}
	service, err := NewService(repository, vault)
	if err != nil {
		t.Fatal(err)
	}
	service.now = func() time.Time { return now }
	service.random = bytes.NewReader(bytes.Repeat([]byte{0x42}, 32))
	version := testVersion(now)
	profile := testProfile(now)
	if err := service.RegisterVersion(context.Background(), version); err != nil {
		t.Fatal(err)
	}
	if err := service.RegisterDifficultyProfile(context.Background(), profile); err != nil {
		t.Fatal(err)
	}

	prepared, err := service.Prepare(context.Background(), PrepareRequest{
		Mode: "practice", ScopeType: "practice_session", ScopeID: "session-1",
		ParticipantID: "player-1", DifficultyID: profile.ID, Version: version.Key,
		IdempotencyKey: "prepare-session-1",
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(prepared.Metadata.SeedCiphertext) != 0 || len(prepared.Metadata.SeedNonce) != 0 {
		t.Fatal("public puzzle metadata exposed encrypted seed fields")
	}
	stored, err := repository.GetPuzzle(context.Background(), prepared.Metadata.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(stored.SeedCiphertext) <= seedBytes || len(stored.SeedNonce) == 0 {
		t.Fatal("repository did not persist sealed seed material")
	}
	if bytes.Contains(stored.SeedCiphertext, prepared.Seed.Bytes()) {
		t.Fatal("ciphertext contains plaintext seed")
	}
	recovered, err := service.RevealSeed(context.Background(), prepared.Metadata.ID)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(recovered.Bytes(), prepared.Seed.Bytes()) {
		t.Fatal("recovered seed differs from prepared seed")
	}
	retried, err := service.Prepare(context.Background(), PrepareRequest{
		Mode: "practice", ScopeType: "practice_session", ScopeID: "session-1",
		ParticipantID: "player-1", DifficultyID: profile.ID, Version: version.Key,
		IdempotencyKey: "prepare-session-1",
	})
	if err != nil {
		t.Fatal(err)
	}
	if retried.Metadata.ID != prepared.Metadata.ID || !bytes.Equal(retried.Seed.Bytes(), prepared.Seed.Bytes()) {
		t.Fatal("idempotent preparation retry did not return the original puzzle")
	}
	encoded, err := json.Marshal(stored)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(encoded, prepared.Seed.Bytes()) || strings.Contains(string(encoded), "SeedCiphertext") {
		t.Fatal("serialized metadata exposed secret seed material")
	}
}

func TestSeedVaultRejectsTamperingAndScopeChangesOutput(t *testing.T) {
	random := append(bytes.Repeat([]byte{0x11}, 44), bytes.Repeat([]byte{0x22}, 44)...)
	vault, err := newSeedVault(testDerivationKey, testEncryptionKey, bytes.NewReader(random))
	if err != nil {
		t.Fatal(err)
	}
	base := SeedScope{
		Mode: "pvp", ScopeType: "match", ScopeID: "match-1",
		DifficultyID: "ranked-1", GeneratorVersion: "generator-v1",
	}
	first, sealed, err := vault.Create(base, "aad")
	if err != nil {
		t.Fatal(err)
	}
	changed := base
	changed.ScopeID = "match-2"
	second, _, err := vault.Create(changed, "aad")
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Equal(first.Bytes(), second.Bytes()) {
		t.Fatal("different match scopes produced the same effective seed")
	}
	tampered := sealed
	tampered.Ciphertext = append([]byte(nil), sealed.Ciphertext...)
	tampered.Ciphertext[0] ^= 0xff
	if _, err := vault.Open(tampered, "aad"); err == nil || strings.Contains(err.Error(), string(first.Bytes())) {
		t.Fatalf("tampered seed error = %v", err)
	}
	if _, err := vault.Open(sealed, "different-aad"); err == nil {
		t.Fatal("seed opened with incorrect authenticated metadata")
	}
}

func TestRandomStreamPublishedVectorAndChunkIndependence(t *testing.T) {
	var material SeedMaterial
	for index := range material.value {
		material.value[index] = byte(index)
	}
	one, err := NewRandomStream(material, "pattern-selection")
	if err != nil {
		t.Fatal(err)
	}
	block := make([]byte, 64)
	if _, err := one.Read(block); err != nil {
		t.Fatal(err)
	}
	const expected = "ac59f107482dab871bd9620ddf6b51d9e9d0dbcb15e62133db79db2cc7865cbb3767a9eae04631d6ab2792f1867e265d18e4e1f50846dd29d25e88a41bf4dcf2"
	if encoded := stringHex(block); encoded != expected {
		t.Fatalf("stream vector = %s", encoded)
	}

	two, _ := NewRandomStream(material, "pattern-selection")
	chunked := make([]byte, 64)
	if _, err := two.Read(chunked[:7]); err != nil {
		t.Fatal(err)
	}
	if _, err := two.Read(chunked[7:]); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(block, chunked) {
		t.Fatal("random stream output depends on caller read chunking")
	}
	other, _ := NewRandomStream(material, "arrow-placement")
	otherBlock := make([]byte, 64)
	_, _ = other.Read(otherBlock)
	if bytes.Equal(block, otherBlock) {
		t.Fatal("random stream domains are not separated")
	}
}

func TestMemoryRepositoryConcurrentUniquenessAndRollback(t *testing.T) {
	now := time.Now().UTC()
	repository := NewMemoryRepository()
	version := testVersion(now)
	profile := testProfile(now)
	if err := repository.RegisterVersion(context.Background(), version); err != nil {
		t.Fatal(err)
	}
	if err := repository.SaveDifficultyProfile(context.Background(), profile); err != nil {
		t.Fatal(err)
	}
	for _, id := range []string{"puzzle-a", "puzzle-b"} {
		if err := repository.CreatePuzzle(context.Background(), preparedMetadata(id, HashFields("seed", id), version, profile, now)); err != nil {
			t.Fatal(err)
		}
	}
	sharedHash := HashFields("puzzle", "same-board")
	var wg sync.WaitGroup
	results := make(chan error, 2)
	for index, id := range []string{"puzzle-a", "puzzle-b"} {
		wg.Add(1)
		go func(index int, id string) {
			defer wg.Done()
			final := validFinalization(id, "match-"+intString(index), sharedHash, now)
			_, err := repository.FinalizeAndAssign(context.Background(), final)
			results <- err
		}(index, id)
	}
	wg.Wait()
	close(results)
	successes, duplicates := 0, 0
	for err := range results {
		switch {
		case err == nil:
			successes++
		case errors.Is(err, ErrDuplicatePuzzle):
			duplicates++
		default:
			t.Fatalf("unexpected concurrent claim error: %v", err)
		}
	}
	if successes != 1 || duplicates != 1 {
		t.Fatalf("successes=%d duplicates=%d", successes, duplicates)
	}

	invalid := validFinalization("puzzle-b", "match-invalid", HashFields("puzzle", "other"), now)
	invalid.Assignment.ScopeID = ""
	if _, err := repository.FinalizeAndAssign(context.Background(), invalid); err == nil {
		t.Fatal("invalid finalization unexpectedly succeeded")
	}
	puzzle, err := repository.GetPuzzle(context.Background(), "puzzle-b")
	if err != nil {
		t.Fatal(err)
	}
	if puzzle.Status != PuzzlePreparing && puzzle.PuzzleHash != sharedHash {
		t.Fatalf("failed transaction left partial puzzle metadata: %+v", puzzle)
	}
}

func TestServiceHonorsCancellationAndExactVersionStatus(t *testing.T) {
	now := time.Now().UTC()
	repository := NewMemoryRepository()
	vault, _ := NewSeedVault(testDerivationKey, testEncryptionKey)
	service, _ := NewService(repository, vault)
	version := testVersion(now)
	version.Status = VersionReplayOnly
	version.NewMatchAllowed = false
	profile := testProfile(now)
	_ = repository.RegisterVersion(context.Background(), version)
	_ = repository.SaveDifficultyProfile(context.Background(), profile)
	_, err := service.Prepare(context.Background(), PrepareRequest{
		Mode: "practice", ScopeType: "practice_session", ScopeID: "session",
		ParticipantID: "player", DifficultyID: profile.ID, Version: version.Key,
		IdempotencyKey: "prepare-session",
	})
	if !errors.Is(err, ErrVersionUnavailable) {
		t.Fatalf("Prepare error = %v", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err = service.Prepare(ctx, PrepareRequest{})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("canceled Prepare error = %v", err)
	}
}

type testProcessor struct {
	calls atomic.Int32
}

func (p *testProcessor) Process(ctx context.Context, input ProcessingInput) (ProcessingResult, error) {
	p.calls.Add(1)
	if err := ctx.Err(); err != nil {
		return ProcessingResult{}, err
	}
	return ProcessingResult{
		GenerationHash: HashFields("processor:generation", input.Metadata.ID),
		PuzzleHash:     HashFields("processor:puzzle", input.Metadata.ID),
		ValidationHash: HashFields("processor:validation", input.Metadata.ID),
		SolutionHash:   HashFields("processor:solution", input.Metadata.ID),
		MinimumActions: 12,
		Analysis: DifficultyAnalysis{
			AnalyzerVersion: 1, Accepted: true, Classification: "matched",
			MeasuredFields: []byte(`{"score":120}`),
			AnalysisHash:   HashFields("processor:analysis", input.Metadata.ID),
		},
	}, nil
}

func TestExecuteIsIdempotentAndRunsProcessorOutsideRepositoryTransaction(t *testing.T) {
	now := time.Now().UTC()
	repository := NewMemoryRepository()
	vault, _ := NewSeedVault(testDerivationKey, testEncryptionKey)
	service, _ := NewService(repository, vault)
	version := testVersion(now)
	profile := testProfile(now)
	_ = service.RegisterVersion(context.Background(), version)
	_ = service.RegisterDifficultyProfile(context.Background(), profile)
	processor := &testProcessor{}
	request := WorkRequest{
		Prepare: PrepareRequest{
			Mode: "pvp", ScopeType: "match", ScopeID: "match-1",
			DifficultyID: profile.ID, Version: version.Key, IdempotencyKey: "match-1",
		},
		AssignmentMode: "pvp", AssignmentType: "match",
		AssignmentID: "match-1", ReusePolicy: ReuseOneUse,
	}
	first, err := service.Execute(context.Background(), request, processor)
	if err != nil {
		t.Fatal(err)
	}
	second, err := service.Execute(context.Background(), request, processor)
	if err != nil {
		t.Fatal(err)
	}
	if first != second || processor.calls.Load() != 1 {
		t.Fatalf("first=%+v second=%+v processor calls=%d", first, second, processor.calls.Load())
	}
}

func TestExecuteConcurrentRetryReturnsOneAssignment(t *testing.T) {
	now := time.Now().UTC()
	repository := NewMemoryRepository()
	vault, _ := NewSeedVault(testDerivationKey, testEncryptionKey)
	service, _ := NewService(repository, vault)
	version := testVersion(now)
	profile := testProfile(now)
	_ = service.RegisterVersion(context.Background(), version)
	_ = service.RegisterDifficultyProfile(context.Background(), profile)
	processor := &testProcessor{}
	request := WorkRequest{
		Prepare: PrepareRequest{
			Mode: "pvp", ScopeType: "match", ScopeID: "match-concurrent",
			DifficultyID: profile.ID, Version: version.Key, IdempotencyKey: "match-concurrent",
		},
		AssignmentMode: "pvp", AssignmentType: "match",
		AssignmentID: "match-concurrent", ReusePolicy: ReuseOneUse,
	}
	results := make(chan Assignment, 2)
	errs := make(chan error, 2)
	var wg sync.WaitGroup
	for range 2 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			assignment, err := service.Execute(context.Background(), request, processor)
			results <- assignment
			errs <- err
		}()
	}
	wg.Wait()
	close(results)
	close(errs)
	for err := range errs {
		if err != nil {
			t.Fatalf("concurrent retry error: %v", err)
		}
	}
	var first Assignment
	for result := range results {
		if first.ID == "" {
			first = result
		} else if result != first {
			t.Fatalf("concurrent retries returned different assignments: %+v and %+v", first, result)
		}
	}
}

func BenchmarkServicePrepare(b *testing.B) {
	now := time.Now().UTC()
	repository := NewMemoryRepository()
	vault, _ := NewSeedVault(testDerivationKey, testEncryptionKey)
	service, _ := NewService(repository, vault)
	version := testVersion(now)
	profile := testProfile(now)
	_ = repository.RegisterVersion(context.Background(), version)
	_ = repository.SaveDifficultyProfile(context.Background(), profile)
	b.ResetTimer()
	for index := 0; index < b.N; index++ {
		_, err := service.Prepare(context.Background(), PrepareRequest{
			Mode: "practice", ScopeType: "practice_session", ScopeID: "session-" + intString(index),
			ParticipantID: "player", DifficultyID: profile.ID, Version: version.Key,
			IdempotencyKey: "prepare-" + intString(index),
		})
		if err != nil {
			b.Fatal(err)
		}
	}
}

func preparedMetadata(id, seedHash string, version GeneratorVersion, profile DifficultyProfile, now time.Time) PuzzleMetadata {
	return PuzzleMetadata{
		ID: id, GameID: GameID, Mode: "pvp", Status: PuzzlePreparing,
		Version: version.Key, DifficultyID: profile.ID, DifficultyHash: profile.ProfileHash,
		RequestHash:   HashFields("request", id),
		SeedReference: "seed-ref-" + id, SeedKeyID: "key", SeedHash: seedHash,
		SeedCiphertext: bytes.Repeat([]byte{1}, 48), SeedNonce: bytes.Repeat([]byte{2}, 12),
		CreatedAt: now,
	}
}

func validFinalization(puzzleID, scopeID, puzzleHash string, now time.Time) Finalization {
	return Finalization{
		PuzzleID: puzzleID, GenerationHash: HashFields("generation", puzzleID),
		PuzzleHash: puzzleHash, ValidationHash: HashFields("validation", puzzleID),
		SolutionHash: HashFields("solution", puzzleID), MinimumActions: 10,
		Analysis: DifficultyAnalysis{
			ID: "analysis-" + puzzleID, PuzzleID: puzzleID, AnalyzerVersion: 1,
			Accepted: true, Classification: "matched", MeasuredFields: []byte(`{"score":100}`),
			AnalysisHash: HashFields("analysis", puzzleID), CreatedAt: now,
		},
		Assignment: Assignment{
			ID: "assignment-" + puzzleID, PuzzleID: puzzleID, Mode: "pvp",
			ScopeType: "match", ScopeID: scopeID, ReusePolicy: ReuseOneUse, AssignedAt: now,
		},
	}
}

func stringHex(value []byte) string {
	const alphabet = "0123456789abcdef"
	result := make([]byte, len(value)*2)
	for index, item := range value {
		result[index*2] = alphabet[item>>4]
		result[index*2+1] = alphabet[item&0xf]
	}
	return string(result)
}
