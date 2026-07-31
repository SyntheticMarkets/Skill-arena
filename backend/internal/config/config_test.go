package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestMetricsTokenCanBeLoadedFromSecretFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "metrics-token")
	if err := os.WriteFile(path, []byte("metrics-secret-from-mounted-file-at-least-32-characters\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("SKILL_ARENA_METRICS_TOKEN", "")
	t.Setenv("SKILL_ARENA_METRICS_TOKEN_FILE", path)
	if got := secretValue("SKILL_ARENA_METRICS_TOKEN", "SKILL_ARENA_METRICS_TOKEN_FILE"); got != "metrics-secret-from-mounted-file-at-least-32-characters" {
		t.Fatalf("secretValue=%q", got)
	}
}

func TestLoadRuntimeSettingsReadsEnvironmentOverrides(t *testing.T) {
	t.Setenv("SKILL_ARENA_FEATURE_MEMORY_ARENA", "true")
	t.Setenv("SKILL_ARENA_TRUST_PVP_MIN", "82")
	t.Setenv("SKILL_ARENA_WITHDRAW_LIMIT_LIMITED", "123")
	t.Setenv("SKILL_ARENA_REPLAY_VERIFICATION_KEYS", `{"retired-v1":"retired-replay-key-material-at-least-32-characters"}`)

	settings := LoadRuntimeSettings()
	if !settings.Features.MemoryArena {
		t.Fatal("expected memory arena feature flag to be enabled")
	}
	if settings.Trust.PvPMinimum != 82 {
		t.Fatalf("pvp trust minimum = %.0f, want 82", settings.Trust.PvPMinimum)
	}
	if settings.Trust.WithdrawalLimits["limited"] != 123 {
		t.Fatalf("limited withdrawal limit = %.0f, want 123", settings.Trust.WithdrawalLimits["limited"])
	}
	if settings.Security.ReplayVerificationKeys["retired-v1"] == "" {
		t.Fatal("expected historical replay verification key")
	}
}

func TestPaymentRoutingConfigurationIsProviderNeutral(t *testing.T) {
	t.Setenv("SKILL_ARENA_PAYMENT_ACTIVE_PROVIDERS", "peach,ozow")
	t.Setenv("SKILL_ARENA_PAYMENT_DEFAULT_PROVIDER", "peach")
	t.Setenv("SKILL_ARENA_PAYMENT_ROUTES", `{
		"PEACH":{"countries":["za"],"currencies":["zar"],"methods":["CARD"],"priority":120,"variableCostBps":250},
		"ozow":{"countries":["ZA"],"currencies":["ZAR"],"methods":["eft"],"priority":110,"fixedCostMinor":50}
	}`)

	settings := LoadRuntimeSettings().Payments
	if settings.DefaultProvider != "peach" || len(settings.ActiveProviders) != 2 || settings.ProviderConfigError != "" {
		t.Fatalf("payment settings=%+v", settings)
	}
	route := settings.ProviderRoutes["peach"]
	if route.Countries[0] != "ZA" || route.Currencies[0] != "ZAR" || route.Methods[0] != "card" {
		t.Fatalf("normalized route=%+v", route)
	}
}

func TestProductionRequiresPostgreSQLAndSecretsFromEnvironment(t *testing.T) {
	t.Setenv("SKILL_ARENA_ENV", "production")
	t.Setenv("SKILL_ARENA_METRICS_TOKEN", "production-test-metrics-token-at-least-32-characters")
	t.Setenv("SKILL_ARENA_DATABASE_URL", "./data")
	t.Setenv("SKILL_ARENA_JWT_SECRET", "test-secret")

	_, err := Load()
	if err == nil || !strings.Contains(err.Error(), "PostgreSQL") {
		t.Fatalf("Load error = %v, want PostgreSQL requirement", err)
	}
}

func TestProductionConfigurationAcceptsExternalServiceURLs(t *testing.T) {
	t.Setenv("SKILL_ARENA_ENV", "production")
	t.Setenv("SKILL_ARENA_METRICS_TOKEN", "production-test-metrics-token-at-least-32-characters")
	t.Setenv("SKILL_ARENA_DATABASE_URL", "postgres://user:pass@localhost:5432/skillarena?sslmode=disable")
	t.Setenv("SKILL_ARENA_REDIS_URL", "redis://localhost:6379")
	t.Setenv("SKILL_ARENA_JWT_SECRET", "production-test-jwt-secret-at-least-32-characters")
	t.Setenv("SKILL_ARENA_MFA_ENCRYPTION_KEY", "production-test-mfa-key-at-least-32-characters")
	t.Setenv("SKILL_ARENA_PUZZLE_SECRET", "production-test-puzzle-derivation-key-at-least-32-characters")
	t.Setenv("SKILL_ARENA_PUZZLE_ENCRYPTION_KEY", "production-test-puzzle-encryption-key-at-least-32-characters")
	t.Setenv("SKILL_ARENA_REPLAY_SIGNING_KEY", "production-test-replay-signing-key-at-least-32-characters")
	t.Setenv("SKILL_ARENA_REPLAY_SIGNING_KEY_ID", "production-replay-v1")
	t.Setenv("SKILL_ARENA_COOKIE_SECURE", "true")
	t.Setenv("SKILL_ARENA_PUBLIC_BASE_URL", "https://arena.example.com")
	t.Setenv("SKILL_ARENA_ADMIN_BASE_URL", "https://operations.example.com")
	t.Setenv("SKILL_ARENA_EMAIL_OUTBOX_ONLY", "false")
	t.Setenv("SKILL_ARENA_SMTP_HOST", "smtp.example.com")
	t.Setenv("SKILL_ARENA_SMTP_PORT", "587")
	t.Setenv("SKILL_ARENA_EMAIL_FROM", "security@arena.example.com")
	t.Setenv("SKILL_ARENA_ALLOWED_ORIGINS", "https://arena.example.com,https://operations.example.com")
	t.Setenv("SKILL_ARENA_STRIPE_SECRET_KEY", "sk_live_production_configuration_test")
	t.Setenv("SKILL_ARENA_STRIPE_WEBHOOK_SECRET", "whsec_production_configuration_test")
	t.Setenv("SKILL_ARENA_STRIPE_MODE", "live")
	t.Setenv("SKILL_ARENA_PAYMENT_ACTIVE_PROVIDERS", "stripe")
	t.Setenv("SKILL_ARENA_PAYMENT_DEFAULT_PROVIDER", "stripe")
	t.Setenv("SKILL_ARENA_STORAGE_PROVIDER", "s3-compatible")
	t.Setenv("SKILL_ARENA_S3_ENDPOINT", "https://objects.example.com")
	t.Setenv("SKILL_ARENA_S3_BUCKET", "skill-arena")
	t.Setenv("SKILL_ARENA_S3_ACCESS_KEY", "production-storage-access")
	t.Setenv("SKILL_ARENA_S3_SECRET_KEY", "production-storage-secret")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.Environment != "production" || cfg.RedisURL == "" || !isPostgresURL(cfg.DatabaseURL) {
		t.Fatalf("unexpected production config: %#v", cfg)
	}
}
