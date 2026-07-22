package db

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"skill-arena/internal/models"

	"github.com/lib/pq"
)

const authIdentitySchema = `
CREATE TABLE IF NOT EXISTS users (
 id TEXT PRIMARY KEY, email TEXT NOT NULL, country TEXT NOT NULL DEFAULT '', date_of_birth DATE,
 terms_accepted_at TIMESTAMPTZ, username TEXT NOT NULL, display_name TEXT NOT NULL,
 hidden_from_public BOOLEAN NOT NULL DEFAULT FALSE, password_hash TEXT NOT NULL,
 role TEXT NOT NULL DEFAULT 'player', email_verified BOOLEAN NOT NULL DEFAULT FALSE,
 kyc_status TEXT NOT NULL DEFAULT 'unverified', status TEXT NOT NULL DEFAULT 'active',
 created_at TIMESTAMPTZ NOT NULL, updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE TABLE IF NOT EXISTS auth_sessions (
 id TEXT PRIMARY KEY, user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
 refresh_token_hash TEXT NOT NULL UNIQUE, user_agent TEXT, ip_address TEXT, device_id TEXT, family_id TEXT,
 created_at TIMESTAMPTZ NOT NULL, expires_at TIMESTAMPTZ NOT NULL, revoked_at TIMESTAMPTZ,
 rotated_at TIMESTAMPTZ, mfa_verified BOOLEAN NOT NULL DEFAULT FALSE,
 enrollment_only BOOLEAN NOT NULL DEFAULT FALSE
);
CREATE TABLE IF NOT EXISTS devices (
 id TEXT PRIMARY KEY, user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
 fingerprint TEXT NOT NULL, device_name TEXT, os TEXT, browser TEXT,
 last_seen TIMESTAMPTZ NOT NULL, created_at TIMESTAMPTZ NOT NULL, revoked_at TIMESTAMPTZ,
 UNIQUE(user_id, fingerprint)
);
CREATE TABLE IF NOT EXISTS audit_logs (
 id TEXT PRIMARY KEY, actor_id TEXT REFERENCES users(id) ON DELETE SET NULL, action TEXT NOT NULL,
 target_id TEXT, metadata JSONB, ip_address TEXT, created_at TIMESTAMPTZ NOT NULL,
 previous_hash TEXT, entry_hash TEXT
);
ALTER TABLE users ADD COLUMN IF NOT EXISTS status TEXT NOT NULL DEFAULT 'active';
ALTER TABLE users ADD COLUMN IF NOT EXISTS updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW();
ALTER TABLE users ADD COLUMN IF NOT EXISTS country TEXT NOT NULL DEFAULT '';
ALTER TABLE users ADD COLUMN IF NOT EXISTS date_of_birth DATE;
ALTER TABLE users ADD COLUMN IF NOT EXISTS terms_accepted_at TIMESTAMPTZ;
ALTER TABLE auth_sessions ADD COLUMN IF NOT EXISTS rotated_at TIMESTAMPTZ;
ALTER TABLE auth_sessions ADD COLUMN IF NOT EXISTS mfa_verified BOOLEAN NOT NULL DEFAULT FALSE;
ALTER TABLE auth_sessions ADD COLUMN IF NOT EXISTS enrollment_only BOOLEAN NOT NULL DEFAULT FALSE;
ALTER TABLE auth_sessions ADD COLUMN IF NOT EXISTS device_id TEXT;
ALTER TABLE auth_sessions ADD COLUMN IF NOT EXISTS family_id TEXT;
ALTER TABLE devices ADD COLUMN IF NOT EXISTS revoked_at TIMESTAMPTZ;
ALTER TABLE audit_logs ADD COLUMN IF NOT EXISTS previous_hash TEXT;
ALTER TABLE audit_logs ADD COLUMN IF NOT EXISTS entry_hash TEXT;
CREATE TABLE IF NOT EXISTS auth_tokens (
 id TEXT PRIMARY KEY, user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
 purpose TEXT NOT NULL, token_hash TEXT NOT NULL UNIQUE, expires_at TIMESTAMPTZ NOT NULL,
 used_at TIMESTAMPTZ, created_at TIMESTAMPTZ NOT NULL, created_ip TEXT, used_ip TEXT
);
CREATE TABLE IF NOT EXISTS mfa_settings (
 user_id TEXT PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE, enabled BOOLEAN NOT NULL DEFAULT FALSE,
 totp_secret_ciphertext TEXT, recovery_code_hashes TEXT[] NOT NULL DEFAULT '{}',
 confirmed_at TIMESTAMPTZ, updated_at TIMESTAMPTZ NOT NULL
);
CREATE TABLE IF NOT EXISTS password_history (
 id BIGSERIAL PRIMARY KEY, user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
 password_hash TEXT NOT NULL, password_stamp TEXT, created_at TIMESTAMPTZ NOT NULL
);
CREATE TABLE IF NOT EXISTS login_security (
 user_id TEXT PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE, failed_count INTEGER NOT NULL DEFAULT 0,
 locked_until TIMESTAMPTZ, last_failed_at TIMESTAMPTZ, last_success_at TIMESTAMPTZ,
 last_ip_address TEXT, last_user_agent TEXT, suspicious_flag TEXT, updated_at TIMESTAMPTZ NOT NULL
);
CREATE UNIQUE INDEX IF NOT EXISTS idx_users_email_lower ON users (LOWER(email));
CREATE INDEX IF NOT EXISTS idx_auth_tokens_lookup ON auth_tokens (token_hash, purpose);
CREATE INDEX IF NOT EXISTS idx_auth_tokens_user_active ON auth_tokens (user_id, purpose, expires_at) WHERE used_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_auth_sessions_active ON auth_sessions (user_id, expires_at) WHERE revoked_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_auth_sessions_family ON auth_sessions (family_id) WHERE family_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_password_history_user_created ON password_history (user_id, created_at DESC);
CREATE UNIQUE INDEX IF NOT EXISTS idx_audit_logs_entry_hash ON audit_logs (entry_hash) WHERE entry_hash IS NOT NULL;
DO $$ BEGIN
 ALTER TABLE users ADD CONSTRAINT users_status_check CHECK (status IN ('active','suspended','disabled'));
EXCEPTION WHEN duplicate_object THEN NULL; END $$;
DO $$ BEGIN
 ALTER TABLE users ADD CONSTRAINT users_role_check CHECK (role IN ('player','admin','super_admin','treasury_manager','fraud_analyst','support','moderator'));
EXCEPTION WHEN duplicate_object THEN NULL; END $$;
DO $$ BEGIN
 ALTER TABLE users ADD CONSTRAINT users_country_check CHECK (country = '' OR country ~ '^[A-Z]{2}$');
EXCEPTION WHEN duplicate_object THEN NULL; END $$;
`

