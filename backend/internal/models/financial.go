package models

import "time"

type MinorUnits int64

const (
	DepositStatusRequested           = "requested"
	DepositStatusPendingProvider     = "pending_provider"
	DepositStatusPendingVerification = "pending_verification"
	DepositStatusCompleted           = "completed"
	DepositStatusFailed              = "failed"
	DepositStatusExpired             = "expired"

	FinancialWithdrawalStatusRequested     = "requested"
	FinancialWithdrawalStatusPendingReview = "pending_review"
	FinancialWithdrawalStatusApproved      = "approved"
	FinancialWithdrawalStatusProcessing    = "processing"
	FinancialWithdrawalStatusCompleted     = "completed"
	FinancialWithdrawalStatusRejected      = "rejected"
	FinancialWithdrawalStatusFailed        = "failed"

	AssessmentStatusNotStarted = "not_started"
	AssessmentStatusSubmitted  = "submitted"
	AssessmentStatusInReview   = "in_review"
	AssessmentStatusComplete   = "complete"
	AssessmentStatusRestricted = "restricted"
)

type FinancialWallet struct {
	UserID                 string     `json:"-"`
	Currency               string     `json:"currency"`
	AvailableMinor         MinorUnits `json:"availableMinor"`
	PendingDepositMinor    MinorUnits `json:"pendingDepositMinor"`
	PendingWithdrawalMinor MinorUnits `json:"pendingWithdrawalMinor"`
	LockedMinor            MinorUnits `json:"lockedMinor"`
	LifetimeDepositMinor   MinorUnits `json:"lifetimeDepositMinor"`
	LifetimeWithdrawMinor  MinorUnits `json:"lifetimeWithdrawalMinor"`
	Version                int64      `json:"version"`
	UpdatedAt              time.Time  `json:"updatedAt"`
}

type FinancialLedgerEntry struct {
	ID                string            `json:"id"`
	UserID            string            `json:"-"`
	Account           string            `json:"account"`
	Direction         string            `json:"direction"`
	AmountMinor       MinorUnits        `json:"amountMinor"`
	Currency          string            `json:"currency"`
	BalanceAfterMinor MinorUnits        `json:"balanceAfterMinor"`
	ReferenceType     string            `json:"referenceType"`
	ReferenceID       string            `json:"referenceId"`
	Description       string            `json:"description"`
	Sequence          int64             `json:"sequence"`
	PreviousHash      string            `json:"previousHash"`
	EntryHash         string            `json:"entryHash"`
	Metadata          map[string]string `json:"-"`
	CreatedAt         time.Time         `json:"createdAt"`
}

type FinancialDeposit struct {
	ID                string            `json:"id"`
	UserID            string            `json:"-"`
	AmountMinor       MinorUnits        `json:"amountMinor"`
	Currency          string            `json:"currency"`
	Method            string            `json:"method"`
	Provider          string            `json:"-"`
	ProviderReference string            `json:"-"`
	CheckoutURL       string            `json:"checkoutUrl,omitempty"`
	Status            string            `json:"status"`
	IdempotencyKey    string            `json:"-"`
	RequestHash       string            `json:"-"`
	Metadata          map[string]string `json:"-"`
	RequestedAt       time.Time         `json:"requestedAt"`
	UpdatedAt         time.Time         `json:"updatedAt"`
	CompletedAt       *time.Time        `json:"completedAt,omitempty"`
}

type FinancialWithdrawal struct {
	ID                string            `json:"id"`
	UserID            string            `json:"-"`
	AmountMinor       MinorUnits        `json:"amountMinor"`
	FeeMinor          MinorUnits        `json:"feeMinor"`
	Currency          string            `json:"currency"`
	Method            string            `json:"method"`
	Provider          string            `json:"-"`
	ProviderReference string            `json:"-"`
	Status            string            `json:"status"`
	PolicyDecision    string            `json:"-"`
	PolicyReasons     []string          `json:"-"`
	IdempotencyKey    string            `json:"-"`
	RequestHash       string            `json:"-"`
	Metadata          map[string]string `json:"-"`
	RequestedAt       time.Time         `json:"requestedAt"`
	UpdatedAt         time.Time         `json:"updatedAt"`
	CompletedAt       *time.Time        `json:"completedAt,omitempty"`
}

type FinancialAssessment struct {
	UserID             string     `json:"-"`
	Status             string     `json:"status"`
	Country            string     `json:"country"`
	Occupation         string     `json:"occupation"`
	SourceOfFunds      string     `json:"sourceOfFunds"`
	RiskClassification string     `json:"riskClassification"`
	VerificationStatus string     `json:"verificationStatus"`
	ResponsibleStatus  string     `json:"responsibleGamingStatus"`
	SubmittedAt        *time.Time `json:"submittedAt,omitempty"`
	UpdatedAt          time.Time  `json:"updatedAt"`
}

