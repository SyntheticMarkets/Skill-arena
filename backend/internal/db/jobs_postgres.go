package db

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"time"

	"github.com/lib/pq"

	"skill-arena/internal/models"
)

const backgroundJobColumns = `id,type,status,payload,attempts,max_attempts,run_after,
started_at,completed_at,COALESCE(last_error,''),COALESCE(worker,''),
COALESCE(result_artifact,''),created_at,updated_at`

const backgroundJobReturningColumns = `job.id,job.type,job.status,job.payload,
job.attempts,job.max_attempts,job.run_after,job.started_at,job.completed_at,
COALESCE(job.last_error,''),COALESCE(job.worker,''),
COALESCE(job.result_artifact,''),job.created_at,job.updated_at`

func scanBackgroundJob(scanner rowScanner) (*models.BackgroundJob, error) {
	var job models.BackgroundJob
	var payload []byte
	if err := scanner.Scan(
		&job.ID, &job.Type, &job.Status, &payload, &job.Attempts,
		&job.MaxAttempts, &job.RunAfter, &job.StartedAt, &job.CompletedAt,
		&job.LastError, &job.Worker, &job.ResultArtifact, &job.CreatedAt,
		&job.UpdatedAt,
	); err != nil {
		return nil, err
	}
	if len(payload) > 0 {
		if err := json.Unmarshal(payload, &job.Payload); err != nil {
			return nil, err
		}
	}
	if job.Payload == nil {
		job.Payload = map[string]string{}
	}
	return &job, nil
}

func (s *Store) cacheBackgroundJob(job *models.BackgroundJob) {
	if job == nil {
		return
	}
	copyJob := *job
	copyJob.Payload = copyStringMap(job.Payload)
	s.mu.Lock()
	s.jobs[job.ID] = &copyJob
	s.mu.Unlock()
}

func (s *Store) loadPostgresJobs(ctx context.Context) error {
	rows, err := s.pg.QueryContext(ctx, `SELECT `+backgroundJobColumns+`
FROM background_jobs ORDER BY created_at`)
	if err != nil {
		return err
	}
	defer rows.Close()
	loaded := make(map[string]*models.BackgroundJob)
	for rows.Next() {
		job, scanErr := scanBackgroundJob(rows)
		if scanErr != nil {
			return scanErr
		}
		loaded[job.ID] = job
	}
	if err := rows.Err(); err != nil {
		return err
	}
	s.mu.Lock()
	s.jobs = loaded
	s.mu.Unlock()
	return nil
}

func (s *Store) enqueuePostgresJob(
	ctx context.Context,
	job *models.BackgroundJob,
) error {
	payload, err := json.Marshal(job.Payload)
	if err != nil {
		return err
	}
	_, err = s.pg.ExecContext(ctx, `INSERT INTO background_jobs(
id,type,status,payload,attempts,max_attempts,run_after,started_at,completed_at,
last_error,worker,result_artifact,created_at,updated_at)
VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,NULLIF($10,''),NULLIF($11,''),
NULLIF($12,''),$13,$14)`,
		job.ID, job.Type, job.Status, payload, job.Attempts, job.MaxAttempts,
		job.RunAfter, job.StartedAt, job.CompletedAt, job.LastError, job.Worker,
		job.ResultArtifact, job.CreatedAt, job.UpdatedAt,
	)
	return err
}

func (s *Store) listPostgresJobs(
	ctx context.Context,
	status string,
) ([]*models.BackgroundJob, error) {
	rows, err := s.pg.QueryContext(ctx, `SELECT `+backgroundJobColumns+`
FROM background_jobs WHERE $1='' OR status=$1 ORDER BY created_at`, status)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var jobs []*models.BackgroundJob
	for rows.Next() {
		job, scanErr := scanBackgroundJob(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		jobs = append(jobs, job)
	}
	return jobs, rows.Err()
}

func (s *Store) claimPostgresJob(
	ctx context.Context,
	worker string,
	jobTypes []string,
	now time.Time,
) (*models.BackgroundJob, error) {
	tx, err := s.pg.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	row := tx.QueryRowContext(ctx, `WITH candidate AS (
    SELECT id FROM background_jobs
    WHERE status='queued' AND run_after<=$1
      AND (cardinality($2::text[])=0 OR type=ANY($2::text[]))
    ORDER BY created_at
    FOR UPDATE SKIP LOCKED
    LIMIT 1
)
UPDATE background_jobs AS job
SET status='running',worker=$3,attempts=job.attempts+1,started_at=$1,updated_at=$1
FROM candidate
WHERE job.id=candidate.id
RETURNING `+backgroundJobReturningColumns, now, pq.Array(jobTypes), worker)
	job, err := scanBackgroundJob(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return job, nil
}

func (s *Store) completePostgresJob(
	ctx context.Context,
	jobID, artifact string,
	now time.Time,
) (*models.BackgroundJob, error) {
	return scanBackgroundJob(s.pg.QueryRowContext(ctx, `UPDATE background_jobs
SET status='completed',completed_at=$2,result_artifact=NULLIF($3,''),updated_at=$2
WHERE id=$1 RETURNING `+backgroundJobColumns, jobID, now, artifact))
}

func (s *Store) failPostgresJob(
	ctx context.Context,
	jobID, failure string,
	now time.Time,
) (*models.BackgroundJob, error) {
	return scanBackgroundJob(s.pg.QueryRowContext(ctx, `UPDATE background_jobs
SET last_error=$2,updated_at=$3,
status=CASE WHEN attempts<max_attempts THEN 'queued' ELSE 'failed' END,
worker=CASE WHEN attempts<max_attempts THEN NULL ELSE worker END,
run_after=CASE WHEN attempts<max_attempts
    THEN $3::timestamptz+(attempts*attempts)*INTERVAL '1 minute' ELSE run_after END,
started_at=CASE WHEN attempts<max_attempts THEN NULL ELSE started_at END,
completed_at=CASE WHEN attempts<max_attempts THEN NULL ELSE $3 END
WHERE id=$1 RETURNING `+backgroundJobColumns, jobID, failure, now))
}

func (s *Store) retryPostgresJob(
	ctx context.Context,
	jobID string,
	now time.Time,
) (*models.BackgroundJob, error) {
	return scanBackgroundJob(s.pg.QueryRowContext(ctx, `UPDATE background_jobs
SET status='queued',run_after=$2,started_at=NULL,completed_at=NULL,worker=NULL,
last_error=NULL,updated_at=$2
WHERE id=$1 RETURNING `+backgroundJobColumns, jobID, now))
}

func (s *Store) cancelPostgresJob(
	ctx context.Context,
	jobID string,
	now time.Time,
) (*models.BackgroundJob, error) {
	return scanBackgroundJob(s.pg.QueryRowContext(ctx, `UPDATE background_jobs
SET status='cancelled',completed_at=$2,updated_at=$2
WHERE id=$1 RETURNING `+backgroundJobColumns, jobID, now))
}