type rowScanner interface {
	Scan(...any) error
}

func (s *Store) usesPostgresAuth() bool {
	return s != nil && s.persistence == "postgres" && s.pg != nil
}

func (s *Store) initPostgresAuth(ctx context.Context) error {
	if _, err := s.pg.ExecContext(ctx, `CREATE TABLE IF NOT EXISTS schema_migrations (version TEXT PRIMARY KEY, checksum TEXT NOT NULL, applied_at TIMESTAMPTZ NOT NULL)`); err != nil {
		return err
	}
	checksumBytes := sha256.Sum256([]byte(authIdentitySchema))
	checksum := hex.EncodeToString(checksumBytes[:])
	var existing string
	err := s.pg.QueryRowContext(ctx, `SELECT checksum FROM schema_migrations WHERE version=$1`, "002_auth_identity").Scan(&existing)
	if err == nil && existing != checksum {
		return errors.New("migration 002_auth_identity checksum does not match the applied schema")
	}
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return err
	}
	if _, err = s.pg.ExecContext(ctx, authIdentitySchema); err != nil {
		return err
	}
	_, err = s.pg.ExecContext(ctx, `INSERT INTO schema_migrations(version,checksum,applied_at) VALUES($1,$2,$3) ON CONFLICT(version) DO NOTHING`, "002_auth_identity", checksum, time.Now().UTC())
	return err
}

func (s *Store) migrateLegacyIdentitySnapshot(ctx context.Context) error {
	s.mu.RLock()
	users := make([]*models.User, 0, len(s.users))
	for _, user := range s.users {
		copyUser := *user
		normalizeNewUser(&copyUser)
		users = append(users, &copyUser)
	}
	mfa := make([]*models.MFASettings, 0, len(s.mfa))
	for _, setting := range s.mfa {
		copySetting := *setting
		copySetting.RecoveryCodeHashes = append([]string(nil), setting.RecoveryCodeHashes...)
		mfa = append(mfa, &copySetting)
	}
	passwords := make(map[string][]*models.PasswordHistoryEntry, len(s.passwords))
	for userID, entries := range s.passwords {
		passwords[userID] = append([]*models.PasswordHistoryEntry(nil), entries...)
	}
	loginSecurity := make([]*models.LoginSecurityState, 0, len(s.loginSecurity))
	for _, state := range s.loginSecurity {
		copyState := *state
		loginSecurity = append(loginSecurity, &copyState)
	}
	devices := make([]*models.Device, 0)
	for _, entries := range s.devices {
		for _, device := range entries {
			copyDevice := *device
			devices = append(devices, &copyDevice)
		}
	}
	s.mu.RUnlock()
	if len(users) == 0 {
		return nil
	}
	tx, err := s.pg.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelSerializable})
	if err != nil {
		return err
	}
	defer tx.Rollback()
	for _, user := range users {
		_, err = tx.ExecContext(ctx, `INSERT INTO users (`+userColumns+`) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15) ON CONFLICT(id) DO NOTHING`,
			user.ID, user.Email, user.Country, user.DateOfBirth, user.TermsAcceptedAt, user.Username, user.DisplayName, user.HiddenFromPublic,
			user.PasswordHash, user.Role, user.EmailVerified, user.KYCStatus, user.Status, user.CreatedAt, user.UpdatedAt)
		if err != nil {
			return err
		}
		entries := passwords[user.ID]
		if len(entries) == 0 {
			entries = []*models.PasswordHistoryEntry{{UserID: user.ID, PasswordHash: user.PasswordHash, CreatedAt: user.CreatedAt}}
		}
		for _, entry := range entries {
			_, err = tx.ExecContext(ctx, `INSERT INTO password_history(user_id,password_hash,password_stamp,created_at) SELECT $1,$2,$3,$4 WHERE NOT EXISTS (SELECT 1 FROM password_history WHERE user_id=$1 AND password_hash=$2)`, user.ID, entry.PasswordHash, nullableString(entry.PasswordStamp), entry.CreatedAt)
			if err != nil {
				return err
			}
		}
	}
	for _, setting := range mfa {
		_, err = tx.ExecContext(ctx, `INSERT INTO mfa_settings(user_id,enabled,totp_secret_ciphertext,recovery_code_hashes,confirmed_at,updated_at) VALUES($1,$2,$3,$4,$5,$6) ON CONFLICT(user_id) DO NOTHING`, setting.UserID, setting.Enabled, nullableString(setting.TOTPSecretCiphertext), pq.Array(setting.RecoveryCodeHashes), setting.ConfirmedAt, setting.UpdatedAt)
		if err != nil {
			return err
		}
	}
	for _, state := range loginSecurity {
		_, err = tx.ExecContext(ctx, `INSERT INTO login_security(user_id,failed_count,locked_until,last_failed_at,last_success_at,last_ip_address,last_user_agent,suspicious_flag,updated_at) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9) ON CONFLICT(user_id) DO NOTHING`, state.UserID, state.FailedCount, state.LockedUntil, state.LastFailedAt, state.LastSuccessAt, nullableString(state.LastIPAddress), nullableString(state.LastUserAgent), nullableString(state.SuspiciousFlag), state.UpdatedAt)
		if err != nil {
			return err
		}
	}
	for _, device := range devices {
		_, err = tx.ExecContext(ctx, `INSERT INTO devices(id,user_id,fingerprint,device_name,os,browser,last_seen,created_at,revoked_at) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9) ON CONFLICT(user_id,fingerprint) DO NOTHING`, device.ID, device.UserID, device.Fingerprint, nullableString(device.DeviceName), nullableString(device.OS), nullableString(device.Browser), device.LastSeen, device.CreatedAt, device.RevokedAt)
		if err != nil {
			return err
		}
	}
	return tx.Commit()
}

