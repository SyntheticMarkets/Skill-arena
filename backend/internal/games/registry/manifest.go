package registry

import (
	"fmt"
	"regexp"
	"strings"

	"skill-arena/internal/games/interfaces"
)

var (
	gameIDPattern      = regexp.MustCompile(`^[a-z][a-z0-9_]{1,63}$`)
	semanticVersion    = regexp.MustCompile(`^[0-9]+\.[0-9]+\.[0-9]+(?:-[0-9A-Za-z.-]+)?$`)
	contractVersion    = regexp.MustCompile(`^[0-9A-Za-z][0-9A-Za-z._-]{0,63}$`)
	rendererKeyPattern = regexp.MustCompile(`^[a-z][a-z0-9-]{1,63}$`)
	hashPattern        = regexp.MustCompile(`^sha256:[0-9a-f]{64}$`)
	modePattern        = regexp.MustCompile(`^[a-z][a-z0-9_]{1,63}$`)
)

func ValidateDescriptor(descriptor interfaces.Descriptor) error {
	if !gameIDPattern.MatchString(descriptor.ID) {
		return invalid("id")
	}
	if strings.TrimSpace(descriptor.Name) == "" {
		return invalid("name")
	}
	if !semanticVersion.MatchString(descriptor.Versions.Game) {
		return invalid("game version")
	}
	if !descriptor.Versions.Complete() ||
		!contractVersion.MatchString(descriptor.Versions.Rules) ||
		!contractVersion.MatchString(descriptor.Versions.Protocol) ||
		!contractVersion.MatchString(descriptor.Versions.Replay) ||
		!contractVersion.MatchString(descriptor.Versions.Renderer) ||
		!contractVersion.MatchString(descriptor.Versions.StateSchema) {
		return invalid("version tuple")
	}
	switch descriptor.Status {
	case interfaces.ModuleActive, interfaces.ModuleReplayOnly, interfaces.ModuleRetired, interfaces.ModuleRevoked:
	default:
		return invalid("status")
	}
	if descriptor.NewMatchAllowed && descriptor.Status != interfaces.ModuleActive {
		return invalid("new-match status")
	}
	if descriptor.MinimumPlayers < 1 || descriptor.MaximumPlayers < descriptor.MinimumPlayers {
		return invalid("player range")
	}
	if descriptor.AverageTimeSec < 1 {
		return invalid("average time")
	}
	if !rendererKeyPattern.MatchString(descriptor.RendererKey) {
		return invalid("renderer key")
	}
	if !hashPattern.MatchString(descriptor.ManifestHash) {
		return invalid("manifest hash")
	}
	if len(descriptor.Modes) == 0 {
		return invalid("modes")
	}
	seen := make(map[string]struct{}, len(descriptor.Modes))
	for _, mode := range descriptor.Modes {
		if !modePattern.MatchString(mode) {
			return invalid("mode")
		}
		if _, exists := seen[mode]; exists {
			return invalid("duplicate mode")
		}
		seen[mode] = struct{}{}
	}
	return nil
}

func invalid(field string) error {
	return fmt.Errorf("%w: %s", ErrInvalidDescriptor, field)
}
