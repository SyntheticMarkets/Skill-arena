package engine

import (
	"sort"
	"strconv"
	"strings"

	"skill-arena/internal/games/maze/generator"
)

type ReplayState struct {
	ParticipantID     string
	StateVersion      uint64
	SuccessfulActions int
	BlockedActions    int
	CurrentCombo      int
	MaximumCombo      int
	Completed         bool
	RemovedIDs        []string
	StateChecksum     string
}

func ReplayProjection(state State, sequence uint64) ReplayState {
	projection := ReplayState{
		ParticipantID: state.ParticipantID, StateVersion: state.Version,
		SuccessfulActions: state.SuccessfulActions, BlockedActions: state.BlockedActions,
		CurrentCombo: state.CurrentCombo, MaximumCombo: state.MaximumCombo,
		Completed:  state.Status == StatusCompleted,
		RemovedIDs: append([]string(nil), state.RemovedIDs...),
	}
	projection.StateChecksum = ReplayStateChecksum(
		state.PuzzleHash, state.ParticipantID, state.SchemaVersion,
		state.Version, sequence, state.RemovedIDs, state.SuccessfulActions,
		state.BlockedActions, state.CurrentCombo, state.MaximumCombo,
		projection.Completed,
	)
	return projection
}

func ReplayStateChecksum(
	puzzleHash string,
	participantID string,
	stateSchemaVersion int,
	stateVersion uint64,
	sequence uint64,
	removedIDs []string,
	successful int,
	blocked int,
	currentCombo int,
	maximumCombo int,
	complete bool,
) string {
	removed := append([]string(nil), removedIDs...)
	sort.Strings(removed)
	return generator.HashFields(
		"skill-arena:maze-replay-state:v2",
		puzzleHash, participantID, strconv.Itoa(stateSchemaVersion),
		strconv.FormatUint(stateVersion, 10), strconv.FormatUint(sequence, 10),
		strings.Join(removed, "\x1f"), strconv.Itoa(successful), strconv.Itoa(blocked),
		strconv.Itoa(currentCombo), strconv.Itoa(maximumCombo), strconv.FormatBool(complete),
	)
}
