package payments

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	"skill-arena/internal/models"
)

type HTTPDoer interface {
	Do(*http.Request) (*http.Response, error)
}

type StripeConfig struct {
	SecretKey        string
	WebhookSecret    string
	APIBase          string
	Client           HTTPDoer
	Clock            func() time.Time
	WebhookTolerance time.Duration
	Descriptor       ProviderDescriptor
}

type StripeProvider struct {
	secretKey        string
	webhookSecret    string
	apiBase          string
	client           HTTPDoer
	clock            func() time.Time
	webhookTolerance time.Duration
	descriptor       ProviderDescriptor
}

func NewStripeProvider(config StripeConfig) *StripeProvider {
	client := config.Client
	if client == nil {
		client = &http.Client{Timeout: 15 * time.Second}
	}
	clock := config.Clock
	if clock == nil {
		clock = time.Now
	}
	tolerance := config.WebhookTolerance
	if tolerance <= 0 {
		tolerance = 5 * time.Minute
	}
	provider := &StripeProvider{
		secretKey: strings.TrimSpace(config.SecretKey), webhookSecret: strings.TrimSpace(config.WebhookSecret),
		apiBase: strings.TrimRight(strings.TrimSpace(config.APIBase), "/"), client: client,
		clock: clock, webhookTolerance: tolerance, descriptor: config.Descriptor,
	}
	if provider.descriptor.ID == "" {
		provider.descriptor = defaultStripeDescriptor()
	}
	return provider
}

func (p *StripeProvider) Name() string {
	return ProviderStripe
}

func (p *StripeProvider) Descriptor() ProviderDescriptor {
	return p.descriptor
}

func (p *StripeProvider) ValidateConfiguration(environment string) error {
	if p.secretKey == "" || p.webhookSecret == "" || p.apiBase == "" {
		return errors.New("credentials are incomplete")
	}
	parsed, err := url.Parse(p.apiBase)
	if err != nil || parsed.Scheme != "https" || parsed.Host == "" {
		return errors.New("API base must be an absolute HTTPS URL")
	}
	if strings.EqualFold(environment, "production") {
		if !strings.HasPrefix(p.secretKey, "sk_live_") || !strings.HasPrefix(p.webhookSecret, "whsec_") ||
			parsed.Host != "api.stripe.com" {
			return errors.New("production requires live credentials and the official API host")
		}
	}
	return nil
}

func defaultStripeDescriptor() ProviderDescriptor {
	return ProviderDescriptor{
		ID: ProviderStripe, DisplayName: "Card", Priority: 100,
		VariableCostBPS: 290, FixedCostMinor: 30,
		Capabilities: ProviderCapabilities{
			Deposits: true, Callbacks: true, SignatureVerify: true, PaymentStatus: true,
			Refunds: true, Payouts: true, PayoutStatus: true, Balances: true,
			Reconciliation: true, HealthChecks: true, Idempotency: true,
		},
		Methods: []string{"card"}, SupportedCurrencies: []string{"*"}, SupportedCountries: []string{"*"},
	}
}

func (p *StripeProvider) ValidatePayoutDestination(value string) error {
	if !regexp.MustCompile(`^acct_[A-Za-z0-9]+$`).MatchString(strings.TrimSpace(value)) {
		return errors.New("valid Stripe Connect account is required")
	}
	return nil
}

