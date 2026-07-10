package config

import (
	"strings"
	"testing"
)

func TestLoadRuntimeSettingsReadsEnvironmentOverrides(t *testing.T) {
	t.Setenv("SKILL_ARENA_FEATURE_MEMORY_ARENA", "true")
	t.Setenv("SKILL_ARENA_TRUST_PVP_MIN", "82")
	t.Setenv("SKILL_ARENA_WITHDRAW_LIMIT_LIMITED", "123")

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
}

func TestProductionRequiresPostgreSQLAndSecretsFromEnvironment(t *testing.T) {
	t.Setenv("SKILL_ARENA_ENV", "production")
	t.Setenv("SKILL_ARENA_DATABASE_URL", "./data")
	t.Setenv("SKILL_ARENA_JWT_SECRET", "test-secret")

	_, err := Load()
	if err == nil || !strings.Contains(err.Error(), "PostgreSQL") {
		t.Fatalf("Load error = %v, want PostgreSQL requirement", err)
	}
}

func TestProductionConfigurationAcceptsExternalServiceURLs(t *testing.T) {
	t.Setenv("SKILL_ARENA_ENV", "production")
	t.Setenv("SKILL_ARENA_DATABASE_URL", "postgres://user:pass@localhost:5432/skillarena?sslmode=disable")
	t.Setenv("SKILL_ARENA_REDIS_URL", "redis://localhost:6379")
	t.Setenv("SKILL_ARENA_JWT_SECRET", "test-secret")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.Environment != "production" || cfg.RedisURL == "" || !isPostgresURL(cfg.DatabaseURL) {
		t.Fatalf("unexpected production config: %#v", cfg)
	}
}
