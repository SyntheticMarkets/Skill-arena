package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"

	"skill-arena/internal/db"
)

func ActiveSeasonHandler(store *db.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}

		season, err := store.GetActiveSeason(r.Context())
		if err != nil {
			http.Error(w, fmt.Sprintf("failed to load active season: %v", err), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(season)
	}
}

func SeasonLeaderboardHandler(store *db.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}

		leaderboard, err := store.GetSeasonLeaderboard(r.Context())
		if err != nil {
			http.Error(w, fmt.Sprintf("failed to load season leaderboard: %v", err), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(leaderboard)
	}
}

func AchievementCatalogHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(db.AchievementCatalog())
	}
}
