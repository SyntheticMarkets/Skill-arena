package models

import "time"

const (
	PermissionDashboardRead       = "dashboard.read"
	PermissionUsersRead           = "users.read"
	PermissionUsersManage         = "users.manage"
	PermissionAdminRolesManage    = "admin_roles.manage"
	PermissionFinanceRead         = "finance.read"
	PermissionWithdrawalsReview   = "withdrawals.review"
	PermissionTreasuryRead        = "treasury.read"
	PermissionReconciliationRun   = "reconciliation.run"
	PermissionKYCRead             = "kyc.read"
	PermissionKYCDecide           = "kyc.decide"
	PermissionComplianceRead      = "compliance.read"
	PermissionComplianceManage    = "compliance.manage"
	PermissionSupportRead         = "support.read"
	PermissionSupportManage       = "support.manage"
	PermissionAuditRead           = "audit.read"
	PermissionNotificationsSend   = "notifications.send"
	PermissionMonitoringRead      = "monitoring.read"
	PermissionAdministratorAccess = "administrator.access"
)

var allAdminPermissions = []string{
	PermissionAdministratorAccess,
	PermissionDashboardRead,
	PermissionUsersRead,
	PermissionUsersManage,
	PermissionAdminRolesManage,
	PermissionFinanceRead,
	PermissionWithdrawalsReview,
	PermissionTreasuryRead,
	PermissionReconciliationRun,
	PermissionKYCRead,
	PermissionKYCDecide,
	PermissionComplianceRead,
	PermissionComplianceManage,
	PermissionSupportRead,
	PermissionSupportManage,
	PermissionAuditRead,
	PermissionNotificationsSend,
	PermissionMonitoringRead,
}

func AdminPermissions(role string) []string {
	var permissions []string
	switch role {
	case RoleSuperAdmin:
		permissions = allAdminPermissions
	case RoleAdmin, RoleOperations:
		permissions = []string{
			PermissionAdministratorAccess, PermissionDashboardRead, PermissionUsersRead,
			PermissionUsersManage, PermissionFinanceRead, PermissionTreasuryRead,
			PermissionKYCRead, PermissionComplianceRead, PermissionSupportRead,
			PermissionSupportManage, PermissionAuditRead, PermissionNotificationsSend,
			PermissionMonitoringRead,
		}
	case RoleTreasuryManager, RoleFinance:
		permissions = []string{
			PermissionAdministratorAccess, PermissionDashboardRead, PermissionUsersRead,
			PermissionFinanceRead, PermissionWithdrawalsReview, PermissionTreasuryRead,
			PermissionReconciliationRun, PermissionAuditRead, PermissionMonitoringRead,
		}
	case RoleFraudAnalyst, RoleCompliance:
		permissions = []string{
			PermissionAdministratorAccess, PermissionDashboardRead, PermissionUsersRead,
			PermissionFinanceRead, PermissionKYCRead, PermissionKYCDecide,
			PermissionComplianceRead, PermissionComplianceManage, PermissionAuditRead,
			PermissionMonitoringRead,
		}
	case RoleSupport:
		permissions = []string{
			PermissionAdministratorAccess, PermissionDashboardRead, PermissionUsersRead,
			PermissionSupportRead, PermissionSupportManage,
		}
	case RoleReadOnly:
		permissions = []string{
			PermissionAdministratorAccess, PermissionDashboardRead, PermissionUsersRead,
			PermissionFinanceRead, PermissionTreasuryRead, PermissionKYCRead,
			PermissionComplianceRead, PermissionSupportRead, PermissionAuditRead,
			PermissionMonitoringRead,
		}
	}
	return append([]string(nil), permissions...)
}

func HasAdminPermission(role, permission string) bool {
	for _, candidate := range AdminPermissions(role) {
		if candidate == permission {
			return true
		}
	}
	return false
}

func IsAdministratorRole(role string) bool {
	return HasAdminPermission(role, PermissionAdministratorAccess)
}

type AdminIdentity struct {
	ID          string   `json:"id"`
	Email       string   `json:"email"`
	DisplayName string   `json:"displayName"`
	Role        string   `json:"role"`
	Permissions []string `json:"permissions"`
}

