package server

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"skill-arena/internal/config"
	"skill-arena/internal/db"
)

func TestServerUsesConfiguredAddressAndCORS(t *testing.T) {
	store, err := db.New(context.Background(), t.TempDir())
	if err != nil {
		t.Fatalf("new store: %v", err)
	}

	srv := New(store, &config.Config{
		HTTPAddr:  ":9099",
		JWTSecret: "test-secret",
		Settings: &config.RuntimeSettings{
			CORS: config.CORSSettings{AllowedOrigins: []string{"https://app.skillarena.test"}},
		},
	})
	if srv.Addr != ":9099" {
		t.Fatalf("server addr = %q, want :9099", srv.Addr)
	}

	req := httptest.NewRequest(http.MethodOptions, "/api/v1/profile", nil)
	req.Header.Set("Origin", "https://app.skillarena.test")
	res := httptest.NewRecorder()
	srv.Handler.ServeHTTP(res, req)

	if res.Code != http.StatusNoContent {
		t.Fatalf("preflight status = %d, want %d", res.Code, http.StatusNoContent)
	}
	if got := res.Header().Get("Access-Control-Allow-Origin"); got != "https://app.skillarena.test" {
		t.Fatalf("allow origin = %q, want configured origin", got)
	}

	blockedReq := httptest.NewRequest(http.MethodOptions, "/api/v1/profile", nil)
	blockedReq.Header.Set("Origin", "https://evil.example")
	blockedRes := httptest.NewRecorder()
	srv.Handler.ServeHTTP(blockedRes, blockedReq)
	if got := blockedRes.Header().Get("Access-Control-Allow-Origin"); got != "" {
		t.Fatalf("blocked origin was allowed: %q", got)
	}
}
