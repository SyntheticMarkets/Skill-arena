package maze

import (
	"context"
	_ "embed"
	"encoding/json"
	"errors"
	"strconv"

	"skill-arena/internal/arena/core"
	"skill-arena/internal/arena/sdk"
	"skill-arena/internal/game"
	"skill-arena/internal/models"
)

const ModuleID = "maze_arena"

//go:embed module.json
var manifestBytes []byte

type Module struct {
	manifest core.Manifest
}

func New() Module {
	var manifest core.Manifest
	if err := json.Unmarshal(manifestBytes, &manifest); err != nil {
		panic(err)
	}
	return Module{manifest: manifest}
}

func (m Module) ID() string {
	return ModuleID
}

func (m Module) Name() string {
	return "Maze Arena"
}

func (m Module) Manifest() core.Manifest {
	return m.manifest
}

func (m Module) Capabilities() core.CapabilityFlags {
	return m.manifest.Capabilities
}

func (m Module) Metadata() core.Metadata {
	manifest := m.Manifest()
	return core.Metadata{
		ID:             manifest.ID,
		Name:           manifest.Name,
		Description:    manifest.Description,
		Category:       manifest.Category,
		Version:        manifest.Version,
		Author:         manifest.Author,
		Difficulty:     manifest.Difficulty,
		MinimumPlayers: manifest.MinimumPlayers,
		MaximumPlayers: manifest.MaximumPlayers,
		AverageTimeSec: manifest.AverageTimeSec,
		Modes:          append([]string(nil), manifest.Modes...),
		RendererKey:    manifest.RendererKey,
		Versions:       manifest.Versions,
		Capabilities:   manifest.Capabilities,
	}
}

func (m Module) CreateSession(context.Context, core.SessionRequest) (core.SessionSpec, error) {
	return core.SessionSpec{GameID: ModuleID, RulesVersion: m.manifest.Versions.Rules, GameVersion: m.manifest.Versions.Game}, nil
}

func (m Module) JoinSession(context.Context, core.SessionRequest) error {
	return nil
}

func (m Module) StartSession(ctx context.Context, session *models.GameSession) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if session == nil {
		return errors.New("session is required")
	}
	return nil
}

func (m Module) ResumeSession(ctx context.Context, req core.SessionRequest) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if req.ActorUserID == "" {
		return errors.New("actor user id is required")
	}
	return nil
}

func (m Module) ValidateAction(ctx context.Context, req core.ActionRequest) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if req.ActorUserID == "" {
		return errors.New("actor user id is required")
	}
	if req.Session == nil {
		return errors.New("session is required")
	}
	if len(req.Actions) == 0 {
		return errors.New("at least one action is required")
	}
	for _, action := range req.Actions {
		if action.ActionType != "click" && action.ActionType != "move" {
			return errors.New("unsupported maze action type")
		}
		if action.TargetID == "" {
			return errors.New("maze action target is required")
		}
	}
	return nil
}

func (m Module) SubmitAction(ctx context.Context, req core.ActionRequest) (core.ActionResult, error) {
	if err := m.ValidateAction(ctx, req); err != nil {
		return core.ActionResult{}, err
	}
	session := req.Session
	arena := sdk.NewContext(ModuleID, session.ID, req.ActorUserID)
	history := make([]models.MazeMove, 0, len(req.Actions))
	for _, action := range req.Actions {
		history = append(history, models.MazeMove{
			Direction: action.TargetID,
			Timestamp: action.ClientTime,
		})
	}

	if len(session.Lines) > 0 {
		clickIDs := make([]string, 0, len(req.Actions))
		for _, action := range req.Actions {
			clickIDs = append(clickIDs, action.TargetID)
		}
		valid, lines, clicks := game.ValidateLineClicks(session.Lines, clickIDs)
		if valid {
			arena.Emit(core.EventPuzzleSolved, map[string]string{"clicks": strconv.Itoa(len(clickIDs))})
		} else {
			arena.Emit(core.EventActionRejected, map[string]string{"reason": "route_incomplete"})
		}
		return core.ActionResult{Accepted: true, Complete: true, Valid: valid, History: history, Lines: lines, Clicks: clicks, Events: arena.Events()}, nil
	}

	directions := make([]string, 0, len(req.Actions))
	for _, action := range req.Actions {
		directions = append(directions, action.TargetID)
	}
	maze := &game.Maze{
		Width:  session.Width,
		Height: session.Height,
		Cells:  session.MazeCells,
		StartX: session.StartX,
		StartY: session.StartY,
		EndX:   session.EndX,
		EndY:   session.EndY,
	}
	valid, validatedHistory, err := game.ValidateMazeMoves(maze, directions)
	if err != nil {
		return core.ActionResult{}, err
	}
	if valid {
		arena.Emit(core.EventPuzzleSolved, map[string]string{"moves": strconv.Itoa(len(directions))})
	} else {
		arena.Emit(core.EventActionRejected, map[string]string{"reason": "route_incomplete"})
	}
	return core.ActionResult{Accepted: true, Complete: true, Valid: valid, History: validatedHistory, Lines: session.Lines, Clicks: session.Clicks, Events: arena.Events()}, nil
}

func (m Module) SyncSession(ctx context.Context, req core.SessionRequest) (core.SessionSpec, error) {
	if err := ctx.Err(); err != nil {
		return core.SessionSpec{}, err
	}
	return core.SessionSpec{GameID: ModuleID, RulesVersion: m.manifest.Versions.Rules, GameVersion: m.manifest.Versions.Game}, nil
}

func (m Module) LeaveSession(ctx context.Context, req core.SessionRequest) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if req.ActorUserID == "" {
		return errors.New("actor user id is required")
	}
	return nil
}

func (m Module) FinishSession(context.Context, *models.GameSession, core.ActionResult) error {
	return nil
}

func (m Module) Replay() core.ReplayContract {
	return core.ReplayContract{Version: m.manifest.Versions.Replay, RequiredStreams: []string{"seed", "actions", "timing"}, IntegrityRequired: true}
}

func (m Module) Statistics() core.StatisticsContract {
	return core.StatisticsContract{Metrics: []string{"completion", "duration", "accuracy", "combo", "blocked_actions"}}
}
