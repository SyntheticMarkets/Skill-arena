package db

import (
	"context"
	"os"
	"testing"

	"skill-arena/internal/config"
	"skill-arena/internal/models"
)

func TestPostgresArenaHubRepository(t *testing.T) {
	databaseURL := os.Getenv("SKILL_ARENA_TEST_POSTGRES_URL")
	if databaseURL == "" {
		t.Skip("SKILL_ARENA_TEST_POSTGRES_URL is not configured")
	}
	originalWorkingDirectory, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	testWorkingDirectory := t.TempDir()
	if err := os.Chdir(testWorkingDirectory); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(originalWorkingDirectory) })

	ctx := context.Background()
	options := Options{
		DatabaseURL: databaseURL,
		Environment: "development",
		Storage:     config.StorageSettings{LocalRoot: t.TempDir()},
	}
	store, err := NewWithOptions(ctx, options)
	if err != nil {
		t.Fatalf("open PostgreSQL store: %v", err)
	}
	if _, err := store.pg.ExecContext(ctx, `
TRUNCATE notification_events,player_notifications,support_tickets,player_profiles,
 progression,auth_tokens,mfa_settings,password_history,login_security,auth_sessions,
 devices,audit_logs,users CASCADE`); err != nil {
		t.Fatalf("reset Hub tables: %v", err)
	}

	user := models.NewUser("postgres-hub-user", "postgres-hub@example.com", "hash")
	user.Country = "ZA"
	user.EmailVerified = true
	if err := store.CreateUser(ctx, user); err != nil {
		t.Fatalf("create Hub user: %v", err)
	}
	profile, err := store.GetPlayerProfile(ctx, user.ID)
	if err != nil {
		t.Fatalf("create profile: %v", err)
	}
	profile.Username = "postgres_hub"
	profile.DisplayName = "Postgres Hub"
	profile.Language = "en"
	if _, err := store.UpdatePlayerProfile(ctx, profile); err != nil {
		t.Fatalf("update profile: %v", err)
	}
	if _, err := store.GetHubProgression(ctx, user.ID); err != nil {
		t.Fatalf("create progression: %v", err)
	}
	notification := &models.Notification{
		UserID: user.ID, Category: "security", Title: "Session protected",
		Message: "Your current session passed the security check.",
	}
	if err := store.CreateNotification(ctx, notification); err != nil {
		t.Fatalf("create notification: %v", err)
	}
	if err := store.UpdateNotificationStatus(ctx, user.ID, notification.ID, models.NotificationStatusRead); err != nil {
		t.Fatalf("read notification: %v", err)
	}
	ticket := &models.SupportTicket{
		UserID: user.ID, Category: "account", Subject: "Repository test",
		Message: "This ticket proves normalized persistence.",
	}
	if err := store.CreateSupportTicket(ctx, ticket); err != nil {
		t.Fatalf("create ticket: %v", err)
	}
	if err := store.Close(ctx); err != nil {
		t.Fatalf("close first store: %v", err)
	}

	reopened, err := NewWithOptions(ctx, options)
	if err != nil {
		t.Fatalf("reopen PostgreSQL store: %v", err)
	}
	t.Cleanup(func() { _ = reopened.Close(context.Background()) })
	persistedProfile, err := reopened.GetPlayerProfile(ctx, user.ID)
	if err != nil || persistedProfile.DisplayName != "Postgres Hub" {
		t.Fatalf("persisted profile=%+v err=%v", persistedProfile, err)
	}
	notifications, err := reopened.ListNotifications(ctx, user.ID, "")
	if err != nil || len(notifications) != 1 || notifications[0].ID != notification.ID ||
		notifications[0].Status != models.NotificationStatusRead {
		t.Fatalf("persisted notifications=%+v err=%v", notifications, err)
	}
	tickets, err := reopened.ListSupportTickets(ctx, user.ID)
	if err != nil || len(tickets) != 1 || tickets[0].ID != ticket.ID {
		t.Fatalf("persisted tickets=%+v err=%v", tickets, err)
	}
	var moduleCount, eventCount int
	if err := reopened.pg.QueryRowContext(ctx, `SELECT COUNT(*) FROM game_modules`).Scan(&moduleCount); err != nil {
		t.Fatal(err)
	}
	if err := reopened.pg.QueryRowContext(ctx, `SELECT COUNT(*) FROM notification_events`).Scan(&eventCount); err != nil {
		t.Fatal(err)
	}
	if moduleCount == 0 || eventCount != 2 {
		t.Fatalf("moduleCount=%d eventCount=%d", moduleCount, eventCount)
	}
}
