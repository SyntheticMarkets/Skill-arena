package models

import "time"

type Season struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Theme       string    `json:"theme"`
	StartsAt    time.Time `json:"startsAt"`
	EndsAt      time.Time `json:"endsAt"`
	IsActive    bool      `json:"isActive"`
	RewardPool  float64   `json:"rewardPool"`
	Description string    `json:"description"`
}

type SeasonLeaderboardEntry struct {
	UserID       string  `json:"userId"`
	Username     string  `json:"username"`
	DisplayName  string  `json:"displayName"`
	LeagueTier   string  `json:"leagueTier"`
	SeasonPoints int     `json:"seasonPoints"`
	Wins         int     `json:"wins"`
	Losses       int     `json:"losses"`
	TrustScore   float64 `json:"trustScore"`
	Rank         int     `json:"rank"`
}

type AchievementCatalogItem struct {
	Code        string `json:"code"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Category    string `json:"category"`
}
