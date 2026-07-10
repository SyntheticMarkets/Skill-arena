package models

import "time"

const (
	TransactionTypeDeposit  = "deposit"
	TransactionTypeWithdraw = "withdraw"
	TransactionTypeFee      = "fee"
	TransactionTypeLock     = "lock"
	TransactionTypeUnlock   = "unlock"
	TransactionTypeStake    = "stake"
	TransactionTypeReward   = "reward"
	TransactionTypeLoss     = "loss"
)

const (
	PaymentStatusProviderSession = "PROVIDER_SESSION"
	PaymentStatusPending         = "PENDING"
	PaymentStatusVerified        = "VERIFIED"
	PaymentStatusSettled         = "SETTLED"
	PaymentStatusFailed          = "FAILED"

	WithdrawalStatusPending          = "PENDING"
	WithdrawalStatusAMLReview        = "AML_REVIEW"
	WithdrawalStatusTreasuryApproval = "TREASURY_APPROVAL"
	WithdrawalStatusProviderPending  = "PROVIDER_PENDING"
	WithdrawalStatusSettled          = "SETTLED"
	WithdrawalStatusFailed           = "FAILED"
	WithdrawalStatusRejected         = "REJECTED"
)

type LedgerEntry struct {
	ID              string            `json:"id"`
	UserID          string            `json:"userId"`
	TransactionType string            `json:"transactionType"`
	Amount          float64           `json:"amount"`
	BalanceBefore   float64           `json:"balanceBefore"`
	BalanceAfter    float64           `json:"balanceAfter"`
	Currency        string            `json:"currency"`
	Reference       string            `json:"reference,omitempty"`
	Metadata        map[string]string `json:"metadata,omitempty"`
	CreatedAt       time.Time         `json:"createdAt"`
}

type PaymentProviderSession struct {
	ID             string            `json:"id"`
	UserID         string            `json:"userId"`
	Provider       string            `json:"provider"`
	Method         string            `json:"method"`
	Amount         float64           `json:"amount"`
	Currency       string            `json:"currency"`
	Status         string            `json:"status"`
	ProviderRef    string            `json:"providerRef,omitempty"`
	CheckoutURL    string            `json:"checkoutUrl,omitempty"`
	IdempotencyKey string            `json:"idempotencyKey,omitempty"`
	Metadata       map[string]string `json:"metadata,omitempty"`
	CreatedAt      time.Time         `json:"createdAt"`
	UpdatedAt      time.Time         `json:"updatedAt"`
	SettledAt      *time.Time        `json:"settledAt,omitempty"`
}

type WithdrawalRequest struct {
	ID             string            `json:"id"`
	UserID         string            `json:"userId"`
	Provider       string            `json:"provider"`
	Method         string            `json:"method"`
	Amount         float64           `json:"amount"`
	Fee            float64           `json:"fee"`
	Currency       string            `json:"currency"`
	Status         string            `json:"status"`
	RiskScore      int               `json:"riskScore"`
	AMLCaseID      string            `json:"amlCaseId,omitempty"`
	ProviderRef    string            `json:"providerRef,omitempty"`
	Reference      string            `json:"reference,omitempty"`
	Metadata       map[string]string `json:"metadata,omitempty"`
	RequestedAt    time.Time         `json:"requestedAt"`
	ApprovedAt     *time.Time        `json:"approvedAt,omitempty"`
	SettledAt      *time.Time        `json:"settledAt,omitempty"`
	CompletedAt    *time.Time        `json:"completedAt,omitempty"`
	LastTransition time.Time         `json:"lastTransition"`
}

type AMLReview struct {
	ID          string            `json:"id"`
	UserID      string            `json:"userId"`
	Scope       string            `json:"scope"`
	ScopeID     string            `json:"scopeId"`
	Status      string            `json:"status"`
	RiskScore   int               `json:"riskScore"`
	Reasons     []string          `json:"reasons,omitempty"`
	Country     string            `json:"country,omitempty"`
	EscalatedTo string            `json:"escalatedTo,omitempty"`
	Decision    string            `json:"decision,omitempty"`
	Metadata    map[string]string `json:"metadata,omitempty"`
	CreatedAt   time.Time         `json:"createdAt"`
	UpdatedAt   time.Time         `json:"updatedAt"`
	ResolvedAt  *time.Time        `json:"resolvedAt,omitempty"`
}
