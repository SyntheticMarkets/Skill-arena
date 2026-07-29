package engine

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"strings"

	"skill-arena/internal/games/interfaces"
)

const ActionArrowClick = "arrow.click"

type ArrowClick struct {
	ArrowID string `json:"arrowId"`
}

type ValidatedAction struct {
	ActionID string
	ArrowID  string
}

func ValidateAction(
	ctx context.Context,
	actionContext interfaces.ActionContext,
	state State,
	envelope interfaces.ActionEnvelope,
) (ValidatedAction, error) {
	if err := ctx.Err(); err != nil {
		return ValidatedAction{}, err
	}
	if err := state.Validate(); err != nil {
		return ValidatedAction{}, err
	}
	if state.Status != StatusActive {
		return ValidatedAction{}, fail(CodeAlreadyComplete, "Maze state is terminal")
	}
	if envelope.MatchID != state.MatchID || actionContext.MatchID != state.MatchID {
		return ValidatedAction{}, fail(CodeMatchMismatch, "action does not belong to this match")
	}
	if actionContext.ParticipantID != state.ParticipantID {
		return ValidatedAction{}, fail(CodeParticipant, "action does not belong to this participant")
	}
	if envelope.ExpectedStateVersion != int64(state.Version) ||
		actionContext.CurrentStateVersion != int64(state.Version) {
		return ValidatedAction{}, fail(CodeStateConflict, "action targets a stale Maze state")
	}
	if actionContext.CurrentSequence <= 0 ||
		actionContext.CurrentSequence <= int64(state.LastSequence) {
		return ValidatedAction{}, fail(CodeActionOutOfOrder, "action sequence is not monotonic")
	}
	if !actionContext.ServerReceivedAt.IsZero() &&
		(actionContext.ServerReceivedAt.UTC().UnixMilli() < state.StartedAtMS ||
			actionContext.ServerReceivedAt.UTC().UnixMilli() >= state.DeadlineAtMS) {
		return ValidatedAction{}, fail(CodeNotLive, "Maze action arrived outside the live window")
	}
	if strings.TrimSpace(envelope.ActionID) == "" || len(envelope.ActionID) > 256 ||
		envelope.Kind != ActionArrowClick {
		return ValidatedAction{}, fail(CodeActionUnsupported, "Maze accepts only arrow.click")
	}
	if len(envelope.Payload) == 0 || len(envelope.Payload) > 4_096 {
		return ValidatedAction{}, fail(CodeActionInvalid, "Maze action payload size is invalid")
	}
	decoder := json.NewDecoder(bytes.NewReader(envelope.Payload))
	decoder.DisallowUnknownFields()
	var click ArrowClick
	if err := decoder.Decode(&click); err != nil {
		return ValidatedAction{}, fail(CodeActionInvalid, "Maze action payload is malformed")
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		return ValidatedAction{}, fail(CodeActionInvalid, "Maze action payload contains trailing data")
	}
	if strings.TrimSpace(click.ArrowID) == "" || len(click.ArrowID) > 128 {
		return ValidatedAction{}, fail(CodeActionInvalid, "Maze arrow identity is invalid")
	}
	for _, removedID := range state.RemovedIDs {
		if removedID == click.ArrowID {
			return ValidatedAction{}, fail(CodeArrowRemoved, "the selected arrow has already left the board")
		}
	}
	for _, arrow := range state.Board.Arrows {
		if arrow.ID == click.ArrowID {
			return ValidatedAction{ActionID: envelope.ActionID, ArrowID: click.ArrowID}, nil
		}
	}
	return ValidatedAction{}, fail(CodeArrowUnknown, "the selected arrow does not exist")
}
