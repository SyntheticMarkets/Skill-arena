package registry

import (
	"errors"
	"sort"

	"skill-arena/internal/arena/core"
)

var ErrModuleNotFound = errors.New("game module not found")

type Registry struct {
	modules map[string]core.GameModule
}

func New(modules ...core.GameModule) *Registry {
	r := &Registry{modules: map[string]core.GameModule{}}
	for _, module := range modules {
		if module != nil {
			r.modules[module.ID()] = module
		}
	}
	return r
}

func (r *Registry) Register(module core.GameModule) {
	if r == nil || module == nil {
		return
	}
	r.modules[module.ID()] = module
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
