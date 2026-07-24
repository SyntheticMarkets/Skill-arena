package db

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"skill-arena/internal/models"
	"skill-arena/migrations"
)

func (s *Store) initPostgresFinancial(ctx context.Context) error {
	steps := []struct {
		version string
		sql     string
	}{
		{version: "004_financial_platform", sql: migrations.FinancialPlatform},
		{version: "005_financial_completion", sql: migrations.FinancialCompletion},
	}
	for _, step := range steps {
		if err := s.applyFinancialMigration(ctx, step.version, step.sql); err != nil {
			return err
		}
	}
	return nil
}

func (s *Store) applyFinancialMigration(ctx context.Context, version, migration string) error {
	checksumBytes := sha256.Sum256([]byte(migration))
	checksum := hex.EncodeToString(checksumBytes[:])
	var existing string
	err := s.pg.QueryRowContext(ctx, `SELECT checksum FROM schema_migrations WHERE version=$1`, version).Scan(&existing)
	if err == nil {
		if existing != checksum {
			return fmt.Errorf("migration %s checksum mismatch", version)
		}
		return nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return err
	}
	tx, err := s.pg.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err := tx.ExecContext(ctx, migration); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO schema_migrations(version,checksum,applied_at) VALUES($1,$2,$3)`, version, checksum, time.Now().UTC()); err != nil {
		return err
	}
	return tx.Commit()
}

func defaultFinancialWallet(userID, currency string, now time.Time) *models.FinancialWallet {
	return &models.FinancialWallet{UserID: userID, Currency: currency, Version: 1, UpdatedAt: now}
}

func defaultFinancialAssessment(userID string, now time.Time) *models.FinancialAssessment {
	return &models.FinancialAssessment{
		UserID: userID, Status: models.AssessmentStatusNotStarted,
		RiskClassification: "unassessed", VerificationStatus: "unverified",
		ResponsibleStatus: "active", UpdatedAt: now,
	}
}

func defaultFinancialLimits(userID, currency string, now time.Time) *models.FinancialLimits {
	return &models.FinancialLimits{
		UserID: userID, Currency: currency,
		DailyDepositMinor: 500_00, MonthlyDepositMinor: 5_000_00,
		DailyWithdrawalMinor: 250_00, MonthlyWithdrawalMinor: 2_500_00,
		UpdatedAt: now,
	}
}

func (s *Store) EnsureFinancialAccount(ctx context.Context, userID, currency string) error {
	currency = normalizeCurrency(currency)
	now := time.Now().UTC()
	if s.usesPostgresAuth() {
		tx, err := s.pg.BeginTx(ctx, nil)
		if err != nil {
			return err
		}
		defer tx.Rollback()
		if _, err := tx.ExecContext(ctx, `
INSERT INTO financial_wallets(user_id,currency,updated_at) VALUES($1,$2,$3)
ON CONFLICT(user_id) DO NOTHING`, userID, currency, now); err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, `
INSERT INTO financial_assessments(user_id,status,updated_at) VALUES($1,$2,$3)
ON CONFLICT(user_id) DO NOTHING`, userID, models.AssessmentStatusNotStarted, now); err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, `
INSERT INTO financial_limits(user_id,currency,daily_deposit_minor,monthly_deposit_minor,daily_withdrawal_minor,monthly_withdrawal_minor,updated_at)
VALUES($1,$2,$3,$4,$5,$6,$7) ON CONFLICT(user_id) DO NOTHING`,
			userID, currency, 500_00, 5_000_00, 250_00, 2_500_00, now); err != nil {
			return err
		}
		return tx.Commit()
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.financialWallets[userID] == nil {
		s.financialWallets[userID] = defaultFinancialWallet(userID, currency, now)
	}
	if s.financialAssessments[userID] == nil {
		s.financialAssessments[userID] = defaultFinancialAssessment(userID, now)
	}
	if s.financialLimits[userID] == nil {
		s.financialLimits[userID] = defaultFinancialLimits(userID, currency, now)
	}
	return nil
}

func (s *Store) GetFinancialWallet(ctx context.Context, userID string) (*models.FinancialWallet, error) {
	if err := s.EnsureFinancialAccount(ctx, userID, "ZAR"); err != nil {
		return nil, err
	}
	if s.usesPostgresAuth() {
		var wallet models.FinancialWallet
		err := s.pg.QueryRowContext(ctx, `
SELECT user_id,currency,available_minor,pending_deposit_minor,pending_withdrawal_minor,locked_minor,
       lifetime_deposit_minor,lifetime_withdrawal_minor,version,updated_at
FROM financial_wallets WHERE user_id=$1`, userID).Scan(
			&wallet.UserID, &wallet.Currency, &wallet.AvailableMinor, &wallet.PendingDepositMinor,
			&wallet.PendingWithdrawalMinor, &wallet.LockedMinor, &wallet.LifetimeDepositMinor,
			&wallet.LifetimeWithdrawMinor, &wallet.Version, &wallet.UpdatedAt,
		)
		return &wallet, err
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	copyWallet := *s.financialWallets[userID]
	return &copyWallet, nil
}

func (s *Store) GetFinancialAssessment(ctx context.Context, userID string) (*models.FinancialAssessment, error) {
	if err := s.EnsureFinancialAccount(ctx, userID, "ZAR"); err != nil {
		return nil, err
	}
	if s.usesPostgresAuth() {
		var item models.FinancialAssessment
		err := s.pg.QueryRowContext(ctx, `
SELECT user_id,status,country,occupation,source_of_funds,risk_classification,verification_status,
       responsible_status,submitted_at,updated_at
FROM financial_assessments WHERE user_id=$1`, userID).Scan(
			&item.UserID, &item.Status, &item.Country, &item.Occupation, &item.SourceOfFunds,
			&item.RiskClassification, &item.VerificationStatus, &item.ResponsibleStatus,
			&item.SubmittedAt, &item.UpdatedAt,
		)
		return &item, err
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	copyItem := *s.financialAssessments[userID]
	return &copyItem, nil
}

func (s *Store) SaveFinancialAssessment(ctx context.Context, assessment models.FinancialAssessment) (*models.FinancialAssessment, error) {
	if len(assessment.Country) != 2 || assessment.Occupation == "" || assessment.SourceOfFunds == "" {
		return nil, errors.New("country, occupation, and source of funds are required")
	}
	assessment.Country = strings.ToUpper(assessment.Country)
	assessment.Status = models.AssessmentStatusSubmitted
	assessment.RiskClassification = "pending_review"
	assessment.ResponsibleStatus = "active"
	now := time.Now().UTC()
	assessment.SubmittedAt = &now
	assessment.UpdatedAt = now
	if err := s.EnsureFinancialAccount(ctx, assessment.UserID, "ZAR"); err != nil {
		return nil, err
	}
	if s.usesPostgresAuth() {
		_, err := s.pg.ExecContext(ctx, `
UPDATE financial_assessments SET status=$2,country=$3,occupation=$4,source_of_funds=$5,
 risk_classification=$6,responsible_status=$7,submitted_at=$8,updated_at=$9 WHERE user_id=$1`,
			assessment.UserID, assessment.Status, assessment.Country, assessment.Occupation,
			assessment.SourceOfFunds, assessment.RiskClassification, assessment.ResponsibleStatus,
			assessment.SubmittedAt, assessment.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		return s.GetFinancialAssessment(ctx, assessment.UserID)
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	assessment.VerificationStatus = s.financialAssessments[assessment.UserID].VerificationStatus
	copyItem := assessment
	s.financialAssessments[assessment.UserID] = &copyItem
	return &copyItem, nil
}

func (s *Store) ReviewFinancialAssessment(ctx context.Context, userID, decision, risk, verification string) (*models.FinancialAssessment, error) {
	if decision != models.AssessmentStatusComplete && decision != models.AssessmentStatusRestricted && decision != models.AssessmentStatusInReview {
		return nil, errors.New("assessment decision is invalid")
	}
	if risk == "" {
		risk = "standard"
	}
	if verification == "" {
		verification = "pending"
	}
	now := time.Now().UTC()
	if err := s.EnsureFinancialAccount(ctx, userID, "ZAR"); err != nil {
		return nil, err
	}
	if s.usesPostgresAuth() {
		result, err := s.pg.ExecContext(ctx, `
UPDATE financial_assessments SET status=$2,risk_classification=$3,verification_status=$4,updated_at=$5
WHERE user_id=$1 AND status <> 'not_started'`, userID, decision, risk, verification, now)
		if err != nil {
			return nil, err
		}
		if affected, _ := result.RowsAffected(); affected != 1 {
			return nil, errors.New("assessment must be submitted before review")
		}
		return s.GetFinancialAssessment(ctx, userID)
	}
	s.mu.Lock()
	item := s.financialAssessments[userID]
	if item == nil || item.Status == models.AssessmentStatusNotStarted {
		s.mu.Unlock()
		return nil, errors.New("assessment must be submitted before review")
	}
	item.Status, item.RiskClassification, item.VerificationStatus, item.UpdatedAt = decision, risk, verification, now
	copyItem := *item
	s.mu.Unlock()
	return &copyItem, nil
}

func (s *Store) GetFinancialLimits(ctx context.Context, userID string) (*models.FinancialLimits, error) {
	if err := s.EnsureFinancialAccount(ctx, userID, "ZAR"); err != nil {
		return nil, err
	}
	var limits models.FinancialLimits
	if s.usesPostgresAuth() {
		err := s.pg.QueryRowContext(ctx, `
SELECT user_id,currency,daily_deposit_minor,monthly_deposit_minor,daily_withdrawal_minor,
 monthly_withdrawal_minor,cooling_off_until,self_excluded_until,updated_at
FROM financial_limits WHERE user_id=$1`, userID).Scan(
			&limits.UserID, &limits.Currency, &limits.DailyDepositMinor, &limits.MonthlyDepositMinor,
			&limits.DailyWithdrawalMinor, &limits.MonthlyWithdrawalMinor, &limits.CoolingOffUntil,
			&limits.SelfExcludedUntil, &limits.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		usage, err := s.financialUsagePostgres(ctx, userID)
		if err != nil {
			return nil, err
		}
		applyFinancialUsage(&limits, usage)
		return &limits, nil
	}
	s.mu.RLock()
	limits = *s.financialLimits[userID]
	deposits := financialDepositsForUser(s.financialDeposits, userID)
	withdrawals := financialWithdrawalsForUser(s.financialWithdrawals, userID)
	s.mu.RUnlock()
	applyFinancialUsage(&limits, calculateFinancialUsage(deposits, withdrawals, time.Now().UTC()))
	return &limits, nil
}

func (s *Store) SaveFinancialLimits(ctx context.Context, limits models.FinancialLimits) (*models.FinancialLimits, error) {
	if limits.DailyDepositMinor < 0 || limits.MonthlyDepositMinor < limits.DailyDepositMinor ||
		limits.DailyWithdrawalMinor < 0 || limits.MonthlyWithdrawalMinor < limits.DailyWithdrawalMinor {
		return nil, errors.New("limits must be non-negative and monthly limits cannot be below daily limits")
	}
	if err := s.EnsureFinancialAccount(ctx, limits.UserID, limits.Currency); err != nil {
		return nil, err
	}
	limits.UpdatedAt = time.Now().UTC()
	if s.usesPostgresAuth() {
		_, err := s.pg.ExecContext(ctx, `
UPDATE financial_limits SET currency=$2,daily_deposit_minor=$3,monthly_deposit_minor=$4,daily_withdrawal_minor=$5,
 monthly_withdrawal_minor=$6,cooling_off_until=$7,self_excluded_until=$8,updated_at=$9 WHERE user_id=$1`,
			limits.UserID, normalizeCurrency(limits.Currency), limits.DailyDepositMinor, limits.MonthlyDepositMinor,
			limits.DailyWithdrawalMinor, limits.MonthlyWithdrawalMinor,
			limits.CoolingOffUntil, limits.SelfExcludedUntil, limits.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		return s.GetFinancialLimits(ctx, limits.UserID)
	}
	s.mu.Lock()
	copyLimits := limits
	s.financialLimits[limits.UserID] = &copyLimits
	s.mu.Unlock()
	return s.GetFinancialLimits(ctx, limits.UserID)
}

func (s *Store) CreateFinancialDeposit(ctx context.Context, deposit *models.FinancialDeposit) (*models.FinancialDeposit, bool, error) {
	if deposit == nil || deposit.UserID == "" || deposit.AmountMinor <= 0 || deposit.IdempotencyKey == "" {
		return nil, false, errors.New("valid deposit and idempotency key are required")
	}
	deposit.Currency = normalizeCurrency(deposit.Currency)
	deposit.ID = newUUID()
	deposit.Status = models.DepositStatusRequested
	now := time.Now().UTC()
	deposit.RequestedAt, deposit.UpdatedAt = now, now
	if err := s.EnsureFinancialAccount(ctx, deposit.UserID, deposit.Currency); err != nil {
		return nil, false, err
	}
	if s.usesPostgresAuth() {
		tx, err := s.pg.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelSerializable})
		if err != nil {
			return nil, false, err
		}
		defer tx.Rollback()
		existing, err := loadFinancialDepositByKey(ctx, tx, deposit.UserID, deposit.IdempotencyKey)
		if err == nil {
			if existing.RequestHash != deposit.RequestHash {
				return nil, false, errors.New("idempotency key was already used with a different request")
			}
			return existing, true, nil
		}
		if !errors.Is(err, sql.ErrNoRows) {
			return nil, false, err
		}
		metadata, _ := json.Marshal(deposit.Metadata)
		if _, err := tx.ExecContext(ctx, `
INSERT INTO financial_deposits(id,user_id,amount_minor,currency,method,provider,status,idempotency_key,request_hash,metadata,requested_at,updated_at)
VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12)`,
			deposit.ID, deposit.UserID, deposit.AmountMinor, deposit.Currency, deposit.Method,
			deposit.Provider, deposit.Status, deposit.IdempotencyKey, deposit.RequestHash,
			metadata, deposit.RequestedAt, deposit.UpdatedAt,
		); err != nil {
			return nil, false, err
		}
		if err := insertFinancialTransition(ctx, tx, "deposit", deposit.ID, "", deposit.Status, "player", deposit.UserID, "", "", nil); err != nil {
			return nil, false, err
		}
		if err := tx.Commit(); err != nil {
			return nil, false, err
		}
		return deposit, false, nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, existing := range s.financialDeposits {
		if existing.UserID == deposit.UserID && existing.IdempotencyKey == deposit.IdempotencyKey {
			if existing.RequestHash != deposit.RequestHash {
				return nil, false, errors.New("idempotency key was already used with a different request")
			}
			copyItem := *existing
			return &copyItem, true, nil
		}
	}
	copyItem := *deposit
	s.financialDeposits[deposit.ID] = &copyItem
	return &copyItem, false, nil
}

func (s *Store) SetFinancialDepositProviderSession(ctx context.Context, depositID, providerRef, checkoutURL string) (*models.FinancialDeposit, error) {
	now := time.Now().UTC()
	if s.usesPostgresAuth() {
		tx, err := s.pg.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelSerializable})
		if err != nil {
			return nil, err
		}
		defer tx.Rollback()
		deposit, err := loadFinancialDepositForUpdate(ctx, tx, depositID)
		if err != nil {
			return nil, err
		}
		if deposit.Status != models.DepositStatusRequested {
			return nil, fmt.Errorf("deposit cannot move from %s to pending_provider", deposit.Status)
		}
		if _, err := tx.ExecContext(ctx, `
