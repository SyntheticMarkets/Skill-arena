package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"skill-arena/internal/db"
	"skill-arena/internal/models"
)

type joinPvPRequest struct {
	QueueType  string  `json:"queueType"`
	WalletType string  `json:"walletType"`
	Stake      float64 `json:"stake"`
}

type submitPvPRequest struct {
	MatchID        string                    `json:"matchId"`
	Moves          []string                  `json:"moves"`
	ClickedLineIDs []string                  `json:"clickedLineIds"`
	Telemetry      *models.GameplayTelemetry `json:"telemetry,omitempty"`
}

type pvpProgressRequest struct {
	MatchID            string  `json:"matchId"`
	CurrentProgress    int     `json:"currentProgress"`
	CurrentCombo       int     `json:"currentCombo"`
	MovesRemaining     int     `json:"movesRemaining"`
	CompletionPercent  float64 `json:"completionPercent"`
	Finished           bool    `json:"finished"`
	Disconnected       bool    `json:"disconnected"`
	LastClientSequence int     `json:"lastClientSequence"`
}

func PvPJoinHandler(store *db.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}

		var req joinPvPRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid request payload", http.StatusBadRequest)
			return
		}
		userID := UserIDFromContext(r.Context())
		if userID == "" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		detail, err := store.JoinPvPQueue(r.Context(), userID, req.QueueType, req.WalletType, req.Stake)
		if err != nil {
			WriteMappedError(w, http.StatusBadRequest, fmt.Errorf("cannot join pvp queue: %v", err))
			return
		}
		_ = store.AppendAuditLog(r.Context(), userID, "pvp.queue.joined", detail.Match.ID, clientIP(r), map[string]string{"status": detail.Match.Status})

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(detail)
	}
}

func PvPMatchesHandler(store *db.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		userID := UserIDFromContext(r.Context())
		if userID == "" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		matches, err := store.ListPvPMatchesByUserID(r.Context(), userID)
		if err != nil {
			http.Error(w, fmt.Sprintf("failed to load pvp matches: %v", err), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(matches)
	}
}

func PvPMatchDetailHandler(store *db.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		userID := UserIDFromContext(r.Context())
		if userID == "" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		matchID := strings.TrimPrefix(r.URL.Path, "/api/v1/pvp/matches/")
		if matchID == "" {
			http.Error(w, "match id required", http.StatusBadRequest)
			return
		}

		detail, err := store.GetPvPMatchDetail(r.Context(), matchID, userID)
		if err != nil {
			WriteMappedError(w, http.StatusNotFound, fmt.Errorf("pvp match not found: %v", err))
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(detail)
	}
}

func PvPSubmitHandler(store *db.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}

		var req submitPvPRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid request payload", http.StatusBadRequest)
			return
		}
		lineIDs := req.ClickedLineIDs
		if len(lineIDs) == 0 {
			lineIDs = req.Moves
		}
		if req.MatchID == "" || len(lineIDs) == 0 {
			http.Error(w, "matchId and clickedLineIds are required", http.StatusBadRequest)
			return
		}
		userID := UserIDFromContext(r.Context())
		if userID == "" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		moveHistory := make([]models.MazeMove, 0, len(lineIDs))
		for _, direction := range lineIDs {
			moveHistory = append(moveHistory, models.MazeMove{
				Direction: direction,
				Timestamp: time.Now().UTC(),
			})
		}

		detail, err := store.SubmitPvPMoves(r.Context(), userID, req.MatchID, moveHistory)
		if err != nil {
			WriteMappedError(w, http.StatusBadRequest, fmt.Errorf("cannot submit pvp moves: %v", err))
			return
		}
		if req.Telemetry != nil {
			req.Telemetry.UserID = userID
			req.Telemetry.Scope = "pvp_match"
			req.Telemetry.ScopeID = detail.Match.ID
			req.Telemetry.UserAgent = r.UserAgent()
			_ = store.RecordGameplayTelemetry(r.Context(), req.Telemetry)
		}
		_ = store.AppendAuditLog(r.Context(), userID, "pvp.match.submitted", detail.Match.ID, clientIP(r), map[string]string{"status": detail.Match.Status, "winnerId": detail.Match.WinnerID})

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(detail)
	}
}

func PvPProgressHandler(store *db.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		var req pvpProgressRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid request payload", http.StatusBadRequest)
			return
		}
		userID := UserIDFromContext(r.Context())
		if userID == "" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		detail, err := store.UpdatePvPProgress(r.Context(), userID, req.MatchID, models.PvPProgress{
			CurrentProgress:    req.CurrentProgress,
			CurrentCombo:       req.CurrentCombo,
			MovesRemaining:     req.MovesRemaining,
			CompletionPercent:  req.CompletionPercent,
			Finished:           req.Finished,
			Disconnected:       req.Disconnected,
			LastClientSequence: req.LastClientSequence,
		})
		if err != nil {
			WriteMappedError(w, http.StatusBadRequest, err)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(detail)
	}
}
