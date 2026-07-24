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
)

func (s *Store) StoreFinancialEvidence(ctx context.Context, userID, evidenceType, contentType string, data []byte) (*models.FinancialEvidence, error) {
	evidenceType = strings.ToLower(strings.TrimSpace(evidenceType))
	if userID == "" || !validEvidenceType(evidenceType) || !validEvidenceContentType(contentType) {
		return nil, errors.New("valid evidence owner, type, and content type are required")
	}
	if len(data) == 0 || len(data) > 10<<20 {
		return nil, errors.New("evidence must be between 1 byte and 10 MiB")
	}
	now := time.Now().UTC()
	id := newUUID()
	key := fmt.Sprintf("financial/evidence/%s/%s", userID, id)
	digest := sha256.Sum256(data)
	item := &models.FinancialEvidence{
		ID: id, UserID: userID, Type: evidenceType, ObjectKey: key, ContentType: contentType,
		SizeBytes: int64(len(data)), SHA256: hex.EncodeToString(digest[:]), Status: "received", CreatedAt: now,
	}
	if err := s.objects.Put(ctx, key, data, contentType); err != nil {
		return nil, fmt.Errorf("store financial evidence: %w", err)
	}
	if s.usesPostgresAuth() {
		_, err := s.pg.ExecContext(ctx, `
INSERT INTO financial_evidence(id,user_id,evidence_type,object_key,content_type,size_bytes,sha256,status,created_at)
VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9)`,
			item.ID, item.UserID, item.Type, item.ObjectKey, item.ContentType, item.SizeBytes, item.SHA256, item.Status, item.CreatedAt)
		if err != nil {
			_ = s.objects.Delete(ctx, key)
			return nil, err
		}
		return item, nil
	}
	s.mu.Lock()
	s.financialEvidence[id] = item
	s.mu.Unlock()
	copyItem := *item
	return &copyItem, nil
}

func (s *Store) ListFinancialEvidence(ctx context.Context, userID string) ([]models.FinancialEvidence, error) {
	if s.usesPostgresAuth() {
		rows, err := s.pg.QueryContext(ctx, `
SELECT id,user_id,evidence_type,object_key,content_type,size_bytes,sha256,status,created_at
FROM financial_evidence WHERE user_id=$1 ORDER BY created_at DESC`, userID)
		if err != nil {
			return nil, err
		}
		defer rows.Close()
		items := []models.FinancialEvidence{}
		for rows.Next() {
			var item models.FinancialEvidence
			if err := rows.Scan(&item.ID, &item.UserID, &item.Type, &item.ObjectKey, &item.ContentType, &item.SizeBytes, &item.SHA256, &item.Status, &item.CreatedAt); err != nil {
				return nil, err
			}
			items = append(items, item)
		}
		return items, rows.Err()
	}
	s.mu.RLock()
	items := []models.FinancialEvidence{}
	for _, item := range s.financialEvidence {
		if item.UserID == userID {
			items = append(items, *item)
		}
	}
	s.mu.RUnlock()
	sort.Slice(items, func(i, j int) bool { return items[i].CreatedAt.After(items[j].CreatedAt) })
	return items, nil
}

func (s *Store) StoreFinancialArtifact(ctx context.Context, userID, artifactType, contentType string, data []byte) (*models.FinancialArtifact, error) {
	artifactType = strings.ToLower(strings.TrimSpace(artifactType))
	if !validArtifactType(artifactType) || len(data) == 0 {
		return nil, errors.New("valid financial artifact is required")
	}
	now := time.Now().UTC()
	id := newUUID()
	ownerPath := userID
	if ownerPath == "" {
		ownerPath = "treasury"
	}
	key := fmt.Sprintf("financial/artifacts/%s/%s", ownerPath, id)
	digest := sha256.Sum256(data)
	item := &models.FinancialArtifact{
		ID: id, UserID: userID, Type: artifactType, ObjectKey: key, ContentType: contentType,
		SizeBytes: int64(len(data)), SHA256: hex.EncodeToString(digest[:]), CreatedAt: now,
	}
	if err := s.objects.Put(ctx, key, data, contentType); err != nil {
		return nil, fmt.Errorf("store financial artifact: %w", err)
	}
	if s.usesPostgresAuth() {
		var owner any
		if userID != "" {
			owner = userID
		}
		_, err := s.pg.ExecContext(ctx, `
INSERT INTO financial_artifacts(id,user_id,artifact_type,object_key,content_type,size_bytes,sha256,created_at)
VALUES($1,$2,$3,$4,$5,$6,$7,$8)`,
			item.ID, owner, item.Type, item.ObjectKey, item.ContentType, item.SizeBytes, item.SHA256, item.CreatedAt)
		if err != nil {
			_ = s.objects.Delete(ctx, key)
			return nil, err
		}
		return item, nil
	}
	s.mu.Lock()
	s.financialArtifacts[id] = item
	s.mu.Unlock()
	copyItem := *item
	return &copyItem, nil
}

