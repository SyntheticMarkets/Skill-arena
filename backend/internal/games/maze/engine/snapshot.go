package engine

import (
	"context"
	"encoding/json"
	"strconv"

	"skill-arena/internal/games/interfaces"
	"skill-arena/internal/games/maze/generator"
)

type ArrowProjection struct {
	ID        string           `json:"id"`
	Cells     []generator.Cell `json:"cells"`
	Direction string           `json:"direction"`
}

type SnapshotPayload struct {
	MatchID       string            `json:"matchId"`
	ParticipantID string            `json:"participantId"`
	PuzzleID      string            `json:"puzzleId,omitempty"`
	PuzzleHash    string            `json:"puzzleHash"`
	Columns       int               `json:"columns,omitempty"`
	Rows          int               `json:"rows,omitempty"`
	Arrows        []ArrowProjection `json:"arrows,omitempty"`
	RemovedIDs    []string          `json:"removedIds,omitempty"`
	Progress      Progress          `json:"progress"`
	Status        string            `json:"status"`
	StartedAtMS   int64             `json:"startedAtMs"`
	DeadlineAtMS  int64             `json:"deadlineAtMs"`
	CompletedAtMS int64             `json:"completedAtMs"`
}

func Snapshot(
	ctx context.Context,
	viewer interfaces.ViewerContext,
	state State,
) (interfaces.RendererSnapshot, error) {
	if err := ctx.Err(); err != nil {
		return interfaces.RendererSnapshot{}, err
	}
	if err := state.Validate(); err != nil {
		return interfaces.RendererSnapshot{}, err
	}
	if viewer.MatchID != state.MatchID {
		return interfaces.RendererSnapshot{}, fail(CodeMatchMismatch, "viewer does not belong to this match")
	}
	allowedRole := map[string]bool{
		"player": true, "opponent": true, "spectator": true,
		"replay": true, "support": true, "integrity": true,
	}
	if !allowedRole[viewer.Role] {
		return interfaces.RendererSnapshot{}, fail(CodeParticipant, "viewer role is not authorized")
	}
	if viewer.Role == "player" && viewer.ParticipantID != state.ParticipantID {
		return interfaces.RendererSnapshot{}, fail(CodeParticipant, "player cannot view another participant board")
	}
	payload := SnapshotPayload{
		MatchID: state.MatchID, ParticipantID: state.ParticipantID,
		PuzzleHash: state.PuzzleHash, Progress: ProgressFor(state), Status: state.Status,
		StartedAtMS: state.StartedAtMS, DeadlineAtMS: state.DeadlineAtMS,
		CompletedAtMS: state.CompletedAtMS,
	}
	full := viewer.Role == "integrity" || viewer.Role == "support" ||
		(viewer.Role == "replay" && state.Status != StatusActive) ||
		(viewer.Role == "player" && viewer.ParticipantID == state.ParticipantID)
	if full {
		payload.PuzzleID = state.PuzzleID
		payload.Columns = state.Board.Columns
		payload.Rows = state.Board.Rows
		payload.RemovedIDs = append([]string(nil), state.RemovedIDs...)
		payload.Arrows = make([]ArrowProjection, len(state.Board.Arrows))
		for index, arrow := range state.Board.Arrows {
			payload.Arrows[index] = ArrowProjection{
				ID: arrow.ID, Cells: append([]generator.Cell(nil), arrow.Cells...),
				Direction: arrow.Direction.String(),
			}
		}
	}
	encoded, err := json.Marshal(payload)
	if err != nil {
		return interfaces.RendererSnapshot{}, err
	}
	snapshotChecksum := generator.HashFields(
		"skill-arena:maze-renderer-snapshot:v1", state.Checksum,
		viewer.Role, viewer.ParticipantID, string(encoded),
	)
	return interfaces.RendererSnapshot{
		RendererVersion: strconv.Itoa(RendererVersion),
		StateVersion:    int64(state.Version), Payload: encoded, Checksum: snapshotChecksum,
	}, nil
}
