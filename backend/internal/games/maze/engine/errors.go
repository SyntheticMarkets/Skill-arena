package engine

import "fmt"

const (
	CodeActionUnsupported = "GAME_ACTION_UNSUPPORTED"
	CodeActionInvalid     = "GAME_ACTION_INVALID"
	CodeStateInvalid      = "GAME_STATE_INVALID"
	CodeStateConflict     = "STATE_VERSION_CONFLICT"
	CodeActionOutOfOrder  = "ACTION_OUT_OF_ORDER"
	CodeMatchMismatch     = "MATCH_CONTEXT_MISMATCH"
	CodeParticipant       = "PARTICIPANT_NOT_FOUND"
	CodeAlreadyComplete   = "MATCH_ALREADY_COMPLETE"
	CodeNotLive           = "MATCH_NOT_LIVE"
	CodeArrowUnknown      = "ARROW_NOT_FOUND"
	CodeArrowRemoved      = "ARROW_ALREADY_REMOVED"
	CodeActionAccepted    = "ACTION_ACCEPTED"
	CodeActionBlocked     = "ACTION_BLOCKED"
	CodeMatchTimedOut     = "MATCH_TIMED_OUT"
)

type Error struct {
	Code    string
	Message string
}

func (e *Error) Error() string {
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

func fail(code, message string) error {
	return &Error{Code: code, Message: message}
}
