package registry_test

import (
	"errors"
	"sync"
	"testing"

	"skill-arena/internal/games/interfaces"
	"skill-arena/internal/games/maze"
	"skill-arena/internal/games/registry"
	"skill-arena/internal/games/shared"
	"skill-arena/internal/games/shared/testkit"
)

type testModule struct {
	descriptor interfaces.Descriptor
}

func (m testModule) Descriptor() interfaces.Descriptor {
	return m.descriptor.Clone()
}

func TestRegistrySupportsActiveAndHistoricalVersions(t *testing.T) {
	active := descriptor("test_arena", "2.0.0", interfaces.ModuleActive, true)
	historical := descriptor("test_arena", "1.0.0", interfaces.ModuleReplayOnly, false)
	catalog, err := registry.New(registration(active), registration(historical))
	if err != nil {
		t.Fatalf("new registry: %v", err)
	}

	current, err := catalog.ResolveForNewMatch("test_arena")
	if err != nil {
		t.Fatalf("resolve active: %v", err)
	}
	if got := current.Descriptor().Versions.Game; got != "2.0.0" {
		t.Fatalf("active version = %s, want 2.0.0", got)
	}
	old, err := catalog.Resolve("test_arena", "1.0.0")
	if err != nil {
		t.Fatalf("resolve historical: %v", err)
	}
	if got := old.Descriptor().Status; got != interfaces.ModuleReplayOnly {
		t.Fatalf("historical status = %s", got)
	}
	if len(catalog.List()) != 2 {
		t.Fatalf("registered versions = %d, want 2", len(catalog.List()))
	}
}

func TestRegistryNeverFallsBackToLatest(t *testing.T) {
	catalog, err := registry.New(registration(descriptor("test_arena", "2.0.0", interfaces.ModuleActive, true)))
	if err != nil {
		t.Fatalf("new registry: %v", err)
	}
	if _, err := catalog.Resolve("test_arena", "1.0.0"); !errors.Is(err, registry.ErrVersionUnavailable) {
		t.Fatalf("resolve missing version error = %v, want %v", err, registry.ErrVersionUnavailable)
	}
	if _, err := catalog.Resolve("missing_arena", "1.0.0"); !errors.Is(err, registry.ErrModuleNotFound) {
		t.Fatalf("resolve missing game error = %v, want %v", err, registry.ErrModuleNotFound)
	}
}

func TestRegistryRejectsConflictsAndFactoryMismatch(t *testing.T) {
	first := descriptor("test_arena", "1.0.0", interfaces.ModuleActive, true)
	catalog, err := registry.New(registration(first))
	if err != nil {
		t.Fatalf("new registry: %v", err)
	}
	if err := catalog.Register(registration(first)); !errors.Is(err, registry.ErrRegistrationConflict) {
		t.Fatalf("duplicate error = %v, want conflict", err)
	}

	secondActive := descriptor("test_arena", "2.0.0", interfaces.ModuleActive, true)
	if err := catalog.Register(registration(secondActive)); !errors.Is(err, registry.ErrRegistrationConflict) {
		t.Fatalf("second active error = %v, want conflict", err)
	}

	mismatch := descriptor("other_arena", "1.0.0", interfaces.ModuleActive, true)
	err = catalog.Register(registry.Registration{
		Descriptor: mismatch,
		Factory: func() (interfaces.Module, error) {
			return testModule{descriptor: first}, nil
		},
	})
	if !errors.Is(err, registry.ErrFactoryMismatch) {
		t.Fatalf("factory mismatch error = %v", err)
	}

	panicDescriptor := descriptor("panic_arena", "1.0.0", interfaces.ModuleActive, true)
	err = catalog.Register(registry.Registration{
		Descriptor: panicDescriptor,
		Factory: func() (interfaces.Module, error) {
			panic("factory failure")
		},
	})
	if !errors.Is(err, registry.ErrFactoryPanic) {
		t.Fatalf("factory panic error = %v", err)
	}
}

