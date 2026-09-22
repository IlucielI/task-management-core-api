package controllers

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/alicebob/miniredis/v2"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	goRedis "github.com/redis/go-redis/v9"

	redisAdapter "task-management/internal/adapters/redis"
	"task-management/internal/config"
	"task-management/internal/constants"
	"task-management/internal/dtos"
	"task-management/internal/pkg/ctxmeta"
	"task-management/internal/repositories"
	"task-management/internal/services"
)

func TestControllers_CreateTask_MissingIdempotencyKey(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctrls := New(config.Config{}, nil)

	body, _ := json.Marshal(dtos.CreateTaskRequest{Title: "New Task"})
	w := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(w)
	ctx.Request = httptest.NewRequest(http.MethodPost, "/v1/tasks", bytes.NewReader(body))
	ctx.Request.Header.Set("Content-Type", "application/json")
	// No Idempotency-Key header

	ctrls.CreateTask(ctx)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", w.Code)
	}
	var resp dtos.BaseResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}
	if resp.Success || resp.Code != constants.ResponseCodeBadRequest {
		t.Fatalf("expected bad request response, got: %+v", resp)
	}
}

func TestControllers_CreateTask_InvalidJSON(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctrls := New(config.Config{}, nil)

	w := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(w)
	ctx.Request = httptest.NewRequest(http.MethodPost, "/v1/tasks", bytes.NewReader([]byte("{malformed-json}")))
	ctx.Request.Header.Set("Content-Type", "application/json")
	ctx.Request.Header.Set("Idempotency-Key", uuid.New().String())

	ctrls.CreateTask(ctx)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", w.Code)
	}
}

func TestControllers_CreateTask_ValidationError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctrls := New(config.Config{}, nil)

	// Missing required title
	body, _ := json.Marshal(dtos.CreateTaskRequest{Title: ""})
	w := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(w)
	ctx.Request = httptest.NewRequest(http.MethodPost, "/v1/tasks", bytes.NewReader(body))
	ctx.Request.Header.Set("Content-Type", "application/json")
	ctx.Request.Header.Set("Idempotency-Key", uuid.New().String())

	ctrls.CreateTask(ctx)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", w.Code)
	}
}

func TestControllers_CreateTask_Unauthorized(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mr := miniredis.RunT(t)
	rawClient := goRedis.NewClient(&goRedis.Options{Addr: mr.Addr()})
	rdb := redisAdapter.NewWithClient(rawClient)
	repo := repositories.New(nil, rdb)
	svc := services.New(config.Config{}, repo, nil)
	ctrls := New(config.Config{}, svc)

	body, _ := json.Marshal(dtos.CreateTaskRequest{Title: "Task"})
	w := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(w)
	ctx.Request = httptest.NewRequest(http.MethodPost, "/v1/tasks", bytes.NewReader(body))
	ctx.Request.Header.Set("Content-Type", "application/json")
	ctx.Request.Header.Set("Idempotency-Key", uuid.New().String())
	// No auth context

	ctrls.CreateTask(ctx)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected status 401, got %d", w.Code)
	}
	var resp dtos.BaseResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}
	if resp.Code != constants.ResponseCodeUnauthorized {
		t.Fatalf("expected code UNAUTHORIZED, got: %s", resp.Code)
	}
}

