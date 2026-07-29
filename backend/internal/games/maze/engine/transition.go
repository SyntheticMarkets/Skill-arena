package engine

import (
	"context"
	"encoding/json"
	"sort"
	"strings"

	"skill-arena/internal/games/interfaces"
	"skill-arena/internal/games/maze/generator"
)

type Presentation struct {
	ArrowID          string          `json:"arrowId"`
	Direction        string          `json:"direction"`
	Blocked          bool            `json:"blocked"`
	BlockerID        string          `json:"blockerId,omitempty"`
	CollisionCell    *generator.Cell `json:"collisionCell,omitempty"`
	ApproachDistance int             `json:"approachDistance,omitempty"`
	EscapeDistance   int             `json:"escapeDistance,omitempty"`
	ReturnToOrigin   bool            `json:"returnToOrigin"`
	RemoveAfterExit  bool            `json:"removeAfterExit"`
}

type Result struct {
	PreviousState State
	NextState     State
	Transition    interfaces.Transition
	Progress      Progress
	ScoreInputs   ScoreInputs
	Presentation  Presentation
}

func ApplyAction(
	ctx context.Context,
	actionContext interfaces.ActionContext,
	state State,
	action ValidatedAction,
) (Result, error) {
	if err := ctx.Err(); err != nil {
		return Result{}, err
	}
	if err := state.Validate(); err != nil {
		return Result{}, err
	}
	if state.Status != StatusActive {
		return Result{}, fail(CodeAlreadyComplete, "Maze state is terminal")
	}
	if actionContext.MatchID != state.MatchID ||
		actionContext.ParticipantID != state.ParticipantID {
		return Result{}, fail(CodeMatchMismatch, "action context does not match Maze state")
	}
	if actionContext.CurrentStateVersion != int64(state.Version) {
		return Result{}, fail(CodeStateConflict, "action context targets a stale Maze state")
	}
	if actionContext.CurrentSequence <= 0 ||
		actionContext.CurrentSequence <= int64(state.LastSequence) {
		return Result{}, fail(CodeActionOutOfOrder, "action sequence is not monotonic")
	}
	if strings.TrimSpace(action.ActionID) == "" || len(action.ActionID) > 256 ||
		strings.TrimSpace(action.ArrowID) == "" || len(action.ArrowID) > 128 {
		return Result{}, fail(CodeActionInvalid, "validated Maze action is incomplete")
	}
	receivedAtMS := actionContext.ServerReceivedAt.UTC().UnixMilli()
	if receivedAtMS < state.StartedAtMS {
		return Result{}, fail(CodeActionInvalid, "authoritative action time is required")
	}
	if receivedAtMS >= state.DeadlineAtMS {
		return Result{}, fail(CodeNotLive, "Maze action arrived after the deadline")
	}
	model, err := NewCollisionModel(state.Board)
	if err != nil {
		return Result{}, err
	}
	collision, err := model.Evaluate(state.RemovedIDs, action.ArrowID)
	if err != nil {
		return Result{}, err
	}
	arrow := state.Board.Arrows[arrowIndex(state.Board, action.ArrowID)]
	next := state.Clone()
	next.JournalVersion++
	if actionContext.CurrentSequence > 0 {
		next.LastSequence = uint64(actionContext.CurrentSequence)
	}
	presentation := Presentation{
		ArrowID: action.ArrowID, Direction: arrow.Direction.String(),
	}
	code := CodeActionBlocked
	eventKind := "maze.action.blocked"
	accepted := false
	if collision.Clear {
		accepted = true
		code = CodeActionAccepted
		eventKind = "maze.action.accepted"
		next.RemovedIDs = append(next.RemovedIDs, action.ArrowID)
		sort.Strings(next.RemovedIDs)
		next.SuccessfulActions++
		next.Version++
		next.CurrentCombo++
		next.MaximumCombo = max(next.MaximumCombo, next.CurrentCombo)
		presentation.EscapeDistance = collision.EscapeDistance
		presentation.RemoveAfterExit = true
		if len(next.RemovedIDs) == len(next.Board.Arrows) {
			next.Status = StatusCompleted
			next.CompletedAtMS = receivedAtMS
		}
	} else {
		next.BlockedActions++
		presentation.Blocked = true
		presentation.BlockerID = collision.BlockerID
		collisionCell := collision.CollisionCell
		presentation.CollisionCell = &collisionCell
		presentation.ApproachDistance = max(0, collision.Distance-1)
		presentation.ReturnToOrigin = true
	}
	next.Checksum = checksum(next)
	if err := next.Validate(); err != nil {
		return Result{}, err
	}
	progress := ProgressFor(next)
	score := ScoreFor(next, receivedAtMS)
	presentationBytes, err := json.Marshal(presentation)
	if err != nil {
		return Result{}, err
	}
	progressBytes, err := json.Marshal(progress)
	if err != nil {
		return Result{}, err
	}
	eventPayload, err := json.Marshal(map[string]any{
		"actionId": action.ActionID, "arrowId": action.ArrowID,
		"code": code, "stateVersion": next.Version,
		"journalVersion": next.JournalVersion, "stateChecksum": next.Checksum,
		"occurredAtMs": receivedAtMS,
	})
	if err != nil {
		return Result{}, err
	}
	nextGeneric, err := next.Generic()
	if err != nil {
		return Result{}, err
	}
	transition := interfaces.Transition{
		Accepted: accepted, Code: code, NextState: nextGeneric,
		Events:   []interfaces.GameEvent{{Kind: eventKind, Payload: eventPayload}},
		Progress: progressBytes, Presentation: presentationBytes,
	}
	if next.Status == StatusCompleted {
		completion := interfaces.CompletionResult{Status: "complete", Reason: "puzzle_cleared"}
		transition.Completion = &completion
		scorePayload, marshalErr := json.Marshal(score)
		if marshalErr != nil {
			return Result{}, marshalErr
		}
		transition.Events = append(transition.Events,
			interfaces.GameEvent{Kind: "maze.completed", Payload: eventPayload},
			interfaces.GameEvent{Kind: "maze.score.inputs.ready", Payload: scorePayload},
		)
	}
	return Result{
		PreviousState: state.Clone(), NextState: next, Transition: transition,
		Progress: progress, ScoreInputs: score, Presentation: presentation,
	}, nil
}

