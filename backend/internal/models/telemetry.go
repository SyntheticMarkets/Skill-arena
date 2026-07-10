package models

import "time"

type MovementSample struct {
	X         float64   `json:"x"`
	Y         float64   `json:"y"`
	Timestamp time.Time `json:"timestamp"`
	Kind      string    `json:"kind,omitempty"`
}

type GameplayTelemetry struct {
	ID                    string           `json:"id"`
	UserID                string           `json:"userId"`
	Scope                 string           `json:"scope"`
	ScopeID               string           `json:"scopeId"`
	ClickTimestamps       []time.Time      `json:"clickTimestamps,omitempty"`
	ClickIntervalsMs      []int            `json:"clickIntervalsMs,omitempty"`
	MouseMovement         []MovementSample `json:"mouseMovement,omitempty"`
	TouchMovement         []MovementSample `json:"touchMovement,omitempty"`
	DeviceFingerprint     string           `json:"deviceFingerprint,omitempty"`
	UserAgent             string           `json:"userAgent,omitempty"`
	ReactionVarianceMs    float64          `json:"reactionVarianceMs,omitempty"`
	Accuracy              float64          `json:"accuracy,omitempty"`
	FailedClicks          int              `json:"failedClicks"`
	SuccessfulClicks      int              `json:"successfulClicks"`
	CollectedAt           time.Time        `json:"collectedAt"`
	PrivacyClassification string           `json:"privacyClassification"`
}

type ReviewCase struct {
	ID         string            `json:"id"`
	Scope      string            `json:"scope"`
	ScopeID    string            `json:"scopeId"`
	UserID     string            `json:"userId"`
	Status     string            `json:"status"`
	Reason     string            `json:"reason"`
	Flags      []string          `json:"flags,omitempty"`
	ReviewerID string            `json:"reviewerId,omitempty"`
	Decision   string            `json:"decision,omitempty"`
	Metadata   map[string]string `json:"metadata,omitempty"`
	CreatedAt  time.Time         `json:"createdAt"`
	UpdatedAt  time.Time         `json:"updatedAt"`
}

type MetricsSnapshot struct {
	PuzzleGenerationCount       int       `json:"puzzleGenerationCount"`
	TotalPuzzleGenerationMs     float64   `json:"totalPuzzleGenerationMs"`
	ReplayReconstructionCount   int       `json:"replayReconstructionCount"`
	TotalReplayReconstructionMs float64   `json:"totalReplayReconstructionMs"`
	MatchmakingCount            int       `json:"matchmakingCount"`
	TotalMatchmakingMs          float64   `json:"totalMatchmakingMs"`
	CompletedMatchCount         int       `json:"completedMatchCount"`
	TotalCompletionSeconds      float64   `json:"totalCompletionSeconds"`
	TotalFailedClicks           int       `json:"totalFailedClicks"`
	APICount                    int       `json:"apiCount"`
	TotalAPILatencyMs           float64   `json:"totalApiLatencyMs"`
	ValidationFailures          int       `json:"validationFailures"`
	UpdatedAt                   time.Time `json:"updatedAt"`
}
