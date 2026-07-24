package db

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"skill-arena/internal/config"
	"skill-arena/internal/models"
)

func TestPostgresFinancialRepository(t *testing.T) {
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
		t.Fatal(err)
	}
	defer store.Close(ctx)

	testID := time.Now().UnixNano()
	user := models.NewUser(fmt.Sprintf("postgres-financial-user-%d", testID), fmt.Sprintf("postgres-financial-%d@example.com", testID), "hash")
	user.EmailVerified = true
	if err := store.CreateUser(ctx, user); err != nil {
		t.Fatal(err)
	}
	deposit, _, err := store.CreateFinancialDeposit(ctx, &models.FinancialDeposit{
		UserID: user.ID, AmountMinor: 7_777, Currency: "ZAR", Method: "card", Provider: "card",
		IdempotencyKey: "postgres-deposit-key-001", RequestHash: "postgres-deposit-hash",
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.SetFinancialDepositProviderSession(ctx, deposit.ID, "postgres-provider-reference", "https://payments.example/checkout"); err != nil {
		t.Fatal(err)
	}
	if _, err := store.AdvanceFinancialDeposit(ctx, deposit.ID, models.DepositStatusPendingVerification, "postgres-event-1"); err != nil {
		t.Fatal(err)
	}
	if _, err := store.SettleFinancialDeposit(ctx, deposit.ID, "postgres-event-1"); err != nil {
		t.Fatal(err)
	}
	var transitionCount int
	if err := store.pg.QueryRowContext(ctx, `
SELECT COUNT(*) FROM financial_transitions WHERE resource_type='deposit' AND resource_id=$1`, deposit.ID).Scan(&transitionCount); err != nil {
		t.Fatal(err)
	}
	if transitionCount != 4 {
		t.Fatalf("deposit transition count=%d, want 4", transitionCount)
	}
	wallet, err := store.GetFinancialWallet(ctx, user.ID)
	if err != nil || wallet.AvailableMinor != 7_777 {
		t.Fatalf("wallet=%+v err=%v", wallet, err)
	}
	if err := store.VerifyFinancialJournal(ctx, user.ID); err != nil {
		t.Fatal(err)
	}
	evidence, err := store.StoreFinancialEvidence(ctx, user.ID, "identity", "application/pdf", []byte("%PDF-1.4 integration evidence"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.SaveFinancialPayoutDestination(ctx, models.FinancialPayoutDestination{
		UserID: user.ID, Provider: "stripe", ProviderAccountID: fmt.Sprintf("acct_%d", testID),
		Status: "verified", EvidenceID: evidence.ID,
	}); err != nil {
		t.Fatal(err)
	}
	check, err := store.RecordTreasuryReserveCheck(ctx, "stripe", "ZAR", 7_777, 0, 7_777, 0, "reconciliation")
	if err != nil || !check.Passed {
		t.Fatalf("reserve check=%+v err=%v", check, err)
	}
}