func scanUser(row rowScanner) (*models.User, error) {
	user := &models.User{}
	var dateOfBirth, termsAccepted sql.NullTime
	err := row.Scan(&user.ID, &user.Email, &user.Country, &dateOfBirth, &termsAccepted, &user.Username, &user.DisplayName, &user.HiddenFromPublic,
		&user.PasswordHash, &user.Role, &user.EmailVerified, &user.KYCStatus, &user.Status, &user.CreatedAt, &user.UpdatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, errors.New("user not found")
	}
	if dateOfBirth.Valid {
		user.DateOfBirth = &dateOfBirth.Time
	}
	if termsAccepted.Valid {
		user.TermsAcceptedAt = &termsAccepted.Time
	}
	return user, err
}

const userColumns = `id, email, country, date_of_birth, terms_accepted_at, username, display_name, hidden_from_public, password_hash, role,
 email_verified, kyc_status, status, created_at, updated_at`

func normalizeNewUser(user *models.User) {
	now := time.Now().UTC()
	user.Email = strings.ToLower(strings.TrimSpace(user.Email))
	if user.ID == "" {
		user.ID = newUUID()
	}
	if user.Username == "" {
		user.Username = strings.Split(user.Email, "@")[0]
	}
	if user.DisplayName == "" {
		user.DisplayName = user.Username
	}
	if user.Role == "" {
		user.Role = models.RolePlayer
	}
	if user.KYCStatus == "" {
		user.KYCStatus = "unverified"
	}
	if user.Status == "" {
		user.Status = "active"
	}
	if user.CreatedAt.IsZero() {
		user.CreatedAt = now
	}
	user.UpdatedAt = now
}

func (s *Store) pgCreateUser(ctx context.Context, user *models.User) error {
	normalizeNewUser(user)
	if s.isConfiguredSuperAdminEmailLocked(user.Email) {
		user.Role = models.RoleSuperAdmin
		user.HiddenFromPublic = true
	} else {
		user.Role = models.RolePlayer
		user.HiddenFromPublic = false
	}
	tx, err := s.pg.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelSerializable})
	if err != nil {
		return err
	}
	defer tx.Rollback()
	_, err = tx.ExecContext(ctx, `INSERT INTO users (`+userColumns+`)
VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15)`, user.ID, user.Email, user.Country, user.DateOfBirth, user.TermsAcceptedAt, user.Username,
		user.DisplayName, user.HiddenFromPublic, user.PasswordHash, user.Role, user.EmailVerified,
		user.KYCStatus, user.Status, user.CreatedAt, user.UpdatedAt)
	if err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "unique") {
			return errors.New("email already exists")
		}
		return err
	}
	if user.PasswordHash != "" {
		_, err = tx.ExecContext(ctx, `INSERT INTO password_history(user_id,password_hash,password_stamp,created_at) VALUES($1,$2,'',$3)`, user.ID, user.PasswordHash, user.CreatedAt)
		if err != nil {
			return err
		}
	}
	return tx.Commit()
}

func (s *Store) pgGetUserByEmail(ctx context.Context, email string) (*models.User, error) {
	return scanUser(s.pg.QueryRowContext(ctx, `SELECT `+userColumns+` FROM users WHERE LOWER(email)=LOWER($1)`, strings.TrimSpace(email)))
}

func (s *Store) pgGetUserByID(ctx context.Context, userID string) (*models.User, error) {
	return scanUser(s.pg.QueryRowContext(ctx, `SELECT `+userColumns+` FROM users WHERE id=$1`, userID))
}

func nullableString(value string) any {
	if value == "" {
		return nil
	}
	return value
}

func auditHash(previous, id, actorID, action, targetID, ipAddress string, metadata map[string]string, createdAt time.Time) string {
	data, _ := json.Marshal(metadata)
	sum := sha256.Sum256([]byte(strings.Join([]string{previous, id, actorID, action, targetID, ipAddress, createdAt.UTC().Format(time.RFC3339Nano), string(data)}, "\x00")))
	return hex.EncodeToString(sum[:])
}

func pgAppendAuditTx(ctx context.Context, tx *sql.Tx, log *models.AuditLog) error {
	var previous sql.NullString
	err := tx.QueryRowContext(ctx, `SELECT entry_hash FROM audit_logs WHERE entry_hash IS NOT NULL ORDER BY created_at DESC,id DESC LIMIT 1 FOR SHARE`).Scan(&previous)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return err
	}
	metadata, err := json.Marshal(log.Metadata)
	if err != nil {
		return err
	}
	log.PreviousHash = previous.String
	log.EntryHash = auditHash(log.PreviousHash, log.ID, log.ActorID, log.Action, log.TargetID, log.IPAddress, log.Metadata, log.CreatedAt)
	_, err = tx.ExecContext(ctx, `INSERT INTO audit_logs(id,actor_id,action,target_id,metadata,ip_address,created_at,previous_hash,entry_hash)
VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9)`, log.ID, nullableString(log.ActorID), log.Action, nullableString(log.TargetID), metadata,
		nullableString(log.IPAddress), log.CreatedAt, nullableString(log.PreviousHash), log.EntryHash)
	return err
}

func (s *Store) pgAppendAudit(ctx context.Context, actorID, action, targetID, ipAddress string, metadata map[string]string) error {
	tx, err := s.pg.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	log := &models.AuditLog{ID: newUUID(), ActorID: actorID, Action: action, TargetID: targetID, IPAddress: ipAddress, Metadata: metadata, CreatedAt: time.Now().UTC()}
	if err := pgAppendAuditTx(ctx, tx, log); err != nil {
		return err
	}
	return tx.Commit()
}

