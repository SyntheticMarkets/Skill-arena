package generator

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"reflect"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"
)

type testIndependentVerifier struct {
	mu    sync.Mutex
	calls int
}

type invalidIndependentVerifier struct{}

func (invalidIndependentVerifier) Verify(context.Context, Board, DependencyGraph) (Verification, error) {
	return Verification{Accepted: true, Classification: "unknown", MinimumActions: -1}, nil
}

type candidateObserver struct {
	observations []CandidateObservation
}

func (o *candidateObserver) ObserveCandidate(_ context.Context, observation CandidateObservation) {
	o.observations = append(o.observations, observation)
}

func (v *testIndependentVerifier) Verify(ctx context.Context, board Board, graph DependencyGraph) (Verification, error) {
	if err := ctx.Err(); err != nil {
		return Verification{}, err
	}
	order, ok := testTopologicalOrder(board, graph)
	if !ok {
		return Verification{Accepted: false}, nil
	}
	boardBytes, err := CanonicalBoard(board)
	if err != nil {
		return Verification{}, err
	}
	orderBytes, err := json.Marshal(order)
	if err != nil {
		return Verification{}, err
	}
	v.mu.Lock()
	v.calls++
	v.mu.Unlock()
	return Verification{
		Accepted: true, Classification: "unique", MinimumActions: len(board.Arrows),
		SolutionHash:  HashBytes("test:solution:v1", orderBytes),
		FinalChecksum: HashBytes("test:final:v1", boardBytes, orderBytes),
	}, nil
}

func (v *testIndependentVerifier) callCount() int {
	v.mu.Lock()
	defer v.mu.Unlock()
	return v.calls
}

func testTopologicalOrder(board Board, graph DependencyGraph) ([]string, bool) {
	remaining := make(map[string]map[string]struct{}, len(board.Arrows))
	for _, arrow := range board.Arrows {
		remaining[arrow.ID] = map[string]struct{}{}
		for _, blocker := range graph.Edges[arrow.ID] {
			remaining[arrow.ID][blocker] = struct{}{}
		}
	}
	order := make([]string, 0, len(board.Arrows))
	for len(order) < len(board.Arrows) {
		open := make([]string, 0)
		for id, blockers := range remaining {
			if len(blockers) == 0 {
				open = append(open, id)
			}
		}
		if len(open) == 0 {
			return nil, false
		}
		sort.Strings(open)
		selected := open[0]
		order = append(order, selected)
		delete(remaining, selected)
		for _, blockers := range remaining {
			delete(blockers, selected)
		}
	}
	return order, true
}

