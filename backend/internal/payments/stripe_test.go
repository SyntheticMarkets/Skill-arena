package payments

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func TestStripeProviderSandboxLifecycle(t *testing.T) {
	webhookSecret := strings.Repeat("w", 32)
	var requests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		username, _, ok := r.BasicAuth()
		if !ok || username != "sk_test_sandbox" {
			http.Error(w, `{"error":{"code":"invalid_api_key","message":"invalid key"}}`, http.StatusUnauthorized)
			return
		}
		switch r.URL.Path {
		case "/v1/checkout/sessions":
			if r.Header.Get("Idempotency-Key") != "deposit-key-0001" {
				t.Errorf("deposit idempotency key=%q", r.Header.Get("Idempotency-Key"))
			}
			_ = r.ParseForm()
			if r.Form.Get("client_reference_id") != "deposit-1" || r.Form.Get("line_items[0][price_data][unit_amount]") != "1250" {
				t.Errorf("checkout form=%v", r.Form)
			}
			_, _ = w.Write([]byte(`{"id":"cs_test_1","url":"https://checkout.stripe.test/c/pay/cs_test_1"}`))
		case "/v1/checkout/sessions/cs_test_1":
			_, _ = w.Write([]byte(`{"id":"cs_test_1","amount_total":1250,"currency":"zar","payment_status":"paid","status":"complete"}`))
		case "/v1/refunds":
			if r.Header.Get("Idempotency-Key") != "refund-key-0001" {
				t.Errorf("refund idempotency key=%q", r.Header.Get("Idempotency-Key"))
			}
			_, _ = w.Write([]byte(`{"id":"re_test_1","status":"succeeded"}`))
		case "/v1/transfers":
			if r.Header.Get("Idempotency-Key") != "withdraw-key-0001-transfer" {
				t.Errorf("withdrawal idempotency key=%q", r.Header.Get("Idempotency-Key"))
			}
			_ = r.ParseForm()
			if r.Form.Get("destination") != "acct_test_player" || r.Form.Get("amount") != "500" {
				t.Errorf("transfer form=%v", r.Form)
			}
			_, _ = w.Write([]byte(`{"id":"tr_test_1","destination":"acct_test_player"}`))
		case "/v1/payouts":
			if r.Header.Get("Idempotency-Key") != "withdraw-key-0001-payout" ||
				r.Header.Get("Stripe-Account") != "acct_test_player" {
				t.Errorf("payout headers=%v", r.Header)
			}
			_, _ = w.Write([]byte(`{"id":"po_test_1","status":"pending"}`))
		case "/v1/payouts/po_test_1":
			_, _ = w.Write([]byte(`{"id":"po_test_1","status":"paid","amount":500,"currency":"zar"}`))
		case "/v1/balance":
			_, _ = w.Write([]byte(`{"available":[{"amount":5000,"currency":"zar"}],"pending":[{"amount":1500,"currency":"zar"}]}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	provider := NewStripeProvider(StripeConfig{
		SecretKey: "sk_test_sandbox", WebhookSecret: webhookSecret, APIBase: server.URL,
	})
	session, err := provider.CreateDepositSession(context.Background(), DepositSessionRequest{
		SessionID: "deposit-1", UserID: "user-1", AmountMinor: 1_250, Currency: "ZAR",
		ReturnURL: "https://app.example/wallet", CancelURL: "https://app.example/wallet",
		IdempotencyKey: "deposit-key-0001",
	})
	if err != nil || session.ProviderRef != "cs_test_1" || session.CheckoutURL == "" {
		t.Fatalf("session=%+v err=%v", session, err)
	}
	balance, err := provider.Balance(context.Background(), "ZAR")
	if err != nil || balance.AvailableMinor != 5_000 || balance.PendingMinor != 1_500 {
		t.Fatalf("balance=%+v err=%v", balance, err)
	}
	withdrawal, err := provider.CreatePayout(context.Background(), PayoutRequest{
		WithdrawalID: "withdrawal-1", UserID: "user-1", AmountMinor: 500, Currency: "ZAR",
		IdempotencyKey: "withdraw-key-0001", Destination: "acct_test_player",
	})
	if err != nil || withdrawal.ProviderRef != "po_test_1" || withdrawal.Metadata["transferId"] != "tr_test_1" {
		t.Fatalf("withdrawal=%+v err=%v", withdrawal, err)
	}
	paymentStatus, err := provider.QueryPaymentStatus(context.Background(), PaymentStatusRequest{ProviderRef: "cs_test_1"})
	if err != nil || paymentStatus.Status != StatusSucceeded || paymentStatus.AmountMinor != 1_250 {
		t.Fatalf("payment status=%+v err=%v", paymentStatus, err)
	}
	refund, err := provider.Refund(context.Background(), RefundRequest{
		ProviderRef: "pi_test_1", ResourceID: "deposit-1", AmountMinor: 1_250,
		Currency: "ZAR", IdempotencyKey: "refund-key-0001",
	})
	if err != nil || refund.Status != StatusSucceeded {
		t.Fatalf("refund=%+v err=%v", refund, err)
	}
	payoutStatus, err := provider.QueryPayoutStatus(context.Background(), PayoutStatusRequest{ProviderRef: "po_test_1"})
	if err != nil || payoutStatus.Status != StatusSucceeded || payoutStatus.AmountMinor != 500 {
		t.Fatalf("payout status=%+v err=%v", payoutStatus, err)
	}
	reconciliation, err := provider.Reconcile(context.Background(), ReconciliationRequest{Currency: "ZAR"})
	if err != nil || reconciliation.Balance.AvailableMinor != 5_000 {
		t.Fatalf("reconciliation=%+v err=%v", reconciliation, err)
	}
	if requests.Load() != 8 {
		t.Fatalf("provider requests=%d, want 8", requests.Load())
	}
}

func TestStripeWebhookSignatureAndEvents(t *testing.T) {
	now := time.Unix(1_800_000_000, 0).UTC()
	webhookSecret := strings.Repeat("w", 32)
	provider := NewStripeProvider(StripeConfig{
		SecretKey: "sk_test_sandbox", WebhookSecret: webhookSecret,
		APIBase: "https://api.stripe.test", Clock: func() time.Time { return now },
	})
	depositBody := []byte(`{"id":"evt_deposit","type":"checkout.session.completed","data":{"object":{"id":"cs_test_1","amount_total":1250,"currency":"zar","payment_status":"paid","metadata":{"deposit_id":"deposit-1"}}}}`)
	callback := CallbackRequest{Headers: map[string][]string{
		"Stripe-Signature": {stripeTestSignature(webhookSecret, now.Unix(), depositBody)},
	}, Body: depositBody}
	if verification, err := provider.VerifySignature(context.Background(), callback); err != nil || !verification.Valid {
		t.Fatalf("verification=%+v err=%v", verification, err)
	}
	event, err := provider.ParseCallback(context.Background(), callback)
	if err != nil || event.ResourceID != "deposit-1" || event.Status != StatusSucceeded || event.AmountMinor != 1_250 {
		t.Fatalf("deposit event=%+v err=%v", event, err)
	}
	transferBody := []byte(`{"id":"evt_transfer","type":"transfer.created","data":{"object":{"id":"tr_test_1","amount":500,"currency":"zar","metadata":{"withdrawal_id":"withdrawal-1"}}}}`)
	event, err = provider.ParseCallback(context.Background(), CallbackRequest{Headers: map[string][]string{
		"Stripe-Signature": {stripeTestSignature(webhookSecret, now.Unix(), transferBody)},
	}, Body: transferBody})
	if err != nil || event.ResourceID != "withdrawal-1" || event.Status != StatusPending {
		t.Fatalf("transfer event=%+v err=%v", event, err)
	}
	payoutBody := []byte(`{"id":"evt_payout","type":"payout.paid","data":{"object":{"id":"po_test_1","amount":500,"currency":"zar","status":"paid","metadata":{"withdrawal_id":"withdrawal-1"}}}}`)
	event, err = provider.ParseCallback(context.Background(), CallbackRequest{Headers: map[string][]string{
		"Stripe-Signature": {stripeTestSignature(webhookSecret, now.Unix(), payoutBody)},
	}, Body: payoutBody})
	if err != nil || event.ResourceID != "withdrawal-1" || event.Status != StatusSucceeded {
		t.Fatalf("payout event=%+v err=%v", event, err)
	}
	if _, err := provider.VerifySignature(context.Background(), CallbackRequest{Headers: map[string][]string{
		"Stripe-Signature": {stripeTestSignature("wrong-secret", now.Unix(), depositBody)},
	}, Body: depositBody}); err == nil {
		t.Fatal("expected invalid Stripe signature to fail")
	}
	if _, err := provider.VerifySignature(context.Background(), CallbackRequest{Headers: map[string][]string{
		"Stripe-Signature": {stripeTestSignature(webhookSecret, now.Add(-10*time.Minute).Unix(), depositBody)},
	}, Body: depositBody}); err == nil {
		t.Fatal("expected expired Stripe signature to fail")
	}
}

func TestStripeProviderFailureAndOutage(t *testing.T) {
	webhookSecret := strings.Repeat("w", 32)
	failure := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
		_, _ = w.Write([]byte(`{"error":{"code":"api_error","message":"temporarily unavailable"}}`))
	}))
	defer failure.Close()
	provider := NewStripeProvider(StripeConfig{
		SecretKey: "sk_test_sandbox", WebhookSecret: webhookSecret, APIBase: failure.URL,
	})
	if _, err := provider.Balance(context.Background(), "ZAR"); err == nil || !strings.Contains(err.Error(), "api_error") {
		t.Fatalf("provider failure err=%v", err)
	}

	timeout := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		time.Sleep(100 * time.Millisecond)
	}))
	defer timeout.Close()
	provider = NewStripeProvider(StripeConfig{
		SecretKey: "sk_test_sandbox", WebhookSecret: webhookSecret, APIBase: timeout.URL,
		Client: &http.Client{Timeout: 10 * time.Millisecond},
	})
	if _, err := provider.Balance(context.Background(), "ZAR"); err == nil {
		t.Fatal("expected Stripe network timeout")
	}
}

func TestStripeConfigurationValidationIsAdapterOwned(t *testing.T) {
	webhookPrefix := "wh" + "sec_"
	sandbox := NewStripeProvider(StripeConfig{
		SecretKey: "sk_test_sandbox", WebhookSecret: webhookPrefix + strings.Repeat("t", 16),
		APIBase: "https://api.stripe.com",
	})
	if err := sandbox.ValidateConfiguration("production"); err == nil {
		t.Fatal("sandbox credentials were accepted in production")
	}
	live := NewStripeProvider(StripeConfig{
		SecretKey: "sk_live_example", WebhookSecret: webhookPrefix + strings.Repeat("e", 24),
		APIBase: "https://api.stripe.com",
	})
	if err := live.ValidateConfiguration("production"); err != nil {
		t.Fatal(err)
	}
}

func stripeTestSignature(secret string, timestamp int64, body []byte) string {
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = fmt.Fprintf(mac, "%d.", timestamp)
	_, _ = mac.Write(body)
	return fmt.Sprintf("t=%d,v1=%s", timestamp, hex.EncodeToString(mac.Sum(nil)))
}
