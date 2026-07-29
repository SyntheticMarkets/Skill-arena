package engine

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"sync"
	"testing"
	"time"

	"skill-arena/internal/games/interfaces"
	"skill-arena/internal/games/maze/generator"
)

const testStartedAtMS int64 = 1_800_000_000_000

func testBoard() generator.Board {
	return generator.Board{
		GeometryVersion: 1, RulesVersion: 1, Columns: 8, Rows: 8,
		Arrows: []generator.Arrow{
			{
				ID: "a0000", Direction: generator.DirectionRight,
				Cells: []generator.Cell{{Column: 6, Row: 2}, {Column: 7, Row: 2}},
			},
			{
				ID: "a0001", Direction: generator.DirectionRight,
				Cells: []generator.Cell{{Column: 3, Row: 2}, {Column: 4, Row: 2}},
			},
		},
	}
}

func testState(t testing.TB) State {
	t.Helper()
	state, err := NewState(StartRequest{
		MatchID: "match-1", ParticipantID: "player-1", PuzzleID: "puzzle-1",
		PuzzleHash:     generator.HashFields("test:puzzle", "one"),
		DifficultyHash: generator.HashFields("test:difficulty", "one"),
		Board:          testBoard(), MinimumActions: 2,
		StartedAtMS: testStartedAtMS, DeadlineAtMS: testStartedAtMS + 60_000,
	})
	if err != nil {
		t.Fatal(err)
	}
	return state
}

func actionContext(state State, sequence int64, offsetMS int64) interfaces.ActionContext {
	return interfaces.ActionContext{
		MatchID: state.MatchID, ParticipantID: state.ParticipantID,
		ServerReceivedAt: time.UnixMilli(state.StartedAtMS + offsetMS),
		CurrentSequence:  sequence, CurrentStateVersion: int64(state.Version),
	}
}

func actionEnvelope(state State, actionID, arrowID string) interfaces.ActionEnvelope {
	payload, _ := json.Marshal(ArrowClick{ArrowID: arrowID})
	return interfaces.ActionEnvelope{
		ActionID: actionID, MatchID: state.MatchID, Kind: ActionArrowClick,
		Payload: payload, ClientSequence: 1, ExpectedStateVersion: int64(state.Version),
	}
}

func applyClick(t testing.TB, state State, sequence int64, offsetMS int64, arrowID string) Result {
	t.Helper()
	action, err := ValidateAction(
		context.Background(), actionContext(state, sequence, offsetMS),
		state, actionEnvelope(state, "action-"+arrowID, arrowID),
	)
	if err != nil {
		t.Fatal(err)
	}
	result, err := ApplyAction(
		context.Background(), actionContext(state, sequence, offsetMS), state, action,
	)
	if err != nil {
		t.Fatal(err)
	}
	return result
}

func TestBlockedAndAcceptedTransitionsAreAuthoritativeAndImmutable(t *testing.T) {
	initial := testState(t)
	initialCopy := initial.Clone()
	blocked := applyClick(t, initial, 1, 100, "a0001")
	if blocked.Transition.Accepted || blocked.Transition.Code != CodeActionBlocked ||
		blocked.NextState.Version != initial.Version ||
		blocked.NextState.JournalVersion != 1 || blocked.NextState.BlockedActions != 1 ||
		len(blocked.NextState.RemovedIDs) != 0 ||
		blocked.Presentation.BlockerID != "a0000" ||
		blocked.Presentation.ApproachDistance != 1 ||
		!blocked.Presentation.ReturnToOrigin {
		t.Fatalf("blocked transition = %+v", blocked)
	}
	if !reflect.DeepEqual(initial, initialCopy) {
		t.Fatal("blocked transition mutated its input state")
	}

	first := applyClick(t, blocked.NextState, 2, 200, "a0000")
	if !first.Transition.Accepted || first.Transition.Code != CodeActionAccepted ||
		first.NextState.Version != 1 || first.NextState.SuccessfulActions != 1 ||
		first.NextState.CurrentCombo != 1 || first.NextState.MaximumCombo != 1 ||
		!first.Presentation.RemoveAfterExit || first.Presentation.EscapeDistance != 2 {
		t.Fatalf("first accepted transition = %+v", first)
	}
	second := applyClick(t, first.NextState, 3, 300, "a0001")
	if !second.Transition.Accepted || second.NextState.Status != StatusCompleted ||
		second.Progress.CompletionBPS != 10_000 || !second.Progress.Complete ||
		second.ScoreInputs.EfficiencyBPS != 6_666 ||
		second.Transition.Completion == nil ||
		len(second.Transition.Events) != 3 {
		t.Fatalf("completion transition = %+v", second)
	}
	completion, err := Completion(t.Context(), second.NextState)
	if err != nil || completion.Status != "complete" {
		t.Fatalf("completion=%+v error=%v", completion, err)
	}
}

