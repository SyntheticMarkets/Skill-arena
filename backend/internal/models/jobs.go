package models

import "time"

const (
	JobReplayExport           = "replay_export"
	JobEmailSend              = "email_send"
	JobBackupRun              = "backup_run"
	JobLeaderboardRecalculate = "leaderboard_recalculate"
	JobTournamentRewardPayout = "tournament_reward_payout"
	JobTelemetryAggregation   = "telemetry_aggregation"
)

const (
	JobStatusQueued    = "queued"
	JobStatusRunning   = "running"
	JobStatusCompleted = "completed"
	JobStatusFailed    = "failed"
	JobStatusCancelled = "cancelled"
)

type BackgroundJob struct {
	ID             string            `json:"id"`
	Type           string            `json:"type"`
	Status         string            `json:"status"`
	Payload        map[string]string `json:"payload,omitempty"`
	Attempts       int               `json:"attempts"`
	MaxAttempts    int               `json:"maxAttempts"`
	RunAfter       time.Time         `json:"runAfter"`
	StartedAt      *time.Time        `json:"startedAt,omitempty"`
	CompletedAt    *time.Time        `json:"completedAt,omitempty"`
	LastError      string            `json:"lastError,omitempty"`
	Worker         string            `json:"worker,omitempty"`
	ResultArtifact string            `json:"resultArtifact,omitempty"`
	CreatedAt      time.Time         `json:"createdAt"`
	UpdatedAt      time.Time         `json:"updatedAt"`
}

type JobQueueStats struct {
	PendingJobs              int               `json:"pendingJobs"`
	RunningJobs              int               `json:"runningJobs"`
	CompletedJobs            int               `json:"completedJobs"`
	FailedJobs               int               `json:"failedJobs"`
	CancelledJobs            int               `json:"cancelledJobs"`
	RetryCount               int               `json:"retryCount"`
	AverageProcessingSeconds float64           `json:"averageProcessingSeconds"`
	ByType                   map[string]int    `json:"byType"`
	WorkerStatus             map[string]string `json:"workerStatus,omitempty"`
	UpdatedAt                time.Time         `json:"updatedAt"`
}

type WorkerHealth struct {
	Name      string    `json:"name"`
	Status    string    `json:"status"`
	LastSeen  time.Time `json:"lastSeen"`
	LastError string    `json:"lastError,omitempty"`
}

type BackupRecord struct {
	ID         string    `json:"id"`
	Type       string    `json:"type"`
	Status     string    `json:"status"`
	Path       string    `json:"path"`
	Verified   bool      `json:"verified"`
	SizeBytes  int64     `json:"sizeBytes"`
	StartedAt  time.Time `json:"startedAt"`
	FinishedAt time.Time `json:"finishedAt"`
	Error      string    `json:"error,omitempty"`
}
