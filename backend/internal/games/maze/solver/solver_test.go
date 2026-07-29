package solver

import (
	"context"
	"errors"
	"reflect"
	"sync"
	"testing"

	"skill-arena/internal/games/maze/generator"
)

func testSolver(t testing.TB) *Solver {
	t.Helper()
	instance, err := New(Config{Version: 1, MaxArrows: 256})
	if err != nil {
		t.Fatal(err)
	}
	return instance
}

func arrow(id string, direction generator.Direction, cells ...generator.Cell) generator.Arrow {
	return generator.Arrow{ID: id, Direction: direction, Cells: cells}
}

func uniqueBoard() generator.Board {
	return generator.Board{
		GeometryVersion: 1, RulesVersion: 1, Columns: 8, Rows: 8,
		Arrows: []generator.Arrow{
			arrow("a0000", generator.DirectionRight,
				generator.Cell{Column: 6, Row: 2}, generator.Cell{Column: 7, Row: 2}),
			arrow("a0001", generator.DirectionRight,
				generator.Cell{Column: 3, Row: 2}, generator.Cell{Column: 4, Row: 2}),
		},
	}
}

func multipleBoard() generator.Board {
	return generator.Board{
		GeometryVersion: 1, RulesVersion: 1, Columns: 8, Rows: 8,
		Arrows: []generator.Arrow{
			arrow("a0000", generator.DirectionRight,
				generator.Cell{Column: 6, Row: 2}, generator.Cell{Column: 7, Row: 2}),
			arrow("a0001", generator.DirectionRight,
				generator.Cell{Column: 3, Row: 2}, generator.Cell{Column: 4, Row: 2}),
			arrow("a0002", generator.DirectionLeft,
				generator.Cell{Column: 1, Row: 5}, generator.Cell{Column: 0, Row: 5}),
			arrow("a0003", generator.DirectionLeft,
				generator.Cell{Column: 4, Row: 5}, generator.Cell{Column: 3, Row: 5}),
		},
	}
}

func deadlockedBoard() generator.Board {
	return generator.Board{
		GeometryVersion: 1, RulesVersion: 1, Columns: 8, Rows: 8,
		Arrows: []generator.Arrow{
			arrow("a0000", generator.DirectionRight,
				generator.Cell{Column: 2, Row: 2}, generator.Cell{Column: 3, Row: 2}),
			arrow("a0001", generator.DirectionLeft,
				generator.Cell{Column: 6, Row: 2}, generator.Cell{Column: 5, Row: 2}),
		},
	}
}

func TestCollisionReportsNearestBlockerAndCompleteEscapeDistance(t *testing.T) {
	board := uniqueBoard()
	index := newOccupancyIndex(board)
	blocked, err := index.collision(board, map[string]bool{}, board.Arrows[1])
	if err != nil {
		t.Fatal(err)
	}
	if blocked.Clear || blocked.BlockerID != "a0000" ||
		blocked.CollisionCell != (generator.Cell{Column: 6, Row: 2}) ||
		blocked.Distance != 2 {
		t.Fatalf("blocked collision = %+v", blocked)
	}
	clear, err := index.collision(board, map[string]bool{}, board.Arrows[0])
	if err != nil {
		t.Fatal(err)
	}
	if !clear.Clear || clear.EscapeDistance != 2 || clear.Distance != 1 {
		t.Fatalf("clear collision = %+v", clear)
	}
	afterRemoval, err := index.collision(board, map[string]bool{"a0000": true}, board.Arrows[1])
	if err != nil {
		t.Fatal(err)
	}
	if !afterRemoval.Clear || afterRemoval.EscapeDistance != 5 {
		t.Fatalf("post-removal collision = %+v", afterRemoval)
	}
}

func TestSolverClassifiesUniqueAndMultipleSolutions(t *testing.T) {
	instance := testSolver(t)
	unique, err := instance.Solve(context.Background(), uniqueBoard(), false)
	if err != nil {
		t.Fatal(err)
	}
	if !unique.Accepted || unique.Classification != "unique" ||
		!reflect.DeepEqual(stepIDs(unique.Steps), []string{"a0000", "a0001"}) {
		t.Fatalf("unique result = %+v", unique)
	}
	if unique.Metrics.InitiallyOpen != 1 || unique.Metrics.DependencyEdges != 1 ||
		unique.Metrics.DependencyDepth != 2 {
		t.Fatalf("unique metrics = %+v", unique.Metrics)
	}

	multiple, err := instance.Solve(context.Background(), multipleBoard(), false)
	if err != nil {
		t.Fatal(err)
	}
	if multiple.Classification != "multiple" ||
		!reflect.DeepEqual(stepIDs(multiple.Steps), []string{"a0000", "a0001", "a0002", "a0003"}) {
		t.Fatalf("multiple result = %+v", multiple)
	}
	if multiple.Steps[0].OpenChoices != 2 {
		t.Fatalf("first open choices = %d", multiple.Steps[0].OpenChoices)
	}
}

