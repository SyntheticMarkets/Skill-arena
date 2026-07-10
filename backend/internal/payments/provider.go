package payments

import (
	"context"
	"errors"
)

const (
	ProviderPayFast = "payfast"
	ProviderOzow    = "ozow"
	ProviderCard    = "card"
	ProviderBankEFT = "bank_eft"
	ProviderCrypto  = "crypto"
)

type DepositSessionRequest struct {
	SessionID      string
	UserID         string
	Amount         float64
	Currency       string
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

type ProviderWithdrawalRequest struct {
	WithdrawalID   string
	UserID         string
	Amount         float64
	Currency       string
	IdempotencyKey string
	Metadata       map[string]string
}

type WithdrawalResponse struct {
	ProviderRef string
	Status      string
	Metadata    map[string]string
}

type WebhookEvent struct {
	Provider       string
	ProviderRef    string
	SessionID      string
	WithdrawalID   string
	Status         string
	Amount         float64
	Currency       string
	SignatureValid bool
	Metadata       map[string]string
}

type Provider interface {
	Name() string
	CreateDepositSession(context.Context, DepositSessionRequest) (DepositSessionResponse, error)
	CreateWithdrawal(context.Context, ProviderWithdrawalRequest) (WithdrawalResponse, error)
	ParseWebhook(context.Context, map[string]string, []byte) (WebhookEvent, error)
}

type Registry struct {
	providers map[string]Provider
}

func NewRegistry(providers ...Provider) *Registry {
	registry := &Registry{providers: map[string]Provider{}}
	for _, provider := range providers {
		if provider != nil {
			registry.providers[provider.Name()] = provider
		}
	}
	return registry
}

func (r *Registry) Provider(name string) (Provider, error) {
	if r == nil {
		return nil, errors.New("payment provider registry is not configured")
	}
	provider := r.providers[name]
	if provider == nil {
		return nil, errors.New("payment provider is not configured")
	}
	return provider, nil
}
