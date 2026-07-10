package game

import (
	"math/rand"
	"strings"
	"time"

	"skill-arena/internal/models"
)

type Maze struct {
	Width  int      `json:"width"`
	Height int      `json:"height"`
	Cells  []string `json:"cells"`
	StartX int      `json:"startX"`
	StartY int      `json:"startY"`
	EndX   int      `json:"endX"`
	EndY   int      `json:"endY"`
}

func init() {
	rand.Seed(time.Now().UnixNano())
}

func GenerateMaze(width, height int) *Maze {
	if width%2 == 0 {
		width++
	}
	if height%2 == 0 {
		height++
	}

	cells := make([][]byte, height)
	for y := range cells {
		cells[y] = make([]byte, width)
		for x := range cells[y] {
			cells[y][x] = '#'
		}
	}

	startX, startY := 1, 1
	endX, endY := width-2, height-2
	crawl(startX, startY, cells)
	cells[startY][startX] = '.'
	cells[endY][endX] = '.'

	rows := make([]string, height)
	for y := 0; y < height; y++ {
		rows[y] = string(cells[y])
	}

	return &Maze{
		Width:  width,
		Height: height,
		Cells:  rows,
		StartX: startX,
		StartY: startY,
		EndX:   endX,
		EndY:   endY,
	}
}

func crawl(x, y int, cells [][]byte) {
	cells[y][x] = '.'
	dirs := []struct{ dx, dy int }{{2, 0}, {-2, 0}, {0, 2}, {0, -2}}
	rand.Shuffle(len(dirs), func(i, j int) { dirs[i], dirs[j] = dirs[j], dirs[i] })

	for _, dir := range dirs {
		nx := x + dir.dx
		ny := y + dir.dy
		if nx <= 0 || nx >= len(cells[0])-1 || ny <= 0 || ny >= len(cells)-1 {
			continue
		}
		if cells[ny][nx] == '#' {
			cells[y+dir.dy/2][x+dir.dx/2] = '.'
			crawl(nx, ny, cells)
		}
	}
}

func ParseMazeCell(maze *Maze, x, y int) byte {
	if y < 0 || y >= len(maze.Cells) {
		return '#'
	}
	row := maze.Cells[y]
	if x < 0 || x >= len(row) {
		return '#'
	}
	return row[x]
}

func ValidateMazeMoves(maze *Maze, moves []string) (bool, []models.MazeMove, error) {
	x, y := maze.StartX, maze.StartY
	validated := make([]models.MazeMove, 0, len(moves))

	for _, raw := range moves {
		dir := strings.ToLower(strings.TrimSpace(raw))
		dx, dy := 0, 0
		switch dir {
		case "up":
			dy = -1
		case "down":
			dy = 1
		case "left":
			dx = -1
		case "right":
			dx = 1
		default:
			return false, nil, nil
		}

		nx := x + dx
		ny := y + dy
		if ParseMazeCell(maze, nx, ny) == '#' {
			return false, append(validated, models.MazeMove{Direction: dir, X: nx, Y: ny, Timestamp: time.Now().UTC()}), nil
		}
		x, y = nx, ny
		validated = append(validated, models.MazeMove{Direction: dir, X: x, Y: y, Timestamp: time.Now().UTC()})
	}

	return x == maze.EndX && y == maze.EndY, validated, nil
}

type point struct {
	x int
	y int
}

func ShortestPathLength(maze *Maze) int {
	start := point{x: maze.StartX, y: maze.StartY}
	end := point{x: maze.EndX, y: maze.EndY}
	queue := []point{start}
	distance := map[point]int{start: 0}
	directions := []point{{x: 1}, {x: -1}, {y: 1}, {y: -1}}

	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]
		if current == end {
			return distance[current]
		}

		for _, dir := range directions {
			next := point{x: current.x + dir.x, y: current.y + dir.y}
			if ParseMazeCell(maze, next.x, next.y) == '#' {
				continue
			}
			if _, seen := distance[next]; seen {
				continue
			}
			distance[next] = distance[current] + 1
			queue = append(queue, next)
		}
	}

	return 0
}
