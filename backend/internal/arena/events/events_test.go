package events

import (
	"testing"

	"skill-arena/internal/arena/core"
)

func TestBusPublishesToSpecificAndWildcardSubscribers(t *testing.T) {
	bus := NewBus()
	received := 0
	bus.Subscribe(core.EventPuzzleSolved, func(core.Event) { received++ })
	bus.Subscribe("*", func(core.Event) { received++ })

	bus.Publish(core.Event{Type: core.EventPuzzleSolved, GameID: "test_arena"})

	if received != 2 {
		t.Fatalf("received events = %d, want 2", received)
	}
	if len(bus.History()) != 1 {
		t.Fatalf("history length = %d, want 1", len(bus.History()))
	}
}
