package models

import "time"

const (
	AuthTokenPurposeEmailVerification = "email_verification"
	AuthTokenPurposePasswordReset     = "password_reset"
	AuthTokenPurposeMFAChallenge      = "mfa_challenge"
)

type AuthSession struct {
	ID               string     `json:"id"`
	UserID           string     `json:"userId"`
	RefreshTokenHash string     `json:"refreshTokenHash"`
	UserAgent        string     `json:"userAgent,omitempty"`
	IPAddress        string     `json:"ipAddress,omitempty"`
	CreatedAt        time.Time  `json:"createdAt"`
	ExpiresAt        time.Time  `json:"expiresAt"`
	RevokedAt        *time.Time `json:"revokedAt,omitempty"`
	RotatedAt        *time.Time `json:"rotatedAt,omitempty"`
}

type AuditLog struct {
	ID        string            `json:"id"`
	ActorID   string            `json:"actorId"`
	Action    string            `json:"action"`
	TargetID  string            `json:"targetId,omitempty"`
	IPAddress string            `json:"ipAddress,omitempty"`
	Metadata  map[string]string `json:"metadata,omitempty"`
	CreatedAt time.Time         `json:"createdAt"`
}

type AuthToken struct {
	ID        string     `json:"id"`
	UserID    string     `json:"userId"`
	Purpose   string     `json:"purpose"`
	TokenHash string     `json:"tokenHash"`
	ExpiresAt time.Time  `json:"expiresAt"`
	UsedAt    *time.Time `json:"usedAt,omitempty"`
	CreatedAt time.Time  `json:"createdAt"`
	CreatedIP string     `json:"createdIp,omitempty"`
	UsedIP    string     `json:"usedIp,omitempty"`
}

type MFASettings struct {
	UserID               string    `json:"userId"`
	Enabled              bool      `json:"enabled"`
	TOTPSecretCiphertext string    `json:"totpSecretCiphertext,omitempty"`
	RecoveryCodeHashes   []string  `json:"recoveryCodeHashes,omitempty"`
	ConfirmedAt          time.Time `json:"confirmedAt,omitempty"`
	UpdatedAt            time.Time `json:"updatedAt"`
}

type PasswordHistoryEntry struct {
	UserID        string    `json:"userId"`
	PasswordHash  string    `json:"passwordHash"`
	PasswordStamp string    `json:"passwordStamp"`
	CreatedAt     time.Time `json:"createdAt"`
}

type LoginSecurityState struct {
	UserID         string     `json:"userId"`
	FailedCount    int        `json:"failedCount"`
	LockedUntil    *time.Time `json:"lockedUntil,omitempty"`
	LastFailedAt   *time.Time `json:"lastFailedAt,omitempty"`
	LastSuccessAt  *time.Time `json:"lastSuccessAt,omitempty"`
	LastIPAddress  string     `json:"lastIpAddress,omitempty"`
	LastUserAgent  string     `json:"lastUserAgent,omitempty"`
	SuspiciousFlag string     `json:"suspiciousFlag,omitempty"`
	UpdatedAt      time.Time  `json:"updatedAt"`
}
