package registry

import (
	"fmt"

	"skill-arena/internal/games/maze"
	"skill-arena/internal/games/maze/generator"
	"skill-arena/internal/storage"
)

type ProductionDependencies struct {
	Puzzles *generator.Service
	Objects storage.ObjectStore
}

// NewProduction explicitly registers only approved production modules.
func NewProduction(dependencies ...ProductionDependencies) (*Registry, error) {
	var values ProductionDependencies
	if len(dependencies) > 0 {
		values = dependencies[0]
	}
	mazeRegistration, err := productionMazeRegistration(values)
	if err != nil {
		return nil, err
	}
	return New(mazeRegistration)
}

func productionMazeRegistration(dependencies ProductionDependencies) (registration Registration, err error) {
	defer func() {
		if recovered := recover(); recovered != nil {
			registration = Registration{}
			err = fmt.Errorf("load Maze module manifest: %v", recovered)
		}
	}()
	module := maze.New()
	versions := LegacyVersions{
		Renderer:    "v1",
		StateSchema: "v1",
	}
	if dependencies.Puzzles == nil {
		return LegacyRegistration(module, versions)
	}
	runtime, err := maze.NewRuntime(dependencies.Puzzles, dependencies.Objects)
	if err != nil {
		return Registration{}, err
	}
	return RuntimeLegacyRegistration(module, runtime, versions)
}
