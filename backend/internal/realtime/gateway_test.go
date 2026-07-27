package realtime

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"skill-arena/internal/config"
	"skill-arena/internal/db"
	"skill-arena/internal/games/testarena"

	"github.com/gorilla/websocket"
)

func TestGatewayNegotiatesHeartbeatAndRejectsUnknownMessages(t *testing.T) {
	ctx := context.Background()
	store, err := db.New(ctx, t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = store.Close(context.Background()) })
	if err := store.ArenaRegistry().Register(newPvPTestModule()); err != nil {
		t.Fatal(err)
	}
	user := createRealtimeUser(t, store, "gateway@example.com")
	settings := config.LoadRuntimeSettings()
	settings.CORS.AllowedOrigins = []string{"http://localhost:3000"}
	store.ConfigureRuntime(settings)
	cfg := &config.Config{Settings: settings}
	service := NewService(store)
	_, match, err := service.Queue(ctx, user.ID, QueueRequest{GameID: testarena.ModuleID, Mode: "practice", Region: "global", LatencyMS: 5})
	if err != nil {
		t.Fatal(err)
	}
	gateway := NewGateway(store, service, cfg)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gateway.ServeAuthenticated(w, r, user.ID, "session-test")
	}))
	defer server.Close()

	header := http.Header{"Origin": []string{"http://localhost:3000"}}
	conn, _, err := websocket.DefaultDialer.Dial("ws"+strings.TrimPrefix(server.URL, "http"), header)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	var response map[string]any
	if err := conn.ReadJSON(&response); err != nil || response["type"] != "session.negotiated" {
		t.Fatalf("session negotiation failed: %+v %v", response, err)
	}
	if err := conn.WriteJSON(map[string]any{"type": "heartbeat", "matchId": match.ID, "latencyMs": 12}); err != nil {
		t.Fatal(err)
	}
	if err := conn.ReadJSON(&response); err != nil || response["type"] != "heartbeat.ack" {
		t.Fatalf("heartbeat failed: %+v %v", response, err)
	}
	if err := conn.WriteJSON(map[string]any{"type": "client_state", "won": true}); err != nil {
		t.Fatal(err)
	}
	if err := conn.ReadJSON(&response); err != nil || response["code"] != "UNSUPPORTED_MESSAGE" {
		t.Fatalf("client state should be rejected: %+v %v", response, err)
	}
}

func TestGatewayRejectsUnapprovedOrigin(t *testing.T) {
	store, err := db.New(context.Background(), t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = store.Close(context.Background()) })
	user := createRealtimeUser(t, store, "origin@example.com")
	settings := config.LoadRuntimeSettings()
	settings.CORS.AllowedOrigins = []string{"https://arena.example.com"}
	store.ConfigureRuntime(settings)
	gateway := NewGateway(store, NewService(store), &config.Config{Settings: settings})
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gateway.ServeAuthenticated(w, r, user.ID, "session-test")
	}))
	defer server.Close()
	header := http.Header{"Origin": []string{"https://evil.example.com"}}
	_, response, err := websocket.DefaultDialer.Dial("ws"+strings.TrimPrefix(server.URL, "http"), header)
	if err == nil || response == nil || response.StatusCode != http.StatusForbidden {
		t.Fatalf("unapproved origin was not rejected: response=%v err=%v", response, err)
	}
}

func TestGatewayRejectsSecondActiveConnectionForPlayer(t *testing.T) {
	store, err := db.New(context.Background(), t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = store.Close(context.Background()) })
	user := createRealtimeUser(t, store, "single-connection@example.com")
	settings := config.LoadRuntimeSettings()
	settings.CORS.AllowedOrigins = []string{"http://localhost:3000"}
	store.ConfigureRuntime(settings)
	gateway := NewGateway(store, NewService(store), &config.Config{Settings: settings})
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gateway.ServeAuthenticated(w, r, user.ID, "session-test")
	}))
	defer server.Close()
	header := http.Header{"Origin": []string{"http://localhost:3000"}}
	first, _, err := websocket.DefaultDialer.Dial("ws"+strings.TrimPrefix(server.URL, "http"), header)
	if err != nil {
		t.Fatal(err)
	}
	defer first.Close()
	var negotiated map[string]any
	if err := first.ReadJSON(&negotiated); err != nil {
		t.Fatal(err)
	}
	second, response, err := websocket.DefaultDialer.Dial("ws"+strings.TrimPrefix(server.URL, "http"), header)
	if second != nil {
		second.Close()
	}
	if err == nil || response == nil || response.StatusCode != http.StatusConflict {
		t.Fatalf("second active connection was not rejected: response=%v err=%v", response, err)
	}
}

func TestGatewayNegotiatesOneHundredConcurrentConnections(t *testing.T) {
	store, err := db.New(context.Background(), t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = store.Close(context.Background()) })
	settings := config.LoadRuntimeSettings()
	settings.CORS.AllowedOrigins = []string{"http://localhost:3000"}
	settings.Realtime.ConnectionLimit = 20
	store.ConfigureRuntime(settings)
	gateway := NewGateway(store, NewService(store), &config.Config{Settings: settings})
	users := make([]string, 100)
	for i := range users {
		users[i] = createRealtimeUser(t, store, fmt.Sprintf("socket-%03d@example.com", i)).ID
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		userID := r.URL.Query().Get("user")
		gateway.ServeAuthenticated(w, r, userID, "session-"+userID)
	}))
	defer server.Close()
	header := http.Header{"Origin": []string{"http://localhost:3000"}}
	var wg sync.WaitGroup
	errs := make(chan error, len(users))
	for _, userID := range users {
		wg.Add(1)
		go func() {
			defer wg.Done()
			conn, _, err := websocket.DefaultDialer.Dial("ws"+strings.TrimPrefix(server.URL, "http")+"?user="+userID, header)
			if err != nil {
				errs <- err
				return
			}
			defer conn.Close()
			var response map[string]any
			if err := conn.ReadJSON(&response); err != nil {
				errs <- err
				return
			}
			if response["type"] != "session.negotiated" {
				errs <- fmt.Errorf("unexpected gateway response %v", response)
			}
		}()
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		t.Error(err)
	}
}
