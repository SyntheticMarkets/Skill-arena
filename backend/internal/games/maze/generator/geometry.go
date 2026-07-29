package generator

import (
	"errors"
	"fmt"
)

type Direction uint8

const (
	DirectionRight Direction = iota
	DirectionUp
	DirectionLeft
	DirectionDown
)

func (d Direction) String() string {
	switch d {
	case DirectionRight:
		return "right"
	case DirectionUp:
		return "up"
	case DirectionLeft:
		return "left"
	case DirectionDown:
		return "down"
	default:
		return "invalid"
	}
}

func (d Direction) vector() (int, int) {
	switch d {
	case DirectionRight:
		return 1, 0
	case DirectionUp:
		return 0, -1
	case DirectionLeft:
		return -1, 0
	case DirectionDown:
		return 0, 1
	default:
		return 0, 0
	}
}

type Cell struct {
	Column int `json:"column"`
	Row    int `json:"row"`
}

func (c Cell) key() uint64 {
	return uint64(uint32(c.Row))<<32 | uint64(uint32(c.Column))
}

type Arrow struct {
	ID        string    `json:"id"`
	Cells     []Cell    `json:"cells"`
	Direction Direction `json:"direction"`
}

func (a Arrow) Head() Cell {
	if len(a.Cells) == 0 {
		return Cell{}
	}
	return a.Cells[len(a.Cells)-1]
}

type Board struct {
	GeometryVersion int     `json:"geometryVersion"`
	RulesVersion    int     `json:"rulesVersion"`
	Columns         int     `json:"columns"`
	Rows            int     `json:"rows"`
	Arrows          []Arrow `json:"arrows"`
}

func (b Board) Clone() Board {
	cloned := b
	cloned.Arrows = make([]Arrow, len(b.Arrows))
	for index, arrow := range b.Arrows {
		cloned.Arrows[index] = arrow
		cloned.Arrows[index].Cells = append([]Cell(nil), arrow.Cells...)
	}
	return cloned
}

func ValidateGeometry(board Board) error {
	if board.GeometryVersion <= 0 || board.RulesVersion <= 0 {
		return errors.New("geometry and rules versions must be positive")
	}
	if board.Columns < 4 || board.Rows < 4 || board.Columns > 64 || board.Rows > 64 {
		return errors.New("board dimensions are outside production bounds")
	}
	if len(board.Arrows) < 2 || len(board.Arrows) > board.Columns*board.Rows/2 {
		return errors.New("arrow count is outside board capacity")
	}
	occupied := make(map[uint64]string, len(board.Arrows)*3)
	ids := make(map[string]struct{}, len(board.Arrows))
	for index, arrow := range board.Arrows {
		expectedID := fmt.Sprintf("a%04d", index)
		if arrow.ID != expectedID {
			return fmt.Errorf("arrow id %q is not canonical", arrow.ID)
		}
		if _, exists := ids[arrow.ID]; exists {
			return fmt.Errorf("duplicate arrow id %q", arrow.ID)
		}
		ids[arrow.ID] = struct{}{}
		dx, dy := arrow.Direction.vector()
		if dx == 0 && dy == 0 {
			return fmt.Errorf("arrow %q has an invalid direction", arrow.ID)
		}
		if len(arrow.Cells) < 2 || len(arrow.Cells) > 32 {
			return fmt.Errorf("arrow %q has an invalid path length", arrow.ID)
		}
		local := make(map[uint64]struct{}, len(arrow.Cells))
		for cellIndex, cell := range arrow.Cells {
			if cell.Column < 0 || cell.Column >= board.Columns || cell.Row < 0 || cell.Row >= board.Rows {
				return fmt.Errorf("arrow %q contains an out-of-bounds cell", arrow.ID)
			}
			key := cell.key()
			if _, exists := local[key]; exists {
				return fmt.Errorf("arrow %q intersects itself", arrow.ID)
			}
			local[key] = struct{}{}
			if owner, exists := occupied[key]; exists {
				return fmt.Errorf("arrow %q overlaps arrow %q", arrow.ID, owner)
			}
			occupied[key] = arrow.ID
			if cellIndex > 0 {
				previous := arrow.Cells[cellIndex-1]
				distance := absInt(cell.Column-previous.Column) + absInt(cell.Row-previous.Row)
				if distance != 1 {
					return fmt.Errorf("arrow %q contains a disconnected path", arrow.ID)
				}
			}
		}
		preHead := arrow.Cells[len(arrow.Cells)-2]
		head := arrow.Head()
		if head.Column-preHead.Column != dx || head.Row-preHead.Row != dy {
			return fmt.Errorf("arrow %q head does not align with its direction", arrow.ID)
		}
	}
	return nil
}

func absInt(value int) int {
	if value < 0 {
		return -value
	}
	return value
}

func clampInt(value, minimum, maximum int) int {
	if value < minimum {
		return minimum
	}
	if value > maximum {
		return maximum
	}
	return value
}
