package game

import (
	"testing"

	"skill-arena/internal/models"
)

func TestValidateLineClicksUsesPhysicalEscapePath(t *testing.T) {
	lines := []models.ArrowLine{
		{
			ID:        "blocker",
			Direction: "up",
			Points:    []models.Point{{X: 30, Y: 30}, {X: 30, Y: 20}},
		},
		{
			ID:        "blocked",
			Direction: "right",
			Points:    []models.Point{{X: 10, Y: 20}, {X: 20, Y: 20}},
		},
	}

	complete, state, clicks := ValidateLineClicks(lines, []string{"blocked"})
	if complete {
		t.Fatal("blocked click completed puzzle")
	}
	if len(clicks) != 1 || clicks[0].Success || clicks[0].FailureReason == "" {
		t.Fatalf("blocked click = %#v, want failed click with reason", clicks)
	}
	if state[1].Removed {
		t.Fatal("blocked line was removed")
	}

	complete, state, clicks = ValidateLineClicks(lines, []string{"blocker", "blocked"})
	if !complete {
		t.Fatal("clearing physical blocker first did not complete puzzle")
	}
	if len(clicks) != 2 || !clicks[0].Success || !clicks[1].Success {
		t.Fatalf("ordered clicks = %#v, want two successes", clicks)
	}
	if !state[0].Removed || !state[1].Removed {
		t.Fatalf("state = %#v, want both lines removed", state)
	}
}

func TestValidateLineClicksDoesNotUseOppositeDirectionToEscape(t *testing.T) {
	lines := []models.ArrowLine{
		{
			ID:        "left-blocker",
			Direction: "up",
			Points:    []models.Point{{X: 10, Y: 30}, {X: 10, Y: 20}},
		},
		{
			ID:        "right-blocker",
			Direction: "up",
			Points:    []models.Point{{X: 30, Y: 30}, {X: 30, Y: 20}},
		},
		{
			ID:        "arrow",
			Direction: "right",
			Points:    []models.Point{{X: 12, Y: 20}, {X: 20, Y: 20}},
		},
	}

	_, state, clicks := ValidateLineClicks(lines, []string{"arrow"})
	if len(clicks) != 1 || clicks[0].Success {
		t.Fatalf("click = %#v, want blocked by right-side blocker", clicks)
	}
	if state[2].Removed {
		t.Fatal("arrow escaped by using the opposite direction")
	}
	if clicks[0].FailureReason != "blocked_by_right-blocker" {
		t.Fatalf("failure reason = %q, want right-side blocker", clicks[0].FailureReason)
	}
}

func TestValidateLineClicksIgnoresDeclaredDependencies(t *testing.T) {
	lines := []models.ArrowLine{
		{
			ID:        "free",
			Direction: "right",
			Points:    []models.Point{{X: 10, Y: 40}, {X: 20, Y: 40}},
			DependsOn: []string{"not-a-physical-blocker"},
		},
	}

	complete, state, clicks := ValidateLineClicks(lines, []string{"free"})
	if !complete || len(clicks) != 1 || !clicks[0].Success || !state[0].Removed {
		t.Fatalf("physical free path should clear despite declared dependency, complete=%v state=%#v clicks=%#v", complete, state, clicks)
	}
}

func TestGenerateLinePuzzleIsSolvableByPhysicalSolver(t *testing.T) {
	lines := GenerateLinePuzzle("test-seed", 40, 4)
	for _, line := range lines {
		if len(line.Points) < 2 {
			t.Fatalf("line %s has no routed geometry: %#v", line.ID, line)
		}
	}
	clicks, solvable := SolveLinePuzzle(lines)
	if !solvable {
		t.Fatalf("generated line puzzle was not physically solvable; partial solution=%v", clicks)
	}

	complete, _, history := ValidateLineClicks(lines, clicks)
	if !complete {
		t.Fatal("generated line puzzle was not solvable using solver output")
	}
	for _, click := range history {
		if !click.Success {
			t.Fatalf("solver order produced failed click: %#v", click)
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

	clicks, solvable := SolveLinePuzzle(lines)
	if !solvable {
		t.Fatal("expected profile-generated puzzle to be physically solvable")
	}
	complete, _, _ := ValidateLineClicks(lines, clicks)
	if !complete {
		t.Fatal("expected profile-generated puzzle to be solvable using solver output")
	}
}
