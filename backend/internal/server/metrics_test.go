package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"skill-arena/internal/db"
	"skill-arena/internal/observability"
)

func TestMetricsHandlerRequiresBearerTokenAndExportsMetrics(t *testing.T) {
	store, err := db.New(t.Context(), t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = store.Close(t.Context()) })

	recorder := observability.NewPrometheusRecorder()
	recorder.ObserveTiming("game_action.total", 20*time.Millisecond)
	handler := metricsHandler(store, recorder, "metrics-secret-at-least-32-characters")

	unauthorized := httptest.NewRecorder()
	handler.ServeHTTP(unauthorized, httptest.NewRequest(http.MethodGet, "/metrics", nil))
	if unauthorized.Code != http.StatusUnauthorized {
		t.Fatalf("unauthorized status=%d", unauthorized.Code)
	}

	request := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	request.Header.Set("Authorization", "Bearer metrics-secret-at-least-32-characters")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("metrics status=%d body=%s", response.Code, response.Body.String())
	}
	for _, expected := range []string{
		"skill_arena_component_duration_seconds",
		"skill_arena_realtime_active_matches",
		"skill_arena_jobs_pending",
	} {
		if !strings.Contains(response.Body.String(), expected) {
			t.Fatalf("metrics missing %q:\n%s", expected, response.Body.String())
		}
	}
}

func TestMetricsHandlerIsUnavailableWithoutConfiguredToken(t *testing.T) {
	store, err := db.New(t.Context(), t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = store.Close(t.Context()) })

	response := httptest.NewRecorder()
	metricsHandler(store, observability.NewPrometheusRecorder(), "").
		ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/metrics", nil))
	if response.Code != http.StatusNotFound {
		t.Fatalf("status=%d, want 404", response.Code)
	}
}
