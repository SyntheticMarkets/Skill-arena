package db

import (
	"context"
	"testing"

	"skill-arena/internal/games/interfaces"
	"skill-arena/internal/games/maze"
	gamesregistry "skill-arena/internal/games/registry"
)

func TestStoreBuildsProductionRuntimeRegistryWithStoreDependencies(t *testing.T) {
	store, err := New(context.Background(), t.TempDir())
	if err != nil {
		t.Fatalf("new store: %v", err)
	}
	t.Cleanup(func() {
		_ = store.Close(context.Background())
	})

	module, err := store.GamesRegistry().Resolve(maze.ModuleID, maze.New().Manifest().Versions.Game)
	if err != nil {
		t.Fatalf("resolve Maze runtime: %v", err)
	}
	if _, ok := module.(interfaces.RuntimeGame); !ok {
		t.Fatal("default Store registry resolved Maze without its production runtime")
	}
	if _, err := store.ArenaRegistry().Get(maze.ModuleID); err != nil {
		t.Fatalf("existing ArenaRegistry contract cannot resolve Maze: %v", err)
	}
}

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
