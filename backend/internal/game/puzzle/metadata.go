package puzzle

import "skill-arena/internal/models"

const (
	ModePractice       = "practice"
	ModeTraining       = "training"
	ModePvP            = "pvp"
	ModeRanked         = "ranked"
	ModeTournament     = "tournament"
	ModeDailyChallenge = "daily_challenge"
	ModeLandingPreview = "landing_preview"
)

type Request struct {
	Mode              string
	Purpose           string
	MatchID           string
	PlayerID          string
	Shared            bool
	Nonce             string
	Level             int
	TrustScore        float64
	DifficultyProfile models.DifficultyProfile
	PuzzleVersion     models.PuzzleVersion
}

type Puzzle struct {
	Lines             []models.ArrowLine
	Solution          []string
	DifficultyProfile models.DifficultyProfile
	Metadata          models.PuzzleMetadata
}

func cloneLines(lines []models.ArrowLine) []models.ArrowLine {
	copied := make([]models.ArrowLine, len(lines))
	for i, line := range lines {
		copied[i] = line
		copied[i].Points = append([]models.Point(nil), line.Points...)
		copied[i].DependsOn = append([]string(nil), line.DependsOn...)
	}
	return copied
}
