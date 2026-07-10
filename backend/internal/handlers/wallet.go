package handlers

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"skill-arena/internal/db"
)

type walletSummaryResponse struct {
	LiveBalance        float64 `json:"liveBalance"`
	LiveLockedBalance  float64 `json:"liveLockedBalance"`
	AvailableLive      float64 `json:"availableLiveBalance"`
	DemoBalance        float64 `json:"demoBalance"`
	DemoLockedBalance  float64 `json:"demoLockedBalance"`
	AvailableDemo      float64 `json:"availableDemoBalance"`
	PendingWithdrawals float64 `json:"pendingWithdrawals"`
	BonusBalance       float64 `json:"bonusBalance"`
}

type walletBalanceResponse struct {
	LiveBalance float64 `json:"liveBalance"`
	DemoBalance float64 `json:"demoBalance"`
}

type walletAvailableResponse struct {
	AvailableLive float64 `json:"availableLiveBalance"`
	AvailableDemo float64 `json:"availableDemoBalance"`
}

type walletTransactionRequest struct {
	Amount    float64 `json:"amount"`
	Currency  string  `json:"currency,omitempty"`
	Reference string  `json:"reference,omitempty"`
	Provider  string  `json:"provider,omitempty"`
	Method    string  `json:"method,omitempty"`
	Country   string  `json:"country,omitempty"`
}

type walletLockRequest struct {
	WalletType string  `json:"walletType"`
	Amount     float64 `json:"amount"`
	Currency   string  `json:"currency,omitempty"`
	Reference  string  `json:"reference,omitempty"`
}

type ledgerEntryResponse struct {
	ID              string            `json:"id"`
	TransactionType string            `json:"transactionType"`
	Amount          float64           `json:"amount"`
	BalanceBefore   float64           `json:"balanceBefore"`
	BalanceAfter    float64           `json:"balanceAfter"`
	Currency        string            `json:"currency"`
	Reference       string            `json:"reference,omitempty"`
	Metadata        map[string]string `json:"metadata,omitempty"`
	CreatedAt       string            `json:"createdAt"`
}

func WalletHandler(store *db.Store) http.HandlerFunc {
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
		json.NewEncoder(w).Encode(walletSummaryResponse{
			LiveBalance:        wallet.LiveBalance,
			LiveLockedBalance:  wallet.LiveLockedBalance,
			AvailableLive:      availableLive,
			DemoBalance:        wallet.DemoBalance,
			DemoLockedBalance:  wallet.DemoLockedBalance,
			AvailableDemo:      availableDemo,
			PendingWithdrawals: wallet.PendingWithdrawals,
			BonusBalance:       wallet.BonusBalance,
		})
	}
}

func WalletBalanceHandler(store *db.Store) http.HandlerFunc {
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

		wallet, err := store.GetWalletByUserID(r.Context(), userID)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(walletBalanceResponse{
			LiveBalance: wallet.LiveBalance,
			DemoBalance: wallet.DemoBalance,
		})
	}
}

func WalletAvailableHandler(store *db.Store) http.HandlerFunc {
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
		json.NewEncoder(w).Encode(walletAvailableResponse{
			AvailableLive: availableLive,
			AvailableDemo: availableDemo,
		})
	}
}

func WalletTransactionsHandler(store *db.Store) http.HandlerFunc {
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

		entries, err := store.GetLedgerEntriesByUserID(r.Context(), userID)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		response := make([]ledgerEntryResponse, 0, len(entries))
		for _, entry := range entries {
			response = append(response, ledgerEntryResponse{
				ID:              entry.ID,
				TransactionType: entry.TransactionType,
				Amount:          entry.Amount,
				BalanceBefore:   entry.BalanceBefore,
				BalanceAfter:    entry.BalanceAfter,
				Currency:        entry.Currency,
				Reference:       entry.Reference,
				Metadata:        entry.Metadata,
				CreatedAt:       entry.CreatedAt.Format(time.RFC3339),
			})
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}
}

