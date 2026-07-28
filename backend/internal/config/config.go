package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/mail"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	HTTPAddr    string
	DatabaseURL string
	JWTSecret   string
	Environment string
	RedisURL    string
	Settings    *RuntimeSettings
}

func Load() (*Config, error) {
	settings := LoadRuntimeSettings()
	cfg := &Config{
		HTTPAddr:    os.Getenv("SKILL_ARENA_HTTP_ADDR"),
		DatabaseURL: os.Getenv("SKILL_ARENA_DATABASE_URL"),
		JWTSecret:   os.Getenv("SKILL_ARENA_JWT_SECRET"),
		Environment: strings.ToLower(envString("SKILL_ARENA_ENV", "development")),
		RedisURL:    os.Getenv("SKILL_ARENA_REDIS_URL"),
		Settings:    settings,
	}

	if cfg.DatabaseURL == "" {
		if cfg.Environment == "production" {
			return nil, errors.New("SKILL_ARENA_DATABASE_URL is required in production")
		}
		cfg.DatabaseURL = "./data"
	}
	if cfg.Environment == "production" && !isPostgresURL(cfg.DatabaseURL) {
		return nil, errors.New("production requires PostgreSQL SKILL_ARENA_DATABASE_URL")
	}
	if cfg.JWTSecret == "" {
		return nil, errors.New("SKILL_ARENA_JWT_SECRET is required")
	}
	if cfg.Environment == "production" {
		if err := validateProduction(cfg); err != nil {
			return nil, err
		}
	}

	return cfg, nil
}

type RuntimeSettings struct {
	Difficulty  DifficultySettings
	Trust       TrustSettings
	Replay      ReplaySettings
	AntiBot     AntiBotSettings
	Tournament  TournamentSettings
	House       HouseSettings
	RateLimit   RateLimitSettings
	Features    FeatureFlags
	Cache       CacheSettings
	Admin       AdminSettings
	Maintenance MaintenanceSettings
	Workers     WorkerSettings
	Backup      BackupSettings
	Platform    PlatformSettings
	Security    SecuritySettings
	Email       EmailSettings
	MFA         MFASettings
	Payments    PaymentSettings
	Financial   FinancialSettings
	Storage     StorageSettings
	CORS        CORSSettings
	Realtime    RealtimeSettings
}

type DifficultySettings struct {
	BaseRating       int
	LevelMultiplier  int
	MinRating        int
	MaxRating        int
	MinLineCount     int
	MaxLineCount     int
	MinDepth         int
	MaxDepth         int
	MinBranching     int
	MaxBranching     int
	MinFalseRoutePct float64
	MaxFalseRoutePct float64
}

type TrustSettings struct {
	PvPMinimum       float64
	TrustedMinimum   float64
	StandardMinimum  float64
	LimitedMinimum   float64
	ReviewMinimum    float64
	WithdrawalLimits map[string]float64
}

type ReplaySettings struct {
	FastClickSeconds       float64
	HighFailedClickPercent float64
}

type AntiBotSettings struct {
	PrivacyClassification string
}

type TournamentSettings struct {
	DefaultMaxParticipants int
}

type HouseSettings struct {
	DefaultTargetHouseEdge float64
}

type RateLimitSettings struct {
	DefaultLimit       int
	DefaultWindow      time.Duration
	LoginLimit         int
	RegisterLimit      int
	MatchCreationLimit int
	ReplayLimit        int
	WithdrawalLimit    int
}

type FeatureFlags struct {
	MazeArena     bool
	MemoryArena   bool
	ReactionArena bool
	LogicArena    bool
	Marketplace   bool
	Guilds        bool
	Streaming     bool
}

type CacheSettings struct {
	DefaultTTLSeconds     int
	LeaderboardTTLSeconds int
	ProfileTTLSeconds     int
	SeasonTTLSeconds      int
	ConfigTTLSeconds      int
}

type AdminSettings struct {
	SuperAdminEmails  []string
	BaseURL           string
	AccessCookieName  string
	RefreshCookieName string
	AccessTTL         time.Duration
	RefreshTTL        time.Duration
	IdleTimeout       time.Duration
}