func (s *Store) pgCreateAuthToken(ctx context.Context, userID, purpose, rawToken, ipAddress string, ttl time.Duration) (*models.AuthToken, error) {
	now := time.Now().UTC()
	token := &models.AuthToken{ID: newUUID(), UserID: userID, Purpose: purpose, TokenHash: hashToken(rawToken), ExpiresAt: now.Add(ttl), CreatedAt: now, CreatedIP: ipAddress}
	tx, err := s.pg.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelSerializable})
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	if _, err = tx.ExecContext(ctx, `UPDATE auth_tokens SET used_at=$1 WHERE user_id=$2 AND purpose=$3 AND used_at IS NULL AND expires_at>$1`, now, userID, purpose); err != nil {
		return nil, err
	}
	if _, err = tx.ExecContext(ctx, `INSERT INTO auth_tokens(id,user_id,purpose,token_hash,expires_at,created_at,created_ip) VALUES($1,$2,$3,$4,$5,$6,$7)`, token.ID, token.UserID, token.Purpose, token.TokenHash, token.ExpiresAt, token.CreatedAt, nullableString(token.CreatedIP)); err != nil {
		return nil, err
	}
	if err = pgAppendAuditTx(ctx, tx, &models.AuditLog{ID: newUUID(), ActorID: userID, Action: "auth.token.created." + purpose, TargetID: token.ID, IPAddress: ipAddress, CreatedAt: now}); err != nil {
		return nil, err
	}
	return token, tx.Commit()
}

func scanAuthToken(row rowScanner) (*models.AuthToken, error) {
	token := &models.AuthToken{}
	err := row.Scan(&token.ID, &token.UserID, &token.Purpose, &token.TokenHash, &token.ExpiresAt, &token.UsedAt, &token.CreatedAt, &token.CreatedIP, &token.UsedIP)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, errors.New("token not found")
	}
	return token, err
}

func (s *Store) pgInspectAuthToken(ctx context.Context, purpose, rawToken string) (*models.AuthToken, *models.User, error) {
	token, err := scanAuthToken(s.pg.QueryRowContext(ctx, `SELECT id,user_id,purpose,token_hash,expires_at,used_at,created_at,COALESCE(created_ip,''),COALESCE(used_ip,'') FROM auth_tokens WHERE purpose=$1 AND token_hash=$2`, purpose, hashToken(rawToken)))
	if err != nil {
		return nil, nil, err
	}
	if token.UsedAt != nil {
		return nil, nil, errors.New("token already used")
	}
	if !token.ExpiresAt.After(time.Now().UTC()) {
		return nil, nil, errors.New("token expired")
	}
	user, err := s.pgGetUserByID(ctx, token.UserID)
	return token, user, err
}

func (s *Store) pgConsumeAuthToken(ctx context.Context, purpose, rawToken, ipAddress string) (*models.AuthToken, *models.User, error) {
	now := time.Now().UTC()
	tx, err := s.pg.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelSerializable})
	if err != nil {
		return nil, nil, err
	}
	defer tx.Rollback()
	token, err := scanAuthToken(tx.QueryRowContext(ctx, `SELECT id,user_id,purpose,token_hash,expires_at,used_at,created_at,COALESCE(created_ip,''),COALESCE(used_ip,'') FROM auth_tokens WHERE purpose=$1 AND token_hash=$2 FOR UPDATE`, purpose, hashToken(rawToken)))
	if err != nil {
		return nil, nil, err
	}
	user, err := scanUser(tx.QueryRowContext(ctx, `SELECT `+userColumns+` FROM users WHERE id=$1`, token.UserID))
	if err != nil {
		return nil, nil, err
	}
	if token.UsedAt != nil {
		return token, user, errors.New("token already used")
	}
	if !token.ExpiresAt.After(now) {
		return nil, nil, errors.New("token expired")
	}
	if _, err = tx.ExecContext(ctx, `UPDATE auth_tokens SET used_at=$1,used_ip=$2 WHERE id=$3`, now, nullableString(ipAddress), token.ID); err != nil {
		return nil, nil, err
	}
	token.UsedAt = &now
	token.UsedIP = ipAddress
	if err = pgAppendAuditTx(ctx, tx, &models.AuditLog{ID: newUUID(), ActorID: user.ID, Action: "auth.token.consumed." + purpose, TargetID: token.ID, IPAddress: ipAddress, CreatedAt: now}); err != nil {
		return nil, nil, err
	}
	return token, user, tx.Commit()
}

func (s *Store) pgCreateAuthSession(ctx context.Context, userID, refreshToken, userAgent, ipAddress, deviceID string, ttl time.Duration, mfaVerified, enrollmentOnly bool) (*models.AuthSession, error) {
	now := time.Now().UTC()
	session := &models.AuthSession{ID: newUUID(), UserID: userID, RefreshTokenHash: hashToken(refreshToken), UserAgent: userAgent, IPAddress: ipAddress, DeviceID: deviceID, CreatedAt: now, ExpiresAt: now.Add(ttl), MFAVerified: mfaVerified, EnrollmentOnly: enrollmentOnly}
	session.FamilyID = session.ID
	tx, err := s.pg.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	var status string
	if err = tx.QueryRowContext(ctx, `SELECT status FROM users WHERE id=$1`, userID).Scan(&status); err != nil {
		return nil, errors.New("user not found")
	}
	if status != "active" {
		return nil, errors.New("account is not active")
	}
	_, err = tx.ExecContext(ctx, `INSERT INTO auth_sessions(id,user_id,refresh_token_hash,user_agent,ip_address,device_id,family_id,created_at,expires_at,mfa_verified,enrollment_only) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)`, session.ID, session.UserID, session.RefreshTokenHash, nullableString(session.UserAgent), nullableString(session.IPAddress), nullableString(session.DeviceID), session.FamilyID, session.CreatedAt, session.ExpiresAt, session.MFAVerified, session.EnrollmentOnly)
	if err != nil {
		return nil, err
	}
	if err = pgAppendAuditTx(ctx, tx, &models.AuditLog{ID: newUUID(), ActorID: userID, Action: "auth.session.created", TargetID: session.ID, IPAddress: ipAddress, CreatedAt: now}); err != nil {
		return nil, err
	}
	return session, tx.Commit()
}

func scanAuthSession(row rowScanner) (*models.AuthSession, error) {
	session := &models.AuthSession{}
	err := row.Scan(&session.ID, &session.UserID, &session.RefreshTokenHash, &session.UserAgent, &session.IPAddress, &session.DeviceID, &session.FamilyID, &session.CreatedAt, &session.ExpiresAt, &session.RevokedAt, &session.RotatedAt, &session.MFAVerified, &session.EnrollmentOnly)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, errors.New("session not found")
	}
	return session, err
}

const authSessionColumns = `id,user_id,refresh_token_hash,COALESCE(user_agent,''),COALESCE(ip_address,''),COALESCE(device_id,''),COALESCE(family_id,''),created_at,expires_at,revoked_at,rotated_at,mfa_verified,enrollment_only`

