package security

import (
	"errors"

	"skill-arena/internal/models"
)

var (
	ErrUnauthenticated = errors.New("authenticated user is required")
	ErrForbidden       = errors.New("actor is not authorized for this game session")
)

type Context struct {
	UserID string
	Role   string
}

func FromUser(userID string, role string) (Context, error) {
	if userID == "" {
		return Context{}, ErrUnauthenticated
	}
	return Context{UserID: userID, Role: role}, nil
}

func AuthorizeSession(actor Context, session *models.GameSession) error {
	if actor.UserID == "" {
		return ErrUnauthenticated
	}
	if session == nil || session.UserID != actor.UserID {
		return ErrForbidden
	}
	return nil
}