type MaintenanceSettings struct {
	Enabled          bool
	Message          string
	AllowSuperAdmins bool
}

type WorkerSettings struct {
	Enabled         bool
	PollSeconds     int
	MaxAttempts     int
	BackupHourUTC   int
	ShutdownSeconds int
}

type BackupSettings struct {
	Directory        string
	RetentionDays    int
	VerificationFile string
}

type PlatformSettings struct {
	LaunchPhase  string
	SupportEmail string
}

type SecuritySettings struct {
	PuzzleSecret        string
	PuzzleEncryptionKey string
	CookieSecure        bool
	CookieDomain        string
	AccessCookieName    string
	RefreshCookieName   string
	AccessTTL           time.Duration
	RefreshTTL          time.Duration
}

type EmailSettings struct {
	BaseURL    string
	From       string
	SMTPHost   string
	SMTPPort   int
	SMTPUser   string
	SMTPPass   string
	OutboxOnly bool
}

type MFASettings struct {
	EncryptionKey string
	Issuer        string
}

type PaymentSettings struct {
	DefaultProvider     string
	ActiveProviders     []string
	ProviderRoutes      map[string]PaymentProviderRoute
	ProviderConfigError string
	StripeSecretKey     string
	StripeWebhookSecret string
	StripeAPIBase       string
	StripeMode          string
}

type PaymentProviderRoute struct {
	Countries       []string `json:"countries"`
	Currencies      []string `json:"currencies"`
	Methods         []string `json:"methods"`
	Priority        int      `json:"priority"`
	VariableCostBPS int      `json:"variableCostBps"`
	FixedCostMinor  int64    `json:"fixedCostMinor"`
}

type FinancialSettings struct {
	DefaultCountry string
	Jurisdictions  map[string]JurisdictionFinancialPolicy
	PolicyError    string
}

type JurisdictionFinancialPolicy struct {
	Currency               string   `json:"currency"`
	MinimumAge             int      `json:"minimumAge"`
	PaymentMethods         []string `json:"paymentMethods"`
	SourceOfFundsRequired  bool     `json:"sourceOfFundsRequired"`
	DailyDepositMinor      int64    `json:"dailyDepositMinor"`
	MonthlyDepositMinor    int64    `json:"monthlyDepositMinor"`
	DailyWithdrawalMinor   int64    `json:"dailyWithdrawalMinor"`
	MonthlyWithdrawalMinor int64    `json:"monthlyWithdrawalMinor"`
}

func (f FinancialSettings) Policy(country string) (JurisdictionFinancialPolicy, bool) {
	policy, ok := f.Jurisdictions[strings.ToUpper(strings.TrimSpace(country))]
	return policy, ok
}

type StorageSettings struct {
	Provider  string
	LocalRoot string
	Endpoint  string
	Bucket    string
	AccessKey string
	SecretKey string
	Region    string
}

type CORSSettings struct {
	AllowedOrigins []string
}

type RealtimeSettings struct {
	QueueTTL        time.Duration
	PresenceTTL     time.Duration
	ReconnectWindow time.Duration
	MaxRatingGap    int
	MaxLatencyMS    int
	MaxMessageBytes int64
	ConnectionLimit int
}

func (p PaymentSettings) ProviderStatus() map[string]bool {
	return map[string]bool{
		"stripe": p.StripeSecretKey != "" && p.StripeWebhookSecret != "" && p.StripeAPIBase != "",
	}
}

func (p PaymentSettings) EnabledProviders() []string {
	enabled := make([]string, 0, len(p.ActiveProviders))
	for _, provider := range p.ActiveProviders {
		if p.ProviderStatus()[provider] {
			enabled = append(enabled, provider)
		}
	}
	return enabled
}

