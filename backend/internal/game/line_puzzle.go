package game

import (
	"crypto/sha256"
	"encoding/binary"
	"math"
	"math/rand"
	"strconv"
	"time"

	"skill-arena/internal/models"
)

func GenerateLinePuzzle(seed string, count int, dependencyDepth int) []models.ArrowLine {
	return generateLinePuzzle(seed, legacyDifficultyProfile(count, dependencyDepth))
}

func generateLinePuzzle(seed string, profile models.DifficultyProfile) []models.ArrowLine {
	if profile.LineCount < 24 {
		profile.LineCount = 24
	}
	if profile.DependencyDepth < 1 {
		profile.DependencyDepth = 1
	}
	if profile.BranchingFactor < 1 {
		profile.BranchingFactor = 1
	}
	if profile.DependencyTrees < 1 {
		profile.DependencyTrees = 1
	}
	rng := rand.New(rand.NewSource(int64(hashSeed(seed))))
	lines := make([]models.ArrowLine, 0, profile.LineCount)
	routes := newRouteState(profile.DependencyTrees)
	treeSpan := profile.LineCount / profile.DependencyTrees
	if treeSpan < 4 {
		treeSpan = 4
	}
	for i := 0; i < profile.LineCount; i++ {
		dependencies := []string{}
		treeRoot := i%treeSpan == 0
		dependencyChance := 0.58 + profile.FalseRouteRate
		if dependencyChance > 0.9 {
			dependencyChance = 0.9
		}
		if i > 0 && !treeRoot && rng.Float64() < dependencyChance {
			dependencyCount := 1
			if profile.BranchingFactor > 1 {
				maxBranch := profile.BranchingFactor
				if maxBranch > i {
					maxBranch = i
				}
				if maxBranch > 1 {
					dependencyCount += rng.Intn(maxBranch)
				}
			}
			if dependencyCount > i {
				dependencyCount = i
			}
			seen := map[int]bool{}
			min := i - profile.DependencyDepth
			if min < 0 {
				min = 0
			}
			treeStart := (i / treeSpan) * treeSpan
			if min < treeStart {
				min = treeStart
			}
			if min >= i {
				min = 0
			}
			available := i - min
			if dependencyCount > available {
				dependencyCount = available
			}
			for len(dependencies) < dependencyCount {
				idx := min + rng.Intn(i-min+1)
				if idx == i || seen[idx] {
					continue
				}
				seen[idx] = true
				dependencies = append(dependencies, lineID(idx))
			}
		}
		if profile.CrossDependencies > 0 && i > 3 && rng.Intn(profile.CrossDependencies+6) > 4 {
			idx := rng.Intn(i)
			candidate := lineID(idx)
			if !containsString(dependencies, candidate) {
				dependencies = append(dependencies, candidate)
			}
		}
		if profile.DeadEndFactor > 0 && i > 1 && rng.Float64() < profile.DeadEndFactor {
			idx := rng.Intn(i)
			candidate := lineID(idx)
			if !containsString(dependencies, candidate) {
				dependencies = append(dependencies, candidate)
			}
		}
		routeIndex := i % len(routes)
		if treeRoot && i > 0 {
			routes[routeIndex] = resetRoute(routeIndex, len(routes), rng)
		}
		points, direction := nextRoutePath(&routes[routeIndex], rng, profile, i)
		lines = append(lines, models.ArrowLine{
			ID:        lineID(i),
			Direction: direction,
			X:         points[0].X,
			Y:         points[0].Y,
			Length:    pathLength(points),
			Points:    points,
			DependsOn: dependencies,
		})
	}
	return lines
}

type routeState struct {
	x       float64
	y       float64
	minX    float64
	maxX    float64
	anchors []models.Point
}

func newRouteState(count int) []routeState {
	if count < 3 {
		count = 3
	}
	if count > 8 {
		count = 8
	}
	routes := make([]routeState, count)
	for i := range routes {
		routes[i] = resetRoute(i, count, nil)
	}
	return routes
}

