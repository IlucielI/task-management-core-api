package config_test

import (
	"testing"
	"time"

	"task-management/internal/config"
)

func TestConfig_LoadDefaults(t *testing.T) {
	cfg := config.Load()

	if cfg.AppName != "task-management-core-api" {
		t.Errorf("expected AppName 'task-management-core-api', got %q", cfg.AppName)
	}
	if cfg.AppEnv != "development" {
		t.Errorf("expected AppEnv 'development', got %q", cfg.AppEnv)
	}
	if cfg.HTTPPort != "8080" {
		t.Errorf("expected HTTPPort '8080', got %q", cfg.HTTPPort)
	}
	if cfg.Version != "0.1.0" {
		t.Errorf("expected Version '0.1.0', got %q", cfg.Version)
	}
	if cfg.GitHash != "dev" {
		t.Errorf("expected GitHash 'dev', got %q", cfg.GitHash)
	}
	if cfg.DBHost == "" {
		t.Error("expected non-empty DBHost")
	}
	if cfg.DBPoolMaxOpenConn <= 0 {
		t.Errorf("expected positive DBPoolMaxOpenConn, got %d", cfg.DBPoolMaxOpenConn)
	}
	if cfg.DBPoolMaxConnLifetime <= 0 {
		t.Errorf("expected positive DBPoolMaxConnLifetime, got %v", cfg.DBPoolMaxConnLifetime)
	}
	if cfg.DBPoolMaxConnIdleTime != 10*time.Minute && cfg.DBPoolMaxConnIdleTime <= 0 {
		t.Errorf("expected valid DBPoolMaxConnIdleTime, got %v", cfg.DBPoolMaxConnIdleTime)
	}
}

func TestConfig_CustomEnv(t *testing.T) {
	t.Setenv("APP_NAME", "custom-api")
	t.Setenv("APP_ENV", "production")
	t.Setenv("HTTP_PORT", "9000")
	t.Setenv("APP_VERSION", "1.0.0")
	t.Setenv("GIT_HASH", "commit123")

	cfg := config.Load()

	if cfg.AppName != "custom-api" {
		t.Errorf("expected AppName 'custom-api', got %q", cfg.AppName)
	}
	if cfg.AppEnv != "production" {
		t.Errorf("expected AppEnv 'production', got %q", cfg.AppEnv)
	}
	if cfg.HTTPPort != "9000" {
		t.Errorf("expected HTTPPort '9000', got %q", cfg.HTTPPort)
	}
	if cfg.Version != "1.0.0" {
		t.Errorf("expected Version '1.0.0', got %q", cfg.Version)
	}
	if cfg.GitHash != "commit123" {
		t.Errorf("expected GitHash 'commit123', got %q", cfg.GitHash)
	}
}

func TestConfig_DSN(t *testing.T) {
	cfg := config.Config{
		DBHost:    "127.0.0.1",
		DBPort:    "5432",
		DBUser:    "myuser",
		DBPass:    "mypassword",
		DBName:    "taskdb",
		DBSSLMode: "disable",
	}

	expected := "host=127.0.0.1 port=5432 user=myuser password=mypassword dbname=taskdb sslmode=disable"
	got := cfg.DSN()

	if got != expected {
		t.Errorf("expected DSN %q, got %q", expected, got)
	}
}

func TestConfig_LoadPostgresEnv(t *testing.T) {
	t.Setenv("POSTGRES_HOST", "custom-pg-host")
	t.Setenv("POSTGRES_PORT", "5433")
	t.Setenv("POSTGRES_USER", "custom-user")
	t.Setenv("POSTGRES_PASSWORD", "custom-secret")
	t.Setenv("POSTGRES_DB", "custom-dbname")
	t.Setenv("POSTGRES_SSLMODE", "require")

	cfg := config.Load()

	if cfg.DBHost != "custom-pg-host" {
		t.Errorf("expected DBHost 'custom-pg-host', got %q", cfg.DBHost)
	}
	if cfg.DBPort != "5433" {
		t.Errorf("expected DBPort '5433', got %q", cfg.DBPort)
	}
	if cfg.DBUser != "custom-user" {
		t.Errorf("expected DBUser 'custom-user', got %q", cfg.DBUser)
	}
	if cfg.DBPass != "custom-secret" {
		t.Errorf("expected DBPass 'custom-secret', got %q", cfg.DBPass)
	}
	if cfg.DBName != "custom-dbname" {
		t.Errorf("expected DBName 'custom-dbname', got %q", cfg.DBName)
	}
	if cfg.DBSSLMode != "require" {
		t.Errorf("expected DBSSLMode 'require', got %q", cfg.DBSSLMode)
	}
}
