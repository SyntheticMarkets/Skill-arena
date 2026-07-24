package server

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"skill-arena/internal/db"
	"skill-arena/internal/models"
	"skill-arena/internal/payments"
)

func TestFinancialAPIIdempotencyWebhookAndNotifications(t *testing.T) {
	store, err := db.New(context.Background(), t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	providerAPI := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v1/checkout/sessions":
			_, _ = w.Write([]byte(`{"id":"cs_test_api","url":"https://checkout.stripe.test/c/pay/cs_test_api"}`))
		case "/v1/balance":
			_, _ = w.Write([]byte(`{"available":[{"amount":100000,"currency":"zar"}],"pending":[]}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer providerAPI.Close()
	cfg := authTestConfig("")
	cfg.Settings.Payments.DefaultProvider = payments.ProviderStripe
	cfg.Settings.Payments.StripeSecretKey = "sk_test_financial_api"
	webhookKey := strings.Repeat("f", 32)
	cfg.Settings.Payments.StripeWebhookSecret = webhookKey
	cfg.Settings.Payments.StripeAPIBase = providerAPI.URL
	handler := New(store, cfg).Handler
	access, _ := registerVerifyLogin(
		t, handler, store,
		[2]string{cfg.Settings.Security.AccessCookieName, cfg.Settings.Security.RefreshCookieName},
		"financial-api@example.com", "StrongPassword!42",
	)
	user, err := store.GetUserByEmail(context.Background(), "financial-api@example.com")
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

	overview := authRequest(t, handler, http.MethodGet, "/api/v1/financial/overview", nil, []*http.Cookie{access})
	if overview.Code != http.StatusOK || !strings.Contains(overview.Body.String(), `"availableMinor":0`) ||
		!strings.Contains(overview.Body.String(), `"available":true`) {
		t.Fatalf("overview status=%d body=%s", overview.Code, overview.Body.String())
	}

	depositBody := map[string]any{"amountMinor": 2_500, "currency": "ZAR", "method": "card"}
	deposit := financialRequest(t, handler, http.MethodPost, "/api/v1/financial/deposits", depositBody, []*http.Cookie{access}, "deposit-api-idempotency-0001", nil)
	if deposit.Code != http.StatusAccepted {
		t.Fatalf("deposit status=%d body=%s", deposit.Code, deposit.Body.String())
	}
	var created models.FinancialDeposit
	if err := json.Unmarshal(deposit.Body.Bytes(), &created); err != nil {
		t.Fatal(err)
	}
	replay := financialRequest(t, handler, http.MethodPost, "/api/v1/financial/deposits", depositBody, []*http.Cookie{access}, "deposit-api-idempotency-0001", nil)
	if replay.Code != http.StatusOK || replay.Header().Get("Idempotent-Replayed") != "true" {
		t.Fatalf("replay status=%d headers=%v body=%s", replay.Code, replay.Header(), replay.Body.String())
	}
	conflict := financialRequest(t, handler, http.MethodPost, "/api/v1/financial/deposits",
		map[string]any{"amountMinor": 2_501, "currency": "ZAR", "method": "card"},
		[]*http.Cookie{access}, "deposit-api-idempotency-0001", nil)
	if conflict.Code != http.StatusConflict {
		t.Fatalf("idempotency conflict status=%d body=%s", conflict.Code, conflict.Body.String())
	}

	webhookBody := []byte(fmt.Sprintf(
		`{"id":"evt_financial_api","type":"checkout.session.completed","data":{"object":{"id":"cs_test_api","amount_total":2500,"currency":"zar","payment_status":"paid","metadata":{"deposit_id":%q}}}}`,
		created.ID,
	))
	webhook := stripeWebhookRequest(t, handler, webhookBody, webhookKey)
	if webhook.Code != http.StatusOK || !strings.Contains(webhook.Body.String(), "deposit_completed") {
		t.Fatalf("webhook status=%d body=%s", webhook.Code, webhook.Body.String())
	}
	duplicate := stripeWebhookRequest(t, handler, webhookBody, webhookKey)
	if duplicate.Code != http.StatusOK || !strings.Contains(duplicate.Body.String(), "duplicate_ignored") {
		t.Fatalf("duplicate status=%d body=%s", duplicate.Code, duplicate.Body.String())
	}
	wallet, err := store.GetFinancialWallet(context.Background(), user.ID)
	if err != nil || wallet.AvailableMinor != 2_500 {
		t.Fatalf("wallet=%+v err=%v", wallet, err)
	}
	notifications, err := store.ListNotifications(context.Background(), user.ID, "")
	if err != nil || len(notifications) < 2 {
		t.Fatalf("notifications=%+v err=%v", notifications, err)
	}
}

func financialRequest(t *testing.T, handler http.Handler, method, path string, body any, cookies []*http.Cookie, idempotency string, headers map[string]string) *httptest.ResponseRecorder {
	t.Helper()
	payload, err := json.Marshal(body)
	if raw, ok := body.(json.RawMessage); ok {
		payload = raw
		err = nil
	}
	if err != nil {
		t.Fatal(err)
	}
	request := httptest.NewRequest(method, path, bytes.NewReader(payload))
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Origin", "https://app.skillarena.test")
	request.Header.Set("X-Device-Fingerprint", "financial-test-device")
	if idempotency != "" {
		request.Header.Set("Idempotency-Key", idempotency)
	}
	for name, value := range headers {
		request.Header.Set(name, value)
	}
	for _, cookie := range cookies {
		request.AddCookie(cookie)
	}
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	return response
}
