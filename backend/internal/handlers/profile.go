package handlers

import (
	"encoding/json"
	"net/http"

	"skill-arena/internal/db"
)

type profileResponse struct {
	ID                 string  `json:"id"`
	Email              string  `json:"email"`
	EmailVerified      bool    `json:"emailVerified"`
	KYCStatus          string  `json:"kycStatus"`
	LiveBalance        float64 `json:"liveBalance"`
	LiveLockedBalance  float64 `json:"liveLockedBalance"`
	AvailableLive      float64 `json:"availableLiveBalance"`
	DemoBalance        float64 `json:"demoBalance"`
	DemoLockedBalance  float64 `json:"demoLockedBalance"`
	AvailableDemo      float64 `json:"availableDemoBalance"`
	PendingWithdrawals float64 `json:"pendingWithdrawals"`
	BonusBalance       float64 `json:"bonusBalance"`
	XP                 int     `json:"xp"`
	Level              int     `json:"level"`
	Prestige           int     `json:"prestige"`
	EloRating          int     `json:"eloRating"`
	LeagueTier         string  `json:"leagueTier"`
	SeasonPoints       int     `json:"seasonPoints"`
	LegacyPoints       int     `json:"legacyPoints"`
	HouseReputation    int     `json:"houseReputation"`
	MatchesPlayed      int     `json:"matchesPlayed"`
	Wins               int     `json:"wins"`
	Losses             int     `json:"losses"`
	CurrentStreak      int     `json:"currentStreak"`
	BestMoves          int     `json:"bestMoves,omitempty"`
	TrustScore         float64 `json:"trustScore"`
}

func ProfileHandler(store *db.Store) http.HandlerFunc {
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

		user, err := store.GetUserByID(r.Context(), userID)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		progression, err := store.GetProgressionByUserID(r.Context(), userID)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		wallet, err := store.GetWalletByUserID(r.Context(), userID)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		availableLive := wallet.LiveBalance - wallet.LiveLockedBalance - wallet.PendingWithdrawals
		if availableLive < 0 {
			availableLive = 0
		}
		availableDemo := wallet.DemoBalance - wallet.DemoLockedBalance
		if availableDemo < 0 {
			availableDemo = 0
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(profileResponse{
			ID:                 user.ID,
			Email:              user.Email,
			EmailVerified:      user.EmailVerified,
			KYCStatus:          user.KYCStatus,
			LiveBalance:        wallet.LiveBalance,
			LiveLockedBalance:  wallet.LiveLockedBalance,
			AvailableLive:      availableLive,
			DemoBalance:        wallet.DemoBalance,
			DemoLockedBalance:  wallet.DemoLockedBalance,
			AvailableDemo:      availableDemo,
			PendingWithdrawals: wallet.PendingWithdrawals,
			BonusBalance:       wallet.BonusBalance,
			XP:                 progression.XP,
			Level:              progression.Level,
			Prestige:           progression.Prestige,
			EloRating:          progression.EloRating,
			LeagueTier:         progression.LeagueTier,
			SeasonPoints:       progression.SeasonPoints,
			LegacyPoints:       progression.LegacyPoints,
			HouseReputation:    progression.HouseRep,
			MatchesPlayed:      progression.MatchesPlayed,
			Wins:               progression.Wins,
			Losses:             progression.Losses,
			CurrentStreak:      progression.CurrentStreak,
			BestMoves:          progression.BestMoves,
			TrustScore:         progression.TrustScore,
		})
	}
}
