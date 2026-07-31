package observability

import (
	"bytes"
	"strings"
	"testing"
	"time"
)

func TestPrometheusRecorderExportsCumulativeHistogramAndHTTPCount(t *testing.T) {
	recorder := NewPrometheusRecorder()
	recorder.ObserveTiming("game_action.total", 40*time.Millisecond)
	recorder.ObserveTiming("game_action.total", 300*time.Millisecond)
	recorder.ObserveHTTPRequest("post", 201, 2*time.Millisecond)

	var output bytes.Buffer
	if err := recorder.WritePrometheus(&output); err != nil {
		t.Fatal(err)
	}
	body := output.String()
	for _, expected := range []string{
		`skill_arena_component_duration_seconds_bucket{component="game_action.total",le="0.05"} 1`,
		`skill_arena_component_duration_seconds_bucket{component="game_action.total",le="0.5"} 2`,
		`skill_arena_component_duration_seconds_count{component="game_action.total"} 2`,
		`skill_arena_http_requests_total{method="POST",status_class="2xx"} 1`,
	} {
		if !strings.Contains(body, expected) {
			t.Fatalf("metrics missing %q:\n%s", expected, body)
		}
	}
}