func LoadRuntimeSettings() *RuntimeSettings {
	return &RuntimeSettings{
		Difficulty: DifficultySettings{
			BaseRating:       envInt("SKILL_ARENA_DIFFICULTY_BASE_RATING", 8),
			LevelMultiplier:  envInt("SKILL_ARENA_DIFFICULTY_LEVEL_MULTIPLIER", 2),
			MinRating:        envInt("SKILL_ARENA_DIFFICULTY_MIN_RATING", 1),
			MaxRating:        envInt("SKILL_ARENA_DIFFICULTY_MAX_RATING", 100),
			MinLineCount:     envInt("SKILL_ARENA_DIFFICULTY_MIN_LINES", 35),
			MaxLineCount:     envInt("SKILL_ARENA_DIFFICULTY_MAX_LINES", 420),
			MinDepth:         envInt("SKILL_ARENA_DIFFICULTY_MIN_DEPTH", 2),
			MaxDepth:         envInt("SKILL_ARENA_DIFFICULTY_MAX_DEPTH", 24),
			MinBranching:     envInt("SKILL_ARENA_DIFFICULTY_MIN_BRANCHING", 1),
			MaxBranching:     envInt("SKILL_ARENA_DIFFICULTY_MAX_BRANCHING", 8),
			MinFalseRoutePct: envFloat("SKILL_ARENA_DIFFICULTY_MIN_FALSE_ROUTE", 0.04),
			MaxFalseRoutePct: envFloat("SKILL_ARENA_DIFFICULTY_MAX_FALSE_ROUTE", 0.34),
		},
		Trust: TrustSettings{
			PvPMinimum:      envFloat("SKILL_ARENA_TRUST_PVP_MIN", 70),
			TrustedMinimum:  envFloat("SKILL_ARENA_TRUST_TIER_TRUSTED", 90),
			StandardMinimum: envFloat("SKILL_ARENA_TRUST_TIER_STANDARD", 75),
			LimitedMinimum:  envFloat("SKILL_ARENA_TRUST_TIER_LIMITED", 60),
			ReviewMinimum:   envFloat("SKILL_ARENA_TRUST_TIER_REVIEW", 40),
			WithdrawalLimits: map[string]float64{
				"trusted":    envFloat("SKILL_ARENA_WITHDRAW_LIMIT_TRUSTED", 10000),
				"standard":   envFloat("SKILL_ARENA_WITHDRAW_LIMIT_STANDARD", 2500),
				"limited":    envFloat("SKILL_ARENA_WITHDRAW_LIMIT_LIMITED", 500),
				"review":     envFloat("SKILL_ARENA_WITHDRAW_LIMIT_REVIEW", 100),
				"restricted": envFloat("SKILL_ARENA_WITHDRAW_LIMIT_RESTRICTED", 0),
			},
		},
		Replay: ReplaySettings{
			FastClickSeconds:       envFloat("SKILL_ARENA_REPLAY_FAST_CLICK_SECONDS", 0.05),
			HighFailedClickPercent: envFloat("SKILL_ARENA_REPLAY_HIGH_FAILED_CLICK_PERCENT", 0.5),
		},
		AntiBot:    AntiBotSettings{PrivacyClassification: envString("SKILL_ARENA_TELEMETRY_PRIVACY_CLASS", "behavioral-security")},
		Tournament: TournamentSettings{DefaultMaxParticipants: envInt("SKILL_ARENA_TOURNAMENT_MAX_PARTICIPANTS", 64)},
		House:      HouseSettings{DefaultTargetHouseEdge: envFloat("SKILL_ARENA_HOUSE_TARGET_EDGE", 0.65)},
		RateLimit: RateLimitSettings{
			DefaultLimit:       envInt("SKILL_ARENA_RATE_DEFAULT_LIMIT", 180),
			DefaultWindow:      time.Duration(envInt("SKILL_ARENA_RATE_DEFAULT_WINDOW_SECONDS", 60)) * time.Second,
			LoginLimit:         envInt("SKILL_ARENA_RATE_LOGIN_LIMIT", 10),
			RegisterLimit:      envInt("SKILL_ARENA_RATE_REGISTER_LIMIT", 5),
			MatchCreationLimit: envInt("SKILL_ARENA_RATE_MATCH_LIMIT", 20),
			ReplayLimit:        envInt("SKILL_ARENA_RATE_REPLAY_LIMIT", 60),
			WithdrawalLimit:    envInt("SKILL_ARENA_RATE_WITHDRAWAL_LIMIT", 5),
		},
		Features: FeatureFlags{
			MazeArena:     envBool("SKILL_ARENA_FEATURE_MAZE_ARENA", true),
			MemoryArena:   envBool("SKILL_ARENA_FEATURE_MEMORY_ARENA", false),
			ReactionArena: envBool("SKILL_ARENA_FEATURE_REACTION_ARENA", false),
			LogicArena:    envBool("SKILL_ARENA_FEATURE_LOGIC_ARENA", false),
			Marketplace:   envBool("SKILL_ARENA_FEATURE_MARKETPLACE", false),
			Guilds:        envBool("SKILL_ARENA_FEATURE_GUILDS", false),
			Streaming:     envBool("SKILL_ARENA_FEATURE_STREAMING", false),
		},
		Cache: CacheSettings{
			DefaultTTLSeconds:     envInt("SKILL_ARENA_CACHE_DEFAULT_TTL_SECONDS", 30),
			LeaderboardTTLSeconds: envInt("SKILL_ARENA_CACHE_LEADERBOARD_TTL_SECONDS", 10),
			ProfileTTLSeconds:     envInt("SKILL_ARENA_CACHE_PROFILE_TTL_SECONDS", 10),
			SeasonTTLSeconds:      envInt("SKILL_ARENA_CACHE_SEASON_TTL_SECONDS", 30),
			ConfigTTLSeconds:      envInt("SKILL_ARENA_CACHE_CONFIG_TTL_SECONDS", 60),
		},
		Admin: AdminSettings{
			SuperAdminEmails: envList("SKILL_ARENA_SUPER_ADMINS", []string{
				"geldenhuysj0106@gmail.com",
				"skillarenagame@gmail.com",
			}),
			BaseURL:           envString("SKILL_ARENA_ADMIN_BASE_URL", "http://localhost:3100"),
			AccessCookieName:  envString("SKILL_ARENA_ADMIN_ACCESS_COOKIE", "sa_admin_access"),
			RefreshCookieName: envString("SKILL_ARENA_ADMIN_REFRESH_COOKIE", "sa_admin_refresh"),
			AccessTTL:         time.Duration(envInt("SKILL_ARENA_ADMIN_ACCESS_TTL_MINUTES", 10)) * time.Minute,
			RefreshTTL:        time.Duration(envInt("SKILL_ARENA_ADMIN_REFRESH_TTL_HOURS", 8)) * time.Hour,
			IdleTimeout:       time.Duration(envInt("SKILL_ARENA_ADMIN_IDLE_TIMEOUT_MINUTES", 30)) * time.Minute,
		},
		Maintenance: MaintenanceSettings{
			Enabled:          envBool("SKILL_ARENA_MAINTENANCE_ENABLED", false),
			Message:          envString("SKILL_ARENA_MAINTENANCE_MESSAGE", "Skill Arena is temporarily in maintenance mode."),
			AllowSuperAdmins: envBool("SKILL_ARENA_MAINTENANCE_ALLOW_SUPER_ADMINS", true),
		},
		Workers: WorkerSettings{
			Enabled:         envBool("SKILL_ARENA_WORKERS_ENABLED", true),
			PollSeconds:     envInt("SKILL_ARENA_WORKER_POLL_SECONDS", 5),
			MaxAttempts:     envInt("SKILL_ARENA_WORKER_MAX_ATTEMPTS", 3),
			BackupHourUTC:   envInt("SKILL_ARENA_BACKUP_HOUR_UTC", 2),
			ShutdownSeconds: envInt("SKILL_ARENA_SHUTDOWN_SECONDS", 20),
		},
		Backup: BackupSettings{
			Directory:        envString("SKILL_ARENA_BACKUP_DIR", "./backups"),
			RetentionDays:    envInt("SKILL_ARENA_BACKUP_RETENTION_DAYS", 30),
			VerificationFile: envString("SKILL_ARENA_BACKUP_VERIFICATION_FILE", "backup_manifest.json"),
		},
		Platform: PlatformSettings{
			LaunchPhase:  strings.ToUpper(envString("SKILL_ARENA_LAUNCH_PHASE", "PRE_LAUNCH")),
			SupportEmail: envString("SKILL_ARENA_SUPPORT_EMAIL", "support@skillarena.local"),
		},
		Security: SecuritySettings{
			PuzzleSecret:        envString("SKILL_ARENA_PUZZLE_SECRET", envString("SKILL_ARENA_JWT_SECRET", "local-development-puzzle-secret-derivation-key")),
			PuzzleEncryptionKey: envString("SKILL_ARENA_PUZZLE_ENCRYPTION_KEY", "local-development-puzzle-encryption-key-material"),
			CookieSecure:        envBool("SKILL_ARENA_COOKIE_SECURE", false),
			CookieDomain:        envString("SKILL_ARENA_COOKIE_DOMAIN", ""),
			AccessCookieName:    envString("SKILL_ARENA_ACCESS_COOKIE", "sa_access"),
			RefreshCookieName:   envString("SKILL_ARENA_REFRESH_COOKIE", "sa_refresh"),
			AccessTTL:           time.Duration(envInt("SKILL_ARENA_ACCESS_TTL_MINUTES", 15)) * time.Minute,
			RefreshTTL:          time.Duration(envInt("SKILL_ARENA_REFRESH_TTL_DAYS", 30)) * 24 * time.Hour,
		},
		Email: EmailSettings{
			BaseURL:    envString("SKILL_ARENA_PUBLIC_BASE_URL", "http://localhost:3000"),
			From:       envString("SKILL_ARENA_EMAIL_FROM", "no-reply@skillarena.local"),
			SMTPHost:   envString("SKILL_ARENA_SMTP_HOST", ""),
			SMTPPort:   envInt("SKILL_ARENA_SMTP_PORT", 587),
			SMTPUser:   envString("SKILL_ARENA_SMTP_USER", ""),
			SMTPPass:   envString("SKILL_ARENA_SMTP_PASS", ""),
			OutboxOnly: envBool("SKILL_ARENA_EMAIL_OUTBOX_ONLY", true),
		},
		MFA: MFASettings{
			EncryptionKey: envString("SKILL_ARENA_MFA_ENCRYPTION_KEY", envString("SKILL_ARENA_JWT_SECRET", "")),
			Issuer:        envString("SKILL_ARENA_MFA_ISSUER", "Skill Arena"),
		},
		Payments:  loadPaymentSettings(),
		Financial: loadFinancialSettings(),
		Storage: StorageSettings{
			Provider:  strings.ToLower(envString("SKILL_ARENA_STORAGE_PROVIDER", "local")),
			LocalRoot: envString("SKILL_ARENA_STORAGE_LOCAL_ROOT", "./data/objects"),
			Endpoint:  envString("SKILL_ARENA_S3_ENDPOINT", ""),
			Bucket:    envString("SKILL_ARENA_S3_BUCKET", ""),
			AccessKey: envString("SKILL_ARENA_S3_ACCESS_KEY", ""),
			SecretKey: envString("SKILL_ARENA_S3_SECRET_KEY", ""),
			Region:    envString("SKILL_ARENA_S3_REGION", "us-east-1"),
		},
		CORS: CORSSettings{
			AllowedOrigins: envList("SKILL_ARENA_ALLOWED_ORIGINS", []string{"http://localhost:3000", "http://localhost:5173"}),
		},
		Realtime: RealtimeSettings{
			QueueTTL:        time.Duration(envInt("SKILL_ARENA_REALTIME_QUEUE_TTL_SECONDS", 120)) * time.Second,
			PresenceTTL:     time.Duration(envInt("SKILL_ARENA_REALTIME_PRESENCE_TTL_SECONDS", 45)) * time.Second,
			ReconnectWindow: time.Duration(envInt("SKILL_ARENA_REALTIME_RECONNECT_SECONDS", 30)) * time.Second,
			MaxRatingGap:    envInt("SKILL_ARENA_REALTIME_MAX_RATING_GAP", 250),
			MaxLatencyMS:    envInt("SKILL_ARENA_REALTIME_MAX_LATENCY_MS", 500),
			MaxMessageBytes: int64(envInt("SKILL_ARENA_REALTIME_MAX_MESSAGE_BYTES", 8192)),
			ConnectionLimit: envInt("SKILL_ARENA_REALTIME_CONNECTIONS_PER_MINUTE", 10),
		},
	}
}