func TestControllers_CreateTask_AssigneeNotInTeam(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mr := miniredis.RunT(t)
	rawClient := goRedis.NewClient(&goRedis.Options{Addr: mr.Addr()})
	rdb := redisAdapter.NewWithClient(rawClient)
	gormDB, mock := setupMockDB(t)
	repo := repositories.New(gormDB, rdb)
	svc := services.New(config.Config{}, repo, nil)
	ctrls := New(config.Config{}, svc)

	creatorID := uuid.New()
	creatorTeamID := uuid.New()
	assigneeID := uuid.New()
	otherTeamID := uuid.New()
	now := time.Now()

	// Assignee has different team
	assigneeRows := sqlmock.NewRows([]string{"id", "name", "email", "password", "team_id", "created_at", "updated_at"}).
		AddRow(assigneeID, "Bob", "bob@example.com", "pass", otherTeamID, now, now)
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "users" WHERE id = $1 ORDER BY "users"."id" LIMIT $2`)).
		WithArgs(assigneeID, 1).
		WillReturnRows(assigneeRows)

	body, _ := json.Marshal(dtos.CreateTaskRequest{
		Title:      "Team Task",
		AssigneeID: &assigneeID,
	})
	w := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(w)
	req := httptest.NewRequest(http.MethodPost, "/v1/tasks", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Idempotency-Key", uuid.New().String())

	// Attach authenticated user to request context
	req = req.WithContext(ctxmeta.WithAuthUser(req.Context(), ctxmeta.AuthUser{
		UserID: creatorID,
		TeamID: creatorTeamID,
	}))
	ctx.Request = req

	ctrls.CreateTask(ctx)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", w.Code)
	}
	var resp dtos.BaseResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}
	if resp.Message != constants.ErrAssigneeNotInTeam.Error() {
		t.Fatalf("expected ErrAssigneeNotInTeam message, got: %s", resp.Message)
	}
}

func TestControllers_CreateTask_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mr := miniredis.RunT(t)
	rawClient := goRedis.NewClient(&goRedis.Options{Addr: mr.Addr()})
	rdb := redisAdapter.NewWithClient(rawClient)
	gormDB, mock := setupMockDB(t)
	repo := repositories.New(gormDB, rdb)
	svc := services.New(config.Config{}, repo, nil)
	ctrls := New(config.Config{}, svc)

	creatorID := uuid.New()
	teamID := uuid.New()
	taskID := uuid.New()

	mock.ExpectBegin()
	mock.ExpectQuery(regexp.QuoteMeta(`INSERT INTO "tasks"`)).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(taskID))
	mock.ExpectQuery(regexp.QuoteMeta(`INSERT INTO "task_logs"`)).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(uuid.New()))
	mock.ExpectCommit()

	body, _ := json.Marshal(dtos.CreateTaskRequest{
		Title:       "Frontend Navigation",
		Description: "Build main navbar",
	})
	w := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(w)
	req := httptest.NewRequest(http.MethodPost, "/v1/tasks", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Idempotency-Key", uuid.New().String())

	req = req.WithContext(ctxmeta.WithAuthUser(req.Context(), ctxmeta.AuthUser{
		UserID: creatorID,
		TeamID: teamID,
	}))
	ctx.Request = req

	ctrls.CreateTask(ctx)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected status 201, got %d: %s", w.Code, w.Body.String())
	}
	var resp dtos.APIResponse[*dtos.TaskResponse]
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if !resp.Success || resp.Data == nil || resp.Data.Title != "Frontend Navigation" {
		t.Fatalf("unexpected response: %+v", resp)
	}
}

func TestControllers_CreateTask_InternalError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mr := miniredis.RunT(t)
	rawClient := goRedis.NewClient(&goRedis.Options{Addr: mr.Addr()})
	rdb := redisAdapter.NewWithClient(rawClient)
	gormDB, mock := setupMockDB(t)
	repo := repositories.New(gormDB, rdb)
	svc := services.New(config.Config{}, repo, nil)
	ctrls := New(config.Config{}, svc)

	mock.ExpectBegin()
	mock.ExpectQuery(regexp.QuoteMeta(`INSERT INTO "tasks"`)).
		WillReturnError(errors.New("db connection failure"))
	mock.ExpectRollback()

	body, _ := json.Marshal(dtos.CreateTaskRequest{Title: "Crashing task"})
	w := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(w)
	req := httptest.NewRequest(http.MethodPost, "/v1/tasks", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Idempotency-Key", uuid.New().String())

	req = req.WithContext(ctxmeta.WithAuthUser(req.Context(), ctxmeta.AuthUser{
		UserID: uuid.New(),
		TeamID: uuid.New(),
	}))
	ctx.Request = req

	ctrls.CreateTask(ctx)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected status 500, got %d", w.Code)
	}
}