UPDATE financial_deposits SET provider_reference=$2,checkout_url=$3,status=$4,updated_at=$5 WHERE id=$1`,
			depositID, providerRef, checkoutURL, models.DepositStatusPendingProvider, now); err != nil {
			return nil, err
		}
		if _, err := tx.ExecContext(ctx, `
UPDATE financial_wallets SET pending_deposit_minor=pending_deposit_minor+$2,version=version+1,updated_at=$3 WHERE user_id=$1`,
			deposit.UserID, deposit.AmountMinor, now); err != nil {
			return nil, err
		}
		if err := insertFinancialTransition(ctx, tx, "deposit", depositID, deposit.Status, models.DepositStatusPendingProvider, "system", "", "", "", nil); err != nil {
			return nil, err
		}
		if err := tx.Commit(); err != nil {
			return nil, err
		}
		return s.GetFinancialDeposit(ctx, depositID)
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	deposit := s.financialDeposits[depositID]
	if deposit == nil {
		return nil, errors.New("deposit not found")
	}
	if deposit.Status != models.DepositStatusRequested {
		return nil, fmt.Errorf("deposit cannot move from %s to pending_provider", deposit.Status)
	}
	deposit.ProviderReference, deposit.CheckoutURL = providerRef, checkoutURL
	deposit.Status, deposit.UpdatedAt = models.DepositStatusPendingProvider, now
	wallet := s.financialWallets[deposit.UserID]
	wallet.PendingDepositMinor += deposit.AmountMinor
	wallet.Version++
	wallet.UpdatedAt = now
	copyItem := *deposit
	return &copyItem, nil
}

func (s *Store) AdvanceFinancialDeposit(ctx context.Context, depositID, targetStatus, providerEventID string) (*models.FinancialDeposit, error) {
	path := []string{
		models.DepositStatusPendingProvider,
		models.DepositStatusPendingVerification,
	}
	targetIndex := stringIndex(path, targetStatus)
	if targetIndex < 0 {
		return nil, fmt.Errorf("invalid deposit target status %s", targetStatus)
	}
	now := time.Now().UTC()
	if s.usesPostgresAuth() {
		tx, err := s.pg.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelSerializable})
		if err != nil {
			return nil, err
		}
		defer tx.Rollback()
		deposit, err := loadFinancialDepositForUpdate(ctx, tx, depositID)
		if err != nil {
			return nil, err
		}
		currentIndex := stringIndex(path, deposit.Status)
		if currentIndex < 0 || currentIndex > targetIndex {
			return nil, fmt.Errorf("deposit cannot move from %s to %s", deposit.Status, targetStatus)
		}
		for index := currentIndex + 1; index <= targetIndex; index++ {
			from, to := path[index-1], path[index]
			if err := insertFinancialTransition(ctx, tx, "deposit", depositID, from, to, "provider", deposit.Provider, "Signed provider status", providerEventID, nil); err != nil {
				return nil, err
			}
		}
		if _, err := tx.ExecContext(ctx, `UPDATE financial_deposits SET status=$2,updated_at=$3 WHERE id=$1`, depositID, targetStatus, now); err != nil {
			return nil, err
		}
		if err := tx.Commit(); err != nil {
			return nil, err
		}
		return s.GetFinancialDeposit(ctx, depositID)
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	deposit := s.financialDeposits[depositID]
	if deposit == nil {
		return nil, errors.New("deposit not found")
	}
	currentIndex := stringIndex(path, deposit.Status)
	if currentIndex < 0 || currentIndex > targetIndex {
		return nil, fmt.Errorf("deposit cannot move from %s to %s", deposit.Status, targetStatus)
	}
	deposit.Status, deposit.UpdatedAt = targetStatus, now
	copyItem := *deposit
	return &copyItem, nil
}

func (s *Store) FailFinancialDeposit(ctx context.Context, depositID, reason string) error {
	return s.endFinancialDeposit(ctx, depositID, models.DepositStatusFailed, reason)
}

func (s *Store) ExpireFinancialDeposit(ctx context.Context, depositID, reason string) error {
	return s.endFinancialDeposit(ctx, depositID, models.DepositStatusExpired, reason)
}

func (s *Store) endFinancialDeposit(ctx context.Context, depositID, targetStatus, reason string) error {
	if targetStatus != models.DepositStatusFailed && targetStatus != models.DepositStatusExpired {
		return errors.New("invalid terminal deposit status")
	}
	now := time.Now().UTC()
	if s.usesPostgresAuth() {
		tx, err := s.pg.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelSerializable})
		if err != nil {
			return err
		}
		defer tx.Rollback()
		deposit, err := loadFinancialDepositForUpdate(ctx, tx, depositID)
		if err != nil {
			return err
		}
		if deposit.Status == models.DepositStatusCompleted || deposit.Status == models.DepositStatusFailed || deposit.Status == models.DepositStatusExpired {
			return nil
		}
		if deposit.Status != models.DepositStatusRequested {
			if _, err := tx.ExecContext(ctx, `UPDATE financial_wallets SET pending_deposit_minor=GREATEST(0,pending_deposit_minor-$2),version=version+1,updated_at=$3 WHERE user_id=$1`, deposit.UserID, deposit.AmountMinor, now); err != nil {
				return err
			}
		}
		if _, err := tx.ExecContext(ctx, `UPDATE financial_deposits SET status=$2,updated_at=$3 WHERE id=$1`, depositID, targetStatus, now); err != nil {
			return err
		}
		if err := insertFinancialTransition(ctx, tx, "deposit", depositID, deposit.Status, targetStatus, "system", "", reason, "", nil); err != nil {
			return err
		}
		return tx.Commit()
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	deposit := s.financialDeposits[depositID]
	if deposit == nil {
		return errors.New("deposit not found")
	}
	if deposit.Status != models.DepositStatusRequested && deposit.Status != models.DepositStatusFailed && deposit.Status != models.DepositStatusExpired {
		s.financialWallets[deposit.UserID].PendingDepositMinor -= deposit.AmountMinor
	}
	deposit.Status, deposit.UpdatedAt = targetStatus, now
	return nil
}

func (s *Store) SettleFinancialDeposit(ctx context.Context, depositID, providerEventID string) (*models.FinancialDeposit, error) {
	now := time.Now().UTC()
	if s.usesPostgresAuth() {
		tx, err := s.pg.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelSerializable})
		if err != nil {
			return nil, err
		}
		defer tx.Rollback()
		deposit, err := loadFinancialDepositForUpdate(ctx, tx, depositID)
		if err != nil {
			return nil, err
		}
		if deposit.Status == models.DepositStatusCompleted {
			return deposit, nil
		}
		if deposit.Status != models.DepositStatusPendingVerification {
			return nil, fmt.Errorf("deposit cannot settle from %s", deposit.Status)
		}
		var balance models.MinorUnits
		if err := tx.QueryRowContext(ctx, `
