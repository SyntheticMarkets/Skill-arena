package registry

import (
	"context"
	"testing"

	"skill-arena/internal/arena/core"
	"skill-arena/internal/games/maze"
	"skill-arena/internal/games/testarena"
	"skill-arena/internal/models"
)

func TestRegistryAcceptsMultipleManifestBackedModules(t *testing.T) {
	r := New(maze.New(), testarena.New())
	list := r.List()
	if len(list) != 2 {
		t.Fatalf("registered modules = %d, want 2", len(list))
	}
	testModule, err := r.Get(testarena.ModuleID)
	if err != nil {
		t.Fatalf("get test arena: %v", err)
	}
	if !testModule.Capabilities().Practice || testModule.Capabilities().PvP {
		t.Fatalf("test arena capabilities = %#v", testModule.Capabilities())
	}

	result, err := testModule.SubmitAction(context.Background(), core.ActionRequest{
		ActorUserID: "user-1",
		Session:     &models.GameSession{ID: "session-1", UserID: "user-1", Mode: testarena.ModuleID},
		Actions:     []core.PlayerAction{{ActionType: "test_move", TargetID: "ok", Sequence: 1}},
	})
	if err != nil {
		t.Fatalf("submit test arena action: %v", err)
	}
	if !result.Accepted || !result.Valid || len(result.Events) != 2 {
		t.Fatalf("test arena result = %#v", result)
	}
}

func TestRegistryRejectsInvalidManifest(t *testing.T) {
	err := ValidateManifest(core.Manifest{ID: "broken"})
	if err == nil {
		t.Fatal("expected invalid manifest to be rejected")
	}
}