func resetRoute(index, count int, rng *rand.Rand) routeState {
	band := 84.0 / float64(count)
	minX := 8 + float64(index)*band
	maxX := minX + band - 2
	x := minX + band*0.5
	y := 8 + float64(index%3)*3
	if rng != nil {
		x = clampFloat(minX+3+rng.Float64()*(band-7), minX+2, maxX-2)
		y = 7 + rng.Float64()*10
	}
	return routeState{x: x, y: y, minX: minX, maxX: maxX, anchors: []models.Point{{X: x, Y: y}}}
}

func nextRoutePath(route *routeState, rng *rand.Rand, profile models.DifficultyProfile, index int) ([]models.Point, string) {
	start := models.Point{X: route.x, Y: route.y}
	if len(route.anchors) > 2 && rng.Float64() < profile.FalseRouteRate*0.5 {
		start = route.anchors[rng.Intn(len(route.anchors))]
	}

	step := 7.5 + rng.Float64()*8
	if profile.NoiseFactor > 0 {
		step += rng.Float64() * profile.NoiseFactor * 3
	}

	points := []models.Point{start}
	direction := "down"
	switch {
	case route.y > 84:
		direction = "up"
	case route.y < 16:
		direction = "down"
	case rng.Float64() < 0.46:
		if rng.Float64() < 0.5 {
			direction = "left"
		} else {
			direction = "right"
		}
	case rng.Float64() < 0.16+profile.FalseRouteRate:
		direction = "up"
	default:
		direction = "down"
	}

	end := movePoint(start, direction, step)
	end.X = clampFloat(end.X, route.minX+1, route.maxX-1)
	end.Y = clampFloat(end.Y, 7, 91)

	// Most lines get one or two orthogonal bends so the board reads as a routed
	// engineering diagram instead of disconnected straight strokes.
	if direction == "left" || direction == "right" {
		if rng.Float64() < 0.64 {
			verticalDirection := "down"
			if rng.Float64() < 0.28 && start.Y > 18 {
				verticalDirection = "up"
			}
			midX := clampFloat((start.X+end.X)/2+rng.Float64()*4-2, route.minX+1, route.maxX-1)
			midY := movePoint(start, verticalDirection, 5+rng.Float64()*9).Y
			midY = clampFloat(midY, 7, 91)
			points = append(points, models.Point{X: midX, Y: start.Y}, models.Point{X: midX, Y: midY}, models.Point{X: end.X, Y: midY})
			end = points[len(points)-1]
		} else {
			points = append(points, end)
		}
	} else {
		if rng.Float64() < 0.42 {
			horizontal := "right"
			if start.X > (route.minX+route.maxX)/2 {
				horizontal = "left"
			}
			offset := 4 + rng.Float64()*7
			elbowX := movePoint(start, horizontal, offset).X
			elbowX = clampFloat(elbowX, route.minX+1, route.maxX-1)
			points = append(points, models.Point{X: elbowX, Y: start.Y}, models.Point{X: elbowX, Y: end.Y}, end)
		} else {
			points = append(points, end)
		}
	}

	points = simplifyPoints(points)
	last := points[len(points)-1]
	route.x = last.X
	route.y = last.Y
	route.anchors = append(route.anchors, last)
	if len(route.anchors) > 12 {
		route.anchors = route.anchors[len(route.anchors)-12:]
	}

	if index%9 == 0 && route.y > 74 {
		route.y = 10 + rng.Float64()*10
		route.x = clampFloat(route.x+rng.Float64()*8-4, route.minX+2, route.maxX-2)
		route.anchors = append(route.anchors, models.Point{X: route.x, Y: route.y})
	}
	return points, directionFromPoints(points)
}

func movePoint(point models.Point, direction string, distance float64) models.Point {
	switch direction {
	case "up":
		point.Y -= distance
	case "down":
		point.Y += distance
	case "left":
		point.X -= distance
	case "right":
		point.X += distance
	}
	return point
}

