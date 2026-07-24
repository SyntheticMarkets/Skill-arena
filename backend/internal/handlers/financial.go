package handlers

import (
	"bytes"
	"context"
	"encoding/csv"
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"skill-arena/internal/config"
	"skill-arena/internal/db"
	"skill-arena/internal/models"
	"skill-arena/internal/payments"
)

type FinancialHandlers struct {
	store    *db.Store
	settings *config.RuntimeSettings
	payments *payments.Core
}

var errTreasuryReserveInsufficient = errors.New("treasury reserve check failed")

func NewFinancialHandlers(store *db.Store, settings *config.RuntimeSettings) *FinancialHandlers {
	return &FinancialHandlers{
		store: store, settings: settings,
		payments: payments.CoreFromSettings(settings.Payments),
	}
}

type financialIntentRequest struct {
	AmountMinor models.MinorUnits `json:"amountMinor"`
	Currency    string            `json:"currency"`
	Method      string            `json:"method"`
	ReturnURL   string            `json:"returnUrl,omitempty"`
	CancelURL   string            `json:"cancelUrl,omitempty"`
}

type financialAssessmentRequest struct {
	Country       string `json:"country"`
	Occupation    string `json:"occupation"`
	SourceOfFunds string `json:"sourceOfFunds"`
}

type financialLimitsRequest struct {
	DailyDepositMinor      models.MinorUnits `json:"dailyDepositMinor"`
	MonthlyDepositMinor    models.MinorUnits `json:"monthlyDepositMinor"`
	DailyWithdrawalMinor   models.MinorUnits `json:"dailyWithdrawalMinor"`
	MonthlyWithdrawalMinor models.MinorUnits `json:"monthlyWithdrawalMinor"`
	CoolingOffDays         int               `json:"coolingOffDays,omitempty"`
	SelfExcludeDays        int               `json:"selfExcludeDays,omitempty"`
}

func (h *FinancialHandlers) Overview(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	userID := UserIDFromContext(r.Context())
	wallet, err := h.store.GetFinancialWallet(r.Context(), userID)
	if err != nil {
		WriteMappedError(w, http.StatusInternalServerError, err)
		return
	}
	assessment, err := h.store.GetFinancialAssessment(r.Context(), userID)
	if err != nil {
		WriteMappedError(w, http.StatusInternalServerError, err)
		return
	}
	limits, err := h.store.GetFinancialLimits(r.Context(), userID)
	if err != nil {
		WriteMappedError(w, http.StatusInternalServerError, err)
		return
	}
	deposits, err := h.store.ListFinancialDeposits(r.Context(), userID)
	if err != nil {
		WriteMappedError(w, http.StatusInternalServerError, err)
		return
	}
	withdrawals, err := h.store.ListFinancialWithdrawals(r.Context(), userID)
	if err != nil {
		WriteMappedError(w, http.StatusInternalServerError, err)
		return
	}
	user, err := h.store.GetUserByID(r.Context(), userID)
	if err != nil {
		WriteMappedError(w, http.StatusInternalServerError, err)
		return
	}
	country := assessment.Country
	if country == "" {
		country = user.Country
	}
	if country == "" {
		country = h.settings.Financial.DefaultCountry
	}
	allowedMethods := []string{"card", "eft", "bank_transfer"}
	if policy, ok := h.settings.Financial.Policy(country); ok {
		allowedMethods = policy.PaymentMethods
	}
	methods := h.payments.Methods(r.Context(), country, wallet.Currency, allowedMethods)
	writeJSON(w, http.StatusOK, models.FinancialOverview{
		Wallet: *wallet, Assessment: *assessment, Limits: *limits,
		VerificationStatus: user.KYCStatus, PaymentMethods: methods,
		Deposits: deposits, Withdrawals: withdrawals,
	})
}

func (h *FinancialHandlers) Transactions(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	entries, err := h.store.ListFinancialJournal(r.Context(), UserIDFromContext(r.Context()), time.Unix(0, 0), time.Now().UTC().Add(time.Second))
	if err != nil {
		WriteMappedError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, entries)
}

func (h *FinancialHandlers) Statement(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	now := time.Now().UTC()
	from := now.AddDate(0, -1, 0)
	to := now.Add(time.Second)
	if raw := r.URL.Query().Get("from"); raw != "" {
		parsed, err := time.Parse(time.RFC3339, raw)
		if err != nil {
			WriteAPIError(w, http.StatusBadRequest, "INVALID_PERIOD", "Statement start must use RFC3339.")
			return
		}
		from = parsed
	}
	if raw := r.URL.Query().Get("to"); raw != "" {
		parsed, err := time.Parse(time.RFC3339, raw)
		if err != nil {
			WriteAPIError(w, http.StatusBadRequest, "INVALID_PERIOD", "Statement end must use RFC3339.")
			return
		}
		to = parsed
	}
	if !from.Before(to) || to.Sub(from) > 366*24*time.Hour {
		WriteAPIError(w, http.StatusBadRequest, "INVALID_PERIOD", "Statement periods must be ordered and no longer than one year.")
		return
	}
	statement, err := h.store.FinancialStatement(r.Context(), UserIDFromContext(r.Context()), from, to)
	if err != nil {
		WriteMappedError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, statement)
}

