package compliance

import (
	"context"
	"time"
)

type EvidenceReference struct {
	ID          string
	Type        string
	ObjectKey   string
	ContentType string
	SHA256      string
}

type VerificationRequest struct {
	UserID      string
	Country     string
	CallbackURL string
	Evidence    []EvidenceReference
}

type VerificationSession struct {
	ProviderReference string
	RedirectURL       string
	ExpiresAt         time.Time
}

type VerificationResult struct {
	EventID            string
	ProviderReference  string
	UserID             string
	Status             string
	RiskClassification string
	ReasonCodes        []string
	OccurredAt         time.Time
}

type ScreeningRequest struct {
	UserID        string
	Country       string
	AmountMinor   int64
	Currency      string
	Operation     string
	VelocityMinor int64
	SourceOfFunds string
}

type ScreeningResult struct {
	ProviderReference string
	Decision          string
	RiskScore         int
	ReasonCodes       []string
}

type Provider interface {
	Name() string
	StartVerification(context.Context, VerificationRequest) (VerificationSession, error)
	ParseVerificationWebhook(context.Context, map[string]string, []byte) (VerificationResult, error)
	Screen(context.Context, ScreeningRequest) (ScreeningResult, error)
	Health(context.Context) error
}