func (p *StripeProvider) CreateDepositSession(ctx context.Context, request DepositSessionRequest) (DepositSessionResponse, error) {
	if request.SessionID == "" || request.AmountMinor <= 0 || request.Currency == "" {
		return DepositSessionResponse{}, errors.New("valid Stripe deposit request is required")
	}
	values := url.Values{
		"mode":                                          {"payment"},
		"success_url":                                   {request.ReturnURL},
		"cancel_url":                                    {request.CancelURL},
		"client_reference_id":                           {request.SessionID},
		"payment_method_types[]":                        {"card"},
		"line_items[0][quantity]":                       {"1"},
		"line_items[0][price_data][currency]":           {strings.ToLower(request.Currency)},
		"line_items[0][price_data][unit_amount]":        {strconv.FormatInt(int64(request.AmountMinor), 10)},
		"line_items[0][price_data][product_data][name]": {"Skill Arena wallet deposit"},
		"metadata[deposit_id]":                          {request.SessionID},
		"metadata[user_id]":                             {request.UserID},
		"payment_intent_data[metadata][deposit_id]":     {request.SessionID},
	}
	if email := strings.TrimSpace(request.Metadata["email"]); email != "" {
		values.Set("customer_email", email)
	}
	body, err := p.doForm(ctx, http.MethodPost, "/v1/checkout/sessions", values, request.IdempotencyKey)
	if err != nil {
		return DepositSessionResponse{}, err
	}
	var response struct {
		ID  string `json:"id"`
		URL string `json:"url"`
	}
	if err := json.Unmarshal(body, &response); err != nil || response.ID == "" || response.URL == "" {
		return DepositSessionResponse{}, errors.New("Stripe returned an invalid Checkout Session")
	}
	return DepositSessionResponse{
		ProviderRef: response.ID, CheckoutURL: response.URL,
		Metadata: map[string]string{"providerObject": "checkout.session"},
	}, nil
}

func (p *StripeProvider) CreatePayout(ctx context.Context, request PayoutRequest) (PayoutResponse, error) {
	destination := strings.TrimSpace(request.Destination)
	if request.WithdrawalID == "" || request.AmountMinor <= 0 || request.Currency == "" || destination == "" {
		return PayoutResponse{}, errors.New("verified Stripe payout destination is required")
	}
	values := url.Values{
		"amount":                  {strconv.FormatInt(int64(request.AmountMinor), 10)},
		"currency":                {strings.ToLower(request.Currency)},
		"destination":             {destination},
		"transfer_group":          {"withdrawal_" + request.WithdrawalID},
		"metadata[withdrawal_id]": {request.WithdrawalID},
		"description":             {"Skill Arena withdrawal " + request.WithdrawalID},
	}
	body, err := p.doForm(ctx, http.MethodPost, "/v1/transfers", values, request.IdempotencyKey+"-transfer")
	if err != nil {
		return PayoutResponse{}, err
	}
	var transfer struct {
		ID          string `json:"id"`
		Destination string `json:"destination"`
	}
	if err := json.Unmarshal(body, &transfer); err != nil || transfer.ID == "" {
		return PayoutResponse{}, errors.New("Stripe returned an invalid transfer")
	}
	payoutValues := url.Values{
		"amount":                  {strconv.FormatInt(int64(request.AmountMinor), 10)},
		"currency":                {strings.ToLower(request.Currency)},
		"metadata[withdrawal_id]": {request.WithdrawalID},
		"description":             {"Skill Arena withdrawal " + request.WithdrawalID},
		"statement_descriptor":    {"SKILL ARENA"},
	}
	body, err = p.doFormForAccount(ctx, http.MethodPost, "/v1/payouts", payoutValues, request.IdempotencyKey+"-payout", destination)
	if err != nil {
		return PayoutResponse{}, err
	}
	var payout struct {
		ID     string `json:"id"`
		Status string `json:"status"`
	}
	if err := json.Unmarshal(body, &payout); err != nil || payout.ID == "" {
		return PayoutResponse{}, errors.New("Stripe returned an invalid payout")
	}
	return PayoutResponse{
		ProviderRef: payout.ID, Status: StatusPending,
		Metadata: map[string]string{
			"destination": transfer.Destination, "transferId": transfer.ID,
			"payoutStatus": payout.Status, "providerObject": "payout",
		},
	}, nil
}

func (p *StripeProvider) Balance(ctx context.Context, currency string) (ProviderBalance, error) {
	body, err := p.doForm(ctx, http.MethodGet, "/v1/balance", nil, "")
	if err != nil {
		return ProviderBalance{}, err
	}
	var response struct {
		Available []struct {
			Amount   models.MinorUnits `json:"amount"`
			Currency string            `json:"currency"`
		} `json:"available"`
		Pending []struct {
			Amount   models.MinorUnits `json:"amount"`
			Currency string            `json:"currency"`
		} `json:"pending"`
	}
	if err := json.Unmarshal(body, &response); err != nil {
		return ProviderBalance{}, errors.New("Stripe returned an invalid balance")
	}
	target := strings.ToLower(strings.TrimSpace(currency))
	result := ProviderBalance{Currency: strings.ToUpper(target)}
	for _, item := range response.Available {
		if strings.EqualFold(item.Currency, target) {
			result.AvailableMinor += item.Amount
		}
	}
	for _, item := range response.Pending {
		if strings.EqualFold(item.Currency, target) {
			result.PendingMinor += item.Amount
		}
	}
	return result, nil
}

