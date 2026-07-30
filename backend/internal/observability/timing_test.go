package observability

import (
	"context"
	"testing"
	"time"
)

type timingRecorderFunc func(string, time.Duration)

func (fn timingRecorderFunc) ObserveTiming(name string, duration time.Duration) {
	fn(name, duration)
}

func TestTimingRecorderUsesContextWithoutChangingUninstrumentedCalls(t *testing.T) {
	ObserveTiming(context.Background(), "ignored", time.Now())

	var observedName string
	var observedDuration time.Duration
	ctx := WithTimingRecorder(context.Background(), timingRecorderFunc(
		func(name string, duration time.Duration) {
			observedName = name
			observedDuration = duration
		},
	))
	ObserveTiming(ctx, "database.query", time.Now().Add(-time.Millisecond))

	if observedName != "database.query" || observedDuration < time.Millisecond {
		t.Fatalf("observation name=%q duration=%s", observedName, observedDuration)
	}
}
