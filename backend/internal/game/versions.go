package game

import "skill-arena/internal/models"

const (
	PuzzleEngineVersion      = "v1.0.0"
	GeneratorVersion         = "v1.0.0"
	DifficultyProfileVersion = "v1"
	GameRulesVersion         = "v1.0.0"
	ReplayVersion            = "v1"
)

func CurrentPuzzleVersion() models.PuzzleVersion {
	return models.PuzzleVersion{
		PuzzleEngineVersion:      PuzzleEngineVersion,
		GeneratorVersion:         GeneratorVersion,
		DifficultyProfileVersion: DifficultyProfileVersion,
		GameRulesVersion:         GameRulesVersion,
		ReplayVersion:            ReplayVersion,
	}
}
