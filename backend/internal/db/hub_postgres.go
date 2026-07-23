package db

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"skill-arena/internal/models"
)

const arenaHubSchema = `
CREATE TABLE IF NOT EXISTS player_profiles (
 user_id TEXT PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
 username TEXT NOT NULL, display_name TEXT NOT NULL, avatar_url TEXT NOT NULL DEFAULT '',
 country TEXT NOT NULL DEFAULT '', language TEXT NOT NULL DEFAULT 'en',
 created_at TIMESTAMPTZ NOT NULL, updated_at TIMESTAMPTZ NOT NULL
);
CREATE TABLE IF NOT EXISTS progression (
 user_id TEXT PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
 xp INTEGER NOT NULL DEFAULT 0, level INTEGER NOT NULL DEFAULT 1, prestige INTEGER NOT NULL DEFAULT 0,
 elo_rating INTEGER NOT NULL DEFAULT 1200, league_tier TEXT NOT NULL DEFAULT 'Bronze',
 season_points INTEGER NOT NULL DEFAULT 0, legacy_points INTEGER NOT NULL DEFAULT 0,
 house_reputation INTEGER NOT NULL DEFAULT 0, matches_played INTEGER NOT NULL DEFAULT 0,
 wins INTEGER NOT NULL DEFAULT 0, losses INTEGER NOT NULL DEFAULT 0, current_streak INTEGER NOT NULL DEFAULT 0,
 best_moves INTEGER NOT NULL DEFAULT 0, trust_score NUMERIC(5,2) NOT NULL DEFAULT 100,
 trust_tier TEXT NOT NULL DEFAULT 'trusted', updated_at TIMESTAMPTZ NOT NULL
);
CREATE TABLE IF NOT EXISTS game_modules (
 id TEXT PRIMARY KEY, name TEXT NOT NULL, description TEXT NOT NULL, category TEXT NOT NULL,
 version TEXT NOT NULL, renderer_key TEXT NOT NULL, modes JSONB NOT NULL,
 average_time_seconds INTEGER NOT NULL, capabilities JSONB NOT NULL, rules_summary JSONB NOT NULL,
 availability TEXT NOT NULL, availability_reason TEXT NOT NULL DEFAULT '', updated_at TIMESTAMPTZ NOT NULL
);
CREATE TABLE IF NOT EXISTS player_notifications (
 id TEXT PRIMARY KEY, user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
 category TEXT NOT NULL, title TEXT NOT NULL, message TEXT NOT NULL, status TEXT NOT NULL,
 action_url TEXT NOT NULL DEFAULT '', metadata JSONB NOT NULL DEFAULT '{}',
 created_at TIMESTAMPTZ NOT NULL, read_at TIMESTAMPTZ, archived_at TIMESTAMPTZ
);
CREATE TABLE IF NOT EXISTS notification_events (
 sequence BIGSERIAL PRIMARY KEY, notification_id TEXT NOT NULL REFERENCES player_notifications(id) ON DELETE CASCADE,
 user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE, event_type TEXT NOT NULL,
 payload JSONB NOT NULL, created_at TIMESTAMPTZ NOT NULL
);
CREATE TABLE IF NOT EXISTS support_tickets (
 id TEXT PRIMARY KEY, user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
 category TEXT NOT NULL, subject TEXT NOT NULL, message TEXT NOT NULL, status TEXT NOT NULL,
 created_at TIMESTAMPTZ NOT NULL, updated_at TIMESTAMPTZ NOT NULL
);
CREATE UNIQUE INDEX IF NOT EXISTS idx_player_profiles_username_lower ON player_profiles(LOWER(username));
CREATE INDEX IF NOT EXISTS idx_progression_rank ON progression(elo_rating DESC, xp DESC);
CREATE INDEX IF NOT EXISTS idx_game_modules_availability ON game_modules(availability, name);
CREATE INDEX IF NOT EXISTS idx_notifications_user_status_created ON player_notifications(user_id, status, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_notification_events_user_sequence ON notification_events(user_id, sequence);
CREATE INDEX IF NOT EXISTS idx_support_tickets_user_updated ON support_tickets(user_id, updated_at DESC);
DO $$ BEGIN
 ALTER TABLE player_notifications ADD CONSTRAINT player_notifications_status_check
 CHECK (status IN ('unread','read','archived'));
EXCEPTION WHEN duplicate_object THEN NULL; END $$;
DO $$ BEGIN
 ALTER TABLE support_tickets ADD CONSTRAINT support_tickets_status_check
 CHECK (status IN ('open','received','closed'));
EXCEPTION WHEN duplicate_object THEN NULL; END $$;
`

type hubDevelopmentState struct {
	Profiles      map[string]*models.PlayerProfile   `json:"profiles"`
	Notifications map[string][]*models.Notification  `json:"notifications"`
	Tickets       map[string][]*models.SupportTicket `json:"tickets"`
}