func TestTransitionIsDeterministicAndConcurrentSafe(t *testing.T) {
	state := testState(t)
	action, err := ValidateAction(
		t.Context(), actionContext(state, 1, 100), state,
		actionEnvelope(state, "action-1", "a0000"),
	)
	if err != nil {
		t.Fatal(err)
	}
	expected, err := ApplyAction(t.Context(), actionContext(state, 1, 100), state, action)
	if err != nil {
		t.Fatal(err)
	}
	const workers = 64
	var wait sync.WaitGroup
	results := make(chan Result, workers)
	errs := make(chan error, workers)
	for range workers {
		wait.Add(1)
		go func() {
			defer wait.Done()
			result, applyErr := ApplyAction(
				context.Background(), actionContext(state, 1, 100), state, action,
			)
			results <- result
			errs <- applyErr
		}()
	}
	wait.Wait()
	close(results)
	close(errs)
	for err := range errs {
		if err != nil {
			t.Fatal(err)
		}
	}
	for result := range results {
		if !reflect.DeepEqual(result, expected) {
			t.Fatal("concurrent application produced a different transition")
		}
	}
}

func TestCollisionAuthorityHonorsEveryFixedDirection(t *testing.T) {
	board := generator.Board{
		GeometryVersion: 1, RulesVersion: 1, Columns: 8, Rows: 8,
		Arrows: []generator.Arrow{
			{ID: "a0000", Direction: generator.DirectionRight, Cells: []generator.Cell{{Column: 6, Row: 1}, {Column: 7, Row: 1}}},
			{ID: "a0001", Direction: generator.DirectionUp, Cells: []generator.Cell{{Column: 1, Row: 1}, {Column: 1, Row: 0}}},
			{ID: "a0002", Direction: generator.DirectionLeft, Cells: []generator.Cell{{Column: 1, Row: 6}, {Column: 0, Row: 6}}},
			{ID: "a0003", Direction: generator.DirectionDown, Cells: []generator.Cell{{Column: 6, Row: 6}, {Column: 6, Row: 7}}},
		},
	}
	model, err := NewCollisionModel(board)
	if err != nil {
		t.Fatal(err)
	}
	for _, arrow := range board.Arrows {
		collision, err := model.Evaluate(nil, arrow.ID)
		if err != nil || !collision.Clear || collision.EscapeDistance != 2 {
			t.Fatalf("arrow=%s direction=%s collision=%+v error=%v", arrow.ID, arrow.Direction, collision, err)
		}
	}
}

func TestActionValidationRejectsClientAuthorityAndStateConflicts(t *testing.T) {
	state := testState(t)
	baseContext := actionContext(state, 1, 100)
	base := actionEnvelope(state, "action-1", "a0000")
	cases := []struct {
		name     string
		context  interfaces.ActionContext
		envelope interfaces.ActionEnvelope
		code     string
	}{
		{name: "unsupported", context: baseContext, envelope: func() interfaces.ActionEnvelope {
			value := base
			value.Kind = "move"
			return value
		}(), code: CodeActionUnsupported},
		{name: "direction injection", context: baseContext, envelope: func() interfaces.ActionEnvelope {
			value := base
			value.Payload = []byte(`{"arrowId":"a0000","direction":"left"}`)
			return value
		}(), code: CodeActionInvalid},
		{name: "stale state", context: baseContext, envelope: func() interfaces.ActionEnvelope {
			value := base
			value.ExpectedStateVersion = 4
			return value
		}(), code: CodeStateConflict},
		{name: "wrong participant", context: func() interfaces.ActionContext {
			value := baseContext
			value.ParticipantID = "other"
			return value
		}(), envelope: base, code: CodeParticipant},
		{name: "unknown arrow", context: baseContext, envelope: actionEnvelope(state, "action-2", "a9999"), code: CodeArrowUnknown},
		{name: "after deadline", context: actionContext(state, 1, 60_000), envelope: base, code: CodeNotLive},
	}
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			_, err := ValidateAction(t.Context(), testCase.context, state, testCase.envelope)
			var gameErr *Error
			if !errors.As(err, &gameErr) || gameErr.Code != testCase.code {
				t.Fatalf("error=%v, want code %s", err, testCase.code)
			}
		})
	}

	first := applyClick(t, state, 1, 100, "a0000").NextState
	_, err := ValidateAction(
		t.Context(), actionContext(first, 2, 200), first,
		actionEnvelope(first, "repeat", "a0000"),
	)
	var gameErr *Error
	if !errors.As(err, &gameErr) || gameErr.Code != CodeArrowRemoved {
		t.Fatalf("removed-arrow error = %v", err)
	}
}