UPDATE financial_wallets SET available_minor=available_minor+$2,
 pending_deposit_minor=GREATEST(0,pending_deposit_minor-$2),lifetime_deposit_minor=lifetime_deposit_minor+$2,
 version=version+1,updated_at=$3 WHERE user_id=$1 RETURNING available_minor`,
			deposit.UserID, deposit.AmountMinor, now).Scan(&balance); err != nil {
			return nil, err
		}
		if _, err := tx.ExecContext(ctx, `UPDATE financial_deposits SET status=$2,updated_at=$3,completed_at=$3 WHERE id=$1`, depositID, models.DepositStatusCompleted, now); err != nil {
			return nil, err
		}
		if err := appendFinancialJournal(ctx, tx, deposit.UserID, "player_available", "credit", deposit.AmountMinor, deposit.Currency, balance, "deposit", deposit.ID, "Deposit settled", deposit.Metadata, now); err != nil {
			return nil, err
		}
		if err := insertFinancialTransition(ctx, tx, "deposit", depositID, deposit.Status, models.DepositStatusCompleted, "provider", deposit.Provider, "Provider settlement verified", providerEventID, nil); err != nil {
			return nil, err
		}
		if err := tx.Commit(); err != nil {
			return nil, err
		}
		return s.GetFinancialDeposit(ctx, depositID)
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	deposit := s.financialDeposits[depositID]
	if deposit == nil {
		return nil, errors.New("deposit not found")
	}
	if deposit.Status == models.DepositStatusCompleted {
		copyItem := *deposit
		return &copyItem, nil
	}
	wallet := s.financialWallets[deposit.UserID]
	wallet.PendingDepositMinor -= deposit.AmountMinor
	wallet.AvailableMinor += deposit.AmountMinor
	wallet.LifetimeDepositMinor += deposit.AmountMinor
	wallet.Version++
	wallet.UpdatedAt = now
	deposit.Status, deposit.UpdatedAt, deposit.CompletedAt = models.DepositStatusCompleted, now, &now
	s.appendFinancialJournalMemory(deposit.UserID, "player_available", "credit", deposit.AmountMinor, deposit.Currency, wallet.AvailableMinor, "deposit", deposit.ID, "Deposit settled", deposit.Metadata, now)
	copyItem := *deposit
	return &copyItem, nil
}

func (s *Store) CreateFinancialWithdrawal(ctx context.Context, withdrawal *models.FinancialWithdrawal) (*models.FinancialWithdrawal, bool, error) {
	if withdrawal == nil || withdrawal.UserID == "" || withdrawal.AmountMinor <= 0 || withdrawal.IdempotencyKey == "" {
		return nil, false, errors.New("valid withdrawal and idempotency key are required")
	}
	withdrawal.ID = newUUID()
	withdrawal.Currency = normalizeCurrency(withdrawal.Currency)
	withdrawal.Status = models.FinancialWithdrawalStatusPendingReview
	if withdrawal.PolicyDecision == "" {
		withdrawal.PolicyDecision = "manual_review"
	}
	if len(withdrawal.PolicyReasons) == 0 {
		withdrawal.PolicyReasons = []string{"Manual review is required during the initial financial policy phase."}
	}
	now := time.Now().UTC()
	withdrawal.RequestedAt, withdrawal.UpdatedAt = now, now
	if err := s.EnsureFinancialAccount(ctx, withdrawal.UserID, withdrawal.Currency); err != nil {
		return nil, false, err
	}
	if s.usesPostgresAuth() {
		tx, err := s.pg.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelSerializable})
		if err != nil {
			return nil, false, err
		}
		defer tx.Rollback()
		existing, err := loadFinancialWithdrawalByKey(ctx, tx, withdrawal.UserID, withdrawal.IdempotencyKey)
		if err == nil {
			if existing.RequestHash != withdrawal.RequestHash {
				return nil, false, errors.New("idempotency key was already used with a different request")
			}
			return existing, true, nil
		}
		if !errors.Is(err, sql.ErrNoRows) {
			return nil, false, err
		}
		var available models.MinorUnits
		if err := tx.QueryRowContext(ctx, `SELECT available_minor FROM financial_wallets WHERE user_id=$1 FOR UPDATE`, withdrawal.UserID).Scan(&available); err != nil {
			return nil, false, err
		}
		total := withdrawal.AmountMinor + withdrawal.FeeMinor
		if available < total {
			return nil, false, errors.New("insufficient available balance")
		}
		reasons, _ := json.Marshal(withdrawal.PolicyReasons)
		metadata, _ := json.Marshal(withdrawal.Metadata)
		if _, err := tx.ExecContext(ctx, `
