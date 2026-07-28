package interfaces

import "time"

type MatchContext struct {
	MatchID        string
	GameID         string
	Mode           string
	Region         string
	ParticipantIDs []string
	Versions       Versions
	ServerTime     time.Time
}

type ParticipantContext struct {
	MatchID       string
	ParticipantID string
	UserID        string
	ViewerRole    string
	Reconnecting  bool
}

type ActionContext struct {
	MatchID             string
	ParticipantID       string
	UserID              string
	ServerReceivedAt    time.Time
	Latency             time.Duration
	CurrentSequence     int64
	CurrentStateVersion int64
}

type ViewerContext struct {
	MatchID       string
	UserID        string
	ParticipantID string
	Role          string
}
