package interfaces

import "strings"

// Versions pins every contract needed to execute or restore a game module.
type Versions struct {
	Game        string `json:"game"`
	Rules       string `json:"rules"`
	Protocol    string `json:"protocol"`
	Replay      string `json:"replay"`
	Renderer    string `json:"renderer"`
	StateSchema string `json:"stateSchema"`
}

func (v Versions) Complete() bool {
	return strings.TrimSpace(v.Game) != "" &&
		strings.TrimSpace(v.Rules) != "" &&
		strings.TrimSpace(v.Protocol) != "" &&
		strings.TrimSpace(v.Replay) != "" &&
		strings.TrimSpace(v.Renderer) != "" &&
		strings.TrimSpace(v.StateSchema) != ""
}

func (v Versions) Key() string {
	return strings.Join([]string{
		v.Game,
		v.Rules,
		v.Protocol,
		v.Replay,
		v.Renderer,
		v.StateSchema,
	}, "\x1f")
}
