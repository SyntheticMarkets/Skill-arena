package models

type PlatformStats struct {
	LaunchPhase        string   `json:"launchPhase"`
	PlayersOnline      int      `json:"playersOnline"`
	LiveMatches        int      `json:"liveMatches"`
	MatchesToday       int      `json:"matchesToday"`
	Countries          int      `json:"countries"`
	CurrentSeason      string   `json:"currentSeason,omitempty"`
	PrizePool          *float64 `json:"prizePool"`
	LeaderboardsStatus string   `json:"leaderboardsStatus"`
}

type PuzzlePreview struct {
	Lines             []ArrowLine       `json:"lines"`
	AnimationLineIDs  []string          `json:"animationLineIds"`
	BlockedAttemptID  string            `json:"blockedAttemptId,omitempty"`
	UnlockedRetryID   string            `json:"unlockedRetryId,omitempty"`
	DifficultyProfile DifficultyProfile `json:"difficultyProfile"`
	PuzzleVersion     PuzzleVersion     `json:"puzzleVersion"`
}
