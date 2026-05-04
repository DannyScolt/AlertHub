package config

import (
	"errors"
	"fmt"
	"log"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

const defaultRedisPassword = "change-me-redis-password"

var ErrInsecureRedisPassword = errors.New("redis password must be changed in staging or production")

type Config struct {
	AppEnv                  string
	HTTPPort                string
	DatabaseURL             string
	JWTAccessSecret         string
	JWTRefreshSecret        string
	JWTAccessTTL            time.Duration
	JWTRefreshTTL           time.Duration
	DeviceAPIKeyPrefix      string
	DeviceRestoreWindowDays int
	RedisURL                string
	EscalationEnabled       bool
	EscalationThreshold     int
	EscalationWindow        time.Duration
	EscalationCooldown      time.Duration
	SwaggerEnabled          bool
}

func LoadConfig() (*Config, error) {
	if err := godotenv.Load(); err != nil && !os.IsNotExist(err) {
		log.Printf("failed to load .env: %v", err)
	}

	cfg := &Config{
		AppEnv:                  getEnv("APP_ENV", "development"),
		HTTPPort:                getEnv("HTTP_PORT", "8080"),
		DatabaseURL:             getEnv("DATABASE_URL", "postgres://alerthub:alerthub@localhost:5432/alerthub?sslmode=disable"),
		JWTAccessSecret:         getEnv("JWT_ACCESS_SECRET", "change-me-access-secret"),
		JWTRefreshSecret:        getEnv("JWT_REFRESH_SECRET", "change-me-refresh-secret"),
		JWTAccessTTL:            getDurationEnv("JWT_ACCESS_TTL", 15*time.Minute),
		JWTRefreshTTL:           getDurationEnv("JWT_REFRESH_TTL", 30*24*time.Hour),
		DeviceAPIKeyPrefix:      getEnv("DEVICE_API_KEY_PREFIX", "ah_dev"),
		DeviceRestoreWindowDays: getIntEnv("DEVICE_RESTORE_WINDOW_DAYS", 90),
		RedisURL: buildRedisURL(
			getEnv("REDIS_HOST", "localhost"),
			getEnv("REDIS_PORT", "6379"),
			getEnv("REDIS_PASSWORD", defaultRedisPassword),
			getEnv("REDIS_DB", "0"),
		),
		EscalationEnabled:   getBoolEnv("ESCALATION_ENABLED", true),
		EscalationThreshold: getIntEnv("ESCALATION_THRESHOLD", 3),
		EscalationWindow:    getDurationEnv("ESCALATION_WINDOW", time.Minute),
		EscalationCooldown:  getDurationEnv("ESCALATION_COOLDOWN", 5*time.Minute),
		SwaggerEnabled:      getBoolEnv("SWAGGER_ENABLED", true),
	}
	if err := validateRedisPassword(cfg); err != nil {
		return nil, err
	}
	log.Printf("configuration loaded: app_env=%s redis_url=%s escalation_enabled=%t", cfg.AppEnv, MaskRedisURL(cfg.RedisURL), cfg.EscalationEnabled)
	return cfg, nil
}

func buildRedisURL(host, port, password, db string) string {
	redisURL := url.URL{
		Scheme: "redis",
		Host:   host + ":" + port,
		User:   url.UserPassword("", password),
		Path:   "/" + db,
	}
	return redisURL.String()
}

func validateRedisPassword(cfg *Config) error {
	if cfg.AppEnv != "staging" && cfg.AppEnv != "production" {
		return nil
	}
	redisURL, err := url.Parse(cfg.RedisURL)
	if err != nil {
		return fmt.Errorf("parse redis url: %w", err)
	}
	password, hasPassword := redisURL.User.Password()
	if !hasPassword || password == "" || password == defaultRedisPassword {
		return ErrInsecureRedisPassword
	}
	return nil
}

func MaskRedisURL(rawURL string) string {
	redisURL, err := url.Parse(rawURL)
	if err != nil {
		return "<invalid redis url>"
	}
	if redisURL.User != nil {
		username := redisURL.User.Username()
		if username == "" {
			redisURL.User = url.UserPassword("", "****")
		} else {
			redisURL.User = url.UserPassword(username, "****")
		}
	}
	return strings.Replace(redisURL.String(), ":****@", ":****@", 1)
}

func getEnv(key, defaultValue string) string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	return value
}

func getDurationEnv(key string, defaultValue time.Duration) time.Duration {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}

	duration, err := time.ParseDuration(value)
	if err != nil {
		return defaultValue
	}
	return duration
}

func getIntEnv(key string, defaultValue int) int {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}

	parsed, err := strconv.Atoi(value)
	if err != nil {
		return defaultValue
	}
	return parsed
}

func getBoolEnv(key string, defaultValue bool) bool {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}

	parsed, err := strconv.ParseBool(value)
	if err != nil {
		return defaultValue
	}
	return parsed
}
