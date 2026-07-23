package db

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"skill-arena/internal/arena/core"
	arenaregistry "skill-arena/internal/arena/registry"
	arenasecurity "skill-arena/internal/arena/security"
	"skill-arena/internal/cache"
	"skill-arena/internal/config"
	"skill-arena/internal/game"
	"skill-arena/internal/game/puzzle"
	mazegame "skill-arena/internal/games/maze"
	"skill-arena/internal/id"
	"skill-arena/internal/matchmaking"
	"skill-arena/internal/models"
	saredis "skill-arena/internal/redis"
	"skill-arena/internal/storage"

	_ "github.com/lib/pq"
)

type Store struct {
	mu             sync.RWMutex
	users          map[string]*models.User
	wallets        map[string]*models.Wallet
	sessions       map[string]*models.GameSession
	ledger         map[string][]*models.LedgerEntry
	devices        map[string][]*models.Device
	profiles       map[string]*models.Progression
	awards         map[string][]*models.Achievement
	auth           map[string]*models.AuthSession
	authTokens     map[string]*models.AuthToken
	mfa            map[string]*models.MFASettings
	passwords      map[string][]*models.PasswordHistoryEntry
	loginSecurity  map[string]*models.LoginSecurityState
	audit          []*models.AuditLog
	treasury       *models.TreasuryState
	season         *models.Season
	tournaments    map[string]*models.Tournament
	participants   map[string][]*models.TournamentParticipant
	tMatches       map[string][]*models.TournamentMatch
	tSubmissions   map[string][]*models.TournamentSubmission
	baselines      map[string]*models.BehavioralBaseline
	telemetry      map[string][]*models.GameplayTelemetry
	reviewCases    map[string]*models.ReviewCase
	metrics        *models.MetricsSnapshot
	jobs           map[string]*models.BackgroundJob
	workerHealth   map[string]*models.WorkerHealth
	backups        []*models.BackupRecord
	cache          *cache.Cache
	arenaRegistry  *arenaregistry.Registry
	settings       *config.RuntimeSettings
	pvpMatches     map[string]*models.PvPMatch
	pvpSubmissions map[string][]*models.PvPSubmission
	puzzleRepo     *puzzle.MemoryRepository
	payments       map[string]*models.PaymentProviderSession
	withdrawals    map[string]*models.WithdrawalRequest
	amlReviews     map[string]*models.AMLReview
	playerProfiles map[string]*models.PlayerProfile
	notifications  map[string][]*models.Notification
	supportTickets map[string][]*models.SupportTicket
	dataDir        string
	persistence    string
	pg             *sql.DB
	redis          saredis.Client
	objects        storage.ObjectStore
}

type Options struct {
	DatabaseURL string
	Environment string
	RedisURL    string
	Storage     config.StorageSettings
}

type storeSnapshot struct {
	Users          map[string]*models.User                    `json:"users"`
	Wallets        map[string]*models.Wallet                  `json:"wallets"`
	Sessions       map[string]*models.GameSession             `json:"sessions"`
	Ledger         map[string][]*models.LedgerEntry           `json:"ledger"`
	Devices        map[string][]*models.Device                `json:"devices"`
	Profiles       map[string]*models.Progression             `json:"profiles"`
	Awards         map[string][]*models.Achievement           `json:"awards"`
	Auth           map[string]*models.AuthSession             `json:"auth"`
	AuthTokens     map[string]*models.AuthToken               `json:"authTokens"`
	MFA            map[string]*models.MFASettings             `json:"mfa"`
	Passwords      map[string][]*models.PasswordHistoryEntry  `json:"passwords"`
	LoginSecurity  map[string]*models.LoginSecurityState      `json:"loginSecurity"`
	Audit          []*models.AuditLog                         `json:"audit"`
	Treasury       *models.TreasuryState                      `json:"treasury"`
	Season         *models.Season                             `json:"season"`
	Tournaments    map[string]*models.Tournament              `json:"tournaments"`
	Participants   map[string][]*models.TournamentParticipant `json:"participants"`
	TMatches       map[string][]*models.TournamentMatch       `json:"tournamentMatches"`
	TSubmissions   map[string][]*models.TournamentSubmission  `json:"tournamentSubmissions"`
	Baselines      map[string]*models.BehavioralBaseline      `json:"baselines"`
	Telemetry      map[string][]*models.GameplayTelemetry     `json:"telemetry"`
	ReviewCases    map[string]*models.ReviewCase              `json:"reviewCases"`
	Metrics        *models.MetricsSnapshot                    `json:"metrics"`
	Jobs           map[string]*models.BackgroundJob           `json:"jobs"`
	WorkerHealth   map[string]*models.WorkerHealth            `json:"workerHealth"`
	Backups        []*models.BackupRecord                     `json:"backups"`
	PvPMatches     map[string]*models.PvPMatch                `json:"pvpMatches"`
	PvPSubmissions map[string][]*models.PvPSubmission         `json:"pvpSubmissions"`
	Payments       map[string]*models.PaymentProviderSession  `json:"payments"`
	Withdrawals    map[string]*models.WithdrawalRequest       `json:"withdrawals"`
	AMLReviews     map[string]*models.AMLReview               `json:"amlReviews"`
}

type storedUser struct {
	ID               string    `json:"id"`
	Email            string    `json:"email"`
	Username         string    `json:"username,omitempty"`
	DisplayName      string    `json:"displayName,omitempty"`
	HiddenFromPublic bool      `json:"hiddenFromPublic,omitempty"`
	PasswordHash     string    `json:"passwordHash,omitempty"`
	Role             string    `json:"role"`
	EmailVerified    bool      `json:"emailVerified"`
	KYCStatus        string    `json:"kycStatus"`
	CreatedAt        time.Time `json:"createdAt"`
}

func New(ctx context.Context, dataDir string) (*Store, error) {
	return NewWithOptions(ctx, Options{DatabaseURL: dataDir, Environment: "development"})
}

func NewWithOptions(ctx context.Context, opts Options) (*Store, error) {
	databaseURL := opts.DatabaseURL
	if databaseURL == "" {
		databaseURL = "./data"
	}
	environment := strings.ToLower(strings.TrimSpace(opts.Environment))
	if environment == "" {
		environment = "development"
	}
	persistence := "json"
	if isPostgresURL(databaseURL) {
		persistence = "postgres"
	}
	if environment == "production" && persistence != "postgres" {
		return nil, errors.New("production store requires PostgreSQL database URL")
	}
	dataDir := databaseURL
	if persistence == "postgres" {
		dataDir = "./data"
	}
	if err := os.MkdirAll(dataDir, 0o755); err != nil {
		return nil, err
	}

	redisClient := saredis.Client(saredis.NewMemoryClient())
	if strings.TrimSpace(opts.RedisURL) != "" {
		redisClient = saredis.NetworkClient{URL: opts.RedisURL}
		if err := redisClient.Health(ctx); err != nil {
			if environment == "production" {
				return nil, fmt.Errorf("redis health check failed: %w", err)
			}
			redisClient = saredis.NewMemoryClient()
		}
	} else if environment == "production" {
		return nil, errors.New("SKILL_ARENA_REDIS_URL is required in production")
	}

	objectStore := storage.ObjectStore(storage.LocalStore{Root: opts.Storage.LocalRoot})
	if opts.Storage.LocalRoot == "" {
		objectStore = storage.LocalStore{Root: filepath.Join(dataDir, "objects")}
	}
	if strings.EqualFold(opts.Storage.Provider, "s3") || strings.EqualFold(opts.Storage.Provider, "s3-compatible") {
		objectStore = storage.S3CompatibleStore{
			Endpoint:  opts.Storage.Endpoint,
			Bucket:    opts.Storage.Bucket,
			AccessKey: opts.Storage.AccessKey,
			SecretKey: opts.Storage.SecretKey,
			Region:    opts.Storage.Region,
		}
	}
	if err := objectStore.Health(ctx); err != nil {
		if environment == "production" {
			return nil, fmt.Errorf("object storage health check failed: %w", err)
		}
		objectStore = storage.LocalStore{Root: filepath.Join(dataDir, "objects")}
		if err := objectStore.Health(ctx); err != nil {
			return nil, err
		}
	}

	store := &Store{
		users:          map[string]*models.User{},
		wallets:        map[string]*models.Wallet{},
		sessions:       map[string]*models.GameSession{},
		ledger:         map[string][]*models.LedgerEntry{},
		devices:        map[string][]*models.Device{},
		profiles:       map[string]*models.Progression{},
		awards:         map[string][]*models.Achievement{},
		auth:           map[string]*models.AuthSession{},
		authTokens:     map[string]*models.AuthToken{},
		mfa:            map[string]*models.MFASettings{},
		passwords:      map[string][]*models.PasswordHistoryEntry{},
		loginSecurity:  map[string]*models.LoginSecurityState{},
		audit:          []*models.AuditLog{},
		treasury:       defaultTreasuryState(),
		season:         defaultSeason(),
		tournaments:    map[string]*models.Tournament{},
		participants:   map[string][]*models.TournamentParticipant{},
		tMatches:       map[string][]*models.TournamentMatch{},
		tSubmissions:   map[string][]*models.TournamentSubmission{},
		baselines:      map[string]*models.BehavioralBaseline{},
		telemetry:      map[string][]*models.GameplayTelemetry{},
		reviewCases:    map[string]*models.ReviewCase{},
		metrics:        &models.MetricsSnapshot{},
		jobs:           map[string]*models.BackgroundJob{},
		workerHealth:   map[string]*models.WorkerHealth{},
		backups:        []*models.BackupRecord{},
		cache:          cache.New(),
		arenaRegistry:  arenaregistry.New(mazegame.New()),
		settings:       config.Runtime(),
		pvpMatches:     map[string]*models.PvPMatch{},
		pvpSubmissions: map[string][]*models.PvPSubmission{},
		puzzleRepo:     puzzle.NewMemoryRepository(),
		payments:       map[string]*models.PaymentProviderSession{},
		withdrawals:    map[string]*models.WithdrawalRequest{},
		amlReviews:     map[string]*models.AMLReview{},
		playerProfiles: map[string]*models.PlayerProfile{},
		notifications:  map[string][]*models.Notification{},
		supportTickets: map[string][]*models.SupportTicket{},
		dataDir:        dataDir,
		persistence:    persistence,
		redis:          redisClient,
		objects:        objectStore,
	}

	if persistence == "postgres" {
		pg, err := sql.Open("postgres", databaseURL)
		if err != nil {
			return nil, err
		}
		pg.SetMaxOpenConns(25)
		pg.SetMaxIdleConns(10)
		pg.SetConnMaxLifetime(30 * time.Minute)
		if err := pg.PingContext(ctx); err != nil {
			_ = pg.Close()
			return nil, err
		}
		store.pg = pg
		if err := store.initPostgresPersistence(ctx); err != nil {
			_ = pg.Close()
			return nil, err
		}
		loaded, err := store.loadPostgresSnapshot(ctx)
		if err != nil {
			_ = pg.Close()
			return nil, err
		}
		if !loaded {
			if environment != "production" {
				if err := store.load(); err != nil {
					_ = pg.Close()
					return nil, err
				}
			}
			store.mu.Lock()
			err = store.persistSnapshotLocked(ctx)
			store.mu.Unlock()
			if err != nil {
				_ = pg.Close()
				return nil, err
			}
		}
		if err := store.migrateLegacyIdentitySnapshot(ctx); err != nil {
			_ = pg.Close()
			return nil, fmt.Errorf("migrate identity snapshot: %w", err)
		}
		if err := store.migrateLegacyHubState(ctx); err != nil {
			_ = pg.Close()
			return nil, fmt.Errorf("migrate Arena Hub state: %w", err)
		}
	} else {
		if err := store.load(); err != nil {
			return nil, err
		}
		if err := store.loadHubState(); err != nil {
			return nil, err
		}
	}

	return store, nil
}

func isPostgresURL(value string) bool {
	lower := strings.ToLower(strings.TrimSpace(value))
	return strings.HasPrefix(lower, "postgres://") || strings.HasPrefix(lower, "postgresql://")
}

func cloneArrowLines(lines []models.ArrowLine) []models.ArrowLine {
	copied := make([]models.ArrowLine, len(lines))
	for i, line := range lines {
		copied[i] = line
		copied[i].Points = append([]models.Point(nil), line.Points...)
		copied[i].DependsOn = append([]string(nil), line.DependsOn...)
	}
	return copied
}

func (s *Store) derivePuzzleSeedLocked(purpose, matchID, playerID string, profile models.DifficultyProfile, version models.PuzzleVersion) (game.SeedDerivation, error) {
	settings := s.settings
	if settings == nil {
		settings = config.Runtime()
	}
	return game.DerivePuzzleSeed(settings.Security.PuzzleSecret, game.SeedDerivationInput{
		Purpose:           purpose,
		MatchID:           matchID,
		PlayerID:          playerID,
		DifficultyProfile: profile,
		PuzzleVersion:     version,
	})
}

func (s *Store) puzzleServiceLocked() *puzzle.Service {
	settings := s.settings
	if settings == nil {
		settings = config.Runtime()
	}
	if s.puzzleRepo == nil {
		s.puzzleRepo = puzzle.NewMemoryRepository()
	}
	return puzzle.NewService(settings.Security.PuzzleSecret, s.puzzleRepo)
}

func (s *Store) arenaModuleForSessionLocked(session *models.GameSession) (core.GameModule, error) {
	if s.arenaRegistry == nil {
		s.arenaRegistry = arenaregistry.New(mazegame.New())
	}
	gameID := session.Mode
	if gameID == "" || gameID == "maze" || gameID == "game" || gameID == "calibration" {
		gameID = mazegame.ModuleID
	}
	return s.arenaRegistry.Get(gameID)
}

func (s *Store) load() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if err := s.loadUsers(); err != nil {
		return err
	}
	if err := s.loadWallets(); err != nil {
		return err
	}
	if err := s.loadSessions(); err != nil {
		return err
	}
	if err := s.loadDevices(); err != nil {
		return err
	}
	if err := s.loadLedger(); err != nil {
		return err
	}
	if err := s.loadProgression(); err != nil {
		return err
	}
	if err := s.loadAchievements(); err != nil {
		return err
	}
	if err := s.loadAuthSessions(); err != nil {
		return err
	}
	if err := s.loadAuthHardening(); err != nil {
		return err
	}
	if err := s.loadAuditLogs(); err != nil {
		return err
	}
	if err := s.loadTreasury(); err != nil {
		return err
	}
	if err := s.loadSeason(); err != nil {
		return err
	}
	if err := s.loadTournaments(); err != nil {
		return err
	}
	if err := s.loadTournamentParticipants(); err != nil {
		return err
	}
	if err := s.loadTournamentMatches(); err != nil {
		return err
	}
	if err := s.loadTournamentSubmissions(); err != nil {
		return err
	}
	if err := s.loadBaselines(); err != nil {
		return err
	}
	if err := s.loadTelemetry(); err != nil {
		return err
	}
	if err := s.loadReviewCases(); err != nil {
		return err
	}
	if err := s.loadMetrics(); err != nil {
		return err
	}
	if err := s.loadPvPMatches(); err != nil {
		return err
	}
	if err := s.loadPvPSubmissions(); err != nil {
		return err
	}
	if err := s.loadPaymentState(); err != nil {
		return err
	}
	if err := s.loadJobs(); err != nil {
		return err
	}
	if err := s.loadWorkerHealth(); err != nil {
		return err
	}
	if err := s.loadBackupRecords(); err != nil {
		return err
	}
	return s.validateAndRepairLoadedDataLocked()
}

func (s *Store) validateAndRepairLoadedDataLocked() error {
	seenEmails := map[string]string{}
	changedUsers := false
	for userID, user := range s.users {
		if user == nil {
			return fmt.Errorf("integrity check failed: nil user for id %s", userID)
		}
		if user.ID == "" {
			return errors.New("integrity check failed: user with empty id")
		}
		if user.Email == "" {
			return fmt.Errorf("integrity check failed: user %s has empty email", user.ID)
		}
		emailKey := strings.ToLower(strings.TrimSpace(user.Email))
		if existingID, ok := seenEmails[emailKey]; ok && existingID != user.ID {
			return fmt.Errorf("integrity check failed: duplicate email %s", user.Email)
		}
		seenEmails[emailKey] = user.ID
		if user.Role == "" {
			user.Role = models.RolePlayer
			changedUsers = true
		}
		if user.KYCStatus == "" {
			user.KYCStatus = "unverified"
			changedUsers = true
		}
		if user.CreatedAt.IsZero() {
			user.CreatedAt = time.Now().UTC()
			changedUsers = true
		}
		if s.isConfiguredSuperAdminEmailLocked(user.Email) {
			if user.Role != models.RoleSuperAdmin || !user.HiddenFromPublic {
				user.Role = models.RoleSuperAdmin
				user.HiddenFromPublic = true
				changedUsers = true
			}
		} else if user.PasswordHash == "" {
			return fmt.Errorf("integrity check failed: active user %s has no password hash", user.Email)
		}
		s.ensureProgressionLocked(user.ID)
		if _, ok := s.wallets[user.ID]; !ok {
			s.wallets[user.ID] = &models.Wallet{UserID: user.ID, DemoBalance: 1000}
		}
	}
	for _, email := range s.settings.Admin.SuperAdminEmails {
		emailKey := strings.ToLower(strings.TrimSpace(email))
		if emailKey == "" {
			continue
		}
		if _, ok := seenEmails[emailKey]; ok {
			continue
		}
		user := models.NewUser(id.New("usr"), emailKey, "")
		user.Role = models.RoleSuperAdmin
		user.HiddenFromPublic = true
		s.users[user.ID] = user
		s.ensureProgressionLocked(user.ID)
		s.wallets[user.ID] = &models.Wallet{UserID: user.ID, DemoBalance: 1000}
		seenEmails[emailKey] = user.ID
		changedUsers = true
	}
	for _, authSession := range s.auth {
		if authSession == nil {
			return errors.New("integrity check failed: nil auth session")
		}
		if _, ok := s.users[authSession.UserID]; !ok {
			return fmt.Errorf("integrity check failed: orphaned auth session %s", authSession.ID)
		}
	}
	for _, session := range s.sessions {
		if session == nil {
			return errors.New("integrity check failed: nil game session")
		}
		if _, ok := s.users[session.UserID]; !ok {
			return fmt.Errorf("integrity check failed: orphaned game session %s", session.ID)
		}
	}
	for _, wallet := range s.wallets {
		if wallet == nil {
			return errors.New("integrity check failed: nil wallet")
		}
		if _, ok := s.users[wallet.UserID]; !ok {
			return fmt.Errorf("integrity check failed: orphaned wallet for user %s", wallet.UserID)
		}
	}
	if changedUsers {
		if err := s.persistUsers(); err != nil {
			return err
		}
		if err := s.persistWallets(); err != nil {
			return err
		}
		if err := s.persistProgression(); err != nil {
			return err
		}
	}
	return nil
}

func (s *Store) loadUsers() error {
	path := filepath.Join(s.dataDir, "users.json")
	if _, err := os.Stat(path); err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	content, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	var userList []*storedUser
	if err := json.Unmarshal(content, &userList); err != nil {
		return err
	}

	for _, stored := range userList {
		user := &models.User{
			ID:               stored.ID,
			Email:            stored.Email,
			Username:         stored.Username,
			DisplayName:      stored.DisplayName,
			HiddenFromPublic: stored.HiddenFromPublic,
			PasswordHash:     stored.PasswordHash,
			Role:             stored.Role,
			EmailVerified:    stored.EmailVerified,
			KYCStatus:        stored.KYCStatus,
			CreatedAt:        stored.CreatedAt,
		}
		if user.Role == "" {
			user.Role = "player"
		}
		if s.isConfiguredSuperAdminEmailLocked(user.Email) {
			user.Role = models.RoleSuperAdmin
			user.HiddenFromPublic = true
		}
		s.users[user.ID] = user
	}
	return nil
}

func (s *Store) loadWallets() error {
	path := filepath.Join(s.dataDir, "wallets.json")
	if _, err := os.Stat(path); err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	content, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	var walletList []*models.Wallet
	if err := json.Unmarshal(content, &walletList); err != nil {
		return err
	}

	for _, wallet := range walletList {
		s.wallets[wallet.UserID] = wallet
	}
	return nil
}

func (s *Store) loadSessions() error {
	path := filepath.Join(s.dataDir, "sessions.json")
	if _, err := os.Stat(path); err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	content, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	var sessionList []*models.GameSession
	if err := json.Unmarshal(content, &sessionList); err != nil {
		return err
	}

	for _, session := range sessionList {
		s.sessions[session.ID] = session
	}
	return nil
}

func (s *Store) persistUsers() error {
	path := filepath.Join(s.dataDir, "users.json")
	userList := make([]*storedUser, 0, len(s.users))
	for _, user := range s.users {
		userList = append(userList, &storedUser{
			ID:               user.ID,
			Email:            user.Email,
			Username:         user.Username,
			DisplayName:      user.DisplayName,
			HiddenFromPublic: user.HiddenFromPublic,
			PasswordHash:     user.PasswordHash,
			Role:             user.Role,
			EmailVerified:    user.EmailVerified,
			KYCStatus:        user.KYCStatus,
			CreatedAt:        user.CreatedAt,
		})
	}

	data, err := json.MarshalIndent(userList, "", "  ")
	if err != nil {
		return err
	}
	return s.persistJSONOrSnapshotLocked(path, data)
}

func writeFileAtomic(path string, data []byte, mode os.FileMode) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(dir, filepath.Base(path)+".*.tmp")
	if err != nil {
		return err
	}
	tmpPath := tmp.Name()
	cleanup := true
	defer func() {
		if cleanup {
			_ = os.Remove(tmpPath)
		}
	}()
	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Chmod(mode); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Sync(); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if !json.Valid(data) {
		return fmt.Errorf("refusing to persist invalid json to %s", path)
	}
	backupPath := path + ".bak"
	_ = os.Remove(backupPath)
	hadOriginal := false
	if _, err := os.Stat(path); err == nil {
		hadOriginal = true
		if err := os.Rename(path, backupPath); err != nil {
			return err
		}
	} else if !os.IsNotExist(err) {
		return err
	}
	if err := os.Rename(tmpPath, path); err != nil {
		if hadOriginal {
			_ = os.Rename(backupPath, path)
		}
		return err
	}
	cleanup = false
	if hadOriginal {
		_ = os.Remove(backupPath)
	}
	return nil
}

func writeJSONFile(path string, value any) error {
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	return writeFileAtomic(path, data, 0o644)
}

func loadJSONFile[T any](path string, target *T, apply func(T)) error {
	if _, err := os.Stat(path); err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	content, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	if err := json.Unmarshal(content, target); err != nil {
		return err
	}
	apply(*target)
	return nil
}

func (s *Store) initPostgresPersistence(ctx context.Context) error {
	_, err := s.pg.ExecContext(ctx, `
CREATE TABLE IF NOT EXISTS store_snapshots (
	name TEXT PRIMARY KEY,
	payload JSONB NOT NULL,
	updated_at TIMESTAMPTZ NOT NULL
);
CREATE TABLE IF NOT EXISTS financial_idempotency (
	idempotency_key TEXT PRIMARY KEY,
	user_id TEXT NOT NULL,
	operation TEXT NOT NULL,
	resource_type TEXT NOT NULL,
	resource_id TEXT NOT NULL,
	request_hash TEXT NOT NULL,
	created_at TIMESTAMPTZ NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_financial_idempotency_user_operation ON financial_idempotency(user_id, operation, created_at DESC);
`)
	if err != nil {
		return err
	}
	if err := s.initPostgresAuth(ctx); err != nil {
		return err
	}
	return s.initPostgresHub(ctx)
}

func (s *Store) loadPostgresSnapshot(ctx context.Context) (bool, error) {
	var data []byte
	err := s.pg.QueryRowContext(ctx, `SELECT payload FROM store_snapshots WHERE name = 'primary'`).Scan(&data)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	var snapshot storeSnapshot
	if err := json.Unmarshal(data, &snapshot); err != nil {
		return false, err
	}
	s.applySnapshot(snapshot)
	return true, s.validateAndRepairLoadedDataLocked()
}

func (s *Store) applySnapshot(snapshot storeSnapshot) {
	if snapshot.Users != nil {
		s.users = snapshot.Users
	}
	if snapshot.Wallets != nil {
		s.wallets = snapshot.Wallets
	}
	if snapshot.Sessions != nil {
		s.sessions = snapshot.Sessions
	}
	if snapshot.Ledger != nil {
		s.ledger = snapshot.Ledger
	}
	if snapshot.Devices != nil {
		s.devices = snapshot.Devices
	}
	if snapshot.Profiles != nil {
		s.profiles = snapshot.Profiles
	}
	if snapshot.Awards != nil {
		s.awards = snapshot.Awards
	}
	if snapshot.Auth != nil {
		s.auth = snapshot.Auth
	}
	if snapshot.AuthTokens != nil {
		s.authTokens = snapshot.AuthTokens
	}
	if snapshot.MFA != nil {
		s.mfa = snapshot.MFA
	}
	if snapshot.Passwords != nil {
		s.passwords = snapshot.Passwords
	}
	if snapshot.LoginSecurity != nil {
		s.loginSecurity = snapshot.LoginSecurity
	}
	if snapshot.Audit != nil {
		s.audit = snapshot.Audit
	}
	if snapshot.Treasury != nil {
		s.treasury = snapshot.Treasury
	}
	if snapshot.Season != nil {
		s.season = snapshot.Season
	}
	if snapshot.Tournaments != nil {
		s.tournaments = snapshot.Tournaments
	}
	if snapshot.Participants != nil {
		s.participants = snapshot.Participants
	}
	if snapshot.TMatches != nil {
		s.tMatches = snapshot.TMatches
	}
	if snapshot.TSubmissions != nil {
		s.tSubmissions = snapshot.TSubmissions
	}
	if snapshot.Baselines != nil {
		s.baselines = snapshot.Baselines
	}
	if snapshot.Telemetry != nil {
		s.telemetry = snapshot.Telemetry
	}
	if snapshot.ReviewCases != nil {
		s.reviewCases = snapshot.ReviewCases
	}
	if snapshot.Metrics != nil {
		s.metrics = snapshot.Metrics
	}
	if snapshot.Jobs != nil {
		s.jobs = snapshot.Jobs
	}
	if snapshot.WorkerHealth != nil {
		s.workerHealth = snapshot.WorkerHealth
	}
	if snapshot.Backups != nil {
		s.backups = snapshot.Backups
	}
	if snapshot.PvPMatches != nil {
		s.pvpMatches = snapshot.PvPMatches
	}
	if snapshot.PvPSubmissions != nil {
		s.pvpSubmissions = snapshot.PvPSubmissions
	}
	if snapshot.Payments != nil {
		s.payments = snapshot.Payments
	}
	if snapshot.Withdrawals != nil {
		s.withdrawals = snapshot.Withdrawals
	}
	if snapshot.AMLReviews != nil {
		s.amlReviews = snapshot.AMLReviews
	}
}

func (s *Store) snapshotLocked() storeSnapshot {
	return storeSnapshot{
		Users:          s.users,
		Wallets:        s.wallets,
		Sessions:       s.sessions,
		Ledger:         s.ledger,
		Devices:        s.devices,
		Profiles:       s.profiles,
		Awards:         s.awards,
		Auth:           s.auth,
		AuthTokens:     s.authTokens,
		MFA:            s.mfa,
		Passwords:      s.passwords,
		LoginSecurity:  s.loginSecurity,
		Audit:          s.audit,
		Treasury:       s.treasury,
		Season:         s.season,
		Tournaments:    s.tournaments,
		Participants:   s.participants,
		TMatches:       s.tMatches,
		TSubmissions:   s.tSubmissions,
		Baselines:      s.baselines,
		Telemetry:      s.telemetry,
		ReviewCases:    s.reviewCases,
		Metrics:        s.metrics,
		Jobs:           s.jobs,
		WorkerHealth:   s.workerHealth,
		Backups:        s.backups,
		PvPMatches:     s.pvpMatches,
		PvPSubmissions: s.pvpSubmissions,
		Payments:       s.payments,
		Withdrawals:    s.withdrawals,
		AMLReviews:     s.amlReviews,
	}
}

func (s *Store) persistSnapshotLocked(ctx context.Context) error {
	if s.persistence != "postgres" {
		return nil
	}
	data, err := json.Marshal(s.snapshotLocked())
	if err != nil {
		return err
	}
	_, err = s.pg.ExecContext(ctx, `
INSERT INTO store_snapshots(name, payload, updated_at)
VALUES('primary', $1, $2)
ON CONFLICT(name) DO UPDATE SET payload = EXCLUDED.payload, updated_at = EXCLUDED.updated_at
`, data, time.Now().UTC())
	return err
}

func (s *Store) persistJSONOrSnapshotLocked(path string, data []byte) error {
	if s.persistence == "postgres" {
		return s.persistSnapshotLocked(context.Background())
	}
	return writeFileAtomic(path, data, 0o644)
}

func (s *Store) persistWallets() error {
	path := filepath.Join(s.dataDir, "wallets.json")
	walletList := make([]*models.Wallet, 0, len(s.wallets))
	for _, wallet := range s.wallets {
		walletList = append(walletList, wallet)
	}

	data, err := json.MarshalIndent(walletList, "", "  ")
	if err != nil {
		return err
	}
	return s.persistJSONOrSnapshotLocked(path, data)
}