var (
	playerUsernamePattern = regexp.MustCompile(`^[A-Za-z0-9_]{3,30}$`)
	countryCodePattern    = regexp.MustCompile(`^[A-Z]{2}$`)
	languageCodePattern   = regexp.MustCompile(`^[a-z]{2}(?:-[a-z]{2})?$`)
)

func (s *Store) initPostgresHub(ctx context.Context) error {
	checksumBytes := sha256.Sum256([]byte(arenaHubSchema))
	checksum := hex.EncodeToString(checksumBytes[:])
	var existing string
	err := s.pg.QueryRowContext(ctx, `SELECT checksum FROM schema_migrations WHERE version=$1`, "003_arena_hub").Scan(&existing)
	if err == nil && existing != checksum {
		return errors.New("migration 003_arena_hub checksum does not match the applied schema")
	}
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return err
	}
	if _, err = s.pg.ExecContext(ctx, arenaHubSchema); err != nil {
		return err
	}
	_, err = s.pg.ExecContext(ctx, `
INSERT INTO schema_migrations(version,checksum,applied_at)
VALUES($1,$2,$3) ON CONFLICT(version) DO NOTHING`, "003_arena_hub", checksum, time.Now().UTC())
	return err
}

func (s *Store) migrateLegacyHubState(ctx context.Context) error {
	if !s.usesPostgresAuth() {
		return nil
	}
	if err := s.loadOrSeedProgression(ctx); err != nil {
		return err
	}
	return s.syncGameCatalog(ctx)
}

func (s *Store) loadOrSeedProgression(ctx context.Context) error {
	var count int
	if err := s.pg.QueryRowContext(ctx, `SELECT COUNT(*) FROM progression`).Scan(&count); err != nil {
		return err
	}
	if count == 0 {
		s.mu.RLock()
		profiles := make([]models.Progression, 0, len(s.profiles))
		for _, progression := range s.profiles {
			profiles = append(profiles, *progression)
		}
		s.mu.RUnlock()
		for i := range profiles {
			var userExists bool
			if err := s.pg.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM users WHERE id=$1)`, profiles[i].UserID).Scan(&userExists); err != nil {
				return err
			}
			if !userExists {
				continue
			}
			if err := s.upsertProgression(ctx, &profiles[i]); err != nil {
				return err
			}
		}
		return nil
	}

	rows, err := s.pg.QueryContext(ctx, `
SELECT user_id,xp,level,prestige,elo_rating,league_tier,season_points,legacy_points,
 house_reputation,matches_played,wins,losses,current_streak,COALESCE(best_moves,0),trust_score,trust_tier,updated_at
FROM progression`)
	if err != nil {
		return err
	}
	defer rows.Close()
	loaded := map[string]*models.Progression{}
	for rows.Next() {
		progression := &models.Progression{}
		if err := rows.Scan(
			&progression.UserID, &progression.XP, &progression.Level, &progression.Prestige,
			&progression.EloRating, &progression.LeagueTier, &progression.SeasonPoints,
			&progression.LegacyPoints, &progression.HouseRep, &progression.MatchesPlayed,
			&progression.Wins, &progression.Losses, &progression.CurrentStreak,
			&progression.BestMoves, &progression.TrustScore, &progression.TrustTier,
			&progression.UpdatedAt,
		); err != nil {
			return err
		}
		loaded[progression.UserID] = progression
	}
	if err := rows.Err(); err != nil {
		return err
	}
	s.mu.Lock()
	for userID, progression := range loaded {
		s.profiles[userID] = progression
	}
	s.mu.Unlock()
	return nil
}

func (s *Store) upsertProgression(ctx context.Context, progression *models.Progression) error {
	if progression == nil {
		return nil
	}
	_, err := s.pg.ExecContext(ctx, `
INSERT INTO progression(
 user_id,xp,level,prestige,elo_rating,league_tier,season_points,legacy_points,
 house_reputation,matches_played,wins,losses,current_streak,best_moves,trust_score,trust_tier,updated_at
) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17)
ON CONFLICT(user_id) DO UPDATE SET
 xp=EXCLUDED.xp,level=EXCLUDED.level,prestige=EXCLUDED.prestige,elo_rating=EXCLUDED.elo_rating,
 league_tier=EXCLUDED.league_tier,season_points=EXCLUDED.season_points,legacy_points=EXCLUDED.legacy_points,
 house_reputation=EXCLUDED.house_reputation,matches_played=EXCLUDED.matches_played,wins=EXCLUDED.wins,
 losses=EXCLUDED.losses,current_streak=EXCLUDED.current_streak,best_moves=EXCLUDED.best_moves,
 trust_score=EXCLUDED.trust_score,trust_tier=EXCLUDED.trust_tier,updated_at=EXCLUDED.updated_at`,
		progression.UserID, progression.XP, progression.Level, progression.Prestige,
		progression.EloRating, progression.LeagueTier, progression.SeasonPoints,
		progression.LegacyPoints, progression.HouseRep, progression.MatchesPlayed,
		progression.Wins, progression.Losses, progression.CurrentStreak,
		progression.BestMoves, progression.TrustScore, progression.TrustTier,
		progression.UpdatedAt,
	)
	return err
}

func (s *Store) persistHubProgressionLocked(ctx context.Context) error {
	if !s.usesPostgresAuth() {
		return nil
	}
	for _, progression := range s.profiles {
		var userExists bool
		if err := s.pg.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM users WHERE id=$1)`, progression.UserID).Scan(&userExists); err != nil {
			return err
		}
		if !userExists {
			continue
		}
		if err := s.upsertProgression(ctx, progression); err != nil {
			return err
		}
	}
	return nil
}