type CRMDashboard struct {
	GeneratedAt time.Time          `json:"generatedAt"`
	Players     CRMPlayerMetrics   `json:"players"`
	Financial   CRMFinanceMetrics  `json:"financial"`
	Games       CRMGameMetrics     `json:"games"`
	Support     CRMSupportMetrics  `json:"support"`
	Compliance  CRMComplianceStats `json:"compliance"`
	System      CRMSystemSummary   `json:"system"`
}

type CRMPlayerMetrics struct {
	TotalUsers       int `json:"totalUsers"`
	OnlineUsers      int `json:"onlineUsers"`
	NewRegistrations int `json:"newRegistrations"`
	PendingVerify    int `json:"pendingVerification"`
}

type CRMFinanceMetrics struct {
	DepositsTodayMinor     MinorUnits `json:"depositsTodayMinor"`
	PendingWithdrawals     int        `json:"pendingWithdrawals"`
	CompletedWithdrawals   int        `json:"completedWithdrawals"`
	TreasuryAvailableMinor MinorUnits `json:"treasuryAvailableMinor"`
	ActivePaymentProviders int        `json:"activePaymentProviders"`
	Currency               string     `json:"currency"`
}

type CRMGameMetrics struct {
	LiveMatches       int `json:"liveMatches"`
	CompletedMatches  int `json:"completedMatches"`
	ActiveTournaments int `json:"activeTournaments"`
	QueueSize         int `json:"queueSize"`
}

type CRMSupportMetrics struct {
	OpenTickets         int     `json:"openTickets"`
	EscalatedTickets    int     `json:"escalatedTickets"`
	AverageResponseMins float64 `json:"averageResponseMinutes"`
}

type CRMComplianceStats struct {
	PendingKYC       int `json:"pendingKyc"`
	PendingReviews   int `json:"pendingReviews"`
	SelfExclusions   int `json:"selfExclusions"`
	CoolingOffActive int `json:"coolingOffActive"`
}

type CRMSystemSummary struct {
	API      string `json:"api"`
	Database string `json:"database"`
	Redis    string `json:"redis"`
	Storage  string `json:"storage"`
	Queue    string `json:"queue"`
}

type CRMUserRecord struct {
	User               *User                 `json:"user"`
	Progression        *Progression          `json:"progression,omitempty"`
	Wallet             *FinancialWallet      `json:"wallet,omitempty"`
	Assessment         *FinancialAssessment  `json:"assessment,omitempty"`
	Limits             *FinancialLimits      `json:"limits,omitempty"`
	Devices            []*Device             `json:"devices"`
	Sessions           []*AuthSession        `json:"sessions"`
	Deposits           []FinancialDeposit    `json:"deposits"`
	Withdrawals        []FinancialWithdrawal `json:"withdrawals"`
	MatchHistory       []*GameSession        `json:"matchHistory"`
	Statement          *FinancialStatement   `json:"statement,omitempty"`
	InternalNotes      []CRMInternalNote     `json:"internalNotes"`
	ActiveRestrictions []CRMRestriction      `json:"activeRestrictions"`
}

type CRMInternalNote struct {
	ID        string    `json:"id"`
	UserID    string    `json:"userId"`
	AuthorID  string    `json:"authorId"`
	Body      string    `json:"body"`
	CreatedAt time.Time `json:"createdAt"`
}

type CRMRestriction struct {
	ID        string     `json:"id"`
	UserID    string     `json:"userId"`
	Type      string     `json:"type"`
	Reason    string     `json:"reason"`
	Status    string     `json:"status"`
	ExpiresAt *time.Time `json:"expiresAt,omitempty"`
	CreatedBy string     `json:"createdBy"`
	CreatedAt time.Time  `json:"createdAt"`
	UpdatedAt time.Time  `json:"updatedAt"`
}

