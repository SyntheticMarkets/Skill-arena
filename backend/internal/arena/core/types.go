package core

import (
	"context"
	"time"

	"skill-arena/internal/models"
)

const (
	EventPlayerRegistered  = "player_registered"
	EventPracticeStarted   = "practice_started"
	EventSessionStarted    = "session_started"
	EventPuzzleGenerated   = "puzzle_generated"
	EventActionAccepted    = "action_accepted"
	EventActionRejected    = "action_rejected"
	EventMoveAccepted      = "move_accepted"
	EventMoveRejected      = "move_rejected"
	EventPuzzleSolved      = "puzzle_solved"
	EventMatchWon          = "match_won"
	EventMatchLost         = "match_lost"
	EventRewardsCalculated = "rewards_calculated"
	EventWalletCredited    = "wallet_credited"
	EventChallengeUpdated  = "challenge_updated"
	EventXPGranted         = "xp_granted"
	EventReplaySigned      = "replay_signed"
	EventReplayReady       = "replay_ready"
	EventStatisticsUpdated = "statistics_updated"
	EventNotificationSent  = "notification_sent"
)

type Metadata struct {
	ID             string          `json:"id"`
	Name           string          `json:"name"`
	Description    string          `json:"description,omitempty"`
	Category       string          `json:"category,omitempty"`
	Version        string          `json:"version"`
	Author         string          `json:"author,omitempty"`
	Difficulty     string          `json:"difficulty,omitempty"`
	MinimumPlayers int             `json:"minimumPlayers"`
	MaximumPlayers int             `json:"maximumPlayers"`
	AverageTimeSec int             `json:"averageTimeSeconds"`
	Modes          []string        `json:"modes"`
	RendererKey    string          `json:"rendererKey"`
	Versions       VersionSet      `json:"versions"`
	Capabilities   CapabilityFlags `json:"capabilities"`
}

type VersionSet struct {
	Game     string `json:"game"`
	Rules    string `json:"rules"`
	Replay   string `json:"replay"`
	Protocol string `json:"protocol"`
}

type CapabilityFlags struct {
	Practice   bool `json:"practice"`
	PvP        bool `json:"pvp"`
	Replay     bool `json:"replay"`
	Tournament bool `json:"tournament"`
	Spectator  bool `json:"spectator"`
	AI         bool `json:"ai"`
	Teams      bool `json:"teams"`
	Coins      bool `json:"coins"`
}

type Manifest struct {
	ID             string          `json:"id"`
	Name           string          `json:"name"`
	Description    string          `json:"description"`
	Category       string          `json:"category"`
	Version        string          `json:"version"`
	Author         string          `json:"author"`
	Difficulty     string          `json:"difficulty"`
	MinimumPlayers int             `json:"minimumPlayers"`
	MaximumPlayers int             `json:"maximumPlayers"`
	AverageTimeSec int             `json:"averageTimeSeconds"`
	RendererKey    string          `json:"rendererKey"`
	Modes          []string        `json:"modes"`
	Versions       VersionSet      `json:"versions"`
	Capabilities   CapabilityFlags `json:"capabilities"`
}

type SessionContext struct {
	ActorUserID   string
	Mode          string
	MatchID       string
	User          *models.User
	Session       *models.GameSession
	Wallet        *models.Wallet
	Season        *models.Season
	League        string
	TrustScore    float64
	HouseTier     string
	TournamentID  string
	Practice      bool
	Configuration map[string]string
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
	Sequence   int
	Metadata   map[string]string
}

type ActionContext struct {
	ActorUserID      string
	Mode             string
	MatchID          string
	User             *models.User
	JWTUserID        string
	Session          *models.GameSession
	Wallet           *models.Wallet
	Season           *models.Season
	League           string
	TrustScore       float64
	HouseTier        string
	TournamentID     string
	Practice         bool
	Configuration    map[string]string
	Action           PlayerAction
	Actions          []PlayerAction
	LatencyMs        int
	ReplayPosition   int
	SequenceNumber   int
	ServerReceivedAt time.Time
}

type SessionRequest = SessionContext
type ActionRequest = ActionContext

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
	Manifest() Manifest
	Capabilities() CapabilityFlags
	Metadata() Metadata
	CreateSession(context.Context, SessionRequest) (SessionSpec, error)
	JoinSession(context.Context, SessionRequest) error
	StartSession(context.Context, *models.GameSession) error
	ResumeSession(context.Context, SessionRequest) error
	SubmitAction(context.Context, ActionRequest) (ActionResult, error)
	ValidateAction(context.Context, ActionRequest) error
	SyncSession(context.Context, SessionRequest) (SessionSpec, error)
	LeaveSession(context.Context, SessionRequest) error
	FinishSession(context.Context, *models.GameSession, ActionResult) error
	Replay() ReplayContract
	Statistics() StatisticsContract
}
