package replay

import (
	"context"
	"errors"
	"reflect"
	"sort"
	"strings"

	"skill-arena/internal/games/maze/engine"
	"skill-arena/internal/games/maze/generator"
	"skill-arena/internal/games/maze/solver"
)

type participantState struct {
	version      uint64
	successful   int
	blocked      int
	currentCombo int
	maximumCombo int
	complete     bool
	completedAt  int64
	checksum     string
}

func (s *Service) reconstructEvents(
	ctx context.Context,
	genesis Genesis,
	board generator.Board,
	participantIDs []string,
	drafts []EventDraft,
) ([]Event, []ParticipantResult, string, error) {
	participants, err := canonicalParticipants(participantIDs)
	if err != nil {
		return nil, nil, "", err
	}
	known := make(map[string]bool, len(participants))
	actions := make(map[string][]solver.SimulationAction, len(participants))
	positions := make(map[string][]int, len(participants))
	for _, participantID := range participants {
		known[participantID] = true
	}
	var previousOffset int64
	for index, draft := range drafts {
		if err := ctx.Err(); err != nil {
			return nil, nil, "", err
		}
		if draft.Sequence != uint64(index+1) || !known[draft.ParticipantID] ||
			strings.TrimSpace(draft.ArrowID) == "" || draft.OffsetMS < 0 ||
			(index > 0 && draft.OffsetMS < previousOffset) ||
			(draft.Kind != EventArrowAccepted && draft.Kind != EventArrowBlocked) ||
			(draft.Kind == EventArrowAccepted && draft.Code != "ACTION_ACCEPTED") ||
			(draft.Kind == EventArrowBlocked && draft.Code != "ACTION_BLOCKED") {
			return nil, nil, "", errors.New("replay event order or schema is invalid")
		}
		previousOffset = draft.OffsetMS
		actions[draft.ParticipantID] = append(actions[draft.ParticipantID], solver.SimulationAction{
			ArrowID: draft.ArrowID, Accepted: draft.Kind == EventArrowAccepted,
		})
		positions[draft.ParticipantID] = append(positions[draft.ParticipantID], index)
	}
	simulationSteps := make([]solver.SimulationStep, len(drafts))
	for _, participantID := range participants {
		simulation, err := s.solver.Simulate(ctx, board, actions[participantID])
		if err != nil {
			return nil, nil, "", err
		}
		for index, step := range simulation.Steps {
			simulationSteps[positions[participantID][index]] = step
		}
	}

	states := make(map[string]*participantState, len(participants))
	for _, participantID := range participants {
		states[participantID] = &participantState{completedAt: -1}
	}
	events := make([]Event, len(drafts))
	previousHash := genesis.GenesisHash
	for index, draft := range drafts {
		step := simulationSteps[index]
		state := states[draft.ParticipantID]
		if step.Accepted {
			state.version++
			state.successful++
			state.currentCombo++
			state.maximumCombo = max(state.maximumCombo, state.currentCombo)
		} else {
			state.blocked++
		}
		if step.Complete && !state.complete {
			state.complete = true
			state.completedAt = draft.OffsetMS
		}
		state.checksum = replayStateChecksum(
			genesis, draft.ParticipantID, state.version, draft.Sequence,
			step.RemovedIDs, state,
		)
		event := Event{
			EventDraft: draft, StateVersion: state.version,
			BlockerID: step.BlockerID, CollisionCell: step.CollisionCell,
			CollisionDistance: step.CollisionDistance, EscapeDistance: step.EscapeDistance,
			StateChecksum: state.checksum, PreviousHash: previousHash,
		}
		event.IntegrityHash = eventHash(genesis, event)
		events[index] = event
		previousHash = event.IntegrityHash
	}
	if len(events) == 0 {
		previousHash = generator.HashBytes(
			"skill-arena:maze-replay-empty-events:v1", []byte(genesis.GenesisHash),
		)
	}
	results := make([]ParticipantResult, 0, len(participants))
	for _, participantID := range participants {
		state := states[participantID]
		if state.checksum == "" {
			state.checksum = replayStateChecksum(genesis, participantID, 0, 0, nil, state)
		}
		result := ParticipantResult{
			ParticipantID: participantID, StateVersion: state.version,
			SuccessfulActions: state.successful, BlockedActions: state.blocked,
			Completed: state.complete, CompletedAtMS: state.completedAt,
			StateChecksum: state.checksum,
		}
		if genesis.Versions.ReplayVersion >= ReplayVersionEngine {
			result.CurrentCombo = state.currentCombo
			result.MaximumCombo = state.maximumCombo
		}
		results = append(results, result)
	}
	return events, results, previousHash, nil
}

func replayStateChecksum(
	genesis Genesis,
	participantID string,
	stateVersion uint64,
	sequence uint64,
	removedIDs []string,
	state *participantState,
) string {
	if genesis.Versions.ReplayVersion == ReplayVersionLegacy {
		return legacyStateChecksum(
			genesis, participantID, stateVersion, sequence, removedIDs,
			state.successful, state.blocked, state.complete,
		)
	}
	return engine.ReplayStateChecksum(
		genesis.PuzzleHash, participantID, genesis.Versions.StateSchemaVersion,
		stateVersion, sequence, removedIDs, state.successful, state.blocked,
		state.currentCombo, state.maximumCombo, state.complete,
	)
}

func canonicalParticipants(participantIDs []string) ([]string, error) {
	if len(participantIDs) < 1 || len(participantIDs) > 2 {
		return nil, errors.New("replay must contain one or two participants")
	}
	result := append([]string(nil), participantIDs...)
	sort.Strings(result)
	for index, id := range result {
		if strings.TrimSpace(id) == "" || len(id) > maxIdentitySize ||
			(index > 0 && result[index-1] == id) {
			return nil, errors.New("replay participant identity is invalid")
		}
	}
	return result, nil
}

func eventsEqual(first, second []Event) bool {
	return reflect.DeepEqual(first, second)
}

func participantsEqual(first, second []ParticipantResult) bool {
	left := append([]ParticipantResult(nil), first...)
	right := append([]ParticipantResult(nil), second...)
	sort.Slice(left, func(i, j int) bool { return left[i].ParticipantID < left[j].ParticipantID })
	sort.Slice(right, func(i, j int) bool { return right[i].ParticipantID < right[j].ParticipantID })
	return reflect.DeepEqual(left, right)
}
