package puzzle

import "skill-arena/internal/models"

func MetadataFromProfile(req Request, derivationHash string, nonce string, seed string, solution []string) models.PuzzleMetadata {
	profile := req.DifficultyProfile
	version := req.PuzzleVersion
	return models.PuzzleMetadata{
		PuzzleID:                 puzzleID(req.Mode, derivationHash),
		Seed:                     seed,
		GenerationHash:           derivationHash,
		Nonce:                    nonce,
		Mode:                     req.Mode,
		Shared:                   req.Shared,
		Difficulty:               profile.Rating,
		Level:                    req.Level,
		BranchCount:              profile.BranchingFactor,
		DependencyDepth:          profile.DependencyDepth,
		MinimumMoves:             len(solution),
		ExpectedSolveTimeSeconds: profile.HumanSolveEstimate,
		ComplexityScore:          profile.ComplexityScore,
		TrustScore:               req.TrustScore,
		GeneratorVersion:         version.GeneratorVersion,
		SolverVersion:            version.GameRulesVersion,
		PuzzleVersion:            version,
	}
}
