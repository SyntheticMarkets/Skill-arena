package realtime

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"skill-arena/internal/config"
	"skill-arena/internal/db"
	"skill-arena/internal/games/interfaces"
	gamesession "skill-arena/internal/games/session"
	"skill-arena/internal/id"

	"github.com/gorilla/websocket"
)

const (
	writeTimeout = 5 * time.Second
	pongTimeout  = 35 * time.Second
	pingInterval = 15 * time.Second
)

type Gateway struct {
	store   *db.Store
	service *Service
	cfg     *config.Config
}

type clientMessage struct {
	Type                 string          `json:"type"`
	MatchID              string          `json:"matchId,omitempty"`
	AfterSequence        int64           `json:"afterSequence,omitempty"`
	LatencyMS            int             `json:"latencyMs,omitempty"`
	Region               string          `json:"region,omitempty"`
	ActionID             string          `json:"actionId,omitempty"`
	Kind                 string          `json:"kind,omitempty"`
	Payload              json.RawMessage `json:"payload,omitempty"`
	ClientSequence       int64           `json:"clientSequence,omitempty"`
	ExpectedStateVersion int64           `json:"expectedStateVersion,omitempty"`
}

func NewGateway(store *db.Store, service *Service, cfg *config.Config) *Gateway {
	return &Gateway{store: store, service: service, cfg: cfg}
}

func (g *Gateway) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	http.Error(w, "authenticated gateway identity is required", http.StatusUnauthorized)
}