func (s *Store) pgValidateAuthSession(ctx context.Context, sessionID, userID string) (*models.AuthSession, *models.User, error) {
	session, err := scanAuthSession(s.pg.QueryRowContext(ctx, `SELECT `+authSessionColumns+` FROM auth_sessions WHERE id=$1 AND user_id=$2`, sessionID, userID))
	if err != nil || session.RevokedAt != nil || !session.ExpiresAt.After(time.Now().UTC()) {
		return nil, nil, errors.New("session is expired or revoked")
	}
	user, err := s.pgGetUserByID(ctx, userID)
	if err != nil || user.Status != "active" {
		return nil, nil, errors.New("account is not active")
	}
	return session, user, nil
}

func (s *Store) pgRotateRefreshToken(ctx context.Context, oldRefreshToken, newRefreshToken, userAgent, ipAddress string, ttl time.Duration) (*models.User, *models.AuthSession, error) {
	now := time.Now().UTC()
	tx, err := s.pg.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelSerializable})
	if err != nil {
		return nil, nil, err
	}
	defer tx.Rollback()
	old, err := scanAuthSession(tx.QueryRowContext(ctx, `SELECT `+authSessionColumns+` FROM auth_sessions WHERE refresh_token_hash=$1 FOR UPDATE`, hashToken(oldRefreshToken)))
	if err != nil {
		return nil, nil, errors.New("refresh token is expired or revoked")
	}
	if old.RevokedAt != nil && old.RotatedAt != nil {
		if old.FamilyID != "" {
			_, err = tx.ExecContext(ctx, `UPDATE auth_sessions SET revoked_at=$1 WHERE family_id=$2 AND revoked_at IS NULL`, now, old.FamilyID)
		} else {
			_, err = tx.ExecContext(ctx, `UPDATE auth_sessions SET revoked_at=$1 WHERE user_id=$2 AND revoked_at IS NULL`, now, old.UserID)
		}
		if err != nil {
			return nil, nil, err
		}
		if err = pgAppendAuditTx(ctx, tx, &models.AuditLog{ID: newUUID(), ActorID: old.UserID, Action: "auth.refresh.reuse_detected", TargetID: old.ID, IPAddress: ipAddress, CreatedAt: now}); err != nil {
			return nil, nil, err
		}
		if err = tx.Commit(); err != nil {
			return nil, nil, err
		}
		return nil, nil, errors.New("refresh token reuse detected")
	}
	if old.RevokedAt != nil || !old.ExpiresAt.After(now) {
		return nil, nil, errors.New("refresh token is expired or revoked")
	}
	user, err := scanUser(tx.QueryRowContext(ctx, `SELECT `+userColumns+` FROM users WHERE id=$1`, old.UserID))
	if err != nil || user.Status != "active" {
		return nil, nil, errors.New("account is not active")
	}
	if _, err = tx.ExecContext(ctx, `UPDATE auth_sessions SET revoked_at=$1,rotated_at=$1 WHERE id=$2`, now, old.ID); err != nil {
		return nil, nil, err
	}
	replacement := &models.AuthSession{ID: newUUID(), UserID: user.ID, RefreshTokenHash: hashToken(newRefreshToken), UserAgent: userAgent, IPAddress: ipAddress, DeviceID: old.DeviceID, FamilyID: old.FamilyID, CreatedAt: now, ExpiresAt: now.Add(ttl), MFAVerified: old.MFAVerified, EnrollmentOnly: old.EnrollmentOnly}
	if replacement.FamilyID == "" {
		replacement.FamilyID = old.ID
	}
	_, err = tx.ExecContext(ctx, `INSERT INTO auth_sessions(id,user_id,refresh_token_hash,user_agent,ip_address,device_id,family_id,created_at,expires_at,mfa_verified,enrollment_only) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)`, replacement.ID, replacement.UserID, replacement.RefreshTokenHash, nullableString(userAgent), nullableString(ipAddress), nullableString(replacement.DeviceID), replacement.FamilyID, replacement.CreatedAt, replacement.ExpiresAt, replacement.MFAVerified, replacement.EnrollmentOnly)
	if err != nil {
		return nil, nil, err
	}
	if err = pgAppendAuditTx(ctx, tx, &models.AuditLog{ID: newUUID(), ActorID: user.ID, Action: "auth.session.rotated", TargetID: replacement.ID, IPAddress: ipAddress, CreatedAt: now}); err != nil {
		return nil, nil, err
	}
	return user, replacement, tx.Commit()
}

func (s *Store) pgRevokeSession(ctx context.Context, sessionID, userID, actorID, ipAddress, reason string) error {
	now := time.Now().UTC()
	tx, err := s.pg.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	result, err := tx.ExecContext(ctx, `UPDATE auth_sessions SET revoked_at=$1 WHERE id=$2 AND user_id=$3 AND revoked_at IS NULL`, now, sessionID, userID)
	if err != nil {
		return err
	}
	count, _ := result.RowsAffected()
	if count == 0 {
		return errors.New("session not found")
	}
	if err = pgAppendAuditTx(ctx, tx, &models.AuditLog{ID: newUUID(), ActorID: actorID, Action: "auth.session.revoked", TargetID: sessionID, IPAddress: ipAddress, Metadata: map[string]string{"reason": reason}, CreatedAt: now}); err != nil {
		return err
	}
	return tx.Commit()
}

func (s *Store) pgRevokeUserSessions(ctx context.Context, userID, actorID, ipAddress, reason string) error {
	now := time.Now().UTC()
	tx, err := s.pg.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err = tx.ExecContext(ctx, `UPDATE auth_sessions SET revoked_at=$1 WHERE user_id=$2 AND revoked_at IS NULL`, now, userID); err != nil {
		return err
	}
	if err = pgAppendAuditTx(ctx, tx, &models.AuditLog{ID: newUUID(), ActorID: actorID, Action: "auth.sessions.revoked", TargetID: userID, IPAddress: ipAddress, Metadata: map[string]string{"reason": reason}, CreatedAt: now}); err != nil {
		return err
	}
	return tx.Commit()
}