func (s *Store) GetFinancialArtifact(ctx context.Context, userID, artifactID string) (*models.FinancialArtifact, []byte, error) {
	var item models.FinancialArtifact
	if s.usesPostgresAuth() {
		var owner sql.NullString
		err := s.pg.QueryRowContext(ctx, `
SELECT id,user_id,artifact_type,object_key,content_type,size_bytes,sha256,created_at
FROM financial_artifacts WHERE id=$1`, artifactID).
			Scan(&item.ID, &owner, &item.Type, &item.ObjectKey, &item.ContentType, &item.SizeBytes, &item.SHA256, &item.CreatedAt)
		if err != nil {
			return nil, nil, err
		}
		item.UserID = owner.String
	} else {
		s.mu.RLock()
		stored := s.financialArtifacts[artifactID]
		if stored != nil {
			item = *stored
		}
		s.mu.RUnlock()
		if item.ID == "" {
			return nil, nil, errors.New("financial artifact not found")
		}
	}
	if item.UserID == "" || item.UserID != userID {
		return nil, nil, errors.New("financial artifact not found")
	}
	data, err := s.objects.Get(ctx, item.ObjectKey)
	if err != nil {
		return nil, nil, err
	}
	digest := sha256.Sum256(data)
	if hex.EncodeToString(digest[:]) != item.SHA256 {
		return nil, nil, errors.New("financial artifact integrity check failed")
	}
	return &item, data, nil
}

func (s *Store) SaveFinancialPayoutDestination(ctx context.Context, item models.FinancialPayoutDestination) (*models.FinancialPayoutDestination, error) {
	item.Provider = strings.ToLower(strings.TrimSpace(item.Provider))
	item.ProviderAccountID = strings.TrimSpace(item.ProviderAccountID)
	if item.UserID == "" || item.Provider == "" || item.ProviderAccountID == "" ||
		(item.Status != "pending" && item.Status != "verified" && item.Status != "disabled") {
		return nil, errors.New("valid payout destination is required")
	}
	item.UpdatedAt = time.Now().UTC()
	if s.usesPostgresAuth() {
		var evidence any
		if item.EvidenceID != "" {
			evidence = item.EvidenceID
		}
		_, err := s.pg.ExecContext(ctx, `
INSERT INTO financial_payout_destinations(user_id,provider,provider_account_id,status,evidence_id,updated_at)
VALUES($1,$2,$3,$4,$5,$6)
ON CONFLICT(user_id,provider) DO UPDATE SET
 provider_account_id=EXCLUDED.provider_account_id,status=EXCLUDED.status,
 evidence_id=EXCLUDED.evidence_id,updated_at=EXCLUDED.updated_at`,
			item.UserID, item.Provider, item.ProviderAccountID, item.Status, evidence, item.UpdatedAt)
		if err != nil {
			return nil, err
		}
		return &item, nil
	}
	s.mu.Lock()
	s.payoutDestinations[item.UserID+":"+item.Provider] = &item
	s.mu.Unlock()
	copyItem := item
	return &copyItem, nil
}

func (s *Store) GetFinancialPayoutDestination(ctx context.Context, userID, provider string) (*models.FinancialPayoutDestination, error) {
	provider = strings.ToLower(strings.TrimSpace(provider))
	if s.usesPostgresAuth() {
		var item models.FinancialPayoutDestination
		var evidence sql.NullString
		err := s.pg.QueryRowContext(ctx, `
SELECT user_id,provider,provider_account_id,status,evidence_id,updated_at
FROM financial_payout_destinations WHERE user_id=$1 AND provider=$2`, userID, provider).
			Scan(&item.UserID, &item.Provider, &item.ProviderAccountID, &item.Status, &evidence, &item.UpdatedAt)
		item.EvidenceID = evidence.String
		return &item, err
	}
	s.mu.RLock()
	item := s.payoutDestinations[userID+":"+provider]
	s.mu.RUnlock()
	if item == nil {
		return nil, errors.New("payout destination not found")
	}
	copyItem := *item
	return &copyItem, nil
}