func (h *FinancialHandlers) StatementExport(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	userID := UserIDFromContext(r.Context())
	to := time.Now().UTC()
	from := to.AddDate(0, -1, 0)
	statement, err := h.store.FinancialStatement(r.Context(), userID, from, to)
	if err != nil {
		WriteMappedError(w, http.StatusInternalServerError, err)
		return
	}
	var output bytes.Buffer
	writer := csv.NewWriter(&output)
	_ = writer.Write([]string{"timestamp", "reference", "description", "direction", "amount_minor", "currency", "balance_after_minor", "entry_hash"})
	for _, entry := range statement.Entries {
		_ = writer.Write([]string{
			entry.CreatedAt.Format(time.RFC3339), entry.ReferenceID, entry.Description, entry.Direction,
			strconv.FormatInt(int64(entry.AmountMinor), 10), entry.Currency,
			strconv.FormatInt(int64(entry.BalanceAfterMinor), 10), entry.EntryHash,
		})
	}
	writer.Flush()
	if err := writer.Error(); err != nil {
		WriteMappedError(w, http.StatusInternalServerError, err)
		return
	}
	artifact, err := h.store.StoreFinancialArtifact(r.Context(), userID, "statement", "text/csv", output.Bytes())
	if err != nil {
		WriteAPIError(w, http.StatusServiceUnavailable, "ARTIFACT_STORAGE_UNAVAILABLE", "The statement could not be stored securely.")
		return
	}
	writeJSON(w, http.StatusCreated, artifact)
}

func (h *FinancialHandlers) TransactionExport(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	userID := UserIDFromContext(r.Context())
	entries, err := h.store.ListFinancialJournal(r.Context(), userID, time.Unix(0, 0), time.Now().UTC().Add(time.Second))
	if err != nil {
		WriteMappedError(w, http.StatusInternalServerError, err)
		return
	}
	var output bytes.Buffer
	writer := csv.NewWriter(&output)
	_ = writer.Write([]string{"sequence", "timestamp", "account", "reference_type", "reference", "description", "direction", "amount_minor", "currency", "balance_after_minor", "previous_hash", "entry_hash"})
	for _, entry := range entries {
		_ = writer.Write([]string{
			strconv.FormatInt(entry.Sequence, 10), entry.CreatedAt.Format(time.RFC3339), entry.Account,
			entry.ReferenceType, entry.ReferenceID, entry.Description, entry.Direction,
			strconv.FormatInt(int64(entry.AmountMinor), 10), entry.Currency,
			strconv.FormatInt(int64(entry.BalanceAfterMinor), 10), entry.PreviousHash, entry.EntryHash,
		})
	}
	writer.Flush()
	if err := writer.Error(); err != nil {
		WriteMappedError(w, http.StatusInternalServerError, err)
		return
	}
	artifact, err := h.store.StoreFinancialArtifact(r.Context(), userID, "financial_export", "text/csv", output.Bytes())
	if err != nil {
		WriteAPIError(w, http.StatusServiceUnavailable, "ARTIFACT_STORAGE_UNAVAILABLE", "The transaction export could not be stored securely.")
		return
	}
	writeJSON(w, http.StatusCreated, artifact)
}

func (h *FinancialHandlers) Artifact(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	artifactID := strings.TrimPrefix(r.URL.Path, "/api/v1/financial/artifacts/")
	item, data, err := h.store.GetFinancialArtifact(r.Context(), UserIDFromContext(r.Context()), artifactID)
	if err != nil {
		WriteAPIError(w, http.StatusNotFound, "ARTIFACT_NOT_FOUND", "The financial artifact was not found.")
		return
	}
	w.Header().Set("Content-Type", item.ContentType)
	w.Header().Set("Content-Disposition", `attachment; filename="skill-arena-`+item.Type+`-`+item.ID+`.csv"`)
	w.Header().Set("X-Content-SHA256", item.SHA256)
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(data)
}

func (h *FinancialHandlers) Evidence(w http.ResponseWriter, r *http.Request) {
	userID := UserIDFromContext(r.Context())
	if r.Method == http.MethodGet {
		items, err := h.store.ListFinancialEvidence(r.Context(), userID)
		if err != nil {
			WriteMappedError(w, http.StatusInternalServerError, err)
			return
		}
		writeJSON(w, http.StatusOK, items)
		return
	}
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	if err := r.ParseMultipartForm(10 << 20); err != nil {
		WriteAPIError(w, http.StatusBadRequest, "INVALID_EVIDENCE", "Evidence must be a PDF, JPEG, or PNG no larger than 10 MiB.")
		return
	}
	evidenceType := r.FormValue("type")
	file, header, err := r.FormFile("file")
	if err != nil {
		WriteAPIError(w, http.StatusBadRequest, "INVALID_EVIDENCE", "An evidence file is required.")
		return
	}
	defer file.Close()
	data, err := io.ReadAll(io.LimitReader(file, (10<<20)+1))
	if err != nil || len(data) > 10<<20 {
		WriteAPIError(w, http.StatusBadRequest, "INVALID_EVIDENCE", "Evidence must be no larger than 10 MiB.")
		return
	}
	contentType := http.DetectContentType(data)
	if declared := strings.TrimSpace(header.Header.Get("Content-Type")); declared != "" && contentType == "application/octet-stream" {
		contentType = declared
	}
	item, err := h.store.StoreFinancialEvidence(r.Context(), userID, evidenceType, contentType, data)
	if err != nil {
		WriteAPIError(w, http.StatusBadRequest, "INVALID_EVIDENCE", err.Error())
		return
	}
	_ = h.notify(r, userID, "financial", "Evidence received", "Your document was stored securely and is awaiting verification.", "/wallet", map[string]string{"evidenceId": item.ID})
	writeJSON(w, http.StatusCreated, item)
}

