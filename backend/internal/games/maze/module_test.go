package maze

import (
	"context"
	"testing"
	"time"

	"skill-arena/internal/arena/core"
	"skill-arena/internal/game"
	"skill-arena/internal/models"
)

func TestModuleSubmitActionUsesIntentOnly(t *testing.T) {
	module := New()
	lines := game.GenerateLinePuzzle("arena-core-test", 24, 2)
	solution, ok := game.SolveLinePuzzle(lines)
	if !ok {
		t.Fatal("generated puzzle is not solvable")
	}
	session := &models.GameSession{ID: "session-1", UserID: "user-1", Lines: lines}
	actions := make([]core.PlayerAction, 0, len(solution))
	for _, lineID := range solution {
		actions = append(actions, core.PlayerAction{ActionType: "click", TargetID: lineID, ClientTime: time.Now().UTC()})
	}

	result, err := module.SubmitAction(context.Background(), core.ActionRequest{
		ActorUserID: "user-1",
		Session:     session,
		Actions:     actions,
	})
	if err != nil {
		t.Fatalf("submit action: %v", err)
	}
	if !result.Accepted || !result.Complete || !result.Valid {
		t.Fatalf("result = accepted %v complete %v valid %v, want all true", result.Accepted, result.Complete, result.Valid)
	}
	if len(result.Events) == 0 || result.Events[0].Type != core.EventPuzzleSolved {
		t.Fatalf("events = %#v, want puzzle solved event", result.Events)
	}
}

func TestModuleRejectsUnsupportedAction(t *testing.T) {
	module := New()
	_, err := module.SubmitAction(context.Background(), core.ActionRequest{
		ActorUserID: "user-1",
		Session:     &models.GameSession{ID: "session-1", UserID: "user-1"},
		Actions:     []core.PlayerAction{{ActionType: "set_score", TargetID: "999999"}},
	})
	if err == nil {
		t.Fatal("expected unsupported client action to be rejected")
	}
}