func validateProduction(cfg *Config) error {
	if len(cfg.JWTSecret) < 32 || strings.Contains(strings.ToLower(cfg.JWTSecret), "development") || cfg.JWTSecret == "test-secret" {
		return errors.New("production SKILL_ARENA_JWT_SECRET must be at least 32 characters and must not be a development secret")
	}
	if len(cfg.Settings.Security.PuzzleSecret) < 32 || len(cfg.Settings.Security.PuzzleEncryptionKey) < 32 {
		return errors.New("production puzzle derivation and encryption keys must each contain at least 32 characters")
	}
	if cfg.Settings.Security.PuzzleSecret == cfg.Settings.Security.PuzzleEncryptionKey {
		return errors.New("production puzzle derivation and encryption keys must be distinct")
	}
	if strings.Contains(strings.ToLower(cfg.Settings.Security.PuzzleSecret), "development") ||
		strings.Contains(strings.ToLower(cfg.Settings.Security.PuzzleEncryptionKey), "development") {
		return errors.New("production puzzle keys must not be development secrets")
	}
	if len(cfg.Settings.MFA.EncryptionKey) < 32 {
		return errors.New("production SKILL_ARENA_MFA_ENCRYPTION_KEY must be at least 32 characters")
	}
	if !cfg.Settings.Security.CookieSecure {
		return errors.New("production requires SKILL_ARENA_COOKIE_SECURE=true")
	}
	publicURL, err := url.Parse(cfg.Settings.Email.BaseURL)
	if err != nil || publicURL.Scheme != "https" || publicURL.Host == "" {
		return errors.New("production SKILL_ARENA_PUBLIC_BASE_URL must be an absolute HTTPS URL")
	}
	adminURL, err := url.Parse(cfg.Settings.Admin.BaseURL)
	if err != nil || adminURL.Scheme != "https" || adminURL.Host == "" {
		return errors.New("production SKILL_ARENA_ADMIN_BASE_URL must be an absolute HTTPS URL")
	}
	if cfg.Settings.Admin.AccessCookieName == cfg.Settings.Security.AccessCookieName ||
		cfg.Settings.Admin.RefreshCookieName == cfg.Settings.Security.RefreshCookieName {
		return errors.New("production Admin CRM cookies must be distinct from player cookies")
	}
	if cfg.Settings.Admin.AccessTTL <= 0 || cfg.Settings.Admin.AccessTTL > 15*time.Minute ||
		cfg.Settings.Admin.RefreshTTL <= 0 || cfg.Settings.Admin.RefreshTTL > 24*time.Hour ||
		cfg.Settings.Admin.IdleTimeout <= 0 || cfg.Settings.Admin.IdleTimeout > time.Hour {
		return errors.New("production Admin CRM session lifetimes exceed the privileged-session policy")
	}
	if cfg.Settings.Email.OutboxOnly {
		return errors.New("production requires SKILL_ARENA_EMAIL_OUTBOX_ONLY=false")
	}
	if cfg.Settings.Email.SMTPHost == "" || cfg.Settings.Email.SMTPPort <= 0 || cfg.Settings.Email.From == "" {
		return errors.New("production SMTP host, port, and from address are required")
	}
	if address, err := mail.ParseAddress(cfg.Settings.Platform.SupportEmail); err != nil || address.Address != cfg.Settings.Platform.SupportEmail {
		return errors.New("production SKILL_ARENA_SUPPORT_EMAIL must be a valid email address")
	}
	if cfg.RedisURL == "" {
		return errors.New("SKILL_ARENA_REDIS_URL is required in production")
	}
	if len(cfg.Settings.CORS.AllowedOrigins) == 0 {
		return errors.New("production CORS allowed origins are required")
	}
	if cfg.Settings.Realtime.QueueTTL <= 0 || cfg.Settings.Realtime.PresenceTTL <= 0 ||
		cfg.Settings.Realtime.ReconnectWindow <= 0 || cfg.Settings.Realtime.MaxRatingGap < 0 ||
		cfg.Settings.Realtime.MaxLatencyMS <= 0 || cfg.Settings.Realtime.MaxMessageBytes < 1024 ||
		cfg.Settings.Realtime.ConnectionLimit <= 0 {
		return errors.New("production realtime limits must be positive and valid")
	}
	for _, origin := range cfg.Settings.CORS.AllowedOrigins {
		parsed, err := url.Parse(origin)
		if err != nil || parsed.Scheme != "https" || parsed.Host == "" || strings.Contains(origin, "*") {
			return fmt.Errorf("production CORS origin %q must be an explicit HTTPS origin", origin)
		}
	}
	if cfg.Settings.Financial.PolicyError != "" {
		return fmt.Errorf("invalid SKILL_ARENA_FINANCIAL_POLICIES: %s", cfg.Settings.Financial.PolicyError)
	}
	if cfg.Settings.Payments.ProviderConfigError != "" {
		return fmt.Errorf("invalid SKILL_ARENA_PAYMENT_ROUTES: %s", cfg.Settings.Payments.ProviderConfigError)
	}
	if len(cfg.Settings.Payments.ActiveProviders) == 0 {
		return errors.New("production requires at least one active payment provider adapter")
	}
	if !containsString(cfg.Settings.Payments.ActiveProviders, cfg.Settings.Payments.DefaultProvider) {
		return errors.New("production default payment provider must be active")
	}
	if !strings.EqualFold(cfg.Settings.Storage.Provider, "s3") && !strings.EqualFold(cfg.Settings.Storage.Provider, "s3-compatible") {
		return errors.New("production requires S3-compatible object storage")
	}
	if cfg.Settings.Storage.Endpoint == "" || cfg.Settings.Storage.Bucket == "" ||
		cfg.Settings.Storage.AccessKey == "" || cfg.Settings.Storage.SecretKey == "" {
		return errors.New("production S3-compatible object storage credentials are incomplete")
	}
	return nil
}

