package db

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"

	"skill-arena/internal/config"
	"skill-arena/internal/game"
	"skill-arena/internal/models"
)

func TestSubmitMazeMovesRejectsWrongUserWithoutSettling(t *testing.T) {
	ctx := context.Background()
	store, err := New(ctx, t.TempDir())
	if err != nil {
		t.Fatalf("new store: %v", err)
	}

	owner := models.NewUser("", "owner@example.com", "hash")
	other := models.NewUser("", "other@example.com", "hash")
	if err := store.CreateUser(ctx, owner); err != nil {
		t.Fatalf("create owner: %v", err)
	}
	if err := store.CreateUser(ctx, other); err != nil {
		t.Fatalf("create other: %v", err)
	}

	session := &models.GameSession{UserID: owner.ID, GameType: "demo", Stake: 10}
	if err := store.StartGameSession(ctx, session); err != nil {
		t.Fatalf("start game: %v", err)
	}

	before, err := store.GetWalletByUserID(ctx, owner.ID)
	if err != nil {
		t.Fatalf("wallet before: %v", err)
	}
	beforeDemo := before.DemoBalance
	beforeLocked := before.DemoLockedBalance

	if _, err := store.SubmitMazeMoves(ctx, other.ID, session.ID, []models.MazeMove{{Direction: "right"}}); err == nil {
		t.Fatal("expected ownership error")
	}

	after, err := store.GetWalletByUserID(ctx, owner.ID)
	if err != nil {
		t.Fatalf("wallet after: %v", err)
	}
	if after.DemoBalance != beforeDemo || after.DemoLockedBalance != beforeLocked {
		t.Fatalf("wallet changed on unauthorized submit: before %.2f/%.2f after %.2f/%.2f", beforeDemo, beforeLocked, after.DemoBalance, after.DemoLockedBalance)
	}
}

func TestWithdrawalRequiresAndReusesIdempotencyKey(t *testing.T) {
	ctx := context.Background()
	store, err := New(ctx, t.TempDir())
	if err != nil {
		t.Fatalf("new store: %v", err)
	}

	user := models.NewUser("", "withdraw-idempotency@example.com", "hash")
	user.EmailVerified = true
	if err := store.CreateUser(ctx, user); err != nil {
		t.Fatalf("create user: %v", err)
	}
	if _, err := store.RecordWalletTransaction(ctx, user.ID, models.TransactionTypeDeposit, 1000, "USD", "test-funding", nil); err != nil {
		t.Fatalf("fund wallet: %v", err)
	}

	if _, _, err := store.CreateWithdrawalRequest(ctx, user.ID, "bank_eft", "bank", 100, "USD", "ref-1", nil); err == nil {
		t.Fatal("expected missing idempotency metadata error")
	}

	metadata := map[string]string{"idempotencyKey": "withdraw-1", "requestHash": "hash-1"}
	first, _, err := store.CreateWithdrawalRequest(ctx, user.ID, "bank_eft", "bank", 100, "USD", "ref-1", metadata)
	if err != nil {
		t.Fatalf("first withdrawal: %v", err)
	}
	second, _, err := store.CreateWithdrawalRequest(ctx, user.ID, "bank_eft", "bank", 100, "USD", "ref-1", metadata)
	if err != nil {
		t.Fatalf("duplicate withdrawal: %v", err)
	}
	if second.ID != first.ID {
		t.Fatalf("duplicate withdrawal id = %q, want %q", second.ID, first.ID)
	}

	different := map[string]string{"idempotencyKey": "withdraw-1", "requestHash": "hash-2"}
	if _, _, err := store.CreateWithdrawalRequest(ctx, user.ID, "bank_eft", "bank", 150, "USD", "ref-2", different); err == nil {
		t.Fatal("expected idempotency key reuse error")
	}
}

func TestFinancialFlowReconcilesLedgerWalletTreasuryAndAudit(t *testing.T) {
	ctx := context.Background()
	store, err := New(ctx, t.TempDir())
	if err != nil {
		t.Fatalf("new store: %v", err)
	}

	player := models.NewUser("", "financial-flow@example.com", "hash")
	player.EmailVerified = true
	player.KYCStatus = "approved"
	if err := store.CreateUser(ctx, player); err != nil {
		t.Fatalf("create player: %v", err)
	}
	treasury := models.NewUser("", "treasury-flow@example.com", "hash")
	treasury.Role = models.RoleTreasuryManager
	if err := store.CreateUser(ctx, treasury); err != nil {
		t.Fatalf("create treasury user: %v", err)
	}

	deposit, err := store.CreateDepositSession(ctx, player.ID, "payfast", "card", 100, "USD", "provider-intent-1", map[string]string{
		"idempotencyKey": "deposit-flow-1",
		"requestHash":    "deposit-hash-1",
	})
	if err != nil {
		t.Fatalf("create deposit session: %v", err)
	}
	if _, err := store.MarkDepositPending(ctx, deposit.ID, "provider-pending-1", nil); err != nil {
		t.Fatalf("mark deposit pending: %v", err)
	}
	if _, err := store.SettleDeposit(ctx, deposit.ID, "provider-settled-1", nil); err != nil {
		t.Fatalf("settle deposit: %v", err)
	}

	session := &models.GameSession{UserID: player.ID, GameType: "live", Stake: 10, RewardRate: 0.8}
	if err := store.StartGameSession(ctx, session); err != nil {
		t.Fatalf("start live game: %v", err)
	}
	solution, ok := game.SolveLinePuzzle(session.Lines)
	if !ok {
		t.Fatal("generated live session puzzle is not solvable")
	}
	clicks := make([]models.MazeMove, 0, len(solution))
	for _, lineID := range solution {
		clicks = append(clicks, models.MazeMove{Direction: lineID})
	}
	if _, err := store.SubmitMazeMoves(ctx, player.ID, session.ID, clicks); err != nil {
		t.Fatalf("submit winning game: %v", err)
	}

	withdrawal, _, err := store.CreateWithdrawalRequest(ctx, player.ID, "bank_eft", "bank", 20, "USD", "withdrawal-ref-1", map[string]string{
		"idempotencyKey": "withdraw-flow-1",
		"requestHash":    "withdraw-hash-1",
		"country":        "ZA",
	})
	if err != nil {
		t.Fatalf("create withdrawal: %v", err)
	}
	if _, err := store.ApproveWithdrawal(ctx, treasury.ID, withdrawal.ID, "127.0.0.1"); err != nil {
		t.Fatalf("approve withdrawal: %v", err)
	}
	if _, err := store.SettleWithdrawal(ctx, treasury.ID, withdrawal.ID, "provider-paid-1", "127.0.0.1"); err != nil {
		t.Fatalf("settle withdrawal: %v", err)
	}

	wallet, err := store.GetWalletByUserID(ctx, player.ID)
	if err != nil {
		t.Fatalf("wallet: %v", err)
	}
	if wallet.LiveLockedBalance != 0 || wallet.PendingWithdrawals != 0 {
		t.Fatalf("wallet locks/pending = %.2f/%.2f, want 0/0", wallet.LiveLockedBalance, wallet.PendingWithdrawals)
	}
	if wallet.LiveBalance != 87.8 {
		t.Fatalf("live balance = %.2f, want 87.80", wallet.LiveBalance)
	}

	entries, err := store.GetLedgerEntriesByUserID(ctx, player.ID)
	if err != nil {
		t.Fatalf("ledger: %v", err)
	}
	balanceChanging := 0.0
	for _, entry := range entries {
		switch entry.TransactionType {
		case models.TransactionTypeDeposit, models.TransactionTypeReward, models.TransactionTypeWithdraw, models.TransactionTypeFee:
			if entry.Reference == "initial-demo-credit" {
				continue
			}
			balanceChanging += entry.Amount
		}
	}
	if balanceChanging != wallet.LiveBalance {
		t.Fatalf("balance-changing ledger sum = %.2f, wallet live = %.2f", balanceChanging, wallet.LiveBalance)
	}
	health, err := store.GetTreasuryHealth(ctx)
	if err != nil {
		t.Fatalf("treasury health: %v", err)
	}
	if !health.IsSolvent {
		t.Fatal("treasury should remain solvent after financial flow")
	}
	audit, err := store.GetAuditLogs(ctx, 100)
	if err != nil {
		t.Fatalf("audit logs: %v", err)
	}
	required := map[string]bool{
		"wallet.deposit.settled":       false,
		"wallet.withdrawal.requested":  false,
		"treasury.withdrawal.approved": false,
		"treasury.withdrawal.settled":  false,
	}
	for _, log := range audit {
		if _, ok := required[log.Action]; ok {
			required[log.Action] = true
		}
	}
	for action, found := range required {
		if !found {
			t.Fatalf("missing audit action %s in %#v", action, audit)
		}
	}
	report, err := store.GetReplayReport(ctx, session.ID)
	if err != nil {
		t.Fatalf("replay report: %v", err)
	}
	if report.ReplaySignature == "" || containsTestString(report.Flags, "puzzle_reconstruction_mismatch") || containsTestString(report.Flags, "missing_reconstruction_metadata") {
		t.Fatalf("replay report signature/flags = %q/%#v", report.ReplaySignature, report.Flags)
	}
}