func (p *StripeProvider) Health(ctx context.Context) error {
	if p.secretKey == "" || p.webhookSecret == "" || p.apiBase == "" {
		return errors.New("Stripe credentials are incomplete")
	}
	_, err := p.Balance(ctx, "")
	return err
}

func (p *StripeProvider) VerifySignature(_ context.Context, request CallbackRequest) (SignatureVerification, error) {
	header := request.Header("Stripe-Signature")
	if err := p.verifySignature(header, request.Body); err != nil {
		return SignatureVerification{}, err
	}
	sum := sha256.Sum256([]byte(header))
	return SignatureVerification{Valid: true, Fingerprint: hex.EncodeToString(sum[:])}, nil
}

func (p *StripeProvider) ParseCallback(_ context.Context, request CallbackRequest) (CallbackEvent, error) {
	var envelope struct {
		ID   string `json:"id"`
		Type string `json:"type"`
		Data struct {
			Object json.RawMessage `json:"object"`
		} `json:"data"`
	}
	if err := json.Unmarshal(request.Body, &envelope); err != nil || envelope.ID == "" || envelope.Type == "" {
		return CallbackEvent{}, errors.New("invalid Stripe event")
	}
	event := CallbackEvent{
		EventID:  envelope.ID,
		Metadata: map[string]string{"eventType": envelope.Type},
	}
	switch envelope.Type {
	case "checkout.session.completed", "checkout.session.async_payment_succeeded", "checkout.session.async_payment_failed", "checkout.session.expired":
		var session struct {
			ID            string            `json:"id"`
			AmountTotal   models.MinorUnits `json:"amount_total"`
			Currency      string            `json:"currency"`
			PaymentStatus string            `json:"payment_status"`
			Metadata      map[string]string `json:"metadata"`
		}
		if err := json.Unmarshal(envelope.Data.Object, &session); err != nil {
			return CallbackEvent{}, errors.New("invalid Stripe Checkout Session event")
		}
		event.Kind = EventDeposit
		event.ProviderRef = session.ID
		event.ResourceID = session.Metadata["deposit_id"]
		event.AmountMinor = session.AmountTotal
		event.Currency = strings.ToUpper(session.Currency)
		switch envelope.Type {
		case "checkout.session.async_payment_failed", "checkout.session.expired":
			if envelope.Type == "checkout.session.expired" {
				event.Status = StatusExpired
			} else {
				event.Status = StatusFailed
			}
		case "checkout.session.completed":
			if session.PaymentStatus == "paid" {
				event.Status = StatusSucceeded
			} else {
				event.Status = StatusPending
			}
		default:
			event.Status = StatusSucceeded
		}
	case "transfer.created", "transfer.reversed", "transfer.failed":
		var transfer struct {
			ID       string            `json:"id"`
			Amount   models.MinorUnits `json:"amount"`
			Currency string            `json:"currency"`
			Metadata map[string]string `json:"metadata"`
		}
		if err := json.Unmarshal(envelope.Data.Object, &transfer); err != nil {
			return CallbackEvent{}, errors.New("invalid Stripe transfer event")
		}
		event.Kind = EventPayout
		event.ProviderRef = transfer.ID
		event.ResourceID = transfer.Metadata["withdrawal_id"]
		event.AmountMinor = transfer.Amount
		event.Currency = strings.ToUpper(transfer.Currency)
		event.Status = StatusPending
		if envelope.Type != "transfer.created" {
			event.Status = StatusFailed
		}
	case "payout.created", "payout.paid", "payout.failed", "payout.canceled":
		var payout struct {
			ID       string            `json:"id"`
			Amount   models.MinorUnits `json:"amount"`
			Currency string            `json:"currency"`
			Status   string            `json:"status"`
			Metadata map[string]string `json:"metadata"`
		}
		if err := json.Unmarshal(envelope.Data.Object, &payout); err != nil {
			return CallbackEvent{}, errors.New("invalid Stripe payout event")
		}
		event.Kind = EventPayout
		event.ProviderRef = payout.ID
		event.ResourceID = payout.Metadata["withdrawal_id"]
		event.AmountMinor = payout.Amount
		event.Currency = strings.ToUpper(payout.Currency)
		event.Status = StatusPending
		if envelope.Type == "payout.paid" {
			event.Status = StatusSucceeded
		}
		if envelope.Type == "payout.failed" || envelope.Type == "payout.canceled" {
			event.Status = StatusFailed
		}
	default:
		event.Kind = EventIgnored
		event.Status = StatusUnknown
	}
	return event, nil
}

