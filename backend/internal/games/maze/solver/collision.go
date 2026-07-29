package solver

import (
	"fmt"

	"skill-arena/internal/games/maze/generator"
)

type Collision struct {
	Clear          bool
	BlockerID      string
	CollisionCell  generator.Cell
	Distance       int
	EscapeDistance int
}

type occupancyIndex struct {
	owners map[uint64]string
}

func newOccupancyIndex(board generator.Board) occupancyIndex {
	owners := make(map[uint64]string, len(board.Arrows)*3)
	for _, arrow := range board.Arrows {
		for _, cell := range arrow.Cells {
			owners[cellKey(cell)] = arrow.ID
		}
	}
	return occupancyIndex{owners: owners}
}

func (index occupancyIndex) collision(
	board generator.Board,
	removed map[string]bool,
	arrow generator.Arrow,
) (Collision, error) {
	dx, dy, ok := directionVector(arrow.Direction)
	if !ok {
		return Collision{}, fmt.Errorf("arrow %q has invalid direction", arrow.ID)
	}
	head := arrow.Head()
	for distance := 1; ; distance++ {
		cell := generator.Cell{Column: head.Column + dx*distance, Row: head.Row + dy*distance}
		if !inside(board, cell) {
			return Collision{
				Clear: true, Distance: distance,
				EscapeDistance: completeEscapeDistance(board, arrow),
			}, nil
		}
		owner, occupied := index.owners[cellKey(cell)]
		if !occupied || owner == arrow.ID || removed[owner] {
			continue
		}
		return Collision{
			BlockerID: owner, CollisionCell: cell, Distance: distance,
		}, nil
	}
}

func completeEscapeDistance(board generator.Board, arrow generator.Arrow) int {
	minColumn, maxColumn := arrow.Cells[0].Column, arrow.Cells[0].Column
	minRow, maxRow := arrow.Cells[0].Row, arrow.Cells[0].Row
	for _, cell := range arrow.Cells[1:] {
		if cell.Column < minColumn {
			minColumn = cell.Column
		}
		if cell.Column > maxColumn {
			maxColumn = cell.Column
		}
		if cell.Row < minRow {
			minRow = cell.Row
		}
		if cell.Row > maxRow {
			maxRow = cell.Row
		}
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
		return 0
	}
}

func directionVector(direction generator.Direction) (int, int, bool) {
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

func inside(board generator.Board, cell generator.Cell) bool {
	return cell.Column >= 0 && cell.Column < board.Columns &&
		cell.Row >= 0 && cell.Row < board.Rows
}

func cellKey(cell generator.Cell) uint64 {
	return uint64(uint32(cell.Row))<<32 | uint64(uint32(cell.Column))
}
