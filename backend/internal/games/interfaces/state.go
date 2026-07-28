package interfaces

import "encoding/json"

type GameState struct {
	SchemaVersion string          `json:"schemaVersion"`
	Version       int64           `json:"version"`
	Payload       json.RawMessage `json:"payload"`
	Checksum      string          `json:"checksum"`
}

type Transition struct {
	Accepted     bool              `json:"accepted"`
	Code         string            `json:"code"`
	NextState    GameState         `json:"nextState"`
	Events       []GameEvent       `json:"events,omitempty"`
	Progress     json.RawMessage   `json:"progress,omitempty"`
	Completion   *CompletionResult `json:"completion,omitempty"`
	Presentation json.RawMessage   `json:"presentation,omitempty"`
}

type GameEvent struct {
	Kind    string          `json:"kind"`
	Payload json.RawMessage `json:"payload"`
}

type CompletionResult struct {
	Status string `json:"status"`
	Reason string `json:"reason,omitempty"`
}

type MatchOutcome struct {
	Status    string   `json:"status"`
	WinnerIDs []string `json:"winnerIds,omitempty"`
	LoserIDs  []string `json:"loserIds,omitempty"`
	Reason    string   `json:"reason,omitempty"`
}