INSERT INTO financial_withdrawals(id,user_id,amount_minor,fee_minor,currency,method,provider,status,policy_decision,
 policy_reasons,idempotency_key,request_hash,metadata,requested_at,updated_at)
VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15)`,
			withdrawal.ID, withdrawal.UserID, withdrawal.AmountMinor, withdrawal.FeeMinor,
			withdrawal.Currency, withdrawal.Method, withdrawal.Provider, withdrawal.Status,
			withdrawal.PolicyDecision, reasons, withdrawal.IdempotencyKey, withdrawal.RequestHash,
			metadata, withdrawal.RequestedAt, withdrawal.UpdatedAt,
		); err != nil {
			return nil, false, err
		}
		if _, err := tx.ExecContext(ctx, `
UPDATE financial_wallets SET available_minor=available_minor-$2,pending_withdrawal_minor=pending_withdrawal_minor+$2,
 version=version+1,updated_at=$3 WHERE user_id=$1`, withdrawal.UserID, total, now); err != nil {
			return nil, false, err
		}
		if err := insertFinancialTransition(ctx, tx, "withdrawal", withdrawal.ID, models.FinancialWithdrawalStatusRequested, withdrawal.Status, "policy", "", strings.Join(withdrawal.PolicyReasons, " "), "", nil); err != nil {
			return nil, false, err
		}
		if err := tx.Commit(); err != nil {
			return nil, false, err
		}
		return withdrawal, false, nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, existing := range s.financialWithdrawals {
		if existing.UserID == withdrawal.UserID && existing.IdempotencyKey == withdrawal.IdempotencyKey {
			if existing.RequestHash != withdrawal.RequestHash {
				return nil, false, errors.New("idempotency key was already used with a different request")
			}
			copyItem := *existing
			return &copyItem, true, nil
		}
	}
	wallet := s.financialWallets[withdrawal.UserID]
	total := withdrawal.AmountMinor + withdrawal.FeeMinor
	if wallet.AvailableMinor < total {
		return nil, false, errors.New("insufficient available balance")
	}
	wallet.AvailableMinor -= total
	wallet.PendingWithdrawalMinor += total
	wallet.Version++
	wallet.UpdatedAt = now
	copyItem := *withdrawal
	s.financialWithdrawals[withdrawal.ID] = &copyItem
	return &copyItem, false, nil
}

func (s *Store) TransitionFinancialWithdrawal(ctx context.Context, withdrawalID, toStatus, actorType, actorID, reason, providerRef string) (*models.FinancialWithdrawal, error) {
	if s.usesPostgresAuth() {
		return s.transitionFinancialWithdrawalPostgres(ctx, withdrawalID, toStatus, actorType, actorID, reason, providerRef)
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	item := s.financialWithdrawals[withdrawalID]
	if item == nil {
		return nil, errors.New("withdrawal not found")
	}
	if item.Status == toStatus {
		copyItem := *item
		return &copyItem, nil
	}
	if !validWithdrawalTransition(item.Status, toStatus) {
		return nil, fmt.Errorf("withdrawal cannot move from %s to %s", item.Status, toStatus)
	}
	now := time.Now().UTC()
	wallet := s.financialWallets[item.UserID]
	total := item.AmountMinor + item.FeeMinor
	if toStatus == models.FinancialWithdrawalStatusRejected || toStatus == models.FinancialWithdrawalStatusFailed {
		wallet.PendingWithdrawalMinor -= total
		wallet.AvailableMinor += total
	}
	if toStatus == models.FinancialWithdrawalStatusCompleted {
		wallet.PendingWithdrawalMinor -= total
		wallet.LifetimeWithdrawMinor += item.AmountMinor
		s.appendFinancialJournalMemory(item.UserID, "player_available", "debit", total, item.Currency, wallet.AvailableMinor, "withdrawal", item.ID, "Withdrawal settled", item.Metadata, now)
		item.CompletedAt = &now
	}
	wallet.Version++
	wallet.UpdatedAt = now
	item.Status, item.UpdatedAt = toStatus, now
	if providerRef != "" {
		item.ProviderReference = providerRef
	}
	copyItem := *item
	return &copyItem, nil
}

func (s *Store) transitionFinancialWithdrawalPostgres(ctx context.Context, withdrawalID, toStatus, actorType, actorID, reason, providerRef string) (*models.FinancialWithdrawal, error) {
	tx, err := s.pg.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelSerializable})
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	item, err := loadFinancialWithdrawalForUpdate(ctx, tx, withdrawalID)
	if err != nil {
		return nil, err
	}
	if item.Status == toStatus {
		return item, nil
	}
	if !validWithdrawalTransition(item.Status, toStatus) {
		return nil, fmt.Errorf("withdrawal cannot move from %s to %s", item.Status, toStatus)
	}
	now := time.Now().UTC()
	total := item.AmountMinor + item.FeeMinor
	var completedAt *time.Time
	if toStatus == models.FinancialWithdrawalStatusRejected || toStatus == models.FinancialWithdrawalStatusFailed {
		if _, err := tx.ExecContext(ctx, `
