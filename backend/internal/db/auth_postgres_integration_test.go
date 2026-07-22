package db

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	"skill-arena/internal/config"
	"skill-arena/internal/models"

	"golang.org/x/crypto/bcrypt"
)

func TestPostgresAuthenticationRepository(t *testing.T) {
	databaseURL := os.Getenv("SKILL_ARENA_TEST_POSTGRES_URL")
	if databaseURL == "" {
		t.Skip("SKILL_ARENA_TEST_POSTGRES_URL is not configured")
	}
	ctx := context.Background()
	store, err := NewWithOptions(ctx, Options{DatabaseURL: databaseURL, Environment: "development", Storage: config.StorageSettings{LocalRoot: t.TempDir()}})
	if err != nil {
		t.Fatalf("open PostgreSQL store: %v", err)
	}
	t.Cleanup(func() { _ = store.Close(context.Background()) })
	if _, err := store.pg.ExecContext(ctx, `TRUNCATE auth_tokens,mfa_settings,password_history,login_security,auth_sessions,devices,audit_logs,users CASCADE`); err != nil {
		t.Fatalf("reset identity tables: %v", err)
	}

	hash, _ := bcrypt.GenerateFromPassword([]byte("StrongPassword!42"), bcrypt.MinCost)
	user := models.NewUser("postgres-auth-user", "postgres-auth@example.com", string(hash))
	user.Country = "ZA"
	if err := store.CreateUser(ctx, user); err != nil {
		t.Fatalf("create user: %v", err)
	}
	loaded, err := store.GetUserByEmail(ctx, strings.ToUpper(user.Email))
	if err != nil || loaded.ID != user.ID {
		t.Fatalf("case-insensitive user lookup: user=%v err=%v", loaded, err)
	}

	rawVerification := NewAuthToken()
	if _, err := store.CreateAuthToken(ctx, user.ID, models.AuthTokenPurposeEmailVerification, rawVerification, "127.0.0.1", time.Minute); err != nil {
		t.Fatalf("create verification token: %v", err)
	}
	if _, _, err := store.ConsumeAuthToken(ctx, models.AuthTokenPurposeEmailVerification, rawVerification, "127.0.0.1"); err != nil {
		t.Fatalf("consume verification token: %v", err)
	}
	if _, _, err := store.ConsumeAuthToken(ctx, models.AuthTokenPurposeEmailVerification, rawVerification, "127.0.0.1"); err == nil {
		t.Fatal("one-time token was accepted twice")
	}

	firstRefresh := NewRefreshToken()
	first, err := store.CreateAuthSession(ctx, user.ID, firstRefresh, "integration", "127.0.0.1", time.Hour)
	if err != nil {
		t.Fatalf("create session: %v", err)
	}
	secondRefresh := NewRefreshToken()
	_, second, err := store.RotateRefreshToken(ctx, firstRefresh, secondRefresh, "integration", "127.0.0.1", time.Hour)
	if err != nil || second.FamilyID != first.FamilyID {
		t.Fatalf("rotate session: second=%v err=%v", second, err)
	}
	if _, _, err := store.RotateRefreshToken(ctx, firstRefresh, NewRefreshToken(), "integration", "127.0.0.1", time.Hour); err == nil {
		t.Fatal("rotated token replay was accepted")
	}
	if _, _, err := store.ValidateAuthSession(ctx, second.ID, user.ID); err == nil {
		t.Fatal("refresh-token family remained active after replay")
	}

	resetToken := NewAuthToken()
	if _, err := store.CreateAuthToken(ctx, user.ID, models.AuthTokenPurposePasswordReset, resetToken, "127.0.0.1", time.Minute); err != nil {
		t.Fatalf("create reset token: %v", err)
	}
	newHash, _ := bcrypt.GenerateFromPassword([]byte("ReplacementPassword!43"), bcrypt.MinCost)
	if err := store.CompletePasswordReset(ctx, resetToken, string(newHash), "127.0.0.1"); err != nil {
		t.Fatalf("complete password reset: %v", err)
	}
	if err := store.CompletePasswordReset(ctx, resetToken, string(newHash), "127.0.0.1"); err == nil {
		t.Fatal("password reset token was accepted twice")
	}
	logs, err := store.GetAuditLogs(ctx, 100)
	if err != nil || len(logs) == 0 || logs[len(logs)-1].EntryHash == "" {
		t.Fatalf("audit hash chain missing: logs=%d err=%v", len(logs), err)
	}
}
