package testkit

import (
	"testing"

	"skill-arena/internal/games/interfaces"
	"skill-arena/internal/games/registry"
)

func AssertModuleContract(t testing.TB, module interfaces.Module) {
	t.Helper()
	if module == nil {
		t.Fatal("module is nil")
	}
	first := module.Descriptor()
	second := module.Descriptor()
	if err := registry.ValidateDescriptor(first); err != nil {
		t.Fatalf("invalid descriptor: %v", err)
	}
	if first.ID != second.ID || first.Versions.Key() != second.Versions.Key() || first.ManifestHash != second.ManifestHash {
		t.Fatal("module descriptor identity changed between reads")
	}
	if len(first.Modes) > 0 {
		first.Modes[0] = "mutated"
		if module.Descriptor().Modes[0] == "mutated" {
			t.Fatal("module descriptor leaked mutable modes")
		}
	}
}