UPDATE financial_wallets SET pending_withdrawal_minor=GREATEST(0,pending_withdrawal_minor-$2),
 available_minor=available_minor+$2,version=version+1,updated_at=$3 WHERE user_id=$1`, item.UserID, total, now); err != nil {
			return nil, err
		}
	}
	if toStatus == models.FinancialWithdrawalStatusCompleted {
		var balance models.MinorUnits
		if err := tx.QueryRowContext(ctx, `
UPDATE financial_wallets SET pending_withdrawal_minor=GREATEST(0,pending_withdrawal_minor-$2),
 lifetime_withdrawal_minor=lifetime_withdrawal_minor+$3,version=version+1,updated_at=$4
WHERE user_id=$1 RETURNING available_minor`, item.UserID, total, item.AmountMinor, now).Scan(&balance); err != nil {
			return nil, err
		}
		if err := appendFinancialJournal(ctx, tx, item.UserID, "player_available", "debit", total, item.Currency, balance, "withdrawal", item.ID, "Withdrawal settled", item.Metadata, now); err != nil {
			return nil, err
		}
		completedAt = &now
	}
	if _, err := tx.ExecContext(ctx, `
UPDATE financial_withdrawals SET status=$2,provider_reference=COALESCE(NULLIF($3,''),provider_reference),
 updated_at=$4,completed_at=COALESCE($5,completed_at) WHERE id=$1`,
		withdrawalID, toStatus, providerRef, now, completedAt); err != nil {
		return nil, err
	}
	if err := insertFinancialTransition(ctx, tx, "withdrawal", withdrawalID, item.Status, toStatus, actorType, actorID, reason, "", nil); err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return s.GetFinancialWithdrawal(ctx, withdrawalID)
}

func validWithdrawalTransition(from, to string) bool {
	allowed := map[string]map[string]bool{
		models.FinancialWithdrawalStatusPendingReview: {models.FinancialWithdrawalStatusApproved: true, models.FinancialWithdrawalStatusRejected: true},
		models.FinancialWithdrawalStatusApproved:      {models.FinancialWithdrawalStatusProcessing: true, models.FinancialWithdrawalStatusRejected: true},
		models.FinancialWithdrawalStatusProcessing:    {models.FinancialWithdrawalStatusCompleted: true, models.FinancialWithdrawalStatusFailed: true},
	}
	return allowed[from][to]
}

func (s *Store) GetFinancialDeposit(ctx context.Context, id string) (*models.FinancialDeposit, error) {
	if s.usesPostgresAuth() {
		return scanFinancialDeposit(s.pg.QueryRowContext(ctx, financialDepositSelect+` WHERE id=$1`, id))
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	item := s.financialDeposits[id]
	if item == nil {
		return nil, errors.New("deposit not found")
	}
	copyItem := *item
	return &copyItem, nil
}

func (s *Store) GetFinancialWithdrawal(ctx context.Context, id string) (*models.FinancialWithdrawal, error) {
	if s.usesPostgresAuth() {
		return scanFinancialWithdrawal(s.pg.QueryRowContext(ctx, financialWithdrawalSelect+` WHERE id=$1`, id))
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	item := s.financialWithdrawals[id]
	if item == nil {
		return nil, errors.New("withdrawal not found")
	}
	copyItem := *item
	return &copyItem, nil
}

func (s *Store) ListFinancialDeposits(ctx context.Context, userID string) ([]models.FinancialDeposit, error) {
	if s.usesPostgresAuth() {
		rows, err := s.pg.QueryContext(ctx, financialDepositSelect+` WHERE user_id=$1 ORDER BY requested_at DESC LIMIT 100`, userID)
		if err != nil {
			return nil, err
		}
		defer rows.Close()
		items := []models.FinancialDeposit{}
		for rows.Next() {
			item, err := scanFinancialDeposit(rows)
			if err != nil {
				return nil, err
			}
			items = append(items, *item)
		}
		return items, rows.Err()
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	return financialDepositsForUser(s.financialDeposits, userID), nil
}

func (s *Store) ListFinancialWithdrawals(ctx context.Context, userID string) ([]models.FinancialWithdrawal, error) {
	if s.usesPostgresAuth() {
		rows, err := s.pg.QueryContext(ctx, financialWithdrawalSelect+` WHERE user_id=$1 ORDER BY requested_at DESC LIMIT 100`, userID)
		if err != nil {
			return nil, err
		}
		defer rows.Close()
		items := []models.FinancialWithdrawal{}
		for rows.Next() {
			item, err := scanFinancialWithdrawal(rows)
			if err != nil {
				return nil, err
			}
			items = append(items, *item)
		}
		return items, rows.Err()
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	return financialWithdrawalsForUser(s.financialWithdrawals, userID), nil
}

func (s *Store) ListFinancialJournal(ctx context.Context, userID string, from, to time.Time) ([]models.FinancialLedgerEntry, error) {
	if s.usesPostgresAuth() {
		rows, err := s.pg.QueryContext(ctx, `
