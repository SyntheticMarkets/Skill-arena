package payments

import (
	"context"
	"errors"
	"testing"
	"time"
)

type contractProvider struct {
	descriptor ProviderDescriptor
	healthErr  error
}

func (p contractProvider) Descriptor() ProviderDescriptor     { return p.descriptor }
func (p contractProvider) ValidateConfiguration(string) error { return nil }
func (p contractProvider) CreateDepositSession(context.Context, DepositSessionRequest) (DepositSessionResponse, error) {
	return DepositSessionResponse{ProviderRef: p.descriptor.ID + "-deposit", CheckoutURL: "https://checkout.example"}, nil
}
func (p contractProvider) VerifySignature(context.Context, CallbackRequest) (SignatureVerification, error) {
	return SignatureVerification{Valid: true, Fingerprint: "signature"}, nil
}
func (p contractProvider) ParseCallback(context.Context, CallbackRequest) (CallbackEvent, error) {
	return CallbackEvent{EventID: "event-1", Kind: EventDeposit, Status: StatusSucceeded}, nil
}
func (p contractProvider) QueryPaymentStatus(context.Context, PaymentStatusRequest) (PaymentStatusResponse, error) {
	return PaymentStatusResponse{Status: StatusSucceeded}, nil
}
func (p contractProvider) Refund(context.Context, RefundRequest) (RefundResponse, error) {
	return RefundResponse{Status: StatusSucceeded}, nil
}
func (p contractProvider) ValidatePayoutDestination(string) error { return nil }
func (p contractProvider) CreatePayout(context.Context, PayoutRequest) (PayoutResponse, error) {
	return PayoutResponse{ProviderRef: p.descriptor.ID + "-payout", Status: StatusPending}, nil
}
func (p contractProvider) QueryPayoutStatus(context.Context, PayoutStatusRequest) (PayoutStatusResponse, error) {
	return PayoutStatusResponse{Status: StatusSucceeded}, nil
}
func (p contractProvider) Balance(context.Context, string) (ProviderBalance, error) {
	return ProviderBalance{AvailableMinor: 10_000, Currency: "ZAR"}, nil
}
func (p contractProvider) Reconcile(_ context.Context, request ReconciliationRequest) (ReconciliationResult, error) {
	return ReconciliationResult{
		Balance:     ProviderBalance{AvailableMinor: 10_000, Currency: request.Currency},
		GeneratedAt: time.Now().UTC(),
	}, nil
}
func (p contractProvider) Health(context.Context) error { return p.healthErr }

func testDescriptor(id string, priority, cost int, countries, currencies, methods []string) ProviderDescriptor {
	return ProviderDescriptor{
		ID: id, DisplayName: id, Priority: priority, VariableCostBPS: cost,
		Capabilities: ProviderCapabilities{
			Deposits: true, Callbacks: true, SignatureVerify: true, PaymentStatus: true,
			Refunds: true, Payouts: true, PayoutStatus: true, Balances: true,
			Reconciliation: true, HealthChecks: true, Idempotency: true,
		},
		Methods: methods, SupportedCurrencies: currencies, SupportedCountries: countries,
	}
}

func TestRegistryRoutesByCountryCurrencyMethodPriorityAndCost(t *testing.T) {
	zaCard := contractProvider{descriptor: testDescriptor("za-card", 100, 300, []string{"ZA"}, []string{"ZAR"}, []string{"card"})}
	globalCard := contractProvider{descriptor: testDescriptor("global-card", 100, 250, []string{"*"}, []string{"*"}, []string{"card"})}
	zaEFT := contractProvider{descriptor: testDescriptor("za-eft", 120, 100, []string{"ZA"}, []string{"ZAR"}, []string{"eft"})}
	registry := NewRegistry(zaCard, globalCard, zaEFT)

	candidates := registry.Candidates(SelectionRequest{
		Operation: OperationDeposit, Country: "ZA", Currency: "ZAR", Method: "card", AmountMinor: 10_000,
	})
	if len(candidates) != 2 || candidates[0].Descriptor().ID != "global-card" {
		t.Fatalf("cost-ranked candidates=%v", providerIDs(candidates))
	}
	candidates = registry.Candidates(SelectionRequest{
		Operation: OperationDeposit, Country: "ZA", Currency: "ZAR", Method: "card",
		Preferred: "za-card", AmountMinor: 10_000,
	})
	if len(candidates) != 2 || candidates[0].Descriptor().ID != "za-card" {
		t.Fatalf("preferred candidates=%v", providerIDs(candidates))
	}
	candidates = registry.Candidates(SelectionRequest{
		Operation: OperationDeposit, Country: "US", Currency: "USD", Method: "card",
	})
	if len(candidates) != 1 || candidates[0].Descriptor().ID != "global-card" {
		t.Fatalf("country candidates=%v", providerIDs(candidates))
	}
}

