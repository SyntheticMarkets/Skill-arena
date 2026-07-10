package game

import (
	"skill-arena/internal/config"
	"skill-arena/internal/models"
)

type DifficultyInput struct {
	PlayerLevel    int
	LeagueTier     string
	TrustScore     float64
	HouseTier      string
	TournamentTier string
	SeasonalTier   string
	Source         string
}

func BuildDifficultyProfile(input DifficultyInput) models.DifficultyProfile {
	level := input.PlayerLevel
	if level < 1 {
		level = 1
	}
	settings := config.Runtime().Difficulty

	rawComplexity := settings.BaseRating*10 + level*settings.LevelMultiplier*12
	rawComplexity += leagueDifficultyBonus(input.LeagueTier) * 10
	rawComplexity += houseDifficultyBonus(input.HouseTier) * 12
	rawComplexity += tournamentDifficultyBonus(input.TournamentTier) * 10
	rawComplexity += seasonalDifficultyBonus(input.SeasonalTier) * 10

	if input.TrustScore > 0 && input.TrustScore < 70 {
		rawComplexity -= 80
	}
	if input.TrustScore >= 90 {
		rawComplexity += 30
	}
	if rawComplexity < 1 {
		rawComplexity = 1
	}

	rating := complexityBand(rawComplexity)
	if rating < settings.MinRating {
		rating = settings.MinRating
	}
	if rating > settings.MaxRating {
		rating = settings.MaxRating
	}

	return ProfileFromComplexity(rawComplexity, rating, input.Source)
}

func ProfileFromRating(rating int, source string) models.DifficultyProfile {
	settings := config.Runtime().Difficulty
	if rating < settings.MinRating {
		rating = settings.MinRating
	}
	if rating > settings.MaxRating {
		rating = settings.MaxRating
	}

	bandWidth := settings.MaxRating - settings.MinRating
	if bandWidth <= 0 {
		bandWidth = 99
	}
	complexity := settings.MinLineCount + ((rating-settings.MinRating)*10_000)/bandWidth
	return ProfileFromComplexity(complexity, rating, source)
}

func ProfileFromComplexity(complexityScore int, rating int, source string) models.DifficultyProfile {
	settings := config.Runtime().Difficulty
	if complexityScore < 1 {
		complexityScore = 1
	}
	if rating < settings.MinRating {
		rating = settings.MinRating
	}
	if rating > settings.MaxRating {
		rating = settings.MaxRating
	}

	complexityTier := complexityScale(complexityScore)
	lineCount := settings.MinLineCount + complexityTier*18
	if lineCount > 1800 {
		lineCount = 1800
	}
	averageEstimate := 20 + complexityScore/4
	profile := models.DifficultyProfile{
		Rating:             rating,
		ComplexityScore:    complexityScore,
		LineCount:          lineCount,
		DependencyDepth:    settings.MinDepth + complexityTier/2,
		BranchingFactor:    settings.MinBranching + complexityTier/8,
		FalseRouteRate:     boundedFloat(settings.MinFalseRoutePct+float64(complexityTier)*0.006, settings.MinFalseRoutePct, 0.75),
		DependencyTrees:    settings.MinBranching + complexityTier/6,
		CrossDependencies:  complexityTier / 3,
		NoiseFactor:        boundedFloat(float64(complexityTier)*0.01, 0, 0.65),
		DeadEndFactor:      boundedFloat(float64(complexityTier)*0.008, 0, 0.5),
		HumanSolveEstimate: averageEstimate,
		ExpectedSolve:      solvePercentiles(averageEstimate, complexityScore),
		Source:             source,
	}
	if profile.DependencyDepth > 80 {
		profile.DependencyDepth = 80
	}
	if profile.BranchingFactor > 24 {
		profile.BranchingFactor = 24
	}
	if profile.DependencyTrees > 64 {
		profile.DependencyTrees = 64
	}
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
	return profile
}