SELECT id,user_id,account,direction,amount_minor,currency,balance_after_minor,reference_type,reference_id,
 description,sequence,previous_hash,entry_hash,metadata,created_at
FROM financial_journal WHERE user_id=$1 AND created_at >= $2 AND created_at < $3 ORDER BY sequence`, userID, from, to)
		if err != nil {
			return nil, err
		}
		defer rows.Close()
		entries := []models.FinancialLedgerEntry{}
		for rows.Next() {
			var item models.FinancialLedgerEntry
			var metadata []byte
			if err := rows.Scan(&item.ID, &item.UserID, &item.Account, &item.Direction, &item.AmountMinor,
				&item.Currency, &item.BalanceAfterMinor, &item.ReferenceType, &item.ReferenceID,
				&item.Description, &item.Sequence, &item.PreviousHash, &item.EntryHash, &metadata, &item.CreatedAt); err != nil {
				return nil, err
			}
			_ = json.Unmarshal(metadata, &item.Metadata)
			entries = append(entries, item)
		}
		return entries, rows.Err()
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	entries := []models.FinancialLedgerEntry{}
	for _, entry := range s.financialJournal[userID] {
		if !entry.CreatedAt.Before(from) && entry.CreatedAt.Before(to) {
			entries = append(entries, entry)
		}
	}
	return entries, nil
}

func (s *Store) FinancialStatement(ctx context.Context, userID string, from, to time.Time) (*models.FinancialStatement, error) {
	entries, err := s.ListFinancialJournal(ctx, userID, from, to)
	if err != nil {
		return nil, err
	}
	statement := &models.FinancialStatement{
		ID: newUUID(), PeriodStart: from, PeriodEnd: to, Currency: "ZAR",
		Entries: entries, GeneratedAt: time.Now().UTC(),
	}
	for _, entry := range entries {
		if entry.Direction == "credit" {
			statement.TotalCreditMinor += entry.AmountMinor
		} else {
			statement.TotalDebitMinor += entry.AmountMinor
		}
		statement.ClosingMinor = entry.BalanceAfterMinor
	}
	statement.OpeningMinor = statement.ClosingMinor - statement.TotalCreditMinor + statement.TotalDebitMinor
	return statement, nil
}

func (s *Store) VerifyFinancialJournal(ctx context.Context, userID string) error {
	entries, err := s.ListFinancialJournal(ctx, userID, time.Unix(0, 0), time.Now().UTC().Add(time.Second))
	if err != nil {
		return err
	}
	previous := ""
	for _, entry := range entries {
		if entry.PreviousHash != previous {
			return fmt.Errorf("journal chain broken at sequence %d", entry.Sequence)
		}
		expected := financialEntryHash(previous, entry.UserID, entry.Account, entry.Direction, entry.AmountMinor, entry.Currency, entry.BalanceAfterMinor, entry.ReferenceType, entry.ReferenceID, entry.CreatedAt)
		if entry.EntryHash != expected {
			return fmt.Errorf("journal hash mismatch at sequence %d", entry.Sequence)
		}
		previous = entry.EntryHash
	}
	return nil
}

func (s *Store) RecordFinancialWebhook(ctx context.Context, provider, eventID, signatureHash, payloadHash, resourceType, resourceID string) (bool, error) {
	key := provider + ":" + eventID
	if s.usesPostgresAuth() {
		var accepted bool
		err := s.pg.QueryRowContext(ctx, `
INSERT INTO payment_webhook_events(id,provider,provider_event_id,signature_hash,payload_hash,resource_type,resource_id,outcome,received_at)
VALUES($1,$2,$3,$4,$5,$6,$7,'processing',$8)
ON CONFLICT(provider,provider_event_id) DO UPDATE
SET outcome='processing',received_at=EXCLUDED.received_at,processed_at=NULL
WHERE payment_webhook_events.outcome='failed'
RETURNING TRUE`,
			newUUID(), provider, eventID, signatureHash, payloadHash, resourceType, resourceID, time.Now().UTC()).Scan(&accepted)
		if errors.Is(err, sql.ErrNoRows) {
			return false, nil
		}
		if err != nil {
			return false, err
		}
		return accepted, nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if outcome, exists := s.financialWebhooks[key]; exists && outcome != "failed" {
		return false, nil
	}
	s.financialWebhooks[key] = "processing"
	return true, nil
}

func (s *Store) CompleteFinancialWebhook(ctx context.Context, provider, eventID, outcome string) error {
	if s.usesPostgresAuth() {
		_, err := s.pg.ExecContext(ctx, `