func TestReplayCanBeRegeneratedAndVerifiedYearsLater(t *testing.T) {
	ctx := context.Background()
	store, err := New(ctx, t.TempDir())
	if err != nil {
		t.Fatalf("new store: %v", err)
	}
	user := models.NewUser("", "replay-years@example.com", "hash")
	if err := store.CreateUser(ctx, user); err != nil {
		t.Fatalf("create user: %v", err)
	}
	session := &models.GameSession{UserID: user.ID, GameType: "demo", Stake: 10}
	if err := store.StartGameSession(ctx, session); err != nil {
		t.Fatalf("start game: %v", err)
	}
	solution, ok := game.SolveLinePuzzle(session.Lines)
	if !ok {
		t.Fatal("generated replay puzzle is not solvable")
	}
	clicks := make([]models.MazeMove, 0, len(solution))
	for _, lineID := range solution {
		clicks = append(clicks, models.MazeMove{Direction: lineID})
	}
	if _, err := store.SubmitMazeMoves(ctx, user.ID, session.ID, clicks); err != nil {
		t.Fatalf("submit game: %v", err)
	}
	report, err := store.GetReplayReport(ctx, session.ID)
	if err != nil {
		t.Fatalf("replay report: %v", err)
	}
	if report.PuzzleVersion.GameRulesVersion == "" || report.PuzzleVersion.ReplayVersion == "" {
		t.Fatalf("missing replay rules versions: %#v", report.PuzzleVersion)
	}
	derived, err := game.DerivePuzzleSeed(config.Runtime().Security.PuzzleSecret, game.SeedDerivationInput{
		Purpose:           "game",
		MatchID:           session.ID,
		PlayerID:          user.ID,
		Nonce:             report.GenerationNonce,
		DifficultyProfile: *report.DifficultyProfile,
		PuzzleVersion:     report.PuzzleVersion,
	})
	if err != nil {
		t.Fatalf("derive seed: %v", err)
	}
	if derived.Seed != report.PuzzleSeed || derived.GenerationHash != report.GenerationHash {
		t.Fatalf("derived seed/hash = %s/%s, want %s/%s", derived.Seed, derived.GenerationHash, report.PuzzleSeed, report.GenerationHash)
	}
	reconstructed := game.GenerateLinePuzzleFromProfile(report.PuzzleSeed, *report.DifficultyProfile)
	if !sameLinePuzzle(reconstructed, report.Lines) {
		t.Fatal("reconstructed line puzzle does not match replay lines")
	}
	payload := map[string]any{
		"sessionId": report.SessionID,
		"userId":    report.UserID,
		"gameType":  report.GameType,
		"outcome":   report.Outcome,
		"events":    report.PlaybackEvents,
		"flags":     report.Flags,
	}
	data, _ := json.Marshal(payload)
	mac := hmac.New(sha256.New, []byte(config.Runtime().Security.PuzzleSecret))
	mac.Write(data)
	expectedSignature := base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
	if report.ReplaySignature != expectedSignature {
		t.Fatalf("replay signature = %q, want regenerated signature %q", report.ReplaySignature, expectedSignature)
	}
}