func (p *StripeProvider) QueryPaymentStatus(ctx context.Context, request PaymentStatusRequest) (PaymentStatusResponse, error) {
	if strings.TrimSpace(request.ProviderRef) == "" {
		return PaymentStatusResponse{}, errors.New("Stripe Checkout Session reference is required")
	}
	body, err := p.doForm(ctx, http.MethodGet, "/v1/checkout/sessions/"+url.PathEscape(request.ProviderRef), nil, "")
	if err != nil {
		return PaymentStatusResponse{}, err
	}
	var response struct {
		ID            string            `json:"id"`
		AmountTotal   models.MinorUnits `json:"amount_total"`
		Currency      string            `json:"currency"`
		PaymentStatus string            `json:"payment_status"`
		Status        string            `json:"status"`
	}
	if err := json.Unmarshal(body, &response); err != nil || response.ID == "" {
		return PaymentStatusResponse{}, errors.New("Stripe returned an invalid Checkout Session status")
	}
	status := StatusPending
	if response.PaymentStatus == "paid" {
		status = StatusSucceeded
	} else if response.Status == "expired" {
		status = StatusExpired
	}
	return PaymentStatusResponse{
		ProviderRef: response.ID, Status: status, AmountMinor: response.AmountTotal,
		Currency: strings.ToUpper(response.Currency),
	}, nil
}

func (p *StripeProvider) Refund(ctx context.Context, request RefundRequest) (RefundResponse, error) {
	if request.ProviderRef == "" || request.AmountMinor <= 0 || request.IdempotencyKey == "" {
		return RefundResponse{}, errors.New("valid Stripe refund request is required")
	}
	values := url.Values{
		"payment_intent":        {request.ProviderRef},
		"amount":                {strconv.FormatInt(int64(request.AmountMinor), 10)},
		"metadata[resource_id]": {request.ResourceID},
	}
	if request.Reason != "" {
		values.Set("metadata[reason]", request.Reason)
	}
	body, err := p.doForm(ctx, http.MethodPost, "/v1/refunds", values, request.IdempotencyKey)
	if err != nil {
		return RefundResponse{}, err
	}
	var response struct {
		ID     string `json:"id"`
		Status string `json:"status"`
	}
	if err := json.Unmarshal(body, &response); err != nil || response.ID == "" {
		return RefundResponse{}, errors.New("Stripe returned an invalid refund")
	}
	status := StatusPending
	if response.Status == "succeeded" {
		status = StatusSucceeded
	} else if response.Status == "failed" || response.Status == "canceled" {
		status = StatusFailed
	}
	return RefundResponse{ProviderRef: response.ID, Status: status}, nil
}

func (p *StripeProvider) QueryPayoutStatus(ctx context.Context, request PayoutStatusRequest) (PayoutStatusResponse, error) {
	if request.ProviderRef == "" {
		return PayoutStatusResponse{}, errors.New("Stripe payout reference is required")
	}
	body, err := p.doForm(ctx, http.MethodGet, "/v1/payouts/"+url.PathEscape(request.ProviderRef), nil, "")
	if err != nil {
		return PayoutStatusResponse{}, err
	}
	var response struct {
		ID       string            `json:"id"`
		Status   string            `json:"status"`
		Amount   models.MinorUnits `json:"amount"`
		Currency string            `json:"currency"`
	}
	if err := json.Unmarshal(body, &response); err != nil || response.ID == "" {
		return PayoutStatusResponse{}, errors.New("Stripe returned an invalid payout status")
	}
	status := StatusPending
	if response.Status == "paid" {
		status = StatusSucceeded
	} else if response.Status == "failed" || response.Status == "canceled" {
		status = StatusFailed
	}
	return PayoutStatusResponse{
		ProviderRef: response.ID, Status: status, AmountMinor: response.Amount,
		Currency: strings.ToUpper(response.Currency),
	}, nil
}