UPDATE payment_webhook_events SET outcome=$3,processed_at=$4 WHERE provider=$1 AND provider_event_id=$2`,
			provider, eventID, outcome, time.Now().UTC())
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.financialWebhooks[provider+":"+eventID] = outcome
	return nil
}

func stringIndex(values []string, target string) int {
	for index, value := range values {
		if value == target {
			return index
		}
	}
	return -1
}

type FinancialReconciliation struct {
	ID                   string            `json:"id"`
	Provider             string            `json:"provider"`
	Currency             string            `json:"currency"`
	ProviderBalanceMinor models.MinorUnits `json:"providerBalanceMinor"`
	JournalBalanceMinor  models.MinorUnits `json:"journalBalanceMinor"`
	DifferenceMinor      models.MinorUnits `json:"differenceMinor"`
	Status               string            `json:"status"`
	ImmutableHash        string            `json:"immutableHash"`
	CreatedAt            time.Time         `json:"createdAt"`
}

func (s *Store) ReconcileFinancialTreasury(ctx context.Context, provider, currency string, providerBalance models.MinorUnits) (*FinancialReconciliation, error) {
	var journalBalance models.MinorUnits
	if s.usesPostgresAuth() {
		if err := s.pg.QueryRowContext(ctx, `
SELECT COALESCE(SUM(CASE WHEN direction='credit' THEN amount_minor ELSE -amount_minor END),0)
FROM financial_journal WHERE currency=$1`, normalizeCurrency(currency)).Scan(&journalBalance); err != nil {
			return nil, err
		}
	} else {
		s.mu.RLock()
		for _, entries := range s.financialJournal {
			for _, entry := range entries {
				if entry.Currency != normalizeCurrency(currency) {
					continue
				}
				if entry.Direction == "credit" {
					journalBalance += entry.AmountMinor
				} else {
					journalBalance -= entry.AmountMinor
				}
			}
		}
		s.mu.RUnlock()
	}
	now := time.Now().UTC()
	item := &FinancialReconciliation{
		ID: newUUID(), Provider: provider, Currency: normalizeCurrency(currency),
		ProviderBalanceMinor: providerBalance, JournalBalanceMinor: journalBalance,
		DifferenceMinor: providerBalance - journalBalance, Status: "balanced", CreatedAt: now,
	}
	if item.DifferenceMinor != 0 {
		item.Status = "variance"
	}
	item.ImmutableHash = financialEntryHash("", "treasury", provider, item.Status, providerBalance, item.Currency, journalBalance, "reconciliation", item.ID, now)
	if s.usesPostgresAuth() {
		_, err := s.pg.ExecContext(ctx, `
INSERT INTO treasury_reconciliations(id,currency,provider,provider_balance_minor,journal_balance_minor,difference_minor,status,immutable_hash,created_at)
VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9)`,
			item.ID, item.Currency, item.Provider, item.ProviderBalanceMinor, item.JournalBalanceMinor,
			item.DifferenceMinor, item.Status, item.ImmutableHash, item.CreatedAt)
		if err != nil {
			return nil, err
		}
	}
	return item, nil
}

const financialDepositSelect = `
SELECT id,user_id,amount_minor,currency,method,provider,provider_reference,checkout_url,status,
 idempotency_key,request_hash,metadata,requested_at,updated_at,completed_at FROM financial_deposits`

const financialWithdrawalSelect = `
SELECT id,user_id,amount_minor,fee_minor,currency,method,provider,provider_reference,status,policy_decision,
 policy_reasons,idempotency_key,request_hash,metadata,requested_at,updated_at,completed_at FROM financial_withdrawals`

type scanner interface {
	Scan(...any) error
}

func scanFinancialDeposit(row scanner) (*models.FinancialDeposit, error) {
	var item models.FinancialDeposit
	var providerRef, checkoutURL sql.NullString
	var metadata []byte
	err := row.Scan(&item.ID, &item.UserID, &item.AmountMinor, &item.Currency, &item.Method, &item.Provider,
		&providerRef, &checkoutURL, &item.Status, &item.IdempotencyKey, &item.RequestHash, &metadata,
		&item.RequestedAt, &item.UpdatedAt, &item.CompletedAt)
	if err != nil {
		return nil, err
	}
	item.ProviderReference, item.CheckoutURL = providerRef.String, checkoutURL.String
	_ = json.Unmarshal(metadata, &item.Metadata)
	return &item, nil
}

func scanFinancialWithdrawal(row scanner) (*models.FinancialWithdrawal, error) {
	var item models.FinancialWithdrawal
	var providerRef sql.NullString
	var reasons, metadata []byte
	err := row.Scan(&item.ID, &item.UserID, &item.AmountMinor, &item.FeeMinor, &item.Currency,
		&item.Method, &item.Provider, &providerRef, &item.Status, &item.PolicyDecision, &reasons,
		&item.IdempotencyKey, &item.RequestHash, &metadata, &item.RequestedAt, &item.UpdatedAt, &item.CompletedAt)
	if err != nil {
		return nil, err
	}
	item.ProviderReference = providerRef.String
	_ = json.Unmarshal(reasons, &item.PolicyReasons)
	_ = json.Unmarshal(metadata, &item.Metadata)
	return &item, nil
}

func loadFinancialDepositByKey(ctx context.Context, tx *sql.Tx, userID, key string) (*models.FinancialDeposit, error) {
	return scanFinancialDeposit(tx.QueryRowContext(ctx, financialDepositSelect+` WHERE user_id=$1 AND idempotency_key=$2`, userID, key))
}

func loadFinancialWithdrawalByKey(ctx context.Context, tx *sql.Tx, userID, key string) (*models.FinancialWithdrawal, error) {
	return scanFinancialWithdrawal(tx.QueryRowContext(ctx, financialWithdrawalSelect+` WHERE user_id=$1 AND idempotency_key=$2`, userID, key))
}

func loadFinancialDepositForUpdate(ctx context.Context, tx *sql.Tx, id string) (*models.FinancialDeposit, error) {
	return scanFinancialDeposit(tx.QueryRowContext(ctx, financialDepositSelect+` WHERE id=$1 FOR UPDATE`, id))
}

func loadFinancialWithdrawalForUpdate(ctx context.Context, tx *sql.Tx, id string) (*models.FinancialWithdrawal, error) {
	return scanFinancialWithdrawal(tx.QueryRowContext(ctx, financialWithdrawalSelect+` WHERE id=$1 FOR UPDATE`, id))
}

func insertFinancialTransition(ctx context.Context, tx *sql.Tx, resourceType, resourceID, from, to, actorType, actorID, reason, providerEventID string, metadata map[string]string) error {
	payload, _ := json.Marshal(metadata)
	_, err := tx.ExecContext(ctx, `
