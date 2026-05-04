package config

import (
	"testing"
	"time"
)

func TestLoadConfigAppliesEscalationDefaults(t *testing.T) {
	t.Setenv("APP_ENV", "development")
	t.Setenv("REDIS_HOST", "")
	t.Setenv("REDIS_PORT", "")
	t.Setenv("REDIS_PASSWORD", "")
	t.Setenv("REDIS_DB", "")
	t.Setenv("ESCALATION_ENABLED", "")
	t.Setenv("ESCALATION_THRESHOLD", "")
	t.Setenv("ESCALATION_WINDOW", "")
	t.Setenv("ESCALATION_COOLDOWN", "")

	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("expected defaults to load, got %v", err)
	}

	if cfg.RedisURL != "redis://:change-me-redis-password@localhost:6379/0" {
		t.Fatalf("unexpected RedisURL: %q", cfg.RedisURL)
	}
	if !cfg.EscalationEnabled {
		t.Fatal("expected escalation enabled by default")
	}
	if cfg.EscalationThreshold != 3 {
		t.Fatalf("expected threshold 3, got %d", cfg.EscalationThreshold)
	}
	if cfg.EscalationWindow != time.Minute {
		t.Fatalf("expected window 1m, got %s", cfg.EscalationWindow)
	}
	if cfg.EscalationCooldown != 5*time.Minute {
		t.Fatalf("expected cooldown 5m, got %s", cfg.EscalationCooldown)
	}
}

func TestLoadConfigBuildsRedisURLFromParts(t *testing.T) {
	t.Setenv("APP_ENV", "development")
	t.Setenv("REDIS_HOST", "redis")
	t.Setenv("REDIS_PORT", "6380")
	t.Setenv("REDIS_PASSWORD", "123456")
	t.Setenv("REDIS_DB", "2")

	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("expected config to load, got %v", err)
	}
	if cfg.RedisURL != "redis://:123456@redis:6380/2" {
		t.Fatalf("unexpected RedisURL: %q", cfg.RedisURL)
	}
}

func TestLoadConfigRejectsStagingDefaultRedisPassword(t *testing.T) {
	t.Setenv("APP_ENV", "staging")
	t.Setenv("REDIS_PASSWORD", "change-me-redis-password")

	_, err := LoadConfig()
	if err == nil {
		t.Fatal("expected staging default Redis password to fail")
	}
}

func TestLoadConfigRejectsProductionEmptyRedisPassword(t *testing.T) {
	t.Setenv("APP_ENV", "production")
	t.Setenv("REDIS_PASSWORD", "")

	_, err := LoadConfig()
	if err == nil {
		t.Fatal("expected production empty Redis password to fail")
	}
}
