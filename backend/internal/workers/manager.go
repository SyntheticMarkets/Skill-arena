package workers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"skill-arena/internal/config"
	"skill-arena/internal/db"
	"skill-arena/internal/id"
	"skill-arena/internal/models"
)

const (
	WorkerReplay      = "replay_worker"
	WorkerEmail       = "email_worker"
	WorkerLeaderboard = "leaderboard_worker"
	WorkerTournament  = "tournament_worker"
	WorkerTelemetry   = "telemetry_worker"
	WorkerBackup      = "backup_worker"
)

type Manager struct {
	store  *db.Store
	cfg    *config.Config
	cancel context.CancelFunc
	wg     sync.WaitGroup
}

func NewManager(store *db.Store, cfg *config.Config) *Manager {
	return &Manager{store: store, cfg: cfg}
}

func (m *Manager) Start(ctx context.Context) {
	if m.cfg.Settings != nil && !m.cfg.Settings.Workers.Enabled {
		return
	}
	ctx, cancel := context.WithCancel(ctx)
	m.cancel = cancel

	m.startWorker(ctx, WorkerReplay, []string{models.JobReplayExport}, m.processReplay)
	m.startWorker(ctx, WorkerEmail, []string{models.JobEmailSend}, m.processEmail)
	m.startWorker(ctx, WorkerLeaderboard, []string{models.JobLeaderboardRecalculate}, m.processLeaderboard)
	m.startWorker(ctx, WorkerTournament, []string{models.JobTournamentRewardPayout}, m.processTournament)
	m.startWorker(ctx, WorkerTelemetry, []string{models.JobTelemetryAggregation}, m.processTelemetry)
	m.startWorker(ctx, WorkerBackup, []string{models.JobBackupRun}, m.processBackup)
	m.startBackupScheduler(ctx)
}

func (m *Manager) Shutdown(ctx context.Context) error {
	if m.cancel != nil {
		m.cancel()
	}
	done := make(chan struct{})
	go func() {
		m.wg.Wait()
		close(done)
	}()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-done:
		return nil
	}
}

func (m *Manager) startWorker(ctx context.Context, name string, jobTypes []string, processor func(context.Context, *models.BackgroundJob) (string, error)) {
	m.store.SetWorkerHealth(ctx, name, "running", "")
	m.wg.Add(1)
	go func() {
		defer m.wg.Done()
		defer m.store.SetWorkerHealth(context.Background(), name, "stopped", "")

		interval := time.Duration(m.cfg.Settings.Workers.PollSeconds) * time.Second
		if interval <= 0 {
			interval = 5 * time.Second
		}
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for {
			if err := m.runOnce(ctx, name, jobTypes, processor); err != nil {
				m.store.SetWorkerHealth(ctx, name, "failed", err.Error())
			} else {
				m.store.SetWorkerHealth(ctx, name, "running", "")
			}

			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
			}
		}
	}()
}

func (m *Manager) runOnce(ctx context.Context, name string, jobTypes []string, processor func(context.Context, *models.BackgroundJob) (string, error)) error {
	job, err := m.store.ClaimNextJob(ctx, name, jobTypes, time.Now().UTC())
	if err != nil || job == nil {
		return err
	}
	artifact, err := processor(ctx, job)
	if err != nil {
		_ = m.store.FailJob(ctx, job.ID, err)
		return err
	}
	return m.store.CompleteJob(ctx, job.ID, artifact)
}

func (m *Manager) startBackupScheduler(ctx context.Context) {
	m.wg.Add(1)
	go func() {
		defer m.wg.Done()
		ticker := time.NewTicker(time.Hour)
		defer ticker.Stop()
		for {
			m.enqueueScheduledBackups(ctx, time.Now().UTC())
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
			}
		}
	}()
}

func (m *Manager) enqueueScheduledBackups(ctx context.Context, now time.Time) {
	hour := m.cfg.Settings.Workers.BackupHourUTC
	if hour < 0 || hour > 23 {
		hour = 2
	}
	if now.Hour() != hour {
		return
	}
	dateKey := now.Format("2006-01-02")
	_, _ = m.store.EnqueueJob(ctx, models.JobBackupRun, map[string]string{"schedule": "daily", "date": dateKey}, now)
	if now.Weekday() == time.Monday {
		_, _ = m.store.EnqueueJob(ctx, models.JobBackupRun, map[string]string{"schedule": "weekly", "date": dateKey}, now)
	}
	if now.Day() == 1 {
		_, _ = m.store.EnqueueJob(ctx, models.JobBackupRun, map[string]string{"schedule": "monthly", "date": dateKey}, now)
	}
}

func (m *Manager) processReplay(ctx context.Context, job *models.BackgroundJob) (string, error) {
	payload := map[string]string{
		"jobId":     job.ID,
		"sessionId": job.Payload["sessionId"],
		"matchId":   job.Payload["matchId"],
		"status":    "replay package exported",
	}
	return m.writeObjectArtifact(ctx, "replays/exports", job.ID+".json", payload)
}

func (m *Manager) processEmail(ctx context.Context, job *models.BackgroundJob) (string, error) {
	if job.Payload["to"] == "" && job.Payload["userId"] == "" {
		return "", errors.New("email job requires to or userId")
	}
	return writeArtifact(m.store.DataDir(), "email_outbox", job.ID+".json", map[string]any{
		"jobId":   job.ID,
		"type":    job.Payload["template"],
		"payload": job.Payload,
		"queued":  time.Now().UTC(),
	})
}

