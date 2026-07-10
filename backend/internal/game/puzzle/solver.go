package puzzle

import (
	"context"
	"errors"

	"skill-arena/internal/game"
	"skill-arena/internal/models"
)

var ErrUnsolvable = errors.New("puzzle solver could not clear generated board")

func Solve(ctx context.Context, lines []models.ArrowLine) ([]string, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	solution, ok := game.SolveLinePuzzle(cloneLines(lines))
	if !ok {
		return nil, ErrUnsolvable
	}
	return solution, nil
}
