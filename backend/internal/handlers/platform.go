package handlers

import (
	"encoding/json"
	"net/http"

	"skill-arena/internal/config"
	"skill-arena/internal/db"
	"skill-arena/internal/game"
	"skill-arena/internal/game/puzzle"
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
		service := puzzle.NewService(config.Runtime().Security.PuzzleSecret, nil)
		generated, err := service.Generate(r.Context(), puzzle.Request{
			Mode:              puzzle.ModeLandingPreview,
			Purpose:           "landing_preview",
			MatchID:           previewID,
			PlayerID:          "public_preview",
			Shared:            false,
			DifficultyProfile: profile,
			PuzzleVersion:     version,
		})
		if err != nil {
			WriteMappedError(w, http.StatusInternalServerError, err)
			return
		}
		animation, blocked, retry := previewAnimation(generated.Lines, generated.Solution)
		response := models.PuzzlePreview{
			Lines:             generated.Lines,
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

func previewAnimation(lines []models.ArrowLine, solution []string) ([]string, string, string) {
	if len(lines) == 0 {
		return nil, "", ""
	}
	animation := make([]string, 0, 9)
	for _, lineID := range solution {
		animation = append(animation, lineID)
		if len(animation) >= 8 {
			break
		}
	}
	for _, line := range lines {
		if len(animation) >= 8 {
			break
		}
		if !containsLineID(animation, line.ID) {
			animation = append(animation, line.ID)
		}
	}
	return animation, "", ""
}

func containsLineID(lineIDs []string, candidate string) bool {
	for _, lineID := range lineIDs {
		if lineID == candidate {
			return true
		}
	}
	return false
}
