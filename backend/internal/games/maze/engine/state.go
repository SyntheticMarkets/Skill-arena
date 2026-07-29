package engine

import (
	"encoding/json"
	"errors"
	"io"
	"sort"
	"strconv"
	"strings"

	"skill-arena/internal/games/interfaces"
	"skill-arena/internal/games/maze/generator"
)

const (
	StateSchemaVersion = 1
	RendererVersion    = 1

	StatusActive    = "active"
	StatusCompleted = "completed"
	StatusTimedOut  = "timed_out"
)

type StartRequest struct {
	MatchID        string
	ParticipantID  string
	PuzzleID       string
	PuzzleHash     string
	DifficultyHash string
	Board          generator.Board
	MinimumActions int
	StartedAtMS    int64
	DeadlineAtMS   int64
}

type State struct {
	MatchID           string          `json:"matchId"`
	ParticipantID     string          `json:"participantId"`
	PuzzleID          string          `json:"puzzleId"`
	PuzzleHash        string          `json:"puzzleHash"`
	BoardHash         string          `json:"boardHash"`
	DifficultyHash    string          `json:"difficultyHash"`
	Board             generator.Board `json:"board"`
	SchemaVersion     int             `json:"schemaVersion"`
	Version           uint64          `json:"version"`
	JournalVersion    uint64          `json:"journalVersion"`
	SystemTransitions int             `json:"systemTransitions"`
	RemovedIDs        []string        `json:"removedIds"`
	SuccessfulActions int             `json:"successfulActions"`
	BlockedActions    int             `json:"blockedActions"`
	CurrentCombo      int             `json:"currentCombo"`
	MaximumCombo      int             `json:"maximumCombo"`
	MinimumActions    int             `json:"minimumActions"`
	LastSequence      uint64          `json:"lastSequence"`
	StartedAtMS       int64           `json:"startedAtMs"`
	DeadlineAtMS      int64           `json:"deadlineAtMs"`
	CompletedAtMS     int64           `json:"completedAtMs"`
	Status            string          `json:"status"`
	Checksum          string          `json:"checksum"`
}

func NewState(request StartRequest) (State, error) {
	boardBytes, err := generator.CanonicalBoard(request.Board)
	if err != nil {
		return State{}, err
	}
	state := State{
		MatchID: request.MatchID, ParticipantID: request.ParticipantID,
		PuzzleID: request.PuzzleID, PuzzleHash: request.PuzzleHash,
		BoardHash:      generator.HashBytes("skill-arena:maze-live-board:v1", boardBytes),
		DifficultyHash: request.DifficultyHash, Board: request.Board.Clone(),
		SchemaVersion: StateSchemaVersion, MinimumActions: request.MinimumActions,
		StartedAtMS: request.StartedAtMS, DeadlineAtMS: request.DeadlineAtMS,
		CompletedAtMS: -1, Status: StatusActive,
	}
	state.Checksum = checksum(state)
	if err := state.Validate(); err != nil {
		return State{}, err
	}
	return state, nil
}

func (s State) Clone() State {
	s.Board = s.Board.Clone()
	s.RemovedIDs = append([]string(nil), s.RemovedIDs...)
	return s
}

