package solver

import (
	"context"
	"errors"
	"fmt"
	"sort"

	"skill-arena/internal/games/maze/generator"
)

var (
	ErrDeadlock      = errors.New("puzzle is deadlocked")
	ErrIsolatedArrow = errors.New("puzzle contains an isolated arrow")
	ErrResourceLimit = errors.New("solver resource limit exceeded")
)

type Config struct {
	Version   int
	MaxArrows int
}

type Solver struct {
	config Config
}

func New(config Config) (*Solver, error) {
	if config.Version <= 0 {
		return nil, errors.New("solver version must be positive")
	}
	if config.MaxArrows < 2 || config.MaxArrows > 2048 {
		return nil, errors.New("solver arrow limit must be between 2 and 2048")
	}
	return &Solver{config: config}, nil
}

func NewQualificationProcessor(
	generationConfig generator.GenerationConfig,
	solverConfig Config,
	observer generator.Observer,
) (*generator.ProductionProcessor, error) {
	instance, err := New(solverConfig)
	if err != nil {
		return nil, err
	}
	return generator.NewProductionProcessor(generationConfig, instance, observer)
}

func (s *Solver) Verify(ctx context.Context, input generator.VerificationInput) (generator.Verification, error) {
	if input.SolverVersion != s.config.Version {
		return generator.Verification{}, errors.New("solver version mismatch")
	}
	result, err := s.Solve(ctx, input.Board, input.AllowIsolated)
	if err != nil {
		if errors.Is(err, ErrDeadlock) || errors.Is(err, ErrIsolatedArrow) {
			return generator.Verification{
				SolverVersion: s.config.Version, Classification: "unsolvable",
			}, nil
		}
		return generator.Verification{}, err
	}
	return generator.Verification{
		Accepted: true, SolverVersion: s.config.Version,
		DependencyHash: result.DependencyHash, SolutionHash: result.SolutionHash,
		MinimumActions: len(result.Steps), Classification: result.Classification,
		FinalChecksum: result.FinalChecksum, Metrics: result.Metrics,
	}, nil
}

func (s *Solver) Solve(ctx context.Context, board generator.Board, allowIsolated bool) (Result, error) {
	if err := ctx.Err(); err != nil {
		return Result{}, err
	}
	if err := generator.ValidateGeometry(board); err != nil {
		return Result{}, fmt.Errorf("invalid board geometry: %w", err)
	}
	if len(board.Arrows) > s.config.MaxArrows {
		return Result{}, ErrResourceLimit
	}
	index := newOccupancyIndex(board)
	graph, err := deriveGraph(ctx, board, index)
	if err != nil {
		return Result{}, err
	}
	metrics, err := graph.metrics(ctx)
	if err != nil {
		return Result{}, err
	}
	if metrics.DependencyDepth == 0 {
		return Result{}, ErrDeadlock
	}
	if metrics.IsolatedArrows > 0 && !allowIsolated {
		return Result{}, ErrIsolatedArrow
	}

	removed := make(map[string]bool, len(board.Arrows))
	steps := make([]Step, 0, len(board.Arrows))
	classification := "unique"
	for len(steps) < len(board.Arrows) {
		if err := ctx.Err(); err != nil {
			return Result{}, err
		}
		type openArrow struct {
			arrow     generator.Arrow
			collision Collision
		}
		open := make([]openArrow, 0, len(board.Arrows)-len(steps))
		for _, arrow := range board.Arrows {
			if removed[arrow.ID] {
				continue
			}
			collision, err := index.collision(board, removed, arrow)
			if err != nil {
				return Result{}, err
			}
			if collision.Clear {
				open = append(open, openArrow{arrow: arrow, collision: collision})
			}
		}
		if len(open) == 0 {
			return Result{}, ErrDeadlock
		}
		sort.Slice(open, func(i, j int) bool { return open[i].arrow.ID < open[j].arrow.ID })
		if len(open) > 1 {
			classification = "multiple"
		}
		selected := open[0]
		removed[selected.arrow.ID] = true
		steps = append(steps, Step{
			Sequence: len(steps) + 1, ArrowID: selected.arrow.ID,
			OpenChoices: len(open), EscapeDistance: selected.collision.EscapeDistance,
		})
	}

	dependencyHash := generator.HashBytes(
		"skill-arena:maze-dependency-graph:v1", graph.canonical(),
		[]byte(fmt.Sprintf("%d", s.config.Version)),
	)
	solutionBytes := canonicalSolution(s.config.Version, classification, steps)
	solutionHash := generator.HashBytes(
		"skill-arena:maze-solver:v1", []byte(dependencyHash), solutionBytes,
	)
	finalBytes, err := canonicalFinalState(board, steps)
	if err != nil {
		return Result{}, err
	}
	finalChecksum := generator.HashBytes(
		"skill-arena:maze-solver-final:v1", finalBytes,
		uint64Bytes(uint64(s.config.Version)),
	)
	return Result{
		Accepted: true, Classification: classification, Steps: steps,
		DependencyHash: dependencyHash, SolutionHash: solutionHash,
		FinalChecksum: finalChecksum, Metrics: metrics,
	}, nil
}