func TestConcurrentLaunchLoadPaths(t *testing.T) {
	ctx := context.Background()
	store, err := New(ctx, t.TempDir())
	if err != nil {
		t.Fatalf("new store: %v", err)
	}

	const players = 200
	userIDs := make([]string, players)
	for i := 0; i < players; i++ {
		user := models.NewUser("", fmt.Sprintf("load-%03d@example.com", i), "hash")
		user.EmailVerified = true
		if err := store.CreateUser(ctx, user); err != nil {
			t.Fatalf("create user %d: %v", i, err)
		}
		userIDs[i] = user.ID
	}

	var wg sync.WaitGroup
	errs := make(chan error, 500)
	for i := 0; i < 100; i++ {
		i := i
		wg.Add(1)
		go func() {
			defer wg.Done()
			if _, err := store.CreateAuthSession(ctx, userIDs[i], "refresh-load-"+userIDs[i], "load-test", "127.0.0.1", time.Hour); err != nil {
				errs <- err
			}
		}()
		wg.Add(1)
		go func() {
			defer wg.Done()
			if _, err := store.CreateDepositSession(ctx, userIDs[i], "payfast", "card", 10, "USD", "load-deposit", map[string]string{
				"idempotencyKey": "load-deposit-" + userIDs[i],
				"requestHash":    "hash-" + userIDs[i],
			}); err != nil {
				errs <- err
			}
		}()
		wg.Add(1)
		go func() {
			defer wg.Done()
			session := &models.GameSession{UserID: userIDs[i], GameType: "demo", Stake: 1}
			if err := store.StartGameSession(ctx, session); err != nil {
				errs <- err
				return
			}
			if _, err := store.GetReplayReport(ctx, session.ID); err != nil {
				errs <- err
			}
		}()
		wg.Add(1)
		go func() {
			defer wg.Done()
			queueType := fmt.Sprintf("load-%03d", i)
			if err := joinPvPWithRetry(ctx, store, userIDs[i], queueType, "demo", 1); err != nil {
				errs <- err
			}
			if err := joinPvPWithRetry(ctx, store, userIDs[i+100], queueType, "demo", 1); err != nil {
				errs <- err
			}
		}()
		wg.Add(1)
		go func() {
			defer wg.Done()
			if _, err := store.GetLeaderboard(); err != nil {
				errs <- err
			}
		}()
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		t.Fatalf("load path error: %v", err)
	}
	matches, err := store.ListPvPMatchesByUserID(ctx, userIDs[0])
	if err != nil {
		t.Fatalf("list pvp matches: %v", err)
	}
	if len(matches) == 0 {
		t.Fatal("expected load test to create or join a pvp match")
	}
	t.Log("load test completed: 100 auth sessions, 100 deposits, 100 replay requests, 100 pvp joins, 100 leaderboard reads")
}

func joinPvPWithRetry(ctx context.Context, store *Store, userID, queueType, walletType string, stake float64) error {
	var lastErr error
	for attempt := 0; attempt < 5; attempt++ {
		_, err := store.JoinPvPQueue(ctx, userID, queueType, walletType, stake)
		if err == nil {
			return nil
		}
		lastErr = err
		if !strings.Contains(err.Error(), "queue is busy") {
			return err
		}
		time.Sleep(time.Duration(attempt+1) * 10 * time.Millisecond)
	}
	return lastErr
}

func containsTestString(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}

func TestStartGameSessionStoresDifficultyProfileAndSeed(t *testing.T) {
	ctx := context.Background()
	store, err := New(ctx, t.TempDir())
	if err != nil {
		t.Fatalf("new store: %v", err)
	}

	user := models.NewUser("", "difficulty@example.com", "hash")
	if err := store.CreateUser(ctx, user); err != nil {
		t.Fatalf("create user: %v", err)
	}

	session := &models.GameSession{UserID: user.ID, GameType: "demo", Stake: 10}
	if err := store.StartGameSession(ctx, session); err != nil {
		t.Fatalf("start game: %v", err)
	}

	if session.PuzzleSeed == "" {
		t.Fatal("expected puzzle seed to be stored")
	}
	if session.GenerationNonce == "" || session.GenerationHash == "" {
		t.Fatalf("expected cryptographic generation metadata, nonce=%q hash=%q", session.GenerationNonce, session.GenerationHash)
	}
	if session.State != models.SessionStateReady {
		t.Fatalf("state = %q, want READY", session.State)
	}
	if session.DifficultyRating <= 0 || session.DifficultyProfile == nil {
		t.Fatalf("expected difficulty metadata, got rating=%d profile=%#v", session.DifficultyRating, session.DifficultyProfile)
	}
	if session.PuzzleVersion.GeneratorVersion == "" || session.PuzzleVersion.DifficultyProfileVersion == "" || session.PuzzleVersion.GameRulesVersion == "" || session.PuzzleVersion.ReplayVersion == "" {
		t.Fatalf("expected puzzle version metadata, got %#v", session.PuzzleVersion)
	}
	if len(session.Lines) != session.DifficultyProfile.LineCount {
		t.Fatalf("line count = %d, want profile count %d", len(session.Lines), session.DifficultyProfile.LineCount)
	}
	reconstructed := game.GenerateLinePuzzleFromProfile(session.PuzzleSeed, *session.DifficultyProfile)
	if len(reconstructed) != len(session.Lines) || !reflect.DeepEqual(reconstructed[0], session.Lines[0]) {
		t.Fatal("expected stored seed to regenerate identical puzzle lines")
	}

	report, err := store.GetReplayReport(ctx, session.ID)
	if err != nil {
		t.Fatalf("replay report: %v", err)
	}
	if report.PuzzleSeed != session.PuzzleSeed || report.DifficultyRating != session.DifficultyRating {
		t.Fatalf("replay metadata = seed %q rating %d, want seed %q rating %d", report.PuzzleSeed, report.DifficultyRating, session.PuzzleSeed, session.DifficultyRating)
	}
	if report.GenerationHash != session.GenerationHash || report.GenerationNonce != session.GenerationNonce {
		t.Fatalf("replay generation metadata = nonce %q hash %q, want nonce %q hash %q", report.GenerationNonce, report.GenerationHash, session.GenerationNonce, session.GenerationHash)
	}
	if report.PuzzleVersion.GeneratorVersion != session.PuzzleVersion.GeneratorVersion {
		t.Fatalf("replay puzzle version = %#v, want %#v", report.PuzzleVersion, session.PuzzleVersion)
	}
}

