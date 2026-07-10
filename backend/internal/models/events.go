package models

import "time"

const (
	WebSocketEventContractVersion = "v1"

	EventMatchStarted        = "match_started"
	EventProgressUpdated     = "progress_updated"
	EventMatchFinished       = "match_finished"
	EventNotificationCreated = "notification_created"
)

type WebSocketEvent struct {
	Version   string    `json:"version"`
	Type      string    `json:"type"`
	EventID   string    `json:"eventId"`
	Scope     string    `json:"scope"`
	ScopeID   string    `json:"scopeId"`
	UserID    string    `json:"userId,omitempty"`
	Payload   any       `json:"payload"`
	CreatedAt time.Time `json:"createdAt"`
}

type MatchStartedPayload struct {
	MatchID          string        `json:"matchId"`
	Mode             string        `json:"mode"`
	DifficultyRating int           `json:"difficultyRating"`
	PuzzleVersion    PuzzleVersion `json:"puzzleVersion"`
	StartedAt        time.Time     `json:"startedAt"`
}

type ProgressUpdatedPayload struct {
	MatchID                 string    `json:"matchId"`
	UserID                  string    `json:"userId"`
	ProgressPercent         float64   `json:"progressPercent"`
	CompletedLines          int       `json:"completedLines"`
	EstimatedMovesRemaining int       `json:"estimatedMovesRemaining"`
	UpdatedAt               time.Time `json:"updatedAt"`
}

type MatchFinishedPayload struct {
	MatchID    string    `json:"matchId"`
	WinnerID   string    `json:"winnerId,omitempty"`
	Outcome    string    `json:"outcome"`
	FinishedAt time.Time `json:"finishedAt"`
}

type NotificationCreatedPayload struct {
	NotificationID string            `json:"notificationId"`
	Category       string            `json:"category"`
	Title          string            `json:"title"`
	Metadata       map[string]string `json:"metadata,omitempty"`
}