func (g *Gateway) ServeAuthenticated(w http.ResponseWriter, r *http.Request, userID, sessionID string) {
	connectionLimit := g.cfg.Settings.Realtime.ConnectionLimit
	if connectionLimit <= 0 {
		connectionLimit = 10
	}
	allowed, err := g.store.Redis().Allow(r.Context(), "realtime:connect:"+userID, connectionLimit, time.Minute)
	if err != nil || !allowed {
		http.Error(w, "connection rate limit exceeded", http.StatusTooManyRequests)
		return
	}
	ownershipToken, acquired, err := g.store.Redis().Lock(r.Context(), "realtime:connection:"+userID, 5*time.Second)
	if err != nil || !acquired {
		http.Error(w, "connection negotiation is busy", http.StatusConflict)
		return
	}
	existing, presenceErr := g.store.GetPresence(r.Context(), userID)
	if presenceErr != nil && !errors.Is(presenceErr, sql.ErrNoRows) {
		_ = g.store.Redis().Unlock(context.Background(), "realtime:connection:"+userID, ownershipToken)
		http.Error(w, "presence is unavailable", http.StatusServiceUnavailable)
		return
	}
	if existing != nil && existing.ConnectionID != "" && existing.ExpiresAt.After(time.Now().UTC()) &&
		existing.State != "disconnected" && existing.State != "offline" {
		_ = g.store.Redis().Unlock(context.Background(), "realtime:connection:"+userID, ownershipToken)
		http.Error(w, "player already has an active realtime connection", http.StatusConflict)
		return
	}
	upgrader := websocket.Upgrader{
		HandshakeTimeout: 5 * time.Second,
		CheckOrigin: func(request *http.Request) bool {
			origin := strings.TrimSpace(request.Header.Get("Origin"))
			if origin == "" {
				return false
			}
			for _, allowedOrigin := range g.cfg.Settings.CORS.AllowedOrigins {
				if strings.EqualFold(strings.TrimSpace(allowedOrigin), origin) {
					return true
				}
			}
			return false
		},
	}
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		_ = g.store.Redis().Unlock(context.Background(), "realtime:connection:"+userID, ownershipToken)
		return
	}
	defer conn.Close()
	connectionID := id.New("con")
	if err := g.service.setPresence(r.Context(), userID, "online", sessionID, connectionID, "", ""); err != nil {
		_ = g.store.Redis().Unlock(context.Background(), "realtime:connection:"+userID, ownershipToken)
		return
	}
	_ = g.store.Redis().Unlock(context.Background(), "realtime:connection:"+userID, ownershipToken)

	messageLimit := g.cfg.Settings.Realtime.MaxMessageBytes
	if messageLimit <= 0 {
		messageLimit = 8 << 10
	}
	conn.SetReadLimit(messageLimit)
	_ = conn.SetReadDeadline(time.Now().Add(pongTimeout))
	conn.SetPongHandler(func(string) error {
		_ = conn.SetReadDeadline(time.Now().Add(pongTimeout))
		return nil
	})
	messages := make(chan clientMessage, 8)
	readErrors := make(chan error, 1)
	go func() {
		for {
			var message clientMessage
			if err := conn.ReadJSON(&message); err != nil {
				readErrors <- err
				return
			}
			select {
			case messages <- message:
			default:
				readErrors <- websocket.ErrReadLimit
				return
			}
		}
	}()
	_ = g.write(conn, map[string]any{"type": "session.negotiated", "connectionId": connectionID, "serverTime": time.Now().UTC(), "protocolVersion": "1"})

	ping := time.NewTicker(pingInterval)
	poll := time.NewTicker(250 * time.Millisecond)
	defer ping.Stop()
	defer poll.Stop()
	var matchID string
	var sequence int64
	for {
		select {
		case <-r.Context().Done():
			_ = g.disconnect(context.Background(), userID, sessionID, connectionID, matchID)
			return
		case <-readErrors:
			_ = g.disconnect(context.Background(), userID, sessionID, connectionID, matchID)
			return
		case message := <-messages:
			switch message.Type {
			case "heartbeat":
				serverTime, err := g.service.Heartbeat(r.Context(), userID, sessionID, connectionID, matchID, message.Region, message.LatencyMS)
				if err != nil {
					_ = g.write(conn, map[string]any{"type": "error", "code": "INVALID_HEARTBEAT"})
					continue
				}
				_ = g.write(conn, map[string]any{"type": "heartbeat.ack", "serverTime": serverTime})
			case "subscribe", "reconnect":
				match, events, err := g.service.Reconnect(r.Context(), userID, message.MatchID, message.AfterSequence)
				if err != nil {
					_ = g.write(conn, map[string]any{"type": "error", "code": "MATCH_ACCESS_DENIED"})
					continue
				}
				matchID, sequence = match.ID, message.AfterSequence
				response := map[string]any{
					"type": "state.sync", "match": match, "events": events,
					"serverTime": time.Now().UTC(),
				}
				if game, syncErr := g.service.GameSync(r.Context(), userID, match.ID); syncErr == nil {
					response["game"] = game
				}
				_ = g.write(conn, response)
				if len(events) > 0 {
					sequence = events[len(events)-1].Sequence
				}
			case "ready":
				match, err := g.service.Ready(r.Context(), userID, message.MatchID)
				if err != nil {
					_ = g.write(conn, map[string]any{"type": "error", "code": "READY_REJECTED"})
					continue
				}
				matchID = match.ID
				_ = g.write(conn, map[string]any{"type": "match.status", "match": match})
			case "leave":
				match, err := g.service.Leave(r.Context(), userID, message.MatchID)
				if err != nil {
					_ = g.write(conn, map[string]any{"type": "error", "code": "LEAVE_REJECTED"})
					continue
				}
				_ = g.write(conn, map[string]any{"type": "match.status", "match": match})
				matchID = ""
			case "game.action":
				result, err := g.service.GameAction(
					r.Context(), userID, message.MatchID, interfaces.ActionEnvelope{
						ActionID: message.ActionID, MatchID: message.MatchID,
						Kind: message.Kind, Payload: message.Payload,
						ClientSequence:       message.ClientSequence,
						ExpectedStateVersion: message.ExpectedStateVersion,
					}, time.Duration(message.LatencyMS)*time.Millisecond,
				)
				if err != nil {
					_ = g.write(conn, map[string]any{
						"type": "error", "code": gameErrorCode(err),
					})
					continue
				}
				matchID = message.MatchID
				_ = g.write(conn, map[string]any{
					"type": "game.action.receipt", "result": result,
				})
			case "game.sync.request":
				result, err := g.service.GameSync(r.Context(), userID, message.MatchID)
				if err != nil {
					_ = g.write(conn, map[string]any{
						"type": "error", "code": "GAME_SYNC_REJECTED",
					})
					continue
				}
				matchID = message.MatchID
				_ = g.write(conn, map[string]any{
					"type": "game.state.sync", "game": result,
					"serverTime": time.Now().UTC(),
				})
			case "ack":
				if message.AfterSequence > sequence {
					sequence = message.AfterSequence
				}
			default:
				_ = g.write(conn, map[string]any{"type": "error", "code": "UNSUPPORTED_MESSAGE"})
			}
		case <-poll.C:
			if matchID == "" {
				continue
			}
			events, err := g.service.Events(r.Context(), userID, matchID, sequence)
			if err != nil {
				continue
			}
			for _, event := range events {
				if err := g.write(conn, map[string]any{"type": "match.event", "event": event}); err != nil {
					return
				}
				sequence = event.Sequence
			}
		case <-ping.C:
			_ = conn.SetWriteDeadline(time.Now().Add(writeTimeout))
			if err := conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

func gameErrorCode(err error) string {
	switch {
	case errors.Is(err, gamesession.ErrActionSequence):
		return "GAME_SEQUENCE_GAP"
	case errors.Is(err, gamesession.ErrActionConflict),
		errors.Is(err, db.ErrRealtimeConflict):
		return "GAME_STATE_CONFLICT"
	case errors.Is(err, gamesession.ErrDuplicateMismatch):
		return "GAME_DUPLICATE_MISMATCH"
	default:
		return "GAME_ACTION_REJECTED"
	}
}

func (g *Gateway) disconnect(ctx context.Context, userID, sessionID, connectionID, matchID string) error {
	return g.service.Disconnect(ctx, userID, sessionID, connectionID, matchID)
}

func (g *Gateway) write(conn *websocket.Conn, value any) error {
	_ = conn.SetWriteDeadline(time.Now().Add(writeTimeout))
	return conn.WriteJSON(value)
}
