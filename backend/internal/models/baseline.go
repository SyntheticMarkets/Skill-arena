package models

import "time"

type BehavioralBaseline struct {
	UserID             string    `json:"userId"`
	CalibrationRuns    int       `json:"calibrationRuns"`
	AverageEfficiency  float64   `json:"averageEfficiency"`
	AverageMoveSeconds float64   `json:"averageMoveSeconds"`
	BestMoveCount      int       `json:"bestMoveCount,omitempty"`
	LastSessionID      string    `json:"lastSessionId,omitempty"`
	LastRunAt          time.Time `json:"lastRunAt"`
	RiskSignal         string    `json:"riskSignal"`
}
