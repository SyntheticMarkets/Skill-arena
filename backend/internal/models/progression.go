package models

import "time"

type Progression struct {
	UserID        string    `json:"userId"`
	XP            int       `json:"xp"`
	Level         int       `json:"level"`
	Prestige      int       `json:"prestige"`
	EloRating     int       `json:"eloRating"`
	LeagueTier    string    `json:"leagueTier"`
	SeasonPoints  int       `json:"seasonPoints"`
	LegacyPoints  int       `json:"legacyPoints"`
	HouseRep      int       `json:"houseReputation"`
	MatchesPlayed int       `json:"matchesPlayed"`
	Wins          int       `json:"wins"`
	Losses        int       `json:"losses"`
	CurrentStreak int       `json:"currentStreak"`
	BestMoves     int       `json:"bestMoves,omitempty"`
	TrustScore    float64   `json:"trustScore"`
	TrustTier     string    `json:"trustTier"`
	UpdatedAt     time.Time `json:"updatedAt"`
}

type Achievement struct {
	ID          string    `json:"id"`
	UserID      string    `json:"userId"`
	Code        string    `json:"code"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	UnlockedAt  time.Time `json:"unlockedAt"`
}