type FinancialLimits struct {
	UserID                 string     `json:"-"`
	Currency               string     `json:"currency"`
	DailyDepositMinor      MinorUnits `json:"dailyDepositMinor"`
	MonthlyDepositMinor    MinorUnits `json:"monthlyDepositMinor"`
	DailyWithdrawalMinor   MinorUnits `json:"dailyWithdrawalMinor"`
	MonthlyWithdrawalMinor MinorUnits `json:"monthlyWithdrawalMinor"`
	DepositUsedTodayMinor  MinorUnits `json:"depositUsedTodayMinor"`
	DepositUsedMonthMinor  MinorUnits `json:"depositUsedMonthMinor"`
	WithdrawUsedTodayMinor MinorUnits `json:"withdrawalUsedTodayMinor"`
	WithdrawUsedMonthMinor MinorUnits `json:"withdrawalUsedMonthMinor"`
	CoolingOffUntil        *time.Time `json:"coolingOffUntil,omitempty"`
	SelfExcludedUntil      *time.Time `json:"selfExcludedUntil,omitempty"`
	UpdatedAt              time.Time  `json:"updatedAt"`
}

type PaymentMethod struct {
	ID          string `json:"id"`
	Type        string `json:"type"`
	DisplayName string `json:"displayName"`
	Currency    string `json:"currency"`
	Available   bool   `json:"available"`
	Reason      string `json:"reason,omitempty"`
}

type FinancialTimelineEvent struct {
	Status      string     `json:"status"`
	Label       string     `json:"label"`
	Description string     `json:"description"`
	OccurredAt  *time.Time `json:"occurredAt,omitempty"`
	Complete    bool       `json:"complete"`
}

type FinancialOverview struct {
	Wallet             FinancialWallet       `json:"wallet"`
	Assessment         FinancialAssessment   `json:"assessment"`
	Limits             FinancialLimits       `json:"limits"`
	VerificationStatus string                `json:"verificationStatus"`
	PaymentMethods     []PaymentMethod       `json:"paymentMethods"`
	Deposits           []FinancialDeposit    `json:"deposits"`
	Withdrawals        []FinancialWithdrawal `json:"withdrawals"`
}

type FinancialStatement struct {
	ID               string                 `json:"id"`
	PeriodStart      time.Time              `json:"periodStart"`
	PeriodEnd        time.Time              `json:"periodEnd"`
	Currency         string                 `json:"currency"`
	OpeningMinor     MinorUnits             `json:"openingMinor"`
	ClosingMinor     MinorUnits             `json:"closingMinor"`
	TotalCreditMinor MinorUnits             `json:"totalCreditMinor"`
	TotalDebitMinor  MinorUnits             `json:"totalDebitMinor"`
	Entries          []FinancialLedgerEntry `json:"entries"`
	GeneratedAt      time.Time              `json:"generatedAt"`
}

type FinancialEvidence struct {
	ID          string    `json:"id"`
	UserID      string    `json:"-"`
	Type        string    `json:"type"`
	ObjectKey   string    `json:"-"`
	ContentType string    `json:"contentType"`
	SizeBytes   int64     `json:"sizeBytes"`
	SHA256      string    `json:"sha256"`
	Status      string    `json:"status"`
	CreatedAt   time.Time `json:"createdAt"`
}

type FinancialArtifact struct {
	ID          string    `json:"id"`
	UserID      string    `json:"-"`
	Type        string    `json:"type"`
	ObjectKey   string    `json:"-"`
	ContentType string    `json:"contentType"`
	SizeBytes   int64     `json:"sizeBytes"`
	SHA256      string    `json:"sha256"`
	CreatedAt   time.Time `json:"createdAt"`
}

type FinancialPayoutDestination struct {
	UserID            string    `json:"userId"`
	Provider          string    `json:"-"`
	ProviderAccountID string    `json:"-"`
	Status            string    `json:"status"`
	EvidenceID        string    `json:"evidenceId,omitempty"`
	UpdatedAt         time.Time `json:"updatedAt"`
}

type TreasuryReserveCheck struct {
	ID                     string     `json:"id"`
	Provider               string     `json:"provider"`
	Currency               string     `json:"currency"`
	ProviderAvailableMinor MinorUnits `json:"providerAvailableMinor"`
	ProviderPendingMinor   MinorUnits `json:"providerPendingMinor"`
	LiabilityMinor         MinorUnits `json:"liabilityMinor"`
	RequestedMinor         MinorUnits `json:"requestedMinor"`
	Purpose                string     `json:"purpose"`
	Passed                 bool       `json:"passed"`
	ImmutableHash          string     `json:"immutableHash"`
	ArtifactKey            string     `json:"-"`
	CreatedAt              time.Time  `json:"createdAt"`
}
