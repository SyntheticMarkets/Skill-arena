package models

type Wallet struct {
	UserID             string  `json:"userId"`
	LiveBalance        float64 `json:"liveBalance"`
	LiveLockedBalance  float64 `json:"liveLockedBalance"`
	DemoBalance        float64 `json:"demoBalance"`
	DemoLockedBalance  float64 `json:"demoLockedBalance"`
	PendingWithdrawals float64 `json:"pendingWithdrawals"`
	BonusBalance       float64 `json:"bonusBalance"`
}
