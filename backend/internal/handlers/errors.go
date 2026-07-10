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
