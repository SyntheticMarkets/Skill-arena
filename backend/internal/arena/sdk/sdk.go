package sdk

import (
	"time"

	"skill-arena/internal/arena/core"
)

type Context struct {
	GameID    string
	SessionID string
	UserID    string
	events    []core.Event
}

func NewContext(gameID string, sessionID string, userID string) *Context {
	return &Context{GameID: gameID, SessionID: sessionID, UserID: userID, events: []core.Event{}}
}

func (c *Context) Emit(eventType string, payload map[string]string) {
	if c == nil {
		return
	}
	c.events = append(c.events, core.Event{
		Type:      eventType,
		GameID:    c.GameID,
		SessionID: c.SessionID,
		UserID:    c.UserID,
		Payload:   payload,
		CreatedAt: time.Now().UTC(),
	})
}

func (c *Context) Events() []core.Event {
	if c == nil {
		return []core.Event{}
	}
	copied := make([]core.Event, len(c.events))
	copy(copied, c.events)
	return copied
}
