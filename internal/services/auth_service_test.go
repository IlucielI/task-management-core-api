package services

import (
	"context"
	"errors"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
	"gorm.io/gorm"

	"task-management/internal/config"
	"task-management/internal/constants"
	"task-management/internal/dtos"
	"task-management/internal/repositories"
)

func TestService_Register_Success(t *testing.T) {
	gormDB, mock := setupMockDB(t)
	repo := repositories.New(gormDB)
	svc := New(config.Config{}, nil, nil, nil)
	svc.SetRepositories(repo)

	teamID := uuid.New()
	userID := uuid.New()
	now := time.Now()

	req := dtos.RegisterRequest{
		Name:     "Alice Smith",
		Email:    "alice@example.com",
		Password: "strongPassword123",
		TeamID:   teamID,
	}

	// 1. Check team exists -> found
	teamRows := sqlmock.NewRows([]string{"id", "name", "created_at", "updated_at"}).
		AddRow(teamID, "Engineering", now, now)
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "teams" WHERE id = $1 ORDER BY "teams"."id" LIMIT $2`)).
		WithArgs(teamID, 1).
		WillReturnRows(teamRows)

	// 2. Check email uniqueness -> not found (unique)
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "users" WHERE LOWER(email) = $1 ORDER BY "users"."id" LIMIT $2`)).
		WithArgs("alice@example.com", 1).
		WillReturnError(gorm.ErrRecordNotFound)

	// 3. Create user
	mock.ExpectBegin()
	mock.ExpectQuery(regexp.QuoteMeta(`INSERT INTO "users"`)).
		WillReturnRows(sqlmock.NewRows([]string{"id", "created_at", "updated_at"}).AddRow(userID, now, now))
	mock.ExpectCommit()

	resp, err := svc.Register(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resp == nil {
		t.Fatal("expected non-nil user response")
	}
	if resp.Email != "alice@example.com" {
		t.Fatalf("expected email alice@example.com, got %s", resp.Email)
	}
	if resp.Name != "Alice Smith" {
		t.Fatalf("expected name Alice Smith, got %s", resp.Name)
	}
	if resp.TeamID != teamID {
		t.Fatalf("expected team ID %v, got %v", teamID, resp.TeamID)
	}
}

func TestService_Register_TeamNotFound(t *testing.T) {
	gormDB, mock := setupMockDB(t)
	repo := repositories.New(gormDB)
	svc := New(config.Config{}, nil, nil, nil)
	svc.SetRepositories(repo)

	teamID := uuid.New()
	req := dtos.RegisterRequest{
		Name:     "Bob",
		Email:    "bob@example.com",
		Password: "password123",
		TeamID:   teamID,
	}

	// Team not found
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "teams" WHERE id = $1 ORDER BY "teams"."id" LIMIT $2`)).
		WithArgs(teamID, 1).
		WillReturnError(gorm.ErrRecordNotFound)

	resp, err := svc.Register(context.Background(), req)
	if !errors.Is(err, constants.ErrTeamNotFound) {
		t.Fatalf("expected ErrTeamNotFound, got %v", err)
	}
	if resp != nil {
		t.Fatalf("expected nil response on team not found, got %+v", resp)
	}
}

func TestService_Register_EmailAlreadyExists(t *testing.T) {
	gormDB, mock := setupMockDB(t)
	repo := repositories.New(gormDB)
	svc := New(config.Config{}, nil, nil, nil)
	svc.SetRepositories(repo)

	teamID := uuid.New()
	existingUserID := uuid.New()
	now := time.Now()

	req := dtos.RegisterRequest{
		Name:     "Charlie",
		Email:    "charlie@example.com",
		Password: "password123",
		TeamID:   teamID,
	}

	// 1. Team found
	teamRows := sqlmock.NewRows([]string{"id", "name", "created_at", "updated_at"}).
		AddRow(teamID, "Product", now, now)
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "teams" WHERE id = $1 ORDER BY "teams"."id" LIMIT $2`)).
		WithArgs(teamID, 1).
		WillReturnRows(teamRows)

	// 2. Email found -> duplicate
	userRows := sqlmock.NewRows([]string{"id", "name", "email", "password", "team_id", "created_at", "updated_at"}).
		AddRow(existingUserID, "Existing Charlie", "charlie@example.com", "hash", teamID, now, now)
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "users" WHERE LOWER(email) = $1 ORDER BY "users"."id" LIMIT $2`)).
		WithArgs("charlie@example.com", 1).
		WillReturnRows(userRows)

	resp, err := svc.Register(context.Background(), req)
	if !errors.Is(err, constants.ErrEmailAlreadyExists) {
		t.Fatalf("expected ErrEmailAlreadyExists, got %v", err)
	}
	if resp != nil {
		t.Fatalf("expected nil response on email conflict, got %+v", resp)
	}
}

func TestService_Register_DatabaseErrors(t *testing.T) {
	gormDB, mock := setupMockDB(t)
	repo := repositories.New(gormDB)
	svc := New(config.Config{}, nil, nil, nil)
	svc.SetRepositories(repo)

	teamID := uuid.New()
	now := time.Now()

	req := dtos.RegisterRequest{
		Name:     "Dan",
		Email:    "dan@example.com",
		Password: "password123",
		TeamID:   teamID,
	}

	// 1. Team check DB failure
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "teams" WHERE id = $1 ORDER BY "teams"."id" LIMIT $2`)).
		WithArgs(teamID, 1).
		WillReturnError(errors.New("db error on team"))

	_, err := svc.Register(context.Background(), req)
	if err == nil {
		t.Fatal("expected error on team db failure, got nil")
	}

	// 2. Email check DB failure
	teamRows := sqlmock.NewRows([]string{"id", "name", "created_at", "updated_at"}).
		AddRow(teamID, "Engineering", now, now)
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "teams" WHERE id = $1 ORDER BY "teams"."id" LIMIT $2`)).
		WithArgs(teamID, 1).
		WillReturnRows(teamRows)
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "users" WHERE LOWER(email) = $1 ORDER BY "users"."id" LIMIT $2`)).
		WithArgs("dan@example.com", 1).
		WillReturnError(errors.New("db error on email"))

	_, err = svc.Register(context.Background(), req)
	if err == nil {
		t.Fatal("expected error on email db failure, got nil")
	}

	// 3. Create user DB failure
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "teams" WHERE id = $1 ORDER BY "teams"."id" LIMIT $2`)).
		WithArgs(teamID, 1).
		WillReturnRows(teamRows)
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "users" WHERE LOWER(email) = $1 ORDER BY "users"."id" LIMIT $2`)).
		WithArgs("dan@example.com", 1).
		WillReturnError(gorm.ErrRecordNotFound)
	mock.ExpectBegin()
	mock.ExpectQuery(regexp.QuoteMeta(`INSERT INTO "users"`)).
		WillReturnError(errors.New("db error on insert"))
	mock.ExpectRollback()

	_, err = svc.Register(context.Background(), req)
	if err == nil {
		t.Fatal("expected error on insert db failure, got nil")
	}
}