func phaseThreeProfile(t testing.TB, now time.Time) DifficultyProfile {
	t.Helper()
	profile := DifficultyProfile{
		ID: "phase-three-practice-v1", GameID: GameID, SchemaVersion: 1, Source: "practice",
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
	hash, err := CanonicalProfileHash(profile)
	if err != nil {
		t.Fatal(err)
	}
	profile.ProfileHash = hash
	return profile
}

func phaseThreeInput(t testing.TB, seedByte byte) ProcessingInput {
	t.Helper()
	now := time.Date(2026, 7, 29, 12, 0, 0, 0, time.UTC)
	version := testVersion(now)
	profile := phaseThreeProfile(t, now)
	var seed SeedMaterial
	for index := range seed.value {
		seed.value[index] = seedByte + byte(index)
	}
	return ProcessingInput{
		Metadata: PuzzleMetadata{
			ID: "puzzle-phase-three", GameID: GameID, Mode: "practice",
			Status: PuzzlePreparing, Version: version.Key,
			DifficultyID: profile.ID, DifficultyHash: profile.ProfileHash,
			SeedHash: HashBytes("test:seed", seed.Bytes()), CreatedAt: now,
		},
		Profile: profile, Seed: seed,
	}
}

func phaseThreeConfig(workers int) GenerationConfig {
	config := DefaultGenerationConfig()
	config.MaximumWorkers = workers
	return config
}

func TestValidateGeometryRejectsMalformedBoards(t *testing.T) {
	valid := Board{
		GeometryVersion: 1, RulesVersion: 1, Columns: 8, Rows: 8,
		Arrows: []Arrow{
			straightArrow(0, Cell{Column: 7, Row: 1}, DirectionRight, 2),
			straightArrow(1, Cell{Column: 4, Row: 4}, DirectionDown, 2),
		},
	}
	if err := ValidateGeometry(valid); err != nil {
		t.Fatalf("valid board rejected: %v", err)
	}
	tests := map[string]Board{
		"overlap": func() Board {
			board := valid.Clone()
			board.Arrows[1] = straightArrow(1, Cell{Column: 7, Row: 1}, DirectionRight, 2)
			return board
		}(),
		"disconnected": func() Board {
			board := valid.Clone()
			board.Arrows[1].Cells[0] = Cell{Column: 1, Row: 1}
			return board
		}(),
		"direction": func() Board {
			board := valid.Clone()
			board.Arrows[1].Direction = DirectionUp
			return board
		}(),
		"canonical id": func() Board {
			board := valid.Clone()
			board.Arrows[1].ID = "custom"
			return board
		}(),
	}
	for name, board := range tests {
		t.Run(name, func(t *testing.T) {
			if err := ValidateGeometry(board); err == nil {
				t.Fatal("malformed board was accepted")
			}
		})
	}
}

func TestDependenciesAreDerivedFromPhysicalEscapeRays(t *testing.T) {
	board := Board{
		GeometryVersion: 1, RulesVersion: 1, Columns: 8, Rows: 8,
		Arrows: []Arrow{
			straightArrow(0, Cell{Column: 7, Row: 2}, DirectionRight, 2),
			straightArrow(1, Cell{Column: 4, Row: 2}, DirectionRight, 2),
			straightArrow(2, Cell{Column: 1, Row: 6}, DirectionDown, 2),
		},
	}
	graph, err := DeriveDependencies(board)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(graph.Edges["a0001"], []string{"a0000"}) {
		t.Fatalf("physical blockers = %v", graph.Edges["a0001"])
	}
	if len(graph.Edges["a0000"]) != 0 || len(graph.Edges["a0002"]) != 0 {
		t.Fatalf("open arrows were incorrectly blocked: %+v", graph.Edges)
	}
	if err := ValidateDependencyGraph(board, graph, true); err != nil {
		t.Fatal(err)
	}
	cyclic := graph
	cyclic.Edges = cloneEdges(graph.Edges)
	cyclic.Dependents = cloneEdges(graph.Dependents)
	cyclic.Edges["a0000"] = []string{"a0001"}
	cyclic.Dependents["a0001"] = append(cyclic.Dependents["a0001"], "a0000")
	if err := ValidateDependencyGraph(board, cyclic, true); err == nil || !strings.Contains(err.Error(), "cycle") {
		t.Fatalf("cyclic graph error = %v", err)
	}
}

func TestPatternSelectionIsDeterministicAndUsesApprovedCatalogue(t *testing.T) {
	input := phaseThreeInput(t, 0x11)
	first, err := NewRandomStream(input.Seed, "pattern-fixture")
	if err != nil {
		t.Fatal(err)
	}
	second, _ := NewRandomStream(input.Seed, "pattern-fixture")
	one, err := SelectPattern(first, input.Profile, 12, 12)
	if err != nil {
		t.Fatal(err)
	}
	two, err := SelectPattern(second, input.Profile, 12, 12)
	if err != nil {
		t.Fatal(err)
	}
	if one != two {
		t.Fatalf("pattern selection is not reproducible: %+v != %+v", one, two)
	}
	approved := map[string]bool{
		"braid": true, "spiral": true, "maze_rows": true, "rings": true,
		"mosaic": true, "piton": true, "diagonal_weave": true, "rays": true,
	}
	if !approved[one.ID] {
		t.Fatalf("unapproved pattern selected: %q", one.ID)
	}
}

func TestProductionProcessorIsReproducibleAcrossWorkerCounts(t *testing.T) {
	input := phaseThreeInput(t, 0x20)
	firstVerifier := &testIndependentVerifier{}
	first, err := NewProductionProcessor(phaseThreeConfig(1), firstVerifier, nil)
	if err != nil {
		t.Fatal(err)
	}
	firstResult, err := first.Generate(context.Background(), input)
	if err != nil {
		t.Fatal(err)
	}
	secondVerifier := &testIndependentVerifier{}
	second, err := NewProductionProcessor(phaseThreeConfig(4), secondVerifier, nil)
	if err != nil {
		t.Fatal(err)
	}
	secondResult, err := second.Generate(context.Background(), input)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(firstResult.Result, secondResult.Result) ||
		firstResult.Candidate.Index != secondResult.Candidate.Index ||
		!reflect.DeepEqual(firstResult.Candidate.Board, secondResult.Candidate.Board) {
		t.Fatalf("worker scheduling changed selection:\nfirst=%+v\nsecond=%+v", firstResult, secondResult)
	}
	if firstVerifier.callCount() != secondVerifier.callCount() {
		t.Fatalf("verifier calls differ: %d != %d", firstVerifier.callCount(), secondVerifier.callCount())
	}
	boardBytes, err := CanonicalBoard(firstResult.Candidate.Board)
	if err != nil {
		t.Fatal(err)
	}
	const expectedBoardVector = "sha256:5c6692370434da0d6af9e927e91f6876c9ed27ddfc5568d6caa4173ddd78f065"
	if got := HashBytes("test:board-vector:v1", boardBytes); got != expectedBoardVector {
		t.Fatalf("canonical board vector = %s", got)
	}
}

func TestProductionProcessorObservesFixedBatchInCandidateOrder(t *testing.T) {
	input := phaseThreeInput(t, 0x25)
	observer := &candidateObserver{}
	processor, err := NewProductionProcessor(phaseThreeConfig(4), &testIndependentVerifier{}, observer)
	if err != nil {
		t.Fatal(err)
	}
	result, err := processor.Generate(context.Background(), input)
	if err != nil {
		t.Fatal(err)
	}
	if len(observer.observations) != phaseThreeConfig(4).CandidateBatch {
		t.Fatalf("observations = %d", len(observer.observations))
	}
	accepted := 0
	for index, observation := range observer.observations {
		if observation.CandidateIndex != index {
			t.Fatalf("observation %d has candidate index %d", index, observation.CandidateIndex)
		}
		if observation.Accepted {
			accepted++
			if observation.RejectionCode != "" {
				t.Fatalf("accepted candidate has rejection code %q", observation.RejectionCode)
			}
		} else if observation.RejectionCode == "" {
			t.Fatal("rejected candidate has no stable rejection code")
		}
	}
	if accepted != result.Report.Accepted || result.Report.Attempted != len(observer.observations) {
		t.Fatalf("observer/report mismatch: accepted=%d report=%+v", accepted, result.Report)
	}
}

func TestProductionProcessorRejectsProfileDriftAndMissingVerifier(t *testing.T) {
	input := phaseThreeInput(t, 0x30)
	if _, err := NewProductionProcessor(phaseThreeConfig(1), nil, nil); err == nil {
		t.Fatal("processor accepted a missing independent verifier")
	}
	processor, err := NewProductionProcessor(phaseThreeConfig(1), &testIndependentVerifier{}, nil)
	if err != nil {
		t.Fatal(err)
	}
	input.Profile.LineCountMax++
	if _, err := processor.Generate(context.Background(), input); err == nil ||
		!strings.Contains(err.Error(), RejectionProfileHash) {
		t.Fatalf("profile drift error = %v", err)
	}
}

func TestProductionProcessorFailsClosedOnInvalidVerifierOutput(t *testing.T) {
	input := phaseThreeInput(t, 0x35)
	config := phaseThreeConfig(1)
	config.CandidateBatch = 1
	config.MaximumWorkers = 1
	processor, err := NewProductionProcessor(config, invalidIndependentVerifier{}, nil)
	if err != nil {
		t.Fatal(err)
	}
	_, err = processor.Generate(context.Background(), input)
	var generationErr *GenerationError
	if !errors.As(err, &generationErr) {
		t.Fatalf("generation error = %v", err)
	}
	if generationErr.Report.RejectionCounts[RejectionVerifierInvalid] != 1 {
		t.Fatalf("rejection report = %+v", generationErr.Report)
	}
}

func TestProductionProcessorHonorsCancellation(t *testing.T) {
	input := phaseThreeInput(t, 0x40)
	processor, err := NewProductionProcessor(phaseThreeConfig(1), &testIndependentVerifier{}, nil)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := processor.Generate(ctx, input); !errors.Is(err, context.Canceled) {
		t.Fatalf("canceled generation error = %v", err)
	}
}

func TestPuzzleServiceExecutesQualifiedGeneratorPipeline(t *testing.T) {
	now := time.Date(2026, 7, 29, 12, 0, 0, 0, time.UTC)
	repository := NewMemoryRepository()
	vault, err := newSeedVault(testDerivationKey, testEncryptionKey, bytes.NewReader(bytes.Repeat([]byte{0x61}, 256)))
	if err != nil {
		t.Fatal(err)
	}
	service, err := NewService(repository, vault)
	if err != nil {
		t.Fatal(err)
	}
	service.now = func() time.Time { return now }
	service.random = bytes.NewReader(bytes.Repeat([]byte{0x62}, 128))
	version := testVersion(now)
	profile := phaseThreeProfile(t, now)
	if err := service.RegisterVersion(context.Background(), version); err != nil {
		t.Fatal(err)
	}
	if err := service.RegisterDifficultyProfile(context.Background(), profile); err != nil {
		t.Fatal(err)
	}
	processor, err := NewProductionProcessor(DefaultGenerationConfig(), &testIndependentVerifier{}, nil)
	if err != nil {
		t.Fatal(err)
	}
	request := WorkRequest{
		Prepare: PrepareRequest{
			Mode: "practice", ScopeType: "practice_session", ScopeID: "phase-three-session",
			ParticipantID: "player-1", DifficultyID: profile.ID, Version: version.Key,
			IdempotencyKey: "phase-three-session",
		},
		AssignmentMode: "practice", AssignmentType: "practice_session",
		AssignmentID: "phase-three-session", ReusePolicy: ReuseOneUse,
	}
	assignment, err := service.Execute(context.Background(), request, processor)
	if err != nil {
		t.Fatal(err)
	}
	puzzle, err := repository.GetPuzzle(context.Background(), assignment.PuzzleID)
	if err != nil {
		t.Fatal(err)
	}
	if puzzle.Status != PuzzleAssigned || !ValidHash(puzzle.PuzzleHash) ||
		!ValidHash(puzzle.GenerationHash) || !ValidHash(puzzle.ValidationHash) {
		t.Fatalf("qualified puzzle metadata is incomplete: %+v", puzzle)
	}
	if len(puzzle.SeedCiphertext) == 0 || len(puzzle.SeedNonce) == 0 {
		t.Fatal("qualified puzzle lost encrypted seed material")
	}
	retried, err := service.Execute(context.Background(), request, processor)
	if err != nil {
		t.Fatal(err)
	}
	if retried != assignment {
		t.Fatalf("idempotent pipeline retry changed assignment: %+v != %+v", retried, assignment)
	}
}

func TestGenerationFailureUsesStableCodesWithoutSeedDisclosure(t *testing.T) {
	input := phaseThreeInput(t, 0x50)
	input.Profile.LineCountMin = 100
	input.Profile.LineCountMax = 100
	hash, err := CanonicalProfileHash(input.Profile)
	if err != nil {
		t.Fatal(err)
	}
	input.Profile.ProfileHash = hash
	input.Metadata.DifficultyHash = hash
	processor, err := NewProductionProcessor(phaseThreeConfig(1), &testIndependentVerifier{}, nil)
	if err != nil {
		t.Fatal(err)
	}
	_, err = processor.Generate(context.Background(), input)
	var generationErr *GenerationError
	if !errors.As(err, &generationErr) {
		t.Fatalf("generation error = %v", err)
	}
	if generationErr.Report.Attempted != phaseThreeConfig(1).CandidateBatch ||
		len(generationErr.Report.RejectionCounts) == 0 {
		t.Fatalf("generation report = %+v", generationErr.Report)
	}
	if strings.Contains(err.Error(), string(input.Seed.Bytes())) {
		t.Fatal("generation error disclosed seed material")
	}
}

func TestGeneratedCandidatesRemainStructurallyValidAcrossSeedCorpus(t *testing.T) {
	for seed := byte(1); seed <= 32; seed++ {
		input := phaseThreeInput(t, seed)
		candidate, code, err := generateCandidate(context.Background(), input, DefaultGenerationConfig(), 0)
		if err != nil {
			t.Fatalf("seed %d: %v", seed, err)
		}
		if code != "" {
			continue
		}
		if err := ValidateGeometry(candidate.Board); err != nil {
			t.Fatalf("seed %d geometry: %v", seed, err)
		}
		derived, err := DeriveDependencies(candidate.Board)
		if err != nil {
			t.Fatalf("seed %d dependencies: %v", seed, err)
		}
		if !reflect.DeepEqual(candidate.Graph, derived) {
			t.Fatalf("seed %d graph is not physically derived", seed)
		}
		if err := ValidateDependencyGraph(candidate.Board, candidate.Graph, false); err != nil {
			t.Fatalf("seed %d graph validation: %v", seed, err)
		}
	}
}

func BenchmarkProductionGenerator(b *testing.B) {
	input := phaseThreeInput(b, 0x71)
	config := DefaultGenerationConfig()
	for index := 0; index < b.N; index++ {
		input.Seed.value[0] = byte(index)
		processor, err := NewProductionProcessor(config, &testIndependentVerifier{}, nil)
		if err != nil {
			b.Fatal(err)
		}
		if _, err := processor.Generate(context.Background(), input); err != nil {
			b.Fatal(err)
		}
	}
}

func cloneEdges(source map[string][]string) map[string][]string {
	result := make(map[string][]string, len(source))
	for id, values := range source {
		result[id] = append([]string(nil), values...)
	}
	return result
}