func TestTimeoutUsesAuthoritativeClockWithoutChangingAcceptedVersion(t *testing.T) {
	state := testState(t)
	if _, err := Expire(t.Context(), state, state.DeadlineAtMS-1); err == nil {
		t.Fatal("state expired before its deadline")
	}
	result, err := Expire(t.Context(), state, state.DeadlineAtMS)
	if err != nil {
		t.Fatal(err)
	}
	if result.NextState.Status != StatusTimedOut ||
		result.NextState.Version != state.Version ||
		result.NextState.SystemTransitions != 1 ||
		result.Transition.Completion == nil ||
		result.Transition.Completion.Status != "timeout" {
		t.Fatalf("timeout result = %+v", result)
	}
}

func TestSnapshotsEnforceViewerVisibility(t *testing.T) {
	state := testState(t)
	player, err := Snapshot(t.Context(), interfaces.ViewerContext{
		MatchID: state.MatchID, ParticipantID: state.ParticipantID, Role: "player",
	}, state)
	if err != nil {
		t.Fatal(err)
	}
	opponent, err := Snapshot(t.Context(), interfaces.ViewerContext{
		MatchID: state.MatchID, ParticipantID: "player-2", Role: "opponent",
	}, state)
	if err != nil {
		t.Fatal(err)
	}
	var playerPayload, opponentPayload SnapshotPayload
	if err := json.Unmarshal(player.Payload, &playerPayload); err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(opponent.Payload, &opponentPayload); err != nil {
		t.Fatal(err)
	}
	if len(playerPayload.Arrows) != len(state.Board.Arrows) || playerPayload.PuzzleID == "" {
		t.Fatal("player snapshot omitted the renderable board")
	}
	if len(opponentPayload.Arrows) != 0 || opponentPayload.PuzzleID != "" ||
		len(opponentPayload.RemovedIDs) != 0 {
		t.Fatal("opponent snapshot exposed private board state")
	}
}

func TestGenericStateRoundTripAndTamperRejection(t *testing.T) {
	state := testState(t)
	generic, err := state.Generic()
	if err != nil {
		t.Fatal(err)
	}
	decoded, err := DecodeState(generic)
	if err != nil || !reflect.DeepEqual(decoded, state) {
		t.Fatalf("decoded=%+v error=%v", decoded, err)
	}
	generic.Checksum = generator.HashFields("tampered", "state")
	if _, err := DecodeState(generic); err == nil {
		t.Fatal("tampered generic state was accepted")
	}
	tampered := state.Clone()
	tampered.Board.Arrows[0].Cells[0].Row = 3
	tampered.Board.Arrows[0].Cells[1].Row = 3
	tampered.Checksum = checksum(tampered)
	if err := tampered.Validate(); err == nil {
		t.Fatal("valid-looking board mutation bypassed the immutable board hash")
	}
}

func TestApplyRejectsForgedValidatedActionAndUnauthorizedViewer(t *testing.T) {
	state := testState(t)
	_, err := ApplyAction(
		t.Context(),
		interfaces.ActionContext{
			MatchID: "other", ParticipantID: state.ParticipantID,
			ServerReceivedAt: time.UnixMilli(state.StartedAtMS + 100),
			CurrentSequence:  1, CurrentStateVersion: int64(state.Version),
		},
		state,
		ValidatedAction{ActionID: "forged", ArrowID: "a0000"},
	)
	if err == nil {
		t.Fatal("forged validated action bypassed context ownership")
	}
	if _, err := Snapshot(t.Context(), interfaces.ViewerContext{
		MatchID: state.MatchID, ParticipantID: "other", Role: "player",
	}, state); err == nil {
		t.Fatal("player viewer accessed another participant's board")
	}
	if _, err := Snapshot(t.Context(), interfaces.ViewerContext{
		MatchID: state.MatchID, Role: "unknown",
	}, state); err == nil {
		t.Fatal("unknown viewer role received a snapshot")
	}
}

