package replay

import (
	"context"
	"errors"
	"strings"

	"skill-arena/internal/games/interfaces"
	"skill-arena/internal/games/maze/generator"
)

const (
	ReplayVersionLegacy = 1
	ReplayVersionEngine = 2

	EventArrowAccepted = "arrow.accepted"
	EventArrowBlocked  = "arrow.blocked"

	StatusVerified    = "verified"
	StatusUnderReview = "under_review"
)

type Versions struct {
	GameVersion        string               `json:"gameVersion"`
	ProtocolVersion    int                  `json:"protocolVersion"`
	ReplayVersion      int                  `json:"replayVersion"`
	RendererVersion    int                  `json:"rendererVersion"`
	StateSchemaVersion int                  `json:"stateSchemaVersion"`
	Generator          generator.VersionKey `json:"generator"`
}

func (v Versions) Validate() error {
	if strings.TrimSpace(v.GameVersion) == "" || v.ProtocolVersion <= 0 ||
		(v.ReplayVersion != ReplayVersionLegacy && v.ReplayVersion != ReplayVersionEngine) ||
		v.RendererVersion <= 0 || v.StateSchemaVersion <= 0 {
		return errors.New("replay version tuple is incomplete")
	}
	return v.Generator.Validate()
}

type Genesis struct {
	ReplayID        string   `json:"replayId"`
	MatchID         string   `json:"matchId"`
	PuzzleID        string   `json:"puzzleId"`
	GameID          string   `json:"gameId"`
	Versions        Versions `json:"versions"`
	SeedReference   string   `json:"seedReference"`
	SeedHash        string   `json:"seedHash"`
	DifficultyID    string   `json:"difficultyProfileId"`
	DifficultyHash  string   `json:"difficultyProfileHash"`
	AnalysisHash    string   `json:"analysisHash"`
	GenerationHash  string   `json:"generationHash"`
	PuzzleHash      string   `json:"puzzleHash"`
	ValidationHash  string   `json:"validationHash"`
	SolutionHash    string   `json:"solutionHash"`
	MinimumActions  int      `json:"minimumActions"`
	CreatedAtUnixMS int64    `json:"createdAtUnixMs"`
	GenesisHash     string   `json:"genesisHash"`
}

type GenesisRequest struct {
	ReplayID        string
	MatchID         string
	PuzzleID        string
	Versions        Versions
	CreatedAtUnixMS int64
}

type EventDraft struct {
	Sequence      uint64 `json:"sequence"`
	ParticipantID string `json:"participantId"`
	OffsetMS      int64  `json:"offsetMs"`
	Kind          string `json:"kind"`
	ArrowID       string `json:"arrowId"`
	Code          string `json:"code"`
}

type Event struct {
	EventDraft
	StateVersion      uint64         `json:"stateVersion"`
	BlockerID         string         `json:"blockerId,omitempty"`
	CollisionCell     generator.Cell `json:"collisionCell"`
	CollisionDistance int            `json:"collisionDistance,omitempty"`
	EscapeDistance    int            `json:"escapeDistance,omitempty"`
	StateChecksum     string         `json:"stateChecksum"`
	PreviousHash      string         `json:"previousHash"`
	IntegrityHash     string         `json:"integrityHash"`
}

type ParticipantResult struct {
	ParticipantID     string `json:"participantId"`
	StateVersion      uint64 `json:"stateVersion"`
	SuccessfulActions int    `json:"successfulActions"`
	BlockedActions    int    `json:"blockedActions"`
	CurrentCombo      int    `json:"currentCombo"`
	MaximumCombo      int    `json:"maximumCombo"`
	Completed         bool   `json:"completed"`
	CompletedAtMS     int64  `json:"completedAtMs,omitempty"`
	StateChecksum     string `json:"stateChecksum"`
}

type Outcome struct {
	Status    string   `json:"status"`
	WinnerIDs []string `json:"winnerIds,omitempty"`
	LoserIDs  []string `json:"loserIds,omitempty"`
	Reason    string   `json:"reason,omitempty"`
}

type SealRequest struct {
	Genesis         Genesis
	ParticipantIDs  []string
	Events          []EventDraft
	Outcome         Outcome
	StartedAtUnixMS int64
	EndedAtUnixMS   int64
}

type Artifact struct {
	Genesis         Genesis                         `json:"genesis"`
	Events          []Event                         `json:"events"`
	Participants    []ParticipantResult             `json:"participants"`
	Outcome         Outcome                         `json:"outcome"`
	StartedAtUnixMS int64                           `json:"startedAtUnixMs"`
	EndedAtUnixMS   int64                           `json:"endedAtUnixMs"`
	EventRootHash   string                          `json:"eventRootHash"`
	ReplayHash      string                          `json:"replayHash"`
	Proof           interfaces.ReplayIntegrityProof `json:"proof"`
}

type VerificationReport struct {
	Status        string              `json:"status"`
	Verified      bool                `json:"verified"`
	ReplayHash    string              `json:"replayHash"`
	EventRootHash string              `json:"eventRootHash"`
	Participants  []ParticipantResult `json:"participants,omitempty"`
	Issues        []string            `json:"issues,omitempty"`
}

type Repository interface {
	Save(context.Context, Artifact) error
	Load(context.Context, string) (Artifact, error)
}
