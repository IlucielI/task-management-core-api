package config

import "os"

type Config struct {
	AppName  string
	AppEnv   string
	HTTPPort string
	Version  string
	GitHash  string
}

func Load() Config {
	return Config{
		AppName:  getEnv("APP_NAME", "task-management-core-api"),
		AppEnv:   getEnv("APP_ENV", "development"),
		HTTPPort: getEnv("HTTP_PORT", "8080"),
		Version:  getEnv("APP_VERSION", "0.1.0"),
		GitHash:  getEnv("GIT_HASH", "dev"),
	}
}

func getEnv(key string, fallback string) string {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}

	return value
}
