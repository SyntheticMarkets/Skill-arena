package db

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/lib/pq"

	"skill-arena/internal/id"
	"skill-arena/internal/models"
	"skill-arena/migrations"
)

func (s *Store) initPostgresAdminCRM(ctx context.Context) error {
	return s.applyFinancialMigration(ctx, "006_admin_crm", migrations.AdminCRM)
}

func (s *Store) SearchCRMUsers(ctx context.Context, query, status string, limit, offset int) ([]*models.User, int, error) {
	query = strings.TrimSpace(query)
	status = strings.TrimSpace(status)
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	if offset < 0 {
		offset = 0
	}
	if s.pg == nil {
		users, err := s.ListUsers(ctx)
		if err != nil {
			return nil, 0, err
		}
		filtered := make([]*models.User, 0, len(users))
		for _, user := range users {
			if status != "" && user.Status != status {
				continue
			}
			needle := strings.ToLower(query)
			if needle != "" && !strings.Contains(strings.ToLower(user.Email+" "+user.Username+" "+user.DisplayName+" "+user.ID), needle) {
				continue
			}
			copyUser := *user
			filtered = append(filtered, &copyUser)
		}
		total := len(filtered)
		if offset >= total {
			return []*models.User{}, total, nil
		}
		end := offset + limit
		if end > total {
			end = total
		}
		return filtered[offset:end], total, nil
	}
	where := `WHERE ($1='' OR email ILIKE '%'||$1||'%' OR username ILIKE '%'||$1||'%' OR display_name ILIKE '%'||$1||'%' OR id ILIKE '%'||$1||'%')
		AND ($2='' OR status=$2)`
	var total int
	if err := s.pg.QueryRowContext(ctx, `SELECT COUNT(*) FROM users `+where, query, status).Scan(&total); err != nil {
		return nil, 0, err
	}
	rows, err := s.pg.QueryContext(ctx, `SELECT id,email,country,date_of_birth,terms_accepted_at,username,display_name,
		hidden_from_public,role,email_verified,kyc_status,status,created_at,updated_at
		FROM users `+where+` ORDER BY created_at DESC LIMIT $3 OFFSET $4`, query, status, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	users := []*models.User{}
	for rows.Next() {
		user := &models.User{}
		if err := rows.Scan(&user.ID, &user.Email, &user.Country, &user.DateOfBirth, &user.TermsAcceptedAt,
			&user.Username, &user.DisplayName, &user.HiddenFromPublic, &user.Role, &user.EmailVerified,
			&user.KYCStatus, &user.Status, &user.CreatedAt, &user.UpdatedAt); err != nil {
			return nil, 0, err
		}
		users = append(users, user)
	}
	return users, total, rows.Err()
}

func (s *Store) BuildCRMDashboard(ctx context.Context) (*models.CRMDashboard, error) {
	now := time.Now().UTC()
	dashboard := &models.CRMDashboard{GeneratedAt: now}
	if s.pg == nil {
		s.mu.RLock()
		defer s.mu.RUnlock()
		dashboard.Financial.Currency = "ZAR"
		if policy, ok := s.settings.Financial.Policy(s.settings.Financial.DefaultCountry); ok && policy.Currency != "" {
			dashboard.Financial.Currency = policy.Currency
		}
		dashboard.Financial.ActivePaymentProviders = len(s.settings.Payments.EnabledProviders())
		dashboard.Players.TotalUsers = len(s.users)
		for _, user := range s.users {
			if user.CreatedAt.After(now.Add(-24 * time.Hour)) {
				dashboard.Players.NewRegistrations++
			}
			if !user.EmailVerified {
				dashboard.Players.PendingVerify++
			}
		}
		dashboard.Players.OnlineUsers = s.activeAuthCountLocked()
		for _, item := range s.financialDeposits {
			if item.Status == models.DepositStatusCompleted && item.CompletedAt != nil && item.CompletedAt.After(dayStart(now)) {
				dashboard.Financial.DepositsTodayMinor += item.AmountMinor
			}
		}
		for _, item := range s.financialWithdrawals {
			if item.Status == models.FinancialWithdrawalStatusPendingReview {
				dashboard.Financial.PendingWithdrawals++
			}
			if item.Status == models.FinancialWithdrawalStatusCompleted {
				dashboard.Financial.CompletedWithdrawals++
			}
		}
		dashboard.Games.LiveMatches = s.activeMatchCountLocked()
		for _, tickets := range s.supportTickets {
			for _, ticket := range tickets {
				if ticket.Status != models.TicketStatusClosed {
					dashboard.Support.OpenTickets++
				}
			}
		}
		for _, assessment := range s.financialAssessments {
			if assessment.Status == models.AssessmentStatusSubmitted || assessment.Status == models.AssessmentStatusInReview {
				dashboard.Compliance.PendingKYC++
			}
		}
		dashboard.System = models.CRMSystemSummary{API: "ok", Database: "development", Redis: "ok", Storage: "ok", Queue: "ok"}
		return dashboard, nil
	}
	queries := []struct {
		query string
		args  []any
		scan  []any
	}{
		{`SELECT COUNT(*),COUNT(*) FILTER (WHERE created_at >= $1),COUNT(*) FILTER (WHERE email_verified=FALSE) FROM users`, []any{now.Add(-24 * time.Hour)}, []any{&dashboard.Players.TotalUsers, &dashboard.Players.NewRegistrations, &dashboard.Players.PendingVerify}},
		{`SELECT COUNT(DISTINCT user_id) FROM auth_sessions WHERE revoked_at IS NULL AND expires_at > $1`, []any{now}, []any{&dashboard.Players.OnlineUsers}},
		{`SELECT COALESCE(SUM(amount_minor) FILTER (WHERE status='completed' AND completed_at >= $1),0),
			COUNT(*) FILTER (WHERE status='pending_review'),COUNT(*) FILTER (WHERE status='completed') FROM financial_withdrawals`, []any{dayStart(now)}, []any{new(models.MinorUnits), &dashboard.Financial.PendingWithdrawals, &dashboard.Financial.CompletedWithdrawals}},
	}
	var ignoredWithdrawalVolume models.MinorUnits
	queries[2].scan[0] = &ignoredWithdrawalVolume
	for _, item := range queries {
		if err := s.pg.QueryRowContext(ctx, item.query, item.args...).Scan(item.scan...); err != nil {
			return nil, err
		}
	}
	if err := s.pg.QueryRowContext(ctx, `SELECT COALESCE(SUM(amount_minor),0) FROM financial_deposits WHERE status='completed' AND completed_at >= $1`, dayStart(now)).Scan(&dashboard.Financial.DepositsTodayMinor); err != nil {
		return nil, err
	}
	if err := s.pg.QueryRowContext(ctx, `SELECT COALESCE(SUM(balance_minor),0),COALESCE(MIN(currency),'ZAR') FROM treasury_accounts`).Scan(&dashboard.Financial.TreasuryAvailableMinor, &dashboard.Financial.Currency); err != nil {
		return nil, err
	}
	dashboard.Financial.ActivePaymentProviders = len(s.settings.Payments.EnabledProviders())
	if err := s.pg.QueryRowContext(ctx, `SELECT COUNT(*) FILTER (WHERE status='active'),COUNT(*) FILTER (WHERE status='completed') FROM pvp_matches`).Scan(&dashboard.Games.LiveMatches, &dashboard.Games.CompletedMatches); err != nil {
		return nil, err
	}
	if err := s.pg.QueryRowContext(ctx, `SELECT COUNT(*) FROM tournaments WHERE status IN ('open','active','registration')`).Scan(&dashboard.Games.ActiveTournaments); err != nil {
		return nil, err
	}
	if stats, err := s.QueueStats(ctx); err == nil {
		dashboard.Games.QueueSize = stats.PendingJobs
	}
	if err := s.pg.QueryRowContext(ctx, `SELECT COUNT(*) FILTER (WHERE status <> 'closed'),COUNT(*) FILTER (WHERE escalated=TRUE) FROM support_tickets`).Scan(&dashboard.Support.OpenTickets, &dashboard.Support.EscalatedTickets); err != nil {
		return nil, err
	}
	_ = s.pg.QueryRowContext(ctx, `SELECT COALESCE(AVG(EXTRACT(EPOCH FROM(first_response_at-created_at))/60),0) FROM support_tickets WHERE first_response_at IS NOT NULL`).Scan(&dashboard.Support.AverageResponseMins)
	if err := s.pg.QueryRowContext(ctx, `SELECT
		COUNT(*) FILTER (WHERE status IN ('submitted','in_review')),
		(SELECT COUNT(*) FROM review_cases WHERE status NOT IN ('APPROVED','REJECTED')),
		COUNT(*) FILTER (WHERE self_excluded_until > $1),
		COUNT(*) FILTER (WHERE cooling_off_until > $1)
		FROM financial_assessments a LEFT JOIN financial_limits l ON l.user_id=a.user_id`, now).Scan(
		&dashboard.Compliance.PendingKYC, &dashboard.Compliance.PendingReviews,
		&dashboard.Compliance.SelfExclusions, &dashboard.Compliance.CoolingOffActive,
	); err != nil {
		return nil, err
	}
	dashboard.System = models.CRMSystemSummary{API: "ok", Database: "ok", Redis: "ok", Storage: "ok", Queue: "ok"}
	if err := s.pg.PingContext(ctx); err != nil {
		dashboard.System.Database = "unavailable"
	}
	if err := s.redis.Health(ctx); err != nil {
		dashboard.System.Redis = "unavailable"
	}
	if err := s.objects.Health(ctx); err != nil {
		dashboard.System.Storage = "unavailable"
	}
	return dashboard, nil
}

func dayStart(value time.Time) time.Time {
	y, m, d := value.Date()
	return time.Date(y, m, d, 0, 0, 0, 0, time.UTC)
}

func (s *Store) GetCRMUserRecord(ctx context.Context, userID string) (*models.CRMUserRecord, error) {
	user, err := s.GetUserByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	record := &models.CRMUserRecord{User: user, Devices: []*models.Device{}, Sessions: []*models.AuthSession{}}
	record.Progression, _ = s.GetProgressionByUserID(ctx, userID)
	record.Wallet, _ = s.GetFinancialWallet(ctx, userID)
	record.Assessment, _ = s.GetFinancialAssessment(ctx, userID)
	record.Limits, _ = s.GetFinancialLimits(ctx, userID)
	record.Devices, _ = s.ListDevices(ctx, userID)
	record.Sessions, _ = s.ListAuthSessions(ctx, userID)
	record.Deposits, _ = s.ListFinancialDeposits(ctx, userID)
	record.Withdrawals, _ = s.ListFinancialWithdrawals(ctx, userID)
	record.MatchHistory, _ = s.GetSessionsByUserID(ctx, userID)
	record.Statement, _ = s.FinancialStatement(ctx, userID, time.Now().UTC().AddDate(0, -1, 0), time.Now().UTC().Add(time.Second))
	record.InternalNotes, _ = s.ListCRMInternalNotes(ctx, userID)
	record.ActiveRestrictions, _ = s.ListCRMRestrictions(ctx, userID, "active")
	return record, nil
}

func (s *Store) AddCRMInternalNote(ctx context.Context, actorID, userID, body string) (*models.CRMInternalNote, error) {
	body = strings.TrimSpace(body)
	if body == "" || len(body) > 4000 {
		return nil, errors.New("note must be between 1 and 4000 characters")
	}
	note := &models.CRMInternalNote{ID: id.New("note"), UserID: userID, AuthorID: actorID, Body: body, CreatedAt: time.Now().UTC()}
	if s.pg != nil {
		_, err := s.pg.ExecContext(ctx, `INSERT INTO crm_internal_notes(id,user_id,author_id,body,created_at) VALUES($1,$2,$3,$4,$5)`,
			note.ID, note.UserID, note.AuthorID, note.Body, note.CreatedAt)
		if err != nil {
			return nil, err
		}
	} else {
		s.mu.Lock()
		s.crmInternalNotes[userID] = append(s.crmInternalNotes[userID], *note)
		s.mu.Unlock()
	}
	return note, nil
}

func (s *Store) ListCRMInternalNotes(ctx context.Context, userID string) ([]models.CRMInternalNote, error) {
	if s.pg == nil {
		s.mu.RLock()
		result := append([]models.CRMInternalNote(nil), s.crmInternalNotes[userID]...)
		s.mu.RUnlock()
		sort.SliceStable(result, func(i, j int) bool { return result[i].CreatedAt.After(result[j].CreatedAt) })
		return result, nil
	}
	rows, err := s.pg.QueryContext(ctx, `SELECT id,user_id,author_id,body,created_at FROM crm_internal_notes WHERE user_id=$1 ORDER BY created_at DESC`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := []models.CRMInternalNote{}
	for rows.Next() {
		var item models.CRMInternalNote
		if err := rows.Scan(&item.ID, &item.UserID, &item.AuthorID, &item.Body, &item.CreatedAt); err != nil {
			return nil, err
		}
		result = append(result, item)
	}
	return result, rows.Err()
}

func (s *Store) SetCRMUserStatus(ctx context.Context, actorID, userID, status, reason, ipAddress, device string) (*models.User, error) {
	if status != "active" && status != "suspended" && status != "disabled" {
		return nil, errors.New("unsupported account status")
	}
	if strings.TrimSpace(reason) == "" {
		return nil, errors.New("reason is required")
	}
	previous, err := s.GetUserByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	if previous.Role == models.RoleSuperAdmin && actorID != userID {
		actor, actorErr := s.GetUserByID(ctx, actorID)
		if actorErr != nil || actor.Role != models.RoleSuperAdmin {
			return nil, errors.New("only a super administrator may change a super administrator")
		}
	}
	now := time.Now().UTC()
	if s.pg != nil {
		tx, err := s.pg.BeginTx(ctx, nil)
		if err != nil {
			return nil, err
		}
		defer tx.Rollback()
		if _, err := tx.ExecContext(ctx, `UPDATE users SET status=$1,updated_at=$2 WHERE id=$3`, status, now, userID); err != nil {
			return nil, err
		}
		if status != "active" {
			if _, err := tx.ExecContext(ctx, `UPDATE auth_sessions SET revoked_at=$1 WHERE user_id=$2 AND revoked_at IS NULL`, now, userID); err != nil {
				return nil, err
			}
		}
		if err := tx.Commit(); err != nil {
			return nil, err
		}
	} else {
		s.mu.Lock()
		target := s.users[userID]
		if target == nil {
			s.mu.Unlock()
			return nil, errors.New("user not found")
		}
		target.Status, target.UpdatedAt = status, now
		if status != "active" {
			for _, session := range s.auth {
				if session.UserID == userID && session.RevokedAt == nil {
					session.RevokedAt = &now
				}
			}
		}
		s.mu.Unlock()
	}
	if err := s.AppendAuditLog(ctx, actorID, "admin.user.status.changed", userID, ipAddress, map[string]string{
		"previous": previous.Status, "new": status, "reason": reason, "device": device,
	}); err != nil {
		return nil, err
	}
	return s.GetUserByID(ctx, userID)
}

func (s *Store) ForceLogoutCRMUser(ctx context.Context, actorID, userID, reason, ipAddress, device string) error {
	if strings.TrimSpace(reason) == "" {
		return errors.New("reason is required")
	}
	now := time.Now().UTC()
	if s.pg != nil {
		if _, err := s.pg.ExecContext(ctx, `UPDATE auth_sessions SET revoked_at=$1 WHERE user_id=$2 AND revoked_at IS NULL`, now, userID); err != nil {
			return err
		}
	} else {
		s.mu.Lock()
		for _, session := range s.auth {
			if session.UserID == userID && session.RevokedAt == nil {
				session.RevokedAt = &now
			}
		}
		s.mu.Unlock()
	}
	return s.AppendAuditLog(ctx, actorID, "admin.user.sessions.revoked", userID, ipAddress, map[string]string{"reason": reason, "device": device})
}

func (s *Store) CreateCRMRestriction(ctx context.Context, item models.CRMRestriction) (*models.CRMRestriction, error) {
	item.Type = strings.ToLower(strings.TrimSpace(item.Type))
	item.Reason = strings.TrimSpace(item.Reason)
	allowedTypes := map[string]bool{
		"account": true, "deposit": true, "withdrawal": true, "competition": true,
		"communication": true, "cooling_off": true, "self_exclusion": true,
	}
	if !allowedTypes[item.Type] {
		return nil, errors.New("unsupported restriction type")
	}
	if (item.Type == "cooling_off" || item.Type == "self_exclusion") && item.ExpiresAt == nil {
		return nil, errors.New("responsible gaming restrictions require an expiry")
	}
	if len(item.Reason) < 4 || len(item.Reason) > 1000 {
		return nil, errors.New("restriction reason must be between 4 and 1000 characters")
	}
	now := time.Now().UTC()
	item.ID, item.Status, item.CreatedAt, item.UpdatedAt = id.New("restriction"), "active", now, now
	if s.pg != nil {
		_, err := s.pg.ExecContext(ctx, `INSERT INTO crm_restrictions(id,user_id,restriction_type,reason,status,expires_at,created_by,created_at,updated_at)
			VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9)`, item.ID, item.UserID, item.Type, item.Reason, item.Status, item.ExpiresAt, item.CreatedBy, item.CreatedAt, item.UpdatedAt)
		if err != nil {
			return nil, err
		}
	} else {
		s.mu.Lock()
		s.crmRestrictions[item.UserID] = append(s.crmRestrictions[item.UserID], item)
		s.mu.Unlock()
	}
	return &item, nil
}

func (s *Store) ListCRMRestrictions(ctx context.Context, userID, status string) ([]models.CRMRestriction, error) {
	if s.pg == nil {
		s.mu.RLock()
		source := append([]models.CRMRestriction(nil), s.crmRestrictions[userID]...)
		s.mu.RUnlock()
		result := []models.CRMRestriction{}
		for _, item := range source {
			if status == "" || item.Status == status {
				result = append(result, item)
			}
		}
		return result, nil
	}
	rows, err := s.pg.QueryContext(ctx, `SELECT id,user_id,restriction_type,reason,status,expires_at,created_by,created_at,updated_at
		FROM crm_restrictions WHERE ($1='' OR user_id=$1) AND ($2='' OR status=$2) ORDER BY updated_at DESC`, userID, status)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := []models.CRMRestriction{}
	for rows.Next() {
		var item models.CRMRestriction
		if err := rows.Scan(&item.ID, &item.UserID, &item.Type, &item.Reason, &item.Status, &item.ExpiresAt, &item.CreatedBy, &item.CreatedAt, &item.UpdatedAt); err != nil {
			return nil, err
		}
		result = append(result, item)
	}
	return result, rows.Err()
}

func (s *Store) HasActiveCRMRestriction(ctx context.Context, userID string, restrictionTypes ...string) (bool, error) {
	allowed := make(map[string]bool, len(restrictionTypes))
	for _, restrictionType := range restrictionTypes {
		allowed[strings.ToLower(strings.TrimSpace(restrictionType))] = true
	}
	items, err := s.ListCRMRestrictions(ctx, userID, "active")
	if err != nil {
		return false, err
	}
	now := time.Now().UTC()
	for _, item := range items {
		if item.ExpiresAt != nil && !item.ExpiresAt.After(now) {
			if err := s.expireCRMRestriction(ctx, item.ID, now); err != nil {
				return false, err
			}
			continue
		}
		if allowed[item.Type] {
			return true, nil
		}
	}
	return false, nil
}

func (s *Store) UpdateCRMRestrictionStatus(ctx context.Context, userID, restrictionID, status string) (*models.CRMRestriction, error) {
	status = strings.ToLower(strings.TrimSpace(status))
	if status != "lifted" {
		return nil, errors.New("only active restrictions may be lifted")
	}
	now := time.Now().UTC()
	if s.pg != nil {
		var item models.CRMRestriction
		err := s.pg.QueryRowContext(ctx, `UPDATE crm_restrictions SET status=$1,updated_at=$2
			WHERE id=$3 AND user_id=$4 AND status='active'
			RETURNING id,user_id,restriction_type,reason,status,expires_at,created_by,created_at,updated_at`,
			status, now, restrictionID, userID).Scan(&item.ID, &item.UserID, &item.Type, &item.Reason, &item.Status,
			&item.ExpiresAt, &item.CreatedBy, &item.CreatedAt, &item.UpdatedAt)
		if err != nil {
			return nil, err
		}
		return &item, nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	for storedUserID, items := range s.crmRestrictions {
		if storedUserID != userID {
			continue
		}
		for index := range items {
			if items[index].ID == restrictionID && items[index].Status == "active" {
				items[index].Status = status
				items[index].UpdatedAt = now
				s.crmRestrictions[storedUserID] = items
				copyItem := items[index]
				return &copyItem, nil
			}
		}
	}
	return nil, errors.New("active restriction not found")
}

func (s *Store) expireCRMRestriction(ctx context.Context, restrictionID string, now time.Time) error {
	if s.pg != nil {
		_, err := s.pg.ExecContext(ctx, `UPDATE crm_restrictions SET status='expired',updated_at=$1
			WHERE id=$2 AND status='active' AND expires_at IS NOT NULL AND expires_at<=$1`, now, restrictionID)
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	for userID, items := range s.crmRestrictions {
		for index := range items {
			if items[index].ID == restrictionID && items[index].Status == "active" &&
				items[index].ExpiresAt != nil && !items[index].ExpiresAt.After(now) {
				items[index].Status = "expired"
				items[index].UpdatedAt = now
				s.crmRestrictions[userID] = items
				return nil
			}
		}
	}
	return nil
}

func (s *Store) CRMFinanceWorkspace(ctx context.Context, status string) (*models.CRMFinanceWorkspace, error) {
	result := &models.CRMFinanceWorkspace{
		Deposits: []models.FinancialDeposit{}, Withdrawals: []models.FinancialWithdrawal{},
		Reconciliations: []models.CRMReconciliation{}, ReserveChecks: []models.TreasuryReserveCheck{},
		Providers: []models.CRMProviderStatus{},
	}
	if s.pg == nil {
		s.mu.RLock()
		for _, item := range s.financialDeposits {
			if status == "" || item.Status == status {
				result.Deposits = append(result.Deposits, *item)
			}
		}
		for _, item := range s.financialWithdrawals {
			if status == "" || item.Status == status {
				result.Withdrawals = append(result.Withdrawals, *item)
			}
		}
		for _, item := range s.treasuryChecks {
			result.ReserveChecks = append(result.ReserveChecks, *item)
		}
		s.mu.RUnlock()
		return result, nil
	}
	depositRows, err := s.pg.QueryContext(ctx, `SELECT id,user_id,amount_minor,currency,method,provider,COALESCE(provider_reference,''),COALESCE(checkout_url,''),status,idempotency_key,request_hash,metadata,requested_at,updated_at,completed_at
		FROM financial_deposits WHERE ($1='' OR status=$1) ORDER BY requested_at DESC LIMIT 200`, status)
	if err != nil {
		return nil, err
	}
	for depositRows.Next() {
		var item models.FinancialDeposit
		if err := scanCRMFinancialDeposit(depositRows, &item); err != nil {
			depositRows.Close()
			return nil, err
		}
		result.Deposits = append(result.Deposits, item)
	}
	depositRows.Close()
	withdrawalRows, err := s.pg.QueryContext(ctx, `SELECT id,user_id,amount_minor,fee_minor,currency,method,provider,COALESCE(provider_reference,''),status,policy_decision,policy_reasons,idempotency_key,request_hash,metadata,requested_at,updated_at,completed_at
		FROM financial_withdrawals WHERE ($1='' OR status=$1) ORDER BY requested_at DESC LIMIT 200`, status)
	if err != nil {
		return nil, err
	}
	for withdrawalRows.Next() {
		var item models.FinancialWithdrawal
		if err := scanCRMFinancialWithdrawal(withdrawalRows, &item); err != nil {
			withdrawalRows.Close()
			return nil, err
		}
		result.Withdrawals = append(result.Withdrawals, item)
	}
	withdrawalRows.Close()
	rows, err := s.pg.QueryContext(ctx, `SELECT id,currency,provider,provider_balance_minor,journal_balance_minor,difference_minor,status,immutable_hash,created_at FROM treasury_reconciliations ORDER BY created_at DESC LIMIT 100`)
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		var item models.CRMReconciliation
		if err := rows.Scan(&item.ID, &item.Currency, &item.Provider, &item.ProviderBalanceMinor, &item.JournalBalanceMinor, &item.DifferenceMinor, &item.Status, &item.ImmutableHash, &item.CreatedAt); err != nil {
			rows.Close()
			return nil, err
		}
		result.Reconciliations = append(result.Reconciliations, item)
	}
	rows.Close()
	checkRows, err := s.pg.QueryContext(ctx, `SELECT id,provider,currency,provider_available_minor,provider_pending_minor,liability_minor,requested_minor,purpose,passed,immutable_hash,artifact_key,created_at
		FROM treasury_reserve_checks ORDER BY created_at DESC LIMIT 100`)
	if err != nil {
		return nil, err
	}
	for checkRows.Next() {
		var item models.TreasuryReserveCheck
		if err := checkRows.Scan(&item.ID, &item.Provider, &item.Currency, &item.ProviderAvailableMinor, &item.ProviderPendingMinor, &item.LiabilityMinor, &item.RequestedMinor, &item.Purpose, &item.Passed, &item.ImmutableHash, &item.ArtifactKey, &item.CreatedAt); err != nil {
			checkRows.Close()
			return nil, err
		}
		result.ReserveChecks = append(result.ReserveChecks, item)
	}
	checkRows.Close()
	return result, nil
}

type crmRowScanner interface {
	Scan(...any) error
}

func scanCRMFinancialDeposit(row crmRowScanner, item *models.FinancialDeposit) error {
	return row.Scan(&item.ID, &item.UserID, &item.AmountMinor, &item.Currency, &item.Method, &item.Provider,
		&item.ProviderReference, &item.CheckoutURL, &item.Status, &item.IdempotencyKey, &item.RequestHash,
		&item.Metadata, &item.RequestedAt, &item.UpdatedAt, &item.CompletedAt)
}

func scanCRMFinancialWithdrawal(row crmRowScanner, item *models.FinancialWithdrawal) error {
	return row.Scan(&item.ID, &item.UserID, &item.AmountMinor, &item.FeeMinor, &item.Currency, &item.Method,
		&item.Provider, &item.ProviderReference, &item.Status, &item.PolicyDecision, &item.PolicyReasons,
		&item.IdempotencyKey, &item.RequestHash, &item.Metadata, &item.RequestedAt, &item.UpdatedAt, &item.CompletedAt)
}

func (s *Store) ListCRMComplianceCases(ctx context.Context, status string) ([]models.CRMComplianceCase, error) {
	users, _, err := s.SearchCRMUsers(ctx, "", "", 100, 0)
	if err != nil {
		return nil, err
	}
	result := []models.CRMComplianceCase{}
	reviews, _ := s.ListReviewCases(ctx)
	for _, user := range users {
		assessment, assessmentErr := s.GetFinancialAssessment(ctx, user.ID)
		if assessmentErr != nil || (status != "" && assessment.Status != status) {
			continue
		}
		evidence, _ := s.ListFinancialEvidence(ctx, user.ID)
		providerResponses, _ := s.ListCRMComplianceProviderResponses(ctx, user.ID)
		userReviews := []*models.ReviewCase{}
		for _, review := range reviews {
			if review.UserID == user.ID {
				userReviews = append(userReviews, review)
			}
		}
		if status == "" || assessment.Status == status || len(evidence) > 0 || len(userReviews) > 0 {
			result = append(result, models.CRMComplianceCase{
				User: user, Assessment: assessment, Evidence: evidence,
				ProviderResponses: providerResponses, Reviews: userReviews,
			})
		}
	}
	return result, nil
}

func (s *Store) RecordCRMComplianceProviderResponse(ctx context.Context, item models.CRMComplianceProviderResponse) (*models.CRMComplianceProviderResponse, error) {
	item.Provider = strings.ToLower(strings.TrimSpace(item.Provider))
	item.ProviderReference = strings.TrimSpace(item.ProviderReference)
	item.CheckType = strings.ToLower(strings.TrimSpace(item.CheckType))
	item.Status = strings.ToLower(strings.TrimSpace(item.Status))
	allowedChecks := map[string]bool{
		"identity": true, "address": true, "age": true, "sanctions": true,
		"pep": true, "aml": true, "source_of_funds": true,
	}
	allowedStatuses := map[string]bool{"pending": true, "clear": true, "review": true, "rejected": true, "error": true}
	if item.UserID == "" || len(item.Provider) < 2 || item.ProviderReference == "" || !allowedChecks[item.CheckType] || !allowedStatuses[item.Status] {
		return nil, errors.New("compliance provider response is invalid")
	}
	if item.Metadata == nil {
		item.Metadata = map[string]string{}
	}
	if item.RiskSignals == nil {
		item.RiskSignals = []string{}
	}
	metadata, err := json.Marshal(item.Metadata)
	if err != nil {
		return nil, err
	}
	item.ID = id.New("provider_response")
	item.ReceivedAt = time.Now().UTC()
	if s.pg != nil {
		_, err := s.pg.ExecContext(ctx, `INSERT INTO crm_compliance_provider_responses(
			id,user_id,provider,provider_reference,check_type,status,risk_signals,metadata,received_at
		) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9)
		ON CONFLICT(provider,provider_reference,check_type) DO UPDATE SET
			id=EXCLUDED.id,status=EXCLUDED.status,risk_signals=EXCLUDED.risk_signals,
			metadata=EXCLUDED.metadata,received_at=EXCLUDED.received_at`,
			item.ID, item.UserID, item.Provider, item.ProviderReference, item.CheckType,
			item.Status, pq.Array(item.RiskSignals), metadata, item.ReceivedAt)
		if err != nil {
			return nil, err
		}
	} else {
		s.mu.Lock()
		s.crmProviderResponses[item.UserID] = append(s.crmProviderResponses[item.UserID], item)
		s.mu.Unlock()
	}
	return &item, nil
}

func (s *Store) ListCRMComplianceProviderResponses(ctx context.Context, userID string) ([]models.CRMComplianceProviderResponse, error) {
	if s.pg == nil {
		s.mu.RLock()
		defer s.mu.RUnlock()
		return append([]models.CRMComplianceProviderResponse{}, s.crmProviderResponses[userID]...), nil
	}
	rows, err := s.pg.QueryContext(ctx, `SELECT id,user_id,provider,provider_reference,check_type,status,risk_signals,metadata,received_at
		FROM crm_compliance_provider_responses WHERE user_id=$1 ORDER BY received_at DESC`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := []models.CRMComplianceProviderResponse{}
	for rows.Next() {
		var item models.CRMComplianceProviderResponse
		var metadata []byte
		if err := rows.Scan(&item.ID, &item.UserID, &item.Provider, &item.ProviderReference, &item.CheckType,
			&item.Status, pq.Array(&item.RiskSignals), &metadata, &item.ReceivedAt); err != nil {
			return nil, err
		}
		if err := json.Unmarshal(metadata, &item.Metadata); err != nil {
			return nil, err
		}
		result = append(result, item)
	}
	return result, rows.Err()
}

func (s *Store) ListCRMSupportTickets(ctx context.Context, status string) ([]models.CRMSupportTicket, error) {
	if s.pg == nil {
		s.mu.RLock()
		defer s.mu.RUnlock()
		result := []models.CRMSupportTicket{}
		for userID, tickets := range s.supportTickets {
			for _, ticket := range tickets {
				if status != "" && ticket.Status != status {
					continue
				}
				result = append(result, models.CRMSupportTicket{
					ID: ticket.ID, UserID: userID, Category: ticket.Category, Subject: ticket.Subject,
					Status: ticket.Status, Priority: "normal", CreatedAt: ticket.CreatedAt, UpdatedAt: ticket.UpdatedAt,
					Messages:    []models.CRMSupportMessage{{ID: ticket.ID + "-initial", TicketID: ticket.ID, AuthorID: userID, Body: ticket.Message, CreatedAt: ticket.CreatedAt}},
					Attachments: []models.SupportAttachment{},
				})
			}
		}
		return result, nil
	}
	rows, err := s.pg.QueryContext(ctx, `SELECT id,user_id,category,subject,message,status,priority,COALESCE(assigned_to,''),escalated,created_at,updated_at,first_response_at
		FROM support_tickets WHERE ($1='' OR status=$1) ORDER BY updated_at DESC LIMIT 200`, status)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := []models.CRMSupportTicket{}
	for rows.Next() {
		var item models.CRMSupportTicket
		var initialBody string
		if err := rows.Scan(&item.ID, &item.UserID, &item.Category, &item.Subject, &initialBody, &item.Status, &item.Priority,
			&item.AssignedTo, &item.Escalated, &item.CreatedAt, &item.UpdatedAt, &item.FirstResponse); err != nil {
			return nil, err
		}
		item.Messages = []models.CRMSupportMessage{{ID: item.ID + "-initial", TicketID: item.ID, AuthorID: item.UserID, Body: initialBody, CreatedAt: item.CreatedAt}}
		messageRows, err := s.pg.QueryContext(ctx, `SELECT id,ticket_id,author_id,body,internal,created_at FROM crm_support_messages WHERE ticket_id=$1 ORDER BY created_at`, item.ID)
		if err != nil {
			return nil, err
		}
		for messageRows.Next() {
			var message models.CRMSupportMessage
			if err := messageRows.Scan(&message.ID, &message.TicketID, &message.AuthorID, &message.Body, &message.Internal, &message.CreatedAt); err != nil {
				messageRows.Close()
				return nil, err
			}
			item.Messages = append(item.Messages, message)
		}
		messageRows.Close()
		result = append(result, item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	rows.Close()
	for index := range result {
		result[index].Attachments, _ = s.ListSupportAttachments(ctx, result[index].ID)
	}
	return result, nil
}

func (s *Store) UpdateCRMSupportTicket(ctx context.Context, actorID, ticketID, status, priority, assignedTo string, escalated bool, reply string, internal bool) (*models.CRMSupportTicket, error) {
	allowedStatus := map[string]bool{"open": true, "received": true, "in_progress": true, "waiting_player": true, "escalated": true, "closed": true}
	allowedPriority := map[string]bool{"low": true, "normal": true, "high": true, "urgent": true}
	if !allowedStatus[status] || !allowedPriority[priority] {
		return nil, errors.New("invalid support status or priority")
	}
	now := time.Now().UTC()
	if s.pg == nil {
		return nil, errors.New("support operations require PostgreSQL")
	}
	tx, err := s.pg.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	if _, err := tx.ExecContext(ctx, `UPDATE support_tickets SET status=$1,priority=$2,assigned_to=NULLIF($3,''),escalated=$4,
		first_response_at=CASE WHEN first_response_at IS NULL AND $5<>'' THEN $6 ELSE first_response_at END,updated_at=$6 WHERE id=$7`,
		status, priority, assignedTo, escalated, strings.TrimSpace(reply), now, ticketID); err != nil {
		return nil, err
	}
	if strings.TrimSpace(reply) != "" {
		if len(reply) > 8000 {
			return nil, errors.New("support reply is too long")
		}
		if _, err := tx.ExecContext(ctx, `INSERT INTO crm_support_messages(id,ticket_id,author_id,body,internal,created_at) VALUES($1,$2,$3,$4,$5,$6)`,
			id.New("support-message"), ticketID, actorID, strings.TrimSpace(reply), internal, now); err != nil {
			return nil, err
		}
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	tickets, err := s.ListCRMSupportTickets(ctx, "")
	if err != nil {
		return nil, err
	}
	for index := range tickets {
		if tickets[index].ID == ticketID {
			return &tickets[index], nil
		}
	}
	return nil, errors.New("support ticket not found")
}

func (s *Store) ListCRMJurisdictions(ctx context.Context) ([]models.CRMJurisdictionPolicy, error) {
	if s.pg == nil {
		s.mu.RLock()
		resultByCountry := s.configuredCRMJurisdictions()
		for _, item := range s.crmJurisdictions {
			resultByCountry[item.Country] = *item
		}
		s.mu.RUnlock()
		result := make([]models.CRMJurisdictionPolicy, 0, len(resultByCountry))
		for _, item := range resultByCountry {
			result = append(result, item)
		}
		sort.Slice(result, func(i, j int) bool { return result[i].Country < result[j].Country })
		return result, nil
	}
	rows, err := s.pg.QueryContext(ctx, `SELECT country,currency,minimum_age,deposit_enabled,withdrawal_enabled,source_of_funds_required,
		daily_deposit_minor,monthly_deposit_minor,daily_withdrawal_minor,monthly_withdrawal_minor,updated_by,updated_at
		FROM crm_jurisdiction_policies ORDER BY country`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	resultByCountry := s.configuredCRMJurisdictions()
	for rows.Next() {
		var item models.CRMJurisdictionPolicy
		if err := rows.Scan(&item.Country, &item.Currency, &item.MinimumAge, &item.DepositEnabled, &item.WithdrawalEnabled,
			&item.SourceOfFundsRequired, &item.DailyDepositMinor, &item.MonthlyDepositMinor, &item.DailyWithdrawalMinor,
			&item.MonthlyWithdrawalMinor, &item.UpdatedBy, &item.UpdatedAt); err != nil {
			return nil, err
		}
		resultByCountry[item.Country] = item
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	result := make([]models.CRMJurisdictionPolicy, 0, len(resultByCountry))
	for _, item := range resultByCountry {
		result = append(result, item)
	}
	sort.Slice(result, func(i, j int) bool { return result[i].Country < result[j].Country })
	return result, nil
}

func (s *Store) configuredCRMJurisdictions() map[string]models.CRMJurisdictionPolicy {
	result := make(map[string]models.CRMJurisdictionPolicy, len(s.settings.Financial.Jurisdictions))
	for country, policy := range s.settings.Financial.Jurisdictions {
		result[country] = models.CRMJurisdictionPolicy{
			Country: country, Currency: policy.Currency, MinimumAge: policy.MinimumAge,
			DepositEnabled: true, WithdrawalEnabled: true, SourceOfFundsRequired: policy.SourceOfFundsRequired,
			DailyDepositMinor:      models.MinorUnits(policy.DailyDepositMinor),
			MonthlyDepositMinor:    models.MinorUnits(policy.MonthlyDepositMinor),
			DailyWithdrawalMinor:   models.MinorUnits(policy.DailyWithdrawalMinor),
			MonthlyWithdrawalMinor: models.MinorUnits(policy.MonthlyWithdrawalMinor),
			UpdatedBy:              "deployment configuration",
		}
	}
	return result
}

func (s *Store) GetCRMJurisdiction(ctx context.Context, country string) (*models.CRMJurisdictionPolicy, error) {
	country = strings.ToUpper(strings.TrimSpace(country))
	if s.pg == nil {
		s.mu.RLock()
		item := s.crmJurisdictions[country]
		s.mu.RUnlock()
		if item == nil {
			return nil, nil
		}
		copyItem := *item
		return &copyItem, nil
	}
	var item models.CRMJurisdictionPolicy
	err := s.pg.QueryRowContext(ctx, `SELECT country,currency,minimum_age,deposit_enabled,withdrawal_enabled,source_of_funds_required,
		daily_deposit_minor,monthly_deposit_minor,daily_withdrawal_minor,monthly_withdrawal_minor,updated_by,updated_at
		FROM crm_jurisdiction_policies WHERE country=$1`, country).Scan(
		&item.Country, &item.Currency, &item.MinimumAge, &item.DepositEnabled, &item.WithdrawalEnabled,
		&item.SourceOfFundsRequired, &item.DailyDepositMinor, &item.MonthlyDepositMinor,
		&item.DailyWithdrawalMinor, &item.MonthlyWithdrawalMinor, &item.UpdatedBy, &item.UpdatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &item, nil
}

func (s *Store) SaveCRMJurisdiction(ctx context.Context, policy models.CRMJurisdictionPolicy) (*models.CRMJurisdictionPolicy, error) {
	policy.Country = strings.ToUpper(strings.TrimSpace(policy.Country))
	policy.Currency = strings.ToUpper(strings.TrimSpace(policy.Currency))
	if len(policy.Country) != 2 || len(policy.Currency) != 3 || policy.MinimumAge < 18 ||
		policy.DailyDepositMinor < 0 || policy.MonthlyDepositMinor < policy.DailyDepositMinor ||
		policy.DailyWithdrawalMinor < 0 || policy.MonthlyWithdrawalMinor < policy.DailyWithdrawalMinor {
		return nil, errors.New("jurisdiction policy is invalid")
	}
	policy.UpdatedAt = time.Now().UTC()
	if s.pg != nil {
		_, err := s.pg.ExecContext(ctx, `INSERT INTO crm_jurisdiction_policies(country,currency,minimum_age,deposit_enabled,withdrawal_enabled,source_of_funds_required,
			daily_deposit_minor,monthly_deposit_minor,daily_withdrawal_minor,monthly_withdrawal_minor,updated_by,updated_at)
			VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12)
			ON CONFLICT(country) DO UPDATE SET currency=EXCLUDED.currency,minimum_age=EXCLUDED.minimum_age,deposit_enabled=EXCLUDED.deposit_enabled,
			withdrawal_enabled=EXCLUDED.withdrawal_enabled,source_of_funds_required=EXCLUDED.source_of_funds_required,
			daily_deposit_minor=EXCLUDED.daily_deposit_minor,monthly_deposit_minor=EXCLUDED.monthly_deposit_minor,
			daily_withdrawal_minor=EXCLUDED.daily_withdrawal_minor,monthly_withdrawal_minor=EXCLUDED.monthly_withdrawal_minor,
			updated_by=EXCLUDED.updated_by,updated_at=EXCLUDED.updated_at`,
			policy.Country, policy.Currency, policy.MinimumAge, policy.DepositEnabled, policy.WithdrawalEnabled,
			policy.SourceOfFundsRequired, policy.DailyDepositMinor, policy.MonthlyDepositMinor,
			policy.DailyWithdrawalMinor, policy.MonthlyWithdrawalMinor, policy.UpdatedBy, policy.UpdatedAt)
		if err != nil {
			return nil, err
		}
	} else {
		s.mu.Lock()
		copyPolicy := policy
		s.crmJurisdictions[policy.Country] = &copyPolicy
		s.mu.Unlock()
	}
	return &policy, nil
}

func (s *Store) CreateCRMAnnouncement(ctx context.Context, announcement models.CRMAnnouncement) (*models.CRMAnnouncement, error) {
	announcement.Category = strings.ToLower(strings.TrimSpace(announcement.Category))
	announcement.Title = strings.TrimSpace(announcement.Title)
	announcement.Message = strings.TrimSpace(announcement.Message)
	announcement.Audience = strings.ToLower(strings.TrimSpace(announcement.Audience))
	if len(announcement.Title) < 4 || len(announcement.Title) > 120 || len(announcement.Message) < 10 || len(announcement.Message) > 4000 {
		return nil, errors.New("announcement title or message length is invalid")
	}
	validCategory := map[string]bool{"announcement": true, "maintenance": true, "security": true, "compliance": true}
	validAudience := map[string]bool{"all": true, "verified": true, "restricted": true, "country": true}
	if !validCategory[announcement.Category] || !validAudience[announcement.Audience] {
		return nil, errors.New("announcement category or audience is invalid")
	}
	now := time.Now().UTC()
	announcement.ID, announcement.Status, announcement.CreatedAt, announcement.SentAt = id.New("announcement"), "sent", now, &now
	if s.pg != nil {
		tx, err := s.pg.BeginTx(ctx, nil)
		if err != nil {
			return nil, err
		}
		defer tx.Rollback()
		if _, err := tx.ExecContext(ctx, `INSERT INTO crm_announcements(id,category,title,message,audience,status,created_by,created_at,sent_at) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9)`,
			announcement.ID, announcement.Category, announcement.Title, announcement.Message, announcement.Audience, announcement.Status, announcement.CreatedBy, announcement.CreatedAt, announcement.SentAt); err != nil {
			return nil, err
		}
		where := ""
		switch announcement.Audience {
		case "verified":
			where = " WHERE email_verified=TRUE"
		case "restricted":
			where = " WHERE status<>'active'"
		case "country":
			where = " WHERE country<>''"
		}
		rows, err := tx.QueryContext(ctx, `SELECT id FROM users`+where)
		if err != nil {
			return nil, err
		}
		for rows.Next() {
			var userID string
			if err := rows.Scan(&userID); err != nil {
				rows.Close()
				return nil, err
			}
			notificationID := id.New("notification")
			if _, err := tx.ExecContext(ctx, `INSERT INTO player_notifications(id,user_id,category,title,message,status,action_url,metadata,created_at)
				VALUES($1,$2,$3,$4,$5,'unread','/notifications',$6,$7)`,
				notificationID, userID, announcement.Category, announcement.Title, announcement.Message,
				fmt.Sprintf(`{"announcementId":%q}`, announcement.ID), now); err != nil {
				rows.Close()
				return nil, err
			}
		}
		rows.Close()
		if err := tx.Commit(); err != nil {
			return nil, err
		}
	} else {
		s.mu.Lock()
		copyAnnouncement := announcement
		s.crmAnnouncements[announcement.ID] = &copyAnnouncement
		s.mu.Unlock()
		users, _ := s.ListUsers(ctx)
		for _, user := range users {
			_ = s.CreateNotification(ctx, &models.Notification{UserID: user.ID, Category: announcement.Category, Title: announcement.Title, Message: announcement.Message, ActionURL: "/notifications"})
		}
	}
	return &announcement, nil
}

func (s *Store) ListCRMAnnouncements(ctx context.Context) ([]models.CRMAnnouncement, error) {
	if s.pg == nil {
		s.mu.RLock()
		result := make([]models.CRMAnnouncement, 0, len(s.crmAnnouncements))
		for _, item := range s.crmAnnouncements {
			result = append(result, *item)
		}
		s.mu.RUnlock()
		return result, nil
	}
	rows, err := s.pg.QueryContext(ctx, `SELECT id,category,title,message,audience,status,created_by,created_at,sent_at FROM crm_announcements ORDER BY created_at DESC LIMIT 100`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := []models.CRMAnnouncement{}
	for rows.Next() {
		var item models.CRMAnnouncement
		if err := rows.Scan(&item.ID, &item.Category, &item.Title, &item.Message, &item.Audience, &item.Status, &item.CreatedBy, &item.CreatedAt, &item.SentAt); err != nil {
			return nil, err
		}
		result = append(result, item)
	}
	return result, rows.Err()
}

func (s *Store) VerifyCRMAuditChain(ctx context.Context) error {
	logs, err := s.GetAuditLogs(ctx, 200)
	if err != nil {
		return err
	}
	sort.Slice(logs, func(i, j int) bool {
		if logs[i].CreatedAt.Equal(logs[j].CreatedAt) {
			return logs[i].ID < logs[j].ID
		}
		return logs[i].CreatedAt.Before(logs[j].CreatedAt)
	})
	previous := ""
	if len(logs) > 0 {
		previous = logs[0].PreviousHash
	}
	for _, item := range logs {
		if item.EntryHash == "" || item.PreviousHash != previous {
			return errors.New("audit record is missing immutable hash")
		}
		expected := auditHash(item.PreviousHash, item.ID, item.ActorID, item.Action, item.TargetID, item.IPAddress, item.Metadata, item.CreatedAt)
		if item.EntryHash != expected {
			return errors.New("audit record hash does not match its contents")
		}
		previous = item.EntryHash
	}
	return nil
}
