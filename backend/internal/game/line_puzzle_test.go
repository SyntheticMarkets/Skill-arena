package game

import (
	"testing"

	"skill-arena/internal/models"
)

func TestValidateLineClicksRequiresDependenciesBeforeRemoval(t *testing.T) {
	lines := []models.ArrowLine{
		{ID: "line-1", Direction: "right", X: 10, Y: 10, Length: 8},
		{ID: "line-2", Direction: "down", X: 20, Y: 20, Length: 8, DependsOn: []string{"line-1"}},
	}

	complete, state, clicks := ValidateLineClicks(lines, []string{"line-2"})
	if complete {
		t.Fatal("blocked click completed puzzle")
	}
	if len(clicks) != 1 || clicks[0].Success || clicks[0].FailureReason == "" {
		t.Fatalf("blocked click = %#v, want failed click with reason", clicks)
	}
	if state[1].Removed {
		t.Fatal("blocked dependent line was removed")
	}

	complete, state, clicks = ValidateLineClicks(lines, []string{"line-1", "line-2"})
	if !complete {
		t.Fatal("ordered dependency clicks did not complete puzzle")
	}
	if len(clicks) != 2 || !clicks[0].Success || !clicks[1].Success {
		t.Fatalf("ordered clicks = %#v, want two successes", clicks)
	}
	if !state[0].Removed || !state[1].Removed {
		t.Fatalf("state = %#v, want both lines removed", state)
	}
}

func TestGenerateLinePuzzleIsSolvableInGeneratedOrder(t *testing.T) {
	lines := GenerateLinePuzzle("test-seed", 40, 4)
	clicks := make([]string, 0, len(lines))
	for _, line := range lines {
		if len(line.Points) < 2 {
			t.Fatalf("line %s has no routed geometry: %#v", line.ID, line)
		}
		clicks = append(clicks, line.ID)
	}

	complete, _, history := ValidateLineClicks(lines, clicks)
	if !complete {
		t.Fatal("generated line puzzle was not solvable in generated order")
	}
	for _, click := range history {
		if !click.Success {
			t.Fatalf("generated order produced failed click: %#v", click)
		}
	}
}

func TestDifficultyProfileScalesPuzzleComplexity(t *testing.T) {
	low := BuildDifficultyProfile(DifficultyInput{PlayerLevel: 1, LeagueTier: "Bronze", TrustScore: 100, Source: "test"})
	mid := BuildDifficultyProfile(DifficultyInput{PlayerLevel: 15, LeagueTier: "Gold", TrustScore: 100, Source: "test"})
	high := BuildDifficultyProfile(DifficultyInput{PlayerLevel: 500, LeagueTier: "Legend", TrustScore: 100, HouseTier: "diamond", TournamentTier: "championship", Source: "test"})

	if low.Rating >= mid.Rating || mid.Rating >= high.Rating {
		t.Fatalf("expected ratings to increase, got low=%d mid=%d high=%d", low.Rating, mid.Rating, high.Rating)
	}
	if low.ComplexityScore >= mid.ComplexityScore || mid.ComplexityScore >= high.ComplexityScore {
		t.Fatalf("expected complexity scores to increase, got low=%d mid=%d high=%d", low.ComplexityScore, mid.ComplexityScore, high.ComplexityScore)
	}
	if low.LineCount >= mid.LineCount || mid.LineCount >= high.LineCount {
		t.Fatalf("expected line counts to increase, got low=%d mid=%d high=%d", low.LineCount, mid.LineCount, high.LineCount)
	}
	if high.Rating != 100 {
		t.Fatalf("expected high profile to clamp at 100, got %d", high.Rating)
	}
	if high.DependencyDepth < 15 || high.BranchingFactor < 5 || high.DependencyTrees < 6 {
		t.Fatalf("expected high profile to produce deep board parameters: %+v", high)
	}
	if high.CrossDependencies == 0 || high.HumanSolveEstimate == 0 {
		t.Fatalf("expected APCE fields to be populated: %+v", high)
	}
	if high.ExpectedSolve.TopOnePercentSeconds == 0 || high.ExpectedSolve.TopTenPercentSeconds == 0 || high.ExpectedSolve.AverageSeconds == 0 {
		t.Fatalf("expected solve percentiles to be populated: %+v", high.ExpectedSolve)
	}
	if high.ExpectedSolve.TopOnePercentSeconds >= high.ExpectedSolve.AverageSeconds {
		t.Fatalf("top percentile estimate should be faster than average: %+v", high.ExpectedSolve)
	}
}

func TestComplexityScoreCanGrowBeyondRatingCap(t *testing.T) {
	elite := BuildDifficultyProfile(DifficultyInput{PlayerLevel: 500, LeagueTier: "Legend", TrustScore: 100, Source: "test"})
	mythic := BuildDifficultyProfile(DifficultyInput{PlayerLevel: 10000, LeagueTier: "Legend", TrustScore: 100, HouseTier: "diamond", TournamentTier: "championship", Source: "test"})

	if elite.Rating != 100 || mythic.Rating != 100 {
		t.Fatalf("expected both ratings to stay capped at 100, got elite=%d mythic=%d", elite.Rating, mythic.Rating)
	}
	if mythic.ComplexityScore <= elite.ComplexityScore {
		t.Fatalf("expected complexity to keep growing, got elite=%d mythic=%d", elite.ComplexityScore, mythic.ComplexityScore)
	}
	if mythic.HumanSolveEstimate <= elite.HumanSolveEstimate {
		t.Fatalf("expected solve estimate to keep growing, got elite=%d mythic=%d", elite.HumanSolveEstimate, mythic.HumanSolveEstimate)
	}
	if mythic.ExpectedSolve.AverageSeconds <= elite.ExpectedSolve.AverageSeconds {
		t.Fatalf("expected percentile estimates to keep growing, got elite=%+v mythic=%+v", elite.ExpectedSolve, mythic.ExpectedSolve)
	}
}

func TestProfileGeneratedPuzzleUsesProfileLineCount(t *testing.T) {
	profile := ProfileFromRating(50, "test")
	lines := GenerateLinePuzzleFromProfile("profile-seed", profile)
	if len(lines) != profile.LineCount {
		t.Fatalf("expected %d lines, got %d", profile.LineCount, len(lines))
	}

	clicks := make([]string, 0, len(lines))
	for _, line := range lines {
		clicks = append(clicks, line.ID)
	}
	complete, _, _ := ValidateLineClicks(lines, clicks)
	if !complete {
		t.Fatal("expected profile-generated puzzle to be solvable in generated order")
	}
}
