package interfaces

import "encoding/json"

type RendererSnapshot struct {
	RendererVersion string          `json:"rendererVersion"`
	StateVersion    int64           `json:"stateVersion"`
	Payload         json.RawMessage `json:"payload"`
	Checksum        string          `json:"checksum"`
}
