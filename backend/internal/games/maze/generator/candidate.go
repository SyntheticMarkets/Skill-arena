package generator

import (
	"context"
	"errors"
	"fmt"
)

type Candidate struct {
	Index   int             `json:"index"`
	Pattern Pattern         `json:"pattern"`
	Board   Board           `json:"board"`
	Graph   DependencyGraph `json:"graph"`
}

type GenerationConfig struct {
	RulesVersion      int
	CandidateBatch    int
	MaximumWorkers    int
	PlacementAttempts int
}

func DefaultGenerationConfig() GenerationConfig {
	return GenerationConfig{
		RulesVersion: 1, CandidateBatch: 16, MaximumWorkers: 4,
		PlacementAttempts: 512,
	}
}

func (c GenerationConfig) Validate() error {
	if c.RulesVersion <= 0 {
		return errors.New("rules version must be positive")
	}
	if c.CandidateBatch < 1 || c.CandidateBatch > 64 {
		return errors.New("candidate batch must be between 1 and 64")
	}
	if c.MaximumWorkers < 1 || c.MaximumWorkers > c.CandidateBatch {
		return errors.New("maximum workers must be positive and not exceed the candidate batch")
	}
	if c.PlacementAttempts < 32 || c.PlacementAttempts > 4096 {
		return errors.New("placement attempts are outside production bounds")
	}
	return nil
}

func generateCandidate(ctx context.Context, input ProcessingInput, config GenerationConfig, index int) (Candidate, string, error) {
	if err := ctx.Err(); err != nil {
		return Candidate{}, RejectionCanceled, err
	}
	dimensionStream, err := NewRandomStream(input.Seed, fmt.Sprintf("candidate:%d:board-dimensions", index))
	if err != nil {
		return Candidate{}, RejectionRandomStream, err
	}
	lineStream, err := NewRandomStream(input.Seed, fmt.Sprintf("candidate:%d:line-count", index))
	if err != nil {
		return Candidate{}, RejectionRandomStream, err
	}
	lineCount, err := boundedInt(lineStream, input.Profile.LineCountMin, input.Profile.LineCountMax)
	if err != nil {
		return Candidate{}, RejectionProfileInvalid, err
	}
	columns, rows, err := boardDimensions(dimensionStream, input.Profile, lineCount)
	if err != nil {
		return Candidate{}, RejectionProfileInvalid, err
	}
	patternStream, err := NewRandomStream(input.Seed, fmt.Sprintf("candidate:%d:pattern-selection", index))
	if err != nil {
		return Candidate{}, RejectionRandomStream, err
	}
	pattern, err := SelectPattern(patternStream, input.Profile, columns, rows)
	if err != nil {
		return Candidate{}, RejectionPatternUnavailable, err
	}
	placementStream, err := NewRandomStream(input.Seed, fmt.Sprintf("candidate:%d:arrow-placement", index))
	if err != nil {
		return Candidate{}, RejectionRandomStream, err
	}
	board := Board{
		GeometryVersion: input.Metadata.Version.GeometrySchemaVersion,
		RulesVersion:    config.RulesVersion, Columns: columns, Rows: rows,
		Arrows: make([]Arrow, 0, lineCount),
	}
	first, err := initialArrow(placementStream, board, pattern)
	if err != nil {
		return Candidate{}, RejectionPlacementExhausted, err
	}
	board.Arrows = append(board.Arrows, first)
	for len(board.Arrows) < lineCount {
		if err := ctx.Err(); err != nil {
			return Candidate{}, RejectionCanceled, err
		}
		arrow, placed := placeDependentArrow(placementStream, board, pattern, config.PlacementAttempts)
		if !placed {
			return Candidate{}, RejectionPlacementExhausted, nil
		}
		board.Arrows = append(board.Arrows, arrow)
	}
	if err := ValidateGeometry(board); err != nil {
		return Candidate{}, RejectionGeometryInvalid, err
	}
	graph, err := DeriveDependencies(board)
	if err != nil {
		return Candidate{}, RejectionDependencyInvalid, err
	}
	if !dependenciesRespectConstructionOrder(board, graph) {
		return Candidate{}, RejectionDependencyOrder, nil
	}
	return Candidate{Index: index, Pattern: pattern, Board: board, Graph: graph}, "", nil
}