func TestSubmitMazeMovesSettlesDemoLossFromLockedStake(t *testing.T) {
	ctx := context.Background()
	store, err := New(ctx, t.TempDir())
	if err != nil {
		t.Fatalf("new store: %v", err)
	}

	user := models.NewUser("", "player@example.com", "hash")
	if err := store.CreateUser(ctx, user); err != nil {
		t.Fatalf("create user: %v", err)
	}

	session := &models.GameSession{UserID: user.ID, GameType: "demo", Stake: 10}
	if err := store.StartGameSession(ctx, session); err != nil {
		t.Fatalf("start game: %v", err)
	}

	finished, err := store.SubmitMazeMoves(ctx, user.ID, session.ID, []models.MazeMove{{Direction: "left"}})
	if err != nil {
		t.Fatalf("submit moves: %v", err)
	}
	if finished.Outcome != "lose" {
		t.Fatalf("outcome = %q, want lose", finished.Outcome)
	}

	wallet, err := store.GetWalletByUserID(ctx, user.ID)
	if err != nil {
		t.Fatalf("wallet: %v", err)
	}
	if wallet.DemoBalance != 990 || wallet.DemoLockedBalance != 0 {
		t.Fatalf("wallet = demo %.2f locked %.2f, want 990/0", wallet.DemoBalance, wallet.DemoLockedBalance)
	}

	progression, err := store.GetProgressionByUserID(ctx, user.ID)
	if err != nil {
		t.Fatalf("progression: %v", err)
	}
	if progression.MatchesPlayed != 1 || progression.Losses != 1 || progression.XP != 10 {
		t.Fatalf("progression = matches %d losses %d xp %d, want 1/1/10", progression.MatchesPlayed, progression.Losses, progression.XP)
	}

	achievements, err := store.GetAchievementsByUserID(ctx, user.ID)
	if err != nil {
		t.Fatalf("achievements: %v", err)
	}
	if len(achievements) != 1 || achievements[0].Code != "first_match" {
		t.Fatalf("achievements = %#v, want first_match only", achievements)
	}

	report, err := store.GetReplayReport(ctx, session.ID)
	if err != nil {
		t.Fatalf("replay report: %v", err)
	}
	if report.SessionID != session.ID || report.MoveCount != 1 || report.ShortestPathLength == 0 {
		t.Fatalf("unexpected replay report: %#v", report)
	}
	if report.IntegrityStatus != "verified" {
		t.Fatalf("integrity status = %q, want verified", report.IntegrityStatus)
	}

	reports, err := store.GetReplayReportsByUserID(ctx, user.ID)
	if err != nil {
		t.Fatalf("replay reports by user: %v", err)
	}
	if len(reports) != 1 {
		t.Fatalf("replay report count = %d, want 1", len(reports))
	}
}

func TestFlaggedReplayCreatesReviewCaseAndMetrics(t *testing.T) {
	ctx := context.Background()
	store, err := New(ctx, t.TempDir())
	if err != nil {
		t.Fatalf("new store: %v", err)
	}

	user := models.NewUser("", "review@example.com", "hash")
	if err := store.CreateUser(ctx, user); err != nil {
		t.Fatalf("create user: %v", err)
	}

	session := &models.GameSession{UserID: user.ID, GameType: "demo", Stake: 10}
	if err := store.StartGameSession(ctx, session); err != nil {
		t.Fatalf("start game: %v", err)
	}

	moves := make([]models.MazeMove, len(session.Lines)+1)
	for i := range moves {
		moves[i] = models.MazeMove{Direction: "missing-line"}
	}
	if _, err := store.SubmitMazeMoves(ctx, user.ID, session.ID, moves); err != nil {
		t.Fatalf("submit suspicious moves: %v", err)
	}

	cases, err := store.ListReviewCases(ctx)
	if err != nil {
		t.Fatalf("list review cases: %v", err)
	}
	if len(cases) != 1 || cases[0].Status != "PENDING_REVIEW" || cases[0].ScopeID != session.ID {
		t.Fatalf("review cases = %#v, want one pending case for session", cases)
	}

	report, err := store.GetReplayReport(ctx, session.ID)
	if err != nil {
		t.Fatalf("replay report: %v", err)
	}
	if report.ReviewStatus != "PENDING_REVIEW" {
		t.Fatalf("review status = %q, want PENDING_REVIEW", report.ReviewStatus)
	}

	metrics, err := store.Metrics(ctx)
	if err != nil {
		t.Fatalf("metrics: %v", err)
	}
	if metrics.CompletedMatchCount == 0 || metrics.TotalFailedClicks == 0 || metrics.ReplayReconstructionCount == 0 {
		t.Fatalf("metrics did not collect completion/replay data: %#v", metrics)
	}
}

func TestPvPQueueMatchesPlayersAndRefundsWhenBothRoutesInvalid(t *testing.T) {
	ctx := context.Background()
	store, err := New(ctx, t.TempDir())
	if err != nil {
		t.Fatalf("new store: %v", err)
	}

	playerA := models.NewUser("", "pvp-a@example.com", "hash")
	playerB := models.NewUser("", "pvp-b@example.com", "hash")
	if err := store.CreateUser(ctx, playerA); err != nil {
		t.Fatalf("create player a: %v", err)
	}
	if err := store.CreateUser(ctx, playerB); err != nil {
		t.Fatalf("create player b: %v", err)
	}

	waiting, err := store.JoinPvPQueue(ctx, playerA.ID, "standard", "demo", 10)
	if err != nil {
		t.Fatalf("player a joins pvp: %v", err)
	}
	if waiting.Match.Status != "waiting" {
		t.Fatalf("first pvp status = %q, want waiting", waiting.Match.Status)
	}

	active, err := store.JoinPvPQueue(ctx, playerB.ID, "standard", "demo", 10)
	if err != nil {
		t.Fatalf("player b joins pvp: %v", err)
	}
	if active.Match.Status != "active" || active.Match.PlayerBID != playerB.ID {
		t.Fatalf("matched pvp = %#v, want active with player b", active.Match)
	}
	if len(active.Match.PlayerBLines) == 0 || len(active.Match.PlayerALines) != 0 || active.Match.PlayerASeed != "" {
		t.Fatalf("player b detail leaked opponent board: playerA lines=%d seed=%q playerB lines=%d", len(active.Match.PlayerALines), active.Match.PlayerASeed, len(active.Match.PlayerBLines))
	}
	playerADetail, err := store.GetPvPMatchDetail(ctx, active.Match.ID, playerA.ID)
	if err != nil {
		t.Fatalf("player a detail: %v", err)
	}
	if len(playerADetail.Match.PlayerALines) == 0 || len(playerADetail.Match.PlayerBLines) != 0 || playerADetail.Match.PlayerBSeed != "" {
		t.Fatalf("player a detail leaked opponent board: playerA lines=%d playerB lines=%d seed=%q", len(playerADetail.Match.PlayerALines), len(playerADetail.Match.PlayerBLines), playerADetail.Match.PlayerBSeed)
	}
	internalMatch := store.pvpMatches[active.Match.ID]
	if internalMatch.PlayerASeed == "" || internalMatch.PlayerBSeed == "" || internalMatch.PlayerASeed != internalMatch.PlayerBSeed {
		t.Fatalf("expected shared internal pvp seed, got %q and %q", internalMatch.PlayerASeed, internalMatch.PlayerBSeed)
	}
	if internalMatch.PlayerAHash == "" || internalMatch.PlayerBHash == "" || internalMatch.PlayerAHash != internalMatch.PlayerBHash {
		t.Fatalf("expected shared pvp generation hash, got %q and %q", internalMatch.PlayerAHash, internalMatch.PlayerBHash)
	}
	if internalMatch.DifficultyProfile == nil || len(internalMatch.PlayerALines) != internalMatch.DifficultyProfile.LineCount || len(internalMatch.PlayerBLines) != internalMatch.DifficultyProfile.LineCount {
		t.Fatal("expected pvp boards to share equivalent difficulty profile line counts")
	}
	if !reflect.DeepEqual(internalMatch.PlayerALines, internalMatch.PlayerBLines) {
		t.Fatal("expected both pvp players to receive identical starting boards")
	}

	walletA, err := store.GetWalletByUserID(ctx, playerA.ID)
	if err != nil {
		t.Fatalf("wallet a locked: %v", err)
	}
	walletB, err := store.GetWalletByUserID(ctx, playerB.ID)
	if err != nil {
		t.Fatalf("wallet b locked: %v", err)
	}
	if walletA.DemoLockedBalance != 10 || walletB.DemoLockedBalance != 10 {
		t.Fatalf("locked stakes = %.2f/%.2f, want 10/10", walletA.DemoLockedBalance, walletB.DemoLockedBalance)
	}

	if _, err := store.SubmitPvPMoves(ctx, playerA.ID, active.Match.ID, []models.MazeMove{{Direction: "left"}}); err != nil {
		t.Fatalf("submit player a pvp moves: %v", err)
	}
	completed, err := store.SubmitPvPMoves(ctx, playerB.ID, active.Match.ID, []models.MazeMove{{Direction: "left"}})
	if err != nil {
		t.Fatalf("submit player b pvp moves: %v", err)
	}
	if completed.Match.Status != "completed" || completed.Match.WinnerID != "" {
		t.Fatalf("completed pvp = status %q winner %q, want completed refund", completed.Match.Status, completed.Match.WinnerID)
	}

	walletA, err = store.GetWalletByUserID(ctx, playerA.ID)
	if err != nil {
		t.Fatalf("wallet a after refund: %v", err)
	}
	walletB, err = store.GetWalletByUserID(ctx, playerB.ID)
	if err != nil {
		t.Fatalf("wallet b after refund: %v", err)
	}
	if walletA.DemoBalance != 1000 || walletA.DemoLockedBalance != 0 || walletB.DemoBalance != 1000 || walletB.DemoLockedBalance != 0 {
		t.Fatalf("wallets after refund = a %.2f/%.2f b %.2f/%.2f, want 1000/0 and 1000/0", walletA.DemoBalance, walletA.DemoLockedBalance, walletB.DemoBalance, walletB.DemoLockedBalance)
	}

	progressionA, err := store.GetProgressionByUserID(ctx, playerA.ID)
	if err != nil {
		t.Fatalf("progression a: %v", err)
	}
	if progressionA.MatchesPlayed != 1 || progressionA.Wins != 0 || progressionA.Losses != 0 {
		t.Fatalf("progression a = matches %d wins %d losses %d, want 1/0/0", progressionA.MatchesPlayed, progressionA.Wins, progressionA.Losses)
	}
}