func TestCoreUsesHealthyProviderPreflightFailover(t *testing.T) {
	unhealthy := contractProvider{
		descriptor: testDescriptor("primary", 200, 100, []string{"ZA"}, []string{"ZAR"}, []string{"card"}),
		healthErr:  errors.New("provider outage"),
	}
	healthy := contractProvider{
		descriptor: testDescriptor("secondary", 100, 200, []string{"ZA"}, []string{"ZAR"}, []string{"card"}),
	}
	core := NewCore(NewRegistry(unhealthy, healthy))
	selected, err := core.SelectDeposit(context.Background(), SelectionRequest{
		Country: "ZA", Currency: "ZAR", Method: "card", Preferred: "primary",
	})
	if err != nil || selected != "secondary" {
		t.Fatalf("selected=%q err=%v", selected, err)
	}
}

func TestProductionConfigurationRequiresRegisteredAdapter(t *testing.T) {
	if err := NewCore(NewRegistry()).ValidateConfiguration("production"); err == nil {
		t.Fatal("production accepted an empty provider registry")
	}
	provider := contractProvider{descriptor: testDescriptor("provider-a", 100, 100, []string{"*"}, []string{"*"}, []string{"card"})}
	if err := NewCore(NewRegistry(provider)).ValidateConfiguration("production"); err != nil {
		t.Fatal(err)
	}
}

func TestCoreExposesGenericMethodsAndFullProviderContract(t *testing.T) {
	provider := contractProvider{descriptor: testDescriptor("provider-a", 100, 100, []string{"ZA"}, []string{"ZAR"}, []string{"card"})}
	core := NewCore(NewRegistry(provider))
	methods := core.Methods(context.Background(), "ZA", "ZAR", []string{"card", "eft"})
	if len(methods) != 2 || !methods[0].Available || methods[1].Available {
		t.Fatalf("methods=%+v", methods)
	}
	if methods[0].ID != "card" || methods[0].DisplayName != "Card" {
		t.Fatalf("player-facing method leaked provider detail: %+v", methods[0])
	}
	event, verification, err := core.VerifyCallback(context.Background(), "provider-a", CallbackRequest{})
	if err != nil || !verification.Valid || event.Status != StatusSucceeded {
		t.Fatalf("event=%+v verification=%+v err=%v", event, verification, err)
	}
	if _, err := core.Refund(context.Background(), "provider-a", RefundRequest{IdempotencyKey: "refund-key"}); err != nil {
		t.Fatal(err)
	}
	if _, err := core.Refund(context.Background(), "provider-a", RefundRequest{}); err == nil {
		t.Fatal("refund without idempotency key was accepted")
	}
	if _, err := core.QueryPaymentStatus(context.Background(), "provider-a", PaymentStatusRequest{}); err != nil {
		t.Fatal(err)
	}
	if _, err := core.QueryPayoutStatus(context.Background(), "provider-a", PayoutStatusRequest{}); err != nil {
		t.Fatal(err)
	}
	if _, err := core.Reconcile(context.Background(), "provider-a", ReconciliationRequest{Currency: "ZAR"}); err != nil {
		t.Fatal(err)
	}
}

func providerIDs(providers []Provider) []string {
	result := make([]string, 0, len(providers))
	for _, provider := range providers {
		result = append(result, provider.Descriptor().ID)
	}
	return result
}