func (s *Store) persistSessions() error {
	path := filepath.Join(s.dataDir, "sessions.json")
	sessionList := make([]*models.GameSession, 0, len(s.sessions))
	for _, session := range s.sessions {
		sessionList = append(sessionList, session)
	}

	data, err := json.MarshalIndent(sessionList, "", "  ")
	if err != nil {
		return err
	}
	return s.persistJSONOrSnapshotLocked(path, data)
}

func (s *Store) loadDevices() error {
	path := filepath.Join(s.dataDir, "devices.json")
	if _, err := os.Stat(path); err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	content, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	var deviceList []*models.Device
	if err := json.Unmarshal(content, &deviceList); err != nil {
		return err
	}

	for _, device := range deviceList {
		s.devices[device.UserID] = append(s.devices[device.UserID], device)
	}
	return nil
}

func (s *Store) persistDevices() error {
	path := filepath.Join(s.dataDir, "devices.json")
	deviceList := make([]*models.Device, 0)
	for _, devices := range s.devices {
		deviceList = append(deviceList, devices...)
	}

	data, err := json.MarshalIndent(deviceList, "", "  ")
	if err != nil {
		return err
	}
	return s.persistJSONOrSnapshotLocked(path, data)
}

func (s *Store) loadProgression() error {
	path := filepath.Join(s.dataDir, "progression.json")
	if _, err := os.Stat(path); err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	content, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	var progressions []*models.Progression
	if err := json.Unmarshal(content, &progressions); err != nil {
		return err
	}

	for _, progression := range progressions {
		s.profiles[progression.UserID] = progression
	}
	return nil
}

func (s *Store) persistProgression() error {
	if err := s.persistHubProgressionLocked(context.Background()); err != nil {
		return err
	}
	path := filepath.Join(s.dataDir, "progression.json")
	progressions := make([]*models.Progression, 0, len(s.profiles))
	for _, progression := range s.profiles {
		progressions = append(progressions, progression)
	}

	data, err := json.MarshalIndent(progressions, "", "  ")
	if err != nil {
		return err
	}
	return s.persistJSONOrSnapshotLocked(path, data)
}

func (s *Store) loadAchievements() error {
	path := filepath.Join(s.dataDir, "achievements.json")
	if _, err := os.Stat(path); err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	content, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	var achievements []*models.Achievement
	if err := json.Unmarshal(content, &achievements); err != nil {
		return err
	}

	for _, achievement := range achievements {
		s.awards[achievement.UserID] = append(s.awards[achievement.UserID], achievement)
	}
	return nil
}

func (s *Store) persistAchievements() error {
	path := filepath.Join(s.dataDir, "achievements.json")
	achievements := make([]*models.Achievement, 0)
	for _, userAchievements := range s.awards {
		achievements = append(achievements, userAchievements...)
	}

	data, err := json.MarshalIndent(achievements, "", "  ")
	if err != nil {
		return err
	}
	return s.persistJSONOrSnapshotLocked(path, data)
}

func (s *Store) loadAuthSessions() error {
	path := filepath.Join(s.dataDir, "auth_sessions.json")
	if _, err := os.Stat(path); err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	content, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	var sessions []*models.AuthSession
	if err := json.Unmarshal(content, &sessions); err != nil {
		return err
	}

	for _, session := range sessions {
		s.auth[session.ID] = session
	}
	return nil
}

func (s *Store) persistAuthSessions() error {
	path := filepath.Join(s.dataDir, "auth_sessions.json")
	sessions := make([]*models.AuthSession, 0, len(s.auth))
	for _, session := range s.auth {
		sessions = append(sessions, session)
	}

	data, err := json.MarshalIndent(sessions, "", "  ")
	if err != nil {
		return err
	}
	return s.persistJSONOrSnapshotLocked(path, data)
}

func (s *Store) loadAuthHardening() error {
	if err := loadJSONFile(filepath.Join(s.dataDir, "auth_tokens.json"), &[]*models.AuthToken{}, func(tokens []*models.AuthToken) {
		for _, token := range tokens {
			s.authTokens[token.ID] = token
		}
	}); err != nil {
		return err
	}
	if err := loadJSONFile(filepath.Join(s.dataDir, "mfa_settings.json"), &[]*models.MFASettings{}, func(settings []*models.MFASettings) {
		for _, setting := range settings {
			s.mfa[setting.UserID] = setting
		}
	}); err != nil {
		return err
	}
	if err := loadJSONFile(filepath.Join(s.dataDir, "password_history.json"), &[]*models.PasswordHistoryEntry{}, func(entries []*models.PasswordHistoryEntry) {
		for _, entry := range entries {
			s.passwords[entry.UserID] = append(s.passwords[entry.UserID], entry)
		}
	}); err != nil {
		return err
	}
	return loadJSONFile(filepath.Join(s.dataDir, "login_security.json"), &[]*models.LoginSecurityState{}, func(states []*models.LoginSecurityState) {
		for _, state := range states {
			s.loginSecurity[state.UserID] = state
		}
	})
}

func (s *Store) persistAuthHardening() error {
	if s.persistence == "postgres" {
		return s.persistSnapshotLocked(context.Background())
	}
	tokens := make([]*models.AuthToken, 0, len(s.authTokens))
	for _, token := range s.authTokens {
		tokens = append(tokens, token)
	}
	if err := writeJSONFile(filepath.Join(s.dataDir, "auth_tokens.json"), tokens); err != nil {
		return err
	}
	mfaSettings := make([]*models.MFASettings, 0, len(s.mfa))
	for _, setting := range s.mfa {
		mfaSettings = append(mfaSettings, setting)
	}
	if err := writeJSONFile(filepath.Join(s.dataDir, "mfa_settings.json"), mfaSettings); err != nil {
		return err
	}
	passwords := make([]*models.PasswordHistoryEntry, 0)
	for _, entries := range s.passwords {
		passwords = append(passwords, entries...)
	}
	if err := writeJSONFile(filepath.Join(s.dataDir, "password_history.json"), passwords); err != nil {
		return err
	}
	states := make([]*models.LoginSecurityState, 0, len(s.loginSecurity))
	for _, state := range s.loginSecurity {
		states = append(states, state)
	}
	return writeJSONFile(filepath.Join(s.dataDir, "login_security.json"), states)
}

func (s *Store) loadAuditLogs() error {
	path := filepath.Join(s.dataDir, "audit_logs.json")
	if _, err := os.Stat(path); err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	content, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	var logs []*models.AuditLog
	if err := json.Unmarshal(content, &logs); err != nil {
		return err
	}
	s.audit = logs
	return nil
}

func (s *Store) persistAuditLogs() error {
	path := filepath.Join(s.dataDir, "audit_logs.json")
	data, err := json.MarshalIndent(s.audit, "", "  ")
	if err != nil {
		return err
	}
	return s.persistJSONOrSnapshotLocked(path, data)
}

func defaultTreasuryState() *models.TreasuryState {
	return &models.TreasuryState{
		PlayerReserve:       100000,
		RevenueReserve:      25000,
		SeasonReserve:       10000,
		ChampionshipReserve: 10000,
		JackpotReserve:      5000,
		EmergencyReserve:    25000,
		UpdatedAt:           time.Now().UTC(),
	}
}

func (s *Store) loadTreasury() error {
	path := filepath.Join(s.dataDir, "treasury.json")
	if _, err := os.Stat(path); err != nil {
		if os.IsNotExist(err) {
			return s.persistTreasury()
		}
		return err
	}

	content, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	var state models.TreasuryState
	if err := json.Unmarshal(content, &state); err != nil {
		return err
	}
	s.treasury = &state
	return nil
}

func (s *Store) persistTreasury() error {
	path := filepath.Join(s.dataDir, "treasury.json")
	data, err := json.MarshalIndent(s.treasury, "", "  ")
	if err != nil {
		return err
	}
	return s.persistJSONOrSnapshotLocked(path, data)
}

func defaultSeason() *models.Season {
	start := time.Now().UTC().Truncate(24 * time.Hour)
	return &models.Season{
		ID:          "season-1",
		Name:        "Season 1",
		Theme:       "Founding Arena",
		StartsAt:    start,
		EndsAt:      start.Add(90 * 24 * time.Hour),
		IsActive:    true,
		RewardPool:  10000,
		Description: "Founding competitive season for Maze Arena.",
	}
}

func (s *Store) loadSeason() error {
	path := filepath.Join(s.dataDir, "season.json")
	if _, err := os.Stat(path); err != nil {
		if os.IsNotExist(err) {
			return s.persistSeason()
		}
		return err
	}

	content, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	var season models.Season
	if err := json.Unmarshal(content, &season); err != nil {
		return err
	}
	s.season = &season
	return nil
}

func (s *Store) persistSeason() error {
	path := filepath.Join(s.dataDir, "season.json")
	data, err := json.MarshalIndent(s.season, "", "  ")
	if err != nil {
		return err
	}
	return s.persistJSONOrSnapshotLocked(path, data)
}

func defaultTournaments() []*models.Tournament {
	now := time.Now().UTC().Truncate(time.Hour)
	return []*models.Tournament{
		{
			ID:              "daily-maze-open",
			Name:            "Daily Maze Open",
			Type:            "daily",
			Status:          "registration",
			EntryFee:        5,
			WalletType:      "demo",
			PrizePool:       250,
			MinimumLevel:    1,
			MinimumTrust:    70,
			MaxParticipants: 64,
			StartsAt:        now.Add(6 * time.Hour),
			EndsAt:          now.Add(30 * time.Hour),
			CreatedAt:       now,
			Description:     "Daily entry-level Maze Arena tournament.",
		},
		{
			ID:              "weekly-maze-cup",
			Name:            "Weekly Maze Cup",
			Type:            "weekly",
			Status:          "registration",
			EntryFee:        25,
			WalletType:      "demo",
			PrizePool:       1500,
			MinimumLevel:    3,
			MinimumTrust:    75,
			MaxParticipants: 128,
			StartsAt:        now.Add(24 * time.Hour),
			EndsAt:          now.Add(8 * 24 * time.Hour),
			CreatedAt:       now,
			Description:     "Weekly tournament for progressing competitors.",
		},
		{
			ID:              "monthly-championship",
			Name:            "Monthly Championship",
			Type:            "monthly",
			Status:          "registration",
			EntryFee:        100,
			WalletType:      "live",
			PrizePool:       10000,
			MinimumLevel:    10,
			MinimumTrust:    85,
			MaxParticipants: 256,
			StartsAt:        now.Add(7 * 24 * time.Hour),
			EndsAt:          now.Add(37 * 24 * time.Hour),
			CreatedAt:       now,
			Description:     "Treasury-backed monthly live championship.",
		},
	}
}

func (s *Store) loadTournaments() error {
	path := filepath.Join(s.dataDir, "tournaments.json")
	if _, err := os.Stat(path); err != nil {
		if os.IsNotExist(err) {
			for _, tournament := range defaultTournaments() {
				s.tournaments[tournament.ID] = tournament
			}
			return s.persistTournaments()
		}
		return err
	}

	content, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	var tournaments []*models.Tournament
	if err := json.Unmarshal(content, &tournaments); err != nil {
		return err
	}
	for _, tournament := range tournaments {
		s.tournaments[tournament.ID] = tournament
	}
	return nil
}

func (s *Store) persistTournaments() error {
	path := filepath.Join(s.dataDir, "tournaments.json")
	tournaments := make([]*models.Tournament, 0, len(s.tournaments))
	for _, tournament := range s.tournaments {
		tournaments = append(tournaments, tournament)
	}
	data, err := json.MarshalIndent(tournaments, "", "  ")
	if err != nil {
		return err
	}
	return s.persistJSONOrSnapshotLocked(path, data)
}

func (s *Store) loadTournamentParticipants() error {
	path := filepath.Join(s.dataDir, "tournament_participants.json")
	if _, err := os.Stat(path); err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	content, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	var participants []*models.TournamentParticipant
	if err := json.Unmarshal(content, &participants); err != nil {
		return err
	}
	for _, participant := range participants {
		s.participants[participant.TournamentID] = append(s.participants[participant.TournamentID], participant)
	}
	return nil
}

func (s *Store) persistTournamentParticipants() error {
	path := filepath.Join(s.dataDir, "tournament_participants.json")
	participants := make([]*models.TournamentParticipant, 0)
	for _, tournamentParticipants := range s.participants {
		participants = append(participants, tournamentParticipants...)
	}
	data, err := json.MarshalIndent(participants, "", "  ")
	if err != nil {
		return err
	}
	return s.persistJSONOrSnapshotLocked(path, data)
}

func (s *Store) loadTournamentMatches() error {
	path := filepath.Join(s.dataDir, "tournament_matches.json")
	if _, err := os.Stat(path); err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	content, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	var matches []*models.TournamentMatch
	if err := json.Unmarshal(content, &matches); err != nil {
		return err
	}
	for _, match := range matches {
		s.tMatches[match.TournamentID] = append(s.tMatches[match.TournamentID], match)
	}
	return nil
}

func (s *Store) persistTournamentMatches() error {
	path := filepath.Join(s.dataDir, "tournament_matches.json")
	matches := make([]*models.TournamentMatch, 0)
	for _, tournamentMatches := range s.tMatches {
		matches = append(matches, tournamentMatches...)
	}
	data, err := json.MarshalIndent(matches, "", "  ")
	if err != nil {
		return err
	}
	return s.persistJSONOrSnapshotLocked(path, data)
}

func (s *Store) loadTournamentSubmissions() error {
	path := filepath.Join(s.dataDir, "tournament_submissions.json")
	if _, err := os.Stat(path); err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	content, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	var submissions []*models.TournamentSubmission
	if err := json.Unmarshal(content, &submissions); err != nil {
		return err
	}
	for _, submission := range submissions {
		s.tSubmissions[submission.MatchID] = append(s.tSubmissions[submission.MatchID], submission)
	}
	return nil
}

func (s *Store) persistTournamentSubmissions() error {
	path := filepath.Join(s.dataDir, "tournament_submissions.json")
	submissions := make([]*models.TournamentSubmission, 0)
	for _, matchSubmissions := range s.tSubmissions {
		submissions = append(submissions, matchSubmissions...)
	}
	data, err := json.MarshalIndent(submissions, "", "  ")
	if err != nil {
		return err
	}
	return s.persistJSONOrSnapshotLocked(path, data)
}

func (s *Store) loadBaselines() error {
	path := filepath.Join(s.dataDir, "baselines.json")
	if _, err := os.Stat(path); err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	content, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	var baselines []*models.BehavioralBaseline
	if err := json.Unmarshal(content, &baselines); err != nil {
		return err
	}
	for _, baseline := range baselines {
		s.baselines[baseline.UserID] = baseline
	}
	return nil
}

func (s *Store) persistBaselines() error {
	path := filepath.Join(s.dataDir, "baselines.json")
	baselines := make([]*models.BehavioralBaseline, 0, len(s.baselines))
	for _, baseline := range s.baselines {
		baselines = append(baselines, baseline)
	}
	data, err := json.MarshalIndent(baselines, "", "  ")
	if err != nil {
		return err
	}
	return s.persistJSONOrSnapshotLocked(path, data)
}

func (s *Store) loadTelemetry() error {
	path := filepath.Join(s.dataDir, "telemetry.json")
	if _, err := os.Stat(path); err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	content, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	var entries []*models.GameplayTelemetry
	if err := json.Unmarshal(content, &entries); err != nil {
		return err
	}
	for _, entry := range entries {
		s.telemetry[entry.ScopeID] = append(s.telemetry[entry.ScopeID], entry)
	}
	return nil
}

func (s *Store) persistTelemetry() error {
	path := filepath.Join(s.dataDir, "telemetry.json")
	entries := make([]*models.GameplayTelemetry, 0)
	for _, scopeEntries := range s.telemetry {
		entries = append(entries, scopeEntries...)
	}
	data, err := json.MarshalIndent(entries, "", "  ")
	if err != nil {
		return err
	}
	return s.persistJSONOrSnapshotLocked(path, data)
}

func (s *Store) loadReviewCases() error {
	path := filepath.Join(s.dataDir, "review_cases.json")
	if _, err := os.Stat(path); err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	content, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	var cases []*models.ReviewCase
	if err := json.Unmarshal(content, &cases); err != nil {
		return err
	}
	for _, reviewCase := range cases {
		s.reviewCases[reviewCase.Scope+":"+reviewCase.ScopeID] = reviewCase
	}
	return nil
}

func (s *Store) persistReviewCases() error {
	path := filepath.Join(s.dataDir, "review_cases.json")
	cases := make([]*models.ReviewCase, 0, len(s.reviewCases))
	for _, reviewCase := range s.reviewCases {
		cases = append(cases, reviewCase)
	}
	data, err := json.MarshalIndent(cases, "", "  ")
	if err != nil {
		return err
	}
	return s.persistJSONOrSnapshotLocked(path, data)
}

func (s *Store) loadMetrics() error {
	path := filepath.Join(s.dataDir, "metrics.json")
	if _, err := os.Stat(path); err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	content, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	var metrics models.MetricsSnapshot
	if err := json.Unmarshal(content, &metrics); err != nil {
		return err
	}
	s.metrics = &metrics
	return nil
}

func (s *Store) persistMetrics() error {
	path := filepath.Join(s.dataDir, "metrics.json")
	data, err := json.MarshalIndent(s.metrics, "", "  ")
	if err != nil {
		return err
	}
	return s.persistJSONOrSnapshotLocked(path, data)
}

func (s *Store) loadPvPMatches() error {
	path := filepath.Join(s.dataDir, "pvp_matches.json")
	if _, err := os.Stat(path); err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	content, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	var matches []*models.PvPMatch
	if err := json.Unmarshal(content, &matches); err != nil {
		return err
	}
	for _, match := range matches {
		s.pvpMatches[match.ID] = match
	}
	return nil
}

func (s *Store) persistPvPMatches() error {
	path := filepath.Join(s.dataDir, "pvp_matches.json")
	matches := make([]*models.PvPMatch, 0, len(s.pvpMatches))
	for _, match := range s.pvpMatches {
		matches = append(matches, match)
	}
	data, err := json.MarshalIndent(matches, "", "  ")
	if err != nil {
		return err
	}
	return s.persistJSONOrSnapshotLocked(path, data)
}

func (s *Store) loadPvPSubmissions() error {
	path := filepath.Join(s.dataDir, "pvp_submissions.json")
	if _, err := os.Stat(path); err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	content, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	var submissions []*models.PvPSubmission
	if err := json.Unmarshal(content, &submissions); err != nil {
		return err
	}
	for _, submission := range submissions {
		s.pvpSubmissions[submission.MatchID] = append(s.pvpSubmissions[submission.MatchID], submission)
	}
	return nil
}

func (s *Store) persistPvPSubmissions() error {
	path := filepath.Join(s.dataDir, "pvp_submissions.json")
	submissions := make([]*models.PvPSubmission, 0)
	for _, matchSubmissions := range s.pvpSubmissions {
		submissions = append(submissions, matchSubmissions...)
	}
	data, err := json.MarshalIndent(submissions, "", "  ")
	if err != nil {
		return err
	}
	return s.persistJSONOrSnapshotLocked(path, data)
}

func (s *Store) loadJobs() error {
	path := filepath.Join(s.dataDir, "jobs.json")
	if _, err := os.Stat(path); err != nil {
		if os.IsNotExist(err) {
			return s.persistJobs()
		}
		return err
	}
	content, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	var jobs []*models.BackgroundJob
	if err := json.Unmarshal(content, &jobs); err != nil {
		return err
	}
	for _, job := range jobs {
		if job != nil {
			s.jobs[job.ID] = job
		}
	}
	return nil
}

func (s *Store) persistJobs() error {
	path := filepath.Join(s.dataDir, "jobs.json")
	jobs := make([]*models.BackgroundJob, 0, len(s.jobs))
	for _, job := range s.jobs {
		jobs = append(jobs, job)
	}
	sort.SliceStable(jobs, func(i, j int) bool {
		return jobs[i].CreatedAt.Before(jobs[j].CreatedAt)
	})
	data, err := json.MarshalIndent(jobs, "", "  ")
	if err != nil {
		return err
	}
	return s.persistJSONOrSnapshotLocked(path, data)
}

func (s *Store) loadWorkerHealth() error {
	path := filepath.Join(s.dataDir, "worker_health.json")
	if _, err := os.Stat(path); err != nil {
		if os.IsNotExist(err) {
			return s.persistWorkerHealth()
		}
		return err
	}
	content, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	var workers []*models.WorkerHealth
	if err := json.Unmarshal(content, &workers); err != nil {
		return err
	}
	for _, worker := range workers {
		if worker != nil {
			s.workerHealth[worker.Name] = worker
		}
	}
	return nil
}

func (s *Store) persistWorkerHealth() error {
	path := filepath.Join(s.dataDir, "worker_health.json")
	workers := make([]*models.WorkerHealth, 0, len(s.workerHealth))
	for _, worker := range s.workerHealth {
		workers = append(workers, worker)
	}
	sort.SliceStable(workers, func(i, j int) bool {
		return workers[i].Name < workers[j].Name
	})
	data, err := json.MarshalIndent(workers, "", "  ")
	if err != nil {
		return err
	}
	return s.persistJSONOrSnapshotLocked(path, data)
}

func (s *Store) loadBackupRecords() error {
	path := filepath.Join(s.dataDir, "backups.json")
	if _, err := os.Stat(path); err != nil {
		if os.IsNotExist(err) {
			return s.persistBackupRecords()
		}
		return err
	}
	content, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	var backups []*models.BackupRecord
	if err := json.Unmarshal(content, &backups); err != nil {
		return err
	}
	s.backups = backups
	return nil
}

func (s *Store) persistBackupRecords() error {
	path := filepath.Join(s.dataDir, "backups.json")
	data, err := json.MarshalIndent(s.backups, "", "  ")
	if err != nil {
		return err
	}
	return s.persistJSONOrSnapshotLocked(path, data)
}

func (s *Store) loadLedger() error {
	path := filepath.Join(s.dataDir, "ledger.json")
	if _, err := os.Stat(path); err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	content, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	var entries []*models.LedgerEntry
	if err := json.Unmarshal(content, &entries); err != nil {
		return err
	}

	for _, entry := range entries {
		s.ledger[entry.UserID] = append(s.ledger[entry.UserID], entry)
	}
	return nil
}

func (s *Store) loadPaymentState() error {
	if err := loadJSONFile(filepath.Join(s.dataDir, "payment_sessions.json"), &[]*models.PaymentProviderSession{}, func(payments []*models.PaymentProviderSession) {
		for _, payment := range payments {
			s.payments[payment.ID] = payment
		}
	}); err != nil {
		return err
	}
	if err := loadJSONFile(filepath.Join(s.dataDir, "withdrawal_requests.json"), &[]*models.WithdrawalRequest{}, func(withdrawals []*models.WithdrawalRequest) {
		for _, withdrawal := range withdrawals {
			s.withdrawals[withdrawal.ID] = withdrawal
		}
	}); err != nil {
		return err
	}
	return loadJSONFile(filepath.Join(s.dataDir, "aml_reviews.json"), &[]*models.AMLReview{}, func(reviews []*models.AMLReview) {
		for _, review := range reviews {
			s.amlReviews[review.ID] = review
		}
	})
}

func (s *Store) persistPaymentState() error {
	if s.persistence == "postgres" {
		return s.persistSnapshotLocked(context.Background())
	}
	payments := make([]*models.PaymentProviderSession, 0, len(s.payments))
	for _, payment := range s.payments {
		payments = append(payments, payment)
	}
	if err := writeJSONFile(filepath.Join(s.dataDir, "payment_sessions.json"), payments); err != nil {
		return err
	}
	withdrawals := make([]*models.WithdrawalRequest, 0, len(s.withdrawals))
	for _, withdrawal := range s.withdrawals {
		withdrawals = append(withdrawals, withdrawal)
	}
	if err := writeJSONFile(filepath.Join(s.dataDir, "withdrawal_requests.json"), withdrawals); err != nil {
		return err
	}
	reviews := make([]*models.AMLReview, 0, len(s.amlReviews))
	for _, review := range s.amlReviews {
		reviews = append(reviews, review)
	}
	return writeJSONFile(filepath.Join(s.dataDir, "aml_reviews.json"), reviews)
}

func calculateAvailableLive(wallet *models.Wallet) float64 {
	available := wallet.LiveBalance - wallet.LiveLockedBalance - wallet.PendingWithdrawals
	if available < 0 {
		return 0
	}
	return available
}

func calculateAvailableDemo(wallet *models.Wallet) float64 {
	available := wallet.DemoBalance - wallet.DemoLockedBalance
	if available < 0 {
		return 0
	}
	return available
}

func requireFinancialIdempotency(metadata map[string]string) (string, string, error) {
	if metadata == nil {
		return "", "", errors.New("financial operation requires idempotency metadata")
	}
	key := strings.TrimSpace(metadata["idempotencyKey"])
	requestHash := strings.TrimSpace(metadata["requestHash"])
	if key == "" {
		return "", "", errors.New("Idempotency-Key is required for financial operations")
	}
	if requestHash == "" {
		return "", "", errors.New("financial operation request hash is required")
	}
	return key, requestHash, nil
}

func (s *Store) findDepositByIdempotencyLocked(userID, key string) *models.PaymentProviderSession {
	for _, payment := range s.payments {
		if payment.UserID == userID && payment.IdempotencyKey == key {
			return payment
		}
		if payment.UserID == userID && payment.Metadata != nil && payment.Metadata["idempotencyKey"] == key {
			return payment
		}
	}
	return nil
}

func (s *Store) findWithdrawalByIdempotencyLocked(userID, key string) *models.WithdrawalRequest {
	for _, withdrawal := range s.withdrawals {
		if withdrawal.UserID == userID && withdrawal.Metadata != nil && withdrawal.Metadata["idempotencyKey"] == key {
			return withdrawal
		}
	}
	return nil
}

func (s *Store) findAMLReviewForScopeLocked(scope, scopeID string) *models.AMLReview {
	for _, review := range s.amlReviews {
		if review.Scope == scope && review.ScopeID == scopeID {
			return review
		}
	}
	return nil
}

func (s *Store) recordFinancialIdempotencyLocked(ctx context.Context, key, userID, operation, resourceType, resourceID, requestHash string) error {
	if s.persistence != "postgres" || s.pg == nil {
		return nil
	}
	_, err := s.pg.ExecContext(ctx, `
INSERT INTO financial_idempotency(idempotency_key, user_id, operation, resource_type, resource_id, request_hash, created_at)
VALUES($1, $2, $3, $4, $5, $6, $7)
ON CONFLICT(idempotency_key) DO NOTHING
`, key, userID, operation, resourceType, resourceID, requestHash, time.Now().UTC())
	return err
}

