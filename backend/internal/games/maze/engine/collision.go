package engine

import (
	"errors"
	"fmt"

	"skill-arena/internal/games/maze/generator"
)

type Collision struct {
	Clear          bool           `json:"clear"`
	BlockerID      string         `json:"blockerId,omitempty"`
	CollisionCell  generator.Cell `json:"collisionCell"`
	Distance       int            `json:"distance"`
	EscapeDistance int            `json:"escapeDistance,omitempty"`
}

type CollisionModel struct {
	board  generator.Board
	arrows map[string]generator.Arrow
	owners map[uint64]string
}

func NewCollisionModel(board generator.Board) (*CollisionModel, error) {
	if err := generator.ValidateGeometry(board); err != nil {
		return nil, err
	}
	model := &CollisionModel{
		board: board.Clone(), arrows: make(map[string]generator.Arrow, len(board.Arrows)),
		owners: make(map[uint64]string, len(board.Arrows)*3),
	}
	for _, arrow := range model.board.Arrows {
		model.arrows[arrow.ID] = arrow
		for _, cell := range arrow.Cells {
			model.owners[cellKey(cell)] = arrow.ID
		}
	}
	return model, nil
}

func (m *CollisionModel) Evaluate(removedIDs []string, arrowID string) (Collision, error) {
	arrow, exists := m.arrows[arrowID]
	if !exists {
		return Collision{}, fail(CodeArrowUnknown, "the selected arrow does not exist")
	}
	removed := make(map[string]bool, len(removedIDs))
	for _, id := range removedIDs {
		if _, exists := m.arrows[id]; !exists {
			return Collision{}, fail(CodeStateInvalid, "removed-arrow state references an unknown arrow")
		}
		removed[id] = true
	}
	if removed[arrowID] {
		return Collision{}, fail(CodeArrowRemoved, "the selected arrow has already left the board")
	}
	dx, dy, ok := DirectionVector(arrow.Direction)
	if !ok {
		return Collision{}, errors.New("arrow has invalid direction")
	}
	head := arrow.Head()
	for distance := 1; ; distance++ {
		cell := generator.Cell{Column: head.Column + dx*distance, Row: head.Row + dy*distance}
		if !Inside(m.board, cell) {
			return Collision{
				Clear: true, Distance: distance,
				EscapeDistance: completeEscapeDistance(m.board, arrow),
			}, nil
		}
		owner, occupied := m.owners[cellKey(cell)]
		if !occupied || owner == arrow.ID || removed[owner] {
			continue
		}
		return Collision{
			BlockerID: owner, CollisionCell: cell, Distance: distance,
		}, nil
	}
}

func DirectionVector(direction generator.Direction) (int, int, bool) {
	switch direction {
	case generator.DirectionRight:
		return 1, 0, true
	case generator.DirectionUp:
		return 0, -1, true
	case generator.DirectionLeft:
		return -1, 0, true
	case generator.DirectionDown:
		return 0, 1, true
	default:
		return 0, 0, false
	}
}

func Inside(board generator.Board, cell generator.Cell) bool {
	return cell.Column >= 0 && cell.Column < board.Columns &&
		cell.Row >= 0 && cell.Row < board.Rows
}

func Move(cell generator.Cell, dx, dy int) generator.Cell {
	return generator.Cell{Column: cell.Column + dx, Row: cell.Row + dy}
}

func completeEscapeDistance(board generator.Board, arrow generator.Arrow) int {
	if len(arrow.Cells) == 0 {
		return 0
	}
	minColumn, maxColumn := arrow.Cells[0].Column, arrow.Cells[0].Column
	minRow, maxRow := arrow.Cells[0].Row, arrow.Cells[0].Row
	for _, cell := range arrow.Cells[1:] {
		minColumn = min(minColumn, cell.Column)
		maxColumn = max(maxColumn, cell.Column)
		minRow = min(minRow, cell.Row)
		maxRow = max(maxRow, cell.Row)
	}
	switch arrow.Direction {
	case generator.DirectionRight:
		return board.Columns - minColumn
	case generator.DirectionLeft:
		return maxColumn + 1
	case generator.DirectionUp:
		return maxRow + 1
	case generator.DirectionDown:
		return board.Rows - minRow
	default:
		panic(fmt.Sprintf("validated arrow %q has invalid direction", arrow.ID))
	}
}

func cellKey(cell generator.Cell) uint64 {
	return uint64(uint32(cell.Row))<<32 | uint64(uint32(cell.Column))
}