func (s *Store) FinancialLiability(ctx context.Context, currency string) (models.MinorUnits, error) {
	currency = normalizeCurrency(currency)
	if s.usesPostgresAuth() {
		var amount models.MinorUnits
		err := s.pg.QueryRowContext(ctx, `
SELECT COALESCE(SUM(available_minor+pending_withdrawal_minor+locked_minor),0)
FROM financial_wallets WHERE currency=$1`, currency).Scan(&amount)
		return amount, err
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	var amount models.MinorUnits
	for _, wallet := range s.financialWallets {
		if wallet.Currency == currency {
			amount += wallet.AvailableMinor + wallet.PendingWithdrawalMinor + wallet.LockedMinor
		}
	}
	return amount, nil
}

func (s *Store) RecordTreasuryReserveCheck(ctx context.Context, provider, currency string, available, pending, liability, requested models.MinorUnits, purpose string) (*models.TreasuryReserveCheck, error) {
	now := time.Now().UTC()
	item := &models.TreasuryReserveCheck{
		ID: newUUID(), Provider: strings.ToLower(provider), Currency: normalizeCurrency(currency),
		ProviderAvailableMinor: available, ProviderPendingMinor: pending, LiabilityMinor: liability,
		RequestedMinor: requested, Purpose: purpose, CreatedAt: now,
	}
	switch purpose {
	case "deposit_settlement":
		item.Passed = available+pending >= liability+requested
	case "withdrawal_processing":
		item.Passed = available >= requested && available+pending >= liability
	case "reconciliation":
		item.Passed = available+pending >= liability
	default:
		return nil, errors.New("invalid treasury reserve-check purpose")
	}
	canonical := fmt.Sprintf("%s|%s|%s|%d|%d|%d|%d|%s|%t|%s",
		item.ID, item.Provider, item.Currency, available, pending, liability, requested, purpose, item.Passed, now.Format(time.RFC3339Nano))
	digest := sha256.Sum256([]byte(canonical))
	item.ImmutableHash = hex.EncodeToString(digest[:])
	artifact, err := s.StoreFinancialArtifact(ctx, "", "treasury_audit", "application/json", mustJSON(item))
	if err != nil {
		return nil, err
	}
	item.ArtifactKey = artifact.ObjectKey
	if s.usesPostgresAuth() {
		_, err := s.pg.ExecContext(ctx, `
INSERT INTO treasury_reserve_checks(id,provider,currency,provider_available_minor,provider_pending_minor,
 liability_minor,requested_minor,purpose,passed,immutable_hash,artifact_key,created_at)
VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12)`,
			item.ID, item.Provider, item.Currency, item.ProviderAvailableMinor, item.ProviderPendingMinor,
			item.LiabilityMinor, item.RequestedMinor, item.Purpose, item.Passed, item.ImmutableHash, item.ArtifactKey, item.CreatedAt)
		if err != nil {
			return nil, err
		}
	} else {
		s.mu.Lock()
		s.treasuryChecks[item.ID] = item
		s.mu.Unlock()
	}
	return item, nil
}

func (s *Store) WithFinancialSettlementLock(ctx context.Context, currency string, operation func() error) error {
	if operation == nil {
		return errors.New("financial settlement operation is required")
	}
	key := "skill-arena-financial:" + normalizeCurrency(currency)
	if s.usesPostgresAuth() {
		connection, err := s.pg.Conn(ctx)
		if err != nil {
			return err
		}
		defer connection.Close()
		if _, err := connection.ExecContext(ctx, `SELECT pg_advisory_lock(hashtext($1))`, key); err != nil {
			return err
		}
		defer connection.ExecContext(context.Background(), `SELECT pg_advisory_unlock(hashtext($1))`, key)
		return operation()
	}
	s.financialGate.Lock()
	defer s.financialGate.Unlock()
	return operation()
}

func validEvidenceType(value string) bool {
	switch value {
	case "identity", "address", "source_of_funds", "payout_destination":
		return true
	default:
		return false
	}
}

func validEvidenceContentType(value string) bool {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "application/pdf", "image/jpeg", "image/png":
		return true
	default:
		return false
	}
}

func validArtifactType(value string) bool {
	switch value {
	case "statement", "financial_export", "treasury_audit", "provider_audit":
		return true
	default:
		return false
	}
}

func mustJSON(value any) []byte {
	data, _ := json.Marshal(value)
	return data
}