func (h *FinancialHandlers) Assessment(w http.ResponseWriter, r *http.Request) {
	userID := UserIDFromContext(r.Context())
	if r.Method == http.MethodGet {
		item, err := h.store.GetFinancialAssessment(r.Context(), userID)
		if err != nil {
			WriteMappedError(w, http.StatusInternalServerError, err)
			return
		}
		writeJSON(w, http.StatusOK, item)
		return
	}
	if r.Method != http.MethodPut {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	var request financialAssessmentRequest
	if err := decodeJSON(r, &request); err != nil {
		WriteAPIError(w, http.StatusBadRequest, "INVALID_ASSESSMENT", err.Error())
		return
	}
	policy, supported := h.settings.Financial.Policy(request.Country)
	if !supported || !validOccupation(request.Occupation) || (policy.SourceOfFundsRequired && !validSourceOfFunds(request.SourceOfFunds)) {
		WriteAPIError(w, http.StatusBadRequest, "INVALID_ASSESSMENT", "Select a supported country, occupation, and source of funds.")
		return
	}
	user, userErr := h.store.GetUserByID(r.Context(), userID)
	if userErr != nil {
		WriteMappedError(w, http.StatusInternalServerError, userErr)
		return
	}
	if user.DateOfBirth == nil || ageAt(*user.DateOfBirth, time.Now().UTC()) < policy.MinimumAge {
		WriteAPIError(w, http.StatusForbidden, "JURISDICTION_AGE_REQUIRED", "You do not meet this jurisdiction's minimum age requirement.")
		return
	}
	item, err := h.store.SaveFinancialAssessment(r.Context(), models.FinancialAssessment{
		UserID: userID, Country: request.Country, Occupation: request.Occupation, SourceOfFunds: request.SourceOfFunds,
	})
	if err != nil {
		WriteMappedError(w, http.StatusBadRequest, err)
		return
	}
	limits, limitsErr := h.store.GetFinancialLimits(r.Context(), userID)
	if limitsErr == nil {
		limits.Currency = policy.Currency
		limits.DailyDepositMinor = minMinor(limits.DailyDepositMinor, models.MinorUnits(policy.DailyDepositMinor))
		limits.MonthlyDepositMinor = minMinor(limits.MonthlyDepositMinor, models.MinorUnits(policy.MonthlyDepositMinor))
		limits.DailyWithdrawalMinor = minMinor(limits.DailyWithdrawalMinor, models.MinorUnits(policy.DailyWithdrawalMinor))
		limits.MonthlyWithdrawalMinor = minMinor(limits.MonthlyWithdrawalMinor, models.MinorUnits(policy.MonthlyWithdrawalMinor))
		_, limitsErr = h.store.SaveFinancialLimits(r.Context(), *limits)
	}
	if limitsErr != nil {
		WriteMappedError(w, http.StatusInternalServerError, limitsErr)
		return
	}
	_ = h.notify(r, userID, "financial", "Financial assessment submitted", "Your assessment is under review. Practice remains available.", "/wallet", map[string]string{"assessmentStatus": item.Status})
	writeJSON(w, http.StatusOK, item)
}

func (h *FinancialHandlers) Limits(w http.ResponseWriter, r *http.Request) {
	userID := UserIDFromContext(r.Context())
	if r.Method == http.MethodGet {
		item, err := h.store.GetFinancialLimits(r.Context(), userID)
		if err != nil {
			WriteMappedError(w, http.StatusInternalServerError, err)
			return
		}
		writeJSON(w, http.StatusOK, item)
		return
	}
	if r.Method != http.MethodPut {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	var request financialLimitsRequest
	if err := decodeJSON(r, &request); err != nil {
		WriteAPIError(w, http.StatusBadRequest, "INVALID_LIMITS", err.Error())
		return
	}
	current, err := h.store.GetFinancialLimits(r.Context(), userID)
	if err != nil {
		WriteMappedError(w, http.StatusInternalServerError, err)
		return
	}
	// Player-controlled changes may lower limits immediately. Increases remain
	// subject to the jurisdiction policy defaults and future CRM review.
	if request.DailyDepositMinor > current.DailyDepositMinor || request.MonthlyDepositMinor > current.MonthlyDepositMinor ||
		request.DailyWithdrawalMinor > current.DailyWithdrawalMinor || request.MonthlyWithdrawalMinor > current.MonthlyWithdrawalMinor {
		WriteAPIError(w, http.StatusConflict, "LIMIT_INCREASE_REQUIRES_REVIEW", "Limit increases require a compliance review.")
		return
	}
	current.DailyDepositMinor = request.DailyDepositMinor
	current.MonthlyDepositMinor = request.MonthlyDepositMinor
	current.DailyWithdrawalMinor = request.DailyWithdrawalMinor
	current.MonthlyWithdrawalMinor = request.MonthlyWithdrawalMinor
	now := time.Now().UTC()
	if request.CoolingOffDays > 0 {
		until := now.AddDate(0, 0, request.CoolingOffDays)
		current.CoolingOffUntil = &until
	}
	if request.SelfExcludeDays > 0 {
		until := now.AddDate(0, 0, request.SelfExcludeDays)
		current.SelfExcludedUntil = &until
	}
	item, err := h.store.SaveFinancialLimits(r.Context(), *current)
	if err != nil {
		WriteMappedError(w, http.StatusBadRequest, err)
		return
	}
	_ = h.notify(r, userID, "responsible_gaming", "Financial limits updated", "Your responsible gaming limits are now active.", "/wallet", nil)
	writeJSON(w, http.StatusOK, item)
}

func (h *FinancialHandlers) Deposit(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		items, err := h.store.ListFinancialDeposits(r.Context(), UserIDFromContext(r.Context()))
		if err != nil {
			WriteMappedError(w, http.StatusInternalServerError, err)
			return
		}
		writeJSON(w, http.StatusOK, items)
		return
	}
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	var request financialIntentRequest
	if err := decodeJSON(r, &request); err != nil {
		WriteAPIError(w, http.StatusBadRequest, "INVALID_DEPOSIT", err.Error())
		return
	}
	idempotencyKey := idempotencyKeyFromRequest(r)
	if !validIdempotencyKey(idempotencyKey) {
		WriteAPIError(w, http.StatusBadRequest, "IDEMPOTENCY_KEY_REQUIRED", "A 16-128 character Idempotency-Key is required.")
		return
	}
	userID := UserIDFromContext(r.Context())
	if err := h.validateFinancialIntent(r, userID, request, false); err != nil {
		WriteMappedError(w, http.StatusConflict, err)
		return
	}
	country, err := h.financialCountry(r, userID)
	if err != nil {
		WriteMappedError(w, http.StatusInternalServerError, err)
		return
	}
	selection := payments.SelectionRequest{
		Country: country, Currency: request.Currency, Method: request.Method,
		Preferred: h.settings.Payments.DefaultProvider, AmountMinor: request.AmountMinor,
	}
	providerID, err := h.payments.SelectDeposit(r.Context(), selection)
	if err != nil {
		WriteAPIError(w, http.StatusServiceUnavailable, "PAYMENT_METHOD_UNAVAILABLE", "This payment method is not currently available.")
		return
	}
	requestHash := financialIntentHash("deposit", userID, request)
	deposit, replayed, err := h.store.CreateFinancialDeposit(r.Context(), &models.FinancialDeposit{
		UserID: userID, AmountMinor: request.AmountMinor, Currency: request.Currency,
		Method: request.Method, Provider: providerID, IdempotencyKey: idempotencyKey, RequestHash: requestHash,
	})
	if err != nil {
		WriteMappedError(w, http.StatusConflict, err)
		return
	}
	// Idempotent retries remain pinned to the adapter selected by the original
	// request. Switching after an ambiguous provider attempt could duplicate a
	// financial operation.
	providerID = deposit.Provider
	if replayed && deposit.Status != models.DepositStatusRequested {
		w.Header().Set("Idempotent-Replayed", "true")
		writeJSON(w, http.StatusOK, deposit)
		return
	}
	// The resource ID and callback URL are assigned after the idempotent internal
	// intent exists. Adapters receive no wallet or ledger state.
	response, err := h.payments.CreateDepositSessionForProvider(r.Context(), providerID, payments.DepositSessionRequest{
		SessionID: deposit.ID, UserID: userID, AmountMinor: deposit.AmountMinor, Currency: deposit.Currency,
		Country: country, Method: request.Method,
		ReturnURL:      safeReturnURL(request.ReturnURL, h.settings.Email.BaseURL+"/wallet"),
		CancelURL:      safeReturnURL(request.CancelURL, h.settings.Email.BaseURL+"/wallet"),
		NotifyURL:      strings.TrimRight(h.settings.Email.BaseURL, "/") + "/api/v1/payments/webhooks/" + providerID,
		IdempotencyKey: idempotencyKey,
	})
	if err != nil {
		WriteAPIError(w, http.StatusBadGateway, "PROVIDER_SESSION_FAILED", "The payment provider could not start a secure session.")
		return
	}
	deposit, err = h.store.SetFinancialDepositProviderSession(r.Context(), deposit.ID, response.ProviderRef, response.CheckoutURL)
	if err != nil {
		WriteMappedError(w, http.StatusInternalServerError, err)
		return
	}
	_ = h.notify(r, userID, "financial", "Deposit pending", "Provider confirmation has not arrived yet.", "/wallet", map[string]string{"depositId": deposit.ID})
	writeJSON(w, http.StatusAccepted, deposit)
}

func (h *FinancialHandlers) Withdrawal(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		items, err := h.store.ListFinancialWithdrawals(r.Context(), UserIDFromContext(r.Context()))
		if err != nil {
			WriteMappedError(w, http.StatusInternalServerError, err)
			return
		}
		writeJSON(w, http.StatusOK, items)
		return
	}
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	var request financialIntentRequest
	if err := decodeJSON(r, &request); err != nil {
		WriteAPIError(w, http.StatusBadRequest, "INVALID_WITHDRAWAL", err.Error())
		return
	}
	idempotencyKey := idempotencyKeyFromRequest(r)
	if !validIdempotencyKey(idempotencyKey) {
		WriteAPIError(w, http.StatusBadRequest, "IDEMPOTENCY_KEY_REQUIRED", "A 16-128 character Idempotency-Key is required.")
		return
	}
	userID := UserIDFromContext(r.Context())
	if err := h.validateFinancialIntent(r, userID, request, true); err != nil {
		WriteMappedError(w, http.StatusConflict, err)
		return
	}
	country, err := h.financialCountry(r, userID)
	if err != nil {
		WriteMappedError(w, http.StatusInternalServerError, err)
		return
	}
	providerID, err := h.payments.SelectPayout(r.Context(), payments.SelectionRequest{
		Country: country, Currency: request.Currency, Method: request.Method,
		Preferred: h.settings.Payments.DefaultProvider, AmountMinor: request.AmountMinor,
	})
	if err != nil {
		WriteAPIError(w, http.StatusServiceUnavailable, "PAYMENT_METHOD_UNAVAILABLE", "This withdrawal method is not currently available.")
		return
	}
	policyReasons, err := h.withdrawalPolicyReasons(r, userID, request)
	if err != nil {
		WriteMappedError(w, http.StatusInternalServerError, err)
		return
	}
	item, replayed, err := h.store.CreateFinancialWithdrawal(r.Context(), &models.FinancialWithdrawal{
		UserID: userID, AmountMinor: request.AmountMinor, Currency: request.Currency,
		Method: request.Method, Provider: providerID, IdempotencyKey: idempotencyKey,
		RequestHash: financialIntentHash("withdrawal", userID, request), PolicyDecision: "manual_review",
		PolicyReasons: policyReasons,
	})
	if err != nil {
		WriteMappedError(w, http.StatusConflict, err)
		return
	}
	if replayed {
		w.Header().Set("Idempotent-Replayed", "true")
		writeJSON(w, http.StatusOK, item)
		return
	}
	_ = h.notify(r, userID, "financial", "Withdrawal submitted", "Your withdrawal is pending review.", "/wallet", map[string]string{"withdrawalId": item.ID})
	writeJSON(w, http.StatusAccepted, item)
}