func (p *StripeProvider) Reconcile(ctx context.Context, request ReconciliationRequest) (ReconciliationResult, error) {
	balance, err := p.Balance(ctx, request.Currency)
	if err != nil {
		return ReconciliationResult{}, err
	}
	return ReconciliationResult{
		Balance: balance, ProviderRef: "balance:" + strings.ToUpper(request.Currency),
		GeneratedAt: p.clock().UTC(),
	}, nil
}

func (p *StripeProvider) verifySignature(header string, body []byte) error {
	if p.webhookSecret == "" || strings.TrimSpace(header) == "" {
		return errors.New("Stripe webhook signature is required")
	}
	var timestamp int64
	signatures := []string{}
	for _, part := range strings.Split(header, ",") {
		key, value, found := strings.Cut(strings.TrimSpace(part), "=")
		if !found {
			continue
		}
		switch key {
		case "t":
			timestamp, _ = strconv.ParseInt(value, 10, 64)
		case "v1":
			signatures = append(signatures, value)
		}
	}
	if timestamp <= 0 || len(signatures) == 0 {
		return errors.New("Stripe webhook signature is malformed")
	}
	eventTime := time.Unix(timestamp, 0)
	delta := p.clock().Sub(eventTime)
	if delta < 0 {
		delta = -delta
	}
	if delta > p.webhookTolerance {
		return errors.New("Stripe webhook signature has expired")
	}
	mac := hmac.New(sha256.New, []byte(p.webhookSecret))
	_, _ = fmt.Fprintf(mac, "%d.", timestamp)
	_, _ = mac.Write(body)
	expected := hex.EncodeToString(mac.Sum(nil))
	for _, signature := range signatures {
		if hmac.Equal([]byte(strings.ToLower(signature)), []byte(expected)) {
			return nil
		}
	}
	return errors.New("Stripe webhook signature is invalid")
}

func (p *StripeProvider) doForm(ctx context.Context, method, path string, values url.Values, idempotencyKey string) ([]byte, error) {
	return p.doFormForAccount(ctx, method, path, values, idempotencyKey, "")
}

func (p *StripeProvider) doFormForAccount(ctx context.Context, method, path string, values url.Values, idempotencyKey, account string) ([]byte, error) {
	if p.secretKey == "" || p.apiBase == "" {
		return nil, errors.New("Stripe credentials are incomplete")
	}
	var body io.Reader
	if values != nil {
		body = strings.NewReader(values.Encode())
	}
	request, err := http.NewRequestWithContext(ctx, method, p.apiBase+path, body)
	if err != nil {
		return nil, err
	}
	request.SetBasicAuth(p.secretKey, "")
	if values != nil {
		request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	}
	if idempotencyKey != "" {
		request.Header.Set("Idempotency-Key", idempotencyKey)
	}
	if account != "" {
		request.Header.Set("Stripe-Account", account)
	}
	response, err := p.client.Do(request)
	if err != nil {
		return nil, fmt.Errorf("Stripe request failed: %w", err)
	}
	defer response.Body.Close()
	payload, err := io.ReadAll(io.LimitReader(response.Body, 1<<20))
	if err != nil {
		return nil, err
	}
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		var providerError struct {
			Error struct {
				Code    string `json:"code"`
				Message string `json:"message"`
			} `json:"error"`
		}
		_ = json.Unmarshal(payload, &providerError)
		if providerError.Error.Code == "" {
			providerError.Error.Code = "provider_error"
		}
		return nil, fmt.Errorf("Stripe %s: %s", providerError.Error.Code, providerError.Error.Message)
	}
	return payload, nil
}
