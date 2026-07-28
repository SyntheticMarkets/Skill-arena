package interfaces

import (
	"context"
	"encoding/json"
)

type ModuleStatus string

const (
	ModuleActive     ModuleStatus = "active"
	ModuleReplayOnly ModuleStatus = "replay_only"
	ModuleRetired    ModuleStatus = "retired"
	ModuleRevoked    ModuleStatus = "revoked"
)

type Capabilities struct {
	Practice       bool `json:"practice"`
	PvP            bool `json:"pvp"`
	Ranked         bool `json:"ranked"`
	HouseChallenge bool `json:"houseChallenge"`
	DailyChallenge bool `json:"dailyChallenge"`
	Tournament     bool `json:"tournament"`
	Replay         bool `json:"replay"`
	Spectator      bool `json:"spectator"`
	AI             bool `json:"ai"`
	Teams          bool `json:"teams"`
}

type Descriptor struct {
	ID              string       `json:"id"`
	Name            string       `json:"name"`
	Description     string       `json:"description,omitempty"`
	Category        string       `json:"category,omitempty"`
	Author          string       `json:"author,omitempty"`
	Versions        Versions     `json:"versions"`
	Status          ModuleStatus `json:"status"`
	NewMatchAllowed bool         `json:"newMatchAllowed"`
	MinimumPlayers  int          `json:"minimumPlayers"`
	MaximumPlayers  int          `json:"maximumPlayers"`
	AverageTimeSec  int          `json:"averageTimeSeconds"`
	Modes           []string     `json:"modes"`
	Capabilities    Capabilities `json:"capabilities"`
	RendererKey     string       `json:"rendererKey"`
	ManifestHash    string       `json:"manifestHash"`
}

func (d Descriptor) Clone() Descriptor {
	d.Modes = append([]string(nil), d.Modes...)
	return d
}

// Module is the minimum contract required for discovery and version resolution.
type Module interface {
	Descriptor() Descriptor
}

// RuntimeGame is the game-neutral execution contract implemented by production games.
type RuntimeGame interface {
	Module
	InitializeMatch(context.Context, MatchContext, MatchRequest) (GameState, error)
	InitializeParticipant(context.Context, ParticipantContext, GameState) (GameState, error)
	GenerateState(context.Context, MatchContext, GenerationRequest) (GeneratedState, error)
	ValidateAction(context.Context, ActionContext, GameState, ActionEnvelope) (ValidatedAction, error)
	ApplyAction(context.Context, ActionContext, GameState, ValidatedAction) (Transition, error)
	Snapshot(context.Context, ViewerContext, GameState) (RendererSnapshot, error)
	Completion(context.Context, MatchContext, GameState) (CompletionResult, error)
	DetermineWinner(context.Context, MatchContext, []GameState) (MatchOutcome, error)
	SerializeReplay(context.Context, ReplaySource) (ReplayMetadata, error)
	RestoreReplay(context.Context, ReplayMetadata, []ReplayEvent) (GameState, error)
	Cleanup(context.Context, MatchContext) (CleanupInstructions, error)
}

type MatchRequest struct {
	Mode          string          `json:"mode"`
	Configuration json.RawMessage `json:"configuration,omitempty"`
}

type GenerationRequest struct {
	Mode              string          `json:"mode"`
	DifficultyProfile json.RawMessage `json:"difficultyProfile"`
	SeedReference     string          `json:"seedReference"`
}

type GeneratedState struct {
	Reference string          `json:"reference"`
	Metadata  json.RawMessage `json:"metadata"`
}

type CleanupInstructions struct {
	RetainReplay bool     `json:"retainReplay"`
	ReleaseKeys  []string `json:"releaseKeys,omitempty"`
}
