package repositories

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"task-management/internal/constants"
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

	filter := TeamFilter{
		Name:    "Engineering",
		OrderBy: constants.SortByNameAsc,
		Offset:  0,
		Limit:   10,
	}

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT count(*) FROM "teams" WHERE LOWER(name) LIKE LOWER($1)`)).
		WithArgs("%Engineering%").
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))

	rows := sqlmock.NewRows([]string{"id", "name", "created_at", "updated_at"}).
		AddRow(teamID, "Engineering", now, now)

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "teams" WHERE LOWER(name) LIKE LOWER($1) ORDER BY name ASC LIMIT $2`)).
		WithArgs("%Engineering%", 10).
		WillReturnRows(rows)

	teams, total, err := repo.FindAllTeams(context.Background(), filter)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if total != 1 {
		t.Fatalf("expected total 1, got %d", total)
	}
	if len(teams) != 1 {
		t.Fatalf("expected 1 team, got %d", len(teams))
	}
	if teams[0].Name != "Engineering" {
		t.Fatalf("expected team name Engineering, got %s", teams[0].Name)
	}
}

func TestRepositories_FindAllTeams_CustomOrderBy(t *testing.T) {
	gormDB, mock := setupMockDB(t)
	repo := New(gormDB)

	teamID := uuid.New()
	now := time.Now()

	testCases := []struct {
		orderBy     constants.SortOrder
		expectedSQL string
	}{
		{constants.SortByNameDesc, `ORDER BY name DESC`},
		{constants.SortByLatest, `ORDER BY created_at DESC`},
		{constants.SortByEarliest, `ORDER BY created_at ASC`},
		{constants.SortByNameAsc, `ORDER BY name ASC`},
	}

	for _, tc := range testCases {
		filter := TeamFilter{
			OrderBy: tc.orderBy,
			Offset:  0,
			Limit:   5,
		}

		mock.ExpectQuery(regexp.QuoteMeta(`SELECT count(*) FROM "teams"`)).
			WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))

		rows := sqlmock.NewRows([]string{"id", "name", "created_at", "updated_at"}).
			AddRow(teamID, "Product", now, now)

		mock.ExpectQuery(regexp.QuoteMeta(fmt.Sprintf(`SELECT * FROM "teams" %s LIMIT $1`, tc.expectedSQL))).
			WithArgs(5).
			WillReturnRows(rows)

		teams, total, err := repo.FindAllTeams(context.Background(), filter)
		if err != nil {
			t.Fatalf("unexpected error for %s: %v", tc.orderBy, err)
		}
		if total != 1 || len(teams) != 1 {
			t.Fatalf("expected 1 team, got %d (total: %d)", len(teams), total)
		}
	}
}

func TestRepositories_FindAllTeams_CountDBError(t *testing.T) {
	gormDB, mock := setupMockDB(t)
	repo := New(gormDB)

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT count(*) FROM "teams"`)).
		WillReturnError(errors.New("db count failed"))

	teams, total, err := repo.FindAllTeams(context.Background(), TeamFilter{})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if total != 0 {
		t.Fatalf("expected 0 total on error, got %d", total)
	}
	if teams != nil {
		t.Fatalf("expected nil teams on error, got: %+v", teams)
	}
}

func TestRepositories_FindAllTeams_SelectDBError(t *testing.T) {
	gormDB, mock := setupMockDB(t)
	repo := New(gormDB)

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT count(*) FROM "teams"`)).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "teams" ORDER BY name ASC`)).
		WillReturnError(errors.New("db select failed"))

	teams, total, err := repo.FindAllTeams(context.Background(), TeamFilter{})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if total != 0 {
		t.Fatalf("expected 0 total on error, got %d", total)
	}
	if teams != nil {
		t.Fatalf("expected nil teams on error, got: %+v", teams)
	}
}

func TestRepositories_FindTeamByID(t *testing.T) {
	gormDB, mock := setupMockDB(t)
	repo := New(gormDB)

	teamID := uuid.New()
	now := time.Now()

	// 1. Success found
	rows := sqlmock.NewRows([]string{"id", "name", "created_at", "updated_at"}).
		AddRow(teamID, "Engineering", now, now)

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "teams" WHERE id = $1 ORDER BY "teams"."id" LIMIT $2`)).
		WithArgs(teamID, 1).
		WillReturnRows(rows)

	team, err := repo.FindTeamByID(context.Background(), teamID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if team == nil || team.ID != teamID {
		t.Fatalf("expected team ID %v, got %+v", teamID, team)
	}

	// 2. Not found (returns nil, nil)
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "teams" WHERE id = $1 ORDER BY "teams"."id" LIMIT $2`)).
		WithArgs(teamID, 1).
		WillReturnError(gorm.ErrRecordNotFound)

	teamNotFound, err := repo.FindTeamByID(context.Background(), teamID)
	if err != nil {
		t.Fatalf("expected nil error on not found, got %v", err)
	}
	if teamNotFound != nil {
		t.Fatalf("expected nil team on not found, got %+v", teamNotFound)
	}

	// 3. Database error
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "teams" WHERE id = $1 ORDER BY "teams"."id" LIMIT $2`)).
		WithArgs(teamID, 1).
		WillReturnError(errors.New("connection failed"))

	teamErr, err := repo.FindTeamByID(context.Background(), teamID)
	if err == nil {
		t.Fatal("expected error on db failure, got nil")
	}
	if teamErr != nil {
		t.Fatalf("expected nil team on error, got %+v", teamErr)
	}
}