func (h *FinancialHandlers) Webhook(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	providerName := strings.TrimPrefix(r.URL.Path, "/api/v1/payments/webhooks/")
	body, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	if err != nil {
		WriteAPIError(w, http.StatusBadRequest, "INVALID_WEBHOOK", "Webhook payload could not be read.")
		return
	}
	event, verification, err := h.payments.VerifyCallback(r.Context(), providerName, payments.CallbackRequest{
		Headers: r.Header.Clone(), Body: body,
	})
	if err != nil || !verification.Valid {
		WriteAPIError(w, http.StatusUnauthorized, "INVALID_WEBHOOK_SIGNATURE", "Webhook signature verification failed.")
		return
	}
	eventID := event.EventID
	resourceType, resourceID := "deposit", event.ResourceID
	if event.Kind == payments.EventPayout {
		resourceType = "withdrawal"
	}
	signatureHash := verification.Fingerprint
	payloadHash := sha256Hex(string(body))
	first, err := h.store.RecordFinancialWebhook(r.Context(), providerName, eventID, signatureHash, payloadHash, resourceType, resourceID)
	if err != nil {
		WriteMappedError(w, http.StatusInternalServerError, err)
		return
	}
	if !first {
		writeJSON(w, http.StatusOK, map[string]string{"status": "duplicate_ignored"})
		return
	}
	outcome := "ignored"
	if resourceType == "deposit" && event.Status != payments.StatusUnknown {
		deposit, loadErr := h.store.GetFinancialDeposit(r.Context(), resourceID)
		if loadErr != nil || deposit.Provider != providerName || deposit.ProviderReference != event.ProviderRef ||
			deposit.AmountMinor != event.AmountMinor || deposit.Currency != strings.ToUpper(event.Currency) {
			_ = h.store.CompleteFinancialWebhook(r.Context(), providerName, eventID, "mismatch")
			WriteAPIError(w, http.StatusConflict, "WEBHOOK_MISMATCH", "Webhook does not match the recorded deposit.")
			return
		}
		switch event.Status {
		case payments.StatusFailed:
			err = h.store.FailFinancialDeposit(r.Context(), resourceID, "Signed provider failure")
			outcome = "deposit_failed"
		case payments.StatusExpired:
			err = h.store.ExpireFinancialDeposit(r.Context(), resourceID, "Signed provider expiry")
			outcome = "deposit_expired"
		case payments.StatusPending:
			deposit, err = h.store.AdvanceFinancialDeposit(r.Context(), resourceID, models.DepositStatusPendingProvider, eventID)
			outcome = "deposit_pending"
		case payments.StatusSucceeded:
			deposit, err = h.store.AdvanceFinancialDeposit(r.Context(), resourceID, models.DepositStatusPendingVerification, eventID)
			if err == nil {
				err = h.store.WithFinancialSettlementLock(r.Context(), deposit.Currency, func() error {
					if reserveErr := h.verifyProviderReserve(r, providerName, deposit.Currency, deposit.AmountMinor, "deposit_settlement"); reserveErr != nil {
						return reserveErr
					}
					var settlementErr error
					deposit, settlementErr = h.store.SettleFinancialDeposit(r.Context(), resourceID, eventID)
					return settlementErr
				})
			}
			if err == nil {
				outcome = "deposit_completed"
				_ = h.notify(r, deposit.UserID, "financial", "Deposit completed", "Deposit settled. Your live balance is now available for competition.", "/wallet", map[string]string{"depositId": deposit.ID})
			}
		}
	} else if resourceType == "withdrawal" && (event.Status == payments.StatusSucceeded || event.Status == payments.StatusFailed) {
		withdrawal, loadErr := h.store.GetFinancialWithdrawal(r.Context(), resourceID)
		if loadErr != nil || withdrawal.Provider != providerName || withdrawal.ProviderReference != event.ProviderRef ||
			withdrawal.AmountMinor != event.AmountMinor || withdrawal.Currency != strings.ToUpper(event.Currency) {
			_ = h.store.CompleteFinancialWebhook(r.Context(), providerName, eventID, "mismatch")
			WriteAPIError(w, http.StatusConflict, "WEBHOOK_MISMATCH", "Webhook does not match the recorded withdrawal.")
			return
		}
		target := models.FinancialWithdrawalStatusCompleted
		if event.Status == payments.StatusFailed {
			target = models.FinancialWithdrawalStatusFailed
		}
		withdrawal, err = h.store.TransitionFinancialWithdrawal(r.Context(), resourceID, target, "provider", providerName, "Signed provider callback", event.ProviderRef)
		if err == nil {
			outcome = "withdrawal_" + target
			_ = h.notify(r, withdrawal.UserID, "financial", "Withdrawal "+target, withdrawalMessage(target), "/wallet", map[string]string{"withdrawalId": withdrawal.ID})
		}
	}
	if err != nil {
		_ = h.store.CompleteFinancialWebhook(r.Context(), providerName, eventID, "failed")
		WriteMappedError(w, http.StatusConflict, err)
		return
	}
	_ = h.store.CompleteFinancialWebhook(r.Context(), providerName, eventID, outcome)
	writeJSON(w, http.StatusOK, map[string]string{"status": outcome})
}

