package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"skill-arena/internal/db"
)

func ReplayListHandler(store *db.Store) http.HandlerFunc {
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

		reports, err := store.GetReplayReportsByUserID(r.Context(), userID)
		if err != nil {
			http.Error(w, fmt.Sprintf("failed to load replays: %v", err), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(reports)
	}
}

func ReplayDetailHandler(store *db.Store) http.HandlerFunc {
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

		sessionID := strings.TrimPrefix(r.URL.Path, "/api/v1/replays/")
		if sessionID == "" {
			http.Error(w, "session id required", http.StatusBadRequest)
			return
		}

		report, err := store.GetReplayReport(r.Context(), sessionID)
		if err != nil {
			WriteMappedError(w, http.StatusNotFound, fmt.Errorf("replay not found: %v", err))
			return
		}
		if report.UserID != userID {
			http.Error(w, "replay does not belong to user", http.StatusForbidden)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(report)
	}
}

func AdminReplayDetailHandler(store *db.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}

		sessionID := strings.TrimPrefix(r.URL.Path, "/api/v1/admin/replays/")
		if sessionID == "" {
			http.Error(w, "session id required", http.StatusBadRequest)
			return
		}

		report, err := store.GetReplayReport(r.Context(), sessionID)
		if err != nil {
			WriteMappedError(w, http.StatusNotFound, fmt.Errorf("replay not found: %v", err))
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(report)
	}
}
