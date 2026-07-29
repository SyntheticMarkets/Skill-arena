package interfaces

import (
	"context"
	"encoding/json"
	"time"
)

type ReplaySource struct {
	ReplayID        string        `json:"replayId"`
	MatchID         string        `json:"matchId"`
	States          []GameState   `json:"states"`
	Events          []ReplayEvent `json:"events"`
	Outcome         MatchOutcome  `json:"outcome"`
	StartedAtUnixMS int64         `json:"startedAtUnixMs"`
	EndedAtUnixMS   int64         `json:"endedAtUnixMs"`
}

type ReplayMetadata struct {
	ReplayID   string          `json:"replayId"`
	MatchID    string          `json:"matchId"`
	Versions   Versions        `json:"versions"`
	GameData   json.RawMessage `json:"gameData"`
	ReplayHash string          `json:"replayHash"`
}

type ReplayEvent struct {
	Sequence      int64           `json:"sequence"`
	StateVersion  int64           `json:"stateVersion"`
	OccurredAt    time.Time       `json:"occurredAt"`
	ParticipantID string          `json:"participantId,omitempty"`
	Kind          string          `json:"kind"`
	Payload       json.RawMessage `json:"payload"`
}

type FinalizedReplay struct {
	ReplayID      string               `json:"replayId"`
	ReplayHash    string               `json:"replayHash"`
	EventRootHash string               `json:"eventRootHash"`
	EventCount    int                  `json:"eventCount"`
	Proof         ReplayIntegrityProof `json:"proof"`
	StorageKey    string               `json:"storageKey"`
	Status        string               `json:"status"`
}

type AuthoritativeReplayRuntime interface {
	FinalizeAuthoritativeReplay(
		context.Context,
		MatchContext,
		ReplaySource,
		ReplayIntegrityService,
	) (FinalizedReplay, error)
}

type ReplayIntegrityRequest struct {
	MatchID       string
	GameID        string
	ReplayHash    string
	EventRootHash string
	EventCount    int
}

type ReplayIntegrityProof struct {
	Algorithm string `json:"algorithm"`
	KeyID     string `json:"keyId"`
	Signature string `json:"signature"`
}

type ReplayIntegrityService interface {
	SignReplayIntegrity(context.Context, ReplayIntegrityRequest) (ReplayIntegrityProof, error)
	VerifyReplayIntegrity(context.Context, ReplayIntegrityRequest, ReplayIntegrityProof) error
}
