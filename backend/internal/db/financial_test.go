package db

import (
	"context"
	"testing"
	"time"

	"skill-arena/internal/models"
)

func TestFinancialLifecycleUsesMinorUnitsAndReconciles(t *testing.T) {
	ctx := context.Background()
	store, err := New(ctx, t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	user := models.NewUser("financial-user", "financial@example.com", "hash")
	user.EmailVerified = true
	user.Country = "ZA"
	if err := store.CreateUser(ctx, user); err != nil {
		t.Fatal(err)
	}
	if err := store.EnsureFinancialAccount(ctx, user.ID, "ZAR"); err != nil {
		t.Fatal(err)
	}
	if _, err := store.SaveFinancialAssessment(ctx, models.FinancialAssessment{
		UserID: user.ID, Country: "ZA", Occupation: "employed", SourceOfFunds: "salary",
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := store.ReviewFinancialAssessment(ctx, user.ID, models.AssessmentStatusComplete, "standard", "verified"); err != nil {
		t.Fatal(err)
	}

	depositRequest := &models.FinancialDeposit{
		UserID: user.ID, AmountMinor: 12_345, Currency: "ZAR", Method: "card",
		Provider: "card", IdempotencyKey: "deposit-idempotency-0001", RequestHash: "deposit-hash",
	}
	deposit, replayed, err := store.CreateFinancialDeposit(ctx, depositRequest)
	if err != nil || replayed {
		t.Fatalf("create deposit replayed=%v err=%v", replayed, err)
	}
	replay, replayed, err := store.CreateFinancialDeposit(ctx, &models.FinancialDeposit{
		UserID: user.ID, AmountMinor: 12_345, Currency: "ZAR", Method: "card",
		Provider: "card", IdempotencyKey: "deposit-idempotency-0001", RequestHash: "deposit-hash",
	})
	if err != nil || !replayed || replay.ID != deposit.ID {
		t.Fatalf("idempotent replay=%+v replayed=%v err=%v", replay, replayed, err)
	}
	if _, _, err := store.CreateFinancialDeposit(ctx, &models.FinancialDeposit{
		UserID: user.ID, AmountMinor: 12_346, Currency: "ZAR", Method: "card",
		Provider: "card", IdempotencyKey: "deposit-idempotency-0001", RequestHash: "different-hash",
	}); err == nil {
		t.Fatal("expected idempotency conflict")
	}
	if _, err := store.SetFinancialDepositProviderSession(ctx, deposit.ID, "provider-deposit", "https://payments.example/checkout"); err != nil {
		t.Fatal(err)
	}
	for _, status := range []string{
		models.DepositStatusPendingVerification,
	} {
		deposit, err = store.AdvanceFinancialDeposit(ctx, deposit.ID, status, "provider-event-1")
		if err != nil || deposit.Status != status {
			t.Fatalf("advance deposit to %s: deposit=%+v err=%v", status, deposit, err)
		}
	}
	if _, err := store.SettleFinancialDeposit(ctx, deposit.ID, "provider-event-1"); err != nil {
		t.Fatal(err)
	}
	wallet, err := store.GetFinancialWallet(ctx, user.ID)
	if err != nil || wallet.AvailableMinor != 12_345 || wallet.PendingDepositMinor != 0 {
		t.Fatalf("wallet after deposit=%+v err=%v", wallet, err)
	}

	withdrawal, replayed, err := store.CreateFinancialWithdrawal(ctx, &models.FinancialWithdrawal{
		UserID: user.ID, AmountMinor: 2_345, Currency: "ZAR", Method: "eft",
		Provider: "ozow", IdempotencyKey: "withdraw-idempotency-001", RequestHash: "withdraw-hash",
	})
	if err != nil || replayed || withdrawal.Status != models.FinancialWithdrawalStatusPendingReview {
		t.Fatalf("create withdrawal=%+v replayed=%v err=%v", withdrawal, replayed, err)
	}
	for _, status := range []string{
		models.FinancialWithdrawalStatusApproved,
		models.FinancialWithdrawalStatusProcessing,
		models.FinancialWithdrawalStatusCompleted,
	} {
		withdrawal, err = store.TransitionFinancialWithdrawal(ctx, withdrawal.ID, status, "test", "reviewer", "test transition", "provider-withdrawal")
		if err != nil {
			t.Fatalf("transition to %s: %v", status, err)
		}
	}
	wallet, err = store.GetFinancialWallet(ctx, user.ID)
	if err != nil || wallet.AvailableMinor != 10_000 || wallet.PendingWithdrawalMinor != 0 ||
		wallet.LifetimeDepositMinor != 12_345 || wallet.LifetimeWithdrawMinor != 2_345 {
		t.Fatalf("wallet after withdrawal=%+v err=%v", wallet, err)
	}
	if err := store.VerifyFinancialJournal(ctx, user.ID); err != nil {
		t.Fatal(err)
	}
	statement, err := store.FinancialStatement(ctx, user.ID, time.Now().Add(-time.Hour), time.Now().Add(time.Hour))
	if err != nil || statement.TotalCreditMinor != 12_345 || statement.TotalDebitMinor != 2_345 ||
		statement.ClosingMinor != 10_000 {
		t.Fatalf("statement=%+v err=%v", statement, err)
	}
}

func TestFinancialWebhookRetriesOnlyAfterFailure(t *testing.T) {
	ctx := context.Background()
	store, err := New(ctx, t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	arguments := []string{"card", "event-1", "signature-hash", "payload-hash", "deposit", "deposit-1"}
	first, err := store.RecordFinancialWebhook(ctx, arguments[0], arguments[1], arguments[2], arguments[3], arguments[4], arguments[5])
	if err != nil || !first {
		t.Fatalf("first webhook accepted=%v err=%v", first, err)
	}
	duplicate, err := store.RecordFinancialWebhook(ctx, arguments[0], arguments[1], arguments[2], arguments[3], arguments[4], arguments[5])
	if err != nil || duplicate {
		t.Fatalf("in-flight duplicate accepted=%v err=%v", duplicate, err)
	}
	if err := store.CompleteFinancialWebhook(ctx, arguments[0], arguments[1], "failed"); err != nil {
		t.Fatal(err)
	}
	retry, err := store.RecordFinancialWebhook(ctx, arguments[0], arguments[1], arguments[2], arguments[3], arguments[4], arguments[5])
	if err != nil || !retry {
		t.Fatalf("failed webhook retry accepted=%v err=%v", retry, err)
	}
	if err := store.CompleteFinancialWebhook(ctx, arguments[0], arguments[1], "deposit_completed"); err != nil {
		t.Fatal(err)
	}
	replay, err := store.RecordFinancialWebhook(ctx, arguments[0], arguments[1], arguments[2], arguments[3], arguments[4], arguments[5])
	if err != nil || replay {
		t.Fatalf("completed webhook replay accepted=%v err=%v", replay, err)
	}
}

func TestFinancialEvidenceArtifactsDestinationsAndReserveChecks(t *testing.T) {
	ctx := context.Background()
	store, err := New(ctx, t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	user := models.NewUser("financial-artifacts-user", "financial-artifacts@example.com", "hash")
	if err := store.CreateUser(ctx, user); err != nil {
		t.Fatal(err)
	}
	evidence, err := store.StoreFinancialEvidence(ctx, user.ID, "identity", "image/png", []byte("verified-test-image"))
	if err != nil || evidence.SHA256 == "" || evidence.Status != "received" {
		t.Fatalf("evidence=%+v err=%v", evidence, err)
	}
	items, err := store.ListFinancialEvidence(ctx, user.ID)
	if err != nil || len(items) != 1 || items[0].ID != evidence.ID {
		t.Fatalf("evidence list=%+v err=%v", items, err)
	}
	artifact, err := store.StoreFinancialArtifact(ctx, user.ID, "statement", "text/csv", []byte("amount_minor\n100\n"))
	if err != nil {
		t.Fatal(err)
	}
	loaded, data, err := store.GetFinancialArtifact(ctx, user.ID, artifact.ID)
	if err != nil || loaded.SHA256 != artifact.SHA256 || string(data) != "amount_minor\n100\n" {
		t.Fatalf("artifact=%+v data=%q err=%v", loaded, data, err)
	}
	if _, _, err := store.GetFinancialArtifact(ctx, "another-user", artifact.ID); err == nil {
		t.Fatal("expected cross-account artifact access to fail")
	}
	destination, err := store.SaveFinancialPayoutDestination(ctx, models.FinancialPayoutDestination{
		UserID: user.ID, Provider: "stripe", ProviderAccountID: "acct_test_player",
		Status: "verified", EvidenceID: evidence.ID,
	})
	if err != nil || destination.Status != "verified" {
		t.Fatalf("destination=%+v err=%v", destination, err)
	}
	if _, err := store.GetFinancialPayoutDestination(ctx, user.ID, "stripe"); err != nil {
		t.Fatal(err)
	}
	passed, err := store.RecordTreasuryReserveCheck(ctx, "stripe", "ZAR", 500, 500, 900, 100, "deposit_settlement")
	if err != nil || !passed.Passed || passed.ImmutableHash == "" || passed.ArtifactKey == "" {
		t.Fatalf("passed reserve=%+v err=%v", passed, err)
	}
	failed, err := store.RecordTreasuryReserveCheck(ctx, "stripe", "ZAR", 50, 0, 50, 100, "withdrawal_processing")
	if err != nil || failed.Passed {
		t.Fatalf("failed reserve=%+v err=%v", failed, err)
	}
}

func TestRejectedWithdrawalReturnsReservedFunds(t *testing.T) {
	ctx := context.Background()
	store, err := New(ctx, t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	user := models.NewUser("rejection-user", "rejection@example.com", "hash")
	if err := store.CreateUser(ctx, user); err != nil {
		t.Fatal(err)
	}
	deposit, _, err := store.CreateFinancialDeposit(ctx, &models.FinancialDeposit{
		UserID: user.ID, AmountMinor: 5_000, Currency: "ZAR", Method: "eft", Provider: "ozow",
		IdempotencyKey: "rejection-deposit-key", RequestHash: "rejection-deposit",
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.SetFinancialDepositProviderSession(ctx, deposit.ID, "deposit-ref", "https://payments.example"); err != nil {
		t.Fatal(err)
	}
	if _, err := store.SettleFinancialDeposit(ctx, deposit.ID, "event"); err != nil {
		t.Fatal(err)
	}
	withdrawal, _, err := store.CreateFinancialWithdrawal(ctx, &models.FinancialWithdrawal{
		UserID: user.ID, AmountMinor: 4_000, Currency: "ZAR", Method: "eft", Provider: "ozow",
		IdempotencyKey: "rejection-withdraw-key", RequestHash: "rejection-withdraw",
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.TransitionFinancialWithdrawal(ctx, withdrawal.ID, models.FinancialWithdrawalStatusRejected, "treasury", "reviewer", "verification required", ""); err != nil {
		t.Fatal(err)
	}
	wallet, err := store.GetFinancialWallet(ctx, user.ID)
	if err != nil || wallet.AvailableMinor != 5_000 || wallet.PendingWithdrawalMinor != 0 {
		t.Fatalf("wallet=%+v err=%v", wallet, err)
	}
}
