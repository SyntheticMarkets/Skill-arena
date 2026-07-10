package workers

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"skill-arena/internal/config"
	"skill-arena/internal/db"
	"skill-arena/internal/models"
)

func TestBackupAndRecoveryValidation(t *testing.T) {
	ctx := context.Background()
	store, err := db.New(ctx, t.TempDir())
	if err != nil {
		t.Fatalf("new store: %v", err)
	}
	user := models.NewUser("", "backup@example.com", "hash")
	if err := store.CreateUser(ctx, user); err != nil {
		t.Fatalf("create user: %v", err)
	}
	settings := config.LoadRuntimeSettings()
	settings.Backup.Directory = t.TempDir()

	record, err := RunBackup(ctx, store, settings)
	if err != nil {
		t.Fatalf("run backup: %v", err)
	}
	if !record.Verified || record.Status != "completed" {
		t.Fatalf("backup record = %#v, want verified completed", record)
	}

	report, err := ValidateRecovery(ctx, record.Path)
	if err != nil {
		t.Fatalf("validate recovery: %v", err)
	}
	if !report.Passed {
		t.Fatalf("recovery report = %#v, want passed", report)
	}
}

func TestBackupDeleteAndRestoreIntegrity(t *testing.T) {
	ctx := context.Background()
	sourceDir := t.TempDir()
	store, err := db.New(ctx, sourceDir)
	if err != nil {
		t.Fatalf("new store: %v", err)
	}
	user := models.NewUser("", "restore@example.com", "hash")
	if err := store.CreateUser(ctx, user); err != nil {
		t.Fatalf("create user: %v", err)
	}
	if _, err := store.RecordWalletTransaction(ctx, user.ID, models.TransactionTypeDeposit, 75, "USD", "restore-funding", nil); err != nil {
		t.Fatalf("fund wallet: %v", err)
	}

	settings := config.LoadRuntimeSettings()
	settings.Backup.Directory = t.TempDir()
	record, err := RunBackup(ctx, store, settings)
	if err != nil {
		t.Fatalf("run backup: %v", err)
	}
	if err := os.RemoveAll(sourceDir); err != nil {
		t.Fatalf("delete source data dir: %v", err)
	}

	restored, err := db.New(ctx, record.Path)
	if err != nil {
		t.Fatalf("restore store from backup: %v", err)
	}
	restoredUser, err := restored.GetUserByID(ctx, user.ID)
	if err != nil {
		t.Fatalf("restored user: %v", err)
	}
	if restoredUser.Email != user.Email {
		t.Fatalf("restored email = %q, want %q", restoredUser.Email, user.Email)
	}
	wallet, err := restored.GetWalletByUserID(ctx, user.ID)
	if err != nil {
		t.Fatalf("restored wallet: %v", err)
	}
	if wallet.LiveBalance != 75 {
		t.Fatalf("restored live balance = %.2f, want 75", wallet.LiveBalance)
	}
	report, err := ValidateRecovery(ctx, record.Path)
	if err != nil {
		t.Fatalf("validate recovery: %v", err)
	}
	if !report.Passed {
		t.Fatalf("recovery report = %#v, want passed", report)
	}
}

func TestValidateRecoveryFailsWhenJobQueueMissing(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	for _, name := range []string{"users.json", "wallets.json", "sessions.json", "metrics.json", "replays.json"} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte("[]"), 0o644); err != nil {
			t.Fatalf("write %s: %v", name, err)
		}
	}

	report, err := ValidateRecovery(ctx, dir)
	if err != nil {
		t.Fatalf("validate recovery: %v", err)
	}
	if report.Passed || report.JobQueueRestore {
		t.Fatalf("recovery report = %#v, want failed job queue restore", report)
	}
}
