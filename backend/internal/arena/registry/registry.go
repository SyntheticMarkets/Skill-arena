package registry

import (
	"errors"
	"sort"

	"skill-arena/internal/arena/core"
)

var (
	ErrModuleNotFound  = errors.New("game module not found")
	ErrInvalidManifest = errors.New("invalid game module manifest")
)

type Registry struct {
	modules map[string]core.GameModule
}

func New(modules ...core.GameModule) *Registry {
	r := &Registry{modules: map[string]core.GameModule{}}
	for _, module := range modules {
		if module != nil {
			_ = r.Register(module)
		}
	}
	return r
}

func (r *Registry) Register(module core.GameModule) error {
	if r == nil || module == nil {
		return ErrModuleNotFound
	}
	if err := ValidateManifest(module.Manifest()); err != nil {
		return err
	}
	r.modules[module.ID()] = module
	return nil
}

func (r *Registry) Get(id string) (core.GameModule, error) {
	if r == nil {
		return nil, ErrModuleNotFound
	}
	module, ok := r.modules[id]
	if !ok {
		return nil, ErrModuleNotFound
	}
	return module, nil
}

func ValidateManifest(manifest core.Manifest) error {
	if manifest.ID == "" || manifest.Name == "" || manifest.Version == "" {
		return ErrInvalidManifest
	}
	if manifest.Versions.Rules == "" || manifest.Versions.Replay == "" || manifest.Versions.Protocol == "" {
		return ErrInvalidManifest
	}
	if manifest.MinimumPlayers < 1 || manifest.MaximumPlayers < manifest.MinimumPlayers {
		return ErrInvalidManifest
	}
	return nil
}

func (r *Registry) List() []core.Metadata {
	if r == nil {
		return []core.Metadata{}
	}
	items := make([]core.Metadata, 0, len(r.modules))
	for _, module := range r.modules {
		items = append(items, module.Metadata())
	}
	sort.SliceStable(items, func(i, j int) bool {
		return items[i].ID < items[j].ID
	})
	return items
}
