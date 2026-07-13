package core

import (
	"context"
	"time"

	"skill-arena/internal/models"
)

const (
	EventSessionStarted = "session_started"
	EventActionAccepted = "action_accepted"
	EventActionRejected = "action_rejected"
	EventPuzzleSolved   = "puzzle_solved"
	EventMatchWon       = "match_won"
	EventMatchLost      = "match_lost"
	EventReplayReady    = "replay_ready"
)

type Metadata struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Version     string   `json:"version"`
	Modes       []string `json:"modes"`
	RendererKey string   `json:"rendererKey"`
}

type SessionRequest struct {
	ActorUserID string
	Mode        string
	MatchID     string
}

type SessionSpec struct {
	GameID       string
	Seed         string
	RulesVersion string
	GameVersion  string
	Metadata     map[string]string
}

type PlayerAction struct {
	ActionType string
	TargetID   string
	ClientTime time.Time
	Metadata   map[string]string
}

type ActionRequest struct {
	ActorUserID string
	Session     *models.GameSession
	Actions     []PlayerAction
}

type ActionResult struct {
	Accepted      bool
	Complete      bool
	Valid         bool
	History       []models.MazeMove
	Lines         []models.ArrowLine
	Clicks        []models.ArrowClick
	RejectedEvent string
	Events        []Event
}

type ReplayContract struct {
	Version           string   `json:"version"`
	RequiredStreams   []string `json:"requiredStreams"`
	IntegrityRequired bool     `json:"integrityRequired"`
}

type StatisticsContract struct {
	Metrics []string `json:"metrics"`
}

type Event struct {
	Type      string
	GameID    string
	SessionID string
	UserID    string
	Payload   map[string]string
	CreatedAt time.Time
}

type GameModule interface {
	ID() string
	Name() string
	Metadata() Metadata
	CreateSession(context.Context, SessionRequest) (SessionSpec, error)
	JoinSession(context.Context, SessionRequest) error
	StartSession(context.Context, *models.GameSession) error
	SubmitAction(context.Context, ActionRequest) (ActionResult, error)
	ValidateAction(context.Context, ActionRequest) error
	FinishSession(context.Context, *models.GameSession, ActionResult) error
	Replay() ReplayContract
	Statistics() StatisticsContract
}