func TestTutorialFixturesAreCanonicalAndSolvable(t *testing.T) {
	expectedHashes := []string{
		"sha256:7f4566dfc7774a3a129416dafa43d7d2214b6636665ef43d1ea8777127817559",
		"sha256:b6b41f555089dee4d2048e92f71263741c2b3018b5e8887cc9285c41a5e8fc02",
		"sha256:de737da2765ffab5579014c97f9f4b99a8528ac208867b1c7d2480192e4389f3",
		"sha256:c1e9d6b484e1eb88f098cc3ff3971120827492dff5a30b45dfac962842b88465",
		"sha256:5df3b1cd4d6fb40fcd1c868d70a90c9602e9c9ef280cec1cb6db80051019ef28",
	}
	for level := 1; level <= 5; level++ {
		board, err := TutorialBoard(level)
		if err != nil {
			t.Fatal(err)
		}
		model, err := NewCollisionModel(board)
		if err != nil {
			t.Fatal(err)
		}
		removed := make([]string, 0, len(board.Arrows))
		for _, arrow := range board.Arrows {
			collision, err := model.Evaluate(removed, arrow.ID)
			if err != nil || !collision.Clear {
				t.Fatalf("tutorial %d arrow %s collision=%+v error=%v", level, arrow.ID, collision, err)
			}
			removed = append(removed, arrow.ID)
		}
		boardBytes, err := generator.CanonicalBoard(board)
		if err != nil {
			t.Fatal(err)
		}
		hash := generator.HashBytes("skill-arena:maze-tutorial:v1", boardBytes)
		if hash != expectedHashes[level-1] {
			t.Fatalf("tutorial %d hash = %s", level, hash)
		}
	}
}

func FuzzActionPayloadNeverIntroducesClientAuthority(f *testing.F) {
	f.Add([]byte(`{"arrowId":"a0000"}`))
	f.Add([]byte(`{"arrowId":"a0001"}`))
	f.Add([]byte(`{"arrowId":"a0000","direction":"left"}`))
	f.Add([]byte(`null`))
	state := testState(f)
	f.Fuzz(func(t *testing.T, payload []byte) {
		envelope := actionEnvelope(state, "fuzz-action", "a0000")
		envelope.Payload = payload
		validated, err := ValidateAction(
			context.Background(), actionContext(state, 1, 100), state, envelope,
		)
		if err == nil && validated.ArrowID != "a0000" && validated.ArrowID != "a0001" {
			t.Fatalf("unexpected validated arrow %q", validated.ArrowID)
		}
	})
}

func BenchmarkApplyActionStandard(b *testing.B) {
	board := dependencyChainBoard(20)
	state, err := NewState(StartRequest{
		MatchID: "benchmark-match", ParticipantID: "benchmark-player",
		PuzzleID:       "benchmark-puzzle",
		PuzzleHash:     generator.HashFields("benchmark:puzzle", "standard"),
		DifficultyHash: generator.HashFields("benchmark:difficulty", "standard"),
		Board:          board, MinimumActions: len(board.Arrows),
		StartedAtMS: testStartedAtMS, DeadlineAtMS: testStartedAtMS + 60_000,
	})
	if err != nil {
		b.Fatal(err)
	}
	action, err := ValidateAction(
		context.Background(), actionContext(state, 1, 100), state,
		actionEnvelope(state, "benchmark-action", "a0000"),
	)
	if err != nil {
		b.Fatal(err)
	}
	b.ResetTimer()
	for range b.N {
		if _, err := ApplyAction(
			context.Background(), actionContext(state, 1, 100), state, action,
		); err != nil {
			b.Fatal(err)
		}
	}
}

func dependencyChainBoard(count int) generator.Board {
	columns := count*3 + 1
	arrows := make([]generator.Arrow, count)
	for index := 0; index < count; index++ {
		head := columns - 1 - index*3
		arrows[index] = generator.Arrow{
			ID: fmt.Sprintf("a%04d", index), Direction: generator.DirectionRight,
			Cells: []generator.Cell{
				{Column: head - 1, Row: 2},
				{Column: head, Row: 2},
			},
		}
	}
	return generator.Board{
		GeometryVersion: 1, RulesVersion: 1,
		Columns: columns, Rows: 8, Arrows: arrows,
	}
}
