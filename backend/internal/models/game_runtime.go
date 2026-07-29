package models

import (
	"encoding/json"
	"time"
)

type GameParticipantState struct {
	MatchID            string          `json:"matchId"`
	UserID             string          `json:"userId"`
	GameID             string          `json:"gameId"`
	PuzzleID           string          `json:"puzzleId"`
	StateSchema        string          `json:"stateSchemaVersion"`
	StateVersion       int64           `json:"stateVersion"`
	State              json.RawMessage `json:"state"`
	StateChecksum      string          `json:"stateChecksum"`
	LastClientSequence int64           `json:"lastClientSequence"`
	LastServerSequence int64           `json:"lastServerSequence"`
	Status             string          `json:"status"`
	UpdatedAt          time.Time       `json:"updatedAt"`
}

type GameActionReceipt struct {
	ActionID             string          `json:"actionId"`
	MatchID              string          `json:"matchId"`
	UserID               string          `json:"userId"`
	ClientSequence       int64           `json:"clientSequence"`
	ExpectedStateVersion int64           `json:"expectedStateVersion"`
	ActionKind           string          `json:"actionKind"`
	ActionPayloadHash    string          `json:"actionPayloadHash"`
	Accepted             bool            `json:"accepted"`
	ResultCode           string          `json:"resultCode"`
	StateVersionBefore   int64           `json:"stateVersionBefore"`
	StateVersionAfter    int64           `json:"stateVersionAfter"`
	FirstEventSequence   int64           `json:"firstEventSequence"`
	LastEventSequence    int64           `json:"lastEventSequence"`
	Transition           json.RawMessage `json:"transition"`
	ReceiptHash          string          `json:"receiptHash"`
	ServerReceivedAt     time.Time       `json:"serverReceivedAt"`
	ProcessedAt          time.Time       `json:"processedAt"`
}

type GameEventDraft struct {
	Type    string
	Payload json.RawMessage
}