func TestTournamentMatchSubmissionsCompleteMatch(t *testing.T) {
	ctx := context.Background()
	store, err := New(ctx, t.TempDir())
	if err != nil {
		t.Fatalf("new store: %v", err)
	}

	playerA := models.NewUser("", "tour-a@example.com", "hash")
	playerB := models.NewUser("", "tour-b@example.com", "hash")
	if err := store.CreateUser(ctx, playerA); err != nil {
		t.Fatalf("create player a: %v", err)
	}
	if err := store.CreateUser(ctx, playerB); err != nil {
		t.Fatalf("create player b: %v", err)
	}
	if _, err := store.RegisterTournament(ctx, playerA.ID, "daily-maze-open"); err != nil {
		t.Fatalf("register player a: %v", err)
	}
	if _, err := store.RegisterTournament(ctx, playerB.ID, "daily-maze-open"); err != nil {
		t.Fatalf("register player b: %v", err)
	}

	matches, err := store.GenerateTournamentBracket(ctx, playerA.ID, "daily-maze-open")
	if err != nil {
		t.Fatalf("generate bracket: %v", err)
	}
	if len(matches) != 1 {
		t.Fatalf("matches = %d, want 1", len(matches))
	}
	match := matches[0]
	if len(match.PlayerALines) == 0 || len(match.PlayerBLines) == 0 {
		t.Fatal("generated tournament match is missing player line boards")
	}
	if match.PlayerASeed == "" || match.PlayerASeed != match.PlayerBSeed {
		t.Fatalf("expected shared tournament seed, got %q and %q", match.PlayerASeed, match.PlayerBSeed)
	}
	if match.PlayerAHash == "" || match.PlayerAHash != match.PlayerBHash {
		t.Fatalf("expected shared tournament generation hash, got %q and %q", match.PlayerAHash, match.PlayerBHash)
	}
	if !reflect.DeepEqual(match.PlayerALines, match.PlayerBLines) {
		t.Fatal("expected both tournament players to receive identical starting boards")
	}

	clicksA, ok := game.SolveLinePuzzle(match.PlayerALines)
	if !ok {
		t.Fatal("generated tournament player A puzzle is not solvable")
	}
	clicksB, ok := game.SolveLinePuzzle(match.PlayerBLines)
	if !ok {
		t.Fatal("generated tournament player B puzzle is not solvable")
	}
	if _, err := store.SubmitTournamentMatchClicks(ctx, playerA.ID, "daily-maze-open", match.ID, clicksA); err != nil {
		t.Fatalf("submit player a: %v", err)
	}
	detail, err := store.SubmitTournamentMatchClicks(ctx, playerB.ID, "daily-maze-open", match.ID, clicksB)
	if err != nil {
		t.Fatalf("submit player b: %v", err)
	}
	if len(detail.Matches) != 1 || detail.Matches[0].Status != "completed" || detail.Matches[0].WinnerID == "" {
		t.Fatalf("match after submissions = %#v, want completed with winner", detail.Matches)
	}
	if len(detail.Submissions) != 2 {
		t.Fatalf("submissions = %d, want 2", len(detail.Submissions))
	}
}

func TestCreateUserKeepsNormalUsersAsPlayersWhenSuperAdminsConfigured(t *testing.T) {
	ctx := context.Background()
	store, err := New(ctx, t.TempDir())
	if err != nil {
		t.Fatalf("new store: %v", err)
	}

	first := models.NewUser("", "founder@example.com", "hash")
	second := models.NewUser("", "player@example.com", "hash")
	if err := store.CreateUser(ctx, first); err != nil {
		t.Fatalf("create first user: %v", err)
	}
	if err := store.CreateUser(ctx, second); err != nil {
		t.Fatalf("create second user: %v", err)
	}

	if first.Role != models.RolePlayer {
		t.Fatalf("first role = %q, want player", first.Role)
	}
	if second.Role != models.RolePlayer {
		t.Fatalf("second role = %q, want player", second.Role)
	}
}

