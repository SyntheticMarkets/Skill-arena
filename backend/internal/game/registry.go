package game

import "context"

type Metadata struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Version     string   `json:"version"`
	Modes       []string `json:"modes"`
	Enabled     bool     `json:"enabled"`
	RendererKey string   `json:"rendererKey"`
}

type RendererContract struct {
	RendererKey string `json:"rendererKey"`
	Protocol    string `json:"protocol"`
	Version     string `json:"version"`
}

type ReplayContract struct {
	Version           string   `json:"version"`
	RequiredStreams   []string `json:"requiredStreams"`
	IntegrityRequired bool     `json:"integrityRequired"`
}

type TournamentContract struct {
	SupportedFormats []string `json:"supportedFormats"`
	ScoringMode      string   `json:"scoringMode"`
}

type Module interface {
	Metadata() Metadata
	Renderer() RendererContract
	Replay() ReplayContract
	Tournament() TournamentContract
	ValidateConfig(context.Context) error
}

type Registry struct {
	modules map[string]Module
}

func NewRegistry(modules ...Module) *Registry {
	registry := &Registry{modules: map[string]Module{}}
	for _, module := range modules {
		if module != nil {
			registry.modules[module.Metadata().ID] = module
		}
	}
	return registry
}

func (r *Registry) Get(id string) (Module, bool) {
	if r == nil {
		return nil, false
	}
	module, ok := r.modules[id]
	return module, ok
}

func (r *Registry) List() []Metadata {
	if r == nil {
		return []Metadata{}
	}
	items := make([]Metadata, 0, len(r.modules))
	for _, module := range r.modules {
		items = append(items, module.Metadata())
	}
	return items
}

type MazeArenaModule struct {
	Enabled bool
}

func (m MazeArenaModule) Metadata() Metadata {
	return Metadata{ID: "maze_arena", Name: "Maze Arena", Version: GameRulesVersion, Modes: []string{"practice", "ranked", "pvp", "tournament", "house"}, Enabled: m.Enabled, RendererKey: "maze-arena"}
}

func (m MazeArenaModule) Renderer() RendererContract {
	return RendererContract{RendererKey: "maze-arena", Protocol: "skill-arena-renderer", Version: "v1"}
}

func (m MazeArenaModule) Replay() ReplayContract {
	return ReplayContract{Version: ReplayVersion, RequiredStreams: []string{"moves", "clicks", "line_events", "timing"}, IntegrityRequired: true}
}

func (m MazeArenaModule) Tournament() TournamentContract {
	return TournamentContract{SupportedFormats: []string{"single_elimination", "ranked_ladder"}, ScoringMode: "completion_time_valid_route"}
}

func (m MazeArenaModule) ValidateConfig(context.Context) error {
	return nil
}
