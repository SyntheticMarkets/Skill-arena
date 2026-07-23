package models

import "time"

const (
	NotificationStatusUnread   = "unread"
	NotificationStatusRead     = "read"
	NotificationStatusArchived = "archived"

	TicketStatusOpen     = "open"
	TicketStatusReceived = "received"
	TicketStatusClosed   = "closed"
)

type PlayerProfile struct {
	UserID      string    `json:"userId"`
	Username    string    `json:"username"`
	DisplayName string    `json:"displayName"`
	AvatarURL   string    `json:"avatarUrl,omitempty"`
	Country     string    `json:"country"`
	Language    string    `json:"language"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

type GameCatalogEntry struct {
	ID                 string          `json:"id"`
	Name               string          `json:"name"`
	Description        string          `json:"description"`
	Category           string          `json:"category"`
	Version            string          `json:"version"`
	RendererKey        string          `json:"rendererKey"`
	Modes              []string        `json:"modes"`
	AverageTimeSeconds int             `json:"averageTimeSeconds"`
	Capabilities       CapabilityFlags `json:"capabilities"`
	Availability       string          `json:"availability"`
	AvailabilityReason string          `json:"availabilityReason,omitempty"`
	RulesSummary       []string        `json:"rulesSummary"`
}

type CapabilityFlags struct {
	Practice   bool `json:"practice"`
	PvP        bool `json:"pvp"`
	Replay     bool `json:"replay"`
	Tournament bool `json:"tournament"`
	Spectator  bool `json:"spectator"`
	AI         bool `json:"ai"`
	Teams      bool `json:"teams"`
	Coins      bool `json:"coins"`
}

type Notification struct {
	ID         string            `json:"id"`
	UserID     string            `json:"-"`
	Category   string            `json:"category"`
	Title      string            `json:"title"`
	Message    string            `json:"message"`
	Status     string            `json:"status"`
	ActionURL  string            `json:"actionUrl,omitempty"`
	Metadata   map[string]string `json:"metadata,omitempty"`
	CreatedAt  time.Time         `json:"createdAt"`
	ReadAt     *time.Time        `json:"readAt,omitempty"`
	ArchivedAt *time.Time        `json:"archivedAt,omitempty"`
}

type NotificationSummary struct {
	Unread int `json:"unread"`
	Total  int `json:"total"`
}

type DailyObjective struct {
	ID          string `json:"id"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Progress    int    `json:"progress"`
	Target      int    `json:"target"`
	Complete    bool   `json:"complete"`
	ActionURL   string `json:"actionUrl"`
}

type HubActivity struct {
	ID          string    `json:"id"`
	Type        string    `json:"type"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	ActionURL   string    `json:"actionUrl,omitempty"`
	OccurredAt  time.Time `json:"occurredAt"`
}

type HubAction struct {
	ID          string `json:"id"`
	Label       string `json:"label"`
	Description string `json:"description"`
	ActionURL   string `json:"actionUrl"`
	Reason      string `json:"reason"`
}

type HubEligibility struct {
	EmailVerified   bool     `json:"emailVerified"`
	ProfileComplete bool     `json:"profileComplete"`
	MFAEnabled      bool     `json:"mfaEnabled"`
	WalletVisible   bool     `json:"walletVisible"`
	LiveEligible    bool     `json:"liveEligible"`
	Blockers        []string `json:"blockers"`
}

type HubWalletSummary struct {
	Currency           string  `json:"currency"`
	AvailableBalance   float64 `json:"availableBalance"`
	PendingDeposits    float64 `json:"pendingDeposits"`
	PendingWithdrawals float64 `json:"pendingWithdrawals"`
	AccountStatus      string  `json:"accountStatus"`
	VerificationStatus string  `json:"verificationStatus"`
}

type HubTournament struct {
	ID               string    `json:"id"`
	Name             string    `json:"name"`
	Status           string    `json:"status"`
	StartsAt         time.Time `json:"startsAt"`
	Eligible         bool      `json:"eligible"`
	IneligibleReason string    `json:"ineligibleReason,omitempty"`
}

type HubChallenge struct {
	ID        string `json:"id"`
	Type      string `json:"type"`
	Title     string `json:"title"`
	Status    string `json:"status"`
	Reason    string `json:"reason,omitempty"`
	ActionURL string `json:"actionUrl,omitempty"`
}

type HubSnapshot struct {
	GeneratedAt       time.Time           `json:"generatedAt"`
	Profile           PlayerProfile       `json:"profile"`
	Progression       Progression         `json:"progression"`
	Wallet            HubWalletSummary    `json:"wallet"`
	Notifications     NotificationSummary `json:"notifications"`
	Objectives        []DailyObjective    `json:"objectives"`
	RecommendedAction HubAction           `json:"recommendedAction"`
	ContinueActivity  *HubActivity        `json:"continueActivity,omitempty"`
	RecentActivity    []HubActivity       `json:"recentActivity"`
	Tournaments       []HubTournament     `json:"tournaments"`
	Challenges        []HubChallenge      `json:"challenges"`
	Games             []GameCatalogEntry  `json:"games"`
	Eligibility       HubEligibility      `json:"eligibility"`
}

type SupportTicket struct {
	ID        string    `json:"id"`
	UserID    string    `json:"-"`
	Category  string    `json:"category"`
	Subject   string    `json:"subject"`
	Message   string    `json:"message"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

type SupportArticle struct {
	ID       string `json:"id"`
	Category string `json:"category"`
	Title    string `json:"title"`
	Body     string `json:"body"`
}

type SupportContent struct {
	Articles     []SupportArticle `json:"articles"`
	ContactEmail string           `json:"contactEmail"`
}
