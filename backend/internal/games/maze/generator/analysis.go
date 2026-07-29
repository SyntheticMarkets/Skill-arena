package generator

import (
	"encoding/json"
	"errors"
	"sort"
)

type MeasuredDifficulty struct {
	BoardColumns        int   `json:"boardColumns"`
	BoardRows           int   `json:"boardRows"`
	ArrowCount          int   `json:"arrowCount"`
	OccupiedCellCount   int   `json:"occupiedCellCount"`
	DensityBPS          int   `json:"densityBps"`
	MinimumActions      int   `json:"minimumActions"`
	InitiallyOpen       int   `json:"initiallyOpen"`
	DependencyEdges     int   `json:"dependencyEdges"`
	DependencyDepth     int   `json:"dependencyDepth"`
	Branching           int   `json:"branching"`
	CrossDependencies   int   `json:"crossDependencies"`
	IsolatedArrows      int   `json:"isolatedArrows"`
	BlockedChoiceBPS    int   `json:"blockedChoiceBps"`
	MaximumPathLength   int   `json:"maximumPathLength"`
	DirectionDiversity  int   `json:"directionDiversity"`
	VisualComplexity    int   `json:"visualComplexity"`
	ComplexityScore     int64 `json:"complexityScore"`
	ExpectedSolveTimeMS int64 `json:"expectedSolveTimeMs"`
}

func MeasureDifficulty(board Board, graph DependencyGraph) (MeasuredDifficulty, error) {
	if err := ValidateDependencyGraph(board, graph, true); err != nil {
		return MeasuredDifficulty{}, err
	}
	measured := MeasuredDifficulty{
		BoardColumns: board.Columns, BoardRows: board.Rows,
		ArrowCount: len(board.Arrows), MinimumActions: len(board.Arrows),
		InitiallyOpen: len(initiallyOpen(graph)),
	}
	directions := map[Direction]struct{}{}
	for _, arrow := range board.Arrows {
		measured.OccupiedCellCount += len(arrow.Cells)
		measured.MaximumPathLength = maxInt(measured.MaximumPathLength, len(arrow.Cells))
		directions[arrow.Direction] = struct{}{}
		blockers := len(graph.Edges[arrow.ID])
		measured.DependencyEdges += blockers
		if blockers > 1 {
			measured.CrossDependencies += blockers - 1
		}
		if blockers == 0 && len(graph.Dependents[arrow.ID]) == 0 {
			measured.IsolatedArrows++
		}
		measured.Branching = maxInt(measured.Branching, len(graph.Dependents[arrow.ID]))
	}
	measured.DirectionDiversity = len(directions)
	measured.DependencyDepth = dependencyDepth(graph)
	area := board.Columns * board.Rows
	if area <= 0 {
		return MeasuredDifficulty{}, errors.New("board area is invalid")
	}
	measured.DensityBPS = measured.OccupiedCellCount * 10_000 / area
	measured.BlockedChoiceBPS = (measured.ArrowCount - measured.InitiallyOpen) * 10_000 / measured.ArrowCount
	measured.VisualComplexity = clampInt(
		measured.DensityBPS/1000+measured.DirectionDiversity+measured.MaximumPathLength/2,
		0, 100,
	)
	measured.ComplexityScore =
		int64(measured.ArrowCount*100 +
			measured.DependencyEdges*120 +
			measured.DependencyDepth*180 +
			measured.Branching*90 +
			measured.CrossDependencies*80 +
			measured.DensityBPS/10 +
			measured.VisualComplexity*100)
	measured.ExpectedSolveTimeMS =
		int64(measured.ArrowCount*900 +
			measured.DependencyEdges*350 +
			measured.DependencyDepth*500 +
			measured.VisualComplexity*180)
	return measured, nil
}

func dependencyDepth(graph DependencyGraph) int {
	memo := map[string]int{}
	var visit func(string) int
	visit = func(id string) int {
		if depth, exists := memo[id]; exists {
			return depth
		}
		depth := 1
		for _, blocker := range graph.Edges[id] {
			depth = maxInt(depth, visit(blocker)+1)
		}
		memo[id] = depth
		return depth
	}
	longest := 0
	ids := make([]string, 0, len(graph.Edges))
	for id := range graph.Edges {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	for _, id := range ids {
		longest = maxInt(longest, visit(id))
	}
	return longest
}

func (m MeasuredDifficulty) Canonical() ([]byte, error) {
	return json.Marshal(m)
}

func CompareDifficulty(profile DifficultyProfile, measured MeasuredDifficulty, pattern Pattern) (bool, string) {
	checks := []struct {
		value   int64
		minimum int64
		maximum int64
		code    string
	}{
		{measured.ComplexityScore, profile.ComplexityMin, profile.ComplexityMax, RejectionComplexityMismatch},
		{int64(measured.ArrowCount), int64(profile.LineCountMin), int64(profile.LineCountMax), RejectionLineCountMismatch},
		{int64(measured.DependencyDepth), int64(profile.DependencyDepthMin), int64(profile.DependencyDepthMax), RejectionDepthMismatch},
		{int64(measured.Branching), int64(profile.BranchingMin), int64(profile.BranchingMax), RejectionBranchingMismatch},
		{int64(measured.CrossDependencies), int64(profile.FalseRoutesMin), int64(profile.FalseRoutesMax), RejectionFalseRoutesMismatch},
		{int64(measured.DensityBPS), int64(profile.DensityMinBPS), int64(profile.DensityMaxBPS), RejectionDensityMismatch},
		{measured.ExpectedSolveTimeMS, profile.ExpectedSolveTimeMinMS, profile.ExpectedSolveTimeMaxMS, RejectionSolveTimeMismatch},
		{int64(measured.VisualComplexity), int64(profile.VisualComplexityMin), int64(profile.VisualComplexityMax), RejectionVisualMismatch},
	}
	for _, check := range checks {
		if check.value < check.minimum || check.value > check.maximum {
			return false, check.code
		}
	}
	if profile.Source != "tutorial" && measured.IsolatedArrows > 0 {
		return false, RejectionIsolatedArrow
	}
	if profile.PatternBias != "" && profile.PatternBias != "balanced" && profile.PatternBias != "any" &&
		profile.PatternBias != pattern.ID {
		return false, RejectionPatternMismatch
	}
	return true, ""
}

func candidateRank(profile DifficultyProfile, measured MeasuredDifficulty, pattern Pattern, index int) [5]int64 {
	return [5]int64{
		absInt64(measured.ComplexityScore - midpoint64(profile.ComplexityMin, profile.ComplexityMax)),
		absInt64(int64(measured.VisualComplexity) - midpoint64(int64(profile.VisualComplexityMin), int64(profile.VisualComplexityMax))),
		absInt64(int64(measured.DependencyDepth)-midpoint64(int64(profile.DependencyDepthMin), int64(profile.DependencyDepthMax))) +
			absInt64(int64(measured.Branching)-midpoint64(int64(profile.BranchingMin), int64(profile.BranchingMax))),
		patternPenalty(profile.PatternBias, pattern.ID),
		int64(index),
	}
}

func rankLess(first, second [5]int64) bool {
	for index := range first {
		if first[index] != second[index] {
			return first[index] < second[index]
		}
	}
	return false
}

func midpoint64(minimum, maximum int64) int64 {
	return minimum + (maximum-minimum)/2
}

func absInt64(value int64) int64 {
	if value < 0 {
		return -value
	}
	return value
}

func patternPenalty(bias, selected string) int64 {
	if bias == "" || bias == "balanced" || bias == "any" || bias == selected {
		return 0
	}
	return 1
}