func TestRegistryRejectsInvalidDescriptors(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*interfaces.Descriptor)
	}{
		{name: "id", mutate: func(d *interfaces.Descriptor) { d.ID = "Maze Arena" }},
		{name: "game version", mutate: func(d *interfaces.Descriptor) { d.Versions.Game = "latest" }},
		{name: "version tuple", mutate: func(d *interfaces.Descriptor) { d.Versions.Replay = "" }},
		{name: "status", mutate: func(d *interfaces.Descriptor) { d.Status = "unknown" }},
		{name: "new match status", mutate: func(d *interfaces.Descriptor) {
			d.Status, d.NewMatchAllowed = interfaces.ModuleReplayOnly, true
		}},
		{name: "players", mutate: func(d *interfaces.Descriptor) { d.MaximumPlayers = 0 }},
		{name: "renderer", mutate: func(d *interfaces.Descriptor) { d.RendererKey = "Maze Renderer" }},
		{name: "hash", mutate: func(d *interfaces.Descriptor) { d.ManifestHash = "invalid" }},
		{name: "duplicate mode", mutate: func(d *interfaces.Descriptor) { d.Modes = []string{"practice", "practice"} }},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			value := descriptor("test_arena", "1.0.0", interfaces.ModuleActive, true)
			tt.mutate(&value)
			if _, err := registry.New(registration(value)); !errors.Is(err, registry.ErrInvalidDescriptor) {
				t.Fatalf("error = %v, want invalid descriptor", err)
			}
		})
	}
}

func TestRegistryDescriptorsAreImmutableAndConcurrentReadsAreSafe(t *testing.T) {
	value := descriptor("test_arena", "1.0.0", interfaces.ModuleActive, true)
	catalog, err := registry.New(registration(value))
	if err != nil {
		t.Fatalf("new registry: %v", err)
	}
	module, err := catalog.ResolveForNewMatch(value.ID)
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	testkit.AssertModuleContract(t, module)

	list := catalog.List()
	list[0].Modes[0] = "mutated"
	if catalog.List()[0].Modes[0] == "mutated" {
		t.Fatal("registry list leaked mutable descriptor state")
	}

	var wait sync.WaitGroup
	for i := 0; i < 100; i++ {
		wait.Add(1)
		go func() {
			defer wait.Done()
			if _, resolveErr := catalog.Resolve(value.ID, value.Versions.Game); resolveErr != nil {
				t.Errorf("concurrent resolve: %v", resolveErr)
			}
			_ = catalog.List()
		}()
	}
	wait.Wait()
}

func TestProductionBootstrapLoadsOnlyApprovedModule(t *testing.T) {
	catalog, err := registry.NewProduction()
	if err != nil {
		t.Fatalf("production registry: %v", err)
	}
	items := catalog.List()
	if len(items) != 1 || items[0].ID != maze.ModuleID {
		t.Fatalf("production modules = %#v", items)
	}
	module, err := catalog.ResolveForNewMatch(maze.ModuleID)
	if err != nil {
		t.Fatalf("resolve Maze adapter: %v", err)
	}
	testkit.AssertModuleContract(t, module)

	legacy, err := catalog.ArenaRegistry()
	if err != nil {
		t.Fatalf("Arena compatibility registry: %v", err)
	}
	if _, err := legacy.Get(maze.ModuleID); err != nil {
		t.Fatalf("resolve Maze from compatibility registry: %v", err)
	}
	if _, err := legacy.Get("test_arena"); err == nil {
		t.Fatal("test arena must not be registered in production")
	}
}

func descriptor(id, gameVersion string, status interfaces.ModuleStatus, newMatchAllowed bool) interfaces.Descriptor {
	return interfaces.Descriptor{
		ID:              id,
		Name:            "Test Arena",
		Versions:        interfaces.Versions{Game: gameVersion, Rules: "v1", Protocol: "v1", Replay: "v1", Renderer: "v1", StateSchema: "v1"},
		Status:          status,
		NewMatchAllowed: newMatchAllowed,
		MinimumPlayers:  1,
		MaximumPlayers:  2,
		AverageTimeSec:  60,
		Modes:           []string{"practice"},
		Capabilities:    interfaces.Capabilities{Practice: true, Replay: true},
		RendererKey:     "test-arena",
		ManifestHash:    shared.HashFields("test/game-manifest/v1", id, gameVersion),
	}
}

func registration(value interfaces.Descriptor) registry.Registration {
	return registry.Registration{
		Descriptor: value,
		Factory: func() (interfaces.Module, error) {
			return testModule{descriptor: value}, nil
		},
	}
}

func BenchmarkRegistryResolve(b *testing.B) {
	value := descriptor("test_arena", "1.0.0", interfaces.ModuleActive, true)
	catalog, err := registry.New(registration(value))
	if err != nil {
		b.Fatalf("new registry: %v", err)
	}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := catalog.Resolve(value.ID, value.Versions.Game); err != nil {
			b.Fatal(err)
		}
	}
}