func (s *Store) pgRevokeRefreshToken(ctx context.Context, refreshToken, actorID, ipAddress string) error {
	session, err := scanAuthSession(s.pg.QueryRowContext(ctx, `SELECT `+authSessionColumns+` FROM auth_sessions WHERE refresh_token_hash=$1`, hashToken(refreshToken)))
	if err != nil {
		return errors.New("refresh token not found")
	}
	return s.pgRevokeSession(ctx, session.ID, session.UserID, actorID, ipAddress, "logout")
}

func (s *Store) pgListAuthSessions(ctx context.Context, userID string) ([]*models.AuthSession, error) {
	rows, err := s.pg.QueryContext(ctx, `SELECT `+authSessionColumns+` FROM auth_sessions WHERE user_id=$1 ORDER BY created_at DESC LIMIT 100`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := []*models.AuthSession{}
	for rows.Next() {
		session, err := scanAuthSession(rows)
		if err != nil {
			return nil, err
		}
		result = append(result, session)
	}
	return result, rows.Err()
}

func (s *Store) pgMarkSessionMFA(ctx context.Context, sessionID, userID string, verified, enrollmentOnly bool) error {
	result, err := s.pg.ExecContext(ctx, `UPDATE auth_sessions SET mfa_verified=$1,enrollment_only=$2 WHERE id=$3 AND user_id=$4 AND revoked_at IS NULL`, verified, enrollmentOnly, sessionID, userID)
	if err != nil {
		return err
	}
	count, _ := result.RowsAffected()
	if count == 0 {
		return errors.New("session not found")
	}
	return nil
}

func (s *Store) pgLoginSecurityState(ctx context.Context, userID string) (*models.LoginSecurityState, error) {
	now := time.Now().UTC()
	_, err := s.pg.ExecContext(ctx, `INSERT INTO login_security(user_id,updated_at) VALUES($1,$2) ON CONFLICT(user_id) DO NOTHING`, userID, now)
	if err != nil {
		return nil, err
	}
	state := &models.LoginSecurityState{}
	err = s.pg.QueryRowContext(ctx, `SELECT user_id,failed_count,locked_until,last_failed_at,last_success_at,COALESCE(last_ip_address,''),COALESCE(last_user_agent,''),COALESCE(suspicious_flag,''),updated_at FROM login_security WHERE user_id=$1`, userID).Scan(&state.UserID, &state.FailedCount, &state.LockedUntil, &state.LastFailedAt, &state.LastSuccessAt, &state.LastIPAddress, &state.LastUserAgent, &state.SuspiciousFlag, &state.UpdatedAt)
	return state, err
}

func (s *Store) pgRecordLoginFailure(ctx context.Context, userID, ipAddress, userAgent string) (*models.LoginSecurityState, error) {
	now := time.Now().UTC()
	tx, err := s.pg.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	_, err = tx.ExecContext(ctx, `INSERT INTO login_security(user_id,failed_count,last_failed_at,updated_at) VALUES($1,1,$2,$2)
ON CONFLICT(user_id) DO UPDATE SET failed_count=login_security.failed_count+1,last_failed_at=$2,updated_at=$2,
locked_until=CASE WHEN login_security.failed_count+1>=5 THEN $2+INTERVAL '15 minutes' ELSE login_security.locked_until END`, userID, now)
	if err != nil {
		return nil, err
	}
	var failedCount int
	if err = tx.QueryRowContext(ctx, `SELECT failed_count FROM login_security WHERE user_id=$1`, userID).Scan(&failedCount); err != nil {
		return nil, err
	}
	if err = pgAppendAuditTx(ctx, tx, &models.AuditLog{ID: newUUID(), ActorID: userID, Action: "auth.login.failed", TargetID: userID, IPAddress: ipAddress, Metadata: map[string]string{"failedCount": fmt.Sprintf("%d", failedCount), "userAgent": userAgent}, CreatedAt: now}); err != nil {
		return nil, err
	}
	if err = tx.Commit(); err != nil {
		return nil, err
	}
	return s.pgLoginSecurityState(ctx, userID)
}

func (s *Store) pgRecordLoginSuccess(ctx context.Context, userID, ipAddress, userAgent string) error {
	now := time.Now().UTC()
	tx, err := s.pg.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	var previousIP string
	_ = tx.QueryRowContext(ctx, `SELECT COALESCE(last_ip_address,'') FROM login_security WHERE user_id=$1`, userID).Scan(&previousIP)
	suspicious := ""
	if previousIP != "" && previousIP != ipAddress {
		suspicious = "ip_changed"
		if err = pgAppendAuditTx(ctx, tx, &models.AuditLog{ID: newUUID(), ActorID: userID, Action: "auth.login.suspicious", TargetID: userID, IPAddress: ipAddress, Metadata: map[string]string{"previousIp": previousIP, "userAgent": userAgent}, CreatedAt: now}); err != nil {
			return err
		}
	}
	_, err = tx.ExecContext(ctx, `INSERT INTO login_security(user_id,last_success_at,last_ip_address,last_user_agent,suspicious_flag,updated_at) VALUES($1,$2,$3,$4,$5,$2)
ON CONFLICT(user_id) DO UPDATE SET failed_count=0,locked_until=NULL,last_success_at=$2,last_ip_address=$3,last_user_agent=$4,suspicious_flag=$5,updated_at=$2`, userID, now, nullableString(ipAddress), nullableString(userAgent), nullableString(suspicious))
	if err != nil {
		return err
	}
	return tx.Commit()
}

func (s *Store) pgPasswordHistory(ctx context.Context, userID string) ([]*models.PasswordHistoryEntry, error) {
	rows, err := s.pg.QueryContext(ctx, `SELECT user_id,password_hash,COALESCE(password_stamp,''),created_at FROM password_history WHERE user_id=$1 ORDER BY created_at DESC LIMIT 5`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := []*models.PasswordHistoryEntry{}
	for rows.Next() {
		entry := &models.PasswordHistoryEntry{}
		if err := rows.Scan(&entry.UserID, &entry.PasswordHash, &entry.PasswordStamp, &entry.CreatedAt); err != nil {
			return nil, err
		}
		result = append(result, entry)
	}
	return result, rows.Err()
}

func (s *Store) pgUpdatePassword(ctx context.Context, userID, passwordHash, passwordStamp, ipAddress string) error {
	now := time.Now().UTC()
	tx, err := s.pg.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelSerializable})
	if err != nil {
		return err
	}
	defer tx.Rollback()
	result, err := tx.ExecContext(ctx, `UPDATE users SET password_hash=$1,updated_at=$2 WHERE id=$3`, passwordHash, now, userID)
	if err != nil {
		return err
	}
	count, _ := result.RowsAffected()
	if count == 0 {
		return errors.New("user not found")
	}
	if _, err = tx.ExecContext(ctx, `INSERT INTO password_history(user_id,password_hash,password_stamp,created_at) VALUES($1,$2,$3,$4)`, userID, passwordHash, nullableString(passwordStamp), now); err != nil {
		return err
	}
	if _, err = tx.ExecContext(ctx, `DELETE FROM password_history WHERE id IN (SELECT id FROM password_history WHERE user_id=$1 ORDER BY created_at DESC OFFSET 5)`, userID); err != nil {
		return err
	}
	if err = pgAppendAuditTx(ctx, tx, &models.AuditLog{ID: newUUID(), ActorID: userID, Action: "auth.password.changed", TargetID: userID, IPAddress: ipAddress, CreatedAt: now}); err != nil {
		return err
	}
	return tx.Commit()
}

