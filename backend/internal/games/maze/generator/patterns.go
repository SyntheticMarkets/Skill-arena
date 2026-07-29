package generator

import (
	"errors"
	"strings"
)

type Pattern struct {
	ID             string `json:"id"`
	Version        int    `json:"version"`
	MinimumColumns int    `json:"minimumColumns"`
	MinimumRows    int    `json:"minimumRows"`
	Weight         int    `json:"weight"`
}

var patternCatalogueV1 = []Pattern{
	{ID: "braid", Version: 1, MinimumColumns: 8, MinimumRows: 8, Weight: 120},
	{ID: "spiral", Version: 1, MinimumColumns: 9, MinimumRows: 9, Weight: 105},
	{ID: "maze_rows", Version: 1, MinimumColumns: 8, MinimumRows: 8, Weight: 115},
	{ID: "rings", Version: 1, MinimumColumns: 10, MinimumRows: 10, Weight: 90},
	{ID: "mosaic", Version: 1, MinimumColumns: 8, MinimumRows: 8, Weight: 110},
	{ID: "piton", Version: 1, MinimumColumns: 8, MinimumRows: 8, Weight: 85},
	{ID: "diagonal_weave", Version: 1, MinimumColumns: 9, MinimumRows: 9, Weight: 95},
	{ID: "rays", Version: 1, MinimumColumns: 8, MinimumRows: 8, Weight: 100},
}

func SelectPattern(stream *RandomStream, profile DifficultyProfile, columns, rows int) (Pattern, error) {
	if stream == nil {
		return Pattern{}, errors.New("pattern random stream is required")
	}
	eligible := make([]Pattern, 0, len(patternCatalogueV1))
	totalWeight := 0
	for _, pattern := range patternCatalogueV1 {
		if columns < pattern.MinimumColumns || rows < pattern.MinimumRows {
			continue
		}
		weight := pattern.Weight
		bias := strings.ToLower(strings.TrimSpace(profile.PatternBias))
		if bias != "" && bias != "balanced" && bias != "any" {
			if bias == pattern.ID {
				weight *= 8
			} else {
				weight = maxInt(1, weight/8)
			}
		}
		pattern.Weight = weight
		totalWeight += weight
		eligible = append(eligible, pattern)
	}
	if len(eligible) == 0 || totalWeight <= 0 {
		return Pattern{}, errors.New("no pattern is eligible for the board dimensions")
	}
	selected, err := stream.Uint64n(uint64(totalWeight))
	if err != nil {
		return Pattern{}, err
	}
	cursor := int(selected)
	for _, pattern := range eligible {
		if cursor < pattern.Weight {
			return pattern, nil
		}
		cursor -= pattern.Weight
	}
	return eligible[len(eligible)-1], nil
}

func maxInt(first, second int) int {
	if first > second {
		return first
	}
	return second
}