func WalletDepositHandler(store *db.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}

		var req walletTransactionRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid request payload", http.StatusBadRequest)
			return
		}

		if req.Amount <= 0 {
			http.Error(w, "amount must be greater than zero", http.StatusBadRequest)
			return
		}
		idempotencyKey := idempotencyKeyFromRequest(r)
		if idempotencyKey == "" {
			http.Error(w, "Idempotency-Key header is required", http.StatusBadRequest)
			return
		}

		userID := UserIDFromContext(r.Context())
		if userID == "" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		user, err := store.GetUserByID(r.Context(), userID)
		if err != nil {
			http.Error(w, "failed to load user", http.StatusInternalServerError)
			return
		}
		if !user.EmailVerified {
			http.Error(w, "email verification required before deposit", http.StatusForbidden)
			return
		}

		metadata := map[string]string{"providerFeeAbsorbed": "true"}
		metadata["idempotencyKey"] = idempotencyKey
		metadata["requestHash"] = financialRequestHash("deposit", userID, req)
		if req.Country != "" {
			metadata["country"] = req.Country
		}
		depositSession, err := store.CreateDepositSession(r.Context(), userID, req.Provider, req.Method, req.Amount, req.Currency, req.Reference, metadata)
		if err != nil {
			http.Error(w, fmt.Sprintf("failed to deposit: %v", err), http.StatusBadRequest)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusAccepted)
		json.NewEncoder(w).Encode(depositSession)
	}
}

func WalletWithdrawHandler(store *db.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}

		var req walletTransactionRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid request payload", http.StatusBadRequest)
			return
		}

		if req.Amount <= 0 {
			http.Error(w, "amount must be greater than zero", http.StatusBadRequest)
			return
		}
		idempotencyKey := idempotencyKeyFromRequest(r)
		if idempotencyKey == "" {
			http.Error(w, "Idempotency-Key header is required", http.StatusBadRequest)
			return
		}

		userID := UserIDFromContext(r.Context())
		if userID == "" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		user, err := store.GetUserByID(r.Context(), userID)
		if err != nil {
			http.Error(w, "failed to load user", http.StatusInternalServerError)
			return
		}

		if !user.EmailVerified {
			http.Error(w, "email verification required for withdrawals", http.StatusForbidden)
			return
		}
		if req.Amount > 500 && user.KYCStatus != "approved" {
			http.Error(w, "KYC approval required for withdrawals over 500", http.StatusForbidden)
			return
		}

		metadata := map[string]string{}
		metadata["idempotencyKey"] = idempotencyKey
		metadata["requestHash"] = financialRequestHash("withdrawal", userID, req)
		if req.Country != "" {
			metadata["country"] = req.Country
		}
		withdrawal, review, err := store.CreateWithdrawalRequest(r.Context(), userID, req.Provider, req.Method, req.Amount, req.Currency, req.Reference, metadata)
		if err != nil {
			WriteMappedError(w, http.StatusBadRequest, fmt.Errorf("failed to withdraw: %v", err))
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusAccepted)
		json.NewEncoder(w).Encode(map[string]any{"withdrawal": withdrawal, "amlReview": review})
	}
}

func idempotencyKeyFromRequest(r *http.Request) string {
	return strings.TrimSpace(r.Header.Get("Idempotency-Key"))
}

func financialRequestHash(operation, userID string, req walletTransactionRequest) string {
	sum := sha256.Sum256([]byte(fmt.Sprintf("%s|%s|%.2f|%s|%s|%s|%s",
		operation,
		userID,
		req.Amount,
		strings.ToUpper(strings.TrimSpace(req.Currency)),
		strings.ToLower(strings.TrimSpace(req.Provider)),
		strings.ToLower(strings.TrimSpace(req.Method)),
		strings.ToUpper(strings.TrimSpace(req.Country)),
	)))
	return hex.EncodeToString(sum[:])
}

func WalletLockHandler(store *db.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}

		var req walletLockRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid request payload", http.StatusBadRequest)
			return
		}

		if req.Amount <= 0 || req.WalletType == "" {
			http.Error(w, "walletType and amount are required", http.StatusBadRequest)
			return
		}

		userID := UserIDFromContext(r.Context())
		if userID == "" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		entry, err := store.LockWalletTokens(r.Context(), userID, req.WalletType, req.Amount, req.Currency, req.Reference, nil)
		if err != nil {
			http.Error(w, fmt.Sprintf("failed to lock tokens: %v", err), http.StatusBadRequest)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(entry)
	}
}

func WalletUnlockHandler(store *db.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}

		var req walletLockRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid request payload", http.StatusBadRequest)
			return
		}

		if req.Amount <= 0 || req.WalletType == "" {
			http.Error(w, "walletType and amount are required", http.StatusBadRequest)
			return
		}

		userID := UserIDFromContext(r.Context())
		if userID == "" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		entry, err := store.UnlockWalletTokens(r.Context(), userID, req.WalletType, req.Amount, req.Currency, req.Reference, nil)
		if err != nil {
			http.Error(w, fmt.Sprintf("failed to unlock tokens: %v", err), http.StatusBadRequest)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(entry)
	}
}
