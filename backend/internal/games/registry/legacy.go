package registry

import (
	"fmt"
	"strconv"
	"strings"

	"skill-arena/internal/arena/core"
	arenaregistry "skill-arena/internal/arena/registry"
	"skill-arena/internal/games/interfaces"
	"skill-arena/internal/games/shared"
)

type LegacyVersions struct {
	Renderer    string
	StateSchema string
}

type LegacyModule struct {
	module     core.GameModule
	descriptor interfaces.Descriptor
}

type RuntimeLegacyModule struct {
	LegacyModule
	interfaces.Runtime
}

type RuntimeDeadlineLegacyModule struct {
	RuntimeLegacyModule
	interfaces.DeadlineRuntime
}

type RuntimeReplayLegacyModule struct {
	RuntimeLegacyModule
	interfaces.AuthoritativeReplayRuntime
}

type RuntimeFullLegacyModule struct {
	RuntimeDeadlineLegacyModule
	interfaces.AuthoritativeReplayRuntime
}

func (m LegacyModule) Descriptor() interfaces.Descriptor {
	return m.descriptor.Clone()
}

func (m LegacyModule) CoreModule() core.GameModule {
	return m.module
}

func LegacyRegistration(module core.GameModule, versions LegacyVersions) (Registration, error) {
	return legacyRegistration(module, nil, versions)
}

func RuntimeLegacyRegistration(
	module core.GameModule,
	runtime interfaces.Runtime,
	versions LegacyVersions,
) (Registration, error) {
	if runtime == nil {
		return Registration{}, fmt.Errorf("%w: game runtime is required", ErrInvalidDescriptor)
	}
	return legacyRegistration(module, runtime, versions)
}

func legacyRegistration(
	module core.GameModule,
	runtime interfaces.Runtime,
	versions LegacyVersions,
) (Registration, error) {
	if module == nil {
		return Registration{}, fmt.Errorf("%w: legacy module is required", ErrInvalidDescriptor)
	}
	manifest := module.Manifest()
	if manifest.ID != module.ID() {
		return Registration{}, fmt.Errorf("%w: legacy manifest id does not match module id", ErrInvalidDescriptor)
	}
	descriptor := descriptorFromLegacy(manifest, versions)
	if err := ValidateDescriptor(descriptor); err != nil {
		return Registration{}, err
	}
	return Registration{
		Descriptor: descriptor,
		Factory: func() (interfaces.Module, error) {
			legacy := LegacyModule{module: module, descriptor: descriptor}
			if runtime != nil {
				combined := RuntimeLegacyModule{LegacyModule: legacy, Runtime: runtime}
				deadline, hasDeadline := runtime.(interfaces.DeadlineRuntime)
				replay, hasReplay := runtime.(interfaces.AuthoritativeReplayRuntime)
				if hasDeadline && hasReplay {
					return RuntimeFullLegacyModule{
						RuntimeDeadlineLegacyModule: RuntimeDeadlineLegacyModule{
							RuntimeLegacyModule: combined, DeadlineRuntime: deadline,
						},
						AuthoritativeReplayRuntime: replay,
					}, nil
				}
				if hasDeadline {
					return RuntimeDeadlineLegacyModule{
						RuntimeLegacyModule: combined, DeadlineRuntime: deadline,
					}, nil
				}
				if hasReplay {
					return RuntimeReplayLegacyModule{
						RuntimeLegacyModule: combined, AuthoritativeReplayRuntime: replay,
					}, nil
				}
				return combined, nil
			}
			return legacy, nil
		},
	}, nil
}

func descriptorFromLegacy(manifest core.Manifest, versions LegacyVersions) interfaces.Descriptor {
	modes := append([]string(nil), manifest.Modes...)
	capabilities := interfaces.Capabilities{
		Practice:       manifest.Capabilities.Practice,
		PvP:            manifest.Capabilities.PvP,
		Ranked:         hasMode(modes, "ranked"),
		HouseChallenge: hasMode(modes, "house") || hasMode(modes, "house_challenge"),
		DailyChallenge: hasMode(modes, "daily") || hasMode(modes, "daily_challenge"),
		Tournament:     manifest.Capabilities.Tournament,
		Replay:         manifest.Capabilities.Replay,
		Spectator:      manifest.Capabilities.Spectator,
		AI:             manifest.Capabilities.AI,
		Teams:          manifest.Capabilities.Teams,
	}
	hash := shared.HashFields(
		"skill-arena/game-manifest/v1",
		manifest.ID,
		manifest.Name,
		manifest.Description,
		manifest.Category,
		manifest.Version,
		manifest.Author,
		manifest.Difficulty,
		strconv.Itoa(manifest.MinimumPlayers),
		strconv.Itoa(manifest.MaximumPlayers),
		strconv.Itoa(manifest.AverageTimeSec),
		manifest.RendererKey,
		strings.Join(modes, "\x1f"),
		manifest.Versions.Game,
		manifest.Versions.Rules,
		manifest.Versions.Replay,
		manifest.Versions.Protocol,
		versions.Renderer,
		versions.StateSchema,
		strconv.FormatBool(manifest.Capabilities.Practice),
		strconv.FormatBool(manifest.Capabilities.PvP),
		strconv.FormatBool(manifest.Capabilities.Replay),
		strconv.FormatBool(manifest.Capabilities.Tournament),
		strconv.FormatBool(manifest.Capabilities.Spectator),
		strconv.FormatBool(manifest.Capabilities.AI),
		strconv.FormatBool(manifest.Capabilities.Teams),
	)
	return interfaces.Descriptor{
		ID:              manifest.ID,
		Name:            manifest.Name,
		Description:     manifest.Description,
		Category:        manifest.Category,
		Author:          manifest.Author,
		Versions:        interfaces.Versions{Game: manifest.Versions.Game, Rules: manifest.Versions.Rules, Protocol: manifest.Versions.Protocol, Replay: manifest.Versions.Replay, Renderer: versions.Renderer, StateSchema: versions.StateSchema},
		Status:          interfaces.ModuleActive,
		NewMatchAllowed: true,
		MinimumPlayers:  manifest.MinimumPlayers,
		MaximumPlayers:  manifest.MaximumPlayers,
		AverageTimeSec:  manifest.AverageTimeSec,
		Modes:           modes,
		Capabilities:    capabilities,
		RendererKey:     manifest.RendererKey,
		ManifestHash:    hash,
	}
}

func (r *Registry) ArenaRegistry() (*arenaregistry.Registry, error) {
	legacy := arenaregistry.New()
	for _, descriptor := range r.List() {
		if descriptor.Status != interfaces.ModuleActive || !descriptor.NewMatchAllowed {
			continue
		}
		module, err := r.Resolve(descriptor.ID, descriptor.Versions.Game)
		if err != nil {
			return nil, err
		}
		provider, ok := module.(interface{ CoreModule() core.GameModule })
		if !ok {
			return nil, fmt.Errorf("%w: %s %s has no Sprint 5 adapter", ErrFactoryMismatch, descriptor.ID, descriptor.Versions.Game)
		}
		if err := legacy.Register(provider.CoreModule()); err != nil {
			return nil, err
		}
	}
	return legacy, nil
}

func hasMode(modes []string, wanted string) bool {
	for _, mode := range modes {
		if mode == wanted {
			return true
		}
	}
	return false
}
