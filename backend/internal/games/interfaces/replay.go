package interfaces

import (
	"encoding/json"
	"time"
)

type ReplaySource struct {
	MatchID string        `json:"matchId"`
	States  []GameState   `json:"states"`
	Events  []ReplayEvent `json:"events"`
}

type ReplayMetadata struct {
	ReplayID   string          `json:"replayId"`
	MatchID    string          `json:"matchId"`
	Versions   Versions        `json:"versions"`
	GameData   json.RawMessage `json:"gameData"`
	ReplayHash string          `json:"replayHash"`
}

type ReplayEvent struct {
	Sequence     int64           `json:"sequence"`
	StateVersion int64           `json:"stateVersion"`
	OccurredAt   time.Time       `json:"occurredAt"`
	Kind         string          `json:"kind"`
	Payload      json.RawMessage `json:"payload"`
}
