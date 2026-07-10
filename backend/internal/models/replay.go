package models

import "time"

type ReplayReport struct {
	SessionID          string             `json:"sessionId"`
	UserID             string             `json:"userId"`
	GameType           string             `json:"gameType"`
	Mode               string             `json:"mode,omitempty"`
	HouseTier          string             `json:"houseTier,omitempty"`
	Difficulty         int                `json:"difficulty,omitempty"`
	DifficultyRating   int                `json:"difficultyRating,omitempty"`
	DifficultyProfile  *DifficultyProfile `json:"difficultyProfile,omitempty"`
	PuzzleSeed         string             `json:"puzzleSeed,omitempty"`
	GenerationNonce    string             `json:"generationNonce,omitempty"`
	GenerationHash     string             `json:"generationHash,omitempty"`
	PuzzleVersion      PuzzleVersion      `json:"puzzleVersion,omitempty"`
	ReviewStatus       string             `json:"reviewStatus,omitempty"`
	Outcome            string             `json:"outcome"`
	Stake              float64            `json:"stake"`
	Reward             float64            `json:"reward"`
	IsFinished         bool               `json:"isFinished"`
	IsValidRoute       bool               `json:"isValidRoute"`
	MoveCount          int                `json:"moveCount"`
	ShortestPathLength int                `json:"shortestPathLength"`
	RouteEfficiency    float64            `json:"routeEfficiency"`
	DurationSeconds    float64            `json:"durationSeconds"`
	IntegrityStatus    string             `json:"integrityStatus"`
	Flags              []string           `json:"flags"`
	MazeCells          []string           `json:"mazeCells,omitempty"`
	Moves              []MazeMove         `json:"moves,omitempty"`
	Lines              []ArrowLine        `json:"lines,omitempty"`
	Clicks             []ArrowClick       `json:"clicks,omitempty"`
	PlaybackEvents     []ReplayEvent      `json:"playbackEvents,omitempty"`
	ReplaySignature    string             `json:"replaySignature,omitempty"`
	ReplaySignedAt     *time.Time         `json:"replaySignedAt,omitempty"`
	CreatedAt          time.Time          `json:"createdAt"`
	CompletedAt        *time.Time         `json:"completedAt,omitempty"`
}

type ReplayEvent struct {
	Type     string            `json:"type"`
	AtMs     int64             `json:"atMs"`
	LineID   string            `json:"lineId,omitempty"`
	Move     *MazeMove         `json:"move,omitempty"`
	Click    *ArrowClick       `json:"click,omitempty"`
	Metadata map[string]string `json:"metadata,omitempty"`
}