func TestConfiguredSuperAdminIsImmutable(t *testing.T) {
	ctx := context.Background()
	store, err := New(ctx, t.TempDir())
	if err != nil {
		t.Fatalf("new store: %v", err)
	}

	super := models.NewUser("", "skillarenagame@gmail.com", "hash")
	target := models.NewUser("", "target-admin@example.com", "hash")
	if err := store.CreateUser(ctx, super); err != nil {
		t.Fatalf("create super admin: %v", err)
	}
	if err := store.CreateUser(ctx, target); err != nil {
		t.Fatalf("create target: %v", err)
	}
	if super.Role != models.RoleSuperAdmin {
		t.Fatalf("super role = %q, want super_admin", super.Role)
	}

	updated, err := store.UpdateUserRole(ctx, super.ID, target.ID, models.RoleAdmin, "127.0.0.1")
	if err != nil {
		t.Fatalf("promote target: %v", err)
	}
	if updated.Role != models.RoleAdmin {
		t.Fatalf("updated role = %q, want admin", updated.Role)
	}
	if _, err := store.UpdateUserRole(ctx, super.ID, super.ID, models.RolePlayer, "127.0.0.1"); err == nil {
		t.Fatal("expected super admin demotion to fail")
	}
}

func TestStartupIntegrityAllowsHashlessConfiguredSuperAdmin(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	usersJSON := `[
		{
			"id": "super-1",
			"email": "skillarenagame@gmail.com",
			"role": "player",
			"emailVerified": false,
			"kycStatus": "unverified",
			"createdAt": "2026-01-01T00:00:00Z"
		}
	]`
	if err := os.WriteFile(filepath.Join(dir, "users.json"), []byte(usersJSON), 0o644); err != nil {
		t.Fatalf("write users: %v", err)
	}

	store, err := New(ctx, dir)
	if err != nil {
		t.Fatalf("new store: %v", err)
	}
	user, err := store.GetUserByEmail(ctx, "skillarenagame@gmail.com")
	if err != nil {
		t.Fatalf("get super admin: %v", err)
	}
	if user.Role != models.RoleSuperAdmin || !user.HiddenFromPublic {
		t.Fatalf("super admin role/public flags = %q/%v, want super_admin/hidden", user.Role, user.HiddenFromPublic)
	}
}

func TestStartupIntegrityRejectsHashlessNormalUser(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	usersJSON := `[
		{
			"id": "player-1",
			"email": "player@example.com",
			"role": "player",
			"emailVerified": false,
			"kycStatus": "unverified",
			"createdAt": "2026-01-01T00:00:00Z"
		}
	]`
	if err := os.WriteFile(filepath.Join(dir, "users.json"), []byte(usersJSON), 0o644); err != nil {
		t.Fatalf("write users: %v", err)
	}

	if _, err := New(ctx, dir); err == nil {
		t.Fatal("expected startup integrity failure for hashless normal user")
	}
}

func TestSystemHealthReturnsOperationalSnapshot(t *testing.T) {
	ctx := context.Background()
	store, err := New(ctx, t.TempDir())
	if err != nil {
		t.Fatalf("new store: %v", err)
	}
	health, err := store.SystemHealth(ctx)
	if err != nil {
		t.Fatalf("system health: %v", err)
	}
	if health.APIStatus != "ok" || health.DatabaseStatus != "ok" || health.CacheHealth == "" {
		t.Fatalf("unexpected health snapshot: %#v", health)
	}
}

func TestBackgroundJobQueuePersistsQueuedJobs(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	store, err := New(ctx, dir)
	if err != nil {
		t.Fatalf("new store: %v", err)
	}
	if _, err := store.EnqueueJob(ctx, models.JobReplayExport, map[string]string{"sessionId": "session-1"}, time.Time{}); err != nil {
		t.Fatalf("enqueue job: %v", err)
	}
	jobs, err := store.ListJobs(ctx, "queued")
	if err != nil {
		t.Fatalf("list jobs: %v", err)
	}
	if len(jobs) != 1 || jobs[0].Type != models.JobReplayExport {
		t.Fatalf("jobs = %#v, want one replay export", jobs)
	}

	reloaded, err := New(ctx, dir)
	if err != nil {
		t.Fatalf("reload store: %v", err)
	}
	reloadedJobs, err := reloaded.ListJobs(ctx, "queued")
	if err != nil {
		t.Fatalf("list reloaded jobs: %v", err)
	}
	if len(reloadedJobs) != 1 {
		t.Fatalf("reloaded jobs = %d, want 1", len(reloadedJobs))
	}
}

func TestBackgroundJobLifecycleAndStats(t *testing.T) {
	ctx := context.Background()
	store, err := New(ctx, t.TempDir())
	if err != nil {
		t.Fatalf("new store: %v", err)
	}

	queued, err := store.EnqueueJob(ctx, models.JobTelemetryAggregation, nil, time.Time{})
	if err != nil {
		t.Fatalf("enqueue job: %v", err)
	}
	claimed, err := store.ClaimNextJob(ctx, "telemetry_worker", []string{models.JobTelemetryAggregation}, time.Now().UTC())
	if err != nil {
		t.Fatalf("claim job: %v", err)
	}
	if claimed == nil || claimed.ID != queued.ID || claimed.Status != models.JobStatusRunning {
		t.Fatalf("claimed job = %#v, want running job %s", claimed, queued.ID)
	}
	if err := store.CompleteJob(ctx, claimed.ID, "analytics/result.json"); err != nil {
		t.Fatalf("complete job: %v", err)
	}
	stats, err := store.QueueStats(ctx)
	if err != nil {
		t.Fatalf("queue stats: %v", err)
	}
	if stats.CompletedJobs != 1 || stats.RunningJobs != 0 {
		t.Fatalf("stats completed/running = %d/%d, want 1/0", stats.CompletedJobs, stats.RunningJobs)
	}
}

