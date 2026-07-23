package handlers

import (
	"encoding/json"
	"net/http"

	"skill-arena/internal/db"
)

type profileResponse struct {
	ID                 string  `json:"id"`
	Email              string  `json:"email"`
	Username           string  `json:"username"`
	DisplayName        string  `json:"displayName"`
	AvatarURL          string  `json:"avatarUrl,omitempty"`
	Country            string  `json:"country"`
	Language           string  `json:"language"`
	EmailVerified      bool    `json:"emailVerified"`
	KYCStatus          string  `json:"kycStatus"`
	AccountStatus      string  `json:"accountStatus"`
	MFAEnabled         bool    `json:"mfaEnabled"`
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
		if r.Method != http.MethodGet && r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}

		userID := UserIDFromContext(r.Context())
		if userID == "" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		if r.Method == http.MethodPost {
			var request struct {
				Username    string `json:"username"`
				DisplayName string `json:"displayName"`
				AvatarURL   string `json:"avatarUrl"`
				Country     string `json:"country"`
				Language    string `json:"language"`
			}
			if err := decodeJSON(r, &request); err != nil {
				WriteAPIError(w, http.StatusBadRequest, ErrInvalidRequest, "request body is invalid")
				return
			}
			if !validAvatarKey(request.AvatarURL) {
				WriteAPIError(w, http.StatusBadRequest, ErrInvalidRequest, "avatarUrl must identify an approved Arena avatar")
				return
			}
			current, err := store.GetPlayerProfile(r.Context(), userID)
			if err != nil {
				WriteMappedError(w, http.StatusInternalServerError, err)
				return
			}
			current.Username = request.Username
			current.DisplayName = request.DisplayName
			current.AvatarURL = request.AvatarURL
			current.Country = request.Country
			current.Language = request.Language
			updated, err := store.UpdatePlayerProfile(r.Context(), current)
			if err != nil {
				WriteMappedError(w, http.StatusBadRequest, err)
				return
			}
			_ = store.AppendAuditLog(r.Context(), userID, "profile.updated", userID, clientIP(r), nil)
			writeJSON(w, http.StatusOK, updated)
			return
		}

		user, err := store.GetUserByID(r.Context(), userID)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		playerProfile, err := store.GetPlayerProfile(r.Context(), userID)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		mfa, err := store.GetMFASettings(r.Context(), userID)
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
			Username:           playerProfile.Username,
			DisplayName:        playerProfile.DisplayName,
			AvatarURL:          playerProfile.AvatarURL,
			Country:            playerProfile.Country,
			Language:           playerProfile.Language,
			EmailVerified:      user.EmailVerified,
			KYCStatus:          user.KYCStatus,
			AccountStatus:      user.Status,
			MFAEnabled:         mfa.Enabled,
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
