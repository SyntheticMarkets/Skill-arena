package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"skill-arena/internal/db"
	"skill-arena/internal/models"
)

type tournamentRegisterRequest struct {
	TournamentID string `json:"tournamentId"`
}

type tournamentBracketRequest struct {
	TournamentID string `json:"tournamentId"`
}

type tournamentResultRequest struct {
	TournamentID string `json:"tournamentId"`
	MatchID      string `json:"matchId"`
	WinnerID     string `json:"winnerId"`
}

type tournamentSubmitRequest struct {
	TournamentID   string                    `json:"tournamentId"`
	MatchID        string                    `json:"matchId"`
	ClickedLineIDs []string                  `json:"clickedLineIds"`
	Telemetry      *models.GameplayTelemetry `json:"telemetry,omitempty"`
}

func TournamentListHandler(store *db.Store) http.HandlerFunc {
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

		tournaments, err := store.ListTournaments(r.Context(), userID)
		if err != nil {
			http.Error(w, fmt.Sprintf("failed to load tournaments: %v", err), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(tournaments)
	}
}

func TournamentDetailHandler(store *db.Store) http.HandlerFunc {
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

		tournamentID := strings.TrimPrefix(r.URL.Path, "/api/v1/tournaments/")
		if tournamentID == "" {
			http.Error(w, "tournament id required", http.StatusBadRequest)
			return
		}

		detail, err := store.GetTournamentDetail(r.Context(), tournamentID, userID)
		if err != nil {
			http.Error(w, fmt.Sprintf("tournament not found: %v", err), http.StatusNotFound)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(detail)
	}
}

func TournamentRegisterHandler(store *db.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}

		var req tournamentRegisterRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid request payload", http.StatusBadRequest)
			return
		}
		if req.TournamentID == "" {
			http.Error(w, "tournamentId is required", http.StatusBadRequest)
			return
		}

		userID := UserIDFromContext(r.Context())
		if userID == "" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		participant, err := store.RegisterTournament(r.Context(), userID, req.TournamentID)
		if err != nil {
			http.Error(w, fmt.Sprintf("failed to register tournament: %v", err), http.StatusBadRequest)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(participant)
	}
}

func TournamentSubmitMatchHandler(store *db.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}

		var req tournamentSubmitRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid request payload", http.StatusBadRequest)
			return
		}
		if req.TournamentID == "" || req.MatchID == "" || len(req.ClickedLineIDs) == 0 {
			http.Error(w, "tournamentId, matchId, and clickedLineIds are required", http.StatusBadRequest)
			return
		}

		userID := UserIDFromContext(r.Context())
		if userID == "" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		detail, err := store.SubmitTournamentMatchClicks(r.Context(), userID, req.TournamentID, req.MatchID, req.ClickedLineIDs)
		if err != nil {
			http.Error(w, fmt.Sprintf("failed to submit tournament match: %v", err), http.StatusBadRequest)
			return
		}
		if req.Telemetry != nil {
			req.Telemetry.UserID = userID
			req.Telemetry.Scope = "tournament_match"
			req.Telemetry.ScopeID = req.MatchID
			req.Telemetry.UserAgent = r.UserAgent()
			_ = store.RecordGameplayTelemetry(r.Context(), req.Telemetry)
		}
		_ = store.AppendAuditLog(r.Context(), userID, "tournament.match.submitted", req.MatchID, clientIP(r), map[string]string{"tournamentId": req.TournamentID})

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(detail)
	}
}

func AdminGenerateTournamentBracketHandler(store *db.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}

		var req tournamentBracketRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid request payload", http.StatusBadRequest)
			return
		}
		if req.TournamentID == "" {
			http.Error(w, "tournamentId is required", http.StatusBadRequest)
			return
		}

		matches, err := store.GenerateTournamentBracket(r.Context(), UserIDFromContext(r.Context()), req.TournamentID)
		if err != nil {
			http.Error(w, fmt.Sprintf("failed to generate bracket: %v", err), http.StatusBadRequest)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(matches)
	}
}

func AdminReportTournamentResultHandler(store *db.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}

		var req tournamentResultRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid request payload", http.StatusBadRequest)
			return
		}
		if req.TournamentID == "" || req.MatchID == "" || req.WinnerID == "" {
			http.Error(w, "tournamentId, matchId, and winnerId are required", http.StatusBadRequest)
			return
		}

		match, err := store.ReportTournamentMatchResult(r.Context(), UserIDFromContext(r.Context()), req.TournamentID, req.MatchID, req.WinnerID)
		if err != nil {
			http.Error(w, fmt.Sprintf("failed to report result: %v", err), http.StatusBadRequest)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(match)
	}
}
