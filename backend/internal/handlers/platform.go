package handlers

import (
	"encoding/json"
	"net/http"

	"skill-arena/internal/config"
	"skill-arena/internal/db"
	"skill-arena/internal/game"
	"skill-arena/internal/id"
	"skill-arena/internal/models"
)

func PlatformStatsHandler(store *db.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		stats, err := store.PlatformStats(r.Context())
		if err != nil {
			WriteMappedError(w, http.StatusInternalServerError, err)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(stats)
	}
}

func PlatformPuzzlePreviewHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		previewID := id.New("preview")
		profile := game.ProfileFromComplexity(180, 28, "landing_preview")
		profile.LineCount = 56
		profile.DependencyDepth = 7
		profile.BranchingFactor = 3
		profile.DependencyTrees = 5
		profile.FalseRouteRate = 0.22
		version := game.CurrentPuzzleVersion()
		derivation, err := game.DerivePuzzleSeed(config.Runtime().Security.PuzzleSecret, game.SeedDerivationInput{
			Purpose:           "landing_preview",
			MatchID:           previewID,
			PlayerID:          "public_preview",
			DifficultyProfile: profile,
			PuzzleVersion:     version,
		})
		if err != nil {
			WriteMappedError(w, http.StatusInternalServerError, err)
			return
		}
		lines := game.GenerateLinePuzzleFromProfile(derivation.Seed, profile)
		animation, blocked, retry := previewAnimation(lines)
		response := models.PuzzlePreview{
			Lines:             lines,
			AnimationLineIDs:  animation,
			BlockedAttemptID:  blocked,
			UnlockedRetryID:   retry,
			DifficultyProfile: profile,
			PuzzleVersion:     version,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}
}

func previewAnimation(lines []models.ArrowLine) ([]string, string, string) {
	if len(lines) == 0 {
		return nil, "", ""
	}
	removed := map[string]bool{}
	animation := make([]string, 0, 9)
	blocked := ""
	retry := ""
	for _, line := range lines {
		if len(line.DependsOn) == 0 {
			animation = append(animation, line.ID)
			removed[line.ID] = true
			if len(animation) >= 3 {
				break
			}
		}
	}
	for _, line := range lines {
		if len(line.DependsOn) == 0 {
			continue
		}
		for _, dependency := range line.DependsOn {
			if !removed[dependency] {
				blocked = line.ID
				animation = append(animation, dependency)
				removed[dependency] = true
				retry = line.ID
				break
			}
		}
		if blocked != "" {
			break
		}
	}
	for _, line := range lines {
		if len(animation) >= 8 {
			break
		}
		if removed[line.ID] || line.ID == blocked {
			continue
		}
		ready := true
		for _, dependency := range line.DependsOn {
			if !removed[dependency] {
				ready = false
				break
			}
		}
		if ready {
			animation = append(animation, line.ID)
			removed[line.ID] = true
		}
	}
	return animation, blocked, retry
}
