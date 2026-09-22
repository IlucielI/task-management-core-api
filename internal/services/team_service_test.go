package services

import (
	"context"
	"errors"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"task-management/internal/config"
	"task-management/internal/dtos"
	"task-management/internal/repositories"
)

func setupMockDB(t *testing.T) (*gorm.DB, sqlmock.Sqlmock) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to open sqlmock: %v", err)
	}

	gormDB, err := gorm.Open(postgres.New(postgres.Config{
		Conn: db,
	}), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to initialize gorm with sqlmock: %v", err)
	}

	return gormDB, mock
}

func TestService_GetTeams_Success(t *testing.T) {
	gormDB, mock := setupMockDB(t)
	repo := repositories.New(gormDB)
	svc := New(config.Config{}, repo, nil)

	teamID := uuid.New()
	now := time.Now()

	rows := sqlmock.NewRows([]string{"id", "name", "created_at", "updated_at"}).
		AddRow(teamID, "Product", now, now)

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "teams" WHERE LOWER(name) LIKE $1 ORDER BY name ASC`)).
		WithArgs("%product%").
		WillReturnRows(rows)

	teams, err := svc.GetTeams(context.Background(), dtos.TeamFilterQuery{Name: "Product"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(teams) != 1 {
		t.Fatalf("expected 1 team, got %d", len(teams))
	}
	if teams[0].Name != "Product" {
		t.Fatalf("expected team name Product, got %s", teams[0].Name)
	}
}

func TestService_GetTeams_Error(t *testing.T) {
	gormDB, mock := setupMockDB(t)
	repo := repositories.New(gormDB)
	svc := New(config.Config{}, repo, nil)

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "teams" ORDER BY name ASC`)).
		WillReturnError(errors.New("db error"))

	teams, err := svc.GetTeams(context.Background(), dtos.TeamFilterQuery{})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if teams != nil {
		t.Fatalf("expected nil teams on error, got: %+v", teams)
	}
}
