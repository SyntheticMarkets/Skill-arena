package models

import "time"

type PvPMatch struct {
	ID                string             `json:"id"`
	PlayerAID         string             `json:"playerAId"`
	PlayerBID         string             `json:"playerBId,omitempty"`
	QueueType         string             `json:"queueType"`
	WalletType        string             `json:"walletType"`
	Stake             float64            `json:"stake"`
	PrizePool         float64            `json:"prizePool"`
	PlatformFee       float64            `json:"platformFee"`
	Status            string             `json:"status"`
	WinnerID          string             `json:"winnerId,omitempty"`
	DifficultyRating  int                `json:"difficultyRating,omitempty"`
	DifficultyProfile *DifficultyProfile `json:"difficultyProfile,omitempty"`
	PuzzleVersion     PuzzleVersion      `json:"puzzleVersion,omitempty"`
	PlayerASeed       string             `json:"playerASeed,omitempty"`
	PlayerBSeed       string             `json:"playerBSeed,omitempty"`
	PlayerANonce      string             `json:"playerANonce,omitempty"`
	PlayerBNonce      string             `json:"playerBNonce,omitempty"`
	PlayerAHash       string             `json:"playerAGenerationHash,omitempty"`
	PlayerBHash       string             `json:"playerBGenerationHash,omitempty"`
	MazeCells         []string           `json:"mazeCells,omitempty"`
	Width             int                `json:"width,omitempty"`
	Height            int                `json:"height,omitempty"`
	StartX            int                `json:"startX,omitempty"`
	StartY            int                `json:"startY,omitempty"`
	EndX              int                `json:"endX,omitempty"`
	EndY              int                `json:"endY,omitempty"`
	PlayerALines      []ArrowLine        `json:"playerALines,omitempty"`
	PlayerBLines      []ArrowLine        `json:"playerBLines,omitempty"`
	PlayerAProgress   *PvPProgress       `json:"playerAProgress,omitempty"`
	PlayerBProgress   *PvPProgress       `json:"playerBProgress,omitempty"`
	CreatedAt         time.Time          `json:"createdAt"`
	StartedAt         *time.Time         `json:"startedAt,omitempty"`
	CompletedAt       *time.Time         `json:"completedAt,omitempty"`
}

type PvPProgress struct {
	UserID             string     `json:"userId"`
	CurrentProgress    int        `json:"currentProgress"`
	CurrentCombo       int        `json:"currentCombo"`
	MovesRemaining     int        `json:"movesRemaining"`
	CompletionPercent  float64    `json:"completionPercent"`
	Finished           bool       `json:"finished"`
	Disconnected       bool       `json:"disconnected"`
	LastEventAt        time.Time  `json:"lastEventAt"`
	FinishedAt         *time.Time `json:"finishedAt,omitempty"`
	LastClientSequence int        `json:"lastClientSequence"`
}

type PvPSubmission struct {
	ID              string       `json:"id"`
	MatchID         string       `json:"matchId"`
	UserID          string       `json:"userId"`
	Moves           []MazeMove   `json:"moves"`
	Clicks          []ArrowClick `json:"clicks,omitempty"`
	IsValidRoute    bool         `json:"isValidRoute"`
	MoveCount       int          `json:"moveCount"`
	DurationSeconds float64      `json:"durationSeconds"`
	SubmittedAt     time.Time    `json:"submittedAt"`
}

type PvPMatchDetail struct {
	Match       *PvPMatch        `json:"match"`
	Submissions []*PvPSubmission `json:"submissions"`
}
