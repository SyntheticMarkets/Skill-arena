package models

import "time"

const (
	SessionStateCreated    = "CREATED"
	SessionStateGenerating = "GENERATING"
	SessionStateReady      = "READY"
	SessionStateActive     = "ACTIVE"
	SessionStateCompleted  = "COMPLETED"
	SessionStateCancelled  = "CANCELLED"
	SessionStateExpired    = "EXPIRED"
)

type PuzzleVersion struct {
	PuzzleEngineVersion      string `json:"puzzleEngineVersion"`
	GeneratorVersion         string `json:"generatorVersion"`
	DifficultyProfileVersion string `json:"difficultyProfileVersion"`
	GameRulesVersion         string `json:"gameRulesVersion"`
	ReplayVersion            string `json:"replayVersion"`
}

type PuzzleMetadata struct {
	PuzzleID                 string        `json:"puzzleId"`
	Seed                     string        `json:"seed"`
	GenerationHash           string        `json:"generationHash"`
	Nonce                    string        `json:"nonce,omitempty"`
	Mode                     string        `json:"mode"`
	Shared                   bool          `json:"shared"`
	Difficulty               int           `json:"difficulty"`
	Level                    int           `json:"level"`
	BranchCount              int           `json:"branchCount"`
	DependencyDepth          int           `json:"dependencyDepth"`
	MinimumMoves             int           `json:"minimumMoves"`
	ExpectedSolveTimeSeconds int           `json:"expectedSolveTimeSeconds"`
	ComplexityScore          int           `json:"complexityScore"`
	TrustScore               float64       `json:"trustScore,omitempty"`
	GeneratorVersion         string        `json:"generatorVersion"`
	SolverVersion            string        `json:"solverVersion"`
	PuzzleVersion            PuzzleVersion `json:"puzzleVersion"`
}

type GameSession struct {
	ID                string             `json:"id"`
	UserID            string             `json:"userId"`
	GameType          string             `json:"gameType"`
	Mode              string             `json:"mode,omitempty"`
	HouseTier         string             `json:"houseTier,omitempty"`
	Calibration       bool               `json:"calibration,omitempty"`
	Stake             float64            `json:"stake"`
	RewardRate        float64            `json:"rewardRate,omitempty"`
	Difficulty        int                `json:"difficulty,omitempty"`
	DifficultyRating  int                `json:"difficultyRating,omitempty"`
	DifficultyProfile *DifficultyProfile `json:"difficultyProfile,omitempty"`
	PuzzleSeed        string             `json:"puzzleSeed,omitempty"`
	GenerationNonce   string             `json:"generationNonce,omitempty"`
	GenerationHash    string             `json:"generationHash,omitempty"`
	PuzzleVersion     PuzzleVersion      `json:"puzzleVersion,omitempty"`
	PuzzleMetadata    *PuzzleMetadata    `json:"puzzleMetadata,omitempty"`
	State             string             `json:"state,omitempty"`
	Outcome           string             `json:"outcome,omitempty"`
	Reward            float64            `json:"reward,omitempty"`
	MazeCells         []string           `json:"mazeCells,omitempty"`
	Width             int                `json:"width,omitempty"`
	Height            int                `json:"height,omitempty"`
	StartX            int                `json:"startX,omitempty"`
	StartY            int                `json:"startY,omitempty"`
	EndX              int                `json:"endX,omitempty"`
	EndY              int                `json:"endY,omitempty"`
	Moves             []MazeMove         `json:"moves,omitempty"`
	Lines             []ArrowLine        `json:"lines,omitempty"`
	Clicks            []ArrowClick       `json:"clicks,omitempty"`
	CreatedAt         time.Time          `json:"createdAt"`
	CompletedAt       *time.Time         `json:"completedAt,omitempty"`
	IsFinished        bool               `json:"isFinished"`
}

type DifficultyProfile struct {
	Rating             int              `json:"rating"`
	ComplexityScore    int              `json:"complexityScore"`
	LineCount          int              `json:"lineCount"`
	DependencyDepth    int              `json:"dependencyDepth"`
	BranchingFactor    int              `json:"branchingFactor"`
	FalseRouteRate     float64          `json:"falseRouteRate"`
	DependencyTrees    int              `json:"dependencyTrees"`
	CrossDependencies  int              `json:"crossDependencies"`
	NoiseFactor        float64          `json:"noiseFactor"`
	DeadEndFactor      float64          `json:"deadEndFactor"`
	HumanSolveEstimate int              `json:"humanSolveEstimateSeconds"`
	ExpectedSolve      SolvePercentiles `json:"expectedSolvePercentiles"`
	Source             string           `json:"source,omitempty"`
}

type SolvePercentiles struct {
	TopOnePercentSeconds int `json:"topOnePercentSeconds"`
	TopTenPercentSeconds int `json:"topTenPercentSeconds"`
	AverageSeconds       int `json:"averageSeconds"`
}

type MazeMove struct {
	Direction string    `json:"direction"`
	X         int       `json:"x"`
	Y         int       `json:"y"`
	Timestamp time.Time `json:"timestamp"`
}

type ArrowLine struct {
	ID        string   `json:"id"`
	Direction string   `json:"direction"`
	X         float64  `json:"x"`
	Y         float64  `json:"y"`
	Length    float64  `json:"length"`
	Points    []Point  `json:"points,omitempty"`
	DependsOn []string `json:"dependsOn,omitempty"`
	Blocked   bool     `json:"blocked,omitempty"`
	Removed   bool     `json:"removed,omitempty"`
}

type Point struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
}

type ArrowClick struct {
	LineID        string    `json:"lineId"`
	Timestamp     time.Time `json:"timestamp"`
	Success       bool      `json:"success"`
	FailureReason string    `json:"failureReason,omitempty"`
	Combo         int       `json:"combo"`
	ClearedCount  int       `json:"clearedCount"`
}

type LeaderboardEntry struct {
	UserID      string  `json:"userId"`
	Username    string  `json:"username"`
	DisplayName string  `json:"displayName"`
	LeagueTier  string  `json:"leagueTier"`
	Rating      int     `json:"rating"`
	Country     string  `json:"country"`
	Score       float64 `json:"score"`
	Rank        int     `json:"rank"`
}
