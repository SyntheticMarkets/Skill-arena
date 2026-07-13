package testarena

import (
	"context"
	_ "embed"
	"encoding/json"
	"errors"

	"skill-arena/internal/arena/core"
	"skill-arena/internal/arena/sdk"
	"skill-arena/internal/models"
)

const ModuleID = "test_arena"

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

func (m Module) ID() string { return ModuleID }

func (m Module) Name() string { return m.manifest.Name }

func (m Module) Manifest() core.Manifest { return m.manifest }

func (m Module) Capabilities() core.CapabilityFlags { return m.manifest.Capabilities }

func (m Module) Metadata() core.Metadata {
	return core.Metadata{
		ID:             m.manifest.ID,
		Name:           m.manifest.Name,
		Description:    m.manifest.Description,
		Category:       m.manifest.Category,
		Version:        m.manifest.Version,
		Author:         m.manifest.Author,
		Difficulty:     m.manifest.Difficulty,
		MinimumPlayers: m.manifest.MinimumPlayers,
		MaximumPlayers: m.manifest.MaximumPlayers,
		AverageTimeSec: m.manifest.AverageTimeSec,
		Modes:          append([]string(nil), m.manifest.Modes...),
		RendererKey:    m.manifest.RendererKey,
		Versions:       m.manifest.Versions,
		Capabilities:   m.manifest.Capabilities,
	}
}

func (m Module) CreateSession(context.Context, core.SessionRequest) (core.SessionSpec, error) {
	return core.SessionSpec{GameID: ModuleID, RulesVersion: m.manifest.Versions.Rules, GameVersion: m.manifest.Versions.Game}, nil
}

func (m Module) JoinSession(context.Context, core.SessionRequest) error { return nil }

func (m Module) StartSession(context.Context, *models.GameSession) error { return nil }

func (m Module) ResumeSession(context.Context, core.SessionRequest) error { return nil }

func (m Module) ValidateAction(ctx context.Context, req core.ActionRequest) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if req.ActorUserID == "" {
		return errors.New("actor user id is required")
	}
	if len(req.Actions) == 0 {
		return errors.New("test arena action is required")
	}
	for _, action := range req.Actions {
		if action.ActionType != "test_move" {
			return errors.New("unsupported test arena action")
		}
	}
	return nil
}

func (m Module) SubmitAction(ctx context.Context, req core.ActionRequest) (core.ActionResult, error) {
	if err := m.ValidateAction(ctx, req); err != nil {
		return core.ActionResult{}, err
	}
	sessionID := ""
	if req.Session != nil {
		sessionID = req.Session.ID
	}
	arena := sdk.NewContext(ModuleID, sessionID, req.ActorUserID)
	arena.Emit(core.EventActionAccepted, map[string]string{"module": ModuleID})
	arena.Emit(core.EventPuzzleSolved, map[string]string{"module": ModuleID})
	return core.ActionResult{Accepted: true, Complete: true, Valid: true, Events: arena.Events()}, nil
}

func (m Module) SyncSession(context.Context, core.SessionRequest) (core.SessionSpec, error) {
	return core.SessionSpec{GameID: ModuleID, RulesVersion: m.manifest.Versions.Rules, GameVersion: m.manifest.Versions.Game}, nil
}

func (m Module) LeaveSession(context.Context, core.SessionRequest) error { return nil }

func (m Module) FinishSession(context.Context, *models.GameSession, core.ActionResult) error {
	return nil
}

func (m Module) Replay() core.ReplayContract {
	return core.ReplayContract{Version: m.manifest.Versions.Replay, RequiredStreams: []string{"actions"}, IntegrityRequired: true}
}

func (m Module) Statistics() core.StatisticsContract {
	return core.StatisticsContract{Metrics: []string{"accepted_actions"}}
}
