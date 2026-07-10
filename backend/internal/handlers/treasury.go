package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"skill-arena/internal/db"
)

type treasuryWithdrawalActionRequest struct {
	WithdrawalID string `json:"withdrawalId"`
	Reason       string `json:"reason,omitempty"`
	ProviderRef  string `json:"providerRef,omitempty"`
}

func TreasuryHealthHandler(store *db.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}

		health, err := store.GetTreasuryHealth(r.Context())
		if err != nil {
			http.Error(w, fmt.Sprintf("failed to load treasury health: %v", err), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(health)
	}
}

func PublicTreasuryStatusHandler(store *db.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}

		health, err := store.GetTreasuryHealth(r.Context())
		if err != nil {
			http.Error(w, fmt.Sprintf("failed to load treasury status: %v", err), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"coverageRatio": health.CoverageRatio,
			"isSolvent":     health.IsSolvent,
			"houseExposure": health.HouseExposure,
		})
	}
}

func TreasuryApproveWithdrawalHandler(store *db.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		var req treasuryWithdrawalActionRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.WithdrawalID == "" {
			http.Error(w, "withdrawalId is required", http.StatusBadRequest)
			return
		}
		withdrawal, err := store.ApproveWithdrawal(r.Context(), UserIDFromContext(r.Context()), req.WithdrawalID, clientIP(r))
		if err != nil {
			WriteMappedError(w, http.StatusBadRequest, err)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(withdrawal)
	}
}

func TreasuryRejectWithdrawalHandler(store *db.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		var req treasuryWithdrawalActionRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.WithdrawalID == "" {
			http.Error(w, "withdrawalId is required", http.StatusBadRequest)
			return
		}
		withdrawal, err := store.RejectWithdrawal(r.Context(), UserIDFromContext(r.Context()), req.WithdrawalID, req.Reason, clientIP(r))
		if err != nil {
			WriteMappedError(w, http.StatusBadRequest, err)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(withdrawal)
	}
}

func TreasurySettleWithdrawalHandler(store *db.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		var req treasuryWithdrawalActionRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.WithdrawalID == "" {
			http.Error(w, "withdrawalId is required", http.StatusBadRequest)
			return
		}
		entries, err := store.SettleWithdrawal(r.Context(), UserIDFromContext(r.Context()), req.WithdrawalID, req.ProviderRef, clientIP(r))
		if err != nil {
			WriteMappedError(w, http.StatusBadRequest, err)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(entries)
	}
}

func HouseRiskHandler(store *db.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}

		tierID := strings.TrimPrefix(r.URL.Path, "/api/v1/admin/house-risk/")
		if tierID == "" {
			http.Error(w, "tier id required", http.StatusBadRequest)
			return
		}

		report, err := store.HouseRiskReport(r.Context(), tierID)
		if err != nil {
			http.Error(w, fmt.Sprintf("failed to load house risk: %v", err), http.StatusBadRequest)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(report)
	}
}
