package puzzle

import (
	"context"
	"errors"

	"skill-arena/internal/game"
	"skill-arena/internal/models"
)

var ErrValidationFailed = errors.New("puzzle validation failed")

func Validate(ctx context.Context, lines []models.ArrowLine, solution []string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if len(lines) == 0 || len(solution) == 0 {
		return ErrValidationFailed
	}
	complete, _, _ := game.ValidateLineClicks(cloneLines(lines), solution)
	if !complete {
		return ErrValidationFailed
	}
	return nil
}
