package game

import "testing"

func TestShortestPathLength(t *testing.T) {
	maze := &Maze{
		Width:  5,
		Height: 5,
		Cells: []string{
			"#####",
			"#...#",
			"###.#",
			"#...#",
			"#####",
		},
		StartX: 1,
		StartY: 1,
		EndX:   3,
		EndY:   3,
	}

	if got := ShortestPathLength(maze); got != 4 {
		t.Fatalf("shortest path = %d, want 4", got)
	}
}
