package engine

import (
	"fmt"

	"skill-arena/internal/games/maze/generator"
)

func TutorialBoard(level int) (generator.Board, error) {
	if level < 1 || level > 5 {
		return generator.Board{}, fmt.Errorf("tutorial level must be between 1 and 5")
	}
	count := level + 1
	columns := count*3 + 1
	arrows := make([]generator.Arrow, count)
	for index := 0; index < count; index++ {
		head := columns - 1 - index*3
		arrows[index] = generator.Arrow{
			ID: fmt.Sprintf("a%04d", index), Direction: generator.DirectionRight,
			Cells: []generator.Cell{
				{Column: head - 1, Row: 2},
				{Column: head, Row: 2},
			},
		}
	}
	board := generator.Board{
		GeometryVersion: 1, RulesVersion: 1,
		Columns: columns, Rows: 6, Arrows: arrows,
	}
	if err := generator.ValidateGeometry(board); err != nil {
		return generator.Board{}, err
	}
	return board, nil
}