INSERT INTO financial_transitions(resource_type,resource_id,from_status,to_status,actor_type,actor_id,reason,provider_event_id,metadata,created_at)
VALUES($1,$2,$3,$4,$5,NULLIF($6,''),NULLIF($7,''),NULLIF($8,''),$9,$10)
ON CONFLICT(resource_type,resource_id,to_status,provider_event_id) DO NOTHING`,
		resourceType, resourceID, from, to, actorType, actorID, reason, providerEventID, payload, time.Now().UTC())
	return err
}

func appendFinancialJournal(ctx context.Context, tx *sql.Tx, userID, account, direction string, amount models.MinorUnits, currency string, balance models.MinorUnits, referenceType, referenceID, description string, metadata map[string]string, now time.Time) error {
	now = now.UTC().Truncate(time.Microsecond)
	var previous string
	err := tx.QueryRowContext(ctx, `SELECT entry_hash FROM financial_journal WHERE user_id=$1 ORDER BY sequence DESC LIMIT 1`, userID).Scan(&previous)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return err
	}
	entryHash := financialEntryHash(previous, userID, account, direction, amount, currency, balance, referenceType, referenceID, now)
	payload, _ := json.Marshal(metadata)
	_, err = tx.ExecContext(ctx, `
INSERT INTO financial_journal(id,user_id,account,direction,amount_minor,currency,balance_after_minor,reference_type,
 reference_id,description,previous_hash,entry_hash,metadata,created_at)
VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14)`,
		newUUID(), userID, account, direction, amount, currency, balance, referenceType, referenceID,
		description, previous, entryHash, payload, now)
	return err
}

func (s *Store) appendFinancialJournalMemory(userID, account, direction string, amount models.MinorUnits, currency string, balance models.MinorUnits, referenceType, referenceID, description string, metadata map[string]string, now time.Time) {
	now = now.UTC().Truncate(time.Microsecond)
	entries := s.financialJournal[userID]
	previous := ""
	if len(entries) > 0 {
		previous = entries[len(entries)-1].EntryHash
	}
	entry := models.FinancialLedgerEntry{
		ID: newUUID(), UserID: userID, Account: account, Direction: direction,
		AmountMinor: amount, Currency: currency, BalanceAfterMinor: balance,
		ReferenceType: referenceType, ReferenceID: referenceID, Description: description,
		Sequence: int64(len(entries) + 1), PreviousHash: previous, Metadata: metadata, CreatedAt: now,
	}
	entry.EntryHash = financialEntryHash(previous, userID, account, direction, amount, currency, balance, referenceType, referenceID, now)
	s.financialJournal[userID] = append(entries, entry)
}

func financialEntryHash(previous, userID, account, direction string, amount models.MinorUnits, currency string, balance models.MinorUnits, referenceType, referenceID string, now time.Time) string {
	raw := fmt.Sprintf("%s|%s|%s|%s|%d|%s|%d|%s|%s|%s", previous, userID, account, direction, amount, currency, balance, referenceType, referenceID, now.UTC().Format(time.RFC3339Nano))
	sum := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(sum[:])
}

type financialUsage struct {
	depositDay, depositMonth, withdrawalDay, withdrawalMonth models.MinorUnits
}

func (s *Store) financialUsagePostgres(ctx context.Context, userID string) (financialUsage, error) {
	var usage financialUsage
	now := time.Now().UTC()
	day := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
	month := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
	err := s.pg.QueryRowContext(ctx, `
SELECT
 COALESCE(SUM(amount_minor) FILTER (WHERE requested_at >= $2 AND status NOT IN ('failed','expired')),0),
 COALESCE(SUM(amount_minor) FILTER (WHERE requested_at >= $3 AND status NOT IN ('failed','expired')),0)
FROM financial_deposits WHERE user_id=$1`, userID, day, month).Scan(&usage.depositDay, &usage.depositMonth)
	if err != nil {
		return usage, err
	}
	err = s.pg.QueryRowContext(ctx, `
SELECT
 COALESCE(SUM(amount_minor) FILTER (WHERE requested_at >= $2 AND status NOT IN ('rejected','failed')),0),
 COALESCE(SUM(amount_minor) FILTER (WHERE requested_at >= $3 AND status NOT IN ('rejected','failed')),0)
FROM financial_withdrawals WHERE user_id=$1`, userID, day, month).Scan(&usage.withdrawalDay, &usage.withdrawalMonth)
	return usage, err
}

func calculateFinancialUsage(deposits []models.FinancialDeposit, withdrawals []models.FinancialWithdrawal, now time.Time) financialUsage {
	var usage financialUsage
	for _, item := range deposits {
		if item.Status == models.DepositStatusFailed || item.Status == models.DepositStatusExpired {
			continue
		}
		if sameUTCMonth(item.RequestedAt, now) {
			usage.depositMonth += item.AmountMinor
		}
		if sameUTCDay(item.RequestedAt, now) {
			usage.depositDay += item.AmountMinor
		}
	}
	for _, item := range withdrawals {
		if item.Status == models.FinancialWithdrawalStatusRejected || item.Status == models.FinancialWithdrawalStatusFailed {
			continue
		}
		if sameUTCMonth(item.RequestedAt, now) {
			usage.withdrawalMonth += item.AmountMinor
		}
		if sameUTCDay(item.RequestedAt, now) {
			usage.withdrawalDay += item.AmountMinor
		}
	}
	return usage
}

func applyFinancialUsage(limits *models.FinancialLimits, usage financialUsage) {
	limits.DepositUsedTodayMinor = usage.depositDay
	limits.DepositUsedMonthMinor = usage.depositMonth
	limits.WithdrawUsedTodayMinor = usage.withdrawalDay
	limits.WithdrawUsedMonthMinor = usage.withdrawalMonth
}

func financialDepositsForUser(source map[string]*models.FinancialDeposit, userID string) []models.FinancialDeposit {
	items := []models.FinancialDeposit{}
	for _, item := range source {
		if item.UserID == userID {
			items = append(items, *item)
		}
	}
	sort.Slice(items, func(i, j int) bool { return items[i].RequestedAt.After(items[j].RequestedAt) })
	return items
}

func financialWithdrawalsForUser(source map[string]*models.FinancialWithdrawal, userID string) []models.FinancialWithdrawal {
	items := []models.FinancialWithdrawal{}
	for _, item := range source {
		if item.UserID == userID {
			items = append(items, *item)
		}
	}
	sort.Slice(items, func(i, j int) bool { return items[i].RequestedAt.After(items[j].RequestedAt) })
	return items
}

func sameUTCDay(a, b time.Time) bool {
	a, b = a.UTC(), b.UTC()
	return a.Year() == b.Year() && a.YearDay() == b.YearDay()
}

func sameUTCMonth(a, b time.Time) bool {
	a, b = a.UTC(), b.UTC()
	return a.Year() == b.Year() && a.Month() == b.Month()
}

func normalizeCurrency(currency string) string {
	currency = strings.ToUpper(strings.TrimSpace(currency))
	if len(currency) != 3 {
		return "ZAR"
	}
	return currency
}