func boardDimensions(stream *RandomStream, profile DifficultyProfile, lineCount int) (int, int, error) {
	density := (profile.DensityMinBPS + profile.DensityMaxBPS) / 2
	if density <= 0 {
		density = 2500
	}
	estimatedCells := lineCount * 3
	area := (estimatedCells*10_000 + density - 1) / density
	side := ceilSquareRoot(maxInt(area, 64))
	variance, err := stream.Uint64n(5)
	if err != nil {
		return 0, 0, err
	}
	columns := clampInt(side+int(variance)-2, 8, 32)
	otherVariance, err := stream.Uint64n(5)
	if err != nil {
		return 0, 0, err
	}
	rows := clampInt(side+int(otherVariance)-2, 8, 32)
	for columns*rows < area {
		if columns <= rows && columns < 32 {
			columns++
		} else if rows < 32 {
			rows++
		} else {
			return 0, 0, errors.New("difficulty density requires a board above production bounds")
		}
	}
	return columns, rows, nil
}

func ceilSquareRoot(value int) int {
	if value <= 0 {
		return 0
	}
	result := 1
	for result*result < value {
		result++
	}
	return result
}

func initialArrow(stream *RandomStream, board Board, pattern Pattern) (Arrow, error) {
	offset, err := stream.Uint64n(4)
	if err != nil {
		return Arrow{}, err
	}
	preferred := patternDirection(pattern.ID, 0)
	direction := Direction((int(preferred) + int(offset)) % 4)
	lengthValue, err := stream.Uint64n(3)
	if err != nil {
		return Arrow{}, err
	}
	length := 2 + int(lengthValue)
	switch direction {
	case DirectionRight:
		row, err := boundedInt(stream, 1, board.Rows-2)
		if err != nil {
			return Arrow{}, err
		}
		return straightArrow(0, Cell{Column: board.Columns - 1, Row: row}, direction, length), nil
	case DirectionLeft:
		row, err := boundedInt(stream, 1, board.Rows-2)
		if err != nil {
			return Arrow{}, err
		}
		return straightArrow(0, Cell{Column: 0, Row: row}, direction, length), nil
	case DirectionUp:
		column, err := boundedInt(stream, 1, board.Columns-2)
		if err != nil {
			return Arrow{}, err
		}
		return straightArrow(0, Cell{Column: column, Row: 0}, direction, length), nil
	default:
		column, err := boundedInt(stream, 1, board.Columns-2)
		if err != nil {
			return Arrow{}, err
		}
		return straightArrow(0, Cell{Column: column, Row: board.Rows - 1}, direction, length), nil
	}
}

func placeDependentArrow(stream *RandomStream, board Board, pattern Pattern, maximumAttempts int) (Arrow, bool) {
	occupied := occupiedCells(board)
	nextIndex := len(board.Arrows)
	for attempt := 0; attempt < maximumAttempts; attempt++ {
		targetIndex, err := selectTargetArrow(stream, pattern.ID, nextIndex)
		if err != nil {
			return Arrow{}, false
		}
		target := board.Arrows[targetIndex]
		cellIndex, err := boundedInt(stream, 0, len(target.Cells)-1)
		if err != nil {
			return Arrow{}, false
		}
		targetCell := target.Cells[cellIndex]
		offset, err := stream.Uint64n(4)
		if err != nil {
			return Arrow{}, false
		}
		preferred := patternDirection(pattern.ID, nextIndex+attempt)
		direction := Direction((int(preferred) + int(offset)) % 4)
		distanceValue, err := stream.Uint64n(4)
		if err != nil {
			return Arrow{}, false
		}
		distance := 1 + int(distanceValue)
		lengthValue, err := stream.Uint64n(3)
		if err != nil {
			return Arrow{}, false
		}
		length := 2 + int(lengthValue)
		dx, dy := direction.vector()
		head := Cell{
			Column: targetCell.Column - dx*distance,
			Row:    targetCell.Row - dy*distance,
		}
		arrow := straightArrow(nextIndex, head, direction, length)
		if !arrowFits(board, arrow, occupied) {
			continue
		}
		tentative := board.Clone()
		tentative.Arrows = append(tentative.Arrows, arrow)
		graph, err := DeriveDependencies(tentative)
		if err != nil || len(graph.Edges[arrow.ID]) == 0 {
			continue
		}
		if !dependenciesRespectConstructionOrder(tentative, graph) {
			continue
		}
		return arrow, true
	}
	return Arrow{}, false
}