type financialAssessmentDecisionRequest struct {
	UserID       string `json:"userId"`
	Decision     string `json:"decision"`
	Risk         string `json:"riskClassification"`
	Verification string `json:"verificationStatus"`
}

func (h *FinancialHandlers) AdminAssessmentDecision(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	var request financialAssessmentDecisionRequest
	if err := decodeJSON(r, &request); err != nil || request.UserID == "" {
		WriteAPIError(w, http.StatusBadRequest, "INVALID_DECISION", "User and assessment decision are required.")
		return
	}
	item, err := h.store.ReviewFinancialAssessment(r.Context(), request.UserID, request.Decision, request.Risk, request.Verification)
	if err != nil {
		WriteMappedError(w, http.StatusConflict, err)
		return
	}
	_ = h.notify(r, request.UserID, "financial", "Financial assessment updated", "Your financial eligibility status has changed.", "/wallet", map[string]string{"assessmentStatus": item.Status})
	_ = h.store.AppendAuditLog(r.Context(), UserIDFromContext(r.Context()), "financial.assessment.decision", request.UserID, clientIP(r), map[string]string{"decision": request.Decision})
	writeJSON(w, http.StatusOK, item)
}

type financialWithdrawalTransitionRequest struct {
	WithdrawalID string `json:"withdrawalId"`
	Status       string `json:"status"`
	Reason       string `json:"reason"`
}

