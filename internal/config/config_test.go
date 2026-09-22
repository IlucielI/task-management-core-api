package config_test

import (
	"testing"

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
