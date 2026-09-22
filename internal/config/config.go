package config

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

type Config struct {
	AppName  string
	AppEnv   string
	HTTPPort string
	Version  string
	GitHash  string

	// Database Configuration
	DBHost                 string
	DBPort                 string
	DBUser                 string
	DBPass                 string
	DBName                 string
	DBSSLMode              string
	DBPoolMaxOpenConn      int
	DBPoolMaxIdleConn      int
	DBPoolMaxConnLifetime  time.Duration
	DBPoolMaxConnIdleTime  time.Duration
}

func Load() Config {
	return Config{
		AppName:  getEnv("APP_NAME", "task-management-core-api"),
		AppEnv:   getEnv("APP_ENV", "development"),
		HTTPPort: getEnv("HTTP_PORT", "8080"),
		Version:  getEnv("APP_VERSION", "0.1.0"),
		GitHash:  getEnv("GIT_HASH", "dev"),

		// Database settings (POSTGRES_* matching official Docker image, with DB_* fallback)
		DBHost:                 getEnvWithFallback("POSTGRES_HOST", "DB_HOST", "localhost"),
		DBPort:                 getEnvWithFallback("POSTGRES_PORT", "DB_PORT", "5432"),
		DBUser:                 getEnvWithFallback("POSTGRES_USER", "DB_USER", "postgres"),
		DBPass:                 getEnvWithFallback("POSTGRES_PASSWORD", "DB_PASS", "postgres"),
		DBName:                 getEnvWithFallback("POSTGRES_DB", "DB_NAME", "task_management"),
		DBSSLMode:              getEnvWithFallback("POSTGRES_SSLMODE", "DB_SSLMODE", "disable"),
		DBPoolMaxOpenConn:      getEnvInt("DB_POOL_MAX_OPEN_CONN", 25),
		DBPoolMaxIdleConn:      getEnvInt("DB_POOL_MAX_IDLE_CONN", 10),
		DBPoolMaxConnLifetime:  getEnvDuration("DB_POOL_MAX_CONN_LIFETIME", 30*time.Minute),
		DBPoolMaxConnIdleTime:  getEnvDuration("DB_POOL_MAX_CONN_IDLE_TIME", 10*time.Minute),
	}
}

// DSN constructs the PostgreSQL Data Source Name connection string.
func (c Config) DSN() string {
	return fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		c.DBHost, c.DBPort, c.DBUser, c.DBPass, c.DBName, c.DBSSLMode,
	)
}

func getEnv(key string, fallback string) string {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	return value
}

func getEnvWithFallback(primaryKey string, secondaryKey string, fallback string) string {
	if val := os.Getenv(primaryKey); val != "" {
		return val
	}
	if val := os.Getenv(secondaryKey); val != "" {
		return val
	}
	return fallback
}

func getEnvInt(key string, fallback int) int {
	valStr := os.Getenv(key)
	if valStr == "" {
		return fallback
	}
	val, err := strconv.Atoi(valStr)
	if err != nil {
		return fallback
	}
	return val
}

func getEnvDuration(key string, fallback time.Duration) time.Duration {
	valStr := os.Getenv(key)
	if valStr == "" {
		return fallback
	}
	val, err := time.ParseDuration(valStr)
	if err != nil {
		return fallback
	}
	return val
}
