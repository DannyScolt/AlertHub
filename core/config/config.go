package config

import (
	"log"
	"os"
	"strconv"
	"time"

	"github.com/joho/godotenv"
)

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
	SwaggerEnabled          bool
}

func LoadConfig() *Config {
	if err := godotenv.Load(); err != nil && !os.IsNotExist(err) {
		log.Printf("failed to load .env: %v", err)
	}

	return &Config{
		AppEnv:                  getEnv("APP_ENV", "development"),
		HTTPPort:                getEnv("HTTP_PORT", "8080"),
		DatabaseURL:             getEnv("DATABASE_URL", "postgres://alerthub:alerthub@localhost:5432/alerthub?sslmode=disable"),
		JWTAccessSecret:         getEnv("JWT_ACCESS_SECRET", "change-me-access-secret"),
		JWTRefreshSecret:        getEnv("JWT_REFRESH_SECRET", "change-me-refresh-secret"),
		JWTAccessTTL:            getDurationEnv("JWT_ACCESS_TTL", 15*time.Minute),
		JWTRefreshTTL:           getDurationEnv("JWT_REFRESH_TTL", 30*24*time.Hour),
		DeviceAPIKeyPrefix:      getEnv("DEVICE_API_KEY_PREFIX", "ah_dev"),
		DeviceRestoreWindowDays: getIntEnv("DEVICE_RESTORE_WINDOW_DAYS", 90),
		SwaggerEnabled:          getBoolEnv("SWAGGER_ENABLED", true),
	}
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
