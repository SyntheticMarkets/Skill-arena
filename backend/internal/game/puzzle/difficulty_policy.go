package puzzle

import "skill-arena/internal/models"

func normalizeProfile(req Request) models.DifficultyProfile {
	profile := req.DifficultyProfile
	maxLines := 120
	switch req.Mode {
	case ModePvP, ModeRanked, ModeTournament, ModeDailyChallenge:
		maxLines = 96
	case ModeLandingPreview:
		maxLines = 64
	}
	if req.Level >= 20 {
		maxLines = 160
	} else if req.Level >= 12 && maxLines < 120 {
		maxLines = 120
	} else if req.Level > 0 && req.Level <= 5 && maxLines > 64 {
		maxLines = 64
	}
	if profile.LineCount > maxLines {
		profile.LineCount = maxLines
	}
	if profile.LineCount < 24 {
		profile.LineCount = 24
	}
	if profile.DependencyDepth > profile.LineCount/3 {
		profile.DependencyDepth = profile.LineCount / 3
	}
	if profile.DependencyDepth < 1 {
		profile.DependencyDepth = 1
	}
	if profile.BranchingFactor > 8 {
		profile.BranchingFactor = 8
	}
	if profile.BranchingFactor < 1 {
		profile.BranchingFactor = 1
	}
	if profile.DependencyTrees > 8 {
		profile.DependencyTrees = 8
	}
	if profile.DependencyTrees < 1 {
		profile.DependencyTrees = 1
	}
	return profile
}
