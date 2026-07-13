package events

import (
	"sync"
	"time"

	"skill-arena/internal/arena/core"
)

type Sink interface {
	Emit(event core.Event)
}

type Handler func(core.Event)

type Bus struct {
	mu       sync.RWMutex
	handlers map[string][]Handler
	history  []core.Event
}

func NewBus() *Bus {
	return &Bus{handlers: map[string][]Handler{}, history: []core.Event{}}
}

func (b *Bus) Subscribe(eventType string, handler Handler) {
	if b == nil || handler == nil {
		return
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	b.handlers[eventType] = append(b.handlers[eventType], handler)
}

func (b *Bus) Publish(event core.Event) {
	if b == nil {
		return
	}
	if event.CreatedAt.IsZero() {
		event.CreatedAt = time.Now().UTC()
	}
	b.mu.Lock()
	b.history = append(b.history, event)
	handlers := append([]Handler(nil), b.handlers[event.Type]...)
	handlers = append(handlers, b.handlers["*"]...)
	b.mu.Unlock()
	for _, handler := range handlers {
		handler(event)
	}
}

func (b *Bus) History() []core.Event {
	if b == nil {
		return []core.Event{}
	}
	b.mu.RLock()
	defer b.mu.RUnlock()
	copied := make([]core.Event, len(b.history))
	copy(copied, b.history)
	return copied
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
