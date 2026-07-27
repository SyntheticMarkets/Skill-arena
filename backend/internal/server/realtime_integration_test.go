package server

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"skill-arena/internal/db"
	"skill-arena/internal/models"
)

func TestRealtimeAPILifecycleAndAuthorization(t *testing.T) {
	store, err := db.New(context.Background(), t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = store.Close(context.Background()) })
	cfg := authTestConfig("")
	handler := New(store, cfg).Handler
	access, _ := registerVerifyLogin(
		t, handler, store,
		[2]string{cfg.Settings.Security.AccessCookieName, cfg.Settings.Security.RefreshCookieName},
		"realtime-api@example.com", "StrongPassword!42",
	)
	request := map[string]any{
		"gameId": "maze_arena", "mode": "practice", "walletCategory": "practice",
		"region": "af-south", "jurisdiction": "ZA", "latencyMs": 15,
	}
	unauthorized := authRequest(t, handler, http.MethodPost, "/api/v1/realtime/queue", request, nil)
	if unauthorized.Code != http.StatusUnauthorized {
		t.Fatalf("unauthorized queue status=%d body=%s", unauthorized.Code, unauthorized.Body.String())
	}
	queued := authRequest(t, handler, http.MethodPost, "/api/v1/realtime/queue", request, []*http.Cookie{access})
	if queued.Code != http.StatusOK {
		t.Fatalf("queue status=%d body=%s", queued.Code, queued.Body.String())
	}
	var result struct {
		Match models.RealtimeMatch `json:"match"`
	}
	if err := json.Unmarshal(queued.Body.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if result.Match.ID == "" || result.Match.GameID != "maze_arena" || result.Match.Status != models.MatchReady {
		t.Fatalf("unexpected match: %+v", result.Match)
	}
	ready := authRequest(t, handler, http.MethodPost, "/api/v1/realtime/matches/"+result.Match.ID+"/ready", map[string]any{}, []*http.Cookie{access})
	if ready.Code != http.StatusOK || !containsJSONStatus(ready.Body.Bytes(), models.MatchLive) {
		t.Fatalf("ready status=%d body=%s", ready.Code, ready.Body.String())
	}
	reconnect := authRequest(t, handler, http.MethodPost, "/api/v1/realtime/matches/"+result.Match.ID+"/reconnect", map[string]any{"afterSequence": 0}, []*http.Cookie{access})
	if reconnect.Code != http.StatusOK {
		t.Fatalf("reconnect status=%d body=%s", reconnect.Code, reconnect.Body.String())
	}
	leave := authRequest(t, handler, http.MethodPost, "/api/v1/realtime/matches/"+result.Match.ID+"/leave", map[string]any{}, []*http.Cookie{access})
	if leave.Code != http.StatusOK || !containsJSONStatus(leave.Body.Bytes(), models.MatchAbandoned) {
		t.Fatalf("leave status=%d body=%s", leave.Code, leave.Body.String())
	}
	invalid := request
	invalid["won"] = true
	rejected := authRequest(t, handler, http.MethodPost, "/api/v1/realtime/queue", invalid, []*http.Cookie{access})
	if rejected.Code != http.StatusBadRequest {
		t.Fatalf("client state field was accepted: status=%d body=%s", rejected.Code, rejected.Body.String())
	}
}

func containsJSONStatus(body []byte, expected string) bool {
	var value struct {
		Status string `json:"status"`
	}
	return json.Unmarshal(body, &value) == nil && value.Status == expected
}