func TestRefreshTokenSessionCanBeRevoked(t *testing.T) {
	ctx := context.Background()
	store, err := New(ctx, t.TempDir())
	if err != nil {
		t.Fatalf("new store: %v", err)
	}

	user := models.NewUser("", "player@example.com", "hash")
	if err := store.CreateUser(ctx, user); err != nil {
		t.Fatalf("create user: %v", err)
	}

	refreshToken := NewRefreshToken()
	if _, err := store.CreateAuthSession(ctx, user.ID, refreshToken, "test", "127.0.0.1", 0); err != nil {
		t.Fatalf("create auth session: %v", err)
	}

	if _, _, err := store.GetUserByRefreshToken(ctx, refreshToken); err != nil {
		t.Fatalf("refresh lookup before revoke: %v", err)
	}
	if err := store.RevokeRefreshToken(ctx, refreshToken, user.ID, "127.0.0.1"); err != nil {
		t.Fatalf("revoke refresh token: %v", err)
	}
	if _, _, err := store.GetUserByRefreshToken(ctx, refreshToken); err == nil {
		t.Fatal("expected revoked refresh token to fail")
	}

	logs, err := store.GetAuditLogs(ctx, 10)
	if err != nil {
		t.Fatalf("audit logs: %v", err)
	}
	if len(logs) < 2 {
		t.Fatalf("audit log count = %d, want at least 2", len(logs))
	}
}

func TestStartHouseChallengeUsesTierRules(t *testing.T) {
	ctx := context.Background()
	store, err := New(ctx, t.TempDir())
	if err != nil {
		t.Fatalf("new store: %v", err)
	}

	user := models.NewUser("", "house@example.com", "hash")
	if err := store.CreateUser(ctx, user); err != nil {
		t.Fatalf("create user: %v", err)
	}

	tiers, err := store.ListHouseTiers(ctx, user.ID)
	if err != nil {
		t.Fatalf("list tiers: %v", err)
	}
	if len(tiers) == 0 || tiers[0].ID != "bronze" {
		t.Fatalf("unexpected tiers: %#v", tiers)
	}

	session, tier, err := store.StartHouseChallenge(ctx, user.ID, "bronze", "demo")
	if err != nil {
		t.Fatalf("start house challenge: %v", err)
	}
	if session.Mode != "house" || session.HouseTier != "bronze" {
		t.Fatalf("session mode/tier = %q/%q, want house/bronze", session.Mode, session.HouseTier)
	}
	if session.Stake != tier.Stake || session.RewardRate != tier.RewardRate || session.Difficulty != tier.Difficulty {
		t.Fatalf("session tier rules = stake %.2f rate %.2f difficulty %d, tier %#v", session.Stake, session.RewardRate, session.Difficulty, tier)
	}
	if session.Width <= 11 {
		t.Fatalf("house challenge width = %d, want difficulty-sized maze", session.Width)
	}
}

func TestTreasuryHealthIncludesLiabilities(t *testing.T) {
	ctx := context.Background()
	store, err := New(ctx, t.TempDir())
	if err != nil {
		t.Fatalf("new store: %v", err)
	}

	user := models.NewUser("", "treasury@example.com", "hash")
	if err := store.CreateUser(ctx, user); err != nil {
		t.Fatalf("create user: %v", err)
	}
	if _, err := store.RecordWalletTransaction(ctx, user.ID, models.TransactionTypeDeposit, 100, "USD", "test", nil); err != nil {
		t.Fatalf("deposit: %v", err)
	}

	health, err := store.GetTreasuryHealth(ctx)
	if err != nil {
		t.Fatalf("treasury health: %v", err)
	}
	if health.PlayerLiabilities < 100 {
		t.Fatalf("liabilities = %.2f, want at least 100", health.PlayerLiabilities)
	}
	if !health.IsSolvent {
		t.Fatal("expected default treasury to be solvent")
	}
}

func TestLiveHouseChallengeRequiresTreasuryCoverage(t *testing.T) {
	ctx := context.Background()
	store, err := New(ctx, t.TempDir())
	if err != nil {
		t.Fatalf("new store: %v", err)
	}

	user := models.NewUser("", "risk@example.com", "hash")
	if err := store.CreateUser(ctx, user); err != nil {
		t.Fatalf("create user: %v", err)
	}
	if _, err := store.RecordWalletTransaction(ctx, user.ID, models.TransactionTypeDeposit, 100, "USD", "test", nil); err != nil {
		t.Fatalf("deposit: %v", err)
	}

	store.mu.Lock()
	store.treasury.PlayerReserve = 1
	store.treasury.RevenueReserve = 0
	store.treasury.SeasonReserve = 0
	store.treasury.ChampionshipReserve = 0
	store.treasury.JackpotReserve = 0
	store.treasury.EmergencyReserve = 0
	store.mu.Unlock()

	if _, _, err := store.StartHouseChallenge(ctx, user.ID, "bronze", "live"); err == nil {
		t.Fatal("expected insufficient treasury coverage error")
	}
}

func TestSeasonLeaderboardRanksBySeasonPoints(t *testing.T) {
	ctx := context.Background()
	store, err := New(ctx, t.TempDir())
	if err != nil {
		t.Fatalf("new store: %v", err)
	}

	first := models.NewUser("", "first@example.com", "hash")
	second := models.NewUser("", "second@example.com", "hash")
	if err := store.CreateUser(ctx, first); err != nil {
		t.Fatalf("create first user: %v", err)
	}
	if err := store.CreateUser(ctx, second); err != nil {
		t.Fatalf("create second user: %v", err)
	}

	store.mu.Lock()
	store.profiles[first.ID].SeasonPoints = 25
	store.profiles[second.ID].SeasonPoints = 50
	store.mu.Unlock()

	season, err := store.GetActiveSeason(ctx)
	if err != nil {
		t.Fatalf("active season: %v", err)
	}
	if !season.IsActive || season.ID == "" {
		t.Fatalf("unexpected season: %#v", season)
	}

	leaderboard, err := store.GetSeasonLeaderboard(ctx)
	if err != nil {
		t.Fatalf("season leaderboard: %v", err)
	}
	if len(leaderboard) != 2 {
		t.Fatalf("leaderboard count = %d, want 2", len(leaderboard))
	}
	if leaderboard[0].UserID != second.ID || leaderboard[0].Rank != 1 {
		t.Fatalf("top entry = %#v, want second user rank 1", leaderboard[0])
	}
}

func TestPublicLeaderboardHidesAdminAccountsAndEmails(t *testing.T) {
	ctx := context.Background()
	store, err := New(ctx, t.TempDir())
	if err != nil {
		t.Fatalf("new store: %v", err)
	}

	admin := models.NewUser("", "geldenhuysj0106@gmail.com", "hash")
	player := models.NewUser("", "player@example.com", "hash")
	if err := store.CreateUser(ctx, admin); err != nil {
		t.Fatalf("create admin: %v", err)
	}
	if err := store.CreateUser(ctx, player); err != nil {
		t.Fatalf("create player: %v", err)
	}

	leaderboard, err := store.GetLeaderboard()
	if err != nil {
		t.Fatalf("leaderboard: %v", err)
	}
	if len(leaderboard) != 1 {
		t.Fatalf("leaderboard count = %d, want 1", len(leaderboard))
	}
	if leaderboard[0].UserID != player.ID || leaderboard[0].DisplayName == "" {
		t.Fatalf("leaderboard entry = %#v, want public player display name", leaderboard[0])
	}
}

