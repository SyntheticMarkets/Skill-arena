package handlers

import (
	"encoding/json"
	"net/http"
)

const (
	ErrInvalidRequest  = "INVALID_REQUEST"
	ErrUnauthorized    = "AUTH_INVALID_TOKEN"
	ErrForbidden       = "FORBIDDEN"
	ErrRateLimited     = "RATE_LIMITED"
	ErrMatchNotFound   = "MATCH_NOT_FOUND"
	ErrSessionNotFound = "SESSION_NOT_FOUND"
	ErrTrustTooLow     = "TRUST_TOO_LOW"
	ErrHouseLocked     = "HOUSE_LOCKED"
	ErrMaintenance     = "MAINTENANCE_MODE"
	ErrInternal        = "INTERNAL_ERROR"
	ErrConflict        = "CONFLICT"
	ErrNotFound        = "NOT_FOUND"
	ErrEmailUnverified = "AUTH_EMAIL_UNVERIFIED"
	ErrMFARequired     = "AUTH_MFA_REQUIRED"
	ErrAccountLocked   = "AUTH_ACCOUNT_LOCKED"
	ErrPasswordPolicy  = "AUTH_PASSWORD_POLICY"
	ErrTokenExpired    = "AUTH_TOKEN_EXPIRED"
	ErrTokenUsed       = "AUTH_TOKEN_USED"
)

type APIError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func WriteAPIError(w http.ResponseWriter, status int, code string, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(APIError{Code: code, Message: message})
}