func (m *Manager) processLeaderboard(ctx context.Context, job *models.BackgroundJob) (string, error) {
	stats, err := m.store.QueueStats(ctx)
	if err != nil {
		return "", err
	}
	return m.writeObjectArtifact(ctx, "exports/analytics", "leaderboard-"+job.ID+".json", map[string]any{
		"jobId":      job.ID,
		"queueStats": stats,
		"calculated": time.Now().UTC(),
	})
}

func (m *Manager) processTournament(ctx context.Context, job *models.BackgroundJob) (string, error) {
	return m.writeObjectArtifact(ctx, "exports/tournament_jobs", job.ID+".json", map[string]any{
		"jobId":   job.ID,
		"payload": job.Payload,
		"status":  "tournament progression task recorded",
	})
}

func (m *Manager) processTelemetry(ctx context.Context, job *models.BackgroundJob) (string, error) {
	metrics, err := m.store.Metrics(ctx)
	if err != nil {
		return "", err
	}
	return m.writeObjectArtifact(ctx, "exports/analytics", "telemetry-"+job.ID+".json", map[string]any{
		"jobId":   job.ID,
		"metrics": metrics,
		"created": time.Now().UTC(),
	})
}

func (m *Manager) processBackup(ctx context.Context, job *models.BackgroundJob) (string, error) {
	record, err := RunBackup(ctx, m.store, m.cfg.Settings)
	if err != nil {
		return "", err
	}
	if err := m.store.AddBackupRecord(ctx, record); err != nil {
		return "", err
	}
	snapshot, err := m.store.ExportSnapshotJSON(ctx)
	if err != nil {
		return "", err
	}
	key := "backups/" + filepath.Base(record.Path) + "/store_snapshot.json"
	if err := m.store.ObjectStore().Put(ctx, key, snapshot, "application/json"); err != nil {
		return "", err
	}
	record.Path = key
	_ = m.store.AddBackupRecord(ctx, record)
	return key, nil
}

func (m *Manager) writeObjectArtifact(ctx context.Context, folder, name string, payload any) (string, error) {
	data, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return "", err
	}
	key := strings.Trim(folder, "/") + "/" + sanitizeFileName(name)
	return key, m.store.ObjectStore().Put(ctx, key, data, "application/json")
}

func RunBackup(ctx context.Context, store *db.Store, settings *config.RuntimeSettings) (*models.BackupRecord, error) {
	started := time.Now().UTC()
	if err := store.Close(ctx); err != nil {
		return nil, err
	}
	backupDir := settings.Backup.Directory
	if backupDir == "" {
		backupDir = "./backups"
	}
	if err := os.MkdirAll(backupDir, 0o755); err != nil {
		return nil, err
	}
	target := filepath.Join(backupDir, "skill-arena-"+started.Format("20060102-150405")+"-"+id.New("bak"))
	record := &models.BackupRecord{
		ID:        id.New("bak"),
		Type:      "manual",
		Status:    "running",
		Path:      target,
		StartedAt: started,
	}
	if err := copyDir(ctx, store.DataDir(), target); err != nil {
		record.Status = "failed"
		record.Error = err.Error()
		record.FinishedAt = time.Now().UTC()
		return record, err
	}
	if err := os.MkdirAll(filepath.Join(target, "replay_exports"), 0o755); err != nil {
		record.Status = "failed"
		record.Error = err.Error()
		record.FinishedAt = time.Now().UTC()
		return record, err
	}
	if _, err := writeArtifact(target, ".", "backup_manifest.json", map[string]any{
		"id":        record.ID,
		"startedAt": started,
		"source":    store.DataDir(),
	}); err != nil {
		record.Status = "failed"
		record.Error = err.Error()
		record.FinishedAt = time.Now().UTC()
		return record, err
	}
	size, err := dirSize(target)
	if err != nil {
		record.Status = "failed"
		record.Error = err.Error()
		record.FinishedAt = time.Now().UTC()
		return record, err
	}
	record.SizeBytes = size
	record.Verified = verifyBackup(target)
	record.Status = "completed"
	if !record.Verified {
		record.Status = "verification_failed"
	}
	record.FinishedAt = time.Now().UTC()
	return record, nil
}

func writeArtifact(dataDir, folder, name string, payload any) (string, error) {
	dir := filepath.Join(dataDir, folder)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	path := filepath.Join(dir, sanitizeFileName(name))
	data, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return "", err
	}
	return path, os.WriteFile(path, data, 0o644)
}

func copyDir(ctx context.Context, src, dst string) error {
	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		if rel == "." {
			return os.MkdirAll(dst, 0o755)
		}
		target := filepath.Join(dst, rel)
		if info.IsDir() {
			return os.MkdirAll(target, info.Mode())
		}
		return copyFile(path, target, info.Mode())
	})
}

func copyFile(src, dst string, mode os.FileMode) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}
	out, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, mode)
	if err != nil {
		return err
	}
	defer out.Close()
	_, err = io.Copy(out, in)
	return err
}

func dirSize(path string) (int64, error) {
	var size int64
	err := filepath.Walk(path, func(_ string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			size += info.Size()
		}
		return nil
	})
	return size, err
}

func verifyBackup(path string) bool {
	required := []string{"backup_manifest.json", "jobs.json"}
	for _, name := range required {
		if _, err := os.Stat(filepath.Join(path, name)); err != nil {
			return false
		}
	}
	return true
}

func sanitizeFileName(name string) string {
	name = strings.ReplaceAll(name, "..", "")
	name = strings.ReplaceAll(name, string(filepath.Separator), "-")
	if strings.TrimSpace(name) == "" {
		return fmt.Sprintf("%s.json", id.New("artifact"))
	}
	return name
}
