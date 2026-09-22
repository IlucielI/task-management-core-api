package services

import (
	"context"
	"errors"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"

	"task-management/internal/config"
	"task-management/internal/constants"
	"task-management/internal/dtos"
	"task-management/internal/pkg/ctxmeta"
	"task-management/internal/repositories"
)

func TestService_ListUsers(t *testing.T) {
	cfg := config.Config{}
	callerID := uuid.New()
	teamID := uuid.New()
	user1ID := uuid.New()
	user2ID := uuid.New()
	now := time.Now()

	t.Run("success_with_page_and_limit", func(t *testing.T) {
		gormDB, mock := setupMockDB(t)
		repo := repositories.New(gormDB)
		svc := New(cfg, repo, nil)

		ctx := ctxmeta.WithAuthUser(context.Background(), ctxmeta.AuthUser{
			UserID: callerID,
			TeamID: teamID,
		})

		query := dtos.ListUsersQuery{
			Page:  1,
			Limit: 10,
		}

		mock.ExpectQuery(regexp.QuoteMeta(`SELECT count(*) FROM "users" WHERE team_id = $1`)).
			WithArgs(teamID).
			WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(2))

		rows := sqlmock.NewRows([]string{"id", "name", "email", "team_id", "created_at", "updated_at"}).
			AddRow(user1ID, "Alice", "alice@example.com", teamID, now, now).
			AddRow(user2ID, "Bob", "bob@example.com", teamID, now, now)
		mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "users" WHERE team_id = $1 ORDER BY name ASC LIMIT $2`)).
			WithArgs(teamID, 10).
			WillReturnRows(rows)

		resp, err := svc.ListUsers(ctx, query)
		if err != nil {
			t.Fatalf("unexpected error listing users: %v", err)
		}
		if resp == nil || len(resp.Items) != 2 {
			t.Fatalf("expected 2 items, got %+v", resp)
		}
		if resp.Metadata.Count != 2 || resp.Metadata.Limit != 10 || resp.Metadata.Page != 1 || resp.Metadata.TotalPages != 1 {
			t.Fatalf("unexpected metadata: %+v", resp.Metadata)
		}
	})

	t.Run("success_with_name_and_email_filters", func(t *testing.T) {
		gormDB, mock := setupMockDB(t)
		repo := repositories.New(gormDB)
		svc := New(cfg, repo, nil)

		ctx := ctxmeta.WithAuthUser(context.Background(), ctxmeta.AuthUser{
			UserID: callerID,
			TeamID: teamID,
		})

		query := dtos.ListUsersQuery{
			Page:   1,
			Limit:  10,
			Name:   "Alice",
			Email:  "alice@example.com",
			TeamID: &teamID,
		}

		mock.ExpectQuery(regexp.QuoteMeta(`SELECT count(*) FROM "users" WHERE team_id = $1 AND LOWER(name) LIKE LOWER($2) AND LOWER(email) LIKE LOWER($3)`)).
			WithArgs(teamID, "%Alice%", "%alice@example.com%").
			WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))

		rows := sqlmock.NewRows([]string{"id", "name", "email", "team_id", "created_at", "updated_at"}).
			AddRow(user1ID, "Alice", "alice@example.com", teamID, now, now)
		mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "users" WHERE team_id = $1 AND LOWER(name) LIKE LOWER($2) AND LOWER(email) LIKE LOWER($3) ORDER BY name ASC LIMIT $4`)).
			WithArgs(teamID, "%Alice%", "%alice@example.com%", 10).
			WillReturnRows(rows)

		resp, err := svc.ListUsers(ctx, query)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if resp == nil || len(resp.Items) != 1 || resp.Items[0].Name != "Alice" {
			t.Fatalf("unexpected response: %+v", resp)
		}
	})

	t.Run("cross_team_query_returns_not_found", func(t *testing.T) {
		svc := New(cfg, nil, nil)

		ctx := ctxmeta.WithAuthUser(context.Background(), ctxmeta.AuthUser{
			UserID: callerID,
			TeamID: teamID,
		})

		otherTeamID := uuid.New()
		_, err := svc.ListUsers(ctx, dtos.ListUsersQuery{
			Page:   1,
			Limit:  10,
			TeamID: &otherTeamID,
		})
		if !errors.Is(err, constants.ErrTeamNotFound) {
			t.Fatalf("expected ErrTeamNotFound on cross-team query, got: %v", err)
		}
	})

	t.Run("unauthorized_missing_auth_user", func(t *testing.T) {
		svc := New(cfg, nil, nil)

		_, err := svc.ListUsers(context.Background(), dtos.ListUsersQuery{Page: 1, Limit: 10})
		if !errors.Is(err, constants.ErrUnauthorized) {
			t.Fatalf("expected ErrUnauthorized, got: %v", err)
		}
	})

	t.Run("db_error", func(t *testing.T) {
		gormDB, mock := setupMockDB(t)
		repo := repositories.New(gormDB)
		svc := New(cfg, repo, nil)

		ctx := ctxmeta.WithAuthUser(context.Background(), ctxmeta.AuthUser{
			UserID: callerID,
			TeamID: teamID,
		})

		mock.ExpectQuery(regexp.QuoteMeta(`SELECT count(*) FROM "users" WHERE team_id = $1`)).
			WithArgs(teamID).
			WillReturnError(errors.New("db error"))

		resp, err := svc.ListUsers(ctx, dtos.ListUsersQuery{Page: 1, Limit: 10})
		if err == nil {
			t.Fatal("expected error on db failure, got nil")
		}
		if resp != nil {
			t.Fatalf("expected nil response, got %+v", resp)
		}
	})

	t.Run("success_with_custom_order_by", func(t *testing.T) {
		gormDB, mock := setupMockDB(t)
		repo := repositories.New(gormDB)
		svc := New(cfg, repo, nil)

		ctx := ctxmeta.WithAuthUser(context.Background(), ctxmeta.AuthUser{
			UserID: callerID,
			TeamID: teamID,
		})

		query := dtos.ListUsersQuery{
			Page:    1,
			Limit:   10,
			OrderBy: constants.SortByNameDesc,
		}

		mock.ExpectQuery(regexp.QuoteMeta(`SELECT count(*) FROM "users" WHERE team_id = $1`)).
			WithArgs(teamID).
			WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))

		rows := sqlmock.NewRows([]string{"id", "name", "email", "team_id", "created_at", "updated_at"}).
			AddRow(user1ID, "Alice", "alice@example.com", teamID, now, now)
		mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "users" WHERE team_id = $1 ORDER BY name DESC LIMIT $2`)).
			WithArgs(teamID, 10).
			WillReturnRows(rows)

		resp, err := svc.ListUsers(ctx, query)
		if err != nil {
			t.Fatalf("unexpected error listing users: %v", err)
		}
		if resp == nil || len(resp.Items) != 1 {
			t.Fatalf("expected 1 item, got %+v", resp)
		}
	})
}

