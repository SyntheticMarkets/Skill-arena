package replay

import (
	"context"
	"fmt"
	"testing"
	"time"

	"skill-arena/internal/games/interfaces"
	"skill-arena/internal/games/maze/engine"
)

func TestReplayReconstructsLiveEngineState(t *testing.T) {
	fixture := newFixtureWithReplayVersion(t, ReplayVersionEngine)
	solution, err := fixture.solver.Solve(
		t.Context(), fixture.qualified.Candidate.Board, false,
	)
	if err != nil {
		t.Fatal(err)
	}
	state, err := engine.NewState(engine.StartRequest{
		MatchID: fixture.genesis.MatchID, ParticipantID: "player-1",
		PuzzleID: fixture.genesis.PuzzleID, PuzzleHash: fixture.genesis.PuzzleHash,
		DifficultyHash: fixture.genesis.DifficultyHash,
		Board:          fixture.qualified.Candidate.Board,
		MinimumActions: fixture.genesis.MinimumActions,
		StartedAtMS:    fixture.genesis.CreatedAtUnixMS,
		DeadlineAtMS:   fixture.genesis.CreatedAtUnixMS + 60_000,
	})
	if err != nil {
		t.Fatal(err)
	}
	model, err := engine.NewCollisionModel(fixture.qualified.Candidate.Board)
	if err != nil {
		t.Fatal(err)
	}
	blockedArrowID := ""
	for _, arrow := range fixture.qualified.Candidate.Board.Arrows {
		collision, evaluateErr := model.Evaluate(nil, arrow.ID)
		if evaluateErr != nil {
			t.Fatal(evaluateErr)
		}
		if !collision.Clear {
			blockedArrowID = arrow.ID
			break
		}
	}
	if blockedArrowID == "" {
		t.Fatal("qualified competitive fixture contains no initially blocked action")
	}
	blockedPayload := []byte(fmt.Sprintf(`{"arrowId":%q}`, blockedArrowID))
	blockedEnvelope := interfaces.ActionEnvelope{
		ActionID: "action-blocked", MatchID: state.MatchID,
		Kind: engine.ActionArrowClick, Payload: blockedPayload,
		ExpectedStateVersion: int64(state.Version),
	}
	blockedContext := interfaces.ActionContext{
		MatchID: state.MatchID, ParticipantID: state.ParticipantID,
		ServerReceivedAt: time.UnixMilli(state.StartedAtMS + 10),
		CurrentSequence:  1, CurrentStateVersion: int64(state.Version),
	}
	blockedAction, err := engine.ValidateAction(t.Context(), blockedContext, state, blockedEnvelope)
	if err != nil {
		t.Fatal(err)
	}
	blockedResult, err := engine.ApplyAction(t.Context(), blockedContext, state, blockedAction)
	if err != nil {
		t.Fatal(err)
	}
	if blockedResult.Transition.Accepted {
		t.Fatal("known blocked action was accepted by the live engine")
	}
	state = blockedResult.NextState
	drafts := []EventDraft{{
		Sequence: 1, ParticipantID: state.ParticipantID, OffsetMS: 10,
		Kind: EventArrowBlocked, ArrowID: blockedArrowID, Code: "ACTION_BLOCKED",
	}}
	for index, step := range solution.Steps {
		sequence := index + 2
		offset := int64(sequence * 25)
		payload := []byte(fmt.Sprintf(`{"arrowId":%q}`, step.ArrowID))
		envelope := interfaces.ActionEnvelope{
			ActionID: fmt.Sprintf("action-%d", sequence), MatchID: state.MatchID,
			Kind: engine.ActionArrowClick, Payload: payload,
			ExpectedStateVersion: int64(state.Version),
		}
		actionContext := interfaces.ActionContext{
			MatchID: state.MatchID, ParticipantID: state.ParticipantID,
			ServerReceivedAt: time.UnixMilli(state.StartedAtMS + offset),
			CurrentSequence:  int64(sequence), CurrentStateVersion: int64(state.Version),
		}
		validated, err := engine.ValidateAction(t.Context(), actionContext, state, envelope)
		if err != nil {
			t.Fatal(err)
		}
		result, err := engine.ApplyAction(t.Context(), actionContext, state, validated)
		if err != nil {
			t.Fatal(err)
		}
		state = result.NextState
		drafts = append(drafts, EventDraft{
			Sequence: uint64(sequence), ParticipantID: state.ParticipantID,
			OffsetMS: offset, Kind: EventArrowAccepted,
			ArrowID: step.ArrowID, Code: "ACTION_ACCEPTED",
		})
	}
	artifact, err := fixture.service.Seal(context.Background(), SealRequest{
		Genesis: fixture.genesis, ParticipantIDs: []string{state.ParticipantID},
		Events:          drafts,
		Outcome:         Outcome{Status: "completed", WinnerIDs: []string{state.ParticipantID}},
		StartedAtUnixMS: state.StartedAtMS, EndedAtUnixMS: state.CompletedAtMS,
	})
	if err != nil {
		t.Fatal(err)
	}
	projection := engine.ReplayProjection(state, uint64(len(drafts)))
	participant := artifact.Participants[0]
	if participant.StateVersion != projection.StateVersion ||
		participant.SuccessfulActions != projection.SuccessfulActions ||
		participant.BlockedActions != projection.BlockedActions ||
		participant.CurrentCombo != projection.CurrentCombo ||
		participant.MaximumCombo != projection.MaximumCombo ||
		participant.Completed != projection.Completed ||
		participant.StateChecksum != projection.StateChecksum {
		t.Fatalf("replay participant=%+v live projection=%+v", participant, projection)
	}
	report, err := fixture.service.Verify(t.Context(), artifact)
	if err != nil || !report.Verified {
		t.Fatalf("report=%+v error=%v", report, err)
	}
}
