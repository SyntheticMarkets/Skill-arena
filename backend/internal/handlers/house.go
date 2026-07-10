package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"

	"skill-arena/internal/db"
)

type startHouseChallengeRequest struct {
	TierID     string `json:"tierId"`
	WalletType string `json:"walletType"`
}

type startHouseChallengeResponse struct {
	SessionID  string   `json:"sessionId"`
	TierID     string   `json:"tierId"`
	TierName   string   `json:"tierName"`
	WalletType string   `json:"walletType"`
	Stake      float64  `json:"stake"`
	RewardRate float64  `json:"rewardRate"`
	Difficulty int      `json:"difficulty"`
	Maze       []string `json:"maze"`
	Width      int      `json:"width"`
	Height     int      `json:"height"`
	StartX     int      `json:"startX"`
	StartY     int      `json:"startY"`
	EndX       int      `json:"endX"`
	EndY       int      `json:"endY"`
}

func HouseTiersHandler(store *db.Store) http.HandlerFunc {
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

		tiers, err := store.ListHouseTiers(r.Context(), userID)
		if err != nil {
			http.Error(w, fmt.Sprintf("failed to load house tiers: %v", err), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(tiers)
	}
}

func StartHouseChallengeHandler(store *db.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}

		var req startHouseChallengeRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid request payload", http.StatusBadRequest)
			return
		}
		if req.TierID == "" {
			http.Error(w, "tierId is required", http.StatusBadRequest)
			return
		}

		userID := UserIDFromContext(r.Context())
		if userID == "" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		session, tier, err := store.StartHouseChallenge(r.Context(), userID, req.TierID, req.WalletType)
		if err != nil {
			http.Error(w, fmt.Sprintf("cannot start house challenge: %v", err), http.StatusBadRequest)
			return
		}
		_ = store.AppendAuditLog(r.Context(), userID, "house.challenge.started", session.ID, clientIP(r), map[string]string{"tier": tier.ID, "walletType": session.GameType})

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(startHouseChallengeResponse{
			SessionID:  session.ID,
			TierID:     tier.ID,
			TierName:   tier.Name,
			WalletType: session.GameType,
			Stake:      session.Stake,
			RewardRate: session.RewardRate,
			Difficulty: session.Difficulty,
			Maze:       session.MazeCells,
			Width:      session.Width,
			Height:     session.Height,
			StartX:     session.StartX,
			StartY:     session.StartY,
			EndX:       session.EndX,
			EndY:       session.EndY,
		})
	}
}