func (s *Store) LockWalletTokens(ctx context.Context, userID string, walletType string, amount float64, currency string, reference string, metadata map[string]string) (*models.LedgerEntry, error) {
	if amount <= 0 {
		return nil, errors.New("amount must be greater than zero")
	}
	if currency == "" {
		currency = "USD"
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	wallet, ok := s.wallets[userID]
	if !ok {
		return nil, errors.New("wallet not found")
	}

	var available float64
	entry := &models.LedgerEntry{
		ID:              newUUID(),
		UserID:          userID,
		TransactionType: models.TransactionTypeLock,
		Amount:          amount,
		Currency:        currency,
		Reference:       reference,
		Metadata:        metadata,
		CreatedAt:       time.Now().UTC(),
	}

	switch walletType {
	case "live":
		available = calculateAvailableLive(wallet)
		if available < amount {
			return nil, errors.New("insufficient available live balance")
		}
		entry.BalanceBefore = available
		wallet.LiveLockedBalance += amount
		entry.BalanceAfter = calculateAvailableLive(wallet)
	case "demo":
		available = calculateAvailableDemo(wallet)
		if available < amount {
			return nil, errors.New("insufficient available demo balance")
		}
		entry.BalanceBefore = available
		wallet.DemoLockedBalance += amount
		entry.BalanceAfter = calculateAvailableDemo(wallet)
	default:
		return nil, errors.New("unsupported wallet type")
	}

	s.ledger[userID] = append(s.ledger[userID], entry)
	if err := s.persistWallets(); err != nil {
		return nil, err
	}
	if err := s.persistLedger(); err != nil {
		return nil, err
	}

	return entry, nil
}

func (s *Store) UnlockWalletTokens(ctx context.Context, userID string, walletType string, amount float64, currency string, reference string, metadata map[string]string) (*models.LedgerEntry, error) {
	if amount <= 0 {
		return nil, errors.New("amount must be greater than zero")
	}
	if currency == "" {
		currency = "USD"
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	wallet, ok := s.wallets[userID]
	if !ok {
		return nil, errors.New("wallet not found")
	}

	var availableBefore float64
	entry := &models.LedgerEntry{
		ID:              newUUID(),
		UserID:          userID,
		TransactionType: models.TransactionTypeUnlock,
		Amount:          amount,
		Currency:        currency,
		Reference:       reference,
		Metadata:        metadata,
		CreatedAt:       time.Now().UTC(),
	}

	switch walletType {
	case "live":
		if wallet.LiveLockedBalance < amount {
			return nil, errors.New("insufficient locked live balance")
		}
		availableBefore = calculateAvailableLive(wallet)
		wallet.LiveLockedBalance -= amount
		entry.BalanceBefore = availableBefore
		entry.BalanceAfter = calculateAvailableLive(wallet)
	case "demo":
		if wallet.DemoLockedBalance < amount {
			return nil, errors.New("insufficient locked demo balance")
		}
		availableBefore = calculateAvailableDemo(wallet)
		wallet.DemoLockedBalance -= amount
		entry.BalanceBefore = availableBefore
		entry.BalanceAfter = calculateAvailableDemo(wallet)
	default:
		return nil, errors.New("unsupported wallet type")
	}

	s.ledger[userID] = append(s.ledger[userID], entry)
	if err := s.persistWallets(); err != nil {
		return nil, err
	}
	if err := s.persistLedger(); err != nil {
		return nil, err
	}

	return entry, nil
}

func (s *Store) ProcessWithdrawal(ctx context.Context, userID string, amount float64, currency string, reference string, metadata map[string]string) ([]*models.LedgerEntry, error) {
	if amount <= 0 {
		return nil, errors.New("amount must be greater than zero")
	}
	if currency == "" {
		currency = "USD"
	}

	fee := amount * 0.01
	totalDebit := amount + fee

	s.mu.Lock()
	defer s.mu.Unlock()

	wallet, ok := s.wallets[userID]
	if !ok {
		return nil, errors.New("wallet not found")
	}
	progression := s.ensureProgressionLocked(userID)
	limit := withdrawalLimitForTrust(progression.TrustTier)
	if amount > limit {
		return nil, fmt.Errorf("withdrawal exceeds trust tier limit %.2f", limit)
	}

	available := calculateAvailableLive(wallet)
	if available < totalDebit {
		return nil, errors.New("insufficient available live balance")
	}

	withdrawEntry := &models.LedgerEntry{
		ID:              newUUID(),
		UserID:          userID,
		TransactionType: models.TransactionTypeWithdraw,
		Amount:          -amount,
		BalanceBefore:   wallet.LiveBalance,
		Currency:        currency,
		Reference:       reference,
		Metadata:        metadata,
		CreatedAt:       time.Now().UTC(),
	}
	wallet.LiveBalance -= amount
	withdrawEntry.BalanceAfter = wallet.LiveBalance

	feeEntry := &models.LedgerEntry{
		ID:              newUUID(),
		UserID:          userID,
		TransactionType: models.TransactionTypeFee,
		Amount:          -fee,
		BalanceBefore:   wallet.LiveBalance,
		BalanceAfter:    wallet.LiveBalance - fee,
		Currency:        currency,
		Reference:       "withdrawal-fee",
		Metadata:        metadata,
		CreatedAt:       time.Now().UTC(),
	}
	wallet.LiveBalance -= fee

	s.ledger[userID] = append(s.ledger[userID], withdrawEntry, feeEntry)
	s.recomputeTrustLocked(userID, "withdrawal")
	s.audit = append(s.audit, &models.AuditLog{
		ID:        newUUID(),
		ActorID:   userID,
		Action:    "wallet.withdrawal.created",
		TargetID:  withdrawEntry.ID,
		Metadata:  map[string]string{"amount": fmt.Sprintf("%.2f", amount), "trustTier": progression.TrustTier},
		CreatedAt: time.Now().UTC(),
	})
	if err := s.persistWallets(); err != nil {
		return nil, err
	}
	if err := s.persistLedger(); err != nil {
		return nil, err
	}
	if err := s.persistProgression(); err != nil {
		return nil, err
	}
	if err := s.persistAuditLogs(); err != nil {
		return nil, err
	}

	return []*models.LedgerEntry{withdrawEntry, feeEntry}, nil
}

func (s *Store) CreateDepositSession(ctx context.Context, userID, provider, method string, amount float64, currency, reference string, metadata map[string]string) (*models.PaymentProviderSession, error) {
	if amount <= 0 {
		return nil, errors.New("amount must be greater than zero")
	}
	idempotencyKey, requestHash, err := requireFinancialIdempotency(metadata)
	if err != nil {
		return nil, err
	}
	if currency == "" {
		currency = "USD"
	}
	if provider == "" {
		provider = "manual"
	}
	if method == "" {
		method = provider
	}
	now := time.Now().UTC()
	session := &models.PaymentProviderSession{
		ID:             newUUID(),
		UserID:         userID,
		Provider:       provider,
		Method:         method,
		Amount:         amount,
		Currency:       currency,
		Status:         models.PaymentStatusProviderSession,
		ProviderRef:    reference,
		IdempotencyKey: newUUID(),
		Metadata:       metadata,
		CreatedAt:      now,
		UpdatedAt:      now,
	}
	if session.CheckoutURL == "" {
		session.CheckoutURL = fmt.Sprintf("/api/v1/wallet/deposit/%s/provider-session", session.ID)
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.wallets[userID]; !ok {
		return nil, errors.New("wallet not found")
	}
	if existing := s.findDepositByIdempotencyLocked(userID, idempotencyKey); existing != nil {
		if existing.Metadata != nil && existing.Metadata["requestHash"] != requestHash {
			return nil, errors.New("idempotency key reused with different deposit request")
		}
		return existing, nil
	}
	session.IdempotencyKey = idempotencyKey
	s.payments[session.ID] = session
	if err := s.recordFinancialIdempotencyLocked(ctx, idempotencyKey, userID, "deposit", "payment_session", session.ID, requestHash); err != nil {
		return nil, err
	}
	s.audit = append(s.audit, &models.AuditLog{
		ID:        newUUID(),
		ActorID:   userID,
		Action:    "wallet.deposit.provider_session.created",
		TargetID:  session.ID,
		Metadata:  map[string]string{"provider": provider, "amount": fmt.Sprintf("%.2f", amount)},
		CreatedAt: now,
	})
	if err := s.persistPaymentState(); err != nil {
		return nil, err
	}
	if err := s.persistAuditLogs(); err != nil {
		return nil, err
	}

	return session, nil
}

func (s *Store) MarkDepositPending(ctx context.Context, sessionID, providerRef string, metadata map[string]string) (*models.PaymentProviderSession, error) {
	now := time.Now().UTC()
	s.mu.Lock()
	defer s.mu.Unlock()
	session := s.payments[sessionID]
	if session == nil {
		return nil, errors.New("payment session not found")
	}
	if session.Status == models.PaymentStatusSettled {
		return session, nil
	}
	session.Status = models.PaymentStatusPending
	if providerRef != "" {
		session.ProviderRef = providerRef
	}
	if session.Metadata == nil {
		session.Metadata = map[string]string{}
	}
	for key, value := range metadata {
		session.Metadata[key] = value
	}
	session.UpdatedAt = now
	s.audit = append(s.audit, &models.AuditLog{
		ID:        newUUID(),
		ActorID:   "provider",
		Action:    "wallet.deposit.pending",
		TargetID:  session.ID,
		CreatedAt: now,
	})
	if err := s.persistPaymentState(); err != nil {
		return nil, err
	}
	if err := s.persistAuditLogs(); err != nil {
		return nil, err
	}
	return session, nil
}

func (s *Store) SettleDeposit(ctx context.Context, sessionID, providerRef string, metadata map[string]string) (*models.LedgerEntry, error) {
	now := time.Now().UTC()
	s.mu.Lock()
	defer s.mu.Unlock()
	session := s.payments[sessionID]
	if session == nil {
		return nil, errors.New("payment session not found")
	}
	if session.Status == models.PaymentStatusSettled {
		return nil, errors.New("payment session already settled")
	}
	wallet := s.wallets[session.UserID]
	if wallet == nil {
		return nil, errors.New("wallet not found")
	}
	if providerRef != "" {
		session.ProviderRef = providerRef
	}
	session.Status = models.PaymentStatusSettled
	session.UpdatedAt = now
	session.SettledAt = &now
	if session.Metadata == nil {
		session.Metadata = map[string]string{}
	}
	for key, value := range metadata {
		session.Metadata[key] = value
	}
	entry := &models.LedgerEntry{
		ID:              newUUID(),
		UserID:          session.UserID,
		TransactionType: models.TransactionTypeDeposit,
		Amount:          session.Amount,
		BalanceBefore:   wallet.LiveBalance,
		Currency:        session.Currency,
		Reference:       session.ID,
		Metadata:        map[string]string{"provider": session.Provider, "providerRef": session.ProviderRef},
		CreatedAt:       now,
	}
	wallet.LiveBalance += session.Amount
	entry.BalanceAfter = wallet.LiveBalance
	s.ledger[session.UserID] = append(s.ledger[session.UserID], entry)
	s.audit = append(s.audit, &models.AuditLog{
		ID:        newUUID(),
		ActorID:   "provider",
		Action:    "wallet.deposit.settled",
		TargetID:  session.ID,
		CreatedAt: now,
	})
	if err := s.persistWallets(); err != nil {
		return nil, err
	}
	if err := s.persistLedger(); err != nil {
		return nil, err
	}
	if err := s.persistPaymentState(); err != nil {
		return nil, err
	}
	if err := s.persistAuditLogs(); err != nil {
		return nil, err
	}
	return entry, nil
}

func (s *Store) CreateWithdrawalRequest(ctx context.Context, userID, provider, method string, amount float64, currency, reference string, metadata map[string]string) (*models.WithdrawalRequest, *models.AMLReview, error) {
	if amount <= 0 {
		return nil, nil, errors.New("amount must be greater than zero")
	}
	idempotencyKey, requestHash, err := requireFinancialIdempotency(metadata)
	if err != nil {
		return nil, nil, err
	}
	if currency == "" {
		currency = "USD"
	}
	if provider == "" {
		provider = "manual"
	}
	if method == "" {
		method = provider
	}
	now := time.Now().UTC()
	fee := amount * 0.01
	totalHold := amount + fee

	s.mu.Lock()
	defer s.mu.Unlock()
	wallet := s.wallets[userID]
	if wallet == nil {
		return nil, nil, errors.New("wallet not found")
	}
	if existing := s.findWithdrawalByIdempotencyLocked(userID, idempotencyKey); existing != nil {
		if existing.Metadata != nil && existing.Metadata["requestHash"] != requestHash {
			return nil, nil, errors.New("idempotency key reused with different withdrawal request")
		}
		return existing, s.findAMLReviewForScopeLocked("withdrawal", existing.ID), nil
	}
	progression := s.ensureProgressionLocked(userID)
	limit := withdrawalLimitForTrust(progression.TrustTier)
	if amount > limit {
		return nil, nil, fmt.Errorf("withdrawal exceeds trust tier limit %.2f", limit)
	}
	if calculateAvailableLive(wallet) < totalHold {
		return nil, nil, errors.New("insufficient available live balance")
	}
	riskScore := 0
	reasons := []string{}
	if amount >= 1000 {
		riskScore += 40
		reasons = append(reasons, "large_withdrawal")
	}
	recentCount := 0
	for _, withdrawal := range s.withdrawals {
		if withdrawal.UserID == userID && withdrawal.RequestedAt.After(now.Add(-24*time.Hour)) {
			recentCount++
		}
	}
	if recentCount >= 3 {
		riskScore += 35
		reasons = append(reasons, "velocity")
	}
	country := ""
	if metadata != nil {
		country = strings.ToUpper(strings.TrimSpace(metadata["country"]))
	}
	if country != "" && country != "ZA" && country != "US" {
		riskScore += 30
		reasons = append(reasons, "country_rule")
	}
	status := models.WithdrawalStatusTreasuryApproval
	if riskScore >= 40 {
		status = models.WithdrawalStatusAMLReview
	}
	request := &models.WithdrawalRequest{
		ID:             newUUID(),
		UserID:         userID,
		Provider:       provider,
		Method:         method,
		Amount:         amount,
		Fee:            fee,
		Currency:       currency,
		Status:         status,
		RiskScore:      riskScore,
		Reference:      reference,
		Metadata:       metadata,
		RequestedAt:    now,
		LastTransition: now,
	}
	var review *models.AMLReview
	if riskScore > 0 {
		review = &models.AMLReview{
			ID:          newUUID(),
			UserID:      userID,
			Scope:       "withdrawal",
			ScopeID:     request.ID,
			Status:      "OPEN",
			RiskScore:   riskScore,
			Reasons:     reasons,
			Country:     country,
			EscalatedTo: models.RoleFraudAnalyst,
			CreatedAt:   now,
			UpdatedAt:   now,
		}
		request.AMLCaseID = review.ID
		s.amlReviews[review.ID] = review
	}
	wallet.PendingWithdrawals += totalHold
	s.withdrawals[request.ID] = request
	if err := s.recordFinancialIdempotencyLocked(ctx, idempotencyKey, userID, "withdrawal", "withdrawal_request", request.ID, requestHash); err != nil {
		return nil, nil, err
	}
	s.audit = append(s.audit, &models.AuditLog{
		ID:        newUUID(),
		ActorID:   userID,
		Action:    "wallet.withdrawal.requested",
		TargetID:  request.ID,
		Metadata:  map[string]string{"status": status, "riskScore": fmt.Sprintf("%d", riskScore)},
		CreatedAt: now,
	})
	if err := s.persistWallets(); err != nil {
		return nil, nil, err
	}
	if err := s.persistPaymentState(); err != nil {
		return nil, nil, err
	}
	if err := s.persistAuditLogs(); err != nil {
		return nil, nil, err
	}
	return request, review, nil
}

func (s *Store) ApproveWithdrawal(ctx context.Context, actorID, withdrawalID, ipAddress string) (*models.WithdrawalRequest, error) {
	now := time.Now().UTC()
	s.mu.Lock()
	defer s.mu.Unlock()
	request := s.withdrawals[withdrawalID]
	if request == nil {
		return nil, errors.New("withdrawal not found")
	}
	if request.Status != models.WithdrawalStatusTreasuryApproval && request.Status != models.WithdrawalStatusAMLReview {
		return nil, errors.New("withdrawal is not awaiting approval")
	}
	if request.AMLCaseID != "" {
		review := s.amlReviews[request.AMLCaseID]
		if review != nil && review.Status == "OPEN" {
			review.Status = "APPROVED"
			review.Decision = "approved_for_treasury"
			review.UpdatedAt = now
			review.ResolvedAt = &now
		}
	}
	request.Status = models.WithdrawalStatusProviderPending
	request.ApprovedAt = &now
	request.LastTransition = now
	s.audit = append(s.audit, &models.AuditLog{
		ID:        newUUID(),
		ActorID:   actorID,
		Action:    "treasury.withdrawal.approved",
		TargetID:  withdrawalID,
		IPAddress: ipAddress,
		CreatedAt: now,
	})
	if err := s.persistPaymentState(); err != nil {
		return nil, err
	}
	if err := s.persistAuditLogs(); err != nil {
		return nil, err
	}
	return request, nil
}

func (s *Store) RejectWithdrawal(ctx context.Context, actorID, withdrawalID, reason, ipAddress string) (*models.WithdrawalRequest, error) {
	now := time.Now().UTC()
	s.mu.Lock()
	defer s.mu.Unlock()
	request := s.withdrawals[withdrawalID]
	if request == nil {
		return nil, errors.New("withdrawal not found")
	}
	if request.Status == models.WithdrawalStatusSettled {
		return nil, errors.New("settled withdrawal cannot be rejected")
	}
	wallet := s.wallets[request.UserID]
	if wallet != nil {
		hold := request.Amount + request.Fee
		if wallet.PendingWithdrawals >= hold {
			wallet.PendingWithdrawals -= hold
		} else {
			wallet.PendingWithdrawals = 0
		}
	}
	request.Status = models.WithdrawalStatusRejected
	request.CompletedAt = &now
	request.LastTransition = now
	if request.Metadata == nil {
		request.Metadata = map[string]string{}
	}
	request.Metadata["rejectionReason"] = reason
	if request.AMLCaseID != "" {
		review := s.amlReviews[request.AMLCaseID]
		if review != nil && review.Status == "OPEN" {
			review.Status = "REJECTED"
			review.Decision = reason
			review.UpdatedAt = now
			review.ResolvedAt = &now
		}
	}
	s.audit = append(s.audit, &models.AuditLog{
		ID:        newUUID(),
		ActorID:   actorID,
		Action:    "treasury.withdrawal.rejected",
		TargetID:  withdrawalID,
		IPAddress: ipAddress,
		Metadata:  map[string]string{"reason": reason},
		CreatedAt: now,
	})
	if err := s.persistWallets(); err != nil {
		return nil, err
	}
	if err := s.persistPaymentState(); err != nil {
		return nil, err
	}
	if err := s.persistAuditLogs(); err != nil {
		return nil, err
	}
	return request, nil
}

func (s *Store) SettleWithdrawal(ctx context.Context, actorID, withdrawalID, providerRef, ipAddress string) ([]*models.LedgerEntry, error) {
	now := time.Now().UTC()
	s.mu.Lock()
	defer s.mu.Unlock()
	request := s.withdrawals[withdrawalID]
	if request == nil {
		return nil, errors.New("withdrawal not found")
	}
	if request.Status == models.WithdrawalStatusSettled {
		return nil, errors.New("withdrawal already settled")
	}
	if request.Status != models.WithdrawalStatusProviderPending && request.Status != models.WithdrawalStatusTreasuryApproval {
		return nil, errors.New("withdrawal is not settlement-ready")
	}
	wallet := s.wallets[request.UserID]
	if wallet == nil {
		return nil, errors.New("wallet not found")
	}
	hold := request.Amount + request.Fee
	if wallet.PendingWithdrawals >= hold {
		wallet.PendingWithdrawals -= hold
	} else {
		wallet.PendingWithdrawals = 0
	}
	if wallet.LiveBalance < hold {
		return nil, errors.New("insufficient live balance for settlement")
	}
	withdrawEntry := &models.LedgerEntry{
		ID:              newUUID(),
		UserID:          request.UserID,
		TransactionType: models.TransactionTypeWithdraw,
		Amount:          -request.Amount,
		BalanceBefore:   wallet.LiveBalance,
		Currency:        request.Currency,
		Reference:       request.ID,
		Metadata:        map[string]string{"provider": request.Provider, "providerRef": providerRef},
		CreatedAt:       now,
	}
	wallet.LiveBalance -= request.Amount
	withdrawEntry.BalanceAfter = wallet.LiveBalance
	feeEntry := &models.LedgerEntry{
		ID:              newUUID(),
		UserID:          request.UserID,
		TransactionType: models.TransactionTypeFee,
		Amount:          -request.Fee,
		BalanceBefore:   wallet.LiveBalance,
		Currency:        request.Currency,
		Reference:       request.ID + ":fee",
		Metadata:        map[string]string{"provider": request.Provider, "providerRef": providerRef},
		CreatedAt:       now,
	}
	wallet.LiveBalance -= request.Fee
	feeEntry.BalanceAfter = wallet.LiveBalance
	request.Status = models.WithdrawalStatusSettled
	request.ProviderRef = providerRef
	request.SettledAt = &now
	request.CompletedAt = &now
	request.LastTransition = now
	s.ledger[request.UserID] = append(s.ledger[request.UserID], withdrawEntry, feeEntry)
	s.audit = append(s.audit, &models.AuditLog{
		ID:        newUUID(),
		ActorID:   actorID,
		Action:    "treasury.withdrawal.settled",
		TargetID:  withdrawalID,
		IPAddress: ipAddress,
		CreatedAt: now,
	})
	if err := s.persistWallets(); err != nil {
		return nil, err
	}
	if err := s.persistLedger(); err != nil {
		return nil, err
	}
	if err := s.persistPaymentState(); err != nil {
		return nil, err
	}
	if err := s.persistAuditLogs(); err != nil {
		return nil, err
	}
	return []*models.LedgerEntry{withdrawEntry, feeEntry}, nil
}

func (s *Store) VerifyEmail(ctx context.Context, userID string) error {
	if s.usesPostgresAuth() {
		return s.pgVerifyEmail(ctx, userID)
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	user, ok := s.users[userID]
	if !ok {
		return errors.New("user not found")
	}
	user.EmailVerified = true
	s.recomputeTrustLocked(userID, "email_verified")
	if err := s.persistUsers(); err != nil {
		return err
	}
	if err := s.persistProgression(); err != nil {
		return err
	}
	return s.persistAuditLogs()
}

func (s *Store) SubmitKYC(ctx context.Context, userID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	user, ok := s.users[userID]
	if !ok {
		return errors.New("user not found")
	}
	user.KYCStatus = "pending"
	s.recomputeTrustLocked(userID, "kyc_submitted")
	if err := s.persistUsers(); err != nil {
		return err
	}
	if err := s.persistProgression(); err != nil {
		return err
	}
	return s.persistAuditLogs()
}

func (s *Store) GetKYCStatus(ctx context.Context, userID string) (string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	user, ok := s.users[userID]
	if !ok {
		return "", errors.New("user not found")
	}
	return user.KYCStatus, nil
}

func (s *Store) persistLedger() error {
	path := filepath.Join(s.dataDir, "ledger.json")
	entries := make([]*models.LedgerEntry, 0)
	for _, userEntries := range s.ledger {
		entries = append(entries, userEntries...)
	}

	data, err := json.MarshalIndent(entries, "", "  ")
	if err != nil {
		return err
	}
	return s.persistJSONOrSnapshotLocked(path, data)
}

func (s *Store) RegisterDevice(ctx context.Context, userID, fingerprint, deviceName, osName, browser string) (*models.Device, error) {
	if fingerprint == "" {
		return nil, errors.New("device fingerprint is required")
	}
	if s.usesPostgresAuth() {
		return s.pgRegisterDevice(ctx, userID, fingerprint, deviceName, osName, browser)
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.users[userID]; !ok {
		return nil, errors.New("user not found")
	}

	devices := s.devices[userID]
	var device *models.Device
	for _, existing := range devices {
		if existing.Fingerprint == fingerprint {
			existing.LastSeen = time.Now().UTC()
			existing.DeviceName = deviceName
			existing.OS = osName
			existing.Browser = browser
			device = existing
			break
		}
	}

	if device == nil {
		device = &models.Device{
			ID:          newUUID(),
			UserID:      userID,
			Fingerprint: fingerprint,
			DeviceName:  deviceName,
			OS:          osName,
			Browser:     browser,
			LastSeen:    time.Now().UTC(),
			CreatedAt:   time.Now().UTC(),
		}
		s.devices[userID] = append(s.devices[userID], device)
	}
	s.recomputeTrustLocked(userID, "device_registered")

	if err := s.persistDevices(); err != nil {
		return nil, err
	}
	if err := s.persistProgression(); err != nil {
		return nil, err
	}
	if err := s.persistAuditLogs(); err != nil {
		return nil, err
	}
	return device, nil
}

func (s *Store) ListDevices(ctx context.Context, userID string) ([]*models.Device, error) {
	if s.usesPostgresAuth() {
		return s.pgListDevices(ctx, userID)
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make([]*models.Device, 0, len(s.devices[userID]))
	for _, device := range s.devices[userID] {
		copyDevice := *device
		result = append(result, &copyDevice)
	}
	return result, nil
}

func (s *Store) RevokeDevice(ctx context.Context, userID, deviceID, actorID, ipAddress string) error {
	if s.usesPostgresAuth() {
		return s.pgRevokeDevice(ctx, userID, deviceID, actorID, ipAddress)
	}
	now := time.Now().UTC()
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, device := range s.devices[userID] {
		if device.ID == deviceID {
			device.RevokedAt = &now
			for _, session := range s.auth {
				if session.UserID == userID && session.DeviceID == deviceID && session.RevokedAt == nil {
					session.RevokedAt = &now
				}
			}
			if err := s.persistDevices(); err != nil {
				return err
			}
			return s.persistAuthSessions()
		}
	}
	return errors.New("device not found")
}

func (s *Store) GetDevicesByUserID(ctx context.Context, userID string) ([]*models.Device, error) {
	s.mu.RLock()
	devices := s.devices[userID]
	s.mu.RUnlock()
	if devices == nil {
		return []*models.Device{}, nil
	}
	return devices, nil
}

func (s *Store) addLedgerEntry(ctx context.Context, entry *models.LedgerEntry) error {
	if entry.ID == "" {
		entry.ID = newUUID()
	}
	if entry.CreatedAt.IsZero() {
		entry.CreatedAt = time.Now().UTC()
	}

	s.mu.Lock()
	s.ledger[entry.UserID] = append(s.ledger[entry.UserID], entry)
	s.mu.Unlock()

	return s.persistLedger()
}

func (s *Store) GetLedgerEntriesByUserID(ctx context.Context, userID string) ([]*models.LedgerEntry, error) {
	s.mu.RLock()
	entries, ok := s.ledger[userID]
	s.mu.RUnlock()
	if !ok {
		return []*models.LedgerEntry{}, nil
	}
	return entries, nil
}

func (s *Store) RecordWalletTransaction(ctx context.Context, userID string, txnType string, amount float64, currency string, reference string, metadata map[string]string) (*models.LedgerEntry, error) {
	if amount <= 0 {
		return nil, errors.New("amount must be greater than zero")
	}
	if currency == "" {
		currency = "USD"
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	wallet, ok := s.wallets[userID]
	if !ok {
		return nil, errors.New("wallet not found")
	}

	entry := &models.LedgerEntry{
		ID:              newUUID(),
		UserID:          userID,
		TransactionType: txnType,
		Amount:          amount,
		Currency:        currency,
		Reference:       reference,
		Metadata:        metadata,
		CreatedAt:       time.Now().UTC(),
	}

	switch txnType {
	case models.TransactionTypeDeposit:
		entry.BalanceBefore = wallet.LiveBalance
		wallet.LiveBalance += amount
		entry.BalanceAfter = wallet.LiveBalance
		entry.Amount = amount
	case models.TransactionTypeWithdraw:
		if wallet.LiveBalance < amount {
			return nil, errors.New("insufficient live balance")
		}
		entry.BalanceBefore = wallet.LiveBalance
		wallet.LiveBalance -= amount
		entry.BalanceAfter = wallet.LiveBalance
		entry.Amount = -amount
	case models.TransactionTypeFee:
		if wallet.LiveBalance < amount {
			return nil, errors.New("insufficient live balance for fee")
		}
		entry.BalanceBefore = wallet.LiveBalance
		wallet.LiveBalance -= amount
		entry.BalanceAfter = wallet.LiveBalance
		entry.Amount = -amount
	default:
		return nil, errors.New("unsupported transaction type")
	}

	s.ledger[userID] = append(s.ledger[userID], entry)

	if err := s.persistWallets(); err != nil {
		return nil, err
	}
	if err := s.persistLedger(); err != nil {
		return nil, err
	}

	return entry, nil
}

func defaultProgression(userID string) *models.Progression {
	now := time.Now().UTC()
	return &models.Progression{
		UserID:     userID,
		Level:      1,
		EloRating:  1200,
		LeagueTier: "Bronze",
		TrustScore: 100,
		TrustTier:  "trusted",
		UpdatedAt:  now,
	}
}

func levelFromXP(xp int) (int, int) {
	if xp < 0 {
		xp = 0
	}
	level := (xp / 100) + 1
	prestige := 0
	if level > 100 {
		prestige = (level - 1) / 100
		level = ((level - 1) % 100) + 1
	}
	return level, prestige
}

func leagueFromElo(elo int) string {
	switch {
	case elo >= 2200:
		return "Legend"
	case elo >= 2000:
		return "Elite"
	case elo >= 1800:
		return "Diamond"
	case elo >= 1600:
		return "Platinum"
	case elo >= 1400:
		return "Gold"
	case elo >= 1200:
		return "Silver"
	default:
		return "Bronze"
	}
}

func clampTrust(score float64) float64 {
	if score < 0 {
		return 0
	}
	if score > 100 {
		return 100
	}
	return score
}

func trustTier(score float64) string {
	settings := config.Runtime().Trust
	switch {
	case score >= settings.TrustedMinimum:
		return "trusted"
	case score >= settings.StandardMinimum:
		return "standard"
	case score >= settings.LimitedMinimum:
		return "limited"
	case score >= settings.ReviewMinimum:
		return "review"
	default:
		return "restricted"
	}
}

func withdrawalLimitForTrust(tier string) float64 {
	if limit, ok := config.Runtime().Trust.WithdrawalLimits[tier]; ok {
		return limit
	}
	return 0
}

var houseTiers = []*models.HouseTier{
	{ID: "bronze", Name: "Bronze House", MinimumLevel: 1, MinimumTrust: 70, Stake: 10, RewardRate: 0.45, Difficulty: 1, TargetHouseEdge: 0.65, Description: "Entry house challenge for new competitive players."},
	{ID: "silver", Name: "Silver House", MinimumLevel: 3, MinimumTrust: 75, Stake: 25, RewardRate: 0.75, Difficulty: 2, TargetHouseEdge: 0.65, Description: "Intermediate challenge with tighter route pressure."},
	{ID: "gold", Name: "Gold House", MinimumLevel: 5, MinimumTrust: 80, Stake: 50, RewardRate: 1.15, Difficulty: 3, TargetHouseEdge: 0.65, Description: "Advanced challenge requiring consistent skill."},
	{ID: "platinum", Name: "Platinum House", MinimumLevel: 10, MinimumTrust: 85, Stake: 100, RewardRate: 1.75, Difficulty: 4, TargetHouseEdge: 0.65, Description: "High-stakes challenge with stricter eligibility."},
	{ID: "elite", Name: "Elite House", MinimumLevel: 20, MinimumTrust: 90, Stake: 250, RewardRate: 2.5, Difficulty: 5, TargetHouseEdge: 0.65, Description: "Top-tier challenge for proven players."},
	{ID: "legend", Name: "Legend House", MinimumLevel: 40, MinimumTrust: 95, Stake: 500, RewardRate: 4.0, Difficulty: 6, TargetHouseEdge: 0.65, Description: "Prestige challenge with maximum scrutiny."},
}

func findHouseTier(tierID string) *models.HouseTier {
	for _, tier := range houseTiers {
		if tier.ID == tierID {
			copy := *tier
			return &copy
		}
	}
	return nil
}

func (s *Store) playerLiabilitiesLocked() float64 {
	total := 0.0
	for _, wallet := range s.wallets {
		total += wallet.LiveBalance + wallet.LiveLockedBalance + wallet.PendingWithdrawals + wallet.BonusBalance
	}
	return total
}

func (s *Store) houseExposureLocked() float64 {
	exposure := 0.0
	for _, session := range s.sessions {
		if session.Mode == "house" && !session.IsFinished && session.GameType == "live" {
			exposure += session.Stake * session.RewardRate
		}
	}
	return exposure
}

func treasuryTotal(state *models.TreasuryState) float64 {
	return state.PlayerReserve + state.RevenueReserve + state.SeasonReserve + state.ChampionshipReserve + state.JackpotReserve + state.EmergencyReserve
}

func (s *Store) treasuryHealthLocked() *models.TreasuryHealth {
	liabilities := s.playerLiabilitiesLocked()
	totalReserves := treasuryTotal(s.treasury)
	coverageRatio := 1.0
	if liabilities > 0 {
		coverageRatio = totalReserves / liabilities
	}
	return &models.TreasuryHealth{
		PlayerLiabilities: liabilities,
		TotalReserves:     totalReserves,
		CoverageRatio:     coverageRatio,
		IsSolvent:         totalReserves >= liabilities,
		HouseExposure:     s.houseExposureLocked(),
		State:             *s.treasury,
	}
}

func (s *Store) GetTreasuryHealth(ctx context.Context) (*models.TreasuryHealth, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.treasuryHealthLocked(), nil
}

func (s *Store) HouseRiskReport(ctx context.Context, tierID string) (*models.HouseRiskReport, error) {
	tier := findHouseTier(tierID)
	if tier == nil {
		return nil, errors.New("house tier not found")
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	report := &models.HouseRiskReport{
		TierID:            tier.ID,
		TargetHouseEdge:   tier.TargetHouseEdge,
		RecommendedAction: "maintain",
	}
	for _, session := range s.sessions {
		if session.Mode != "house" || session.HouseTier != tier.ID || !session.IsFinished {
			continue
		}
		report.Attempts++
		if session.Outcome == "win" {
			report.Wins++
		} else {
			report.Losses++
		}
	}
	if report.Attempts > 0 {
		report.PlayerWinRate = float64(report.Wins) / float64(report.Attempts)
	}
	if report.Attempts >= 10 {
		houseWinRate := 1 - report.PlayerWinRate
		if houseWinRate < 0.60 {
			report.RecommendedAction = "increase_difficulty"
		}
		if houseWinRate > 0.70 {
			report.RecommendedAction = "decrease_difficulty"
		}
	}
	return report, nil
}

func (s *Store) ListHouseTiers(ctx context.Context, userID string) ([]*models.HouseTier, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.users[userID]; !ok {
		return nil, errors.New("user not found")
	}
	progression := s.ensureProgressionLocked(userID)
	tiers := make([]*models.HouseTier, 0, len(houseTiers))
	for _, tier := range houseTiers {
		copy := *tier
		if progression.Level < tier.MinimumLevel || progression.TrustScore < tier.MinimumTrust {
			copy.Description += " Locked until eligibility is met."
		}
		tiers = append(tiers, &copy)
	}
	return tiers, nil
}

func (s *Store) ensureProgressionLocked(userID string) *models.Progression {
	progression, ok := s.profiles[userID]
	if !ok {
		progression = defaultProgression(userID)
		s.profiles[userID] = progression
	}
	if progression.TrustTier == "" {
		progression.TrustTier = trustTier(progression.TrustScore)
	}
	return progression
}

func (s *Store) recomputeTrustLocked(userID, reason string) {
	user, ok := s.users[userID]
	if !ok {
		return
	}
	progression := s.ensureProgressionLocked(userID)
	score := 50.0
	accountAgeDays := time.Since(user.CreatedAt).Hours() / 24
	if accountAgeDays >= 30 {
		score += 12
	} else if accountAgeDays >= 7 {
		score += 7
	} else if accountAgeDays >= 1 {
		score += 3
	}
	if progression.MatchesPlayed >= 100 {
		score += 15
	} else if progression.MatchesPlayed >= 25 {
		score += 10
	} else if progression.MatchesPlayed >= 5 {
		score += 5
	}
	if user.EmailVerified {
		score += 8
	}
	if user.KYCStatus == "approved" {
		score += 10
	} else if user.KYCStatus == "pending" {
		score += 4
	}
	devices := s.devices[userID]
	if len(devices) == 1 {
		score += 8
	} else if len(devices) > 3 {
		score -= float64(len(devices)-3) * 4
	}
	withdrawals := 0
	for _, entry := range s.ledger[userID] {
		if entry.TransactionType == models.TransactionTypeWithdraw {
			withdrawals++
		}
	}
	if withdrawals > 10 {
		score -= 8
	} else if withdrawals > 0 {
		score += 2
	}
	for _, reviewCase := range s.reviewCases {
		if reviewCase.UserID != userID {
			continue
		}
		switch reviewCase.Status {
		case "PENDING_REVIEW", "MANUAL_REVIEW":
			score -= 10
		case "REJECTED":
			score -= 25
		case "APPROVED":
			score += 3
		}
		if reviewCase.Reason == "fraud_flag" {
			score -= 15
		}
	}
	oldScore := progression.TrustScore
	oldTier := progression.TrustTier
	progression.TrustScore = clampTrust(score)
	progression.TrustTier = trustTier(progression.TrustScore)
	progression.UpdatedAt = time.Now().UTC()
	if oldScore != progression.TrustScore || oldTier != progression.TrustTier {
		s.audit = append(s.audit, &models.AuditLog{
			ID:       id.Audit(),
			ActorID:  "system",
			Action:   "trust.score.changed",
			TargetID: userID,
			Metadata: map[string]string{
				"reason":   reason,
				"oldScore": fmt.Sprintf("%.2f", oldScore),
				"newScore": fmt.Sprintf("%.2f", progression.TrustScore),
				"oldTier":  oldTier,
				"newTier":  progression.TrustTier,
			},
			CreatedAt: time.Now().UTC(),
		})
	}
}

func transitionSessionState(session *models.GameSession, next string) error {
	if session.State == "" {
		session.State = models.SessionStateCreated
	}
	allowed := map[string][]string{
		models.SessionStateCreated:    {models.SessionStateGenerating, models.SessionStateCancelled, models.SessionStateExpired},
		models.SessionStateGenerating: {models.SessionStateReady, models.SessionStateCancelled, models.SessionStateExpired},
		models.SessionStateReady:      {models.SessionStateActive, models.SessionStateCancelled, models.SessionStateExpired},
		models.SessionStateActive:     {models.SessionStateCompleted, models.SessionStateCancelled, models.SessionStateExpired},
	}
	for _, candidate := range allowed[session.State] {
		if candidate == next {
			session.State = next
			return nil
		}
	}
	if session.State == next {
		return nil
	}
	return fmt.Errorf("invalid session state transition from %s to %s", session.State, next)
}

func (s *Store) difficultyProfileForUserLocked(userID, source, houseTier, tournamentTier string, difficultyBoost int) models.DifficultyProfile {
	progression := s.ensureProgressionLocked(userID)
	profile := game.BuildDifficultyProfile(game.DifficultyInput{
		PlayerLevel:    progression.Level + difficultyBoost,
		LeagueTier:     progression.LeagueTier,
		TrustScore:     progression.TrustScore,
		HouseTier:      houseTier,
		TournamentTier: tournamentTier,
		Source:         source,
	})
	return profile
}

func (s *Store) matchDifficultyProfileLocked(playerAID, playerBID, source, tournamentTier string) models.DifficultyProfile {
	playerA := s.difficultyProfileForUserLocked(playerAID, source, "", tournamentTier, 0)
	playerB := s.difficultyProfileForUserLocked(playerBID, source, "", tournamentTier, 0)
	rating := (playerA.Rating + playerB.Rating) / 2
	if rating < playerA.Rating-10 {
		rating = playerA.Rating - 10
	}
	if rating < playerB.Rating-10 {
		rating = playerB.Rating - 10
	}
	return game.ProfileFromRating(rating, source)
}

func (s *Store) unlockAchievementLocked(userID, code, title, description string) {
	for _, existing := range s.awards[userID] {
		if existing.Code == code {
			return
		}
	}
	s.awards[userID] = append(s.awards[userID], &models.Achievement{
		ID:          newUUID(),
		UserID:      userID,
		Code:        code,
		Title:       title,
		Description: description,
		UnlockedAt:  time.Now().UTC(),
	})
}

func (s *Store) applyGameProgressionLocked(session *models.GameSession, valid bool) {
	progression := s.ensureProgressionLocked(session.UserID)
	progression.MatchesPlayed++
	progression.XP += 10
	progression.SeasonPoints += 5
	progression.LegacyPoints++

	if valid {
		progression.Wins++
		progression.CurrentStreak++
		progression.XP += 50
		progression.SeasonPoints += 25
		progression.LegacyPoints += 5
		progression.EloRating += 16
		if progression.BestMoves == 0 || len(session.Moves) < progression.BestMoves {
			progression.BestMoves = len(session.Moves)
		}
		progression.TrustScore = clampTrust(progression.TrustScore + 0.2)
	} else {
		progression.Losses++
		progression.CurrentStreak = 0
		progression.EloRating -= 12
		if progression.EloRating < 1000 {
			progression.EloRating = 1000
		}
		progression.TrustScore = clampTrust(progression.TrustScore - 1)
	}

	if session.Mode == "house" && valid {
		progression.HouseRep += 2
	}

	progression.Level, progression.Prestige = levelFromXP(progression.XP)
	progression.LeagueTier = leagueFromElo(progression.EloRating)
	progression.UpdatedAt = time.Now().UTC()
	s.recomputeTrustLocked(session.UserID, "game_progression")

	s.unlockAchievementLocked(session.UserID, "first_match", "First Run", "Complete your first Maze Arena session.")
	if valid {
		s.unlockAchievementLocked(session.UserID, "first_win", "Maze Breaker", "Win a Maze Arena session.")
	}
	if progression.MatchesPlayed >= 10 {
		s.unlockAchievementLocked(session.UserID, "ten_matches", "Arena Regular", "Complete ten Maze Arena sessions.")
	}
	if progression.CurrentStreak >= 3 {
		s.unlockAchievementLocked(session.UserID, "three_streak", "Clean Streak", "Win three Maze Arena sessions in a row.")
	}
}

func (s *Store) GetProgressionByUserID(ctx context.Context, userID string) (*models.Progression, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.users[userID]; !ok {
		return nil, errors.New("user not found")
	}
	progression := s.ensureProgressionLocked(userID)
	if err := s.persistProgression(); err != nil {
		return nil, err
	}
	return progression, nil
}

func (s *Store) GetAchievementsByUserID(ctx context.Context, userID string) ([]*models.Achievement, error) {
	s.mu.RLock()
	achievements := s.awards[userID]
	s.mu.RUnlock()
	if achievements == nil {
		return []*models.Achievement{}, nil
	}
	return achievements, nil
}

func hashToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

func NewRefreshToken() string {
	return newUUID() + "." + newUUID()
}

func NewAuthToken() string {
	return newUUID() + "." + newUUID() + "." + newUUID()
}

func (s *Store) CreateAuthToken(ctx context.Context, userID, purpose, rawToken, ipAddress string, ttl time.Duration) (*models.AuthToken, error) {
	if userID == "" || purpose == "" || rawToken == "" {
		return nil, errors.New("user id, purpose, and token are required")
	}
	if ttl <= 0 {
		ttl = 30 * time.Minute
	}
	if s.usesPostgresAuth() {
		return s.pgCreateAuthToken(ctx, userID, purpose, rawToken, ipAddress, ttl)
	}
	now := time.Now().UTC()
	token := &models.AuthToken{
		ID:        newUUID(),
		UserID:    userID,
		Purpose:   purpose,
		TokenHash: hashToken(rawToken),
		ExpiresAt: now.Add(ttl),
		CreatedAt: now,
		CreatedIP: ipAddress,
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.users[userID]; !ok {
		return nil, errors.New("user not found")
	}
	for _, existing := range s.authTokens {
		if existing.UserID == userID && existing.Purpose == purpose && existing.UsedAt == nil && existing.ExpiresAt.After(now) {
			expired := now
			existing.UsedAt = &expired
		}
	}
	s.authTokens[token.ID] = token
	s.audit = append(s.audit, &models.AuditLog{
		ID:        newUUID(),
		ActorID:   userID,
		Action:    "auth.token.created." + purpose,
		TargetID:  token.ID,
		IPAddress: ipAddress,
		CreatedAt: now,
	})
	if err := s.persistAuthHardening(); err != nil {
		return nil, err
	}
	if err := s.persistAuditLogs(); err != nil {
		return nil, err
	}
	return token, nil
}

func (s *Store) ConsumeAuthToken(ctx context.Context, purpose, rawToken, ipAddress string) (*models.AuthToken, *models.User, error) {
	if purpose == "" || rawToken == "" {
		return nil, nil, errors.New("purpose and token are required")
	}
	if s.usesPostgresAuth() {
		return s.pgConsumeAuthToken(ctx, purpose, rawToken, ipAddress)
	}
	tokenHash := hashToken(rawToken)
	now := time.Now().UTC()

	s.mu.Lock()
	defer s.mu.Unlock()
	for _, token := range s.authTokens {
		if token.Purpose != purpose || token.TokenHash != tokenHash {
			continue
		}
		user := s.users[token.UserID]
		if user == nil {
			return nil, nil, errors.New("user not found")
		}
		if token.UsedAt != nil {
			copyToken := *token
			copyUser := *user
			return &copyToken, &copyUser, errors.New("token already used")
		}
		if !token.ExpiresAt.After(now) {
			return nil, nil, errors.New("token expired")
		}
		token.UsedAt = &now
		token.UsedIP = ipAddress
		s.audit = append(s.audit, &models.AuditLog{
			ID:        newUUID(),
			ActorID:   user.ID,
			Action:    "auth.token.consumed." + purpose,
			TargetID:  token.ID,
			IPAddress: ipAddress,
			CreatedAt: now,
		})
		if err := s.persistAuthHardening(); err != nil {
			return nil, nil, err
		}
		if err := s.persistAuditLogs(); err != nil {
			return nil, nil, err
		}
		return token, user, nil
	}
	return nil, nil, errors.New("token not found")
}

func (s *Store) InspectAuthToken(ctx context.Context, purpose, rawToken string) (*models.AuthToken, *models.User, error) {
	if purpose == "" || rawToken == "" {
		return nil, nil, errors.New("purpose and token are required")
	}
	if s.usesPostgresAuth() {
		return s.pgInspectAuthToken(ctx, purpose, rawToken)
	}
	tokenHash := hashToken(rawToken)
	now := time.Now().UTC()
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, token := range s.authTokens {
		if token.Purpose != purpose || token.TokenHash != tokenHash {
			continue
		}
		if token.UsedAt != nil {
			return nil, nil, errors.New("token already used")
		}
		if !token.ExpiresAt.After(now) {
			return nil, nil, errors.New("token expired")
		}
		user := s.users[token.UserID]
		if user == nil {
			return nil, nil, errors.New("user not found")
		}
		copyToken := *token
		copyUser := *user
		return &copyToken, &copyUser, nil
	}
	return nil, nil, errors.New("token not found")
}

func (s *Store) CreateAuthSession(ctx context.Context, userID, refreshToken, userAgent, ipAddress string, ttl time.Duration) (*models.AuthSession, error) {
	return s.CreateAuthSessionWithState(ctx, userID, refreshToken, userAgent, ipAddress, ttl, false, false)
}

func (s *Store) CreateAuthSessionWithState(ctx context.Context, userID, refreshToken, userAgent, ipAddress string, ttl time.Duration, mfaVerified, enrollmentOnly bool) (*models.AuthSession, error) {
	return s.CreateAuthSessionForDevice(ctx, userID, refreshToken, userAgent, ipAddress, "", ttl, mfaVerified, enrollmentOnly)
}

func (s *Store) CreateAuthSessionForDevice(ctx context.Context, userID, refreshToken, userAgent, ipAddress, deviceID string, ttl time.Duration, mfaVerified, enrollmentOnly bool) (*models.AuthSession, error) {
	if userID == "" || refreshToken == "" {
		return nil, errors.New("user id and refresh token are required")
	}
	if ttl <= 0 {
		ttl = 30 * 24 * time.Hour
	}
	if s.usesPostgresAuth() {
		return s.pgCreateAuthSession(ctx, userID, refreshToken, userAgent, ipAddress, deviceID, ttl, mfaVerified, enrollmentOnly)
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.users[userID]; !ok {
		return nil, errors.New("user not found")
	}

	session := &models.AuthSession{
		ID:               newUUID(),
		UserID:           userID,
		RefreshTokenHash: hashToken(refreshToken),
		UserAgent:        userAgent,
		IPAddress:        ipAddress,
		DeviceID:         deviceID,
		CreatedAt:        time.Now().UTC(),
		ExpiresAt:        time.Now().UTC().Add(ttl),
		MFAVerified:      mfaVerified,
		EnrollmentOnly:   enrollmentOnly,
	}
	session.FamilyID = session.ID
	s.auth[session.ID] = session
	s.audit = append(s.audit, &models.AuditLog{
		ID:        newUUID(),
		ActorID:   userID,
		Action:    "auth.session.created",
		TargetID:  session.ID,
		IPAddress: ipAddress,
		CreatedAt: time.Now().UTC(),
	})
	if err := s.persistAuthSessions(); err != nil {
		return nil, err
	}
	if err := s.persistAuditLogs(); err != nil {
		return nil, err
	}
	if err := s.setRedisJSON(ctx, "session:"+session.ID, session, ttl); err != nil {
		return nil, err
	}
	if err := s.redis.Set(ctx, "presence:"+userID, "online", 5*time.Minute); err != nil {
		return nil, err
	}
	return session, nil
}

func (s *Store) RotateRefreshToken(ctx context.Context, oldRefreshToken, newRefreshToken, userAgent, ipAddress string, ttl time.Duration) (*models.User, *models.AuthSession, error) {
	if oldRefreshToken == "" || newRefreshToken == "" {
		return nil, nil, errors.New("old and new refresh tokens are required")
	}
	oldHash := hashToken(oldRefreshToken)
	now := time.Now().UTC()
	if ttl <= 0 {
		ttl = 30 * 24 * time.Hour
	}
	if s.usesPostgresAuth() {
		return s.pgRotateRefreshToken(ctx, oldRefreshToken, newRefreshToken, userAgent, ipAddress, ttl)
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	for _, session := range s.auth {
		if session.RefreshTokenHash != oldHash {
			continue
		}
		if session.RevokedAt != nil && session.RotatedAt != nil {
			for _, related := range s.auth {
				if (session.FamilyID != "" && related.FamilyID == session.FamilyID) || (session.FamilyID == "" && related.UserID == session.UserID) {
					revokedAt := now
					related.RevokedAt = &revokedAt
					_ = s.redis.Del(ctx, "session:"+related.ID)
				}
			}
			s.audit = append(s.audit, &models.AuditLog{ID: newUUID(), ActorID: session.UserID, Action: "auth.refresh.reuse_detected", TargetID: session.ID, IPAddress: ipAddress, CreatedAt: now})
			if err := s.persistAuthSessions(); err != nil {
				return nil, nil, err
			}
			if err := s.persistAuditLogs(); err != nil {
				return nil, nil, err
			}
			return nil, nil, errors.New("refresh token reuse detected")
		}
		if session.RevokedAt != nil || !session.ExpiresAt.After(now) {
			return nil, nil, errors.New("refresh token is expired or revoked")
		}
		user := s.users[session.UserID]
		if user == nil {
			return nil, nil, errors.New("user not found")
		}
		session.RevokedAt = &now
		session.RotatedAt = &now
		replacement := &models.AuthSession{
			ID:               newUUID(),
			UserID:           user.ID,
			RefreshTokenHash: hashToken(newRefreshToken),
			UserAgent:        userAgent,
			IPAddress:        ipAddress,
			DeviceID:         session.DeviceID,
			FamilyID:         session.FamilyID,
			CreatedAt:        now,
			ExpiresAt:        now.Add(ttl),
			MFAVerified:      session.MFAVerified,
			EnrollmentOnly:   session.EnrollmentOnly,
		}
		if replacement.FamilyID == "" {
			replacement.FamilyID = session.ID
		}
		s.auth[replacement.ID] = replacement
		s.audit = append(s.audit, &models.AuditLog{
			ID:        newUUID(),
			ActorID:   user.ID,
			Action:    "auth.session.rotated",
			TargetID:  replacement.ID,
			IPAddress: ipAddress,
			CreatedAt: now,
		})
		if err := s.persistAuthSessions(); err != nil {
			return nil, nil, err
		}
		if err := s.persistAuditLogs(); err != nil {
			return nil, nil, err
		}
		_ = s.redis.Del(ctx, "session:"+session.ID)
		if err := s.setRedisJSON(ctx, "session:"+replacement.ID, replacement, ttl); err != nil {
			return nil, nil, err
		}
		if err := s.redis.Set(ctx, "presence:"+user.ID, "online", 5*time.Minute); err != nil {
			return nil, nil, err
		}
		return user, replacement, nil
	}
	return nil, nil, errors.New("refresh token not found")
}

func (s *Store) RevokeUserSessions(ctx context.Context, userID, actorID, ipAddress, reason string) error {
	if s.usesPostgresAuth() {
		return s.pgRevokeUserSessions(ctx, userID, actorID, ipAddress, reason)
	}
	now := time.Now().UTC()
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, session := range s.auth {
		if session.UserID == userID && session.RevokedAt == nil {
			session.RevokedAt = &now
		}
	}
	s.audit = append(s.audit, &models.AuditLog{
		ID:        newUUID(),
		ActorID:   actorID,
		Action:    "auth.sessions.revoked",
		TargetID:  userID,
		IPAddress: ipAddress,
		Metadata:  map[string]string{"reason": reason},
		CreatedAt: now,
	})
	if err := s.persistAuthSessions(); err != nil {
		return err
	}
	for _, session := range s.auth {
		if session.UserID == userID {
			_ = s.redis.Del(ctx, "session:"+session.ID)
		}
	}
	return s.persistAuditLogs()
}

func (s *Store) GetUserByRefreshToken(ctx context.Context, refreshToken string) (*models.User, *models.AuthSession, error) {
	if s.usesPostgresAuth() {
		tokenHash := hashToken(refreshToken)
		session, err := scanAuthSession(s.pg.QueryRowContext(ctx, `SELECT `+authSessionColumns+` FROM auth_sessions WHERE refresh_token_hash=$1`, tokenHash))
		if err != nil || session.RevokedAt != nil || !session.ExpiresAt.After(time.Now().UTC()) {
			return nil, nil, errors.New("refresh token is expired or revoked")
		}
		user, err := s.pgGetUserByID(ctx, session.UserID)
		return user, session, err
	}
	tokenHash := hashToken(refreshToken)
	now := time.Now().UTC()

	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, session := range s.auth {
		if session.RefreshTokenHash != tokenHash {
			continue
		}
		if session.RevokedAt != nil || !session.ExpiresAt.After(now) {
			return nil, nil, errors.New("refresh token is expired or revoked")
		}
		user, ok := s.users[session.UserID]
		if !ok {
			return nil, nil, errors.New("user not found")
		}
		return user, session, nil
	}
	return nil, nil, errors.New("refresh token not found")
}

func (s *Store) RevokeRefreshToken(ctx context.Context, refreshToken, actorID, ipAddress string) error {
	if s.usesPostgresAuth() {
		return s.pgRevokeRefreshToken(ctx, refreshToken, actorID, ipAddress)
	}
	tokenHash := hashToken(refreshToken)
	now := time.Now().UTC()

	s.mu.Lock()
	defer s.mu.Unlock()

	for _, session := range s.auth {
		if session.RefreshTokenHash != tokenHash {
			continue
		}
		session.RevokedAt = &now
		s.audit = append(s.audit, &models.AuditLog{
			ID:        newUUID(),
			ActorID:   actorID,
			Action:    "auth.session.revoked",
			TargetID:  session.ID,
			IPAddress: ipAddress,
			CreatedAt: now,
		})
		if err := s.persistAuthSessions(); err != nil {
			return err
		}
		_ = s.redis.Del(ctx, "session:"+session.ID)
		return s.persistAuditLogs()
	}
	return errors.New("refresh token not found")
}

func (s *Store) ValidateAuthSession(ctx context.Context, sessionID, userID string) (*models.AuthSession, *models.User, error) {
	if sessionID == "" || userID == "" {
		return nil, nil, errors.New("session is required")
	}
	if s.usesPostgresAuth() {
		return s.pgValidateAuthSession(ctx, sessionID, userID)
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	session := s.auth[sessionID]
	if session == nil || session.UserID != userID || session.RevokedAt != nil || !session.ExpiresAt.After(time.Now().UTC()) {
		return nil, nil, errors.New("session is expired or revoked")
	}
	user := s.users[userID]
	if user == nil || (user.Status != "" && user.Status != "active") {
		return nil, nil, errors.New("account is not active")
	}
	copySession := *session
	copyUser := *user
	return &copySession, &copyUser, nil
}

func (s *Store) ListAuthSessions(ctx context.Context, userID string) ([]*models.AuthSession, error) {
	if s.usesPostgresAuth() {
		return s.pgListAuthSessions(ctx, userID)
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := []*models.AuthSession{}
	for _, session := range s.auth {
		if session.UserID == userID {
			copySession := *session
			result = append(result, &copySession)
		}
	}
	sort.Slice(result, func(i, j int) bool { return result[i].CreatedAt.After(result[j].CreatedAt) })
	return result, nil
}

func (s *Store) MarkSessionMFA(ctx context.Context, sessionID, userID string, verified, enrollmentOnly bool) error {
	if s.usesPostgresAuth() {
		return s.pgMarkSessionMFA(ctx, sessionID, userID, verified, enrollmentOnly)
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	session := s.auth[sessionID]
	if session == nil || session.UserID != userID || session.RevokedAt != nil {
		return errors.New("session not found")
	}
	session.MFAVerified = verified
	session.EnrollmentOnly = enrollmentOnly
	return s.persistAuthSessions()
}

func (s *Store) RevokeAuthSession(ctx context.Context, sessionID, userID, actorID, ipAddress, reason string) error {
	if s.usesPostgresAuth() {
		return s.pgRevokeSession(ctx, sessionID, userID, actorID, ipAddress, reason)
	}
	now := time.Now().UTC()
	s.mu.Lock()
	defer s.mu.Unlock()
	session := s.auth[sessionID]
	if session == nil || session.UserID != userID {
		return errors.New("session not found")
	}
	session.RevokedAt = &now
	_ = s.redis.Del(ctx, "session:"+session.ID)
	return s.persistAuthSessions()
}

func (s *Store) LoginSecurityState(ctx context.Context, userID string) (*models.LoginSecurityState, error) {
	if s.usesPostgresAuth() {
		return s.pgLoginSecurityState(ctx, userID)
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	state := s.loginSecurity[userID]
	if state == nil {
		state = &models.LoginSecurityState{UserID: userID, UpdatedAt: time.Now().UTC()}
		s.loginSecurity[userID] = state
		_ = s.persistAuthHardening()
	}
	copy := *state
	return &copy, nil
}

func (s *Store) RecordLoginFailure(ctx context.Context, userID, ipAddress, userAgent string) (*models.LoginSecurityState, error) {
	if s.usesPostgresAuth() {
		return s.pgRecordLoginFailure(ctx, userID, ipAddress, userAgent)
	}
	now := time.Now().UTC()
	s.mu.Lock()
	defer s.mu.Unlock()
	state := s.loginSecurity[userID]
	if state == nil {
		state = &models.LoginSecurityState{UserID: userID}
		s.loginSecurity[userID] = state
	}
	state.FailedCount++
	state.LastFailedAt = &now
	state.UpdatedAt = now
	if state.FailedCount >= 5 {
		lockedUntil := now.Add(15 * time.Minute)
		state.LockedUntil = &lockedUntil
	}
	s.audit = append(s.audit, &models.AuditLog{
		ID:        newUUID(),
		ActorID:   userID,
		Action:    "auth.login.failed",
		TargetID:  userID,
		IPAddress: ipAddress,
		Metadata:  map[string]string{"failedCount": fmt.Sprintf("%d", state.FailedCount), "userAgent": userAgent},
		CreatedAt: now,
	})
	if err := s.persistAuthHardening(); err != nil {
		return nil, err
	}
	if err := s.persistAuditLogs(); err != nil {
		return nil, err
	}
	copy := *state
	return &copy, nil
}

func (s *Store) RecordLoginSuccess(ctx context.Context, userID, ipAddress, userAgent string) error {
	if s.usesPostgresAuth() {
		return s.pgRecordLoginSuccess(ctx, userID, ipAddress, userAgent)
	}
	now := time.Now().UTC()
	s.mu.Lock()
	defer s.mu.Unlock()
	state := s.loginSecurity[userID]
	if state == nil {
		state = &models.LoginSecurityState{UserID: userID}
		s.loginSecurity[userID] = state
	}
	if state.LastIPAddress != "" && state.LastIPAddress != ipAddress {
		state.SuspiciousFlag = "ip_changed"
		s.audit = append(s.audit, &models.AuditLog{
			ID:        newUUID(),
			ActorID:   userID,
			Action:    "auth.login.suspicious",
			TargetID:  userID,
			IPAddress: ipAddress,
			Metadata:  map[string]string{"previousIp": state.LastIPAddress, "userAgent": userAgent},
			CreatedAt: now,
		})
	}
	state.FailedCount = 0
	state.LockedUntil = nil
	state.LastSuccessAt = &now
	state.LastIPAddress = ipAddress
	state.LastUserAgent = userAgent
	state.UpdatedAt = now
	if err := s.persistAuthHardening(); err != nil {
		return err
	}
	return s.persistAuditLogs()
}

func (s *Store) UpdatePassword(ctx context.Context, userID, passwordHash, passwordStamp, ipAddress string) error {
	if s.usesPostgresAuth() {
		return s.pgUpdatePassword(ctx, userID, passwordHash, passwordStamp, ipAddress)
	}
	now := time.Now().UTC()
	s.mu.Lock()
	defer s.mu.Unlock()
	user := s.users[userID]
	if user == nil {
		return errors.New("user not found")
	}
	user.PasswordHash = passwordHash
	s.passwords[userID] = append(s.passwords[userID], &models.PasswordHistoryEntry{
		UserID:        userID,
		PasswordHash:  passwordHash,
		PasswordStamp: passwordStamp,
		CreatedAt:     now,
	})
	if len(s.passwords[userID]) > 5 {
		s.passwords[userID] = s.passwords[userID][len(s.passwords[userID])-5:]
	}
	s.audit = append(s.audit, &models.AuditLog{
		ID:        newUUID(),
		ActorID:   userID,
		Action:    "auth.password.changed",
		TargetID:  userID,
		IPAddress: ipAddress,
		CreatedAt: now,
	})
	if err := s.persistUsers(); err != nil {
		return err
	}
	if err := s.persistAuthHardening(); err != nil {
		return err
	}
	return s.persistAuditLogs()
}

func (s *Store) CompletePasswordReset(ctx context.Context, rawToken, passwordHash, ipAddress string) error {
	if rawToken == "" || passwordHash == "" {
		return errors.New("token and password hash are required")
	}
	if s.usesPostgresAuth() {
		return s.pgCompletePasswordReset(ctx, rawToken, passwordHash, ipAddress)
	}
	now := time.Now().UTC()
	tokenHash := hashToken(rawToken)
	s.mu.Lock()
	defer s.mu.Unlock()
	var token *models.AuthToken
	for _, candidate := range s.authTokens {
		if candidate.Purpose == models.AuthTokenPurposePasswordReset && candidate.TokenHash == tokenHash {
			token = candidate
			break
		}
	}
	if token == nil {
		return errors.New("token not found")
	}
	if token.UsedAt != nil {
		return errors.New("token already used")
	}
	if !token.ExpiresAt.After(now) {
		return errors.New("token expired")
	}
	user := s.users[token.UserID]
	if user == nil {
		return errors.New("user not found")
	}
	token.UsedAt = &now
	token.UsedIP = ipAddress
	user.PasswordHash = passwordHash
	user.UpdatedAt = now
	s.passwords[user.ID] = append(s.passwords[user.ID], &models.PasswordHistoryEntry{UserID: user.ID, PasswordHash: passwordHash, CreatedAt: now})
	if len(s.passwords[user.ID]) > 5 {
		s.passwords[user.ID] = s.passwords[user.ID][len(s.passwords[user.ID])-5:]
	}
	for _, session := range s.auth {
		if session.UserID == user.ID && session.RevokedAt == nil {
			revokedAt := now
			session.RevokedAt = &revokedAt
		}
	}
	s.audit = append(s.audit,
		&models.AuditLog{ID: newUUID(), ActorID: user.ID, Action: "auth.token.consumed." + models.AuthTokenPurposePasswordReset, TargetID: token.ID, IPAddress: ipAddress, CreatedAt: now},
		&models.AuditLog{ID: newUUID(), ActorID: user.ID, Action: "auth.password.reset", TargetID: user.ID, IPAddress: ipAddress, CreatedAt: now},
		&models.AuditLog{ID: newUUID(), ActorID: user.ID, Action: "auth.sessions.revoked", TargetID: user.ID, IPAddress: ipAddress, Metadata: map[string]string{"reason": "password_reset"}, CreatedAt: now},
	)
	if err := s.persistUsers(); err != nil {
		return err
	}
	if err := s.persistAuthHardening(); err != nil {
		return err
	}
	return s.persistAuditLogs()
}

func (s *Store) PasswordHistory(ctx context.Context, userID string) ([]*models.PasswordHistoryEntry, error) {
	if s.usesPostgresAuth() {
		return s.pgPasswordHistory(ctx, userID)
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	return append([]*models.PasswordHistoryEntry(nil), s.passwords[userID]...), nil
}

func (s *Store) GetMFASettings(ctx context.Context, userID string) (*models.MFASettings, error) {
	if s.usesPostgresAuth() {
		return s.pgGetMFASettings(ctx, userID)
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	setting := s.mfa[userID]
	if setting == nil {
		return &models.MFASettings{UserID: userID}, nil
	}
	copy := *setting
	copy.RecoveryCodeHashes = append([]string(nil), setting.RecoveryCodeHashes...)
	return &copy, nil
}

func (s *Store) SaveMFASettings(ctx context.Context, setting *models.MFASettings, actorID, ipAddress string) error {
	if setting == nil || setting.UserID == "" {
		return errors.New("mfa setting requires user id")
	}
	if s.usesPostgresAuth() {
		return s.pgSaveMFASettings(ctx, setting, actorID, ipAddress)
	}
	now := time.Now().UTC()
	setting.UpdatedAt = now
	s.mu.Lock()
	defer s.mu.Unlock()
	s.mfa[setting.UserID] = setting
	s.audit = append(s.audit, &models.AuditLog{
		ID:        newUUID(),
		ActorID:   actorID,
		Action:    "auth.mfa.updated",
		TargetID:  setting.UserID,
		IPAddress: ipAddress,
		Metadata:  map[string]string{"enabled": fmt.Sprintf("%t", setting.Enabled)},
		CreatedAt: now,
	})
	if err := s.persistAuthHardening(); err != nil {
		return err
	}
	return s.persistAuditLogs()
}

func (s *Store) ConsumeRecoveryCode(ctx context.Context, userID, codeHash, ipAddress string) (bool, error) {
	if s.usesPostgresAuth() {
		return s.pgConsumeRecoveryCode(ctx, userID, codeHash, ipAddress)
	}
	now := time.Now().UTC()
	s.mu.Lock()
	defer s.mu.Unlock()
	setting := s.mfa[userID]
	if setting == nil || !setting.Enabled {
		return false, nil
	}
	remaining := setting.RecoveryCodeHashes[:0]
	used := false
	for _, stored := range setting.RecoveryCodeHashes {
		if stored == codeHash && !used {
			used = true
			continue
		}
		remaining = append(remaining, stored)
	}
	if used {
		setting.RecoveryCodeHashes = remaining
		setting.UpdatedAt = now
		s.audit = append(s.audit, &models.AuditLog{
			ID:        newUUID(),
			ActorID:   userID,
			Action:    "auth.mfa.recovery_code.used",
			TargetID:  userID,
			IPAddress: ipAddress,
			CreatedAt: now,
		})
		if err := s.persistAuthHardening(); err != nil {
			return false, err
		}
		if err := s.persistAuditLogs(); err != nil {
			return false, err
		}
	}
	return used, nil
}

func (s *Store) AppendAuditLog(ctx context.Context, actorID, action, targetID, ipAddress string, metadata map[string]string) error {
	if s.usesPostgresAuth() {
		return s.pgAppendAudit(ctx, actorID, action, targetID, ipAddress, metadata)
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	s.audit = append(s.audit, &models.AuditLog{
		ID:        newUUID(),
		ActorID:   actorID,
		Action:    action,
		TargetID:  targetID,
		Metadata:  metadata,
		IPAddress: ipAddress,
		CreatedAt: time.Now().UTC(),
	})
	return s.persistAuditLogs()
}

func (s *Store) GetAuditLogs(ctx context.Context, limit int) ([]*models.AuditLog, error) {
	if limit <= 0 || limit > 200 {
		limit = 100
	}
	if s.usesPostgresAuth() {
		return s.pgGetAuditLogs(ctx, limit)
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	start := len(s.audit) - limit
	if start < 0 {
		start = 0
	}
	logs := append([]*models.AuditLog(nil), s.audit[start:]...)
	sort.SliceStable(logs, func(i, j int) bool {
		return logs[i].CreatedAt.After(logs[j].CreatedAt)
	})
	return logs, nil
}

func (s *Store) ListUsers(ctx context.Context) ([]*models.User, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	users := make([]*models.User, 0, len(s.users))
	for _, user := range s.users {
		users = append(users, user)
	}
	sort.SliceStable(users, func(i, j int) bool {
		return users[i].CreatedAt.After(users[j].CreatedAt)
	})
	return users, nil
}

func (s *Store) ApproveKYC(ctx context.Context, actorID, userID, ipAddress string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	user, ok := s.users[userID]
	if !ok {
		return errors.New("user not found")
	}
	user.KYCStatus = "approved"
	s.recomputeTrustLocked(userID, "kyc_approved")
	s.audit = append(s.audit, &models.AuditLog{
		ID:        newUUID(),
		ActorID:   actorID,
		Action:    "identity.kyc.approved",
		TargetID:  userID,
		IPAddress: ipAddress,
		CreatedAt: time.Now().UTC(),
	})
	if err := s.persistUsers(); err != nil {
		return err
	}
	if err := s.persistProgression(); err != nil {
		return err
	}
	return s.persistAuditLogs()
}

func (s *Store) Close(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.persistUsers(); err != nil {
		return err
	}
	if err := s.persistWallets(); err != nil {
		return err
	}
	if err := s.persistSessions(); err != nil {
		return err
	}
	if err := s.persistLedger(); err != nil {
		return err
	}
	if err := s.persistJobs(); err != nil {
		return err
	}
	if err := s.persistWorkerHealth(); err != nil {
		return err
	}
	if err := s.persistBackupRecords(); err != nil {
		return err
	}
	if err := s.persistAuthHardening(); err != nil {
		return err
	}
	if err := s.persistPaymentState(); err != nil {
		return err
	}
	if err := s.persistSnapshotLocked(ctx); err != nil {
		return err
	}
	if s.pg != nil {
		return s.pg.Close()
	}
	return nil
}

func (s *Store) DataDir() string {
	return s.dataDir
}

func (s *Store) ConfigureRuntime(settings *config.RuntimeSettings) {
	if settings == nil {
		return
	}
	s.mu.Lock()
	s.settings = settings
	s.mu.Unlock()
}

func (s *Store) Redis() saredis.Client {
	return s.redis
}

func (s *Store) ObjectStore() storage.ObjectStore {
	return s.objects
}

func (s *Store) ExportSnapshotJSON(ctx context.Context) ([]byte, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	return json.MarshalIndent(s.snapshotLocked(), "", "  ")
}

func (s *Store) setRedisJSON(ctx context.Context, key string, value any, ttl time.Duration) error {
	if s.redis == nil {
		return nil
	}
	data, err := json.Marshal(value)
	if err != nil {
		return err
	}
	return s.redis.Set(ctx, key, string(data), ttl)
}

func (s *Store) isConfiguredSuperAdminEmailLocked(email string) bool {
	normalized := strings.ToLower(strings.TrimSpace(email))
	for _, candidate := range s.settings.Admin.SuperAdminEmails {
		if normalized == strings.ToLower(strings.TrimSpace(candidate)) {
			return true
		}
	}
	return false
}

func (s *Store) UpdateUserRole(ctx context.Context, actorID, targetID, role, ipAddress string) (*models.User, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	actor := s.users[actorID]
	if actor == nil || actor.Role != models.RoleSuperAdmin {
		return nil, errors.New("super admin role required")
	}
	target := s.users[targetID]
	if target == nil {
		return nil, errors.New("user not found")
	}
	if s.isConfiguredSuperAdminEmailLocked(target.Email) && role != models.RoleSuperAdmin {
		return nil, errors.New("configured super admin cannot be demoted")
	}
	if role == "" {
		return nil, errors.New("role is required")
	}
	target.Role = role
	if role != models.RolePlayer {
		target.HiddenFromPublic = true
	}
	s.audit = append(s.audit, &models.AuditLog{
		ID:        newUUID(),
		ActorID:   actorID,
		Action:    "admin.role.updated",
		TargetID:  targetID,
		IPAddress: ipAddress,
		Metadata:  map[string]string{"role": role},
		CreatedAt: time.Now().UTC(),
	})
	if err := s.persistUsers(); err != nil {
		return nil, err
	}
	if err := s.persistAuditLogs(); err != nil {
		return nil, err
	}
	copyUser := *target
	return &copyUser, nil
}

func (s *Store) SuspendAdmin(ctx context.Context, actorID, targetID, ipAddress string) (*models.User, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	actor := s.users[actorID]
	if actor == nil || actor.Role != models.RoleSuperAdmin {
		return nil, errors.New("super admin role required")
	}
	target := s.users[targetID]
	if target == nil {
		return nil, errors.New("user not found")
	}
	if s.isConfiguredSuperAdminEmailLocked(target.Email) {
		return nil, errors.New("configured super admin cannot be suspended")
	}
	target.Role = models.RolePlayer
	target.HiddenFromPublic = true
	s.audit = append(s.audit, &models.AuditLog{
		ID:        newUUID(),
		ActorID:   actorID,
		Action:    "admin.suspended",
		TargetID:  targetID,
		IPAddress: ipAddress,
		CreatedAt: time.Now().UTC(),
	})
	if err := s.persistUsers(); err != nil {
		return nil, err
	}
	if err := s.persistAuditLogs(); err != nil {
		return nil, err
	}
	copyUser := *target
	return &copyUser, nil
}

func (s *Store) ResetAdminMFA(ctx context.Context, actorID, targetID, ipAddress string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	actor := s.users[actorID]
	if actor == nil || actor.Role != models.RoleSuperAdmin {
		return errors.New("super admin role required")
	}
	target := s.users[targetID]
	if target == nil {
		return errors.New("user not found")
	}
	delete(s.mfa, targetID)
	s.audit = append(s.audit, &models.AuditLog{
		ID:        newUUID(),
		ActorID:   actorID,
		Action:    "admin.mfa.reset",
		TargetID:  targetID,
		IPAddress: ipAddress,
		CreatedAt: time.Now().UTC(),
	})
	if err := s.persistAuthHardening(); err != nil {
		return err
	}
	return s.persistAuditLogs()
}

func (s *Store) EnqueueJob(ctx context.Context, jobType string, payload map[string]string, runAfter time.Time) (*models.BackgroundJob, error) {
	if jobType == "" {
		return nil, errors.New("job type is required")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now().UTC()
	if runAfter.IsZero() {
		runAfter = now
	}
	copiedPayload := map[string]string{}
	for key, value := range payload {
		copiedPayload[key] = value
	}
	job := &models.BackgroundJob{
		ID:          id.New("job"),
		Type:        jobType,
		Status:      models.JobStatusQueued,
		Payload:     copiedPayload,
		MaxAttempts: s.settings.Workers.MaxAttempts,
		RunAfter:    runAfter.UTC(),
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	if job.MaxAttempts <= 0 {
		job.MaxAttempts = 3
	}
	s.jobs[job.ID] = job
	if err := s.persistJobs(); err != nil {
		return nil, err
	}
	if err := s.setRedisJSON(ctx, "queue:"+job.Type+":"+job.ID, job, time.Until(job.RunAfter)+24*time.Hour); err != nil {
		return nil, err
	}
	copyJob := *job
	return &copyJob, nil
}

func (s *Store) ListJobs(ctx context.Context, status string) ([]*models.BackgroundJob, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	jobs := make([]*models.BackgroundJob, 0, len(s.jobs))
	for _, job := range s.jobs {
		if status != "" && job.Status != status {
			continue
		}
		copyJob := *job
		copyJob.Payload = copyStringMap(job.Payload)
		jobs = append(jobs, &copyJob)
	}
	sort.SliceStable(jobs, func(i, j int) bool {
		return jobs[i].CreatedAt.Before(jobs[j].CreatedAt)
	})
	return jobs, nil
}

func (s *Store) ClaimNextJob(ctx context.Context, worker string, jobTypes []string, now time.Time) (*models.BackgroundJob, error) {
	locked, err := s.redis.Lock(ctx, "jobs:claim", 15*time.Second)
	if err != nil {
		return nil, err
	}
	if !locked {
		return nil, nil
	}
	defer s.redis.Unlock(ctx, "jobs:claim")

	s.mu.Lock()
	defer s.mu.Unlock()
	if now.IsZero() {
		now = time.Now().UTC()
	}
	allowed := map[string]bool{}
	for _, jobType := range jobTypes {
		allowed[jobType] = true
	}
	var selected *models.BackgroundJob
	for _, job := range s.jobs {
		if job.Status != models.JobStatusQueued || job.RunAfter.After(now) {
			continue
		}
		if len(allowed) > 0 && !allowed[job.Type] {
			continue
		}
		if selected == nil || job.CreatedAt.Before(selected.CreatedAt) {
			selected = job
		}
	}
	if selected == nil {
		return nil, nil
	}
	selected.Status = models.JobStatusRunning
	selected.Worker = worker
	selected.Attempts++
	started := now.UTC()
	selected.StartedAt = &started
	selected.UpdatedAt = started
	if err := s.persistJobs(); err != nil {
		return nil, err
	}
	copyJob := *selected
	copyJob.Payload = copyStringMap(selected.Payload)
	return &copyJob, nil
}

func (s *Store) CompleteJob(ctx context.Context, jobID, artifact string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	job := s.jobs[jobID]
	if job == nil {
		return errors.New("job not found")
	}
	now := time.Now().UTC()
	job.Status = models.JobStatusCompleted
	job.CompletedAt = &now
	job.ResultArtifact = artifact
	job.UpdatedAt = now
	return s.persistJobs()
}

func (s *Store) FailJob(ctx context.Context, jobID string, failure error) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	job := s.jobs[jobID]
	if job == nil {
		return errors.New("job not found")
	}
	now := time.Now().UTC()
	job.LastError = failure.Error()
	job.UpdatedAt = now
	if job.Attempts < job.MaxAttempts {
		job.Status = models.JobStatusQueued
		job.Worker = ""
		job.RunAfter = now.Add(time.Duration(job.Attempts*job.Attempts) * time.Minute)
		job.StartedAt = nil
	} else {
		job.Status = models.JobStatusFailed
		job.CompletedAt = &now
	}
	return s.persistJobs()
}

func (s *Store) RetryJob(ctx context.Context, jobID string) (*models.BackgroundJob, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	job := s.jobs[jobID]
	if job == nil {
		return nil, errors.New("job not found")
	}
	now := time.Now().UTC()
	job.Status = models.JobStatusQueued
	job.RunAfter = now
	job.StartedAt = nil
	job.CompletedAt = nil
	job.Worker = ""
	job.LastError = ""
	job.UpdatedAt = now
	if err := s.persistJobs(); err != nil {
		return nil, err
	}
	copyJob := *job
	copyJob.Payload = copyStringMap(job.Payload)
	return &copyJob, nil
}

func (s *Store) CancelJob(ctx context.Context, jobID string) (*models.BackgroundJob, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	job := s.jobs[jobID]
	if job == nil {
		return nil, errors.New("job not found")
	}
	now := time.Now().UTC()
	job.Status = models.JobStatusCancelled
	job.CompletedAt = &now
	job.UpdatedAt = now
	if err := s.persistJobs(); err != nil {
		return nil, err
	}
	copyJob := *job
	copyJob.Payload = copyStringMap(job.Payload)
	return &copyJob, nil
}

func (s *Store) QueueStats(ctx context.Context) (*models.JobQueueStats, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.queueStatsLocked(), nil
}

func (s *Store) SetWorkerHealth(ctx context.Context, name, status, lastError string) error {
	if name == "" {
		return errors.New("worker name is required")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.workerHealth[name] = &models.WorkerHealth{
		Name:      name,
		Status:    status,
		LastSeen:  time.Now().UTC(),
		LastError: lastError,
	}
	return s.persistWorkerHealth()
}

func (s *Store) AddBackupRecord(ctx context.Context, record *models.BackupRecord) error {
	if record == nil {
		return errors.New("backup record is required")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.backups = append(s.backups, record)
	return s.persistBackupRecords()
}

func (s *Store) ListBackupRecords(ctx context.Context) ([]*models.BackupRecord, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	backups := make([]*models.BackupRecord, 0, len(s.backups))
	for _, backup := range s.backups {
		copyBackup := *backup
		backups = append(backups, &copyBackup)
	}
	sort.SliceStable(backups, func(i, j int) bool {
		return backups[i].StartedAt.After(backups[j].StartedAt)
	})
	return backups, nil
}

func (s *Store) SystemHealth(ctx context.Context) (*models.SystemHealth, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var mem runtime.MemStats
	runtime.ReadMemStats(&mem)
	queueStats := s.queueStatsLocked()
	workerHealth := map[string]string{}
	for name, health := range s.workerHealth {
		workerHealth[name] = health.Status
	}
	backupStatus := "none"
	if len(s.backups) > 0 {
		backupStatus = s.backups[len(s.backups)-1].Status
	}
	return &models.SystemHealth{
		APIStatus:             "ok",
		DatabaseStatus:        "ok",
		QueueStatus:           "ok",
		CacheHealth:           "ok",
		StorageUsageBytes:     dirSizeBestEffort(s.dataDir),
		ReplayGenerationQueue: queueStats.ByType[models.JobReplayExport],
		ActiveMatches:         s.activeMatchCountLocked(),
		PlayersOnline:         s.activeAuthCountLocked(),
		MemoryUsageBytes:      mem.Alloc,
		BackupStatus:          backupStatus,
		DeploymentVersion:     "local",
		MaintenanceEnabled:    s.settings.Maintenance.Enabled,
		MaintenanceMessage:    s.settings.Maintenance.Message,
		WorkerHealth:          workerHealth,
		QueueStats:            queueStats,
		CheckedAt:             time.Now().UTC(),
	}, nil
}

func (s *Store) queueStatsLocked() *models.JobQueueStats {
	stats := &models.JobQueueStats{
		ByType:       map[string]int{},
		WorkerStatus: map[string]string{},
		UpdatedAt:    time.Now().UTC(),
	}
	var totalProcessing float64
	var processed int
	for _, job := range s.jobs {
		stats.ByType[job.Type]++
		stats.RetryCount += job.Attempts
		switch job.Status {
		case models.JobStatusQueued:
			stats.PendingJobs++
		case models.JobStatusRunning:
			stats.RunningJobs++
		case models.JobStatusCompleted:
			stats.CompletedJobs++
		case models.JobStatusFailed:
			stats.FailedJobs++
		case models.JobStatusCancelled:
			stats.CancelledJobs++
		}
		if job.StartedAt != nil && job.CompletedAt != nil {
			totalProcessing += job.CompletedAt.Sub(*job.StartedAt).Seconds()
			processed++
		}
	}
	if processed > 0 {
		stats.AverageProcessingSeconds = totalProcessing / float64(processed)
	}
	for name, health := range s.workerHealth {
		stats.WorkerStatus[name] = health.Status
	}
	return stats
}

func (s *Store) activeMatchCountLocked() int {
	count := 0
	for _, match := range s.pvpMatches {
		if match.Status == "active" {
			count++
		}
	}
	return count
}

func (s *Store) activeAuthCountLocked() int {
	now := time.Now().UTC()
	count := 0
	for _, session := range s.auth {
		if session.RevokedAt == nil && session.ExpiresAt.After(now) {
			count++
		}
	}
	return count
}

func copyStringMap(values map[string]string) map[string]string {
	if values == nil {
		return nil
	}
	copied := make(map[string]string, len(values))
	for key, value := range values {
		copied[key] = value
	}
	return copied
}

func dirSizeBestEffort(path string) int64 {
	var total int64
	_ = filepath.Walk(path, func(_ string, info os.FileInfo, err error) error {
		if err != nil || info == nil || info.IsDir() {
			return nil
		}
		total += info.Size()
		return nil
	})
	return total
}

func (s *Store) CreateUser(ctx context.Context, user *models.User) error {
	if user == nil {
		return errors.New("user is required")
	}
	if s.usesPostgresAuth() {
		return s.pgCreateUser(ctx, user)
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	normalizeNewUser(user)
	for _, existing := range s.users {
		if existing.Email == user.Email {
			if existing.PasswordHash == "" && user.PasswordHash != "" {
				existing.PasswordHash = user.PasswordHash
				s.passwords[existing.ID] = append(s.passwords[existing.ID], &models.PasswordHistoryEntry{
					UserID:        existing.ID,
					PasswordHash:  user.PasswordHash,
					PasswordStamp: "",
					CreatedAt:     time.Now().UTC(),
				})
				if s.isConfiguredSuperAdminEmailLocked(existing.Email) {
					existing.Role = models.RoleSuperAdmin
					existing.HiddenFromPublic = true
				}
				if existing.KYCStatus == "" {
					existing.KYCStatus = "unverified"
				}
				user.ID = existing.ID
				user.Role = existing.Role
				user.HiddenFromPublic = existing.HiddenFromPublic
				user.EmailVerified = existing.EmailVerified
				user.KYCStatus = existing.KYCStatus
				user.CreatedAt = existing.CreatedAt
				if err := s.persistUsers(); err != nil {
					return err
				}
				if err := s.persistAuthHardening(); err != nil {
					return err
				}
				return s.persistAuditLogs()
			}
			return errors.New("email already exists")
		}
	}
	if user.Role == "" {
		user.Role = "player"
	}
	if s.isConfiguredSuperAdminEmailLocked(user.Email) {
		user.Role = models.RoleSuperAdmin
		user.HiddenFromPublic = true
	} else if len(s.users) == 0 {
		user.Role = "admin"
		user.HiddenFromPublic = true
	}

	user.CreatedAt = time.Now().UTC()
	s.users[user.ID] = user
	if user.PasswordHash != "" {
		s.passwords[user.ID] = append(s.passwords[user.ID], &models.PasswordHistoryEntry{
			UserID:        user.ID,
			PasswordHash:  user.PasswordHash,
			PasswordStamp: "",
			CreatedAt:     user.CreatedAt,
		})
	}
	s.ensureProgressionLocked(user.ID)
	if err := s.persistUsers(); err != nil {
		return err
	}
	if err := s.persistAuthHardening(); err != nil {
		return err
	}
	if err := s.persistProgression(); err != nil {
		return err
	}

	if _, ok := s.wallets[user.ID]; !ok {
		wallet := &models.Wallet{
			UserID:             user.ID,
			LiveBalance:        0,
			LiveLockedBalance:  0,
			DemoBalance:        1000,
			DemoLockedBalance:  0,
			PendingWithdrawals: 0,
			BonusBalance:       0,
		}
		s.wallets[user.ID] = wallet
		if err := s.persistWallets(); err != nil {
			return err
		}

		initialEntry := &models.LedgerEntry{
			ID:              newUUID(),
			UserID:          user.ID,
			TransactionType: models.TransactionTypeDeposit,
			Amount:          1000,
			BalanceBefore:   0,
			BalanceAfter:    wallet.DemoBalance,
			Currency:        "USD",
			Reference:       "initial-demo-credit",
			CreatedAt:       time.Now().UTC(),
		}
		s.ledger[user.ID] = append(s.ledger[user.ID], initialEntry)
		if err := s.persistLedger(); err != nil {
			return err
		}
	}

	return nil
}

func (s *Store) GetUserByEmail(ctx context.Context, email string) (*models.User, error) {
	if s.usesPostgresAuth() {
		return s.pgGetUserByEmail(ctx, email)
	}
	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, user := range s.users {
		if user.Email == email {
			return user, nil
		}
	}
	return nil, errors.New("user not found")
}

func (s *Store) GetUserByID(ctx context.Context, userID string) (*models.User, error) {
	if s.usesPostgresAuth() {
		return s.pgGetUserByID(ctx, userID)
	}
	s.mu.RLock()
	defer s.mu.RUnlock()

	user, ok := s.users[userID]
	if !ok {
		return nil, errors.New("user not found")
	}
	return user, nil
}

func (s *Store) GetWalletByUserID(ctx context.Context, userID string) (*models.Wallet, error) {
	s.mu.RLock()
	wallet, ok := s.wallets[userID]
	s.mu.RUnlock()
	if ok {
		return wallet, nil
	}

	wallet = &models.Wallet{
		UserID:             userID,
		LiveBalance:        0,
		LiveLockedBalance:  0,
		DemoBalance:        1000,
		DemoLockedBalance:  0,
		PendingWithdrawals: 0,
		BonusBalance:       0,
	}
	s.mu.Lock()
	s.wallets[userID] = wallet
	s.mu.Unlock()
	if err := s.persistWallets(); err != nil {
		return nil, err
	}

	initialEntry := &models.LedgerEntry{
		ID:              newUUID(),
		UserID:          userID,
		TransactionType: models.TransactionTypeDeposit,
		Amount:          1000,
		BalanceBefore:   0,
		BalanceAfter:    wallet.DemoBalance,
		Currency:        "USD",
		Reference:       "initial-demo-credit",
		CreatedAt:       time.Now().UTC(),
	}
	s.mu.Lock()
	s.ledger[userID] = append(s.ledger[userID], initialEntry)
	s.mu.Unlock()
	if err := s.persistLedger(); err != nil {
		return nil, err
	}

	return wallet, nil
}

func (s *Store) GetSessionByID(ctx context.Context, sessionID string) (*models.GameSession, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	session, ok := s.sessions[sessionID]
	if !ok {
		return nil, errors.New("session not found")
	}
	return session, nil
}

func (s *Store) GetSessionsByUserID(ctx context.Context, userID string) ([]*models.GameSession, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	sessions := make([]*models.GameSession, 0)
	for _, session := range s.sessions {
		if session.UserID == userID {
			sessions = append(sessions, session)
		}
	}
	return sessions, nil
}

func replayEventsFromSession(session *models.GameSession) []models.ReplayEvent {
	events := make([]models.ReplayEvent, 0, len(session.Moves)+len(session.Clicks)+2)
	events = append(events, models.ReplayEvent{Type: "session_started", AtMs: 0})
	for i := range session.Moves {
		move := session.Moves[i]
		events = append(events, models.ReplayEvent{
			Type: "move",
			AtMs: int64(move.Timestamp.Sub(session.CreatedAt).Milliseconds()),
			Move: &move,
		})
	}
	for i := range session.Clicks {
		click := session.Clicks[i]
		eventType := "line_click_blocked"
		if click.Success {
			eventType = "line_click_success"
		}
		events = append(events, models.ReplayEvent{
			Type:   eventType,
			AtMs:   int64(click.Timestamp.Sub(session.CreatedAt).Milliseconds()),
			LineID: click.LineID,
			Click:  &click,
		})
	}
	if session.CompletedAt != nil {
		events = append(events, models.ReplayEvent{Type: "session_completed", AtMs: int64(session.CompletedAt.Sub(session.CreatedAt).Milliseconds()), Metadata: map[string]string{"outcome": session.Outcome}})
	}
	sort.SliceStable(events, func(i, j int) bool {
		return events[i].AtMs < events[j].AtMs
	})
	return events
}

func signReplay(report *models.ReplayReport) {
	signedAt := time.Now().UTC()
	report.ReplaySignedAt = &signedAt
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
	report.ReplaySignature = base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}

func replayReportFromSession(session *models.GameSession) *models.ReplayReport {
	replaySettings := config.Runtime().Replay
	if len(session.Lines) > 0 {
		duration := 0.0
		if session.CompletedAt != nil {
			duration = session.CompletedAt.Sub(session.CreatedAt).Seconds()
		}
		moveCount := len(session.Clicks)
		successes := 0
		failures := 0
		for _, click := range session.Clicks {
			if click.Success {
				successes++
			} else {
				failures++
			}
		}
		efficiency := 0.0
		if moveCount > 0 {
			efficiency = float64(successes) / float64(moveCount)
		}
		flags := make([]string, 0)
		if moveCount > 0 && float64(failures)/float64(len(session.Lines)) > replaySettings.HighFailedClickPercent {
			flags = append(flags, "high_failed_click_rate")
		}
		if session.IsFinished && duration > 0 && moveCount >= 10 && duration/float64(moveCount) < replaySettings.FastClickSeconds {
			flags = append(flags, "click_timing_too_fast")
		}
		if session.PuzzleSeed == "" || session.GenerationHash == "" || session.GenerationNonce == "" || session.DifficultyProfile == nil {
			flags = append(flags, "missing_reconstruction_metadata")
		} else {
			reconstructed := game.GenerateLinePuzzleFromProfile(session.PuzzleSeed, *session.DifficultyProfile)
			if !sameLinePuzzle(reconstructed, session.Lines) {
				flags = append(flags, "puzzle_reconstruction_mismatch")
			}
		}
		status := "pending"
		if session.IsFinished {
			status = "verified"
			if len(flags) > 0 {
				status = "flagged"
			}
		}
		report := &models.ReplayReport{
			SessionID:          session.ID,
			UserID:             session.UserID,
			GameType:           session.GameType,
			Mode:               session.Mode,
			HouseTier:          session.HouseTier,
			Difficulty:         session.Difficulty,
			DifficultyRating:   session.DifficultyRating,
			DifficultyProfile:  session.DifficultyProfile,
			PuzzleSeed:         session.PuzzleSeed,
			GenerationNonce:    session.GenerationNonce,
			GenerationHash:     session.GenerationHash,
			PuzzleVersion:      session.PuzzleVersion,
			Outcome:            session.Outcome,
			Stake:              session.Stake,
			Reward:             session.Reward,
			IsFinished:         session.IsFinished,
			IsValidRoute:       session.Outcome == "win" || session.Outcome == "calibrated",
			MoveCount:          moveCount,
			ShortestPathLength: len(session.Lines),
			RouteEfficiency:    efficiency,
			DurationSeconds:    duration,
			IntegrityStatus:    status,
			Flags:              flags,
			Lines:              session.Lines,
			Clicks:             session.Clicks,
			PlaybackEvents:     replayEventsFromSession(session),
			CreatedAt:          session.CreatedAt,
			CompletedAt:        session.CompletedAt,
		}
		signReplay(report)
		return report
	}

	maze := &game.Maze{
		Width:  session.Width,
		Height: session.Height,
		Cells:  session.MazeCells,
		StartX: session.StartX,
		StartY: session.StartY,
		EndX:   session.EndX,
		EndY:   session.EndY,
	}

	directions := make([]string, 0, len(session.Moves))
	for _, move := range session.Moves {
		directions = append(directions, move.Direction)
	}
	valid, _, _ := game.ValidateMazeMoves(maze, directions)
	shortest := game.ShortestPathLength(maze)
	moveCount := len(session.Moves)
	efficiency := 0.0
	if moveCount > 0 && shortest > 0 {
		efficiency = float64(shortest) / float64(moveCount)
	}

	duration := 0.0
	if session.CompletedAt != nil {
		duration = session.CompletedAt.Sub(session.CreatedAt).Seconds()
	}

	flags := make([]string, 0)
	if session.IsFinished && session.Outcome == "win" && !valid {
		flags = append(flags, "winning_replay_failed_validation")
	}
	if session.IsFinished && session.Outcome == "win" && moveCount > 0 && shortest > 0 && moveCount < shortest {
		flags = append(flags, "move_count_below_shortest_path")
	}
	if session.IsFinished && duration > 0 && moveCount >= 5 && duration/float64(moveCount) < replaySettings.FastClickSeconds {
		flags = append(flags, "input_timing_too_fast")
	}

	status := "pending"
	if session.IsFinished {
		status = "verified"
		if len(flags) > 0 {
			status = "flagged"
		}
	}

	report := &models.ReplayReport{
		SessionID:          session.ID,
		UserID:             session.UserID,
		GameType:           session.GameType,
		Mode:               session.Mode,
		HouseTier:          session.HouseTier,
		Difficulty:         session.Difficulty,
		DifficultyRating:   session.DifficultyRating,
		DifficultyProfile:  session.DifficultyProfile,
		PuzzleSeed:         session.PuzzleSeed,
		GenerationNonce:    session.GenerationNonce,
		GenerationHash:     session.GenerationHash,
		PuzzleVersion:      session.PuzzleVersion,
		Outcome:            session.Outcome,
		Stake:              session.Stake,
		Reward:             session.Reward,
		IsFinished:         session.IsFinished,
		IsValidRoute:       valid,
		MoveCount:          moveCount,
		ShortestPathLength: shortest,
		RouteEfficiency:    efficiency,
		DurationSeconds:    duration,
		IntegrityStatus:    status,
		Flags:              flags,
		MazeCells:          session.MazeCells,
		Moves:              session.Moves,
		PlaybackEvents:     replayEventsFromSession(session),
		CreatedAt:          session.CreatedAt,
		CompletedAt:        session.CompletedAt,
	}
	signReplay(report)
	return report
}

func sameLinePuzzle(expected []models.ArrowLine, actual []models.ArrowLine) bool {
	if len(expected) != len(actual) {
		return false
	}
	for i := range expected {
		if expected[i].ID != actual[i].ID ||
			expected[i].Direction != actual[i].Direction ||
			expected[i].X != actual[i].X ||
			expected[i].Y != actual[i].Y ||
			expected[i].Length != actual[i].Length ||
			len(expected[i].Points) != len(actual[i].Points) ||
			len(expected[i].DependsOn) != len(actual[i].DependsOn) {
			return false
		}
		for j := range expected[i].Points {
			if expected[i].Points[j].X != actual[i].Points[j].X ||
				expected[i].Points[j].Y != actual[i].Points[j].Y {
				return false
			}
		}
		for j := range expected[i].DependsOn {
			if expected[i].DependsOn[j] != actual[i].DependsOn[j] {
				return false
			}
		}
	}
	return true
}

func (s *Store) GetReplayReport(ctx context.Context, sessionID string) (*models.ReplayReport, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	session, ok := s.sessions[sessionID]
	if !ok {
		return nil, errors.New("session not found")
	}
	started := time.Now()
	report := replayReportFromSession(session)
	s.recordReplayReconstructionLocked(time.Since(started))
	if reviewCase, ok := s.reviewCases["game_session:"+sessionID]; ok {
		report.ReviewStatus = reviewCase.Status
	}
	_ = s.persistMetrics()
	return report, nil
}

func (s *Store) GetReplayReportsByUserID(ctx context.Context, userID string) ([]*models.ReplayReport, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	reports := make([]*models.ReplayReport, 0)
	for _, session := range s.sessions {
		if session.UserID == userID && session.IsFinished {
			report := replayReportFromSession(session)
			if reviewCase, ok := s.reviewCases["game_session:"+session.ID]; ok {
				report.ReviewStatus = reviewCase.Status
			}
			reports = append(reports, report)
		}
	}
	sort.SliceStable(reports, func(i, j int) bool {
		return reports[i].CreatedAt.After(reports[j].CreatedAt)
	})
	return reports, nil
}

func (s *Store) updateBaselineLocked(session *models.GameSession) {
	report := replayReportFromSession(session)
	moveSeconds := 0.0
	if report.MoveCount > 0 && report.DurationSeconds > 0 {
		moveSeconds = report.DurationSeconds / float64(report.MoveCount)
	}

	baseline := s.baselines[session.UserID]
	if baseline == nil {
		baseline = &models.BehavioralBaseline{
			UserID:     session.UserID,
			RiskSignal: "insufficient_data",
		}
		s.baselines[session.UserID] = baseline
	}

	runs := float64(baseline.CalibrationRuns)
	baseline.AverageEfficiency = ((baseline.AverageEfficiency * runs) + report.RouteEfficiency) / (runs + 1)
	baseline.AverageMoveSeconds = ((baseline.AverageMoveSeconds * runs) + moveSeconds) / (runs + 1)
	baseline.CalibrationRuns++
	if baseline.BestMoveCount == 0 || (report.MoveCount > 0 && report.MoveCount < baseline.BestMoveCount) {
		baseline.BestMoveCount = report.MoveCount
	}
	baseline.LastSessionID = session.ID
	if session.CompletedAt != nil {
		baseline.LastRunAt = *session.CompletedAt
	} else {
		baseline.LastRunAt = time.Now().UTC()
	}
	baseline.RiskSignal = "normal"
	if baseline.CalibrationRuns < 3 {
		baseline.RiskSignal = "insufficient_data"
	}
	if baseline.CalibrationRuns >= 3 && baseline.AverageMoveSeconds > 0 && moveSeconds > 0 && moveSeconds < baseline.AverageMoveSeconds*0.35 {
		baseline.RiskSignal = "major_timing_shift"
	}
	if report.RouteEfficiency >= 1 && report.MoveCount >= 5 && moveSeconds > 0 && moveSeconds < 0.08 {
		baseline.RiskSignal = "automation_suspected"
	}
}

func (s *Store) GetBaselineByUserID(ctx context.Context, userID string) (*models.BehavioralBaseline, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	baseline := s.baselines[userID]
	if baseline == nil {
		return &models.BehavioralBaseline{UserID: userID, RiskSignal: "insufficient_data"}, nil
	}
	copyBaseline := *baseline
	return &copyBaseline, nil
}

func (s *Store) RecordGameplayTelemetry(ctx context.Context, telemetry *models.GameplayTelemetry) error {
	if telemetry == nil {
		return nil
	}
	if telemetry.UserID == "" || telemetry.ScopeID == "" {
		return errors.New("telemetry userId and scopeId are required")
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	if telemetry.ID == "" {
		telemetry.ID = newUUID()
	}
	if telemetry.CollectedAt.IsZero() {
		telemetry.CollectedAt = time.Now().UTC()
	}
	if telemetry.PrivacyClassification == "" {
		telemetry.PrivacyClassification = s.settings.AntiBot.PrivacyClassification
	}
	s.telemetry[telemetry.ScopeID] = append(s.telemetry[telemetry.ScopeID], telemetry)
	return s.persistTelemetry()
}

func (s *Store) GetTelemetryByScope(ctx context.Context, scopeID string) ([]*models.GameplayTelemetry, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	entries := append([]*models.GameplayTelemetry(nil), s.telemetry[scopeID]...)
	return entries, nil
}

func (s *Store) ensureReviewCaseLocked(scope, scopeID, userID, reason string, flags []string) *models.ReviewCase {
	key := scope + ":" + scopeID
	if existing, ok := s.reviewCases[key]; ok {
		existing.Flags = flags
		existing.UpdatedAt = time.Now().UTC()
		return existing
	}
	now := time.Now().UTC()
	reviewCase := &models.ReviewCase{
		ID:        newUUID(),
		Scope:     scope,
		ScopeID:   scopeID,
		UserID:    userID,
		Status:    "PENDING_REVIEW",
		Reason:    reason,
		Flags:     flags,
		CreatedAt: now,
		UpdatedAt: now,
	}
	s.reviewCases[key] = reviewCase
	s.audit = append(s.audit, &models.AuditLog{
		ID:        newUUID(),
		ActorID:   "system",
		Action:    "review.case.created",
		TargetID:  reviewCase.ID,
		Metadata:  map[string]string{"scope": scope, "scopeId": scopeID, "reason": reason},
		CreatedAt: now,
	})
	return reviewCase
}

func validReviewTransition(current, next string) bool {
	allowed := map[string][]string{
		"PENDING_REVIEW": {"MANUAL_REVIEW", "APPROVED", "REJECTED"},
		"MANUAL_REVIEW":  {"APPROVED", "REJECTED"},
	}
	for _, candidate := range allowed[current] {
		if candidate == next {
			return true
		}
	}
	return current == next
}

func (s *Store) TransitionReviewCase(ctx context.Context, actorID, caseID, nextStatus, decision, ipAddress string) (*models.ReviewCase, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	var reviewCase *models.ReviewCase
	for _, candidate := range s.reviewCases {
		if candidate.ID == caseID {
			reviewCase = candidate
			break
		}
	}
	if reviewCase == nil {
		return nil, errors.New("review case not found")
	}
	if !validReviewTransition(reviewCase.Status, nextStatus) {
		return nil, fmt.Errorf("invalid review transition from %s to %s", reviewCase.Status, nextStatus)
	}
	reviewCase.Status = nextStatus
	reviewCase.Decision = decision
	reviewCase.ReviewerID = actorID
	reviewCase.UpdatedAt = time.Now().UTC()
	s.audit = append(s.audit, &models.AuditLog{
		ID:        newUUID(),
		ActorID:   actorID,
		Action:    "review.case.transitioned",
		TargetID:  reviewCase.ID,
		IPAddress: ipAddress,
		Metadata:  map[string]string{"status": nextStatus, "decision": decision},
		CreatedAt: time.Now().UTC(),
	})
	s.recomputeTrustLocked(reviewCase.UserID, "review_case_"+nextStatus)
	if err := s.persistReviewCases(); err != nil {
		return nil, err
	}
	if err := s.persistProgression(); err != nil {
		return nil, err
	}
	if err := s.persistAuditLogs(); err != nil {
		return nil, err
	}
	return reviewCase, nil
}

func (s *Store) ListReviewCases(ctx context.Context) ([]*models.ReviewCase, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	cases := make([]*models.ReviewCase, 0, len(s.reviewCases))
	for _, reviewCase := range s.reviewCases {
		copyCase := *reviewCase
		cases = append(cases, &copyCase)
	}
	sort.SliceStable(cases, func(i, j int) bool {
		return cases[i].UpdatedAt.After(cases[j].UpdatedAt)
	})
	return cases, nil
}

func (s *Store) Metrics(ctx context.Context) (*models.MetricsSnapshot, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.metrics == nil {
		return &models.MetricsSnapshot{}, nil
	}
	copyMetrics := *s.metrics
	return &copyMetrics, nil
}

func (s *Store) recordPuzzleGenerationLocked(duration time.Duration) {
	if s.metrics == nil {
		s.metrics = &models.MetricsSnapshot{}
	}
	s.metrics.PuzzleGenerationCount++
	s.metrics.TotalPuzzleGenerationMs += float64(duration.Microseconds()) / 1000
	s.metrics.UpdatedAt = time.Now().UTC()
}

func (s *Store) recordReplayReconstructionLocked(duration time.Duration) {
	if s.metrics == nil {
		s.metrics = &models.MetricsSnapshot{}
	}
	s.metrics.ReplayReconstructionCount++
	s.metrics.TotalReplayReconstructionMs += float64(duration.Microseconds()) / 1000
	s.metrics.UpdatedAt = time.Now().UTC()
}

func (s *Store) recordMatchmakingLocked(duration time.Duration) {
	if s.metrics == nil {
		s.metrics = &models.MetricsSnapshot{}
	}
	s.metrics.MatchmakingCount++
	s.metrics.TotalMatchmakingMs += float64(duration.Microseconds()) / 1000
	s.metrics.UpdatedAt = time.Now().UTC()
}

func (s *Store) recordCompletionLocked(session *models.GameSession) {
	if s.metrics == nil {
		s.metrics = &models.MetricsSnapshot{}
	}
	if session.CompletedAt != nil {
		s.metrics.CompletedMatchCount++
		s.metrics.TotalCompletionSeconds += session.CompletedAt.Sub(session.CreatedAt).Seconds()
	}
	for _, click := range session.Clicks {
		if !click.Success {
			s.metrics.TotalFailedClicks++
		}
	}
	s.metrics.UpdatedAt = time.Now().UTC()
}

func (s *Store) ListBaselines(ctx context.Context) ([]*models.BehavioralBaseline, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	baselines := make([]*models.BehavioralBaseline, 0, len(s.baselines))
	for _, baseline := range s.baselines {
		copyBaseline := *baseline
		baselines = append(baselines, &copyBaseline)
	}
	sort.SliceStable(baselines, func(i, j int) bool {
		return baselines[i].LastRunAt.After(baselines[j].LastRunAt)
	})
	return baselines, nil
}

func (s *Store) StartGameSession(ctx context.Context, session *models.GameSession) error {
	s.mu.Lock()

	wallet, ok := s.wallets[session.UserID]
	if !ok {
		s.mu.Unlock()
		return errors.New("wallet not found")
	}

	if session.Stake <= 0 && !session.Calibration {
		s.mu.Unlock()
		return errors.New("stake must be greater than zero")
	}

	session.ID = id.Session()
	session.CreatedAt = time.Now().UTC()
	session.IsFinished = false
	session.State = models.SessionStateCreated
	if err := transitionSessionState(session, models.SessionStateGenerating); err != nil {
		s.mu.Unlock()
		return err
	}
	if session.Mode == "" {
		session.Mode = "maze"
	}
	if session.RewardRate <= 0 {
		session.RewardRate = 0.8
	}

	if !session.Calibration {
		switch session.GameType {
		case "demo":
			availableDemo := calculateAvailableDemo(wallet)
			if availableDemo < session.Stake {
				s.mu.Unlock()
				return errors.New("insufficient available demo balance")
			}
			entry := &models.LedgerEntry{
				ID:              newUUID(),
				UserID:          session.UserID,
				TransactionType: models.TransactionTypeLock,
				Amount:          session.Stake,
				BalanceBefore:   availableDemo,
				Reference:       "game-stake",
				Currency:        "USD",
				Metadata:        map[string]string{"gameType": session.GameType},
				CreatedAt:       time.Now().UTC(),
			}
			wallet.DemoLockedBalance += session.Stake
			entry.BalanceAfter = calculateAvailableDemo(wallet)
			s.ledger[session.UserID] = append(s.ledger[session.UserID], entry)
		case "live":
			availableLive := calculateAvailableLive(wallet)
			if availableLive < session.Stake {
				s.mu.Unlock()
				return errors.New("insufficient available live balance")
			}
			entry := &models.LedgerEntry{
				ID:              newUUID(),
				UserID:          session.UserID,
				TransactionType: models.TransactionTypeLock,
				Amount:          session.Stake,
				BalanceBefore:   availableLive,
				Reference:       "game-stake",
				Currency:        "USD",
				Metadata:        map[string]string{"gameType": session.GameType},
				CreatedAt:       time.Now().UTC(),
			}
			wallet.LiveLockedBalance += session.Stake
			entry.BalanceAfter = calculateAvailableLive(wallet)
			s.ledger[session.UserID] = append(s.ledger[session.UserID], entry)
		default:
			s.mu.Unlock()
			return errors.New("invalid game type")
		}
	}

	mazeSize := 11 + (session.Difficulty * 2)
	if mazeSize < 11 {
		mazeSize = 11
	}
	if mazeSize > 23 {
		mazeSize = 23
	}
	maze := game.GenerateMaze(mazeSize, mazeSize)
	session.MazeCells = maze.Cells
	session.Width = maze.Width
	session.Height = maze.Height
	session.StartX = maze.StartX
	session.StartY = maze.StartY
	session.EndX = maze.EndX
	session.EndY = maze.EndY
	session.Moves = nil
	source := session.Mode
	if source == "" || source == "maze" {
		source = "game"
	}
	profile := s.difficultyProfileForUserLocked(session.UserID, source, session.HouseTier, "", session.Difficulty)
	if session.Calibration {
		profile = game.ProfileFromRating(6, "calibration")
	}
	session.DifficultyRating = profile.Rating
	session.DifficultyProfile = &profile
	session.PuzzleVersion = game.CurrentPuzzleVersion()
	puzzleMode := puzzle.ModePractice
	if session.Calibration {
		puzzleMode = puzzle.ModeTraining
	}
	service := s.puzzleServiceLocked()
	s.sessions[session.ID] = session
	s.mu.Unlock()

	generationStarted := time.Now()
	generated, err := service.Generate(ctx, puzzle.Request{
		Mode:              puzzleMode,
		Purpose:           source,
		MatchID:           session.ID,
		PlayerID:          session.UserID,
		Shared:            false,
		Level:             session.Difficulty,
		DifficultyProfile: profile,
		PuzzleVersion:     session.PuzzleVersion,
	})
	if err != nil {
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	existing, ok := s.sessions[session.ID]
	if ok && existing.State != models.SessionStateGenerating {
		return errors.New("game session state changed during puzzle generation")
	}
	session.DifficultyProfile = &generated.DifficultyProfile
	session.DifficultyRating = generated.DifficultyProfile.Rating
	session.PuzzleSeed = generated.Metadata.Seed
	session.GenerationNonce = generated.Metadata.Nonce
	session.GenerationHash = generated.Metadata.GenerationHash
	session.PuzzleMetadata = &generated.Metadata
	session.Lines = generated.Lines
	s.recordPuzzleGenerationLocked(time.Since(generationStarted))
	session.Clicks = nil
	if err := transitionSessionState(session, models.SessionStateReady); err != nil {
		return err
	}

	s.sessions[session.ID] = session
	if err := s.persistWallets(); err != nil {
		return err
	}
	if err := s.persistSessions(); err != nil {
		return err
	}
	if err := s.persistLedger(); err != nil {
		return err
	}
	return s.persistMetrics()
}

func (s *Store) StartHouseChallenge(ctx context.Context, userID, tierID, walletType string) (*models.GameSession, *models.HouseTier, error) {
	tier := findHouseTier(tierID)
	if tier == nil {
		return nil, nil, errors.New("house tier not found")
	}
	if walletType == "" {
		walletType = "demo"
	}
	if walletType != "demo" && walletType != "live" {
		return nil, nil, errors.New("wallet type must be demo or live")
	}

	s.mu.Lock()
	if _, ok := s.users[userID]; !ok {
		s.mu.Unlock()
		return nil, nil, errors.New("user not found")
	}
	progression := s.ensureProgressionLocked(userID)
	if progression.Level < tier.MinimumLevel {
		s.mu.Unlock()
		return nil, nil, fmt.Errorf("minimum level %d required", tier.MinimumLevel)
	}
	if progression.TrustScore < tier.MinimumTrust {
		s.mu.Unlock()
		return nil, nil, fmt.Errorf("minimum trust score %.0f required", tier.MinimumTrust)
	}
	if walletType == "live" {
		health := s.treasuryHealthLocked()
		potentialPayout := tier.Stake * tier.RewardRate
		if !health.IsSolvent || s.treasury.PlayerReserve < health.PlayerLiabilities+health.HouseExposure+potentialPayout {
			s.mu.Unlock()
			return nil, nil, errors.New("treasury reserve coverage is insufficient for this challenge")
		}
	}
	s.mu.Unlock()

	if riskReport, err := s.HouseRiskReport(ctx, tier.ID); err == nil && riskReport.RecommendedAction == "increase_difficulty" && tier.Difficulty < 6 {
		tier.Difficulty++
	}

	session := &models.GameSession{
		UserID:     userID,
		GameType:   walletType,
		Mode:       "house",
		HouseTier:  tier.ID,
		Stake:      tier.Stake,
		RewardRate: tier.RewardRate,
		Difficulty: tier.Difficulty,
	}
	if err := s.StartGameSession(ctx, session); err != nil {
		return nil, nil, err
	}
	return session, tier, nil
}

func (s *Store) StartDailyCalibration(ctx context.Context, userID string) (*models.GameSession, error) {
	today := time.Now().UTC().Format("2006-01-02")
	s.mu.RLock()
	if _, ok := s.users[userID]; !ok {
		s.mu.RUnlock()
		return nil, errors.New("user not found")
	}
	for _, session := range s.sessions {
		if session.UserID == userID && session.Calibration && session.CreatedAt.UTC().Format("2006-01-02") == today {
			s.mu.RUnlock()
			return nil, errors.New("daily calibration already started")
		}
	}
	s.mu.RUnlock()

	session := &models.GameSession{
		UserID:      userID,
		GameType:    "demo",
		Mode:        "calibration",
		Calibration: true,
		Stake:       0,
		RewardRate:  0,
		Difficulty:  1,
	}
	if err := s.StartGameSession(ctx, session); err != nil {
		return nil, err
	}
	return session, nil
}

func (s *Store) SubmitMazeMoves(ctx context.Context, userID string, sessionID string, moves []models.MazeMove) (*models.GameSession, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	session, ok := s.sessions[sessionID]
	if !ok {
		return nil, errors.New("session not found")
	}
	if session.UserID != userID {
		return nil, errors.New("session does not belong to user")
	}
	if session.IsFinished {
		return nil, errors.New("session already completed")
	}
	if session.State == "" {
		session.State = models.SessionStateReady
	}
	if session.State != models.SessionStateReady && session.State != models.SessionStateActive {
		return nil, fmt.Errorf("session is not playable from state %s", session.State)
	}
	if session.State == models.SessionStateReady {
		if err := transitionSessionState(session, models.SessionStateActive); err != nil {
			return nil, err
		}
	}

	actor, err := arenasecurity.FromUser(userID, "")
	if err != nil {
		return nil, err
	}
	if err := arenasecurity.AuthorizeSession(actor, session); err != nil {
		return nil, err
	}
	module, err := s.arenaModuleForSessionLocked(session)
	if err != nil {
		return nil, err
	}
	actions := make([]core.PlayerAction, 0, len(moves))
	for i, move := range moves {
		actionType := "click"
		if len(session.Lines) == 0 {
			actionType = "move"
		}
		actions = append(actions, core.PlayerAction{ActionType: actionType, TargetID: move.Direction, ClientTime: move.Timestamp, Metadata: map[string]string{"sequence": strconv.Itoa(i)}})
	}
	result, err := module.SubmitAction(ctx, core.ActionRequest{
		ActorUserID: actor.UserID,
		Session:     session,
		Actions:     actions,
	})
	if err != nil {
		return nil, err
	}

	valid := result.Valid
	session.Moves = result.History
	session.Lines = result.Lines
	session.Clicks = result.Clicks
	completedAt := time.Now().UTC()
	session.CompletedAt = &completedAt
	session.IsFinished = true
	if err := transitionSessionState(session, models.SessionStateCompleted); err != nil {
		return nil, err
	}

	wallet, ok := s.wallets[session.UserID]
	if !ok {
		return nil, errors.New("wallet not found")
	}

	if session.Calibration {
		if valid {
			session.Outcome = "calibrated"
		} else {
			session.Outcome = "calibration_incomplete"
		}
		session.Reward = 0
		s.updateBaselineLocked(session)
		s.recordCompletionLocked(session)
		if err := s.persistSessions(); err != nil {
			return nil, err
		}
		if err := s.persistBaselines(); err != nil {
			return nil, err
		}
		if err := s.persistMetrics(); err != nil {
			return nil, err
		}
		return session, nil
	}

	if valid {
		session.Outcome = "win"
		session.Reward = session.Stake * session.RewardRate
		if session.GameType == "live" {
			availableBefore := calculateAvailableLive(wallet)
			wallet.LiveLockedBalance -= session.Stake
			wallet.LiveBalance += session.Reward
			entry := &models.LedgerEntry{
				ID:              newUUID(),
				UserID:          session.UserID,
				TransactionType: models.TransactionTypeReward,
				Amount:          session.Reward,
				BalanceBefore:   availableBefore,
				BalanceAfter:    calculateAvailableLive(wallet),
				Currency:        "USD",
				Reference:       "game-reward",
				Metadata:        map[string]string{"gameType": session.GameType},
				CreatedAt:       time.Now().UTC(),
			}
			s.ledger[session.UserID] = append(s.ledger[session.UserID], entry)
		} else {
			availableBefore := calculateAvailableDemo(wallet)
			wallet.DemoLockedBalance -= session.Stake
			wallet.DemoBalance += session.Reward
			entry := &models.LedgerEntry{
				ID:              newUUID(),
				UserID:          session.UserID,
				TransactionType: models.TransactionTypeReward,
				Amount:          session.Reward,
				BalanceBefore:   availableBefore,
				BalanceAfter:    calculateAvailableDemo(wallet),
				Currency:        "USD",
				Reference:       "game-reward",
				Metadata:        map[string]string{"gameType": session.GameType},
				CreatedAt:       time.Now().UTC(),
			}
			s.ledger[session.UserID] = append(s.ledger[session.UserID], entry)
		}
	} else {
		session.Outcome = "lose"
		session.Reward = 0
		if session.GameType == "live" {
			availableBefore := calculateAvailableLive(wallet)
			if wallet.LiveLockedBalance < session.Stake || wallet.LiveBalance < session.Stake {
				return nil, errors.New("insufficient live balance to settle game")
			}
			wallet.LiveLockedBalance -= session.Stake
			wallet.LiveBalance -= session.Stake
			entry := &models.LedgerEntry{
				ID:              newUUID(),
				UserID:          session.UserID,
				TransactionType: models.TransactionTypeLoss,
				Amount:          -session.Stake,
				BalanceBefore:   availableBefore,
				BalanceAfter:    calculateAvailableLive(wallet),
				Currency:        "USD",
				Reference:       "game-loss",
				Metadata:        map[string]string{"gameType": session.GameType},
				CreatedAt:       time.Now().UTC(),
			}
			s.ledger[session.UserID] = append(s.ledger[session.UserID], entry)
		} else {
			availableBefore := calculateAvailableDemo(wallet)
			if wallet.DemoLockedBalance < session.Stake || wallet.DemoBalance < session.Stake {
				return nil, errors.New("insufficient demo balance to settle game")
			}
			wallet.DemoLockedBalance -= session.Stake
			wallet.DemoBalance -= session.Stake
			entry := &models.LedgerEntry{
				ID:              newUUID(),
				UserID:          session.UserID,
				TransactionType: models.TransactionTypeLoss,
				Amount:          -session.Stake,
				BalanceBefore:   availableBefore,
				BalanceAfter:    calculateAvailableDemo(wallet),
				Currency:        "USD",
				Reference:       "game-loss",
				Metadata:        map[string]string{"gameType": session.GameType},
				CreatedAt:       time.Now().UTC(),
			}
			s.ledger[session.UserID] = append(s.ledger[session.UserID], entry)
		}
	}

	s.applyGameProgressionLocked(session, valid)
	s.recordCompletionLocked(session)
	report := replayReportFromSession(session)
	if len(report.Flags) > 0 {
		s.ensureReviewCaseLocked("game_session", session.ID, session.UserID, "replay_flags", report.Flags)
	}

	if err := s.persistWallets(); err != nil {
		return nil, err
	}
	if err := s.persistSessions(); err != nil {
		return nil, err
	}
	if err := s.persistLedger(); err != nil {
		return nil, err
	}
	if err := s.persistProgression(); err != nil {
		return nil, err
	}
	if err := s.persistAchievements(); err != nil {
		return nil, err
	}
	if err := s.persistReviewCases(); err != nil {
		return nil, err
	}
	if err := s.persistAuditLogs(); err != nil {
		return nil, err
	}
	if err := s.persistMetrics(); err != nil {
		return nil, err
	}

	return session, nil
}

func (s *Store) JoinPvPQueue(ctx context.Context, userID, queueType, walletType string, stake float64) (*models.PvPMatchDetail, error) {
	started := time.Now()
	if queueType == "" {
		queueType = "standard"
	}
	if walletType == "" {
		walletType = "demo"
	}
	if walletType != "demo" && walletType != "live" {
		return nil, errors.New("wallet type must be demo or live")
	}
	if stake <= 0 {
		return nil, errors.New("stake must be greater than zero")
	}
	matcher := matchmaking.NewService()
	now := time.Now().UTC()
	request := matchmaking.JoinRequest{UserID: userID, QueueType: queueType, WalletType: walletType, Stake: stake, Now: now}
	_ = s.redis.Set(ctx, "presence:"+userID, "online", 5*time.Minute)
	lockKey := fmt.Sprintf("matchmaking:%s:%s:%.2f", queueType, walletType, stake)
	locked, err := s.redis.Lock(ctx, lockKey, 15*time.Second)
	if err != nil {
		return nil, err
	}
	if !locked {
		return nil, errors.New("matchmaking queue is busy, retry shortly")
	}
	defer s.redis.Unlock(ctx, lockKey)

	s.mu.Lock()
	if _, ok := s.users[userID]; !ok {
		s.mu.Unlock()
		return nil, errors.New("user not found")
	}
	progression := s.ensureProgressionLocked(userID)
	if progression.TrustScore < s.settings.Trust.PvPMinimum {
		s.mu.Unlock()
		return nil, fmt.Errorf("minimum trust score %.0f required for pvp", s.settings.Trust.PvPMinimum)
	}
	expired := matcher.ExpireStale(s.pvpMatches, now)
	if len(expired) > 0 {
		_ = s.persistPvPMatches()
	}
	if match := matcher.ActiveOrWaitingForUser(s.pvpMatches, userID); match != nil {
		detail := s.pvpDetailLocked(match, userID)
		s.recordMatchmakingLocked(time.Since(started))
		_ = s.persistMetrics()
		s.mu.Unlock()
		return detail, nil
	}
	waitingMatch := matcher.FindWaitingMatch(s.pvpMatches, request)
	s.mu.Unlock()

	if _, err := s.LockWalletTokens(ctx, userID, walletType, stake, "USD", "pvp-stake", map[string]string{"queueType": queueType}); err != nil {
		return nil, err
	}

	s.mu.Lock()

	matcher.ExpireStale(s.pvpMatches, time.Now().UTC())
	if match := matcher.ActiveOrWaitingForUser(s.pvpMatches, userID); match != nil {
		s.unlockPvPStakeLocked(userID, walletType, stake, "pvp-duplicate-refund", map[string]string{"matchId": match.ID})
		s.recordMatchmakingLocked(time.Since(started))
		_ = s.persistWallets()
		_ = s.persistLedger()
		_ = s.persistMetrics()
		detail := s.pvpDetailLocked(match, userID)
		s.mu.Unlock()
		return detail, nil
	}

	if waitingMatch != nil {
		current, ok := s.pvpMatches[waitingMatch.ID]
		if ok && current.Status == "waiting" && current.PlayerAID != userID && current.QueueType == queueType && current.WalletType == walletType && current.Stake == stake {
			now := time.Now().UTC()
			if !matcher.Activate(current, userID, now) {
				current.Status = "aborted"
				_ = s.unlockPvPStakeLocked(userID, walletType, stake, "pvp-self-match-refund", map[string]string{"matchId": current.ID})
				_ = s.persistWallets()
				_ = s.persistLedger()
				_ = s.persistPvPMatches()
				s.mu.Unlock()
				return nil, errors.New("self-match prevented")
			}
			current.PlatformFee = current.Stake * 2 * 0.1
			current.PrizePool = current.Stake*2 - current.PlatformFee
			maze := game.GenerateMaze(13, 13)
			current.MazeCells = maze.Cells
			current.Width = maze.Width
			current.Height = maze.Height
			current.StartX = maze.StartX
			current.StartY = maze.StartY
			current.EndX = maze.EndX
			current.EndY = maze.EndY
			profile := s.matchDifficultyProfileLocked(current.PlayerAID, current.PlayerBID, "pvp", "")
			current.DifficultyRating = profile.Rating
			current.DifficultyProfile = &profile
			current.PuzzleVersion = game.CurrentPuzzleVersion()
			service := s.puzzleServiceLocked()
			puzzleRequest := puzzle.Request{
				Mode:              puzzle.ModePvP,
				Purpose:           "pvp_match",
				MatchID:           current.ID,
				PlayerID:          current.PlayerAID + ":" + current.PlayerBID,
				Shared:            true,
				DifficultyProfile: profile,
				PuzzleVersion:     current.PuzzleVersion,
			}
			s.mu.Unlock()

			generationStarted := time.Now()
			generated, err := service.Generate(ctx, puzzleRequest)
			if err != nil {
				return nil, err
			}
			s.mu.Lock()
			current, ok = s.pvpMatches[waitingMatch.ID]
			if !ok || current.Status != "active" {
				s.mu.Unlock()
				return nil, errors.New("pvp match changed during puzzle generation")
			}
			current.DifficultyProfile = &generated.DifficultyProfile
			current.DifficultyRating = generated.DifficultyProfile.Rating
			current.PlayerASeed = generated.Metadata.Seed
			current.PlayerBSeed = generated.Metadata.Seed
			current.PlayerANonce = generated.Metadata.Nonce
			current.PlayerBNonce = generated.Metadata.Nonce
			current.PlayerAHash = generated.Metadata.GenerationHash
			current.PlayerBHash = generated.Metadata.GenerationHash
			current.PuzzleMetadata = &generated.Metadata
			current.PlayerALines = cloneArrowLines(generated.Lines)
			current.PlayerBLines = cloneArrowLines(generated.Lines)
			s.recordPuzzleGenerationLocked(time.Since(generationStarted))
			s.recordMatchmakingLocked(time.Since(started))
			if err := s.persistPvPMatches(); err != nil {
				s.mu.Unlock()
				return nil, err
			}
			_ = s.persistMetrics()
			detail := s.pvpDetailLocked(current, userID)
			s.mu.Unlock()
			return detail, nil
		}
	}

	match := &models.PvPMatch{
		ID:         id.Match(),
		PlayerAID:  userID,
		QueueType:  queueType,
		WalletType: walletType,
		Stake:      stake,
		Status:     "waiting",
		CreatedAt:  now,
	}
	s.pvpMatches[match.ID] = match
	s.recordMatchmakingLocked(time.Since(started))
	if err := s.persistPvPMatches(); err != nil {
		s.mu.Unlock()
		return nil, err
	}
	_ = s.persistMetrics()
	detail := s.pvpDetailLocked(match, userID)
	s.mu.Unlock()
	return detail, nil
}

func (s *Store) ListPvPMatchesByUserID(ctx context.Context, userID string) ([]*models.PvPMatchDetail, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if _, ok := s.users[userID]; !ok {
		return nil, errors.New("user not found")
	}
	details := make([]*models.PvPMatchDetail, 0)
	for _, match := range s.pvpMatches {
		if match.PlayerAID != "" && match.PlayerAID == match.PlayerBID {
			continue
		}
		if match.PlayerAID == userID || match.PlayerBID == userID {
			details = append(details, s.pvpDetailLocked(match, userID))
		}
	}
	sort.SliceStable(details, func(i, j int) bool {
		return details[i].Match.CreatedAt.After(details[j].Match.CreatedAt)
	})
	return details, nil
}

func (s *Store) GetPvPMatchDetail(ctx context.Context, matchID, userID string) (*models.PvPMatchDetail, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	match, ok := s.pvpMatches[matchID]
	if !ok {
		return nil, errors.New("pvp match not found")
	}
	if match.PlayerAID != userID && match.PlayerBID != userID {
		return nil, errors.New("pvp match does not belong to user")
	}
	return s.pvpDetailLocked(match, userID), nil
}

func (s *Store) SubmitPvPMoves(ctx context.Context, userID, matchID string, moves []models.MazeMove) (*models.PvPMatchDetail, error) {
	if len(moves) == 0 {
		return nil, errors.New("moves are required")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	match, ok := s.pvpMatches[matchID]
	if !ok {
		return nil, errors.New("pvp match not found")
	}
	if match.PlayerAID != userID && match.PlayerBID != userID {
		return nil, errors.New("pvp match does not belong to user")
	}
	if match.Status != "active" {
		return nil, errors.New("pvp match is not active")
	}
	for _, submission := range s.pvpSubmissions[matchID] {
		if submission.UserID == userID {
			return nil, errors.New("pvp submission already recorded")
		}
	}

	directions := make([]string, len(moves))
	for i, move := range moves {
		directions[i] = move.Direction
	}
	valid := false
	history := moves
	clicks := []models.ArrowClick{}
	if userID == match.PlayerAID && len(match.PlayerALines) > 0 {
		valid, match.PlayerALines, clicks = game.ValidateLineClicks(match.PlayerALines, directions)
	} else if userID == match.PlayerBID && len(match.PlayerBLines) > 0 {
		valid, match.PlayerBLines, clicks = game.ValidateLineClicks(match.PlayerBLines, directions)
	} else {
		maze := &game.Maze{
			Width:  match.Width,
			Height: match.Height,
			Cells:  match.MazeCells,
			StartX: match.StartX,
			StartY: match.StartY,
			EndX:   match.EndX,
			EndY:   match.EndY,
		}
		var err error
		valid, history, err = game.ValidateMazeMoves(maze, directions)
		if err != nil {
			return nil, err
		}
	}

	duration := 0.0
	if len(moves) > 1 {
		first := moves[0].Timestamp
		last := moves[len(moves)-1].Timestamp
		if !first.IsZero() && !last.IsZero() && last.After(first) {
			duration = last.Sub(first).Seconds()
		}
	}
	submission := &models.PvPSubmission{
		ID:              newUUID(),
		MatchID:         matchID,
		UserID:          userID,
		Moves:           history,
		Clicks:          clicks,
		IsValidRoute:    valid,
		MoveCount:       len(directions),
		DurationSeconds: duration,
		SubmittedAt:     time.Now().UTC(),
	}
	s.pvpSubmissions[matchID] = append(s.pvpSubmissions[matchID], submission)

	if len(s.pvpSubmissions[matchID]) >= 2 {
		if err := s.settlePvPMatchLocked(match); err != nil {
			return nil, err
		}
	}

	if err := s.persistPvPSubmissions(); err != nil {
		return nil, err
	}
	if err := s.persistPvPMatches(); err != nil {
		return nil, err
	}
	if err := s.persistWallets(); err != nil {
		return nil, err
	}
	if err := s.persistLedger(); err != nil {
		return nil, err
	}
	if err := s.persistProgression(); err != nil {
		return nil, err
	}
	if err := s.persistAchievements(); err != nil {
		return nil, err
	}

	return s.pvpDetailLocked(match, userID), nil
}

func (s *Store) UpdatePvPProgress(ctx context.Context, userID, matchID string, progress models.PvPProgress) (*models.PvPMatchDetail, error) {
	now := time.Now().UTC()
	s.mu.Lock()
	defer s.mu.Unlock()
	match := s.pvpMatches[matchID]
	if match == nil {
		return nil, errors.New("match not found")
	}
	if match.PlayerAID != userID && match.PlayerBID != userID {
		return nil, errors.New("user is not part of match")
	}
	progress.UserID = userID
	progress.LastEventAt = now
	if progress.MovesRemaining < 0 {
		progress.MovesRemaining = 0
	}
	if progress.CompletionPercent < 0 {
		progress.CompletionPercent = 0
	}
	if progress.CompletionPercent > 100 {
		progress.CompletionPercent = 100
	}
	if progress.Finished && progress.FinishedAt == nil {
		finishedAt := now
		progress.FinishedAt = &finishedAt
	}
	if match.PlayerAID == userID {
		match.PlayerAProgress = &progress
	} else {
		match.PlayerBProgress = &progress
	}
	s.audit = append(s.audit, &models.AuditLog{
		ID:       newUUID(),
		ActorID:  userID,
		Action:   "pvp.progress.updated",
		TargetID: match.ID,
		Metadata: map[string]string{
			"completionPercent": fmt.Sprintf("%.2f", progress.CompletionPercent),
			"finished":          fmt.Sprintf("%t", progress.Finished),
			"disconnected":      fmt.Sprintf("%t", progress.Disconnected),
		},
		CreatedAt: now,
	})
	if err := s.persistPvPMatches(); err != nil {
		return nil, err
	}
	if err := s.persistAuditLogs(); err != nil {
		return nil, err
	}
	return s.pvpDetailLocked(match, userID), nil
}

func (s *Store) settlePvPMatchLocked(match *models.PvPMatch) error {
	submissions := s.pvpSubmissions[match.ID]
	if len(submissions) < 2 || match.Status != "active" {
		return nil
	}

	var playerASubmission *models.PvPSubmission
	var playerBSubmission *models.PvPSubmission
	for _, submission := range submissions {
		switch submission.UserID {
		case match.PlayerAID:
			playerASubmission = submission
		case match.PlayerBID:
			playerBSubmission = submission
		}
	}
	if playerASubmission == nil || playerBSubmission == nil {
		return nil
	}

	winnerID := pvpWinner(match, playerASubmission, playerBSubmission)
	if winnerID == "" {
		if err := s.unlockPvPStakeLocked(match.PlayerAID, match.WalletType, match.Stake, "pvp-refund", map[string]string{"matchId": match.ID}); err != nil {
			return err
		}
		if err := s.unlockPvPStakeLocked(match.PlayerBID, match.WalletType, match.Stake, "pvp-refund", map[string]string{"matchId": match.ID}); err != nil {
			return err
		}
	} else {
		if err := s.consumePvPStakeLocked(match.PlayerAID, match.WalletType, match.Stake, map[string]string{"matchId": match.ID}); err != nil {
			return err
		}
		if err := s.consumePvPStakeLocked(match.PlayerBID, match.WalletType, match.Stake, map[string]string{"matchId": match.ID}); err != nil {
			return err
		}
		if err := s.creditPvPRewardLocked(winnerID, match.WalletType, match.PrizePool, map[string]string{"matchId": match.ID}); err != nil {
			return err
		}
	}

	now := time.Now().UTC()
	match.Status = "completed"
	match.WinnerID = winnerID
	match.CompletedAt = &now
	s.applyPvPProgressionLocked(match, winnerID)
	return nil
}

func pvpWinner(match *models.PvPMatch, playerA, playerB *models.PvPSubmission) string {
	if playerA.IsValidRoute && !playerB.IsValidRoute {
		return match.PlayerAID
	}
	if playerB.IsValidRoute && !playerA.IsValidRoute {
		return match.PlayerBID
	}
	if !playerA.IsValidRoute && !playerB.IsValidRoute {
		return ""
	}
	if playerA.MoveCount != playerB.MoveCount {
		if playerA.MoveCount < playerB.MoveCount {
			return match.PlayerAID
		}
		return match.PlayerBID
	}
	if playerA.DurationSeconds > 0 && playerB.DurationSeconds > 0 && playerA.DurationSeconds != playerB.DurationSeconds {
		if playerA.DurationSeconds < playerB.DurationSeconds {
			return match.PlayerAID
		}
		return match.PlayerBID
	}
	if playerA.SubmittedAt.Before(playerB.SubmittedAt) {
		return match.PlayerAID
	}
	return match.PlayerBID
}

func (s *Store) unlockPvPStakeLocked(userID, walletType string, amount float64, reference string, metadata map[string]string) error {
	wallet, ok := s.wallets[userID]
	if !ok {
		return errors.New("wallet not found")
	}
	entry := &models.LedgerEntry{
		ID:              newUUID(),
		UserID:          userID,
		TransactionType: models.TransactionTypeUnlock,
		Amount:          amount,
		Currency:        "USD",
		Reference:       reference,
		Metadata:        metadata,
		CreatedAt:       time.Now().UTC(),
	}
	switch walletType {
	case "live":
		if wallet.LiveLockedBalance < amount {
			return errors.New("insufficient locked live balance")
		}
		entry.BalanceBefore = calculateAvailableLive(wallet)
		wallet.LiveLockedBalance -= amount
		entry.BalanceAfter = calculateAvailableLive(wallet)
	case "demo":
		if wallet.DemoLockedBalance < amount {
			return errors.New("insufficient locked demo balance")
		}
		entry.BalanceBefore = calculateAvailableDemo(wallet)
		wallet.DemoLockedBalance -= amount
		entry.BalanceAfter = calculateAvailableDemo(wallet)
	default:
		return errors.New("unsupported wallet type")
	}
	s.ledger[userID] = append(s.ledger[userID], entry)
	return nil
}

func (s *Store) consumePvPStakeLocked(userID, walletType string, amount float64, metadata map[string]string) error {
	wallet, ok := s.wallets[userID]
	if !ok {
		return errors.New("wallet not found")
	}
	entry := &models.LedgerEntry{
		ID:              newUUID(),
		UserID:          userID,
		TransactionType: models.TransactionTypeStake,
		Amount:          -amount,
		Currency:        "USD",
		Reference:       "pvp-stake-settlement",
		Metadata:        metadata,
		CreatedAt:       time.Now().UTC(),
	}
	switch walletType {
	case "live":
		if wallet.LiveLockedBalance < amount || wallet.LiveBalance < amount {
			return errors.New("insufficient live balance to settle pvp stake")
		}
		entry.BalanceBefore = calculateAvailableLive(wallet)
		wallet.LiveLockedBalance -= amount
		wallet.LiveBalance -= amount
		entry.BalanceAfter = calculateAvailableLive(wallet)
	case "demo":
		if wallet.DemoLockedBalance < amount || wallet.DemoBalance < amount {
			return errors.New("insufficient demo balance to settle pvp stake")
		}
		entry.BalanceBefore = calculateAvailableDemo(wallet)
		wallet.DemoLockedBalance -= amount
		wallet.DemoBalance -= amount
		entry.BalanceAfter = calculateAvailableDemo(wallet)
	default:
		return errors.New("unsupported wallet type")
	}
	s.ledger[userID] = append(s.ledger[userID], entry)
	return nil
}

func (s *Store) creditPvPRewardLocked(userID, walletType string, amount float64, metadata map[string]string) error {
	wallet, ok := s.wallets[userID]
	if !ok {
		return errors.New("wallet not found")
	}
	entry := &models.LedgerEntry{
		ID:              newUUID(),
		UserID:          userID,
		TransactionType: models.TransactionTypeReward,
		Amount:          amount,
		Currency:        "USD",
		Reference:       "pvp-reward",
		Metadata:        metadata,
		CreatedAt:       time.Now().UTC(),
	}
	switch walletType {
	case "live":
		entry.BalanceBefore = calculateAvailableLive(wallet)
		wallet.LiveBalance += amount
		entry.BalanceAfter = calculateAvailableLive(wallet)
	case "demo":
		entry.BalanceBefore = calculateAvailableDemo(wallet)
		wallet.DemoBalance += amount
		entry.BalanceAfter = calculateAvailableDemo(wallet)
	default:
		return errors.New("unsupported wallet type")
	}
	s.ledger[userID] = append(s.ledger[userID], entry)
	return nil
}

func (s *Store) applyPvPProgressionLocked(match *models.PvPMatch, winnerID string) {
	for _, userID := range []string{match.PlayerAID, match.PlayerBID} {
		progression := s.ensureProgressionLocked(userID)
		progression.MatchesPlayed++
		progression.XP += 15
		progression.SeasonPoints += 8
		progression.LegacyPoints++
		if winnerID == userID {
			progression.Wins++
			progression.CurrentStreak++
			progression.XP += 60
			progression.SeasonPoints += 35
			progression.LegacyPoints += 5
			progression.EloRating += 18
			progression.TrustScore = clampTrust(progression.TrustScore + 0.2)
			s.unlockAchievementLocked(userID, "first_win", "Maze Breaker", "Win a Maze Arena session.")
		} else if winnerID != "" {
			progression.Losses++
			progression.CurrentStreak = 0
			progression.EloRating -= 14
			if progression.EloRating < 1000 {
				progression.EloRating = 1000
			}
			progression.TrustScore = clampTrust(progression.TrustScore - 0.5)
		}
		progression.Level, progression.Prestige = levelFromXP(progression.XP)
		progression.LeagueTier = leagueFromElo(progression.EloRating)
		progression.UpdatedAt = time.Now().UTC()
		s.unlockAchievementLocked(userID, "first_match", "First Run", "Complete your first Maze Arena session.")
		if progression.MatchesPlayed >= 10 {
			s.unlockAchievementLocked(userID, "ten_matches", "Arena Regular", "Complete ten Maze Arena sessions.")
		}
		if progression.CurrentStreak >= 3 {
			s.unlockAchievementLocked(userID, "three_streak", "Clean Streak", "Win three Maze Arena sessions in a row.")
		}
	}
}

func (s *Store) pvpDetailLocked(match *models.PvPMatch, viewerID string) *models.PvPMatchDetail {
	copyMatch := *match
	copyMatch.MazeCells = append([]string(nil), match.MazeCells...)
	copyMatch.PlayerALines = append([]models.ArrowLine(nil), match.PlayerALines...)
	copyMatch.PlayerBLines = append([]models.ArrowLine(nil), match.PlayerBLines...)
	copyMatch.PlayerASeed = ""
	copyMatch.PlayerBSeed = ""
	copyMatch.PlayerANonce = ""
	copyMatch.PlayerBNonce = ""
	copyMatch.PlayerAHash = ""
	copyMatch.PlayerBHash = ""
	if viewerID == match.PlayerAID {
		copyMatch.PlayerBLines = nil
	}
	if viewerID == match.PlayerBID {
		copyMatch.PlayerALines = nil
	}
	submissions := append([]*models.PvPSubmission(nil), s.pvpSubmissions[match.ID]...)
	copySubmissions := make([]*models.PvPSubmission, 0, len(submissions))
	for _, submission := range submissions {
		copySubmission := *submission
		copySubmission.Moves = append([]models.MazeMove(nil), submission.Moves...)
		copySubmission.Clicks = append([]models.ArrowClick(nil), submission.Clicks...)
		if viewerID != "" && submission.UserID != viewerID {
			copySubmission.Moves = nil
			copySubmission.Clicks = nil
		}
		copySubmissions = append(copySubmissions, &copySubmission)
	}
	return &models.PvPMatchDetail{
		Match:       &copyMatch,
		Submissions: copySubmissions,
	}
}

func (s *Store) GetLeaderboard() ([]*models.LeaderboardEntry, error) {
	s.mu.RLock()
	if cached, ok := s.cache.Get("leaderboard"); ok {
		if leaderboard, ok := cached.([]*models.LeaderboardEntry); ok {
			copyLeaderboard := append([]*models.LeaderboardEntry(nil), leaderboard...)
			s.mu.RUnlock()
			return copyLeaderboard, nil
		}
	}
	defer s.mu.RUnlock()

	leaderboard := make([]*models.LeaderboardEntry, 0, len(s.users))
	for _, user := range s.users {
		if !s.isPublicUserLocked(user) {
			continue
		}
		wallet := s.wallets[user.ID]
		if wallet == nil {
			wallet = &models.Wallet{UserID: user.ID, LiveBalance: 0, DemoBalance: 0}
		}
		progression := s.ensureProgressionLocked(user.ID)
		score := wallet.LiveBalance + wallet.DemoBalance
		leaderboard = append(leaderboard, &models.LeaderboardEntry{
			UserID:      user.ID,
			Username:    publicUsername(user),
			DisplayName: publicDisplayName(user),
			LeagueTier:  progression.LeagueTier,
			Rating:      progression.EloRating,
			Country:     "Global",
			Score:       score,
		})
	}

	sort.SliceStable(leaderboard, func(i, j int) bool {
		return leaderboard[i].Score > leaderboard[j].Score
	})

	for idx, entry := range leaderboard {
		entry.Rank = idx + 1
	}

	if len(leaderboard) > 20 {
		leaderboard = leaderboard[:20]
	}

	s.cache.Set("leaderboard", append([]*models.LeaderboardEntry(nil), leaderboard...), time.Duration(s.settings.Cache.LeaderboardTTLSeconds)*time.Second)
	return leaderboard, nil
}

func (s *Store) PlatformStats(ctx context.Context) (*models.PlatformStats, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	launchPhase := strings.ToUpper(s.settings.Platform.LaunchPhase)
	if launchPhase == "" {
		launchPhase = "PRE_LAUNCH"
	}
	stats := &models.PlatformStats{
		LaunchPhase:        launchPhase,
		PlayersOnline:      s.activeAuthCountLocked(),
		LiveMatches:        s.activeMatchCountLocked(),
		MatchesToday:       s.matchesTodayLocked(),
		Countries:          s.publicCountryCountLocked(),
		LeaderboardsStatus: "Live",
	}
	if s.season != nil {
		stats.CurrentSeason = s.season.Name
		prizePool := s.season.RewardPool
		stats.PrizePool = &prizePool
	}
	switch launchPhase {
	case "PRE_LAUNCH":
		stats.PlayersOnline = 0
		stats.LiveMatches = 0
		stats.MatchesToday = 0
		stats.Countries = 0
		stats.CurrentSeason = "Coming Soon"
		stats.PrizePool = nil
		stats.LeaderboardsStatus = "Opens at Launch"
	case "BETA":
		if stats.CurrentSeason == "" {
			stats.CurrentSeason = "Beta Season"
		}
		stats.LeaderboardsStatus = "Live"
	}
	return stats, nil
}

func (s *Store) matchesTodayLocked() int {
	start := time.Now().UTC().Truncate(24 * time.Hour)
	count := 0
	for _, session := range s.sessions {
		if session.CreatedAt.After(start) || session.CreatedAt.Equal(start) {
			count++
		}
	}
	return count
}

func (s *Store) publicCountryCountLocked() int {
	hasPublic := false
	for _, user := range s.users {
		if s.isPublicUserLocked(user) {
			hasPublic = true
			break
		}
	}
	if hasPublic {
		return 1
	}
	return 0
}

func (s *Store) GetActiveSeason(ctx context.Context) (*models.Season, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	season := *s.season
	return &season, nil
}

func (s *Store) GetSeasonLeaderboard(ctx context.Context) ([]*models.SeasonLeaderboardEntry, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	entries := make([]*models.SeasonLeaderboardEntry, 0, len(s.profiles))
	for userID, progression := range s.profiles {
		user := s.users[userID]
		if !s.isPublicUserLocked(user) {
			continue
		}
		entries = append(entries, &models.SeasonLeaderboardEntry{
			UserID:       userID,
			Username:     publicUsername(user),
			DisplayName:  publicDisplayName(user),
			LeagueTier:   progression.LeagueTier,
			SeasonPoints: progression.SeasonPoints,
			Wins:         progression.Wins,
			Losses:       progression.Losses,
			TrustScore:   progression.TrustScore,
		})
	}
	sort.SliceStable(entries, func(i, j int) bool {
		if entries[i].SeasonPoints == entries[j].SeasonPoints {
			return entries[i].Wins > entries[j].Wins
		}
		return entries[i].SeasonPoints > entries[j].SeasonPoints
	})
	for idx, entry := range entries {
		entry.Rank = idx + 1
	}
	if len(entries) > 50 {
		entries = entries[:50]
	}
	return entries, nil
}

func (s *Store) isPublicUserLocked(user *models.User) bool {
	if user == nil || user.HiddenFromPublic {
		return false
	}
	switch user.Role {
	case models.RoleSuperAdmin, models.RoleAdmin, models.RoleTreasuryManager, models.RoleFraudAnalyst, models.RoleSupport, models.RoleModerator:
		return false
	default:
		return true
	}
}

func publicUsername(user *models.User) string {
	if user.Username != "" {
		return user.Username
	}
	if at := strings.Index(user.Email, "@"); at > 0 {
		return user.Email[:at]
	}
	return user.ID
}

func publicDisplayName(user *models.User) string {
	if user.DisplayName != "" {
		return user.DisplayName
	}
	return publicUsername(user)
}

func AchievementCatalog() []*models.AchievementCatalogItem {
	return []*models.AchievementCatalogItem{
		{Code: "first_match", Title: "First Run", Description: "Complete your first Maze Arena session.", Category: "gameplay"},
		{Code: "first_win", Title: "Maze Breaker", Description: "Win a Maze Arena session.", Category: "gameplay"},
		{Code: "ten_matches", Title: "Arena Regular", Description: "Complete ten Maze Arena sessions.", Category: "progression"},
		{Code: "three_streak", Title: "Clean Streak", Description: "Win three Maze Arena sessions in a row.", Category: "skill"},
	}
}

func (s *Store) ListTournaments(ctx context.Context, userID string) ([]*models.TournamentDetail, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	details := make([]*models.TournamentDetail, 0, len(s.tournaments))
	for _, tournament := range s.tournaments {
		copyTournament := *tournament
		registered := false
		rawParticipants := append([]*models.TournamentParticipant{}, s.participants[tournament.ID]...)
		for _, participant := range rawParticipants {
			if participant.UserID == userID {
				registered = true
				break
			}
		}
		participants := s.publicTournamentParticipantsLocked(rawParticipants)
		details = append(details, &models.TournamentDetail{
			Tournament:   &copyTournament,
			Registered:   registered,
			Participants: participants,
			Matches:      s.tournamentMatchesForViewerLocked(tournament.ID, userID),
			Submissions:  s.tournamentSubmissionsByTournamentLocked(tournament.ID),
		})
	}
	sort.SliceStable(details, func(i, j int) bool {
		return details[i].Tournament.StartsAt.Before(details[j].Tournament.StartsAt)
	})
	return details, nil
}

func (s *Store) GetTournamentDetail(ctx context.Context, tournamentID, userID string) (*models.TournamentDetail, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	tournament, ok := s.tournaments[tournamentID]
	if !ok {
		return nil, errors.New("tournament not found")
	}
	copyTournament := *tournament
	rawParticipants := append([]*models.TournamentParticipant{}, s.participants[tournamentID]...)
	registered := false
	for _, participant := range rawParticipants {
		if participant.UserID == userID {
			registered = true
			break
		}
	}
	participants := s.publicTournamentParticipantsLocked(rawParticipants)
	return &models.TournamentDetail{
		Tournament:   &copyTournament,
		Registered:   registered,
		Participants: participants,
		Matches:      s.tournamentMatchesForViewerLocked(tournamentID, userID),
		Submissions:  s.tournamentSubmissionsByTournamentLocked(tournamentID),
	}, nil
}

func (s *Store) tournamentMatchesForViewerLocked(tournamentID, viewerID string) []*models.TournamentMatch {
	matches := s.tMatches[tournamentID]
	copies := make([]*models.TournamentMatch, 0, len(matches))
	for _, match := range matches {
		copyMatch := *match
		copyMatch.PlayerALines = append([]models.ArrowLine(nil), match.PlayerALines...)
		copyMatch.PlayerBLines = append([]models.ArrowLine(nil), match.PlayerBLines...)
		copyMatch.PlayerASeed = ""
		copyMatch.PlayerBSeed = ""
		copyMatch.PlayerANonce = ""
		copyMatch.PlayerBNonce = ""
		copyMatch.PlayerAHash = ""
		copyMatch.PlayerBHash = ""
		switch viewerID {
		case match.PlayerAID:
			copyMatch.PlayerBLines = nil
		case match.PlayerBID:
			copyMatch.PlayerALines = nil
		default:
			copyMatch.PlayerALines = nil
			copyMatch.PlayerBLines = nil
		}
		copies = append(copies, &copyMatch)
	}
	return copies
}

func (s *Store) tournamentSubmissionsByTournamentLocked(tournamentID string) []*models.TournamentSubmission {
	submissions := make([]*models.TournamentSubmission, 0)
	for _, matchSubmissions := range s.tSubmissions {
		for _, submission := range matchSubmissions {
			if submission.TournamentID == tournamentID {
				copySubmission := *submission
				copySubmission.Clicks = append([]models.ArrowClick(nil), submission.Clicks...)
				submissions = append(submissions, &copySubmission)
			}
		}
	}
	return submissions
}

func (s *Store) publicTournamentParticipantsLocked(participants []*models.TournamentParticipant) []*models.TournamentParticipant {
	publicParticipants := make([]*models.TournamentParticipant, 0, len(participants))
	for _, participant := range participants {
		user := s.users[participant.UserID]
		if !s.isPublicUserLocked(user) {
			continue
		}
		copyParticipant := *participant
		copyParticipant.Username = publicUsername(user)
		copyParticipant.DisplayName = publicDisplayName(user)
		publicParticipants = append(publicParticipants, &copyParticipant)
	}
	return publicParticipants
}

func (s *Store) RegisterTournament(ctx context.Context, userID, tournamentID string) (*models.TournamentParticipant, error) {
	s.mu.Lock()
	tournament, ok := s.tournaments[tournamentID]
	if !ok {
		s.mu.Unlock()
		return nil, errors.New("tournament not found")
	}
	if tournament.Status != "registration" {
		s.mu.Unlock()
		return nil, errors.New("tournament is not open for registration")
	}
	user, ok := s.users[userID]
	if !ok {
		s.mu.Unlock()
		return nil, errors.New("user not found")
	}
	progression := s.ensureProgressionLocked(userID)
	if progression.Level < tournament.MinimumLevel {
		s.mu.Unlock()
		return nil, fmt.Errorf("minimum level %d required", tournament.MinimumLevel)
	}
	if progression.TrustScore < tournament.MinimumTrust {
		s.mu.Unlock()
		return nil, fmt.Errorf("minimum trust score %.0f required", tournament.MinimumTrust)
	}
	participants := s.participants[tournamentID]
	if tournament.MaxParticipants > 0 && len(participants) >= tournament.MaxParticipants {
		s.mu.Unlock()
		return nil, errors.New("tournament is full")
	}
	for _, participant := range participants {
		if participant.UserID == userID {
			s.mu.Unlock()
			return nil, errors.New("already registered")
		}
	}
	if tournament.WalletType == "live" {
		health := s.treasuryHealthLocked()
		if !health.IsSolvent || s.treasury.PlayerReserve < health.PlayerLiabilities+tournament.PrizePool {
			s.mu.Unlock()
			return nil, errors.New("treasury reserve coverage is insufficient for this tournament")
		}
	}
	s.mu.Unlock()

	if _, err := s.LockWalletTokens(ctx, userID, tournament.WalletType, tournament.EntryFee, "USD", "tournament-entry", map[string]string{"tournamentId": tournamentID}); err != nil {
		return nil, err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	participants = s.participants[tournamentID]
	for _, participant := range participants {
		if participant.UserID == userID {
			return nil, errors.New("already registered")
		}
	}
	participant := &models.TournamentParticipant{
		ID:           newUUID(),
		TournamentID: tournamentID,
		UserID:       userID,
		Username:     publicUsername(user),
		DisplayName:  publicDisplayName(user),
		Seed:         len(participants) + 1,
		Status:       "registered",
		RegisteredAt: time.Now().UTC(),
	}
	s.participants[tournamentID] = append(s.participants[tournamentID], participant)
	s.audit = append(s.audit, &models.AuditLog{
		ID:        newUUID(),
		ActorID:   userID,
		Action:    "tournament.registered",
		TargetID:  tournamentID,
		Metadata:  map[string]string{"participantId": participant.ID},
		CreatedAt: time.Now().UTC(),
	})
	if err := s.persistTournamentParticipants(); err != nil {
		return nil, err
	}
	if err := s.persistAuditLogs(); err != nil {
		return nil, err
	}
	return participant, nil
}

func (s *Store) GenerateTournamentBracket(ctx context.Context, actorID, tournamentID string) ([]*models.TournamentMatch, error) {
	s.mu.Lock()

	tournament, ok := s.tournaments[tournamentID]
	if !ok {
		s.mu.Unlock()
		return nil, errors.New("tournament not found")
	}
	if len(s.tMatches[tournamentID]) > 0 {
		s.mu.Unlock()
		return nil, errors.New("bracket already generated")
	}
	participants := append([]*models.TournamentParticipant(nil), s.participants[tournamentID]...)
	if len(participants) < 2 {
		s.mu.Unlock()
		return nil, errors.New("at least two participants are required")
	}
	sort.SliceStable(participants, func(i, j int) bool {
		return participants[i].Seed < participants[j].Seed
	})

	type pendingTournamentPuzzle struct {
		match   *models.TournamentMatch
		request puzzle.Request
		result  puzzle.Puzzle
		elapsed time.Duration
	}
	matches := make([]*models.TournamentMatch, 0)
	pending := make([]pendingTournamentPuzzle, 0)
	now := time.Now().UTC()
	for i := 0; i < len(participants); i += 2 {
		match := &models.TournamentMatch{
			ID:           id.Match(),
			TournamentID: tournamentID,
			Round:        1,
			MatchNumber:  len(matches) + 1,
			PlayerAID:    participants[i].UserID,
			Status:       "scheduled",
			CreatedAt:    now,
		}
		if i+1 < len(participants) {
			match.PlayerBID = participants[i+1].UserID
			profile := s.matchDifficultyProfileLocked(match.PlayerAID, match.PlayerBID, "tournament", tournament.Type)
			match.DifficultyRating = profile.Rating
			match.DifficultyProfile = &profile
			match.PuzzleVersion = game.CurrentPuzzleVersion()
			pending = append(pending, pendingTournamentPuzzle{
				match: match,
				request: puzzle.Request{
					Mode:              puzzle.ModeTournament,
					Purpose:           "tournament_match",
					MatchID:           match.ID,
					PlayerID:          tournamentID,
					Shared:            true,
					DifficultyProfile: profile,
					PuzzleVersion:     match.PuzzleVersion,
				},
			})
		} else {
			match.WinnerID = participants[i].UserID
			match.Status = "completed"
			match.CompletedAt = &now
		}
		matches = append(matches, match)
	}
	service := s.puzzleServiceLocked()
	s.mu.Unlock()

	for i := range pending {
		generationStarted := time.Now()
		generated, err := service.Generate(ctx, pending[i].request)
		if err != nil {
			return nil, err
		}
		pending[i].result = generated
		pending[i].elapsed = time.Since(generationStarted)
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	if len(s.tMatches[tournamentID]) > 0 {
		return nil, errors.New("bracket already generated")
	}
	for _, item := range pending {
		item.match.DifficultyProfile = &item.result.DifficultyProfile
		item.match.DifficultyRating = item.result.DifficultyProfile.Rating
		item.match.PlayerASeed = item.result.Metadata.Seed
		item.match.PlayerBSeed = item.result.Metadata.Seed
		item.match.PlayerANonce = item.result.Metadata.Nonce
		item.match.PlayerBNonce = item.result.Metadata.Nonce
		item.match.PlayerAHash = item.result.Metadata.GenerationHash
		item.match.PlayerBHash = item.result.Metadata.GenerationHash
		item.match.PuzzleMetadata = &item.result.Metadata
		item.match.PlayerALines = cloneArrowLines(item.result.Lines)
		item.match.PlayerBLines = cloneArrowLines(item.result.Lines)
		s.recordPuzzleGenerationLocked(item.elapsed)
	}
	tournament, ok = s.tournaments[tournamentID]
	if !ok {
		return nil, errors.New("tournament not found")
	}
	tournament.Status = "active"
	s.tMatches[tournamentID] = matches
	s.audit = append(s.audit, &models.AuditLog{
		ID:        newUUID(),
		ActorID:   actorID,
		Action:    "tournament.bracket.generated",
		TargetID:  tournamentID,
		CreatedAt: now,
	})
	if err := s.persistTournaments(); err != nil {
		return nil, err
	}
	if err := s.persistTournamentMatches(); err != nil {
		return nil, err
	}
	if err := s.persistAuditLogs(); err != nil {
		return nil, err
	}
	if err := s.persistMetrics(); err != nil {
		return nil, err
	}
	return matches, nil
}

func (s *Store) SubmitTournamentMatchClicks(ctx context.Context, userID, tournamentID, matchID string, clickedLineIDs []string) (*models.TournamentDetail, error) {
	if len(clickedLineIDs) == 0 {
		return nil, errors.New("clicked line ids are required")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	tournament, ok := s.tournaments[tournamentID]
	if !ok {
		return nil, errors.New("tournament not found")
	}
	var match *models.TournamentMatch
	for _, existing := range s.tMatches[tournamentID] {
		if existing.ID == matchID {
			match = existing
			break
		}
	}
	if match == nil {
		return nil, errors.New("match not found")
	}
	if match.Status == "completed" {
		return nil, errors.New("match already completed")
	}
	if userID != match.PlayerAID && userID != match.PlayerBID {
		return nil, errors.New("match does not belong to user")
	}
	for _, submission := range s.tSubmissions[matchID] {
		if submission.UserID == userID {
			return nil, errors.New("tournament submission already recorded")
		}
	}

	isComplete := false
	clicks := []models.ArrowClick{}
	if userID == match.PlayerAID {
		isComplete, match.PlayerALines, clicks = game.ValidateLineClicks(match.PlayerALines, clickedLineIDs)
	} else {
		isComplete, match.PlayerBLines, clicks = game.ValidateLineClicks(match.PlayerBLines, clickedLineIDs)
	}
	submission := &models.TournamentSubmission{
		ID:           newUUID(),
		TournamentID: tournamentID,
		MatchID:      matchID,
		UserID:       userID,
		Clicks:       clicks,
		IsComplete:   isComplete,
		MoveCount:    len(clickedLineIDs),
		SubmittedAt:  time.Now().UTC(),
	}
	if len(clicks) > 1 {
		first := clicks[0].Timestamp
		last := clicks[len(clicks)-1].Timestamp
		if last.After(first) {
			submission.DurationSeconds = last.Sub(first).Seconds()
		}
	}
	s.tSubmissions[matchID] = append(s.tSubmissions[matchID], submission)

	if len(s.tSubmissions[matchID]) >= 2 {
		winnerID := tournamentMatchWinner(match, s.tSubmissions[matchID])
		if winnerID != "" {
			now := time.Now().UTC()
			match.WinnerID = winnerID
			match.Status = "completed"
			match.CompletedAt = &now
			s.advanceTournamentLocked(tournament, winnerID, match.Round)
			if err := s.settleTournamentIfCompleteLocked(tournament); err != nil {
				return nil, err
			}
			s.audit = append(s.audit, &models.AuditLog{
				ID:        newUUID(),
				ActorID:   userID,
				Action:    "tournament.match.auto.completed",
				TargetID:  match.ID,
				Metadata:  map[string]string{"tournamentId": tournamentID, "winnerId": winnerID},
				CreatedAt: now,
			})
		}
	}

	if err := s.persistTournamentMatches(); err != nil {
		return nil, err
	}
	if err := s.persistTournamentSubmissions(); err != nil {
		return nil, err
	}
	if err := s.persistTournaments(); err != nil {
		return nil, err
	}
	if err := s.persistWallets(); err != nil {
		return nil, err
	}
	if err := s.persistLedger(); err != nil {
		return nil, err
	}
	if err := s.persistAuditLogs(); err != nil {
		return nil, err
	}
	if err := s.persistMetrics(); err != nil {
		return nil, err
	}

	copyTournament := *tournament
	return &models.TournamentDetail{
		Tournament:   &copyTournament,
		Registered:   true,
		Participants: s.publicTournamentParticipantsLocked(s.participants[tournamentID]),
		Matches:      s.tournamentMatchesForViewerLocked(tournamentID, userID),
		Submissions:  s.tournamentSubmissionsByTournamentLocked(tournamentID),
	}, nil
}

func tournamentMatchWinner(match *models.TournamentMatch, submissions []*models.TournamentSubmission) string {
	var playerA *models.TournamentSubmission
	var playerB *models.TournamentSubmission
	for _, submission := range submissions {
		switch submission.UserID {
		case match.PlayerAID:
			playerA = submission
		case match.PlayerBID:
			playerB = submission
		}
	}
	if playerA == nil || playerB == nil {
		return ""
	}
	if playerA.IsComplete && !playerB.IsComplete {
		return match.PlayerAID
	}
	if playerB.IsComplete && !playerA.IsComplete {
		return match.PlayerBID
	}
	if !playerA.IsComplete && !playerB.IsComplete {
		if playerA.MoveCount == playerB.MoveCount {
			return match.PlayerAID
		}
		if playerA.MoveCount > playerB.MoveCount {
			return match.PlayerAID
		}
		return match.PlayerBID
	}
	if playerA.MoveCount != playerB.MoveCount {
		if playerA.MoveCount < playerB.MoveCount {
			return match.PlayerAID
		}
		return match.PlayerBID
	}
	if playerA.DurationSeconds > 0 && playerB.DurationSeconds > 0 && playerA.DurationSeconds != playerB.DurationSeconds {
		if playerA.DurationSeconds < playerB.DurationSeconds {
			return match.PlayerAID
		}
		return match.PlayerBID
	}
	if playerA.SubmittedAt.Before(playerB.SubmittedAt) {
		return match.PlayerAID
	}
	return match.PlayerBID
}

func (s *Store) ReportTournamentMatchResult(ctx context.Context, actorID, tournamentID, matchID, winnerID string) (*models.TournamentMatch, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	tournament, ok := s.tournaments[tournamentID]
	if !ok {
		return nil, errors.New("tournament not found")
	}
	matches := s.tMatches[tournamentID]
	var match *models.TournamentMatch
	for _, existing := range matches {
		if existing.ID == matchID {
			match = existing
			break
		}
	}
	if match == nil {
		return nil, errors.New("match not found")
	}
	if match.Status == "completed" {
		return nil, errors.New("match already completed")
	}
	if winnerID != match.PlayerAID && winnerID != match.PlayerBID {
		return nil, errors.New("winner must be one of the match players")
	}

	now := time.Now().UTC()
	match.WinnerID = winnerID
	match.Status = "completed"
	match.CompletedAt = &now

	s.advanceTournamentLocked(tournament, winnerID, match.Round)
	if err := s.settleTournamentIfCompleteLocked(tournament); err != nil {
		return nil, err
	}
	s.audit = append(s.audit, &models.AuditLog{
		ID:        newUUID(),
		ActorID:   actorID,
		Action:    "tournament.match.completed",
		TargetID:  match.ID,
		Metadata:  map[string]string{"tournamentId": tournamentID, "winnerId": winnerID},
		CreatedAt: now,
	})
	if err := s.persistTournaments(); err != nil {
		return nil, err
	}
	if err := s.persistTournamentMatches(); err != nil {
		return nil, err
	}
	if err := s.persistWallets(); err != nil {
		return nil, err
	}
	if err := s.persistLedger(); err != nil {
		return nil, err
	}
	if err := s.persistAuditLogs(); err != nil {
		return nil, err
	}
	return match, nil
}

func (s *Store) advanceTournamentLocked(tournament *models.Tournament, winnerID string, completedRound int) {
	matches := s.tMatches[tournament.ID]
	incomplete := 0
	winners := make([]string, 0)
	for _, match := range matches {
		if match.Round != completedRound {
			continue
		}
		if match.Status != "completed" {
			incomplete++
			continue
		}
		winners = append(winners, match.WinnerID)
	}
	if incomplete > 0 || len(winners) <= 1 {
		return
	}
	for _, match := range matches {
		if match.Round == completedRound+1 {
			return
		}
	}
	now := time.Now().UTC()
	for i := 0; i < len(winners); i += 2 {
		next := &models.TournamentMatch{
			ID:           id.Match(),
			TournamentID: tournament.ID,
			Round:        completedRound + 1,
			MatchNumber:  (i / 2) + 1,
			PlayerAID:    winners[i],
			Status:       "scheduled",
			CreatedAt:    now,
		}
		if i+1 < len(winners) {
			next.PlayerBID = winners[i+1]
			profile := s.matchDifficultyProfileLocked(next.PlayerAID, next.PlayerBID, "tournament", tournament.Type)
			next.DifficultyRating = profile.Rating
			next.DifficultyProfile = &profile
			next.PuzzleVersion = game.CurrentPuzzleVersion()
			generationStarted := time.Now()
			generated, err := s.puzzleServiceLocked().Generate(context.Background(), puzzle.Request{
				Mode:              puzzle.ModeTournament,
				Purpose:           "tournament_match",
				MatchID:           next.ID,
				PlayerID:          tournament.ID,
				Shared:            true,
				DifficultyProfile: profile,
				PuzzleVersion:     next.PuzzleVersion,
			})
			if err != nil {
				next.Status = "cancelled"
				s.tMatches[tournament.ID] = append(s.tMatches[tournament.ID], next)
				continue
			}
			next.DifficultyProfile = &generated.DifficultyProfile
			next.DifficultyRating = generated.DifficultyProfile.Rating
			next.PlayerASeed = generated.Metadata.Seed
			next.PlayerBSeed = generated.Metadata.Seed
			next.PlayerANonce = generated.Metadata.Nonce
			next.PlayerBNonce = generated.Metadata.Nonce
			next.PlayerAHash = generated.Metadata.GenerationHash
			next.PlayerBHash = generated.Metadata.GenerationHash
			next.PuzzleMetadata = &generated.Metadata
			next.PlayerALines = cloneArrowLines(generated.Lines)
			next.PlayerBLines = cloneArrowLines(generated.Lines)
			s.recordPuzzleGenerationLocked(time.Since(generationStarted))
		} else {
			next.WinnerID = winners[i]
			next.Status = "completed"
			next.CompletedAt = &now
		}
		s.tMatches[tournament.ID] = append(s.tMatches[tournament.ID], next)
	}
	_ = winnerID
}

func (s *Store) settleTournamentIfCompleteLocked(tournament *models.Tournament) error {
	matches := s.tMatches[tournament.ID]
	if len(matches) == 0 {
		return nil
	}
	highestRound := 0
	var finalMatches []*models.TournamentMatch
	for _, match := range matches {
		if match.Round > highestRound {
			highestRound = match.Round
			finalMatches = []*models.TournamentMatch{match}
		} else if match.Round == highestRound {
			finalMatches = append(finalMatches, match)
		}
	}
	if len(finalMatches) != 1 || finalMatches[0].Status != "completed" || finalMatches[0].PrizeSettled || finalMatches[0].WinnerID == "" {
		return nil
	}
	final := finalMatches[0]
	for _, participant := range s.participants[tournament.ID] {
		wallet := s.wallets[participant.UserID]
		if wallet == nil {
			return errors.New("participant wallet not found")
		}
		before := wallet.DemoBalance
		if tournament.WalletType == "live" {
			if wallet.LiveLockedBalance < tournament.EntryFee || wallet.LiveBalance < tournament.EntryFee {
				return errors.New("insufficient locked live tournament entry balance")
			}
			before = wallet.LiveBalance
			wallet.LiveLockedBalance -= tournament.EntryFee
			wallet.LiveBalance -= tournament.EntryFee
		} else {
			if wallet.DemoLockedBalance < tournament.EntryFee || wallet.DemoBalance < tournament.EntryFee {
				return errors.New("insufficient locked demo tournament entry balance")
			}
			wallet.DemoLockedBalance -= tournament.EntryFee
			wallet.DemoBalance -= tournament.EntryFee
		}
		after := wallet.DemoBalance
		if tournament.WalletType == "live" {
			after = wallet.LiveBalance
		}
		s.ledger[participant.UserID] = append(s.ledger[participant.UserID], &models.LedgerEntry{
			ID:              newUUID(),
			UserID:          participant.UserID,
			TransactionType: models.TransactionTypeFee,
			Amount:          -tournament.EntryFee,
			BalanceBefore:   before,
			BalanceAfter:    after,
			Currency:        "USD",
			Reference:       "tournament-entry-settlement",
			Metadata:        map[string]string{"tournamentId": tournament.ID},
			CreatedAt:       time.Now().UTC(),
		})
	}
	wallet := s.wallets[final.WinnerID]
	if wallet == nil {
		return errors.New("winner wallet not found")
	}
	before := wallet.DemoBalance
	if tournament.WalletType == "live" {
		before = wallet.LiveBalance
		wallet.LiveBalance += tournament.PrizePool
	} else {
		wallet.DemoBalance += tournament.PrizePool
	}
	after := wallet.DemoBalance
	if tournament.WalletType == "live" {
		after = wallet.LiveBalance
	}
	entry := &models.LedgerEntry{
		ID:              newUUID(),
		UserID:          final.WinnerID,
		TransactionType: models.TransactionTypeReward,
		Amount:          tournament.PrizePool,
		BalanceBefore:   before,
		BalanceAfter:    after,
		Currency:        "USD",
		Reference:       "tournament-prize",
		Metadata:        map[string]string{"tournamentId": tournament.ID},
		CreatedAt:       time.Now().UTC(),
	}
	s.ledger[final.WinnerID] = append(s.ledger[final.WinnerID], entry)
	final.PrizeSettled = true
	tournament.Status = "completed"
	return nil
}

func newUUID() string {
	return id.New("obj")
}