func (s *Store) loadHubState() error {
	path := filepath.Join(s.dataDir, "arena_hub.json")
	content, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	var state hubDevelopmentState
	if err := json.Unmarshal(content, &state); err != nil {
		return err
	}
	if state.Profiles != nil {
		s.playerProfiles = state.Profiles
	}
	if state.Notifications != nil {
		s.notifications = state.Notifications
	}
	if state.Tickets != nil {
		s.supportTickets = state.Tickets
	}
	return nil
}

func (s *Store) persistHubStateLocked() error {
	if s.persistence == "postgres" {
		return nil
	}
	state := hubDevelopmentState{
		Profiles:      s.playerProfiles,
		Notifications: s.notifications,
		Tickets:       s.supportTickets,
	}
	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return err
	}
	return writeFileAtomic(filepath.Join(s.dataDir, "arena_hub.json"), data, 0o644)
}

func defaultPlayerProfile(user *models.User) *models.PlayerProfile {
	now := time.Now().UTC()
	username := strings.TrimSpace(user.Username)
	if username == "" {
		username = strings.Split(user.Email, "@")[0]
	}
	displayName := strings.TrimSpace(user.DisplayName)
	if displayName == "" {
		displayName = username
	}
	return &models.PlayerProfile{
		UserID:      user.ID,
		Username:    username,
		DisplayName: displayName,
		Country:     strings.ToUpper(user.Country),
		Language:    "en",
		CreatedAt:   now,
		UpdatedAt:   now,
	}
}