func (s *RuntimeSettings) FeatureEnabled(name string) bool {
	switch strings.ToLower(name) {
	case "maze_arena", "maze":
		return s.Features.MazeArena
	case "memory_arena", "memory":
		return s.Features.MemoryArena
	case "reaction_arena", "reaction":
		return s.Features.ReactionArena
	case "logic_arena", "logic":
		return s.Features.LogicArena
	case "marketplace":
		return s.Features.Marketplace
	case "guilds":
		return s.Features.Guilds
	case "streaming":
		return s.Features.Streaming
	default:
		return false
	}
}

func loadFinancialSettings() FinancialSettings {
	settings := FinancialSettings{
		DefaultCountry: strings.ToUpper(envString("SKILL_ARENA_FINANCIAL_DEFAULT_COUNTRY", "ZA")),
		Jurisdictions: map[string]JurisdictionFinancialPolicy{
			"ZA": {
				Currency: "ZAR", MinimumAge: 18,
				PaymentMethods:        []string{"card", "eft", "bank_transfer"},
				SourceOfFundsRequired: true,
				DailyDepositMinor:     500_00, MonthlyDepositMinor: 5_000_00,
				DailyWithdrawalMinor: 250_00, MonthlyWithdrawalMinor: 2_500_00,
			},
		},
	}
	raw := strings.TrimSpace(os.Getenv("SKILL_ARENA_FINANCIAL_POLICIES"))
	if raw == "" {
		return settings
	}
	var policies map[string]JurisdictionFinancialPolicy
	if err := json.Unmarshal([]byte(raw), &policies); err != nil {
		settings.PolicyError = err.Error()
		return settings
	}
	normalized := make(map[string]JurisdictionFinancialPolicy, len(policies))
	for country, policy := range policies {
		country = strings.ToUpper(strings.TrimSpace(country))
		policy.Currency = strings.ToUpper(strings.TrimSpace(policy.Currency))
		if len(country) != 2 || len(policy.Currency) != 3 || policy.MinimumAge < 18 ||
			policy.DailyDepositMinor < 0 || policy.MonthlyDepositMinor < policy.DailyDepositMinor ||
			policy.DailyWithdrawalMinor < 0 || policy.MonthlyWithdrawalMinor < policy.DailyWithdrawalMinor {
			settings.PolicyError = "each jurisdiction requires a country, currency, minimum age, and valid minor-unit limits"
			return settings
		}
		normalized[country] = policy
	}
	if len(normalized) == 0 {
		settings.PolicyError = "at least one jurisdiction is required"
		return settings
	}
	settings.Jurisdictions = normalized
	return settings
}

