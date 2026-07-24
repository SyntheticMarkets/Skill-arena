package payments

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"skill-arena/internal/models"
)

const (
	ProviderPayFast     = "payfast"
	ProviderOzow        = "ozow"
	ProviderCard        = "card"
	ProviderBankEFT     = "bank_eft"
	ProviderPeach       = "peach"
	ProviderFlutterwave = "flutterwave"
	ProviderPayPal      = "paypal"
	ProviderStripe      = "stripe"
	ProviderXsolla      = "xsolla"
	ProviderCrypto      = "crypto"
)

type Operation string

const (
	OperationDeposit Operation = "deposit"
	OperationPayout  Operation = "payout"
)

type EventKind string

const (
	EventDeposit EventKind = "deposit"
	EventPayout  EventKind = "payout"
	EventIgnored EventKind = "ignored"
)

type ProviderStatus string

const (
	StatusPending   ProviderStatus = "pending"
	StatusSucceeded ProviderStatus = "succeeded"
	StatusFailed    ProviderStatus = "failed"
	StatusExpired   ProviderStatus = "expired"
	StatusUnknown   ProviderStatus = "unknown"
)

type ProviderCapabilities struct {
	Deposits        bool
	Callbacks       bool
	SignatureVerify bool
	PaymentStatus   bool
	Refunds         bool
	Payouts         bool
	PayoutStatus    bool
	Balances        bool
	Reconciliation  bool
	HealthChecks    bool
	Idempotency     bool
}

type ProviderDescriptor struct {
	ID                  string
	DisplayName         string
	Priority            int
	VariableCostBPS     int
	FixedCostMinor      models.MinorUnits
	Capabilities        ProviderCapabilities
	Methods             []string
	SupportedCurrencies []string
	SupportedCountries  []string
}

type DepositSessionRequest struct {
	SessionID      string
	UserID         string
	AmountMinor    models.MinorUnits
	Currency       string
	Country        string
	Method         string
	ReturnURL      string
	CancelURL      string
	NotifyURL      string
	IdempotencyKey string
	Metadata       map[string]string
}

type DepositSessionResponse struct {
	ProviderRef string
	CheckoutURL string
	Metadata    map[string]string
}

type PayoutRequest struct {
	WithdrawalID   string
	UserID         string
	AmountMinor    models.MinorUnits
	Currency       string
	Country        string
	Method         string
	Destination    string
	IdempotencyKey string
	Metadata       map[string]string
}

type PayoutResponse struct {
	ProviderRef string
	Status      ProviderStatus
	Metadata    map[string]string
}

type CallbackRequest struct {
	Headers map[string][]string
	Body    []byte
}

func (r CallbackRequest) Header(name string) string {
	for key, values := range r.Headers {
		if strings.EqualFold(key, name) && len(values) > 0 {
			return values[0]
		}
	}
	return ""
}

type SignatureVerification struct {
	Valid       bool
	Fingerprint string
}

type CallbackEvent struct {
	EventID     string
	Kind        EventKind
	ProviderRef string
	ResourceID  string
	Status      ProviderStatus
	AmountMinor models.MinorUnits
	Currency    string
	Metadata    map[string]string
}

type PaymentStatusRequest struct {
	ProviderRef string
	ResourceID  string
}

type PaymentStatusResponse struct {
	ProviderRef string
	Status      ProviderStatus
	AmountMinor models.MinorUnits
	Currency    string
	Metadata    map[string]string
}

type RefundRequest struct {
	ProviderRef    string
	ResourceID     string
	AmountMinor    models.MinorUnits
	Currency       string
	Reason         string
	IdempotencyKey string
}

type RefundResponse struct {
	ProviderRef string
	Status      ProviderStatus
}

type PayoutStatusRequest struct {
	ProviderRef string
	ResourceID  string
}

type PayoutStatusResponse struct {
	ProviderRef string
	Status      ProviderStatus
	AmountMinor models.MinorUnits
	Currency    string
	Metadata    map[string]string
}

