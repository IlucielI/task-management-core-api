package repositories

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

	"task-management/internal/dtos"
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

func TestRepositories_FindAllTeams_Success(t *testing.T) {
	gormDB, mock := setupMockDB(t)
	repo := New(gormDB)

	teamID := uuid.New()
	now := time.Now()

	rows := sqlmock.NewRows([]string{"id", "name", "created_at", "updated_at"}).
		AddRow(teamID, "Engineering", now, now)

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "teams" WHERE LOWER(name) LIKE $1 ORDER BY name ASC`)).
		WithArgs("%engineering%").
		WillReturnRows(rows)

	teams, err := repo.FindAllTeams(context.Background(), dtos.TeamFilterQuery{Name: "Engineering"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(teams) != 1 {
		t.Fatalf("expected 1 team, got %d", len(teams))
	}
	if teams[0].Name != "Engineering" {
		t.Fatalf("expected team name Engineering, got %s", teams[0].Name)
	}
}

func TestRepositories_FindAllTeams_DBError(t *testing.T) {
	gormDB, mock := setupMockDB(t)
	repo := New(gormDB)

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "teams" ORDER BY name ASC`)).
		WillReturnError(errors.New("db connection failed"))

	teams, err := repo.FindAllTeams(context.Background(), dtos.TeamFilterQuery{})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if teams != nil {
		t.Fatalf("expected nil teams on error, got: %+v", teams)
	}
}
