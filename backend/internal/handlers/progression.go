package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"

	"skill-arena/internal/db"
)

func ProgressionHandler(store *db.Store) http.HandlerFunc {
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

		progression, err := store.GetProgressionByUserID(r.Context(), userID)
		if err != nil {
			http.Error(w, fmt.Sprintf("failed to load progression: %v", err), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(progression)
	}
}

func AchievementsHandler(store *db.Store) http.HandlerFunc {
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

		achievements, err := store.GetAchievementsByUserID(r.Context(), userID)
		if err != nil {
			http.Error(w, fmt.Sprintf("failed to load achievements: %v", err), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(achievements)
	}
}
