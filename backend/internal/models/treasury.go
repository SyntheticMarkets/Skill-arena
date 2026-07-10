package models

import "time"

type TreasuryState struct {
	PlayerReserve       float64   `json:"playerReserve"`
	RevenueReserve      float64   `json:"revenueReserve"`
	SeasonReserve       float64   `json:"seasonReserve"`
	ChampionshipReserve float64   `json:"championshipReserve"`
	JackpotReserve      float64   `json:"jackpotReserve"`
	EmergencyReserve    float64   `json:"emergencyReserve"`
	UpdatedAt           time.Time `json:"updatedAt"`
}

type TreasuryHealth struct {
	PlayerLiabilities float64       `json:"playerLiabilities"`
	TotalReserves     float64       `json:"totalReserves"`
	CoverageRatio     float64       `json:"coverageRatio"`
	IsSolvent         bool          `json:"isSolvent"`
	HouseExposure     float64       `json:"houseExposure"`
	State             TreasuryState `json:"state"`
}

type HouseRiskReport struct {
	TierID            string  `json:"tierId"`
	Attempts          int     `json:"attempts"`
	Wins              int     `json:"wins"`
	Losses            int     `json:"losses"`
	PlayerWinRate     float64 `json:"playerWinRate"`
	TargetHouseEdge   float64 `json:"targetHouseEdge"`
	RecommendedAction string  `json:"recommendedAction"`
}