func (s *Store) pgCompletePasswordReset(ctx context.Context, rawToken, passwordHash, ipAddress string) error {
	now := time.Now().UTC()
	tx, err := s.pg.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelSerializable})
	if err != nil {
		return err
	}
	defer tx.Rollback()
	token, err := scanAuthToken(tx.QueryRowContext(ctx, `SELECT id,user_id,purpose,token_hash,expires_at,used_at,created_at,COALESCE(created_ip,''),COALESCE(used_ip,'') FROM auth_tokens WHERE purpose=$1 AND token_hash=$2 FOR UPDATE`, models.AuthTokenPurposePasswordReset, hashToken(rawToken)))
	if err != nil {
		return err
	}
	if token.UsedAt != nil {
		return errors.New("token already used")
	}
	if !token.ExpiresAt.After(now) {
		return errors.New("token expired")
	}
	if _, err = tx.ExecContext(ctx, `UPDATE auth_tokens SET used_at=$1,used_ip=$2 WHERE id=$3`, now, nullableString(ipAddress), token.ID); err != nil {
		return err
	}
	result, err := tx.ExecContext(ctx, `UPDATE users SET password_hash=$1,updated_at=$2 WHERE id=$3`, passwordHash, now, token.UserID)
	if err != nil {
		return err
	}
	if count, _ := result.RowsAffected(); count == 0 {
		return errors.New("user not found")
	}
	if _, err = tx.ExecContext(ctx, `INSERT INTO password_history(user_id,password_hash,created_at) VALUES($1,$2,$3)`, token.UserID, passwordHash, now); err != nil {
		return err
	}
	if _, err = tx.ExecContext(ctx, `DELETE FROM password_history WHERE id IN (SELECT id FROM password_history WHERE user_id=$1 ORDER BY created_at DESC OFFSET 5)`, token.UserID); err != nil {
		return err
	}
	if _, err = tx.ExecContext(ctx, `UPDATE auth_sessions SET revoked_at=$1 WHERE user_id=$2 AND revoked_at IS NULL`, now, token.UserID); err != nil {
		return err
	}
	for _, audit := range []*models.AuditLog{
		{ID: newUUID(), ActorID: token.UserID, Action: "auth.token.consumed." + models.AuthTokenPurposePasswordReset, TargetID: token.ID, IPAddress: ipAddress, CreatedAt: now},
		{ID: newUUID(), ActorID: token.UserID, Action: "auth.password.reset", TargetID: token.UserID, IPAddress: ipAddress, CreatedAt: now},
		{ID: newUUID(), ActorID: token.UserID, Action: "auth.sessions.revoked", TargetID: token.UserID, IPAddress: ipAddress, Metadata: map[string]string{"reason": "password_reset"}, CreatedAt: now},
	} {
		if err = pgAppendAuditTx(ctx, tx, audit); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func (s *Store) pgGetMFASettings(ctx context.Context, userID string) (*models.MFASettings, error) {
	setting := &models.MFASettings{UserID: userID}
	var confirmed sql.NullTime
	err := s.pg.QueryRowContext(ctx, `SELECT enabled,COALESCE(totp_secret_ciphertext,''),recovery_code_hashes,confirmed_at,updated_at FROM mfa_settings WHERE user_id=$1`, userID).Scan(&setting.Enabled, &setting.TOTPSecretCiphertext, pq.Array(&setting.RecoveryCodeHashes), &confirmed, &setting.UpdatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return setting, nil
	}
	if confirmed.Valid {
		setting.ConfirmedAt = confirmed.Time
	}
	return setting, err
}

func (s *Store) pgSaveMFASettings(ctx context.Context, setting *models.MFASettings, actorID, ipAddress string) error {
	now := time.Now().UTC()
	setting.UpdatedAt = now
	tx, err := s.pg.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	var confirmed any
	if !setting.ConfirmedAt.IsZero() {
		confirmed = setting.ConfirmedAt
	}
	_, err = tx.ExecContext(ctx, `INSERT INTO mfa_settings(user_id,enabled,totp_secret_ciphertext,recovery_code_hashes,confirmed_at,updated_at) VALUES($1,$2,$3,$4,$5,$6)
ON CONFLICT(user_id) DO UPDATE SET enabled=$2,totp_secret_ciphertext=$3,recovery_code_hashes=$4,confirmed_at=$5,updated_at=$6`, setting.UserID, setting.Enabled, nullableString(setting.TOTPSecretCiphertext), pq.Array(setting.RecoveryCodeHashes), confirmed, now)
	if err != nil {
		return err
	}
	if err = pgAppendAuditTx(ctx, tx, &models.AuditLog{ID: newUUID(), ActorID: actorID, Action: "auth.mfa.updated", TargetID: setting.UserID, IPAddress: ipAddress, Metadata: map[string]string{"enabled": fmt.Sprintf("%t", setting.Enabled)}, CreatedAt: now}); err != nil {
		return err
	}
	return tx.Commit()
}

func (s *Store) pgConsumeRecoveryCode(ctx context.Context, userID, codeHash, ipAddress string) (bool, error) {
	tx, err := s.pg.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelSerializable})
	if err != nil {
		return false, err
	}
	defer tx.Rollback()
	setting := &models.MFASettings{}
	if err = tx.QueryRowContext(ctx, `SELECT enabled,recovery_code_hashes FROM mfa_settings WHERE user_id=$1 FOR UPDATE`, userID).Scan(&setting.Enabled, pq.Array(&setting.RecoveryCodeHashes)); err != nil || !setting.Enabled {
		return false, nil
	}
	remaining := make([]string, 0, len(setting.RecoveryCodeHashes))
	used := false
	for _, stored := range setting.RecoveryCodeHashes {
		if !used && stored == codeHash {
			used = true
			continue
		}
		remaining = append(remaining, stored)
	}
	if !used {
		return false, nil
	}
	now := time.Now().UTC()
	if _, err = tx.ExecContext(ctx, `UPDATE mfa_settings SET recovery_code_hashes=$1,updated_at=$2 WHERE user_id=$3`, pq.Array(remaining), now, userID); err != nil {
		return false, err
	}
	if err = pgAppendAuditTx(ctx, tx, &models.AuditLog{ID: newUUID(), ActorID: userID, Action: "auth.mfa.recovery_code.used", TargetID: userID, IPAddress: ipAddress, CreatedAt: now}); err != nil {
		return false, err
	}
	return true, tx.Commit()
}