type payoutDestinationRequest struct {
	UserID            string `json:"userId"`
	Provider          string `json:"provider"`
	ProviderAccountID string `json:"providerAccountId"`
	Status            string `json:"status"`
	EvidenceID        string `json:"evidenceId,omitempty"`
}

func (h *FinancialHandlers) AdminPayoutDestination(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	var request payoutDestinationRequest
	if err := decodeJSON(r, &request); err != nil || request.UserID == "" {
		WriteAPIError(w, http.StatusBadRequest, "INVALID_PAYOUT_DESTINATION", "A valid verified provider destination is required.")
		return
	}
	if err := h.payments.ValidatePayoutDestination(request.Provider, request.ProviderAccountID); err != nil {
		WriteAPIError(w, http.StatusBadRequest, "INVALID_PAYOUT_DESTINATION", "The payout provider is not supported.")
		return
	}
	item, err := h.store.SaveFinancialPayoutDestination(r.Context(), models.FinancialPayoutDestination{
		UserID: request.UserID, Provider: request.Provider, ProviderAccountID: request.ProviderAccountID,
		Status: request.Status, EvidenceID: request.EvidenceID,
	})
	if err != nil {
		WriteMappedError(w, http.StatusConflict, err)
		return
	}
	_ = h.store.AppendAuditLog(r.Context(), UserIDFromContext(r.Context()), "financial.payout_destination.updated", request.UserID, clientIP(r), map[string]string{"provider": item.Provider, "status": item.Status})
	writeJSON(w, http.StatusOK, item)
}

func (h *FinancialHandlers) AdminWithdrawalTransition(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	var request financialWithdrawalTransitionRequest
	if err := decodeJSON(r, &request); err != nil || request.WithdrawalID == "" {
		WriteAPIError(w, http.StatusBadRequest, "INVALID_TRANSITION", "Withdrawal and target status are required.")
		return
	}
	item, err := h.store.GetFinancialWithdrawal(r.Context(), request.WithdrawalID)
	if err != nil {
		WriteMappedError(w, http.StatusNotFound, err)
		return
	}
	providerRef := ""
	transitioned := false
	if request.Status == models.FinancialWithdrawalStatusProcessing {
		if item.Status != models.FinancialWithdrawalStatusApproved {
			WriteAPIError(w, http.StatusConflict, "WITHDRAWAL_NOT_APPROVED", "A withdrawal must be approved before provider processing.")
			return
		}
		destination, destinationErr := h.store.GetFinancialPayoutDestination(r.Context(), item.UserID, item.Provider)
		if destinationErr != nil || destination.Status != "verified" {
			WriteAPIError(w, http.StatusConflict, "PAYOUT_DESTINATION_REQUIRED", "A verified payout destination is required.")
			return
		}
		providerErr := h.store.WithFinancialSettlementLock(r.Context(), item.Currency, func() error {
			if reserveErr := h.verifyProviderReserve(r, item.Provider, item.Currency, item.AmountMinor+item.FeeMinor, "withdrawal_processing"); reserveErr != nil {
				return reserveErr
			}
			response, createErr := h.payments.CreatePayout(r.Context(), item.Provider, payments.PayoutRequest{
				WithdrawalID: item.ID, UserID: item.UserID, AmountMinor: item.AmountMinor,
				Currency: item.Currency, IdempotencyKey: item.IdempotencyKey,
				Method: item.Method, Destination: destination.ProviderAccountID,
			})
			if createErr != nil {
				return createErr
			}
			providerRef = response.ProviderRef
			var transitionErr error
			item, transitionErr = h.store.TransitionFinancialWithdrawal(r.Context(), item.ID, request.Status, "treasury", UserIDFromContext(r.Context()), request.Reason, providerRef)
			if transitionErr == nil {
				transitioned = true
			}
			return transitionErr
		})
		if providerErr != nil {
			if errors.Is(providerErr, errTreasuryReserveInsufficient) {
				WriteAPIError(w, http.StatusConflict, "TREASURY_RESERVE_INSUFFICIENT", "Treasury reserves do not permit this withdrawal.")
				return
			}
			WriteAPIError(w, http.StatusBadGateway, "PROVIDER_WITHDRAWAL_FAILED", "The provider could not start withdrawal processing.")
			return
		}
	}
	if !transitioned {
		item, err = h.store.TransitionFinancialWithdrawal(r.Context(), item.ID, request.Status, "treasury", UserIDFromContext(r.Context()), request.Reason, providerRef)
		if err != nil {
			WriteMappedError(w, http.StatusConflict, err)
			return
		}
	}
	_ = h.notify(r, item.UserID, "financial", "Withdrawal "+item.Status, withdrawalMessage(item.Status), "/wallet", map[string]string{"withdrawalId": item.ID})
	_ = h.store.AppendAuditLog(r.Context(), UserIDFromContext(r.Context()), "financial.withdrawal.transition", item.ID, clientIP(r), map[string]string{"status": item.Status, "reason": request.Reason})
	writeJSON(w, http.StatusOK, item)
}

