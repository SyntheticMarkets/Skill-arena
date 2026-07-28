package db

import (
	"context"
	"testing"

	"skill-arena/internal/games/maze"
	gamesregistry "skill-arena/internal/games/registry"
)

func TestStoreUsesInjectedGamesRegistryWithoutChangingArenaContract(t *testing.T) {
	catalog, err := gamesregistry.NewProduction()
	if err != nil {
		t.Fatalf("production games registry: %v", err)
	}
	store, err := NewWithOptions(context.Background(), Options{
		DatabaseURL:   t.TempDir(),
		Environment:   "development",
		GamesRegistry: catalog,
	})
	if err != nil {
		t.Fatalf("new store: %v", err)
	}
	t.Cleanup(func() {
		_ = store.Close(context.Background())
	})

	if store.GamesRegistry() != catalog {
		t.Fatal("store did not retain the injected games registry")
	}
	if _, err := store.ArenaRegistry().Get(maze.ModuleID); err != nil {
		t.Fatalf("existing ArenaRegistry contract cannot resolve Maze: %v", err)
	}
}