type ProviderBalance struct {
	AvailableMinor models.MinorUnits
	PendingMinor   models.MinorUnits
	Currency       string
}

type ReconciliationRequest struct {
	Currency string
	From     time.Time
	To       time.Time
}

type ReconciliationResult struct {
	Balance     ProviderBalance
	ProviderRef string
	GeneratedAt time.Time
	Metadata    map[string]string
}

type Provider interface {
	Descriptor() ProviderDescriptor
	ValidateConfiguration(environment string) error
	CreateDepositSession(context.Context, DepositSessionRequest) (DepositSessionResponse, error)
	VerifySignature(context.Context, CallbackRequest) (SignatureVerification, error)
	ParseCallback(context.Context, CallbackRequest) (CallbackEvent, error)
	QueryPaymentStatus(context.Context, PaymentStatusRequest) (PaymentStatusResponse, error)
	Refund(context.Context, RefundRequest) (RefundResponse, error)
	ValidatePayoutDestination(string) error
	CreatePayout(context.Context, PayoutRequest) (PayoutResponse, error)
	QueryPayoutStatus(context.Context, PayoutStatusRequest) (PayoutStatusResponse, error)
	Balance(context.Context, string) (ProviderBalance, error)
	Reconcile(context.Context, ReconciliationRequest) (ReconciliationResult, error)
	Health(context.Context) error
}

type SelectionRequest struct {
	Operation   Operation
	Country     string
	Currency    string
	Method      string
	Preferred   string
	AmountMinor models.MinorUnits
}

type Registry struct {
	providers map[string]Provider
}

func NewRegistry(providers ...Provider) *Registry {
	registry := &Registry{providers: map[string]Provider{}}
	for _, provider := range providers {
		if provider == nil {
			continue
		}
		descriptor := provider.Descriptor()
		if descriptor.ID != "" {
			registry.providers[descriptor.ID] = provider
		}
	}
	return registry
}

func (r *Registry) Provider(name string) (Provider, error) {
	if r == nil {
		return nil, errors.New("payment provider registry is not configured")
	}
	provider := r.providers[strings.ToLower(strings.TrimSpace(name))]
	if provider == nil {
		return nil, errors.New("payment provider is not configured")
	}
	return provider, nil
}

func (r *Registry) Candidates(request SelectionRequest) []Provider {
	if r == nil {
		return nil
	}
	candidates := make([]Provider, 0, len(r.providers))
	for _, provider := range r.providers {
		if providerSupports(provider.Descriptor(), request) {
			candidates = append(candidates, provider)
		}
	}
	sort.SliceStable(candidates, func(i, j int) bool {
		left, right := candidates[i].Descriptor(), candidates[j].Descriptor()
		leftPreferred := request.Preferred != "" && left.ID == request.Preferred
		rightPreferred := request.Preferred != "" && right.ID == request.Preferred
		if leftPreferred != rightPreferred {
			return leftPreferred
		}
		leftCost := int64(left.FixedCostMinor) + int64(request.AmountMinor)*int64(left.VariableCostBPS)/10_000
		rightCost := int64(right.FixedCostMinor) + int64(request.AmountMinor)*int64(right.VariableCostBPS)/10_000
		if left.Priority != right.Priority {
			return left.Priority > right.Priority
		}
		if leftCost != rightCost {
			return leftCost < rightCost
		}
		return left.ID < right.ID
	})
	return candidates
}

