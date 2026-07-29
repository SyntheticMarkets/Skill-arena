package solver

import (
	"context"
	"errors"
	"sort"

	"skill-arena/internal/games/maze/generator"
)

var ErrActionMismatch = errors.New("replay action disagrees with authoritative collision")

type SimulationAction struct {
	ArrowID  string
	Accepted bool
}

type SimulationStep struct {
	ArrowID           string
	Accepted          bool
	BlockerID         string
	CollisionCell     generator.Cell
	CollisionDistance int
	EscapeDistance    int
	RemovedIDs        []string
	Complete          bool
}

type SimulationResult struct {
	Steps      []SimulationStep
	RemovedIDs []string
	Complete   bool
}

func (s *Solver) Simulate(
	ctx context.Context,
	board generator.Board,
	actions []SimulationAction,
) (SimulationResult, error) {
	if err := ctx.Err(); err != nil {
		return SimulationResult{}, err
	}
	if err := generator.ValidateGeometry(board); err != nil {
		return SimulationResult{}, err
	}
	if len(board.Arrows) > s.config.MaxArrows {
		return SimulationResult{}, ErrResourceLimit
	}
	index := newOccupancyIndex(board)
	arrows := make(map[string]generator.Arrow, len(board.Arrows))
	for _, arrow := range board.Arrows {
		arrows[arrow.ID] = arrow
	}
	removed := make(map[string]bool, len(board.Arrows))
	steps := make([]SimulationStep, 0, len(actions))
	for _, action := range actions {
		if err := ctx.Err(); err != nil {
			return SimulationResult{}, err
		}
		arrow, exists := arrows[action.ArrowID]
		if !exists || removed[action.ArrowID] {
			return SimulationResult{}, ErrActionMismatch
		}
		collision, err := index.collision(board, removed, arrow)
		if err != nil {
			return SimulationResult{}, err
		}
		if collision.Clear != action.Accepted {
			return SimulationResult{}, ErrActionMismatch
		}
		if action.Accepted {
			removed[action.ArrowID] = true
		}
		removedIDs := sortedRemoved(removed)
		steps = append(steps, SimulationStep{
			ArrowID: action.ArrowID, Accepted: action.Accepted,
			BlockerID: collision.BlockerID, CollisionCell: collision.CollisionCell,
			CollisionDistance: collision.Distance, EscapeDistance: collision.EscapeDistance,
			RemovedIDs: removedIDs, Complete: len(removedIDs) == len(board.Arrows),
		})
	}
	removedIDs := sortedRemoved(removed)
	return SimulationResult{
		Steps: steps, RemovedIDs: removedIDs,
		Complete: len(removedIDs) == len(board.Arrows),
	}, nil
}

func sortedRemoved(removed map[string]bool) []string {
	result := make([]string, 0, len(removed))
	for id, isRemoved := range removed {
		if isRemoved {
			result = append(result, id)
		}
	}
	sort.Strings(result)
	return result
}
