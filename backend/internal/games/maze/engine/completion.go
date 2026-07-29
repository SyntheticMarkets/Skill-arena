package engine

import (
	"context"

	"skill-arena/internal/games/interfaces"
)

func Completion(ctx context.Context, state State) (interfaces.CompletionResult, error) {
	if err := ctx.Err(); err != nil {
		return interfaces.CompletionResult{}, err
	}
	if err := state.Validate(); err != nil {
		return interfaces.CompletionResult{}, err
	}
	switch state.Status {
	case StatusCompleted:
		return interfaces.CompletionResult{Status: "complete", Reason: "puzzle_cleared"}, nil
	case StatusTimedOut:
		return interfaces.CompletionResult{Status: "timeout", Reason: "server_deadline_elapsed"}, nil
	default:
		return interfaces.CompletionResult{Status: "incomplete"}, nil
	}
}