func (s State) Validate() error {
	if strings.TrimSpace(s.MatchID) == "" || strings.TrimSpace(s.ParticipantID) == "" ||
		strings.TrimSpace(s.PuzzleID) == "" || !generator.ValidHash(s.PuzzleHash) ||
		!generator.ValidHash(s.BoardHash) || !generator.ValidHash(s.DifficultyHash) ||
		s.SchemaVersion != StateSchemaVersion {
		return fail(CodeStateInvalid, "maze state identity or version is invalid")
	}
	for _, id := range []string{s.MatchID, s.ParticipantID, s.PuzzleID} {
		if len(id) > 256 {
			return fail(CodeStateInvalid, "maze state identifier exceeds production bounds")
		}
	}
	if err := generator.ValidateGeometry(s.Board); err != nil {
		return fail(CodeStateInvalid, err.Error())
	}
	boardBytes, err := generator.CanonicalBoard(s.Board)
	if err != nil {
		return fail(CodeStateInvalid, err.Error())
	}
	if generator.HashBytes("skill-arena:maze-live-board:v1", boardBytes) != s.BoardHash {
		return fail(CodeStateInvalid, "maze board does not match its integrity hash")
	}
	if s.MinimumActions != len(s.Board.Arrows) || s.StartedAtMS <= 0 ||
		s.DeadlineAtMS <= s.StartedAtMS || s.CompletedAtMS < -1 {
		return fail(CodeStateInvalid, "maze timing or action bounds are invalid")
	}
	if s.Version != uint64(s.SuccessfulActions) ||
		s.JournalVersion != uint64(s.SuccessfulActions+s.BlockedActions+s.SystemTransitions) ||
		s.SuccessfulActions != len(s.RemovedIDs) ||
		s.CurrentCombo < 0 || s.MaximumCombo < s.CurrentCombo {
		return fail(CodeStateInvalid, "maze progress counters are inconsistent")
	}
	if s.SuccessfulActions+s.BlockedActions > 0 && s.LastSequence == 0 {
		return fail(CodeStateInvalid, "maze action state is missing its server sequence")
	}
	known := make(map[string]bool, len(s.Board.Arrows))
	for _, arrow := range s.Board.Arrows {
		known[arrow.ID] = true
	}
	for index, id := range s.RemovedIDs {
		if !known[id] || (index > 0 && s.RemovedIDs[index-1] >= id) {
			return fail(CodeStateInvalid, "removed-arrow state is not canonical")
		}
	}
	switch s.Status {
	case StatusActive:
		if s.CompletedAtMS != -1 || len(s.RemovedIDs) == len(s.Board.Arrows) {
			return fail(CodeStateInvalid, "active state has a terminal marker")
		}
	case StatusCompleted:
		if s.CompletedAtMS < s.StartedAtMS || s.CompletedAtMS > s.DeadlineAtMS ||
			len(s.RemovedIDs) != len(s.Board.Arrows) {
			return fail(CodeStateInvalid, "completed state is incomplete")
		}
	case StatusTimedOut:
		if s.CompletedAtMS < s.DeadlineAtMS || len(s.RemovedIDs) == len(s.Board.Arrows) {
			return fail(CodeStateInvalid, "timed-out state is invalid")
		}
	default:
		return fail(CodeStateInvalid, "maze state status is invalid")
	}
	if s.Checksum != checksum(s) {
		return fail(CodeStateInvalid, "maze state checksum is invalid")
	}
	return nil
}

func (s State) Generic() (interfaces.GameState, error) {
	if err := s.Validate(); err != nil {
		return interfaces.GameState{}, err
	}
	payload, err := json.Marshal(s)
	if err != nil {
		return interfaces.GameState{}, err
	}
	return interfaces.GameState{
		SchemaVersion: strconv.Itoa(s.SchemaVersion), Version: int64(s.Version),
		Payload: payload, Checksum: s.Checksum,
	}, nil
}

func checksum(state State) string {
	removed := append([]string(nil), state.RemovedIDs...)
	sort.Strings(removed)
	return generator.HashFields(
		"skill-arena:maze-live-state:v1",
		state.MatchID, state.ParticipantID, state.PuzzleID, state.PuzzleHash, state.BoardHash,
		state.DifficultyHash, strconv.Itoa(state.SchemaVersion),
		strconv.FormatUint(state.Version, 10),
		strconv.FormatUint(state.JournalVersion, 10),
		strconv.Itoa(state.SystemTransitions),
		strings.Join(removed, "\x1f"),
		strconv.Itoa(state.SuccessfulActions), strconv.Itoa(state.BlockedActions),
		strconv.Itoa(state.CurrentCombo), strconv.Itoa(state.MaximumCombo),
		strconv.Itoa(state.MinimumActions), strconv.FormatUint(state.LastSequence, 10),
		strconv.FormatInt(state.StartedAtMS, 10), strconv.FormatInt(state.DeadlineAtMS, 10),
		strconv.FormatInt(state.CompletedAtMS, 10), state.Status,
	)
}

func DecodeState(gameState interfaces.GameState) (State, error) {
	if gameState.SchemaVersion != strconv.Itoa(StateSchemaVersion) {
		return State{}, fail(CodeStateInvalid, "unsupported Maze state schema")
	}
	decoder := json.NewDecoder(strings.NewReader(string(gameState.Payload)))
	decoder.DisallowUnknownFields()
	var state State
	if err := decoder.Decode(&state); err != nil {
		return State{}, fail(CodeStateInvalid, "Maze state payload is malformed")
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		return State{}, errors.New("maze state contains trailing data")
	}
	if state.Checksum != gameState.Checksum || int64(state.Version) != gameState.Version {
		return State{}, fail(CodeStateInvalid, "generic Maze state envelope disagrees with payload")
	}
	return state, state.Validate()
}
