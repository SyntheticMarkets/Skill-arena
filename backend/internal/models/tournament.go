package models

import "time"

type Tournament struct {
	ID              string    `json:"id"`
	Name            string    `json:"name"`
	Type            string    `json:"type"`
	Status          string    `json:"status"`
	EntryFee        float64   `json:"entryFee"`
	WalletType      string    `json:"walletType"`
	PrizePool       float64   `json:"prizePool"`
	MinimumLevel    int       `json:"minimumLevel"`
	MinimumTrust    float64   `json:"minimumTrust"`
	MaxParticipants int       `json:"maxParticipants"`
	StartsAt        time.Time `json:"startsAt"`
	EndsAt          time.Time `json:"endsAt"`
	CreatedAt       time.Time `json:"createdAt"`
	Description     string    `json:"description"`
}

type TournamentParticipant struct {
	ID           string    `json:"id"`
	TournamentID string    `json:"tournamentId"`
	UserID       string    `json:"userId"`
	Username     string    `json:"username,omitempty"`
	DisplayName  string    `json:"displayName,omitempty"`
	Seed         int       `json:"seed"`
	Status       string    `json:"status"`
	RegisteredAt time.Time `json:"registeredAt"`
}

type TournamentMatch struct {
	ID                string             `json:"id"`
	TournamentID      string             `json:"tournamentId"`
	Round             int                `json:"round"`
	MatchNumber       int                `json:"matchNumber"`
	PlayerAID         string             `json:"playerAId,omitempty"`
	PlayerBID         string             `json:"playerBId,omitempty"`
	WinnerID          string             `json:"winnerId,omitempty"`
	Status            string             `json:"status"`
	PrizeSettled      bool               `json:"prizeSettled"`
	DifficultyRating  int                `json:"difficultyRating,omitempty"`
	DifficultyProfile *DifficultyProfile `json:"difficultyProfile,omitempty"`
	PuzzleVersion     PuzzleVersion      `json:"puzzleVersion,omitempty"`
	PlayerASeed       string             `json:"playerASeed,omitempty"`
	PlayerBSeed       string             `json:"playerBSeed,omitempty"`
	PlayerANonce      string             `json:"playerANonce,omitempty"`
	PlayerBNonce      string             `json:"playerBNonce,omitempty"`
	PlayerAHash       string             `json:"playerAGenerationHash,omitempty"`
	PlayerBHash       string             `json:"playerBGenerationHash,omitempty"`
	PlayerALines      []ArrowLine        `json:"playerALines,omitempty"`
	PlayerBLines      []ArrowLine        `json:"playerBLines,omitempty"`
	CreatedAt         time.Time          `json:"createdAt"`
	CompletedAt       *time.Time         `json:"completedAt,omitempty"`
}

type TournamentSubmission struct {
	ID              string       `json:"id"`
	TournamentID    string       `json:"tournamentId"`
	MatchID         string       `json:"matchId"`
	UserID          string       `json:"userId"`
	Clicks          []ArrowClick `json:"clicks,omitempty"`
	IsComplete      bool         `json:"isComplete"`
	MoveCount       int          `json:"moveCount"`
	DurationSeconds float64      `json:"durationSeconds"`
	SubmittedAt     time.Time    `json:"submittedAt"`
}

type TournamentDetail struct {
	Tournament   *Tournament              `json:"tournament"`
	Registered   bool                     `json:"registered"`
	Participants []*TournamentParticipant `json:"participants"`
	Matches      []*TournamentMatch       `json:"matches"`
	Submissions  []*TournamentSubmission  `json:"submissions,omitempty"`
}
