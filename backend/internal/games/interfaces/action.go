package interfaces

import "encoding/json"

type ActionEnvelope struct {
	ActionID             string          `json:"actionId"`
	MatchID              string          `json:"matchId"`
	Kind                 string          `json:"kind"`
	Payload              json.RawMessage `json:"payload"`
	ClientSequence       int64           `json:"clientSequence"`
	ExpectedStateVersion int64           `json:"expectedStateVersion"`
}

type ValidatedAction struct {
	ActionID string          `json:"actionId"`
	Kind     string          `json:"kind"`
	Payload  json.RawMessage `json:"payload"`
}
