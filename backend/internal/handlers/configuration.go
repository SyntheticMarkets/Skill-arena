package handlers

import (
	"encoding/json"
	"net/http"

	"skill-arena/internal/config"
)

func FeatureFlagsHandler(settings *config.RuntimeSettings) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(settings.Features)
	}
}