func Expire(ctx context.Context, state State, serverTimeMS int64) (Result, error) {
	if err := ctx.Err(); err != nil {
		return Result{}, err
	}
	if err := state.Validate(); err != nil {
		return Result{}, err
	}
	if state.Status != StatusActive {
		return Result{}, fail(CodeAlreadyComplete, "Maze state is terminal")
	}
	if serverTimeMS < state.DeadlineAtMS {
		return Result{}, fail(CodeNotLive, "Maze deadline has not elapsed")
	}
	next := state.Clone()
	next.Status = StatusTimedOut
	next.CompletedAtMS = serverTimeMS
	next.JournalVersion++
	next.SystemTransitions++
	next.Checksum = checksum(next)
	if err := next.Validate(); err != nil {
		return Result{}, err
	}
	progress := ProgressFor(next)
	progressBytes, err := json.Marshal(progress)
	if err != nil {
		return Result{}, err
	}
	eventPayload, err := json.Marshal(map[string]any{
		"code": CodeMatchTimedOut, "stateVersion": next.Version,
		"journalVersion": next.JournalVersion, "occurredAtMs": serverTimeMS,
	})
	if err != nil {
		return Result{}, err
	}
	nextGeneric, err := next.Generic()
	if err != nil {
		return Result{}, err
	}
	completion := interfaces.CompletionResult{Status: "timeout", Reason: "server_deadline_elapsed"}
	return Result{
		PreviousState: state.Clone(), NextState: next, Progress: progress,
		ScoreInputs: ScoreFor(next, serverTimeMS),
		Transition: interfaces.Transition{
			Accepted: true, Code: CodeMatchTimedOut, NextState: nextGeneric,
			Events:   []interfaces.GameEvent{{Kind: "maze.timed_out", Payload: eventPayload}},
			Progress: progressBytes, Completion: &completion,
		},
	}, nil
}

func arrowIndex(board generator.Board, arrowID string) int {
	for index, arrow := range board.Arrows {
		if arrow.ID == arrowID {
			return index
		}
	}
	return -1
}
