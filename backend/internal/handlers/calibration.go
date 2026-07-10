package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"

	"skill-arena/internal/db"
)

func StartCalibrationHandler(store *db.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}

		userID := UserIDFromContext(r.Context())
		if userID == "" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		session, err := store.StartDailyCalibration(r.Context(), userID)
		if err != nil {
			http.Error(w, fmt.Sprintf("cannot start calibration: %v", err), http.StatusBadRequest)
			return
		}
		_ = store.AppendAuditLog(r.Context(), userID, "calibration.started", session.ID, clientIP(r), nil)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(startGameResponse{
			SessionID: session.ID,
			GameType:  session.GameType,
			Stake:     session.Stake,
			Maze:      session.MazeCells,
			Width:     session.Width,
			Height:    session.Height,
			StartX:    session.StartX,
			StartY:    session.StartY,
			EndX:      session.EndX,
			EndY:      session.EndY,
		})
	}
}

func BaselineHandler(store *db.Store) http.HandlerFunc {
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

		baseline, err := store.GetBaselineByUserID(r.Context(), userID)
		if err != nil {
			http.Error(w, fmt.Sprintf("failed to load baseline: %v", err), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(baseline)
	}
}

func AdminBaselinesHandler(store *db.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}

		baselines, err := store.ListBaselines(r.Context())
		if err != nil {
			http.Error(w, fmt.Sprintf("failed to load baselines: %v", err), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(baselines)
	}
}