type reconciliationRequest struct {
	Provider string `json:"provider"`
	Currency string `json:"currency"`
}

func (h *FinancialHandlers) AdminReconcile(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	var request reconciliationRequest
	if err := decodeJSON(r, &request); err != nil || request.Provider == "" {
		WriteAPIError(w, http.StatusBadRequest, "INVALID_RECONCILIATION", "Provider, currency, and balance are required.")
		return
	}
	reconciliation, err := h.payments.Reconcile(r.Context(), request.Provider, payments.ReconciliationRequest{
		Currency: request.Currency, From: time.Now().UTC().AddDate(0, 0, -1), To: time.Now().UTC(),
	})
	if err != nil {
		WriteAPIError(w, http.StatusServiceUnavailable, "PROVIDER_BALANCE_UNAVAILABLE", "The provider balance could not be verified.")
		return
	}
	liability, err := h.store.FinancialLiability(r.Context(), request.Currency)
	if err != nil {
		WriteMappedError(w, http.StatusInternalServerError, err)
		return
	}
	balance := reconciliation.Balance
	check, err := h.store.RecordTreasuryReserveCheck(r.Context(), request.Provider, request.Currency, balance.AvailableMinor, balance.PendingMinor, liability, 0, "reconciliation")
	if err != nil {
		WriteMappedError(w, http.StatusInternalServerError, err)
		return
	}
	item, err := h.store.ReconcileFinancialTreasury(r.Context(), request.Provider, request.Currency, balance.AvailableMinor+balance.PendingMinor)
	if err != nil {
		WriteMappedError(w, http.StatusInternalServerError, err)
		return
	}
	_ = h.store.AppendAuditLog(r.Context(), UserIDFromContext(r.Context()), "financial.treasury.reconciled", item.ID, clientIP(r), map[string]string{"status": item.Status, "reserveCheck": check.ID})
	writeJSON(w, http.StatusOK, map[string]any{"reconciliation": item, "reserveCheck": check})
}

func (h *FinancialHandlers) ProviderHealth(ctx context.Context) error {
	if h == nil || h.payments == nil {
		return nil
	}
	return h.payments.Health(ctx)
}

func (h *FinancialHandlers) verifyProviderReserve(r *http.Request, providerID, currency string, requested models.MinorUnits, purpose string) error {
	balance, err := h.payments.Balance(r.Context(), providerID, currency)
	if err != nil {
		return err
	}
	liability, err := h.store.FinancialLiability(r.Context(), currency)
	if err != nil {
		return err
	}
	check, err := h.store.RecordTreasuryReserveCheck(r.Context(), providerID, currency, balance.AvailableMinor, balance.PendingMinor, liability, requested, purpose)
	if err != nil {
		return err
	}
	if !check.Passed {
		return errTreasuryReserveInsufficient
	}
	return nil
}

func (h *FinancialHandlers) financialCountry(r *http.Request, userID string) (string, error) {
	assessment, err := h.store.GetFinancialAssessment(r.Context(), userID)
	if err != nil {
		return "", err
	}
	if country := strings.ToUpper(strings.TrimSpace(assessment.Country)); country != "" {
		return country, nil
	}
	user, err := h.store.GetUserByID(r.Context(), userID)
	if err != nil {
		return "", err
	}
	if country := strings.ToUpper(strings.TrimSpace(user.Country)); country != "" {
		return country, nil
	}
	return h.settings.Financial.DefaultCountry, nil
}

func (h *FinancialHandlers) validateFinancialIntent(r *http.Request, userID string, request financialIntentRequest, withdrawal bool) error {
	if request.AmountMinor <= 0 || request.AmountMinor > 100_000_000_00 {
		return errors.New("amountMinor must be positive and within platform bounds")
	}
	if request.Method != "card" && request.Method != "eft" && request.Method != "bank_transfer" {
		return errors.New("payment method is invalid")
	}
	user, err := h.store.GetUserByID(r.Context(), userID)
	if err != nil {
		return err
	}
	if !user.EmailVerified {
		return errors.New("email verification is required")
	}
	assessment, err := h.store.GetFinancialAssessment(r.Context(), userID)
	if err != nil {
		return err
	}
	if assessment.Status != models.AssessmentStatusComplete || assessment.ResponsibleStatus != "active" {
		return errors.New("financial assessment and responsible gaming checks must be complete")
	}
	policy, supported := h.settings.Financial.Policy(assessment.Country)
	if !supported || strings.ToUpper(request.Currency) != policy.Currency || !containsString(policy.PaymentMethods, request.Method) {
		return errors.New("currency or payment method is not enabled for this jurisdiction")
	}
	limits, err := h.store.GetFinancialLimits(r.Context(), userID)
	if err != nil {
		return err
	}
	now := time.Now().UTC()
	if (limits.CoolingOffUntil != nil && now.Before(*limits.CoolingOffUntil)) ||
		(limits.SelfExcludedUntil != nil && now.Before(*limits.SelfExcludedUntil)) {
		return errors.New("financial activity is unavailable during cooling-off or self-exclusion")
	}
	if withdrawal {
		if request.AmountMinor+limits.WithdrawUsedTodayMinor > limits.DailyWithdrawalMinor ||
			request.AmountMinor+limits.WithdrawUsedMonthMinor > limits.MonthlyWithdrawalMinor {
			return errors.New("withdrawal limit reached")
		}
		return nil
	}
	if request.AmountMinor+limits.DepositUsedTodayMinor > limits.DailyDepositMinor ||
		request.AmountMinor+limits.DepositUsedMonthMinor > limits.MonthlyDepositMinor {
		return errors.New("deposit limit reached")
	}
	return nil
}