func (s *Store) pgVerifyEmail(ctx context.Context, userID string) error {
	result, err := s.pg.ExecContext(ctx, `UPDATE users SET email_verified=TRUE,updated_at=$1 WHERE id=$2`, time.Now().UTC(), userID)
	if err != nil {
		return err
	}
	count, _ := result.RowsAffected()
	if count == 0 {
		return errors.New("user not found")
	}
	return nil
}

func (s *Store) pgGetAuditLogs(ctx context.Context, limit int) ([]*models.AuditLog, error) {
	rows, err := s.pg.QueryContext(ctx, `SELECT id,COALESCE(actor_id,''),action,COALESCE(target_id,''),COALESCE(metadata,'{}'::jsonb),COALESCE(ip_address,''),created_at,COALESCE(previous_hash,''),COALESCE(entry_hash,'') FROM audit_logs ORDER BY created_at DESC,id DESC LIMIT $1`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := []*models.AuditLog{}
	for rows.Next() {
		log := &models.AuditLog{}
		var metadata []byte
		if err := rows.Scan(&log.ID, &log.ActorID, &log.Action, &log.TargetID, &metadata, &log.IPAddress, &log.CreatedAt, &log.PreviousHash, &log.EntryHash); err != nil {
			return nil, err
		}
		_ = json.Unmarshal(metadata, &log.Metadata)
		result = append(result, log)
	}
	return result, rows.Err()
}

func scanDevice(row rowScanner) (*models.Device, error) {
	device := &models.Device{}
	err := row.Scan(&device.ID, &device.UserID, &device.Fingerprint, &device.DeviceName, &device.OS, &device.Browser, &device.LastSeen, &device.CreatedAt, &device.RevokedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, errors.New("device not found")
	}
	return device, err
}

const deviceColumns = `id,user_id,fingerprint,COALESCE(device_name,''),COALESCE(os,''),COALESCE(browser,''),last_seen,created_at,revoked_at`

func (s *Store) pgRegisterDevice(ctx context.Context, userID, fingerprint, deviceName, osName, browser string) (*models.Device, error) {
	now := time.Now().UTC()
	device := &models.Device{ID: newUUID(), UserID: userID, Fingerprint: fingerprint, DeviceName: deviceName, OS: osName, Browser: browser, LastSeen: now, CreatedAt: now}
	return scanDevice(s.pg.QueryRowContext(ctx, `INSERT INTO devices(id,user_id,fingerprint,device_name,os,browser,last_seen,created_at,revoked_at)
VALUES($1,$2,$3,$4,$5,$6,$7,$8,NULL)
ON CONFLICT(user_id,fingerprint) DO UPDATE SET device_name=$4,os=$5,browser=$6,last_seen=$7,revoked_at=NULL
RETURNING `+deviceColumns, device.ID, device.UserID, device.Fingerprint, nullableString(device.DeviceName), nullableString(device.OS), nullableString(device.Browser), device.LastSeen, device.CreatedAt))
}

func (s *Store) pgListDevices(ctx context.Context, userID string) ([]*models.Device, error) {
	rows, err := s.pg.QueryContext(ctx, `SELECT `+deviceColumns+` FROM devices WHERE user_id=$1 ORDER BY last_seen DESC`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := []*models.Device{}
	for rows.Next() {
		device, err := scanDevice(rows)
		if err != nil {
			return nil, err
		}
		result = append(result, device)
	}
	return result, rows.Err()
}

func (s *Store) pgRevokeDevice(ctx context.Context, userID, deviceID, actorID, ipAddress string) error {
	now := time.Now().UTC()
	tx, err := s.pg.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	result, err := tx.ExecContext(ctx, `UPDATE devices SET revoked_at=$1 WHERE id=$2 AND user_id=$3 AND revoked_at IS NULL`, now, deviceID, userID)
	if err != nil {
		return err
	}
	count, _ := result.RowsAffected()
	if count == 0 {
		return errors.New("device not found")
	}
	if _, err = tx.ExecContext(ctx, `UPDATE auth_sessions SET revoked_at=$1 WHERE user_id=$2 AND device_id=$3 AND revoked_at IS NULL`, now, userID, deviceID); err != nil {
		return err
	}
	if err = pgAppendAuditTx(ctx, tx, &models.AuditLog{ID: newUUID(), ActorID: actorID, Action: "auth.device.revoked", TargetID: deviceID, IPAddress: ipAddress, CreatedAt: now}); err != nil {
		return err
	}
	return tx.Commit()
}

func (s *Store) AuthHealth(ctx context.Context) error {
	if s == nil {
		return errors.New("store unavailable")
	}
	if s.usesPostgresAuth() {
		if err := s.pg.PingContext(ctx); err != nil {
			return fmt.Errorf("postgres: %w", err)
		}
	}
	if s.redis != nil {
		if err := s.redis.Health(ctx); err != nil {
			return fmt.Errorf("redis: %w", err)
		}
	}
	return nil
}
