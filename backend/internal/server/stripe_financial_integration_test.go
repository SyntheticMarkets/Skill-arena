package server

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"skill-arena/internal/db"
	"skill-arena/internal/handlers"
	"skill-arena/internal/models"
	"skill-arena/internal/payments"
)

func TestStripeFinancialLifecycleRetriesSettlesAndWithdraws(t *testing.T) {
	var checkoutRequests atomic.Int32
	providerAPI := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		username, _, authenticated := r.BasicAuth()
		if !authenticated || username != "sk_test_financial_integration" {
			http.Error(w, `{"error":{"code":"invalid_api_key","message":"invalid test key"}}`, http.StatusUnauthorized)
			return
		}
		switch r.URL.Path {
		case "/v1/checkout/sessions":
			attempt := checkoutRequests.Add(1)
			if r.Header.Get("Idempotency-Key") != "stripe-deposit-retry-0001" {
				t.Errorf("checkout idempotency key=%q", r.Header.Get("Idempotency-Key"))
			}
			if attempt == 1 {
				http.Error(w, `{"error":{"code":"api_connection_error","message":"temporary provider outage"}}`, http.StatusServiceUnavailable)
				return
			}
			_, _ = w.Write([]byte(`{"id":"cs_test_financial","url":"https://checkout.stripe.test/c/pay/cs_test_financial"}`))
		case "/v1/balance":
			_, _ = w.Write([]byte(`{"available":[{"amount":100000,"currency":"zar"}],"pending":[]}`))
		case "/v1/transfers":
			if r.Header.Get("Idempotency-Key") != "stripe-withdrawal-0001-transfer" {
				t.Errorf("withdrawal idempotency key=%q", r.Header.Get("Idempotency-Key"))
			}
			_ = r.ParseForm()
			if r.Form.Get("destination") != "acct_test_financial_player" || r.Form.Get("amount") != "1000" {
				t.Errorf("transfer form=%v", r.Form)
			}
			_, _ = w.Write([]byte(`{"id":"tr_test_financial","destination":"acct_test_financial_player"}`))
		case "/v1/payouts":
			if r.Header.Get("Idempotency-Key") != "stripe-withdrawal-0001-payout" ||
				r.Header.Get("Stripe-Account") != "acct_test_financial_player" {
				t.Errorf("payout headers=%v", r.Header)
			}
			_, _ = w.Write([]byte(`{"id":"po_test_financial","status":"pending"}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer providerAPI.Close()

	store, err := db.New(context.Background(), t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	cfg := authTestConfig("")
	cfg.Settings.Payments.DefaultProvider = payments.ProviderStripe
	cfg.Settings.Payments.StripeSecretKey = "sk_test_financial_integration"
	cfg.Settings.Payments.StripeWebhookSecret = strings.Repeat("s", 32)
	cfg.Settings.Payments.StripeAPIBase = providerAPI.URL
	cfg.Settings.Payments.StripeMode = "sandbox"
	handler := New(store, cfg).Handler
	access, _ := registerVerifyLogin(
		t, handler, store,
		[2]string{cfg.Settings.Security.AccessCookieName, cfg.Settings.Security.RefreshCookieName},
		"stripe-financial@example.com", "StrongPassword!42",
	)
	user, err := store.GetUserByEmail(context.Background(), "stripe-financial@example.com")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.SaveFinancialAssessment(context.Background(), models.FinancialAssessment{
		UserID: user.ID, Country: "ZA", Occupation: "employed", SourceOfFunds: "salary",
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := store.ReviewFinancialAssessment(context.Background(), user.ID, models.AssessmentStatusComplete, "standard", "verified"); err != nil {
		t.Fatal(err)
	}

	depositBody := map[string]any{"amountMinor": 2_500, "currency": "ZAR", "method": "card"}
	firstAttempt := financialRequest(t, handler, http.MethodPost, "/api/v1/financial/deposits", depositBody, []*http.Cookie{access}, "stripe-deposit-retry-0001", nil)
	if firstAttempt.Code != http.StatusBadGateway {
		t.Fatalf("first checkout status=%d body=%s", firstAttempt.Code, firstAttempt.Body.String())
	}
	retry := financialRequest(t, handler, http.MethodPost, "/api/v1/financial/deposits", depositBody, []*http.Cookie{access}, "stripe-deposit-retry-0001", nil)
	if retry.Code != http.StatusAccepted {
		t.Fatalf("retried checkout status=%d body=%s", retry.Code, retry.Body.String())
	}
	var deposit models.FinancialDeposit
	if err := json.Unmarshal(retry.Body.Bytes(), &deposit); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(retry.Body.String(), `"provider"`) || strings.Contains(retry.Body.String(), `"providerReference"`) {
		t.Fatalf("player response exposed provider routing: %s", retry.Body.String())
	}
	recordedDeposit, err := store.GetFinancialDeposit(context.Background(), deposit.ID)
	if err != nil || checkoutRequests.Load() != 2 || recordedDeposit.Provider != payments.ProviderStripe ||
		recordedDeposit.ProviderReference != "cs_test_financial" {
		t.Fatalf("checkout requests=%d deposit=%+v err=%v", checkoutRequests.Load(), recordedDeposit, err)
	}

	depositEvent := []byte(fmt.Sprintf(
		`{"id":"evt_financial_deposit","type":"checkout.session.completed","data":{"object":{"id":"cs_test_financial","amount_total":2500,"currency":"zar","payment_status":"paid","metadata":{"deposit_id":%q}}}}`,
		deposit.ID,
	))
	depositWebhook := stripeWebhookRequest(t, handler, depositEvent, cfg.Settings.Payments.StripeWebhookSecret)
	if depositWebhook.Code != http.StatusOK || !strings.Contains(depositWebhook.Body.String(), "deposit_completed") {
		t.Fatalf("deposit webhook status=%d body=%s", depositWebhook.Code, depositWebhook.Body.String())
	}
	duplicate := stripeWebhookRequest(t, handler, depositEvent, cfg.Settings.Payments.StripeWebhookSecret)
	if duplicate.Code != http.StatusOK || !strings.Contains(duplicate.Body.String(), "duplicate_ignored") {
		t.Fatalf("duplicate webhook status=%d body=%s", duplicate.Code, duplicate.Body.String())
	}

	if _, err := store.SaveFinancialPayoutDestination(context.Background(), models.FinancialPayoutDestination{
		UserID: user.ID, Provider: payments.ProviderStripe, ProviderAccountID: "acct_test_financial_player", Status: "verified",
	}); err != nil {
		t.Fatal(err)
	}
	withdrawalResponse := financialRequest(t, handler, http.MethodPost, "/api/v1/financial/withdrawals",
		map[string]any{"amountMinor": 1_000, "currency": "ZAR", "method": "card"},
		[]*http.Cookie{access}, "stripe-withdrawal-0001", nil)
	if withdrawalResponse.Code != http.StatusAccepted {
		t.Fatalf("withdrawal status=%d body=%s", withdrawalResponse.Code, withdrawalResponse.Body.String())
	}
	if strings.Contains(withdrawalResponse.Body.String(), "policyDecision") || strings.Contains(withdrawalResponse.Body.String(), "PolicyReasons") {
		t.Fatalf("player response exposed internal policy data: %s", withdrawalResponse.Body.String())
	}
	var withdrawal models.FinancialWithdrawal
	if err := json.Unmarshal(withdrawalResponse.Body.Bytes(), &withdrawal); err != nil {
		t.Fatal(err)
	}
	if withdrawal.Status != models.FinancialWithdrawalStatusPendingReview {
		t.Fatalf("withdrawal status=%s", withdrawal.Status)
	}
	if _, err := store.TransitionFinancialWithdrawal(context.Background(), withdrawal.ID, models.FinancialWithdrawalStatusApproved, "treasury", "reviewer", "sandbox approval", ""); err != nil {
		t.Fatal(err)
	}

	financial := handlers.NewFinancialHandlers(store, cfg.Settings)
	processBody, _ := json.Marshal(map[string]string{
		"withdrawalId": withdrawal.ID, "status": models.FinancialWithdrawalStatusProcessing, "reason": "sandbox provider processing",
	})
	processRequest := httptest.NewRequest(http.MethodPost, "/api/v1/admin/financial/withdrawals/transition", bytes.NewReader(processBody))
	processResponse := httptest.NewRecorder()
	financial.AdminWithdrawalTransition(processResponse, processRequest)
	if processResponse.Code != http.StatusOK || !strings.Contains(processResponse.Body.String(), models.FinancialWithdrawalStatusProcessing) {
		t.Fatalf("processing status=%d body=%s", processResponse.Code, processResponse.Body.String())
	}

	withdrawalEvent := []byte(fmt.Sprintf(
		`{"id":"evt_financial_withdrawal","type":"payout.paid","data":{"object":{"id":"po_test_financial","amount":1000,"currency":"zar","status":"paid","metadata":{"withdrawal_id":%q}}}}`,
		withdrawal.ID,
	))
	withdrawalWebhook := stripeWebhookRequest(t, handler, withdrawalEvent, cfg.Settings.Payments.StripeWebhookSecret)
	if withdrawalWebhook.Code != http.StatusOK || !strings.Contains(withdrawalWebhook.Body.String(), "withdrawal_completed") {
		t.Fatalf("withdrawal webhook status=%d body=%s", withdrawalWebhook.Code, withdrawalWebhook.Body.String())
	}

	wallet, err := store.GetFinancialWallet(context.Background(), user.ID)
	if err != nil || wallet.AvailableMinor != 1_500 || wallet.PendingWithdrawalMinor != 0 ||
		wallet.LifetimeDepositMinor != 2_500 || wallet.LifetimeWithdrawMinor != 1_000 {
		t.Fatalf("wallet=%+v err=%v", wallet, err)
	}
	if err := store.VerifyFinancialJournal(context.Background(), user.ID); err != nil {
		t.Fatal(err)
	}
}

func TestReadinessFailsWhenConfiguredProviderIsUnavailable(t *testing.T) {
	providerAPI := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, `{"error":{"code":"api_error","message":"provider unavailable"}}`, http.StatusServiceUnavailable)
	}))
	defer providerAPI.Close()
	store, err := db.New(context.Background(), t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	cfg := authTestConfig("")
	cfg.Settings.Payments.DefaultProvider = payments.ProviderStripe
	cfg.Settings.Payments.StripeSecretKey = "sk_test_readiness"
	cfg.Settings.Payments.StripeWebhookSecret = strings.Repeat("r", 32)
	cfg.Settings.Payments.StripeAPIBase = providerAPI.URL
	response := httptest.NewRecorder()
	New(store, cfg).Handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/health/ready", nil))
	if response.Code != http.StatusServiceUnavailable || !strings.Contains(response.Body.String(), `"payment_provider"`) {
		t.Fatalf("readiness status=%d body=%s", response.Code, response.Body.String())
	}
}

func TestLegacyPlayerAudienceFinancialAdministrationIsRetired(t *testing.T) {
	store, err := db.New(context.Background(), t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	cfg := authTestConfig("")
	handler := New(store, cfg).Handler
	access, _ := registerVerifyLogin(
		t, handler, store,
		[2]string{cfg.Settings.Security.AccessCookieName, cfg.Settings.Security.RefreshCookieName},
		"financial-player@example.com", "StrongPassword!42",
	)
	for _, path := range []string{
		"/api/v1/admin/financial/assessments/decision",
		"/api/v1/admin/financial/withdrawals/transition",
		"/api/v1/admin/financial/payout-destinations",
		"/api/v1/admin/financial/reconcile",
	} {
		response := authRequest(t, handler, http.MethodPost, path, map[string]string{}, []*http.Cookie{access})
		if response.Code != http.StatusNotFound {
			t.Fatalf("player access to %s returned %d body=%s", path, response.Code, response.Body.String())
		}
	}
}

func stripeWebhookRequest(t *testing.T, handler http.Handler, body []byte, secret string) *httptest.ResponseRecorder {
	t.Helper()
	timestamp := time.Now().UTC().Unix()
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = fmt.Fprintf(mac, "%d.", timestamp)
	_, _ = mac.Write(body)
	signature := fmt.Sprintf("t=%d,v1=%s", timestamp, hex.EncodeToString(mac.Sum(nil)))
	return financialRequest(t, handler, http.MethodPost, "/api/v1/payments/webhooks/stripe", json.RawMessage(body), nil, "", map[string]string{
		"Stripe-Signature": signature,
	})
}
