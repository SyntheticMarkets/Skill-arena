package registry

import (
	"fmt"

	"skill-arena/internal/games/maze"
)

// NewProduction explicitly registers only approved production modules.
func NewProduction() (*Registry, error) {
	mazeRegistration, err := productionMazeRegistration()
	if err != nil {
		return nil, err
	}
	return New(mazeRegistration)
}

func productionMazeRegistration() (registration Registration, err error) {
	defer func() {
		if recovered := recover(); recovered != nil {
			registration = Registration{}
			err = fmt.Errorf("load Maze module manifest: %v", recovered)
		}
	}()
	return LegacyRegistration(maze.New(), LegacyVersions{
		Renderer:    "v1",
		StateSchema: "v1",
	})
}
