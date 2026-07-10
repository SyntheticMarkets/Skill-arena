package models

import "time"

type SystemHealth struct {
	APIStatus             string            `json:"apiStatus"`
	DatabaseStatus        string            `json:"databaseStatus"`
	QueueStatus           string            `json:"queueStatus"`
	WebSocketConnections  int               `json:"webSocketConnections"`
	CacheHealth           string            `json:"cacheHealth"`
	StorageUsageBytes     int64             `json:"storageUsageBytes"`
	ReplayGenerationQueue int               `json:"replayGenerationQueue"`
	ActiveMatches         int               `json:"activeMatches"`
	PlayersOnline         int               `json:"playersOnline"`
	CPUUsagePercent       float64           `json:"cpuUsagePercent"`
	MemoryUsageBytes      uint64            `json:"memoryUsageBytes"`
	BackupStatus          string            `json:"backupStatus"`
	DeploymentVersion     string            `json:"deploymentVersion"`
	MaintenanceEnabled    bool              `json:"maintenanceEnabled"`
	MaintenanceMessage    string            `json:"maintenanceMessage,omitempty"`
	WorkerHealth          map[string]string `json:"workerHealth,omitempty"`
	QueueStats            *JobQueueStats    `json:"queueStats,omitempty"`
	CheckedAt             time.Time         `json:"checkedAt"`
}