func TestTournamentRegistrationLocksEntryFee(t *testing.T) {
	ctx := context.Background()
	store, err := New(ctx, t.TempDir())
	if err != nil {
		t.Fatalf("new store: %v", err)
	}

	bootstrap := models.NewUser("", "admin@example.com", "hash")
	if err := store.CreateUser(ctx, bootstrap); err != nil {
		t.Fatalf("create bootstrap user: %v", err)
	}
	user := models.NewUser("", "tournament@example.com", "hash")
	if err := store.CreateUser(ctx, user); err != nil {
		t.Fatalf("create user: %v", err)
	}

	participant, err := store.RegisterTournament(ctx, user.ID, "daily-maze-open")
	if err != nil {
		t.Fatalf("register tournament: %v", err)
	}
	if participant.Seed != 1 || participant.Status != "registered" {
		t.Fatalf("unexpected participant: %#v", participant)
	}

	wallet, err := store.GetWalletByUserID(ctx, user.ID)
	if err != nil {
		t.Fatalf("wallet: %v", err)
	}
	if wallet.DemoLockedBalance != 5 {
		t.Fatalf("demo locked = %.2f, want 5", wallet.DemoLockedBalance)
	}

	if _, err := store.RegisterTournament(ctx, user.ID, "daily-maze-open"); err == nil {
		t.Fatal("expected duplicate registration error")
	}

	detail, err := store.GetTournamentDetail(ctx, "daily-maze-open", user.ID)
	if err != nil {
		t.Fatalf("tournament detail: %v", err)
	}
	if !detail.Registered || len(detail.Participants) != 1 {
		t.Fatalf("detail = %#v, want registered with one participant", detail)
	}
}

func TestTournamentBracketResultSettlesPrize(t *testing.T) {
	ctx := context.Background()
	store, err := New(ctx, t.TempDir())
	if err != nil {
		t.Fatalf("new store: %v", err)
	}

	first := models.NewUser("", "one@example.com", "hash")
	second := models.NewUser("", "two@example.com", "hash")
	if err := store.CreateUser(ctx, first); err != nil {
		t.Fatalf("create first user: %v", err)
	}
	if err := store.CreateUser(ctx, second); err != nil {
		t.Fatalf("create second user: %v", err)
	}
	if _, err := store.RegisterTournament(ctx, first.ID, "daily-maze-open"); err != nil {
		t.Fatalf("register first: %v", err)
	}
	if _, err := store.RegisterTournament(ctx, second.ID, "daily-maze-open"); err != nil {
		t.Fatalf("register second: %v", err)
	}

	matches, err := store.GenerateTournamentBracket(ctx, first.ID, "daily-maze-open")
	if err != nil {
		t.Fatalf("generate bracket: %v", err)
	}
	if len(matches) != 1 || matches[0].Status != "scheduled" {
		t.Fatalf("matches = %#v, want one scheduled match", matches)
	}

	if _, err := store.ReportTournamentMatchResult(ctx, first.ID, "daily-maze-open", matches[0].ID, first.ID); err != nil {
		t.Fatalf("report result: %v", err)
	}

	detail, err := store.GetTournamentDetail(ctx, "daily-maze-open", first.ID)
	if err != nil {
		t.Fatalf("detail: %v", err)
	}
	if detail.Tournament.Status != "completed" {
		t.Fatalf("status = %q, want completed", detail.Tournament.Status)
	}

	winnerWallet, err := store.GetWalletByUserID(ctx, first.ID)
	if err != nil {
		t.Fatalf("winner wallet: %v", err)
	}
	if winnerWallet.DemoLockedBalance != 0 || winnerWallet.DemoBalance != 1245 {
		t.Fatalf("winner wallet demo/locked = %.2f/%.2f, want 1245/0", winnerWallet.DemoBalance, winnerWallet.DemoLockedBalance)
	}

	loserWallet, err := store.GetWalletByUserID(ctx, second.ID)
	if err != nil {
		t.Fatalf("loser wallet: %v", err)
	}
	if loserWallet.DemoLockedBalance != 0 || loserWallet.DemoBalance != 995 {
		t.Fatalf("loser wallet demo/locked = %.2f/%.2f, want 995/0", loserWallet.DemoBalance, loserWallet.DemoLockedBalance)
	}
}

func TestDailyCalibrationUpdatesBaselineWithoutWalletOrProgression(t *testing.T) {
	ctx := context.Background()
	store, err := New(ctx, t.TempDir())
	if err != nil {
		t.Fatalf("new store: %v", err)
	}

	user := models.NewUser("", "calibration@example.com", "hash")
	if err := store.CreateUser(ctx, user); err != nil {
		t.Fatalf("create user: %v", err)
	}

	session, err := store.StartDailyCalibration(ctx, user.ID)
	if err != nil {
		t.Fatalf("start calibration: %v", err)
	}
	if !session.Calibration || session.Stake != 0 {
		t.Fatalf("session calibration/stake = %v/%.2f, want true/0", session.Calibration, session.Stake)
	}

	if _, err := store.SubmitMazeMoves(ctx, user.ID, session.ID, []models.MazeMove{{Direction: "left"}}); err != nil {
		t.Fatalf("finish calibration: %v", err)
	}

	wallet, err := store.GetWalletByUserID(ctx, user.ID)
	if err != nil {
		t.Fatalf("wallet: %v", err)
	}
	if wallet.DemoBalance != 1000 || wallet.DemoLockedBalance != 0 {
		t.Fatalf("wallet demo/locked = %.2f/%.2f, want 1000/0", wallet.DemoBalance, wallet.DemoLockedBalance)
	}

	progression, err := store.GetProgressionByUserID(ctx, user.ID)
	if err != nil {
		t.Fatalf("progression: %v", err)
	}
	if progression.XP != 0 || progression.MatchesPlayed != 0 {
		t.Fatalf("progression xp/matches = %d/%d, want 0/0", progression.XP, progression.MatchesPlayed)
	}

	baseline, err := store.GetBaselineByUserID(ctx, user.ID)
	if err != nil {
		t.Fatalf("baseline: %v", err)
	}
	if baseline.CalibrationRuns != 1 || baseline.LastSessionID != session.ID {
		t.Fatalf("baseline = %#v, want one run for session", baseline)
	}
}
