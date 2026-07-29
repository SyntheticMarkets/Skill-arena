package engine

type Progress struct {
	Removed           int  `json:"removed"`
	Remaining         int  `json:"remaining"`
	Total             int  `json:"total"`
	CompletionBPS     int  `json:"completionBps"`
	SuccessfulActions int  `json:"successfulActions"`
	BlockedActions    int  `json:"blockedActions"`
	CurrentCombo      int  `json:"currentCombo"`
	MaximumCombo      int  `json:"maximumCombo"`
	Complete          bool `json:"complete"`
}

func ProgressFor(state State) Progress {
	total := len(state.Board.Arrows)
	removed := len(state.RemovedIDs)
	completion := 0
	if total > 0 {
		completion = removed * 10_000 / total
	}
	return Progress{
		Removed: removed, Remaining: total - removed, Total: total,
		CompletionBPS: completion, SuccessfulActions: state.SuccessfulActions,
		BlockedActions: state.BlockedActions, CurrentCombo: state.CurrentCombo,
		MaximumCombo: state.MaximumCombo, Complete: state.Status == StatusCompleted,
	}
}

type ScoreInputs struct {
	Completed         bool   `json:"completed"`
	ElapsedMS         int64  `json:"elapsedMs"`
	MinimumActions    int    `json:"minimumActions"`
	SuccessfulActions int    `json:"successfulActions"`
	BlockedActions    int    `json:"blockedActions"`
	EfficiencyBPS     int    `json:"efficiencyBps"`
	MaximumCombo      int    `json:"maximumCombo"`
	DifficultyHash    string `json:"difficultyHash"`
	PuzzleHash        string `json:"puzzleHash"`
}

func ScoreFor(state State, atMS int64) ScoreInputs {
	end := atMS
	if state.CompletedAtMS >= 0 {
		end = state.CompletedAtMS
	}
	if end < state.StartedAtMS {
		end = state.StartedAtMS
	}
	if end > state.DeadlineAtMS {
		end = state.DeadlineAtMS
	}
	attempts := state.SuccessfulActions + state.BlockedActions
	efficiency := 0
	if attempts > 0 {
		efficiency = state.MinimumActions * 10_000 / attempts
		if efficiency > 10_000 {
			efficiency = 10_000
		}
	}
	return ScoreInputs{
		Completed: state.Status == StatusCompleted, ElapsedMS: end - state.StartedAtMS,
		MinimumActions: state.MinimumActions, SuccessfulActions: state.SuccessfulActions,
		BlockedActions: state.BlockedActions, EfficiencyBPS: efficiency,
		MaximumCombo: state.MaximumCombo, DifficultyHash: state.DifficultyHash,
		PuzzleHash: state.PuzzleHash,
	}
}