func TestSolverRejectsDeadlockMalformedGeometryAndIsolatedCompetition(t *testing.T) {
	instance := testSolver(t)
	if _, err := instance.Solve(context.Background(), deadlockedBoard(), false); !errors.Is(err, ErrDeadlock) {
		t.Fatalf("deadlock error = %v", err)
	}
	malformed := uniqueBoard()
	malformed.Arrows[1].Cells[0] = malformed.Arrows[0].Cells[0]
	if _, err := instance.Solve(context.Background(), malformed, false); err == nil {
		t.Fatal("overlapping board was accepted")
	}
	isolated := generator.Board{
		GeometryVersion: 1, RulesVersion: 1, Columns: 8, Rows: 8,
		Arrows: []generator.Arrow{
			arrow("a0000", generator.DirectionRight,
				generator.Cell{Column: 6, Row: 1}, generator.Cell{Column: 7, Row: 1}),
			arrow("a0001", generator.DirectionLeft,
				generator.Cell{Column: 1, Row: 6}, generator.Cell{Column: 0, Row: 6}),
		},
	}
	if _, err := instance.Solve(context.Background(), isolated, false); !errors.Is(err, ErrIsolatedArrow) {
		t.Fatalf("isolated competition error = %v", err)
	}
	result, err := instance.Solve(context.Background(), isolated, true)
	if err != nil || result.Classification != "multiple" {
		t.Fatalf("tutorial isolated result=%+v error=%v", result, err)
	}
}

func TestVerifyFailsClosedForVersionDeadlockAndResourceLimit(t *testing.T) {
	instance := testSolver(t)
	if _, err := instance.Verify(context.Background(), generator.VerificationInput{
		Board: uniqueBoard(), SolverVersion: 2,
	}); err == nil {
		t.Fatal("solver accepted a mismatched version")
	}
	verification, err := instance.Verify(context.Background(), generator.VerificationInput{
		Board: deadlockedBoard(), SolverVersion: 1,
	})
	if err != nil {
		t.Fatal(err)
	}
	if verification.Accepted || verification.Classification != "unsolvable" {
		t.Fatalf("deadlock verification = %+v", verification)
	}
	limited, err := New(Config{Version: 1, MaxArrows: 2})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := limited.Solve(context.Background(), multipleBoard(), false); !errors.Is(err, ErrResourceLimit) {
		t.Fatalf("resource limit error = %v", err)
	}
}

func TestSolverPublishedDeterminismVector(t *testing.T) {
	result, err := testSolver(t).Solve(context.Background(), multipleBoard(), false)
	if err != nil {
		t.Fatal(err)
	}
	const dependencyVector = "sha256:48e3fc73088b2ef1642d601bb8cf7fdafe4678a09c98a20b8894a77e8ad5abab"
	const solutionVector = "sha256:a0570d683d8f68f584ea9086a5c5a0e39ad5810866d61fb17966a3554215b059"
	const finalVector = "sha256:b337ed70ea80948d5f343a83ebe30eb7da95ce9514df2d880c293b08adc411b3"
	if result.DependencyHash != dependencyVector {
		t.Fatalf("dependency vector = %s", result.DependencyHash)
	}
	if result.SolutionHash != solutionVector {
		t.Fatalf("solution vector = %s", result.SolutionHash)
	}
	if result.FinalChecksum != finalVector {
		t.Fatalf("final vector = %s", result.FinalChecksum)
	}
}

func TestSolverIsDeterministicUnderConcurrency(t *testing.T) {
	instance := testSolver(t)
	expected, err := instance.Solve(context.Background(), multipleBoard(), false)
	if err != nil {
		t.Fatal(err)
	}
	const workers = 64
	results := make(chan Result, workers)
	errs := make(chan error, workers)
	var wait sync.WaitGroup
	for range workers {
		wait.Add(1)
		go func() {
			defer wait.Done()
			result, solveErr := instance.Solve(context.Background(), multipleBoard(), false)
			results <- result
			errs <- solveErr
		}()
	}
	wait.Wait()
	close(results)
	close(errs)
	for solveErr := range errs {
		if solveErr != nil {
			t.Fatal(solveErr)
		}
	}
	for result := range results {
		if !reflect.DeepEqual(result, expected) {
			t.Fatalf("concurrent result differs:\n%+v\n%+v", expected, result)
		}
	}
}

func TestSolverHonorsCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := testSolver(t).Solve(ctx, multipleBoard(), false); !errors.Is(err, context.Canceled) {
		t.Fatalf("cancellation error = %v", err)
	}
}

func TestAcceptedRemovalNeverCreatesABlocker(t *testing.T) {
	board := multipleBoard()
	result, err := testSolver(t).Solve(context.Background(), board, false)
	if err != nil {
		t.Fatal(err)
	}
	index := newOccupancyIndex(board)
	removed := map[string]bool{}
	for _, step := range result.Steps {
		before := make(map[string]Collision, len(board.Arrows))
		for _, candidate := range board.Arrows {
			if removed[candidate.ID] {
				continue
			}
			before[candidate.ID], err = index.collision(board, removed, candidate)
			if err != nil {
				t.Fatal(err)
			}
		}
		if !before[step.ArrowID].Clear {
			t.Fatalf("canonical step %q was blocked: %+v", step.ArrowID, before[step.ArrowID])
		}
		removed[step.ArrowID] = true
		for _, candidate := range board.Arrows {
			if removed[candidate.ID] || !before[candidate.ID].Clear {
				continue
			}
			after, err := index.collision(board, removed, candidate)
			if err != nil {
				t.Fatal(err)
			}
			if !after.Clear {
				t.Fatalf("removing %q blocked previously open %q", step.ArrowID, candidate.ID)
			}
		}
	}
}

func TestDependencyChainProperty(t *testing.T) {
	instance := testSolver(t)
	for count := 2; count <= 20; count++ {
		board := dependencyChain(count)
		result, err := instance.Solve(context.Background(), board, false)
		if err != nil {
			t.Fatalf("count %d: %v", count, err)
		}
		if result.Classification != "unique" || len(result.Steps) != count ||
			result.Metrics.DependencyDepth != count {
			t.Fatalf("count %d result = %+v", count, result)
		}
		for index, step := range result.Steps {
			expected := "a" + fourDigits(index)
			if step.ArrowID != expected || step.Sequence != index+1 {
				t.Fatalf("count %d step %d = %+v", count, index, step)
			}
		}
	}
}

func FuzzSolverRejectsMalformedGeometry(f *testing.F) {
	f.Add(uint8(0), int8(0), int8(0))
	f.Add(uint8(4), int8(7), int8(2))
	f.Fuzz(func(t *testing.T, direction uint8, column, row int8) {
		board := uniqueBoard()
		board.Arrows[1].Direction = generator.Direction(direction)
		board.Arrows[1].Cells[1] = generator.Cell{Column: int(column), Row: int(row)}
		instance := testSolver(t)
		result, err := instance.Solve(context.Background(), board, true)
		if err == nil {
			if validationErr := generator.ValidateGeometry(board); validationErr != nil {
				t.Fatalf("solver accepted invalid geometry: %v result=%+v", validationErr, result)
			}
			if !result.Accepted || len(result.Steps) != len(board.Arrows) {
				t.Fatalf("solver returned incomplete accepted result: %+v", result)
			}
		}
	})
}

func BenchmarkSolverStandard(b *testing.B) {
	instance := testSolver(b)
	board := dependencyChain(20)
	b.ResetTimer()
	for range b.N {
		if _, err := instance.Solve(context.Background(), board, false); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkSolverConcurrent(b *testing.B) {
	instance := testSolver(b)
	board := dependencyChain(20)
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			if _, err := instance.Solve(context.Background(), board, false); err != nil {
				b.Fatal(err)
			}
		}
	})
}

func stepIDs(steps []Step) []string {
	result := make([]string, len(steps))
	for index, step := range steps {
		result[index] = step.ArrowID
	}
	return result
}

func dependencyChain(count int) generator.Board {
	columns := count*3 + 1
	arrows := make([]generator.Arrow, count)
	for index := 0; index < count; index++ {
		head := columns - 1 - index*3
		arrows[index] = arrow(
			"a"+fourDigits(index), generator.DirectionRight,
			generator.Cell{Column: head - 1, Row: 2},
			generator.Cell{Column: head, Row: 2},
		)
	}
	return generator.Board{
		GeometryVersion: 1, RulesVersion: 1, Columns: columns, Rows: 8, Arrows: arrows,
	}
}

func fourDigits(value int) string {
	return string([]byte{
		byte('0' + value/1000%10),
		byte('0' + value/100%10),
		byte('0' + value/10%10),
		byte('0' + value%10),
	})
}
