package models

import (
	"encoding/json"
	"time"
)

const (
	MatchCreated      = "created"
	MatchWaiting      = "waiting_for_players"
	MatchReady        = "ready"
	MatchStarting     = "starting"
	MatchLive         = "live"
	MatchPaused       = "paused"
	MatchReconnecting = "reconnecting"
	MatchCompleted    = "completed"
	MatchCancelled    = "cancelled"
	MatchAbandoned    = "abandoned"

	PresenceOnline       = "online"
	PresenceOffline      = "offline"
	PresenceInMatch      = "in_match"
	PresenceInQueue      = "in_queue"
	PresenceIdle         = "idle"
	PresenceDisconnected = "disconnected"
	PresenceReconnecting = "reconnecting"

	QueueWaiting   = "waiting"
	QueueMatched   = "matched"
	QueueCancelled = "cancelled"
	QueueExpired   = "expired"
)

type RealtimeMatch struct {
	ID              string                `json:"id"`
	GameID          string                `json:"gameId"`
	GameVersion     string                `json:"gameVersion"`
	RulesVersion    string                `json:"rulesVersion"`
	ProtocolVersion string                `json:"protocolVersion"`
	ReplayVersion   string                `json:"replayVersion"`
	Mode            string                `json:"mode"`
	Status          string                `json:"status"`
	Region          string                `json:"region"`
	WalletCategory  string                `json:"walletCategory"`
	SeedReference   string                `json:"seedReference,omitempty"`
	StateVersion    int64                 `json:"stateVersion"`
	Sequence        int64                 `json:"sequence"`
	Participants    []RealtimeParticipant `json:"participants"`
	CreatedAt       time.Time             `json:"createdAt"`
	UpdatedAt       time.Time             `json:"updatedAt"`
	StartedAt       *time.Time            `json:"startedAt,omitempty"`
	CompletedAt     *time.Time            `json:"completedAt,omitempty"`
}

type RealtimeParticipant struct {
	MatchID      string     `json:"matchId"`
	UserID       string     `json:"userId"`
	Status       string     `json:"status"`
	Ready        bool       `json:"ready"`
	Rating       int        `json:"rating"`
	Region       string     `json:"region"`
	LatencyMS    int        `json:"latencyMs"`
	LastSequence int64      `json:"lastSequence"`
	JoinedAt     time.Time  `json:"joinedAt"`
	LastSeenAt   time.Time  `json:"lastSeenAt"`
	LeftAt       *time.Time `json:"leftAt,omitempty"`
}

type RealtimeQueueEntry struct {
	ID             string    `json:"id"`
	UserID         string    `json:"userId"`
	GameID         string    `json:"gameId"`
	Mode           string    `json:"mode"`
	WalletCategory string    `json:"walletCategory"`
	Region         string    `json:"region"`
	Jurisdiction   string    `json:"jurisdiction"`
	Rating         int       `json:"rating"`
	LatencyMS      int       `json:"latencyMs"`
	Priority       int       `json:"priority"`
	Status         string    `json:"status"`
	MatchID        string    `json:"matchId,omitempty"`
	CreatedAt      time.Time `json:"createdAt"`
	ExpiresAt      time.Time `json:"expiresAt"`
	UpdatedAt      time.Time `json:"updatedAt"`
}

type PresenceRecord struct {
	UserID        string    `json:"userId"`
	State         string    `json:"state"`
	SessionID     string    `json:"sessionId,omitempty"`
	ConnectionID  string    `json:"connectionId,omitempty"`
	MatchID       string    `json:"matchId,omitempty"`
	Region        string    `json:"region,omitempty"`
	LastHeartbeat time.Time `json:"lastHeartbeat"`
	ExpiresAt     time.Time `json:"expiresAt"`
}

type RealtimeEvent struct {
	ID            string          `json:"id"`
	MatchID       string          `json:"matchId"`
	UserID        string          `json:"userId,omitempty"`
	Type          string          `json:"type"`
	Sequence      int64           `json:"sequence"`
	StateVersion  int64           `json:"stateVersion"`
	ServerTime    time.Time       `json:"serverTime"`
	Payload       json.RawMessage `json:"payload,omitempty"`
	PreviousHash  string          `json:"previousHash,omitempty"`
	IntegrityHash string          `json:"integrityHash"`
}

type RealtimeSnapshot struct {
	MatchID   string          `json:"matchId"`
	Version   int64           `json:"version"`
	Sequence  int64           `json:"sequence"`
	State     json.RawMessage `json:"state"`
	Checksum  string          `json:"checksum"`
	CreatedAt time.Time       `json:"createdAt"`
}

type RealtimeReplay struct {
	ID              string    `json:"id"`
	MatchID         string    `json:"matchId"`
	GameID          string    `json:"gameId"`
	GameVersion     string    `json:"gameVersion"`
	RulesVersion    string    `json:"rulesVersion"`
	ProtocolVersion string    `json:"protocolVersion"`
	ReplayVersion   string    `json:"replayVersion"`
	FirstSequence   int64     `json:"firstSequence"`
	LastSequence    int64     `json:"lastSequence"`
	EventCount      int       `json:"eventCount"`
	EventRootHash   string    `json:"eventRootHash"`
	Signature       string    `json:"signature"`
	StorageKey      string    `json:"storageKey,omitempty"`
	Status          string    `json:"status"`
	CreatedAt       time.Time `json:"createdAt"`
}

type RealtimeMetrics struct {
	Connections       int       `json:"connections"`
	OnlinePlayers     int       `json:"onlinePlayers"`
	QueuedPlayers     int       `json:"queuedPlayers"`
	ActiveMatches     int       `json:"activeMatches"`
	Reconnects        int64     `json:"reconnects"`
	MatchesCreated    int64     `json:"matchesCreated"`
	MatchErrors       int64     `json:"matchErrors"`
	ReplayBacklog     int       `json:"replayBacklog"`
	GatewayLatencyMS  float64   `json:"gatewayLatencyMs"`
	OldestQueueSecond int64     `json:"oldestQueueSeconds"`
	CheckedAt         time.Time `json:"checkedAt"`
}