func solvePercentiles(averageEstimate int, complexityScore int) models.SolvePercentiles {
	topOne := averageEstimate / 4
	topTen := (averageEstimate * 2) / 5
	if complexityScore > 1500 {
		topOne = averageEstimate / 5
		topTen = averageEstimate / 3
	}
	if complexityScore > 3000 {
		topOne = averageEstimate / 6
		topTen = averageEstimate / 4
	}
	if topOne < 3 {
		topOne = 3
	}
	if topTen < topOne {
		topTen = topOne
	}
	return models.SolvePercentiles{
		TopOnePercentSeconds: topOne,
		TopTenPercentSeconds: topTen,
		AverageSeconds:       averageEstimate,
	}
}

func GenerateLinePuzzleFromProfile(seed string, profile models.DifficultyProfile) []models.ArrowLine {
	return generateLinePuzzle(seed, profile)
}

func GenerateSolvedLinePuzzleFromProfile(seed string, profile models.DifficultyProfile) ([]models.ArrowLine, []string) {
	return generateSolvedLinePuzzle(seed, profile)
}

func legacyDifficultyProfile(count int, dependencyDepth int) models.DifficultyProfile {
	if count < 24 {
		count = 24
	}
	if dependencyDepth < 1 {
		dependencyDepth = 1
	}
	rating := (count - 35) * 100 / (420 - 35)
	if rating < 1 {
		rating = 1
	}
	if rating > 100 {
		rating = 100
	}
	return models.DifficultyProfile{
		Rating:          rating,
		ComplexityScore: count * dependencyDepth,
		LineCount:       count,
		DependencyDepth: dependencyDepth,
		BranchingFactor: 2,
		FalseRouteRate:  0.08,
		DependencyTrees: 2,
		Source:          "legacy",
	}
}

func complexityBand(complexityScore int) int {
	switch {
	case complexityScore <= 100:
		return 10
	case complexityScore <= 300:
		return 25 + (complexityScore-100)/14
	case complexityScore <= 800:
		return 40 + (complexityScore-300)/20
	case complexityScore <= 1500:
		return 65 + (complexityScore-800)/35
	case complexityScore <= 3000:
		return 85 + (complexityScore-1500)/100
	default:
		return 100
	}
}

func complexityScale(complexityScore int) int {
	scale := 0
	remaining := complexityScore
	step := 80
	for remaining > 0 {
		scale++
		remaining -= step
		if step < 600 {
			step += 20
		}
	}
	return scale
}

func boundedFloat(value float64, min float64, max float64) float64 {
	if value < min {
		return min
	}
	if value > max {
		return max
	}
	return value
}

func leagueDifficultyBonus(league string) int {
	switch league {
	case "Silver":
		return 6
	case "Gold":
		return 14
	case "Platinum":
		return 24
	case "Diamond":
		return 34
	case "Master":
		return 44
	case "Legend":
		return 54
	default:
		return 0
	}
}

func houseDifficultyBonus(tier string) int {
	switch tier {
	case "silver":
		return 8
	case "gold":
		return 16
	case "platinum":
		return 26
	case "diamond":
		return 38
	default:
		return 0
	}
}

func tournamentDifficultyBonus(tier string) int {
	switch tier {
	case "weekly":
		return 8
	case "monthly":
		return 16
	case "seasonal":
		return 28
	case "championship":
		return 42
	default:
		return 4
	}
}

func seasonalDifficultyBonus(tier string) int {
	switch tier {
	case "contender":
		return 5
	case "elite":
		return 10
	case "finals":
		return 18
	default:
		return 0
	}
}

func scaleInt(rating int, minRating int, maxRating int, min int, max int) int {
	if rating <= minRating {
		return min
	}
	if rating >= maxRating {
		return max
	}
	return min + ((max-min)*(rating-minRating))/(maxRating-minRating)
}

func scaleFloat(rating int, minRating int, maxRating int, min float64, max float64) float64 {
	if rating <= minRating {
		return min
	}
	if rating >= maxRating {
		return max
	}
	return min + (max-min)*(float64(rating-minRating)/float64(maxRating-minRating))
}
