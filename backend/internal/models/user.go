package models

import "time"

const (
	RoleSuperAdmin      = "super_admin"
	RoleAdmin           = "admin"
	RoleTreasuryManager = "treasury_manager"
	RoleFraudAnalyst    = "fraud_analyst"
	RoleSupport         = "support"
	RoleModerator       = "moderator"
	RolePlayer          = "player"
)

type User struct {
	ID               string     `json:"id"`
	Email            string     `json:"email"`
	Country          string     `json:"country"`
	DateOfBirth      *time.Time `json:"dateOfBirth,omitempty"`
	TermsAcceptedAt  *time.Time `json:"termsAcceptedAt,omitempty"`
	Username         string     `json:"username"`
	DisplayName      string     `json:"displayName"`
	HiddenFromPublic bool       `json:"hiddenFromPublic"`
	PasswordHash     string     `json:"-"`
	Role             string     `json:"role"`
	EmailVerified    bool       `json:"emailVerified"`
	KYCStatus        string     `json:"kycStatus"`
	Status           string     `json:"status"`
	CreatedAt        time.Time  `json:"createdAt"`
	UpdatedAt        time.Time  `json:"updatedAt"`
}

func RoleRank(role string) int {
	switch role {
	case RoleSuperAdmin:
		return 100
	case RoleAdmin:
		return 90
	case RoleTreasuryManager:
		return 70
	case RoleFraudAnalyst:
		return 60
	case RoleSupport:
		return 50
	case RoleModerator:
		return 40
	default:
		return 10
	}
}

func NewUser(id, email, passwordHash string) *User {
	return &User{
		ID:            id,
		Email:         email,
		PasswordHash:  passwordHash,
		Role:          RolePlayer,
		EmailVerified: false,
		KYCStatus:     "unverified",
		Status:        "active",
		CreatedAt:     time.Now().UTC(),
		UpdatedAt:     time.Now().UTC(),
	}
}
