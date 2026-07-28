package registry

import (
	"errors"
	"fmt"
	"reflect"
	"sort"
	"sync"

	"skill-arena/internal/games/interfaces"
)

var (
	ErrModuleNotFound       = errors.New("game module not found")
	ErrVersionUnavailable   = errors.New("game module version unavailable")
	ErrVersionRevoked       = errors.New("game module version revoked")
	ErrInvalidDescriptor    = errors.New("invalid game module descriptor")
	ErrRegistrationConflict = errors.New("game module registration conflict")
	ErrFactoryMismatch      = errors.New("game module factory descriptor mismatch")
	ErrFactoryPanic         = errors.New("game module factory panicked")
	ErrNewMatchesDisabled   = errors.New("game module does not allow new matches")
)

type Factory func() (interfaces.Module, error)

type Registration struct {
	Descriptor interfaces.Descriptor
	Factory    Factory
}

type moduleKey struct {
	gameID      string
	gameVersion string
}

// Registry stores immutable module factories and exact version descriptors.
type Registry struct {
	mu            sync.RWMutex
	registrations map[moduleKey]Registration
	active        map[string]moduleKey
}

func New(registrations ...Registration) (*Registry, error) {
	registry := &Registry{
		registrations: make(map[moduleKey]Registration),
		active:        make(map[string]moduleKey),
	}
	for _, registration := range registrations {
		if err := registry.Register(registration); err != nil {
			return nil, err
		}
	}
	return registry, nil
}

func (r *Registry) Register(registration Registration) error {
	if r == nil {
		return ErrInvalidDescriptor
	}
	descriptor := registration.Descriptor.Clone()
	if err := ValidateDescriptor(descriptor); err != nil {
		return err
	}
	if registration.Factory == nil {
		return fmt.Errorf("%w: factory is required", ErrInvalidDescriptor)
	}
	module, err := instantiate(registration.Factory)
	if err != nil {
		return fmt.Errorf("create game module %s %s: %w", descriptor.ID, descriptor.Versions.Game, err)
	}
	if module == nil || !sameDescriptorIdentity(descriptor, module.Descriptor()) {
		return ErrFactoryMismatch
	}

	key := moduleKey{gameID: descriptor.ID, gameVersion: descriptor.Versions.Game}
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, exists := r.registrations[key]; exists {
		return fmt.Errorf("%w: %s %s", ErrRegistrationConflict, descriptor.ID, descriptor.Versions.Game)
	}
	if descriptor.Status == interfaces.ModuleActive && descriptor.NewMatchAllowed {
		if existing, exists := r.active[descriptor.ID]; exists {
			return fmt.Errorf("%w: active versions %s and %s for %s", ErrRegistrationConflict, existing.gameVersion, descriptor.Versions.Game, descriptor.ID)
		}
		r.active[descriptor.ID] = key
	}
	registration.Descriptor = descriptor
	r.registrations[key] = registration
	return nil
}

// Resolve returns only the exact requested historical version.
func (r *Registry) Resolve(gameID, gameVersion string) (interfaces.Module, error) {
	if r == nil {
		return nil, ErrModuleNotFound
	}
	r.mu.RLock()
	registration, exists := r.registrations[moduleKey{gameID: gameID, gameVersion: gameVersion}]
	r.mu.RUnlock()
	if !exists {
		if r.hasGame(gameID) {
			return nil, ErrVersionUnavailable
		}
		return nil, ErrModuleNotFound
	}
	if registration.Descriptor.Status == interfaces.ModuleRevoked {
		return nil, ErrVersionRevoked
	}
	module, err := instantiate(registration.Factory)
	if err != nil {
		return nil, fmt.Errorf("create game module %s %s: %w", gameID, gameVersion, err)
	}
	if module == nil || !sameDescriptorIdentity(registration.Descriptor, module.Descriptor()) {
		return nil, ErrFactoryMismatch
	}
	return module, nil
}

func (r *Registry) ResolveForNewMatch(gameID string) (interfaces.Module, error) {
	if r == nil {
		return nil, ErrModuleNotFound
	}
	r.mu.RLock()
	key, exists := r.active[gameID]
	r.mu.RUnlock()
	if !exists {
		if r.hasGame(gameID) {
			return nil, ErrNewMatchesDisabled
		}
		return nil, ErrModuleNotFound
	}
	return r.Resolve(key.gameID, key.gameVersion)
}

func (r *Registry) List() []interfaces.Descriptor {
	if r == nil {
		return []interfaces.Descriptor{}
	}
	r.mu.RLock()
	items := make([]interfaces.Descriptor, 0, len(r.registrations))
	for _, registration := range r.registrations {
		items = append(items, registration.Descriptor.Clone())
	}
	r.mu.RUnlock()
	sort.Slice(items, func(i, j int) bool {
		if items[i].ID == items[j].ID {
			return items[i].Versions.Game < items[j].Versions.Game
		}
		return items[i].ID < items[j].ID
	})
	return items
}

func (r *Registry) hasGame(gameID string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	for key := range r.registrations {
		if key.gameID == gameID {
			return true
		}
	}
	return false
}

func sameDescriptorIdentity(expected, actual interfaces.Descriptor) bool {
	return reflect.DeepEqual(expected.Clone(), actual.Clone())
}

func instantiate(factory Factory) (module interfaces.Module, err error) {
	defer func() {
		if recovered := recover(); recovered != nil {
			module = nil
			err = fmt.Errorf("%w: %v", ErrFactoryPanic, recovered)
		}
	}()
	return factory()
}