type CRMSupportTicket struct {
	ID            string              `json:"id"`
	UserID        string              `json:"userId"`
	Category      string              `json:"category"`
	Subject       string              `json:"subject"`
	Status        string              `json:"status"`
	Priority      string              `json:"priority"`
	AssignedTo    string              `json:"assignedTo,omitempty"`
	Escalated     bool                `json:"escalated"`
	CreatedAt     time.Time           `json:"createdAt"`
	UpdatedAt     time.Time           `json:"updatedAt"`
	FirstResponse *time.Time          `json:"firstResponseAt,omitempty"`
	Messages      []CRMSupportMessage `json:"messages"`
	Attachments   []SupportAttachment `json:"attachments"`
}

type CRMSupportMessage struct {
	ID        string    `json:"id"`
	TicketID  string    `json:"ticketId"`
	AuthorID  string    `json:"authorId"`
	Body      string    `json:"body"`
	Internal  bool      `json:"internal"`
	CreatedAt time.Time `json:"createdAt"`
}

type CRMJurisdictionPolicy struct {
	Country                string     `json:"country"`
	Currency               string     `json:"currency"`
	MinimumAge             int        `json:"minimumAge"`
	DepositEnabled         bool       `json:"depositEnabled"`
	WithdrawalEnabled      bool       `json:"withdrawalEnabled"`
	SourceOfFundsRequired  bool       `json:"sourceOfFundsRequired"`
	DailyDepositMinor      MinorUnits `json:"dailyDepositMinor"`
	MonthlyDepositMinor    MinorUnits `json:"monthlyDepositMinor"`
	DailyWithdrawalMinor   MinorUnits `json:"dailyWithdrawalMinor"`
	MonthlyWithdrawalMinor MinorUnits `json:"monthlyWithdrawalMinor"`
	UpdatedBy              string     `json:"updatedBy"`
	UpdatedAt              time.Time  `json:"updatedAt"`
}

type CRMAnnouncement struct {
	ID        string     `json:"id"`
	Category  string     `json:"category"`
	Title     string     `json:"title"`
	Message   string     `json:"message"`
	Audience  string     `json:"audience"`
	Status    string     `json:"status"`
	CreatedBy string     `json:"createdBy"`
	CreatedAt time.Time  `json:"createdAt"`
	SentAt    *time.Time `json:"sentAt,omitempty"`
}

type CRMFinanceWorkspace struct {
	Deposits        []FinancialDeposit     `json:"deposits"`
	Withdrawals     []FinancialWithdrawal  `json:"withdrawals"`
	Reconciliations []CRMReconciliation    `json:"reconciliations"`
	ReserveChecks   []TreasuryReserveCheck `json:"reserveChecks"`
	Providers       []CRMProviderStatus    `json:"providers"`
}

type CRMProviderStatus struct {
	ID      string `json:"id"`
	Status  string `json:"status"`
	Details string `json:"details,omitempty"`
}

type CRMReconciliation struct {
	ID                   string     `json:"id"`
	Currency             string     `json:"currency"`
	Provider             string     `json:"provider"`
	ProviderBalanceMinor MinorUnits `json:"providerBalanceMinor"`
	JournalBalanceMinor  MinorUnits `json:"journalBalanceMinor"`
	DifferenceMinor      MinorUnits `json:"differenceMinor"`
	Status               string     `json:"status"`
	ImmutableHash        string     `json:"immutableHash"`
	CreatedAt            time.Time  `json:"createdAt"`
}

type CRMComplianceCase struct {
	User              *User                           `json:"user"`
	Assessment        *FinancialAssessment            `json:"assessment,omitempty"`
	Evidence          []FinancialEvidence             `json:"evidence"`
	ProviderResponses []CRMComplianceProviderResponse `json:"providerResponses"`
	Reviews           []*ReviewCase                   `json:"reviews"`
}

type CRMComplianceProviderResponse struct {
	ID                string            `json:"id"`
	UserID            string            `json:"userId"`
	Provider          string            `json:"provider"`
	ProviderReference string            `json:"providerReference"`
	CheckType         string            `json:"checkType"`
	Status            string            `json:"status"`
	RiskSignals       []string          `json:"riskSignals"`
	Metadata          map[string]string `json:"metadata,omitempty"`
	ReceivedAt        time.Time         `json:"receivedAt"`
}
