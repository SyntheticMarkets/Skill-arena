package observability

import (
	"encoding/json"
	"log"
	"sync"
	"time"
)

type Logger struct {
	Service string
}

func (l Logger) Info(message string, fields map[string]any) {
	l.write("info", message, fields)
}

func (l Logger) Error(message string, err error, fields map[string]any) {
	if fields == nil {
		fields = map[string]any{}
	}
	if err != nil {
		fields["error"] = err.Error()
	}
	l.write("error", message, fields)
}

func (l Logger) write(level, message string, fields map[string]any) {
	if fields == nil {
		fields = map[string]any{}
	}
	fields["level"] = level
	fields["message"] = message
	fields["service"] = l.Service
	fields["ts"] = time.Now().UTC().Format(time.RFC3339Nano)
	data, _ := json.Marshal(fields)
	log.Print(string(data))
}

type Metrics struct {
	mu       sync.Mutex
	counters map[string]int64
	timers   map[string]time.Duration
}

func NewMetrics() *Metrics {
	return &Metrics{counters: map[string]int64{}, timers: map[string]time.Duration{}}
}

func (m *Metrics) Inc(name string, delta int64) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.counters[name] += delta
}

func (m *Metrics) ObserveDuration(name string, duration time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.timers[name] += duration
}

func (m *Metrics) Snapshot() map[string]any {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := map[string]any{"counters": map[string]int64{}, "timersMs": map[string]int64{}}
	for key, value := range m.counters {
		out["counters"].(map[string]int64)[key] = value
	}
	for key, value := range m.timers {
		out["timersMs"].(map[string]int64)[key] = value.Milliseconds()
	}
	return out
}

type ComponentHealth struct {
	Name   string `json:"name"`
	Status string `json:"status"`
	Error  string `json:"error,omitempty"`
}
