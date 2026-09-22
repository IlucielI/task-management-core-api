package repositories

import (
	"context"
	"errors"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
	"gorm.io/gorm"

	"task-management/internal/models"
)

func TestRepositories_FindUserByEmail(t *testing.T) {
	gormDB, mock := setupMockDB(t)
	repo := New(gormDB)

	userID := uuid.New()
	teamID := uuid.New()
	now := time.Now()

	// 1. Found user
	rows := sqlmock.NewRows([]string{"id", "name", "email", "password", "team_id", "created_at", "updated_at"}).
		AddRow(userID, "John Doe", "john@example.com", "hashed_pass", teamID, now, now)

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "users" WHERE LOWER(email) = $1 ORDER BY "users"."id" LIMIT $2`)).
		WithArgs("john@example.com", 1).
		WillReturnRows(rows)

	user, err := repo.FindUserByEmail(context.Background(), "JOHN@example.com")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if user == nil || user.Email != "john@example.com" {
		t.Fatalf("expected user with email john@example.com, got: %+v", user)
	}

	// 2. Not found
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "users" WHERE LOWER(email) = $1 ORDER BY "users"."id" LIMIT $2`)).
		WithArgs("unknown@example.com", 1).
		WillReturnError(gorm.ErrRecordNotFound)

	notFoundUser, err := repo.FindUserByEmail(context.Background(), "unknown@example.com")
	if err != nil {
		t.Fatalf("expected nil error on not found, got %v", err)
	}
	if notFoundUser != nil {
		t.Fatalf("expected nil user, got %+v", notFoundUser)
	}

	// 3. Database error
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "users" WHERE LOWER(email) = $1 ORDER BY "users"."id" LIMIT $2`)).
		WithArgs("error@example.com", 1).
		WillReturnError(errors.New("db query failure"))

	errUser, err := repo.FindUserByEmail(context.Background(), "error@example.com")
	if err == nil {
		t.Fatal("expected error on db failure, got nil")
	}
	if errUser != nil {
		t.Fatalf("expected nil user on error, got %+v", errUser)
	}
}

func TestRepositories_CreateUser(t *testing.T) {
	gormDB, mock := setupMockDB(t)
	repo := New(gormDB)

	userID := uuid.New()
	teamID := uuid.New()
	now := time.Now()

	user := &models.User{
		ID:        userID,
		Name:      "Jane Doe",
		Email:     "jane@example.com",
		Password:  "hashed_secret",
		TeamID:    teamID,
		CreatedAt: now,
		UpdatedAt: now,
	}

	// 1. Success create
	mock.ExpectBegin()
	mock.ExpectQuery(regexp.QuoteMeta(`INSERT INTO "users"`)).
		WillReturnRows(sqlmock.NewRows([]string{"id", "created_at", "updated_at"}).AddRow(userID, now, now))
	mock.ExpectCommit()

	if err := repo.CreateUser(context.Background(), user); err != nil {
		t.Fatalf("unexpected error creating user: %v", err)
	}

	// 2. Error create
	mock.ExpectBegin()
	mock.ExpectQuery(regexp.QuoteMeta(`INSERT INTO "users"`)).
		WillReturnError(errors.New("insert failed"))
	mock.ExpectRollback()

	if err := repo.CreateUser(context.Background(), user); err == nil {
		t.Fatal("expected error on insert failure, got nil")
	}
}
