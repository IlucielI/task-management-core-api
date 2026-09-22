package controllers

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"task-management/internal/config"
	"task-management/internal/dtos"
	"task-management/internal/pkg/ctxmeta"
	"task-management/internal/repositories"
	"task-management/internal/services"
)

func TestControllers_ListUsers_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)
	gormDB, mock := setupMockDB(t)
	repo := repositories.New(gormDB)
	cfg := config.Config{}
	svc := services.New(cfg, repo, nil)
	ctrls := New(cfg, svc)

	callerID := uuid.New()
	teamID := uuid.New()
	user1ID := uuid.New()
	now := time.Now()

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT count(*) FROM "users" WHERE team_id = $1 AND LOWER(name) LIKE LOWER($2)`)).
		WithArgs(teamID, "%alice%").
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))

	rows := sqlmock.NewRows([]string{"id", "name", "email", "team_id", "created_at", "updated_at"}).
		AddRow(user1ID, "Alice", "alice@example.com", teamID, now, now)
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "users" WHERE team_id = $1 AND LOWER(name) LIKE LOWER($2) ORDER BY name ASC LIMIT $3`)).
		WithArgs(teamID, "%alice%", 10).
		WillReturnRows(rows)

	w := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(w)
	req := httptest.NewRequest(http.MethodGet, "/v1/users?page=1&limit=10&name=alice", nil)
	req = req.WithContext(ctxmeta.WithAuthUser(req.Context(), ctxmeta.AuthUser{
		UserID: callerID,
		TeamID: teamID,
	}))
	ctx.Request = req

	ctrls.ListUsers(ctx)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp dtos.APIResponse[dtos.ListUsersData]
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if !resp.Success || len(resp.Data.Items) != 1 || resp.Data.Items[0].Name != "Alice" {
		t.Fatalf("unexpected items: %+v", resp.Data)
	}
	if resp.Data.Metadata.Count != 1 || resp.Data.Metadata.Limit != 10 || resp.Data.Metadata.Page != 1 || resp.Data.Metadata.TotalPages != 1 {
		t.Fatalf("unexpected metadata: %+v", resp.Data.Metadata)
	}
}

func TestControllers_ListUsers_ValidationError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctrls := New(config.Config{}, nil)

	w := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(w)
	req := httptest.NewRequest(http.MethodGet, "/v1/users?team_id=00000000-0000-0000-0000-000000000000", nil)
	ctx.Request = req

	ctrls.ListUsers(ctx)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400 for nil UUID team_id, got %d: %s", w.Code, w.Body.String())
	}
}

func TestControllers_ListUsers_ServiceError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	gormDB, mock := setupMockDB(t)
	repo := repositories.New(gormDB)
	cfg := config.Config{}
	svc := services.New(cfg, repo, nil)
	ctrls := New(cfg, svc)

	callerID := uuid.New()
	teamID := uuid.New()

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT count(*) FROM "users" WHERE team_id = $1`)).
		WithArgs(teamID).
		WillReturnError(errors.New("db query error"))

	w := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(w)
	req := httptest.NewRequest(http.MethodGet, "/v1/users?page=1&limit=10", nil)
	req = req.WithContext(ctxmeta.WithAuthUser(req.Context(), ctxmeta.AuthUser{
		UserID: callerID,
		TeamID: teamID,
	}))
	ctx.Request = req

	ctrls.ListUsers(ctx)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected status 500 on db error, got %d: %s", w.Code, w.Body.String())
	}
}