func loadPaymentSettings() PaymentSettings {
	settings := PaymentSettings{
		DefaultProvider:     strings.ToLower(envString("SKILL_ARENA_PAYMENT_DEFAULT_PROVIDER", "")),
		ActiveProviders:     envList("SKILL_ARENA_PAYMENT_ACTIVE_PROVIDERS", nil),
		ProviderRoutes:      map[string]PaymentProviderRoute{},
		StripeSecretKey:     envString("SKILL_ARENA_STRIPE_SECRET_KEY", ""),
		StripeWebhookSecret: envString("SKILL_ARENA_STRIPE_WEBHOOK_SECRET", ""),
		StripeAPIBase:       envString("SKILL_ARENA_STRIPE_API_BASE", "https://api.stripe.com"),
		StripeMode:          strings.ToLower(envString("SKILL_ARENA_STRIPE_MODE", "sandbox")),
	}
	raw := strings.TrimSpace(os.Getenv("SKILL_ARENA_PAYMENT_ROUTES"))
	if raw != "" {
		var routes map[string]PaymentProviderRoute
		if err := json.Unmarshal([]byte(raw), &routes); err != nil {
			settings.ProviderConfigError = err.Error()
		} else {
			for provider, route := range routes {
				provider = strings.ToLower(strings.TrimSpace(provider))
				if provider == "" || route.Priority < 0 || route.VariableCostBPS < 0 || route.FixedCostMinor < 0 {
					settings.ProviderConfigError = "provider routes require an ID and non-negative priority and costs"
					break
				}
				route.Countries = normalizeUpperList(route.Countries)
				route.Currencies = normalizeUpperList(route.Currencies)
				route.Methods = normalizeLowerList(route.Methods)
				settings.ProviderRoutes[provider] = route
			}
		}
	}
	return settings
}

