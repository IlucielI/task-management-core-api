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
	"task-management/internal/constants"
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

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT count(*) FROM "teams" WHERE LOWER(name) LIKE LOWER($1)`)).
		WithArgs("%Product%").
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))

	rows := sqlmock.NewRows([]string{"id", "name", "created_at", "updated_at"}).
		AddRow(teamID, "Product", now, now)

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "teams" WHERE LOWER(name) LIKE LOWER($1) ORDER BY name ASC LIMIT $2`)).
		WithArgs("%Product%", 10).
		WillReturnRows(rows)

	query := dtos.ListTeamsQuery{
		Page:    1,
		Limit:   10,
		Name:    "Product",
		OrderBy: constants.SortByNameAsc,
	}

	resp, err := svc.GetTeams(context.Background(), query)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resp == nil {
		t.Fatal("expected non-nil response")
	}
	if len(resp.Items) != 1 {
		t.Fatalf("expected 1 team item, got %d", len(resp.Items))
	}
	if resp.Items[0].Name != "Product" {
		t.Fatalf("expected team name Product, got %s", resp.Items[0].Name)
	}
	if resp.Metadata.Count != 1 {
		t.Errorf("expected count 1, got %d", resp.Metadata.Count)
	}
	if resp.Metadata.Limit != 10 {
		t.Errorf("expected limit 10, got %d", resp.Metadata.Limit)
	}
	if resp.Metadata.Page != 1 {
		t.Errorf("expected page 1, got %d", resp.Metadata.Page)
	}
	if resp.Metadata.TotalPages != 1 {
		t.Errorf("expected total_pages 1, got %d", resp.Metadata.TotalPages)
	}
}

func TestService_GetTeams_PaginationCalculation(t *testing.T) {
	gormDB, mock := setupMockDB(t)
	repo := repositories.New(gormDB)
	svc := New(config.Config{}, repo, nil)

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT count(*) FROM "teams"`)).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(25))

	rows := sqlmock.NewRows([]string{"id", "name", "created_at", "updated_at"})
	for i := 0; i < 10; i++ {
		rows.AddRow(uuid.New(), "Team", time.Now(), time.Now())
	}

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "teams" ORDER BY name ASC LIMIT $1 OFFSET $2`)).
		WithArgs(10, 10).
		WillReturnRows(rows)

	query := dtos.ListTeamsQuery{
		Page:    2,
		Limit:   10,
		OrderBy: constants.SortByNameAsc,
	}

	resp, err := svc.GetTeams(context.Background(), query)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resp.Metadata.Count != 25 {
		t.Errorf("expected count 25, got %d", resp.Metadata.Count)
	}
	if resp.Metadata.TotalPages != 3 {
		t.Errorf("expected total_pages 3 for 25 items with limit 10, got %d", resp.Metadata.TotalPages)
	}
	if resp.Metadata.Page != 2 {
		t.Errorf("expected page 2, got %d", resp.Metadata.Page)
	}
}

func TestService_GetTeams_Error(t *testing.T) {
	gormDB, mock := setupMockDB(t)
	repo := repositories.New(gormDB)
	svc := New(config.Config{}, repo, nil)

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT count(*) FROM "teams"`)).
		WillReturnError(errors.New("db error"))

	query := dtos.ListTeamsQuery{
		Page:    1,
		Limit:   10,
		OrderBy: constants.SortByNameAsc,
	}

	resp, err := svc.GetTeams(context.Background(), query)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if resp != nil {
		t.Fatalf("expected nil response on error, got: %+v", resp)
	}
}
