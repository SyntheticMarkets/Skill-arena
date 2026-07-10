package handlers

import (
	"net/http"
	"strings"
)

func WriteMappedError(w http.ResponseWriter, status int, err error) {
	if err == nil {
		WriteAPIError(w, status, ErrInternal, "unknown error")
		return
	}
	message := err.Error()
	lower := strings.ToLower(message)
	code := ErrInvalidRequest
	switch {
	case strings.Contains(lower, "not found") && strings.Contains(lower, "match"):
		code = ErrMatchNotFound
	case strings.Contains(lower, "session not found"):
		code = ErrSessionNotFound
	case strings.Contains(lower, "trust"):
		code = ErrTrustTooLow
	case strings.Contains(lower, "house"):
		code = ErrHouseLocked
	case status >= 500:
		code = ErrInternal
	}
	WriteAPIError(w, status, code, message)
}