func directionFromPoints(points []models.Point) string {
	if len(points) < 2 {
		return "right"
	}
	a := points[len(points)-2]
	b := points[len(points)-1]
	if math.Abs(b.X-a.X) >= math.Abs(b.Y-a.Y) {
		if b.X < a.X {
			return "left"
		}
		return "right"
	}
	if b.Y < a.Y {
		return "up"
	}
	return "down"
}

func pathLength(points []models.Point) float64 {
	total := 0.0
	for i := 1; i < len(points); i++ {
		total += math.Abs(points[i].X-points[i-1].X) + math.Abs(points[i].Y-points[i-1].Y)
	}
	if total < 4 {
		return 4
	}
	return total
}

func simplifyPoints(points []models.Point) []models.Point {
	if len(points) < 3 {
		return points
	}
	simplified := []models.Point{points[0]}
	for i := 1; i < len(points)-1; i++ {
		prev := simplified[len(simplified)-1]
		current := points[i]
		next := points[i+1]
		if (almostEqual(prev.X, current.X) && almostEqual(current.X, next.X)) ||
			(almostEqual(prev.Y, current.Y) && almostEqual(current.Y, next.Y)) {
			continue
		}
		if !almostEqual(prev.X, current.X) || !almostEqual(prev.Y, current.Y) {
			simplified = append(simplified, current)
		}
	}
	last := points[len(points)-1]
	if !almostEqual(simplified[len(simplified)-1].X, last.X) || !almostEqual(simplified[len(simplified)-1].Y, last.Y) {
		simplified = append(simplified, last)
	}
	return simplified
}

func almostEqual(a, b float64) bool {
	return math.Abs(a-b) < 0.01
}

func clampFloat(value, min, max float64) float64 {
	if value < min {
		return min
	}
	if value > max {
		return max
	}
	return value
}

func ValidateLineClicks(lines []models.ArrowLine, lineIDs []string) (bool, []models.ArrowLine, []models.ArrowClick) {
	state := make([]models.ArrowLine, len(lines))
	copy(state, lines)
	byID := map[string]int{}
	removed := map[string]bool{}
	for idx, line := range state {
		byID[line.ID] = idx
		if line.Removed {
			removed[line.ID] = true
		}
		state[idx].Blocked = false
	}

	clicks := make([]models.ArrowClick, 0, len(lineIDs))
	combo := 0
	for _, id := range lineIDs {
		click := models.ArrowClick{LineID: id, Timestamp: time.Now().UTC()}
		idx, ok := byID[id]
		if !ok {
			combo = 0
			click.FailureReason = "line_not_found"
			click.ClearedCount = len(removed)
			clicks = append(clicks, click)
			continue
		}
		line := state[idx]
		if line.Removed {
			combo = 0
			click.FailureReason = "already_removed"
			click.ClearedCount = len(removed)
			clicks = append(clicks, click)
			continue
		}
		blocker := ""
		for _, dependency := range line.DependsOn {
			if !removed[dependency] {
				blocker = dependency
				break
			}
		}
		if blocker != "" {
			combo = 0
			state[idx].Blocked = true
			click.FailureReason = "blocked_by_" + blocker
			click.ClearedCount = len(removed)
			clicks = append(clicks, click)
			continue
		}
		combo++
		state[idx].Removed = true
		state[idx].Blocked = false
		removed[id] = true
		click.Success = true
		click.Combo = combo
		click.ClearedCount = len(removed)
		clicks = append(clicks, click)
	}

	return len(removed) == len(lines), state, clicks
}

func hashSeed(seed string) uint64 {
	sum := sha256.Sum256([]byte(seed))
	return binary.BigEndian.Uint64(sum[:8])
}

func lineID(index int) string {
	return "line-" + strconv.Itoa(index+1)
}

func containsString(values []string, candidate string) bool {
	for _, value := range values {
		if value == candidate {
			return true
		}
	}
	return false
}