func straightArrow(index int, head Cell, direction Direction, length int) Arrow {
	dx, dy := direction.vector()
	cells := make([]Cell, length)
	for cellIndex := 0; cellIndex < length; cellIndex++ {
		distanceFromHead := length - 1 - cellIndex
		cells[cellIndex] = Cell{
			Column: head.Column - dx*distanceFromHead,
			Row:    head.Row - dy*distanceFromHead,
		}
	}
	return Arrow{ID: fmt.Sprintf("a%04d", index), Cells: cells, Direction: direction}
}

func arrowFits(board Board, arrow Arrow, occupied map[uint64]struct{}) bool {
	local := map[uint64]struct{}{}
	for _, cell := range arrow.Cells {
		if !inBoard(board, cell) {
			return false
		}
		key := cell.key()
		if _, exists := occupied[key]; exists {
			return false
		}
		if _, exists := local[key]; exists {
			return false
		}
		local[key] = struct{}{}
	}
	return true
}

func occupiedCells(board Board) map[uint64]struct{} {
	result := make(map[uint64]struct{}, len(board.Arrows)*3)
	for _, arrow := range board.Arrows {
		for _, cell := range arrow.Cells {
			result[cell.key()] = struct{}{}
		}
	}
	return result
}

func selectTargetArrow(stream *RandomStream, patternID string, nextIndex int) (int, error) {
	if nextIndex <= 1 {
		return 0, nil
	}
	switch patternID {
	case "spiral", "piton":
		window := minInt(nextIndex, 4)
		offset, err := boundedInt(stream, 0, window-1)
		return nextIndex - 1 - offset, err
	case "braid", "maze_rows", "diagonal_weave":
		window := minInt(nextIndex, 8)
		offset, err := boundedInt(stream, 0, window-1)
		return nextIndex - 1 - offset, err
	default:
		return boundedInt(stream, 0, nextIndex-1)
	}
}

func patternDirection(patternID string, index int) Direction {
	switch patternID {
	case "maze_rows":
		if index%2 == 0 {
			return DirectionRight
		}
		return DirectionLeft
	case "spiral", "rings":
		return Direction(index % 4)
	case "diagonal_weave":
		return Direction((index*3 + 1) % 4)
	case "rays":
		return Direction((index + index/2) % 4)
	case "piton":
		return Direction((index/2 + 2) % 4)
	default:
		return Direction((index*2 + index/3) % 4)
	}
}

func dependenciesRespectConstructionOrder(board Board, graph DependencyGraph) bool {
	indexByID := make(map[string]int, len(board.Arrows))
	for index, arrow := range board.Arrows {
		indexByID[arrow.ID] = index
	}
	for dependent, blockers := range graph.Edges {
		dependentIndex, exists := indexByID[dependent]
		if !exists {
			return false
		}
		for _, blocker := range blockers {
			blockerIndex, exists := indexByID[blocker]
			if !exists || blockerIndex >= dependentIndex {
				return false
			}
		}
	}
	return true
}

func boundedInt(stream *RandomStream, minimum, maximum int) (int, error) {
	if maximum < minimum {
		return 0, errors.New("invalid integer range")
	}
	value, err := stream.Uint64n(uint64(maximum - minimum + 1))
	if err != nil {
		return 0, err
	}
	return minimum + int(value), nil
}

func minInt(first, second int) int {
	if first < second {
		return first
	}
	return second
}
