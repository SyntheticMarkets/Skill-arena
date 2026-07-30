package db

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"skill-arena/internal/config"
	"skill-arena/internal/models"
)

func TestPostgresBackgroundJobLifecycleAndRestart(t *testing.T) {
	databaseURL := os.Getenv("SKILL_ARENA_TEST_POSTGRES_URL")
	if databaseURL == "" {
		t.Skip("SKILL_ARENA_TEST_POSTGRES_URL is not configured")
	}
	ctx := context.Background()
	root := t.TempDir()
	open := func() *Store {
		store, err := NewWithOptions(ctx, Options{
			DatabaseURL: databaseURL,
			Environment: "development",
			Storage:     config.StorageSettings{LocalRoot: root},
		})
		if err != nil {
			t.Fatal(err)
		}
		return store
	}

	store := open()
	if _, err := store.pg.ExecContext(ctx, `TRUNCATE background_jobs`); err != nil {
		t.Fatal(err)
	}
	job, err := store.EnqueueJob(
		ctx, models.JobRealtimeReplayPersist,
		map[string]string{"matchId": "match-1"}, time.Now().UTC(),
	)
	if err != nil {
		t.Fatal(err)
	}
	if err := store.Close(ctx); err != nil {
		t.Fatal(err)
	}

	store = open()
	t.Cleanup(func() { _ = store.Close(context.Background()) })
	jobs, err := store.ListJobs(ctx, models.JobStatusQueued)
	if err != nil || len(jobs) != 1 || jobs[0].ID != job.ID {
		t.Fatalf("reloaded jobs=%+v err=%v", jobs, err)
	}
	claimed, err := store.ClaimNextJob(
		ctx, "worker-1", []string{models.JobRealtimeReplayPersist}, time.Now().UTC(),
	)
	if err != nil || claimed == nil || claimed.Status != models.JobStatusRunning ||
		claimed.Attempts != 1 {
		t.Fatalf("claimed=%+v err=%v", claimed, err)
	}
	if err := store.FailJob(ctx, claimed.ID, errors.New("retry me")); err != nil {
		t.Fatal(err)
	}
	retried, err := store.RetryJob(ctx, claimed.ID)
	if err != nil || retried.Status != models.JobStatusQueued {
		t.Fatalf("retried=%+v err=%v", retried, err)
	}
	claimed, err = store.ClaimNextJob(
		ctx, "worker-2", []string{models.JobRealtimeReplayPersist}, time.Now().UTC(),
	)
	if err != nil || claimed == nil || claimed.Attempts != 2 {
		t.Fatalf("claimed again=%+v err=%v", claimed, err)
	}
	if err := store.CompleteJob(ctx, claimed.ID, "replays/match-1.json"); err != nil {
		t.Fatal(err)
	}
	completed, err := store.ListJobs(ctx, models.JobStatusCompleted)
	if err != nil || len(completed) != 1 ||
		completed[0].ResultArtifact != "replays/match-1.json" {
		t.Fatalf("completed=%+v err=%v", completed, err)
	}
}
