package handlers

import (
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"skill-arena/internal/db"
	"skill-arena/internal/realtime"
)

type RealtimeHandlers struct {
	service *realtime.Service
}

func NewRealtimeHandlers(store *db.Store) *RealtimeHandlers {
	return &RealtimeHandlers{service: realtime.NewService(store)}
}

func (h *RealtimeHandlers) Queue(w http.ResponseWriter, r *http.Request) {
	userID := UserIDFromContext(r.Context())
	switch r.Method {
	case http.MethodPost:
		var request realtime.QueueRequest
		if err := decodeRealtimeJSON(r, &request); err != nil {
			WriteAPIError(w, http.StatusBadRequest, ErrInvalidRequest, err.Error())
			return
		}
		entry, match, err := h.service.Queue(r.Context(), userID, request)
		if err != nil && !errors.Is(err, realtime.ErrAlreadyQueued) {
			writeRealtimeError(w, err)
			return
		}
		writeRealtimeJSON(w, http.StatusOK, map[string]any{"queue": entry, "match": match})
	case http.MethodGet:
		entry, err := h.service.QueueStatus(r.Context(), userID)
		if err != nil {
			writeRealtimeError(w, err)
			return
		}
		writeRealtimeJSON(w, http.StatusOK, entry)
	case http.MethodDelete:
		entry, err := h.service.CancelQueue(r.Context(), userID)
		if err != nil {
			writeRealtimeError(w, err)
			return
		}
		writeRealtimeJSON(w, http.StatusOK, entry)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func (h *RealtimeHandlers) Matches(w http.ResponseWriter, r *http.Request) {
	path := strings.Trim(strings.TrimPrefix(r.URL.Path, "/api/v1/realtime/matches/"), "/")
	parts := strings.Split(path, "/")
	if path == "" || len(parts) == 0 {
		WriteAPIError(w, http.StatusBadRequest, ErrInvalidRequest, "match id is required")
		return
	}
	matchID := parts[0]
	action := ""
	if len(parts) > 1 {
		action = parts[1]
	}
	userID := UserIDFromContext(r.Context())
	if r.Method == http.MethodGet && action == "" {
		match, err := h.service.Match(r.Context(), userID, matchID)
		if err != nil {
			writeRealtimeError(w, err)
			return
		}
		writeRealtimeJSON(w, http.StatusOK, match)
		return
	}
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	switch action {
	case "ready":
		match, err := h.service.Ready(r.Context(), userID, matchID)
		if err != nil {
			writeRealtimeError(w, err)
			return
		}
		writeRealtimeJSON(w, http.StatusOK, match)
	case "leave":
		match, err := h.service.Leave(r.Context(), userID, matchID)
		if err != nil {
			writeRealtimeError(w, err)
			return
		}
		writeRealtimeJSON(w, http.StatusOK, match)
	case "reconnect":
		var request struct {
			AfterSequence int64 `json:"afterSequence"`
		}
		if err := decodeRealtimeJSON(r, &request); err != nil {
			WriteAPIError(w, http.StatusBadRequest, ErrInvalidRequest, err.Error())
			return
		}
		match, events, err := h.service.Reconnect(r.Context(), userID, matchID, request.AfterSequence)
		if err != nil {
			writeRealtimeError(w, err)
			return
		}
		writeRealtimeJSON(w, http.StatusOK, map[string]any{"match": match, "events": events})
	case "heartbeat":
		var request struct {
			ConnectionID string `json:"connectionId"`
			Region       string `json:"region"`
			LatencyMS    int    `json:"latencyMs"`
		}
		if err := decodeRealtimeJSON(r, &request); err != nil {
			WriteAPIError(w, http.StatusBadRequest, ErrInvalidRequest, err.Error())
			return
		}
		serverTime, err := h.service.Heartbeat(r.Context(), userID, SessionIDFromContext(r.Context()), request.ConnectionID, matchID, request.Region, request.LatencyMS)
		if err != nil {
			writeRealtimeError(w, err)
			return
		}
		writeRealtimeJSON(w, http.StatusOK, map[string]any{"serverTime": serverTime})
	default:
		WriteAPIError(w, http.StatusNotFound, ErrNotFound, "realtime action was not found")
	}
}

func (h *RealtimeHandlers) Events(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	matchID := strings.Trim(strings.TrimPrefix(r.URL.Path, "/api/v1/realtime/events/"), "/")
	after, _ := strconv.ParseInt(r.URL.Query().Get("after"), 10, 64)
	events, err := h.service.Events(r.Context(), UserIDFromContext(r.Context()), matchID, after)
	if err != nil {
		writeRealtimeError(w, err)
		return
	}
	writeRealtimeJSON(w, http.StatusOK, map[string]any{"events": events})
}

func (h *RealtimeHandlers) Replay(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	matchID := strings.Trim(strings.TrimPrefix(r.URL.Path, "/api/v1/realtime/replays/"), "/")
	replay, err := h.service.Replay(r.Context(), UserIDFromContext(r.Context()), matchID)
	if err != nil {
		writeRealtimeError(w, err)
		return
	}
	writeRealtimeJSON(w, http.StatusOK, replay)
}

func (h *RealtimeHandlers) Service() *realtime.Service {
	return h.service
}

func decodeRealtimeJSON(r *http.Request, target any) error {
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return errors.New("request body is invalid")
	}
	return nil
}

func writeRealtimeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func writeRealtimeError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, sql.ErrNoRows):
		WriteAPIError(w, http.StatusNotFound, ErrNotFound, "realtime resource was not found")
	case errors.Is(err, realtime.ErrNotParticipant):
		WriteAPIError(w, http.StatusForbidden, ErrForbidden, "match access is forbidden")
	case errors.Is(err, realtime.ErrAlreadyQueued), errors.Is(err, realtime.ErrInvalidTransition), errors.Is(err, db.ErrRealtimeConflict):
		WriteAPIError(w, http.StatusConflict, ErrConflict, err.Error())
	case errors.Is(err, realtime.ErrUnsupported):
		WriteAPIError(w, http.StatusUnprocessableEntity, "MODE_UNSUPPORTED", err.Error())
	default:
		WriteAPIError(w, http.StatusBadRequest, ErrInvalidRequest, err.Error())
	}
}
