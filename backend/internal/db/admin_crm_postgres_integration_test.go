package db

import (
	"context"
	"os"
	"testing"

	"skill-arena/internal/config"
	"skill-arena/internal/models"
)

func TestPostgresAdminCRMRepository(t *testing.T) {
	databaseURL := os.Getenv("SKILL_ARENA_TEST_POSTGRES_URL")
	if databaseURL == "" {
		t.Skip("SKILL_ARENA_TEST_POSTGRES_URL is not configured")
	}
	ctx := context.Background()
	store, err := NewWithOptions(ctx, Options{
		DatabaseURL: databaseURL, Environment: "development",
		Storage: config.StorageSettings{LocalRoot: t.TempDir()},
	})
	if err != nil {
		t.Fatalf("open PostgreSQL store: %v", err)
	}
	t.Cleanup(func() { _ = store.Close(context.Background()) })
	if _, err := store.pg.ExecContext(ctx, `TRUNCATE crm_support_attachments,crm_support_messages,crm_internal_notes,
		crm_restrictions,crm_jurisdiction_policies,crm_announcements,crm_compliance_provider_responses,support_tickets,auth_tokens,mfa_settings,
		password_history,login_security,auth_sessions,devices,audit_logs,users CASCADE`); err != nil {
		t.Fatalf("reset CRM tables: %v", err)
	}

	adminEmail := config.Runtime().Admin.SuperAdminEmails[0]
	admin := models.NewUser("crm-pg-admin", adminEmail, "hash")
	admin.EmailVerified = true
	player := models.NewUser("crm-pg-player", "crm-pg-player@example.com", "hash")
	if err := store.CreateUser(ctx, admin); err != nil {
		t.Fatal(err)
	}
	if err := store.CreateUser(ctx, player); err != nil {
		t.Fatal(err)
	}

	updated, err := store.UpdateUserRole(ctx, admin.ID, player.ID, models.RoleSupport, "127.0.0.1")
	if err != nil || updated.Role != models.RoleSupport {
		t.Fatalf("role update=%+v err=%v", updated, err)
	}
	var storedRole string
	if err := store.pg.QueryRowContext(ctx, `SELECT role FROM users WHERE id=$1`, player.ID).Scan(&storedRole); err != nil || storedRole != models.RoleSupport {
		t.Fatalf("stored role=%q err=%v", storedRole, err)
	}

	ticket := &models.SupportTicket{UserID: player.ID, Category: "account", Subject: "Identity review", Message: "Please review the attached identity evidence."}
	if err := store.CreateSupportTicket(ctx, ticket); err != nil {
		t.Fatal(err)
	}
	attachment, err := store.StoreSupportAttachment(ctx, player.ID, ticket.ID, "evidence.txt", "text/plain", []byte("verified support evidence"))
	if err != nil {
		t.Fatal(err)
	}
	loaded, data, err := store.GetSupportAttachment(ctx, attachment.ID)
	if err != nil || loaded.SHA256 == "" || string(data) != "verified support evidence" {
		t.Fatalf("attachment=%+v data=%q err=%v", loaded, data, err)
	}

	if _, err := store.SaveCRMJurisdiction(ctx, models.CRMJurisdictionPolicy{
		Country: "ZA", Currency: "ZAR", MinimumAge: 18, DepositEnabled: true, WithdrawalEnabled: false,
		SourceOfFundsRequired: true, DailyDepositMinor: 10_000, MonthlyDepositMinor: 100_000,
		DailyWithdrawalMinor: 5_000, MonthlyWithdrawalMinor: 50_000, UpdatedBy: admin.ID,
	}); err != nil {
		t.Fatal(err)
	}
	policy, err := store.GetCRMJurisdiction(ctx, "ZA")
	if err != nil || policy == nil || policy.WithdrawalEnabled {
		t.Fatalf("policy=%+v err=%v", policy, err)
	}
	response, err := store.RecordCRMComplianceProviderResponse(ctx, models.CRMComplianceProviderResponse{
		UserID: player.ID, Provider: "identity-provider", ProviderReference: "pg-check-123",
		CheckType: "sanctions", Status: "clear", Metadata: map[string]string{"source": "sandbox-contract-test"},
	})
	if err != nil {
		t.Fatal(err)
	}
	responses, err := store.ListCRMComplianceProviderResponses(ctx, player.ID)
	if err != nil || len(responses) != 1 || responses[0].ID != response.ID || responses[0].Metadata["source"] != "sandbox-contract-test" {
		t.Fatalf("provider responses=%+v err=%v", responses, err)
	}
	if err := store.VerifyCRMAuditChain(ctx); err != nil {
		logs, _ := store.GetAuditLogs(ctx, 200)
		for _, item := range logs {
			expected := auditHash(item.PreviousHash, item.ID, item.ActorID, item.Action, item.TargetID, item.IPAddress, item.Metadata, item.CreatedAt)
			t.Logf("audit id=%s created=%s previous=%s stored=%s expected=%s metadata=%v", item.ID, item.CreatedAt.Format("2006-01-02T15:04:05.999999999Z07:00"), item.PreviousHash, item.EntryHash, expected, item.Metadata)
		}
		t.Fatalf("audit chain: %v", err)
	}
}
