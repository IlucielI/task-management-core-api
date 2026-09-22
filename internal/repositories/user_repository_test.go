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

func TestRepositories_FindUserByID(t *testing.T) {
	gormDB, mock := setupMockDB(t)
	repo := New(gormDB)

	userID := uuid.New()
	teamID := uuid.New()
	now := time.Now()

	// 1. Found user
	rows := sqlmock.NewRows([]string{"id", "name", "email", "password", "team_id", "created_at", "updated_at"}).
		AddRow(userID, "John Doe", "john@example.com", "hashed_pass", teamID, now, now)

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "users" WHERE id = $1 ORDER BY "users"."id" LIMIT $2`)).
		WithArgs(userID, 1).
		WillReturnRows(rows)

	user, err := repo.FindUserByID(context.Background(), userID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if user == nil || user.ID != userID {
		t.Fatalf("expected user with id %v, got: %+v", userID, user)
	}

	// 2. Not found
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "users" WHERE id = $1 ORDER BY "users"."id" LIMIT $2`)).
		WithArgs(userID, 1).
		WillReturnError(gorm.ErrRecordNotFound)

	notFoundUser, err := repo.FindUserByID(context.Background(), userID)
	if err != nil {
		t.Fatalf("expected nil error on not found, got %v", err)
	}
	if notFoundUser != nil {
		t.Fatalf("expected nil user, got %+v", notFoundUser)
	}

	// 3. Database error
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "users" WHERE id = $1 ORDER BY "users"."id" LIMIT $2`)).
		WithArgs(userID, 1).
		WillReturnError(errors.New("db query failure"))

	errUser, err := repo.FindUserByID(context.Background(), userID)
	if err == nil {
		t.Fatal("expected error on db failure, got nil")
	}
	if errUser != nil {
		t.Fatalf("expected nil user on error, got %+v", errUser)
	}
}

func TestRepositories_FindUsers(t *testing.T) {
	gormDB, mock := setupMockDB(t)
	repo := New(gormDB)

	user1ID := uuid.New()
	user2ID := uuid.New()
	teamID := uuid.New()
	now := time.Now()

	t.Run("success_with_all_filters", func(t *testing.T) {
		filter := UserFilter{
			TeamID: &teamID,
			Name:   "alice",
			Email:  "example.com",
			Offset: 0,
			Limit:  10,
		}

		// 1. Count query
		mock.ExpectQuery(regexp.QuoteMeta(`SELECT count(*) FROM "users" WHERE team_id = $1 AND LOWER(name) LIKE LOWER($2) AND LOWER(email) LIKE LOWER($3)`)).
			WithArgs(teamID, "%alice%", "%example.com%").
			WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))

		// 2. Find query
		rows := sqlmock.NewRows([]string{"id", "name", "email", "team_id", "created_at", "updated_at"}).
			AddRow(user1ID, "Alice", "alice@example.com", teamID, now, now)
		mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "users" WHERE team_id = $1 AND LOWER(name) LIKE LOWER($2) AND LOWER(email) LIKE LOWER($3) ORDER BY name ASC LIMIT $4`)).
			WithArgs(teamID, "%alice%", "%example.com%", 10).
			WillReturnRows(rows)

		users, total, err := repo.FindUsers(context.Background(), filter)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if total != 1 || len(users) != 1 || users[0].Name != "Alice" {
			t.Fatalf("unexpected result: total=%d, len=%d", total, len(users))
		}
	})

	t.Run("success_without_filters", func(t *testing.T) {
		filter := UserFilter{
			Offset: 0,
			Limit:  10,
		}

		mock.ExpectQuery(regexp.QuoteMeta(`SELECT count(*) FROM "users"`)).
			WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(2))

		rows := sqlmock.NewRows([]string{"id", "name", "email", "team_id", "created_at", "updated_at"}).
			AddRow(user1ID, "Alice", "alice@example.com", teamID, now, now).
			AddRow(user2ID, "Bob", "bob@example.com", teamID, now, now)
		mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "users" ORDER BY name ASC LIMIT $1`)).
			WithArgs(10).
			WillReturnRows(rows)

		users, total, err := repo.FindUsers(context.Background(), filter)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if total != 2 || len(users) != 2 {
			t.Fatalf("unexpected result: total=%d, len=%d", total, len(users))
		}
	})

	t.Run("count_db_error", func(t *testing.T) {
		filter := UserFilter{
			Limit: 10,
		}

		mock.ExpectQuery(regexp.QuoteMeta(`SELECT count(*) FROM "users"`)).
			WillReturnError(errors.New("count failed"))

		users, total, err := repo.FindUsers(context.Background(), filter)
		if err == nil {
			t.Fatal("expected error on count failure, got nil")
		}
		if users != nil || total != 0 {
			t.Fatalf("expected nil users and 0 total, got users=%v total=%d", users, total)
		}
	})

	t.Run("find_db_error", func(t *testing.T) {
		filter := UserFilter{
			Limit: 10,
		}

		mock.ExpectQuery(regexp.QuoteMeta(`SELECT count(*) FROM "users"`)).
			WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(5))

		mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "users" ORDER BY name ASC LIMIT $1`)).
			WithArgs(10).
			WillReturnError(errors.New("find failed"))

		users, total, err := repo.FindUsers(context.Background(), filter)
		if err == nil {
			t.Fatal("expected error on find failure, got nil")
		}
		if users != nil || total != 0 {
			t.Fatalf("expected nil users and 0 total, got users=%v total=%d", users, total)
		}
	})
}

