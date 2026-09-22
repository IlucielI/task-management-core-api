package database

import (
	"os"
	"testing"

	"task-management/internal/config"
)

func TestFindMigrationsDir(t *testing.T) {
	dir := findMigrationsDir()
	if dir == "" {
		t.Fatal("expected migrations directory to be found, got empty string")
	}

	info, err := os.Stat(dir)
	if err != nil {
		t.Fatalf("expected migrations dir %s to exist: %v", dir, err)
	}
	if !info.IsDir() {
		t.Fatalf("expected %s to be a directory", dir)
	}
}

func TestNewPostgres_Unreachable(t *testing.T) {
	cfg := config.Config{
		DBHost:            "127.0.0.1",
		DBPort:            "54329",
		DBUser:            "invalid",
		DBPass:            "invalid",
		DBName:            "invalid",
		DBSSLMode:         "disable",
		DBPoolMaxOpenConn: 5,
		DBPoolMaxIdleConn: 2,
	}

	pg, err := NewPostgres(cfg)
	if err == nil {
		t.Fatal("expected error connecting to unreachable postgres, got nil")
	}
	if pg != nil {
		t.Fatal("expected nil Postgres instance on connection failure")
	}
}

func TestPostgres_NilReceiver(t *testing.T) {
	var pg *Postgres
	if pg.DB() != nil {
		t.Fatal("expected nil DB for nil Postgres")
	}
	if pg.SQLDB() != nil {
		t.Fatal("expected nil SQLDB for nil Postgres")
	}
}
