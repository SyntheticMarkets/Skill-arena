package events

import (
	"time"

	"skill-arena/internal/arena/core"
)

type Sink interface {
	Emit(event core.Event)
}

type MemorySink struct {
	items []core.Event
}

func NewMemorySink() *MemorySink {
	return &MemorySink{items: []core.Event{}}
}

func (s *MemorySink) Emit(event core.Event) {
	if event.CreatedAt.IsZero() {
		event.CreatedAt = time.Now().UTC()
	}
	s.items = append(s.items, event)
}

func (s *MemorySink) Items() []core.Event {
	copied := make([]core.Event, len(s.items))
	copy(copied, s.items)
	return copied
}
