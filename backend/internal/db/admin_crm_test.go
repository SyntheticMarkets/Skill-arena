package db

import (
	"context"
	"testing"
	"time"

	"skill-arena/internal/models"
)

func TestCRMRestrictionLifecycleAndEnforcement(t *testing.T) {
	ctx := context.Background()
	store, err := New(ctx, t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	admin := models.NewUser("crm-compliance", "compliance@example.com", "hash")
	admin.Role = models.RoleCompliance
	player := models.NewUser("crm-player", "player@example.com", "hash")
	if err := store.CreateUser(ctx, admin); err != nil {
		t.Fatal(err)
	}
	if err := store.CreateUser(ctx, player); err != nil {
		t.Fatal(err)
	}
	restriction, err := store.CreateCRMRestriction(ctx, models.CRMRestriction{
		UserID: player.ID, Type: "withdrawal", Reason: "Manual compliance review",
		CreatedBy: admin.ID,
	})
	if err != nil {
		t.Fatal(err)
	}
	active, err := store.HasActiveCRMRestriction(ctx, player.ID, "account", "withdrawal")
	if err != nil || !active {
		t.Fatalf("active=%v err=%v", active, err)
	}
	lifted, err := store.UpdateCRMRestrictionStatus(ctx, player.ID, restriction.ID, "lifted")
	if err != nil || lifted.Status != "lifted" {
		t.Fatalf("lifted=%+v err=%v", lifted, err)
	}
	active, err = store.HasActiveCRMRestriction(ctx, player.ID, "withdrawal")
	if err != nil || active {
		t.Fatalf("lifted restriction remained active: active=%v err=%v", active, err)
	}

	expiredAt := time.Now().UTC().Add(-time.Minute)
	if _, err := store.CreateCRMRestriction(ctx, models.CRMRestriction{
		UserID: player.ID, Type: "deposit", Reason: "Expired policy hold",
		ExpiresAt: &expiredAt, CreatedBy: admin.ID,
	}); err != nil {
		t.Fatal(err)
	}
	active, err = store.HasActiveCRMRestriction(ctx, player.ID, "deposit")
	if err != nil || active {
		t.Fatalf("expired restriction remained active: active=%v err=%v", active, err)
	}
}

func TestCRMJurisdictionOverridesAreAuthoritative(t *testing.T) {
	ctx := context.Background()
	store, err := New(ctx, t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	admin := models.NewUser("crm-policy-admin", "policy@example.com", "hash")
	admin.Role = models.RoleCompliance
	if err := store.CreateUser(ctx, admin); err != nil {
		t.Fatal(err)
	}
	saved, err := store.SaveCRMJurisdiction(ctx, models.CRMJurisdictionPolicy{
		Country: "za", Currency: "zar", MinimumAge: 21, DepositEnabled: false, WithdrawalEnabled: true,
		SourceOfFundsRequired: true, DailyDepositMinor: 10_000, MonthlyDepositMinor: 50_000,
		DailyWithdrawalMinor: 5_000, MonthlyWithdrawalMinor: 25_000, UpdatedBy: admin.ID,
	})
	if err != nil {
		t.Fatal(err)
	}
	loaded, err := store.GetCRMJurisdiction(ctx, "ZA")
	if err != nil || loaded == nil || loaded.MinimumAge != 21 || loaded.DepositEnabled || saved.Currency != "ZAR" {
		t.Fatalf("loaded=%+v saved=%+v err=%v", loaded, saved, err)
	}
}

func TestCRMComplianceProviderResponsesAreDurableCaseData(t *testing.T) {
	ctx := context.Background()
	store, err := New(ctx, t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	player := models.NewUser("crm-provider-player", "provider-player@example.com", "hash")
	if err := store.CreateUser(ctx, player); err != nil {
		t.Fatal(err)
	}
	recorded, err := store.RecordCRMComplianceProviderResponse(ctx, models.CRMComplianceProviderResponse{
		UserID: player.ID, Provider: "identity-provider", ProviderReference: "check-123",
		CheckType: "identity", Status: "review", RiskSignals: []string{"document_expiring"},
		Metadata: map[string]string{"country": "ZA"},
	})
	if err != nil {
		t.Fatal(err)
	}
	responses, err := store.ListCRMComplianceProviderResponses(ctx, player.ID)
	if err != nil || len(responses) != 1 || responses[0].ID != recorded.ID ||
		responses[0].RiskSignals[0] != "document_expiring" {
		t.Fatalf("responses=%+v err=%v", responses, err)
	}
	cases, err := store.ListCRMComplianceCases(ctx, "")
	if err != nil {
		t.Fatalf("cases=%+v err=%v", cases, err)
	}
	found := false
	for _, item := range cases {
		if item.User.ID == player.ID && len(item.ProviderResponses) == 1 {
			found = true
		}
	}
	if !found {
		t.Fatalf("provider response was not included in player compliance case: %+v", cases)
	}
}

func TestResponsibleGamingRestrictionsRequireExpiry(t *testing.T) {
	ctx := context.Background()
	store, err := New(ctx, t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	admin := models.NewUser("crm-rg-admin", "rg-admin@example.com", "hash")
	player := models.NewUser("crm-rg-player", "rg-player@example.com", "hash")
	if err := store.CreateUser(ctx, admin); err != nil {
		t.Fatal(err)
	}
	if err := store.CreateUser(ctx, player); err != nil {
		t.Fatal(err)
	}
	if _, err := store.CreateCRMRestriction(ctx, models.CRMRestriction{
		UserID: player.ID, Type: "self_exclusion", Reason: "Player requested exclusion", CreatedBy: admin.ID,
	}); err == nil {
		t.Fatal("self-exclusion without expiry was accepted")
	}
	expiresAt := time.Now().UTC().Add(30 * 24 * time.Hour)
	if _, err := store.CreateCRMRestriction(ctx, models.CRMRestriction{
		UserID: player.ID, Type: "self_exclusion", Reason: "Player requested exclusion",
		CreatedBy: admin.ID, ExpiresAt: &expiresAt,
	}); err != nil {
		t.Fatal(err)
	}
	active, err := store.HasActiveCRMRestriction(ctx, player.ID, "self_exclusion")
	if err != nil || !active {
		t.Fatalf("active=%v err=%v", active, err)
	}
}