func (h *FinancialHandlers) withdrawalPolicyReasons(r *http.Request, userID string, request financialIntentRequest) ([]string, error) {
	assessment, err := h.store.GetFinancialAssessment(r.Context(), userID)
	if err != nil {
		return nil, err
	}
	limits, err := h.store.GetFinancialLimits(r.Context(), userID)
	if err != nil {
		return nil, err
	}
	evidence, err := h.store.ListFinancialEvidence(r.Context(), userID)
	if err != nil {
		return nil, err
	}
	reasons := []string{"Manual review is required during the initial financial policy phase."}
	if assessment.RiskClassification != "standard" {
		reasons = append(reasons, "Financial risk classification requires review.")
	}
	if assessment.VerificationStatus != "verified" {
		reasons = append(reasons, "Identity verification is incomplete.")
	}
	if request.AmountMinor >= limits.DailyWithdrawalMinor/2 {
		reasons = append(reasons, "Withdrawal is high relative to the daily limit.")
	}
	if request.AmountMinor+limits.WithdrawUsedTodayMinor >= limits.DailyWithdrawalMinor*3/4 {
		reasons = append(reasons, "Daily withdrawal velocity requires review.")
	}
	if !hasFinancialEvidence(evidence, "identity") {
		reasons = append(reasons, "Identity evidence is not on file.")
	}
	if !hasFinancialEvidence(evidence, "source_of_funds") {
		reasons = append(reasons, "Source-of-funds evidence is not on file.")
	}
	return reasons, nil
}

func hasFinancialEvidence(items []models.FinancialEvidence, evidenceType string) bool {
	for _, item := range items {
		if item.Type == evidenceType && item.Status != "rejected" && item.Status != "expired" {
			return true
		}
	}
	return false
}

func (h *FinancialHandlers) notify(r *http.Request, userID, category, title, message, actionURL string, metadata map[string]string) error {
	return h.store.CreateNotification(r.Context(), &models.Notification{
		UserID: userID, Category: category, Title: title, Message: message,
		ActionURL: actionURL, Metadata: metadata,
	})
}

func validIdempotencyKey(value string) bool {
	value = strings.TrimSpace(value)
	return len(value) >= 16 && len(value) <= 128
}

func financialIntentHash(operation, userID string, request financialIntentRequest) string {
	raw := operation + "|" + userID + "|" + strconv.FormatInt(int64(request.AmountMinor), 10) + "|" +
		strings.ToUpper(request.Currency) + "|" + request.Method
	return sha256Hex(raw)
}

func safeReturnURL(value, fallback string) string {
	if strings.HasPrefix(value, "https://") || strings.HasPrefix(value, "http://localhost:") {
		return value
	}
	return fallback
}

func validOccupation(value string) bool {
	switch value {
	case "employed", "self_employed", "student", "retired", "unemployed", "other":
		return true
	}
	return false
}

func containsString(items []string, value string) bool {
	for _, item := range items {
		if item == value {
			return true
		}
	}
	return false
}

func filterPaymentMethods(methods []models.PaymentMethod, allowed []string) []models.PaymentMethod {
	result := make([]models.PaymentMethod, 0, len(methods))
	for _, method := range methods {
		if containsString(allowed, method.Type) {
			result = append(result, method)
		}
	}
	return result
}

func minMinor(current, policy models.MinorUnits) models.MinorUnits {
	if current == 0 || policy < current {
		return policy
	}
	return current
}

func ageAt(birth, now time.Time) int {
	age := now.Year() - birth.Year()
	birthday := time.Date(now.Year(), birth.Month(), birth.Day(), 0, 0, 0, 0, time.UTC)
	if now.Before(birthday) {
		age--
	}
	return age
}

func validSourceOfFunds(value string) bool {
	switch value {
	case "salary", "business", "savings", "investments", "pension", "other":
		return true
	}
	return false
}

func withdrawalMessage(status string) string {
	switch status {
	case models.FinancialWithdrawalStatusApproved:
		return "Your withdrawal has been approved and will move to provider processing."
	case models.FinancialWithdrawalStatusProcessing:
		return "Your withdrawal is processing with the payment provider."
	case models.FinancialWithdrawalStatusCompleted:
		return "Your withdrawal has settled and the ledger is complete."
	case models.FinancialWithdrawalStatusRejected:
		return "Your withdrawal was rejected. Review the status details before trying again."
	case models.FinancialWithdrawalStatusFailed:
		return "Provider settlement failed. Reserved funds have returned to your available balance."
	default:
		return "Your withdrawal status has changed."
	}
}