func (r *Registry) Names() []string {
	if r == nil {
		return nil
	}
	names := make([]string, 0, len(r.providers))
	for name := range r.providers {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

type Core struct {
	registry *Registry
}

func NewCore(registry *Registry) *Core {
	if registry == nil {
		registry = NewRegistry()
	}
	return &Core{registry: registry}
}

func (c *Core) Select(ctx context.Context, request SelectionRequest) (Provider, error) {
	for _, provider := range c.registry.Candidates(request) {
		descriptor := provider.Descriptor()
		if descriptor.Capabilities.HealthChecks {
			if err := provider.Health(ctx); err != nil {
				continue
			}
		}
		return provider, nil
	}
	return nil, fmt.Errorf("no healthy payment provider supports %s/%s/%s", request.Country, request.Currency, request.Method)
}

func (c *Core) CreateDepositSession(ctx context.Context, selection SelectionRequest, request DepositSessionRequest) (string, DepositSessionResponse, error) {
	selection.Operation = OperationDeposit
	provider, err := c.Select(ctx, selection)
	if err != nil {
		return "", DepositSessionResponse{}, err
	}
	response, err := c.CreateDepositSessionForProvider(ctx, provider.Descriptor().ID, request)
	return provider.Descriptor().ID, response, err
}

func (c *Core) SelectDeposit(ctx context.Context, selection SelectionRequest) (string, error) {
	selection.Operation = OperationDeposit
	provider, err := c.Select(ctx, selection)
	if err != nil {
		return "", err
	}
	return provider.Descriptor().ID, nil
}

func (c *Core) CreateDepositSessionForProvider(ctx context.Context, providerID string, request DepositSessionRequest) (DepositSessionResponse, error) {
	if strings.TrimSpace(request.IdempotencyKey) == "" {
		return DepositSessionResponse{}, errors.New("payment idempotency key is required")
	}
	provider, err := c.registry.Provider(providerID)
	if err != nil {
		return DepositSessionResponse{}, err
	}
	return provider.CreateDepositSession(ctx, request)
}

func (c *Core) SelectPayout(ctx context.Context, selection SelectionRequest) (string, error) {
	selection.Operation = OperationPayout
	provider, err := c.Select(ctx, selection)
	if err != nil {
		return "", err
	}
	return provider.Descriptor().ID, nil
}

func (c *Core) VerifyCallback(ctx context.Context, providerID string, request CallbackRequest) (CallbackEvent, SignatureVerification, error) {
	provider, err := c.registry.Provider(providerID)
	if err != nil {
		return CallbackEvent{}, SignatureVerification{}, err
	}
	verification, err := provider.VerifySignature(ctx, request)
	if err != nil || !verification.Valid {
		if err == nil {
			err = errors.New("provider callback signature is invalid")
		}
		return CallbackEvent{}, verification, err
	}
	event, err := provider.ParseCallback(ctx, request)
	return event, verification, err
}

func (c *Core) QueryPaymentStatus(ctx context.Context, providerID string, request PaymentStatusRequest) (PaymentStatusResponse, error) {
	provider, err := c.registry.Provider(providerID)
	if err != nil {
		return PaymentStatusResponse{}, err
	}
	return provider.QueryPaymentStatus(ctx, request)
}

func (c *Core) Refund(ctx context.Context, providerID string, request RefundRequest) (RefundResponse, error) {
	if strings.TrimSpace(request.IdempotencyKey) == "" {
		return RefundResponse{}, errors.New("refund idempotency key is required")
	}
	provider, err := c.registry.Provider(providerID)
	if err != nil {
		return RefundResponse{}, err
	}
	return provider.Refund(ctx, request)
}

func (c *Core) ValidatePayoutDestination(providerID, value string) error {
	provider, err := c.registry.Provider(providerID)
	if err != nil {
		return err
	}
	return provider.ValidatePayoutDestination(value)
}

func (c *Core) CreatePayout(ctx context.Context, providerID string, request PayoutRequest) (PayoutResponse, error) {
	if strings.TrimSpace(request.IdempotencyKey) == "" {
		return PayoutResponse{}, errors.New("payout idempotency key is required")
	}
	provider, err := c.registry.Provider(providerID)
	if err != nil {
		return PayoutResponse{}, err
	}
	return provider.CreatePayout(ctx, request)
}

func (c *Core) QueryPayoutStatus(ctx context.Context, providerID string, request PayoutStatusRequest) (PayoutStatusResponse, error) {
	provider, err := c.registry.Provider(providerID)
	if err != nil {
		return PayoutStatusResponse{}, err
	}
	return provider.QueryPayoutStatus(ctx, request)
}

func (c *Core) Balance(ctx context.Context, providerID, currency string) (ProviderBalance, error) {
	provider, err := c.registry.Provider(providerID)
	if err != nil {
		return ProviderBalance{}, err
	}
	return provider.Balance(ctx, currency)
}

func (c *Core) Reconcile(ctx context.Context, providerID string, request ReconciliationRequest) (ReconciliationResult, error) {
	provider, err := c.registry.Provider(providerID)
	if err != nil {
		return ReconciliationResult{}, err
	}
	return provider.Reconcile(ctx, request)
}

func (c *Core) Methods(_ context.Context, country, currency string, allowed []string) []models.PaymentMethod {
	labels := map[string]string{"card": "Card", "eft": "Instant EFT", "bank_transfer": "Bank transfer"}
	methods := make([]models.PaymentMethod, 0, len(allowed))
	for _, method := range allowed {
		candidates := c.registry.Candidates(SelectionRequest{
			Operation: OperationDeposit, Country: country, Currency: currency, Method: method,
		})
		item := models.PaymentMethod{
			ID: method, Type: method, DisplayName: labels[method], Currency: currency, Available: len(candidates) > 0,
		}
		if item.DisplayName == "" {
			item.DisplayName = method
		}
		if len(candidates) == 0 {
			item.Reason = "Unavailable until an approved provider is configured."
		}
		methods = append(methods, item)
	}
	return methods
}

func (c *Core) Health(ctx context.Context) error {
	names := c.registry.Names()
	if len(names) == 0 {
		return nil
	}
	var failures []string
	for _, name := range names {
		provider, _ := c.registry.Provider(name)
		if err := provider.Health(ctx); err != nil {
			failures = append(failures, name+": "+err.Error())
		}
	}
	if len(failures) == len(names) {
		return fmt.Errorf("all payment providers are unhealthy: %s", strings.Join(failures, "; "))
	}
	return nil
}

func (c *Core) ValidateConfiguration(environment string) error {
	names := c.registry.Names()
	if strings.EqualFold(environment, "production") && len(names) == 0 {
		return errors.New("no active payment provider adapter is configured")
	}
	for _, name := range names {
		provider, _ := c.registry.Provider(name)
		if err := provider.ValidateConfiguration(environment); err != nil {
			return fmt.Errorf("%s configuration: %w", name, err)
		}
	}
	return nil
}

func (c *Core) ProviderNames() []string {
	return c.registry.Names()
}

func providerSupports(descriptor ProviderDescriptor, request SelectionRequest) bool {
	capabilities := descriptor.Capabilities
	if request.Operation == OperationDeposit && (!capabilities.Deposits || !capabilities.Callbacks ||
		!capabilities.SignatureVerify || !capabilities.PaymentStatus || !capabilities.Refunds || !capabilities.Idempotency) {
		return false
	}
	if request.Operation == OperationPayout && (!capabilities.Payouts || !capabilities.PayoutStatus ||
		!capabilities.Balances || !capabilities.Reconciliation || !capabilities.Idempotency) {
		return false
	}
	return supportsValue(descriptor.Methods, request.Method) &&
		supportsValue(descriptor.SupportedCurrencies, request.Currency) &&
		supportsValue(descriptor.SupportedCountries, request.Country)
}

func supportsValue(supported []string, value string) bool {
	value = strings.ToUpper(strings.TrimSpace(value))
	if value == "" {
		return true
	}
	for _, item := range supported {
		item = strings.ToUpper(strings.TrimSpace(item))
		if item == "*" || item == value {
			return true
		}
	}
	return false
}