func normalizeUpperList(values []string) []string {
	result := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.ToUpper(strings.TrimSpace(value))
		if value != "" {
			result = append(result, value)
		}
	}
	return result
}

func normalizeLowerList(values []string) []string {
	result := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.ToLower(strings.TrimSpace(value))
		if value != "" {
			result = append(result, value)
		}
	}
	return result
}

func containsString(values []string, target string) bool {
	for _, value := range values {
		if strings.EqualFold(value, target) {
			return true
		}
	}
	return false
}

var runtimeSettings = LoadRuntimeSettings()

func Runtime() *RuntimeSettings {
	if runtimeSettings == nil {
		runtimeSettings = LoadRuntimeSettings()
	}
	return runtimeSettings
}

func envString(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func envInt(key string, fallback int) int {
	if value := os.Getenv(key); value != "" {
		parsed, err := strconv.Atoi(value)
		if err == nil {
			return parsed
		}
	}
	return fallback
}

func envFloat(key string, fallback float64) float64 {
	if value := os.Getenv(key); value != "" {
		parsed, err := strconv.ParseFloat(value, 64)
		if err == nil {
			return parsed
		}
	}
	return fallback
}

func envBool(key string, fallback bool) bool {
	if value := os.Getenv(key); value != "" {
		parsed, err := strconv.ParseBool(value)
		if err == nil {
			return parsed
		}
	}
	return fallback
}

func envList(key string, fallback []string) []string {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	parts := strings.Split(value, ",")
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		trimmed := strings.TrimSpace(strings.ToLower(part))
		if trimmed != "" {
			result = append(result, trimmed)
		}
	}
	if len(result) == 0 {
		return fallback
	}
	return result
}

func isPostgresURL(value string) bool {
	lower := strings.ToLower(strings.TrimSpace(value))
	return strings.HasPrefix(lower, "postgres://") || strings.HasPrefix(lower, "postgresql://")
}
