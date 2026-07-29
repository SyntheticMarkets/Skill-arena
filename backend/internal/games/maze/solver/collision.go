package solver

import (
	"errors"

	"skill-arena/internal/games/maze/engine"
	"skill-arena/internal/games/maze/generator"
)

type Collision = engine.Collision

type occupancyIndex struct {
	owners map[uint64]string
	model  *engine.CollisionModel
	err    error
}

func newOccupancyIndex(board generator.Board) occupancyIndex {
	owners := make(map[uint64]string, len(board.Arrows)*3)
	for _, arrow := range board.Arrows {
		for _, cell := range arrow.Cells {
			owners[cellKey(cell)] = arrow.ID
		}
	}
	model, err := engine.NewCollisionModel(board)
	return occupancyIndex{owners: owners, model: model, err: err}
}

func (index occupancyIndex) collision(
	_ generator.Board,
	removed map[string]bool,
	arrow generator.Arrow,
) (Collision, error) {
	if index.err != nil {
		return Collision{}, index.err
	}
	if index.model == nil {
		return Collision{}, errors.New("collision authority is unavailable")
	}
	removedIDs := make([]string, 0, len(removed))
	for id, isRemoved := range removed {
		if isRemoved {
			removedIDs = append(removedIDs, id)
		}
	}
	return index.model.Evaluate(removedIDs, arrow.ID)
}

func directionVector(direction generator.Direction) (int, int, bool) {
	return engine.DirectionVector(direction)
}

func inside(board generator.Board, cell generator.Cell) bool {
	return engine.Inside(board, cell)
}

func cellKey(cell generator.Cell) uint64 {
	return uint64(uint32(cell.Row))<<32 | uint64(uint32(cell.Column))
}
