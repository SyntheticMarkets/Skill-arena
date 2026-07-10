package models

type HouseTier struct {
	ID              string  `json:"id"`
	Name            string  `json:"name"`
	MinimumLevel    int     `json:"minimumLevel"`
	MinimumTrust    float64 `json:"minimumTrust"`
	Stake           float64 `json:"stake"`
	RewardRate      float64 `json:"rewardRate"`
	Difficulty      int     `json:"difficulty"`
	TargetHouseEdge float64 `json:"targetHouseEdge"`
	Description     string  `json:"description"`
}