func (s *Store) GetPlayerProfile(ctx context.Context, userID string) (*models.PlayerProfile, error) {
	user, err := s.GetUserByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	if s.usesPostgresAuth() {
		profile := &models.PlayerProfile{}
		err := s.pg.QueryRowContext(ctx, `
SELECT user_id,username,display_name,avatar_url,country,language,created_at,updated_at
FROM player_profiles WHERE user_id=$1`, userID).Scan(
			&profile.UserID, &profile.Username, &profile.DisplayName, &profile.AvatarURL,
			&profile.Country, &profile.Language, &profile.CreatedAt, &profile.UpdatedAt,
		)
		if err == nil {
			return profile, nil
		}
		if !errors.Is(err, sql.ErrNoRows) {
			return nil, err
		}
		profile = defaultPlayerProfile(user)
		_, err = s.pg.ExecContext(ctx, `
INSERT INTO player_profiles(user_id,username,display_name,avatar_url,country,language,created_at,updated_at)
VALUES($1,$2,$3,$4,$5,$6,$7,$8)`,
			profile.UserID, profile.Username, profile.DisplayName, profile.AvatarURL,
			profile.Country, profile.Language, profile.CreatedAt, profile.UpdatedAt,
		)
		return profile, err
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	if profile := s.playerProfiles[userID]; profile != nil {
		copyProfile := *profile
		return &copyProfile, nil
	}
	profile := defaultPlayerProfile(user)
	s.playerProfiles[userID] = profile
	if err := s.persistHubStateLocked(); err != nil {
		return nil, err
	}
	copyProfile := *profile
	return &copyProfile, nil
}

func (s *Store) UpdatePlayerProfile(ctx context.Context, profile *models.PlayerProfile) (*models.PlayerProfile, error) {
	if profile == nil || profile.UserID == "" {
		return nil, errors.New("profile is required")
	}
	profile.Username = strings.TrimSpace(profile.Username)
	profile.DisplayName = strings.TrimSpace(profile.DisplayName)
	profile.Country = strings.ToUpper(strings.TrimSpace(profile.Country))
	profile.Language = strings.ToLower(strings.TrimSpace(profile.Language))
	profile.AvatarURL = strings.TrimSpace(profile.AvatarURL)
	if !playerUsernamePattern.MatchString(profile.Username) {
		return nil, errors.New("username must contain 3 to 30 letters, numbers, or underscores")
	}
	if len(profile.DisplayName) < 2 || len(profile.DisplayName) > 60 {
		return nil, errors.New("displayName must contain 2 to 60 characters")
	}
	if !countryCodePattern.MatchString(profile.Country) {
		return nil, errors.New("country must be a two-letter ISO code")
	}
	if !languageCodePattern.MatchString(profile.Language) {
		return nil, errors.New("language must be a supported language code")
	}
	profile.UpdatedAt = time.Now().UTC()
	if profile.CreatedAt.IsZero() {
		profile.CreatedAt = profile.UpdatedAt
	}
	if s.usesPostgresAuth() {
		tx, err := s.pg.BeginTx(ctx, nil)
		if err != nil {
			return nil, err
		}
		defer tx.Rollback()
		if _, err = tx.ExecContext(ctx, `
UPDATE users SET username=$2,display_name=$3,country=$4,updated_at=$5 WHERE id=$1`,
			profile.UserID, profile.Username, profile.DisplayName, profile.Country, profile.UpdatedAt); err != nil {
			return nil, err
		}
		if _, err = tx.ExecContext(ctx, `
INSERT INTO player_profiles(user_id,username,display_name,avatar_url,country,language,created_at,updated_at)
VALUES($1,$2,$3,$4,$5,$6,$7,$8)
ON CONFLICT(user_id) DO UPDATE SET username=EXCLUDED.username,display_name=EXCLUDED.display_name,
 avatar_url=EXCLUDED.avatar_url,country=EXCLUDED.country,language=EXCLUDED.language,updated_at=EXCLUDED.updated_at`,
			profile.UserID, profile.Username, profile.DisplayName, profile.AvatarURL,
			profile.Country, profile.Language, profile.CreatedAt, profile.UpdatedAt); err != nil {
			return nil, err
		}
		if err = tx.Commit(); err != nil {
			return nil, err
		}
		copyProfile := *profile
		return &copyProfile, nil
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	for userID, existing := range s.playerProfiles {
		if userID != profile.UserID && strings.EqualFold(existing.Username, profile.Username) {
			return nil, errors.New("username is already in use")
		}
	}
	if user := s.users[profile.UserID]; user != nil {
		user.Username = profile.Username
		user.DisplayName = profile.DisplayName
		user.Country = profile.Country
		user.UpdatedAt = profile.UpdatedAt
	}
	copyProfile := *profile
	s.playerProfiles[profile.UserID] = &copyProfile
	if err := s.persistUsers(); err != nil {
		return nil, err
	}
	if err := s.persistHubStateLocked(); err != nil {
		return nil, err
	}
	return &copyProfile, nil
}

func (s *Store) GetHubProgression(ctx context.Context, userID string) (*models.Progression, error) {
	if !s.usesPostgresAuth() {
		return s.GetProgressionByUserID(ctx, userID)
	}
	progression := &models.Progression{}
	err := s.pg.QueryRowContext(ctx, `
SELECT user_id,xp,level,prestige,elo_rating,league_tier,season_points,legacy_points,
 house_reputation,matches_played,wins,losses,current_streak,best_moves,trust_score,trust_tier,updated_at
FROM progression WHERE user_id=$1`, userID).Scan(
		&progression.UserID, &progression.XP, &progression.Level, &progression.Prestige,
		&progression.EloRating, &progression.LeagueTier, &progression.SeasonPoints,
		&progression.LegacyPoints, &progression.HouseRep, &progression.MatchesPlayed,
		&progression.Wins, &progression.Losses, &progression.CurrentStreak,
		&progression.BestMoves, &progression.TrustScore, &progression.TrustTier,
		&progression.UpdatedAt,
	)
	if err == nil {
		return progression, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return nil, err
	}
	s.mu.Lock()
	seed := *s.ensureProgressionLocked(userID)
	s.mu.Unlock()
	if err := s.upsertProgression(ctx, &seed); err != nil {
		return nil, err
	}
	return &seed, nil
}

func (s *Store) syncGameCatalog(ctx context.Context) error {
	if !s.usesPostgresAuth() {
		return nil
	}
	for _, gameEntry := range s.gameCatalogEntries() {
		modes, _ := json.Marshal(gameEntry.Modes)
		capabilities, _ := json.Marshal(gameEntry.Capabilities)
		rules, _ := json.Marshal(gameEntry.RulesSummary)
		if _, err := s.pg.ExecContext(ctx, `
INSERT INTO game_modules(
 id,name,description,category,version,renderer_key,modes,average_time_seconds,
 capabilities,rules_summary,availability,availability_reason,updated_at
) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13)
ON CONFLICT(id) DO UPDATE SET name=EXCLUDED.name,description=EXCLUDED.description,
 category=EXCLUDED.category,version=EXCLUDED.version,renderer_key=EXCLUDED.renderer_key,
 modes=EXCLUDED.modes,average_time_seconds=EXCLUDED.average_time_seconds,
 capabilities=EXCLUDED.capabilities,rules_summary=EXCLUDED.rules_summary,
 availability=EXCLUDED.availability,availability_reason=EXCLUDED.availability_reason,
 updated_at=EXCLUDED.updated_at`,
			gameEntry.ID, gameEntry.Name, gameEntry.Description, gameEntry.Category,
			gameEntry.Version, gameEntry.RendererKey, modes, gameEntry.AverageTimeSeconds,
			capabilities, rules, gameEntry.Availability, gameEntry.AvailabilityReason, time.Now().UTC(),
		); err != nil {
			return err
		}
	}
	return nil
}

func (s *Store) gameCatalogEntries() []models.GameCatalogEntry {
	if s.arenaRegistry == nil {
		return []models.GameCatalogEntry{}
	}
	metadata := s.arenaRegistry.List()
	entries := make([]models.GameCatalogEntry, 0, len(metadata))
	for _, item := range metadata {
		enabled := s.settings == nil || s.settings.FeatureEnabled(item.ID)
		availability := "available"
		reason := ""
		if !enabled {
			availability = "disabled"
			reason = "This arena is disabled by the current platform configuration."
		}
		entries = append(entries, models.GameCatalogEntry{
			ID:                 item.ID,
			Name:               item.Name,
			Description:        item.Description,
			Category:           item.Category,
			Version:            item.Version,
			RendererKey:        item.RendererKey,
			Modes:              append([]string(nil), item.Modes...),
			AverageTimeSeconds: item.AverageTimeSec,
			Capabilities: models.CapabilityFlags{
				Practice: item.Capabilities.Practice, PvP: item.Capabilities.PvP,
				Replay: item.Capabilities.Replay, Tournament: item.Capabilities.Tournament,
				Spectator: item.Capabilities.Spectator, AI: item.Capabilities.AI,
				Teams: item.Capabilities.Teams, Coins: item.Capabilities.Coins,
			},
			Availability:       availability,
			AvailabilityReason: reason,
			RulesSummary: []string{
				"Submit player intent only; the server validates every action.",
				"Competitive sessions use server-generated deterministic puzzles.",
				"Completed matches are replayable and integrity checked.",
			},
		})
	}
	return entries
}

func (s *Store) ListGameCatalog(ctx context.Context) ([]models.GameCatalogEntry, error) {
	if !s.usesPostgresAuth() {
		return s.gameCatalogEntries(), nil
	}
	if err := s.syncGameCatalog(ctx); err != nil {
		return nil, err
	}
	rows, err := s.pg.QueryContext(ctx, `
SELECT id,name,description,category,version,renderer_key,modes,average_time_seconds,
 capabilities,rules_summary,availability,availability_reason
FROM game_modules ORDER BY name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	entries := []models.GameCatalogEntry{}
	for rows.Next() {
		var entry models.GameCatalogEntry
		var modes, capabilities, rules []byte
		if err := rows.Scan(
			&entry.ID, &entry.Name, &entry.Description, &entry.Category, &entry.Version,
			&entry.RendererKey, &modes, &entry.AverageTimeSeconds, &capabilities,
			&rules, &entry.Availability, &entry.AvailabilityReason,
		); err != nil {
			return nil, err
		}
		if err := json.Unmarshal(modes, &entry.Modes); err != nil {
			return nil, err
		}
		if err := json.Unmarshal(capabilities, &entry.Capabilities); err != nil {
			return nil, err
		}
		if err := json.Unmarshal(rules, &entry.RulesSummary); err != nil {
			return nil, err
		}
		entries = append(entries, entry)
	}
	return entries, rows.Err()
}

func (s *Store) CreateNotification(ctx context.Context, notification *models.Notification) error {
	if notification == nil || notification.UserID == "" || notification.Title == "" || notification.Message == "" {
		return errors.New("notification user, title, and message are required")
	}
	if notification.ID == "" {
		notification.ID = newUUID()
	}
	if notification.Status == "" {
		notification.Status = models.NotificationStatusUnread
	}
	if notification.CreatedAt.IsZero() {
		notification.CreatedAt = time.Now().UTC()
	}
	if notification.Metadata == nil {
		notification.Metadata = map[string]string{}
	}
	if s.usesPostgresAuth() {
		tx, err := s.pg.BeginTx(ctx, nil)
		if err != nil {
			return err
		}
		defer tx.Rollback()
		metadata, _ := json.Marshal(notification.Metadata)
		if _, err = tx.ExecContext(ctx, `
INSERT INTO player_notifications(
 id,user_id,category,title,message,status,action_url,metadata,created_at,read_at,archived_at
) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)`,
			notification.ID, notification.UserID, notification.Category, notification.Title,
			notification.Message, notification.Status, notification.ActionURL, metadata,
			notification.CreatedAt, notification.ReadAt, notification.ArchivedAt,
		); err != nil {
			return err
		}
		payload, _ := json.Marshal(notification)
		if _, err = tx.ExecContext(ctx, `
INSERT INTO notification_events(notification_id,user_id,event_type,payload,created_at)
VALUES($1,$2,$3,$4,$5)`,
			notification.ID, notification.UserID, "notification.created", payload, notification.CreatedAt,
		); err != nil {
			return err
		}
		return tx.Commit()
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	copyNotification := *notification
	copyNotification.Metadata = copyStringMap(notification.Metadata)
	s.notifications[notification.UserID] = append(s.notifications[notification.UserID], &copyNotification)
	return s.persistHubStateLocked()
}

func (s *Store) ListNotifications(ctx context.Context, userID, status string) ([]models.Notification, error) {
	if s.usesPostgresAuth() {
		query := `
SELECT id,user_id,category,title,message,status,action_url,metadata,created_at,read_at,archived_at
FROM player_notifications WHERE user_id=$1`
		args := []any{userID}
		if status != "" {
			query += " AND status=$2"
			args = append(args, status)
		} else {
			query += " AND status <> 'archived'"
		}
		query += " ORDER BY created_at DESC LIMIT 100"
		rows, err := s.pg.QueryContext(ctx, query, args...)
		if err != nil {
			return nil, err
		}
		defer rows.Close()
		notifications := []models.Notification{}
		for rows.Next() {
			var notification models.Notification
			var metadata []byte
			if err := rows.Scan(
				&notification.ID, &notification.UserID, &notification.Category, &notification.Title,
				&notification.Message, &notification.Status, &notification.ActionURL, &metadata,
				&notification.CreatedAt, &notification.ReadAt, &notification.ArchivedAt,
			); err != nil {
				return nil, err
			}
			if err := json.Unmarshal(metadata, &notification.Metadata); err != nil {
				return nil, err
			}
			notifications = append(notifications, notification)
		}
		return notifications, rows.Err()
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	notifications := []models.Notification{}
	for _, item := range s.notifications[userID] {
		if status != "" && item.Status != status {
			continue
		}
		if status == "" && item.Status == models.NotificationStatusArchived {
			continue
		}
		copyNotification := *item
		copyNotification.Metadata = copyStringMap(item.Metadata)
		notifications = append(notifications, copyNotification)
	}
	sort.SliceStable(notifications, func(i, j int) bool {
		return notifications[i].CreatedAt.After(notifications[j].CreatedAt)
	})
	return notifications, nil
}

func (s *Store) UpdateNotificationStatus(ctx context.Context, userID, notificationID, status string) error {
	if status != models.NotificationStatusRead && status != models.NotificationStatusArchived {
		return errors.New("unsupported notification status")
	}
	now := time.Now().UTC()
	if s.usesPostgresAuth() {
		tx, err := s.pg.BeginTx(ctx, nil)
		if err != nil {
			return err
		}
		defer tx.Rollback()
		result, err := tx.ExecContext(ctx, `
UPDATE player_notifications SET status=$3,
 read_at=CASE WHEN $3='read' AND read_at IS NULL THEN $4 ELSE read_at END,
 archived_at=CASE WHEN $3='archived' THEN $4 ELSE archived_at END
WHERE id=$1 AND user_id=$2`, notificationID, userID, status, now)
		if err != nil {
			return err
		}
		affected, err := result.RowsAffected()
		if err != nil {
			return err
		}
		if affected == 0 {
			return errors.New("notification not found")
		}
		payload, _ := json.Marshal(map[string]string{"notificationId": notificationID, "status": status})
		_, err = tx.ExecContext(ctx, `
INSERT INTO notification_events(notification_id,user_id,event_type,payload,created_at)
VALUES($1,$2,$3,$4,$5)`, notificationID, userID, "notification."+status, payload, now)
		if err != nil {
			return err
		}
		return tx.Commit()
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, notification := range s.notifications[userID] {
		if notification.ID != notificationID {
			continue
		}
		notification.Status = status
		if status == models.NotificationStatusRead && notification.ReadAt == nil {
			notification.ReadAt = &now
		}
		if status == models.NotificationStatusArchived {
			notification.ArchivedAt = &now
		}
		return s.persistHubStateLocked()
	}
	return errors.New("notification not found")
}

func (s *Store) CreateSupportTicket(ctx context.Context, ticket *models.SupportTicket) error {
	if ticket == nil || ticket.UserID == "" || ticket.Subject == "" || ticket.Message == "" {
		return errors.New("ticket user, subject, and message are required")
	}
	if ticket.ID == "" {
		ticket.ID = newUUID()
	}
	if ticket.Status == "" {
		ticket.Status = models.TicketStatusReceived
	}
	now := time.Now().UTC()
	ticket.CreatedAt = now
	ticket.UpdatedAt = now
	if s.usesPostgresAuth() {
		_, err := s.pg.ExecContext(ctx, `
INSERT INTO support_tickets(id,user_id,category,subject,message,status,created_at,updated_at)
VALUES($1,$2,$3,$4,$5,$6,$7,$8)`,
			ticket.ID, ticket.UserID, ticket.Category, ticket.Subject, ticket.Message,
			ticket.Status, ticket.CreatedAt, ticket.UpdatedAt,
		)
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	copyTicket := *ticket
	s.supportTickets[ticket.UserID] = append(s.supportTickets[ticket.UserID], &copyTicket)
	return s.persistHubStateLocked()
}

func (s *Store) ListSupportTickets(ctx context.Context, userID string) ([]models.SupportTicket, error) {
	if s.usesPostgresAuth() {
		rows, err := s.pg.QueryContext(ctx, `
SELECT id,user_id,category,subject,message,status,created_at,updated_at
FROM support_tickets WHERE user_id=$1 ORDER BY updated_at DESC LIMIT 100`, userID)
		if err != nil {
			return nil, err
		}
		defer rows.Close()
		tickets := []models.SupportTicket{}
		for rows.Next() {
			var ticket models.SupportTicket
			if err := rows.Scan(
				&ticket.ID, &ticket.UserID, &ticket.Category, &ticket.Subject,
				&ticket.Message, &ticket.Status, &ticket.CreatedAt, &ticket.UpdatedAt,
			); err != nil {
				return nil, err
			}
			tickets = append(tickets, ticket)
		}
		return tickets, rows.Err()
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	tickets := make([]models.SupportTicket, 0, len(s.supportTickets[userID]))
	for _, item := range s.supportTickets[userID] {
		tickets = append(tickets, *item)
	}
	sort.SliceStable(tickets, func(i, j int) bool {
		return tickets[i].UpdatedAt.After(tickets[j].UpdatedAt)
	})
	return tickets, nil
}

func (s *Store) BuildHubSnapshot(ctx context.Context, userID string) (*models.HubSnapshot, error) {
	user, err := s.GetUserByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	profile, err := s.GetPlayerProfile(ctx, userID)
	if err != nil {
		return nil, err
	}
	progression, err := s.GetHubProgression(ctx, userID)
	if err != nil {
		return nil, err
	}
	wallet, err := s.GetWalletByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}
	games, err := s.ListGameCatalog(ctx)
	if err != nil {
		return nil, err
	}
	notifications, err := s.ListNotifications(ctx, userID, "")
	if err != nil {
		return nil, err
	}
	mfa, err := s.GetMFASettings(ctx, userID)
	if err != nil {
		return nil, err
	}

	unread := 0
	for _, notification := range notifications {
		if notification.Status == models.NotificationStatusUnread {
			unread++
		}
	}
	profileComplete := profile.Username != "" && profile.DisplayName != "" && len(profile.Country) == 2 && profile.Language != ""
	blockers := []string{}
	if !user.EmailVerified {
		blockers = append(blockers, "Verify your email before entering live competition.")
	}
	if user.KYCStatus != "approved" {
		blockers = append(blockers, "Identity verification is required for live competition and withdrawals.")
	}
	if user.Status != "active" {
		blockers = append(blockers, "Your account must be active.")
	}
	if !profileComplete {
		blockers = append(blockers, "Complete your competitor profile.")
	}
	eligibility := models.HubEligibility{
		EmailVerified:   user.EmailVerified,
		ProfileComplete: profileComplete,
		MFAEnabled:      mfa.Enabled,
		WalletVisible:   true,
		LiveEligible:    len(blockers) == 0,
		Blockers:        blockers,
	}

	s.mu.RLock()
	sessions := make([]models.GameSession, 0)
	for _, session := range s.sessions {
		if session.UserID == userID {
			sessions = append(sessions, *session)
		}
	}
	pendingDeposits := 0.0
	for _, payment := range s.payments {
		if payment.UserID == userID && payment.Status != models.PaymentStatusSettled && payment.Status != models.PaymentStatusFailed {
			pendingDeposits += payment.Amount
		}
	}
	tournaments := make([]models.Tournament, 0, len(s.tournaments))
	if s.settings == nil || !strings.EqualFold(s.settings.Platform.LaunchPhase, "PRE_LAUNCH") {
		for _, tournament := range s.tournaments {
			tournaments = append(tournaments, *tournament)
		}
	}
	s.mu.RUnlock()
	sort.SliceStable(sessions, func(i, j int) bool { return sessions[i].CreatedAt.After(sessions[j].CreatedAt) })

	activities := make([]models.HubActivity, 0, 6)
	var continueActivity *models.HubActivity
	for i := range sessions {
		session := sessions[i]
		if !session.IsFinished && continueActivity == nil {
			continueActivity = &models.HubActivity{
				ID: session.ID, Type: "game_session", Title: "Continue your active session",
				Description: "Return to the game state held by the server.", ActionURL: "/games",
				OccurredAt: session.CreatedAt,
			}
		}
		if len(activities) < 6 {
			title := "Practice session started"
			description := "The session is still active."
			if session.IsFinished {
				title = "Practice session completed"
				description = fmt.Sprintf("Result: %s", strings.ToLower(session.Outcome))
			}
			activities = append(activities, models.HubActivity{
				ID: session.ID, Type: "game_session", Title: title, Description: description,
				ActionURL: "/replays", OccurredAt: session.CreatedAt,
			})
		}
	}

	objectives := []models.DailyObjective{
		{
			ID: "profile", Title: "Complete your competitor profile",
			Description: "Set your public identity, country, and language.",
			Progress:    boolInt(profileComplete), Target: 1, Complete: profileComplete, ActionURL: "/profile",
		},
		{
			ID: "security", Title: "Secure your account with MFA",
			Description: "Add a second factor and recovery codes.",
			Progress:    boolInt(mfa.Enabled), Target: 1, Complete: mfa.Enabled, ActionURL: "/auth/mfa/setup",
		},
		{
			ID: "practice", Title: "Complete one practice session",
			Description: "Build skill before entering live competition.",
			Progress:    minInt(progression.MatchesPlayed, 1), Target: 1,
			Complete: progression.MatchesPlayed > 0, ActionURL: "/games",
		},
	}

	recommended := models.HubAction{
		ID: "practice", Label: "Enter Practice",
		Description: "Build skill in a server-verified session.", ActionURL: "/games",
		Reason: "Practice creates meaningful progress without requiring a deposit.",
	}
	if !profileComplete {
		recommended = models.HubAction{
			ID: "complete_profile", Label: "Complete Profile",
			Description: "Choose the identity other competitors will see.", ActionURL: "/profile",
			Reason: "A complete profile is required before live competition.",
		}
	} else if continueActivity != nil {
		recommended = models.HubAction{
			ID: "continue_activity", Label: "Continue Activity",
			Description: continueActivity.Description, ActionURL: continueActivity.ActionURL,
			Reason: "You have an unfinished server-held session.",
		}
	} else if progression.MatchesPlayed > 0 && !mfa.Enabled {
		recommended = models.HubAction{
			ID: "secure_account", Label: "Enable MFA",
			Description: "Protect your progression and future wallet access.", ActionURL: "/auth/mfa/setup",
			Reason: "Your account has progress but no second factor.",
		}
	}

	hubTournaments := make([]models.HubTournament, 0, len(tournaments))
	for _, tournament := range tournaments {
		eligible := progression.Level >= tournament.MinimumLevel && progression.TrustScore >= tournament.MinimumTrust && eligibility.LiveEligible
		reason := ""
		if !eligible {
			reason = "Level, Trust Score, or verification requirements are not yet met."
		}
		hubTournaments = append(hubTournaments, models.HubTournament{
			ID: tournament.ID, Name: tournament.Name, Status: tournament.Status,
			StartsAt: tournament.StartsAt, Eligible: eligible, IneligibleReason: reason,
		})
	}
	sort.SliceStable(hubTournaments, func(i, j int) bool { return hubTournaments[i].StartsAt.Before(hubTournaments[j].StartsAt) })

	challenges := []models.HubChallenge{
		{ID: "practice", Type: "practice", Title: "Practice", Status: "available", ActionURL: "/games"},
		{ID: "daily", Type: "daily", Title: "Daily Calibration", Status: "available", ActionURL: "/challenges"},
		{ID: "ranked", Type: "ranked", Title: "Ranked", Status: eligibilityStatus(eligibility.LiveEligible), Reason: firstBlocker(blockers), ActionURL: "/games"},
		{ID: "house", Type: "house", Title: "House Challenge", Status: "locked", Reason: "Join an eligible House before entering a House Challenge."},
		{ID: "tournament", Type: "tournament", Title: "Tournament", Status: tournamentChallengeStatus(hubTournaments), Reason: tournamentChallengeReason(hubTournaments), ActionURL: "/tournaments"},
	}

	return &models.HubSnapshot{
		GeneratedAt: time.Now().UTC(),
		Profile:     *profile,
		Progression: *progression,
		Wallet: models.HubWalletSummary{
			Currency:           "USD",
			AvailableBalance:   maxFloat(0, wallet.LiveBalance-wallet.LiveLockedBalance-wallet.PendingWithdrawals),
			PendingDeposits:    pendingDeposits,
			PendingWithdrawals: wallet.PendingWithdrawals,
			AccountStatus:      user.Status,
			VerificationStatus: user.KYCStatus,
		},
		Notifications:     models.NotificationSummary{Unread: unread, Total: len(notifications)},
		Objectives:        objectives,
		RecommendedAction: recommended,
		ContinueActivity:  continueActivity,
		RecentActivity:    activities,
		Tournaments:       hubTournaments,
		Challenges:        challenges,
		Games:             games,
		Eligibility:       eligibility,
	}, nil
}

func boolInt(value bool) int {
	if value {
		return 1
	}
	return 0
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func maxFloat(a, b float64) float64 {
	if a > b {
		return a
	}
	return b
}

func eligibilityStatus(eligible bool) string {
	if eligible {
		return "available"
	}
	return "locked"
}

func firstBlocker(blockers []string) string {
	if len(blockers) == 0 {
		return ""
	}
	return blockers[0]
}

func tournamentChallengeStatus(tournaments []models.HubTournament) string {
	if len(tournaments) == 0 {
		return "unavailable"
	}
	for _, tournament := range tournaments {
		if tournament.Eligible {
			return "available"
		}
	}
	return "locked"
}

func tournamentChallengeReason(tournaments []models.HubTournament) string {
	if len(tournaments) == 0 {
		return "No tournament is currently accepting players."
	}
	for _, tournament := range tournaments {
		if tournament.Eligible {
			return ""
		}
	}
	return "Current tournaments require additional progression or verification."
}
