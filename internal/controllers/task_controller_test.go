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
	if resp.Status != constants.ResponseStatusFail || resp.Code != constants.ResponseCodeBadRequest {
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
	if resp.Status != constants.ResponseStatusSuccess || resp.Data == nil || resp.Data.Title != "Frontend Navigation" {
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

func TestControllers_GetTaskByID_InvalidUUID(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctrls := New(config.Config{}, nil)

	cases := []struct {
		name string
		id   string
	}{
		{"malformed id", "invalid-uuid"},
		{"numeric id", "123456"},
		{"nil uuid", uuid.Nil.String()},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			ctx, _ := gin.CreateTestContext(w)
			ctx.Params = gin.Params{{Key: "id", Value: tc.id}}
			ctx.Request = httptest.NewRequest(http.MethodGet, "/v1/tasks/test", nil)

			ctrls.GetTaskByID(ctx)

			if w.Code != http.StatusBadRequest {
				t.Fatalf("expected status 400, got %d", w.Code)
			}
			var resp dtos.BaseResponse
			if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
				t.Fatalf("failed to unmarshal response: %v", err)
			}
			if resp.Status != constants.ResponseStatusFail || resp.Code != constants.ResponseCodeBadRequest {
				t.Fatalf("expected bad request response, got: %+v", resp)
			}
		})
	}
}

func TestControllers_GetTaskByID_Unauthorized(t *testing.T) {
	gin.SetMode(gin.TestMode)

	repo := repositories.New(nil)
	svc := services.New(config.Config{}, repo, nil)
	ctrls := New(config.Config{}, svc)

	taskID := uuid.New()
	w := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(w)
	ctx.Params = gin.Params{{Key: "id", Value: taskID.String()}}
	ctx.Request = httptest.NewRequest(http.MethodGet, "/v1/tasks/"+taskID.String(), nil)
	// No auth context attached

	ctrls.GetTaskByID(ctx)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected status 401, got %d", w.Code)
	}
	var resp dtos.BaseResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if resp.Code != constants.ResponseCodeUnauthorized {
		t.Fatalf("expected UNAUTHORIZED code, got %s", resp.Code)
	}
}

func TestControllers_GetTaskByID_NotFound(t *testing.T) {
	gin.SetMode(gin.TestMode)

	gormDB, mock := setupMockDB(t)
	repo := repositories.New(gormDB)
	svc := services.New(config.Config{}, repo, nil)
	ctrls := New(config.Config{}, svc)

	taskID := uuid.New()
	creatorID := uuid.New()
	teamID := uuid.New()

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "tasks" WHERE id = $1 AND "tasks"."deleted_at" IS NULL ORDER BY "tasks"."id" LIMIT $2`)).
		WithArgs(taskID, 1).
		WillReturnRows(sqlmock.NewRows([]string{"id", "title"}))

	w := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(w)
	ctx.Params = gin.Params{{Key: "id", Value: taskID.String()}}
	req := httptest.NewRequest(http.MethodGet, "/v1/tasks/"+taskID.String(), nil)
	req = req.WithContext(ctxmeta.WithAuthUser(req.Context(), ctxmeta.AuthUser{
		UserID: creatorID,
		TeamID: teamID,
	}))
	ctx.Request = req

	ctrls.GetTaskByID(ctx)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected status 404, got %d", w.Code)
	}
	var resp dtos.BaseResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if resp.Code != constants.ResponseCodeNotFound {
		t.Fatalf("expected NOT_FOUND code, got %s", resp.Code)
	}
}

func TestControllers_GetTaskByID_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	gormDB, mock := setupMockDB(t)
	repo := repositories.New(gormDB)
	svc := services.New(config.Config{}, repo, nil)
	ctrls := New(config.Config{}, svc)

	taskID := uuid.New()
	creatorID := uuid.New()
	teamID := uuid.New()
	now := time.Now()

	rows := sqlmock.NewRows([]string{"id", "title", "description", "status", "creator_id", "team_id", "created_at", "updated_at"}).
		AddRow(taskID, "Task Title", "Description", "todo", creatorID, teamID, now, now)
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "tasks" WHERE id = $1 AND "tasks"."deleted_at" IS NULL ORDER BY "tasks"."id" LIMIT $2`)).
		WithArgs(taskID, 1).
		WillReturnRows(rows)

	creatorRows := sqlmock.NewRows([]string{"id", "name", "email", "team_id", "created_at", "updated_at"}).
		AddRow(creatorID, "Creator User", "creator@example.com", teamID, now, now)
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "users" WHERE "users"."id" = $1`)).
		WithArgs(creatorID).
		WillReturnRows(creatorRows)

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "task_logs" WHERE "task_logs"."task_id" = $1 ORDER BY created_at desc`)).
		WithArgs(taskID).
		WillReturnRows(sqlmock.NewRows([]string{"id", "task_id"}))

	teamRows := sqlmock.NewRows([]string{"id", "name", "created_at", "updated_at"}).
		AddRow(teamID, "Core Team", now, now)
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "teams" WHERE "teams"."id" = $1`)).
		WithArgs(teamID).
		WillReturnRows(teamRows)

	w := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(w)
	ctx.Params = gin.Params{{Key: "id", Value: taskID.String()}}
	req := httptest.NewRequest(http.MethodGet, "/v1/tasks/"+taskID.String(), nil)
	req = req.WithContext(ctxmeta.WithAuthUser(req.Context(), ctxmeta.AuthUser{
		UserID: creatorID,
		TeamID: teamID,
	}))
	ctx.Request = req

	ctrls.GetTaskByID(ctx)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}
	var resp dtos.APIResponse[*dtos.TaskResponse]
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if resp.Status != constants.ResponseStatusSuccess || resp.Data == nil || resp.Data.ID != taskID || resp.Data.Title != "Task Title" {
		t.Fatalf("unexpected task response: %+v", resp)
	}
	if resp.Data.Creator == nil || resp.Data.Creator.Name != "Creator User" {
		t.Fatalf("expected creator to be populated, got: %+v", resp.Data.Creator)
	}
	if resp.Data.Team == nil || resp.Data.Team.Name != "Core Team" {
		t.Fatalf("expected team to be populated, got: %+v", resp.Data.Team)
	}
}

func TestControllers_GetTaskByID_InternalError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	gormDB, mock := setupMockDB(t)
	repo := repositories.New(gormDB)
	svc := services.New(config.Config{}, repo, nil)
	ctrls := New(config.Config{}, svc)

	taskID := uuid.New()
	creatorID := uuid.New()
	teamID := uuid.New()

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "tasks" WHERE id = $1 AND "tasks"."deleted_at" IS NULL ORDER BY "tasks"."id" LIMIT $2`)).
		WithArgs(taskID, 1).
		WillReturnError(errors.New("db query failure"))

	w := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(w)
	ctx.Params = gin.Params{{Key: "id", Value: taskID.String()}}
	req := httptest.NewRequest(http.MethodGet, "/v1/tasks/"+taskID.String(), nil)
	req = req.WithContext(ctxmeta.WithAuthUser(req.Context(), ctxmeta.AuthUser{
		UserID: creatorID,
		TeamID: teamID,
	}))
	ctx.Request = req

	ctrls.GetTaskByID(ctx)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected status 500, got %d", w.Code)
	}
	var resp dtos.BaseResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if resp.Code != constants.ResponseCodeInternalError {
		t.Fatalf("expected INTERNAL_ERROR code, got %s", resp.Code)
	}
}

func TestControllers_DeleteTask_InvalidUUID(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctrls := New(config.Config{}, nil)

	w := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(w)
	ctx.Params = gin.Params{{Key: "id", Value: "invalid-uuid"}}
	ctx.Request = httptest.NewRequest(http.MethodDelete, "/v1/tasks/invalid-uuid", nil)

	ctrls.DeleteTask(ctx)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", w.Code)
	}
	var resp dtos.BaseResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if resp.Status != constants.ResponseStatusFail || resp.Code != constants.ResponseCodeBadRequest {
		t.Fatalf("expected bad request response, got: %+v", resp)
	}
}

func TestControllers_DeleteTask_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	gormDB, mock := setupMockDB(t)
	repo := repositories.New(gormDB)
	svc := services.New(config.Config{}, repo, nil)
	ctrls := New(config.Config{}, svc)

	taskID := uuid.New()
	creatorID := uuid.New()
	teamID := uuid.New()
	now := time.Now()

	// 1. FindTaskByID
	rows := sqlmock.NewRows([]string{"id", "title", "description", "status", "creator_id", "assignee_id", "team_id", "created_at", "updated_at"}).
		AddRow(taskID, "Task Title", "Description", "todo", creatorID, nil, teamID, now, now)
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "tasks" WHERE id = $1 AND "tasks"."deleted_at" IS NULL ORDER BY "tasks"."id" LIMIT $2`)).
		WithArgs(taskID, 1).
		WillReturnRows(rows)

	// 2. DeleteTaskWithLog
	mock.ExpectBegin()
	mock.ExpectExec(regexp.QuoteMeta(`UPDATE "tasks" SET "deleted_at"=$1 WHERE id = $2 AND "tasks"."deleted_at" IS NULL`)).
		WithArgs(sqlmock.AnyArg(), taskID).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectQuery(regexp.QuoteMeta(`INSERT INTO "task_logs"`)).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(uuid.New()))
	mock.ExpectCommit()

	w := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(w)
	ctx.Params = gin.Params{{Key: "id", Value: taskID.String()}}
	req := httptest.NewRequest(http.MethodDelete, "/v1/tasks/"+taskID.String(), nil)
	req = req.WithContext(ctxmeta.WithAuthUser(req.Context(), ctxmeta.AuthUser{
		UserID: creatorID,
		TeamID: teamID,
	}))
	ctx.Request = req

	ctrls.DeleteTask(ctx)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}
	var resp dtos.BaseResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if resp.Status != constants.ResponseStatusSuccess || resp.Code != constants.ResponseCodeSuccess || resp.Message != "Task deleted successfully" {
		t.Fatalf("unexpected delete response: %+v", resp)
	}
}

func TestControllers_DeleteTask_NotFound(t *testing.T) {
	gin.SetMode(gin.TestMode)

	gormDB, mock := setupMockDB(t)
	repo := repositories.New(gormDB)
	svc := services.New(config.Config{}, repo, nil)
	ctrls := New(config.Config{}, svc)

	taskID := uuid.New()
	creatorID := uuid.New()
	teamID := uuid.New()

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "tasks" WHERE id = $1 AND "tasks"."deleted_at" IS NULL ORDER BY "tasks"."id" LIMIT $2`)).
		WithArgs(taskID, 1).
		WillReturnRows(sqlmock.NewRows([]string{"id", "title"}))

	w := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(w)
	ctx.Params = gin.Params{{Key: "id", Value: taskID.String()}}
	req := httptest.NewRequest(http.MethodDelete, "/v1/tasks/"+taskID.String(), nil)
	req = req.WithContext(ctxmeta.WithAuthUser(req.Context(), ctxmeta.AuthUser{
		UserID: creatorID,
		TeamID: teamID,
	}))
	ctx.Request = req

	ctrls.DeleteTask(ctx)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected status 404, got %d", w.Code)
	}
	var resp dtos.BaseResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if resp.Code != constants.ResponseCodeNotFound {
		t.Fatalf("expected NOT_FOUND, got %s", resp.Code)
	}
}

func TestControllers_DeleteTask_InternalError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	gormDB, mock := setupMockDB(t)
	repo := repositories.New(gormDB)
	svc := services.New(config.Config{}, repo, nil)
	ctrls := New(config.Config{}, svc)

	taskID := uuid.New()
	creatorID := uuid.New()
	teamID := uuid.New()

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "tasks" WHERE id = $1 AND "tasks"."deleted_at" IS NULL ORDER BY "tasks"."id" LIMIT $2`)).
		WithArgs(taskID, 1).
		WillReturnError(errors.New("db error"))

	w := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(w)
	ctx.Params = gin.Params{{Key: "id", Value: taskID.String()}}
	req := httptest.NewRequest(http.MethodDelete, "/v1/tasks/"+taskID.String(), nil)
	req = req.WithContext(ctxmeta.WithAuthUser(req.Context(), ctxmeta.AuthUser{
		UserID: creatorID,
		TeamID: teamID,
	}))
	ctx.Request = req

	ctrls.DeleteTask(ctx)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected status 500, got %d", w.Code)
	}
	var resp dtos.BaseResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if resp.Code != constants.ResponseCodeInternalError {
		t.Fatalf("expected INTERNAL_ERROR code, got %s", resp.Code)
	}
}

func TestControllers_ListTasks_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	gormDB, mock := setupMockDB(t)
	repo := repositories.New(gormDB)
	svc := services.New(config.Config{}, repo, nil)
	ctrls := New(config.Config{}, svc)

	creatorID := uuid.New()
	teamID := uuid.New()
	taskID := uuid.New()
	now := time.Now()

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT count(*) FROM "tasks" WHERE team_id = $1 AND status = $2 AND LOWER(title) LIKE LOWER($3) AND "tasks"."deleted_at" IS NULL`)).
		WithArgs(teamID, "todo", "%fix%").
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))

	rows := sqlmock.NewRows([]string{"id", "title", "description", "status", "creator_id", "team_id", "created_at", "updated_at"}).
		AddRow(taskID, "Fix issue", "desc", "todo", creatorID, teamID, now, now)
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "tasks" WHERE team_id = $1 AND status = $2 AND LOWER(title) LIKE LOWER($3) AND "tasks"."deleted_at" IS NULL ORDER BY created_at DESC LIMIT $4`)).
		WithArgs(teamID, "todo", "%fix%", 10).
		WillReturnRows(rows)

	w := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(w)
	req := httptest.NewRequest(http.MethodGet, "/v1/tasks?page=1&limit=10&status=todo&title=fix", nil)
	req = req.WithContext(ctxmeta.WithAuthUser(req.Context(), ctxmeta.AuthUser{
		UserID: creatorID,
		TeamID: teamID,
	}))
	ctx.Request = req

	ctrls.ListTasks(ctx)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp dtos.APIResponse[dtos.ListTasksData]
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if resp.Status != constants.ResponseStatusSuccess || len(resp.Data.Items) != 1 || resp.Data.Items[0].Title != "Fix issue" {
		t.Fatalf("unexpected items: %+v", resp.Data)
	}
	if resp.Data.Metadata.Count != 1 || resp.Data.Metadata.Limit != 10 || resp.Data.Metadata.Page != 1 || resp.Data.Metadata.TotalPages != 1 {
		t.Fatalf("unexpected metadata: %+v", resp.Data.Metadata)
	}
}

func TestControllers_ListTasks_InvalidQuery(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctrls := New(config.Config{}, nil)

	w := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(w)
	ctx.Request = httptest.NewRequest(http.MethodGet, "/v1/tasks?status=invalid_status", nil)

	ctrls.ListTasks(ctx)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", w.Code)
	}
	var resp dtos.BaseResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if resp.Status != constants.ResponseStatusFail || resp.Code != constants.ResponseCodeBadRequest {
		t.Fatalf("expected bad request response, got: %+v", resp)
	}
}

func TestControllers_ListTasks_InternalError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	gormDB, mock := setupMockDB(t)
	repo := repositories.New(gormDB)
	svc := services.New(config.Config{}, repo, nil)
	ctrls := New(config.Config{}, svc)

	creatorID := uuid.New()
	teamID := uuid.New()

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT count(*) FROM "tasks" WHERE team_id = $1 AND "tasks"."deleted_at" IS NULL`)).
		WithArgs(teamID).
		WillReturnError(errors.New("db error"))

	w := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(w)
	req := httptest.NewRequest(http.MethodGet, "/v1/tasks", nil)
	req = req.WithContext(ctxmeta.WithAuthUser(req.Context(), ctxmeta.AuthUser{
		UserID: creatorID,
		TeamID: teamID,
	}))
	ctx.Request = req

	ctrls.ListTasks(ctx)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected status 500, got %d", w.Code)
	}
	var resp dtos.BaseResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if resp.Code != constants.ResponseCodeInternalError {
		t.Fatalf("expected INTERNAL_ERROR code, got %s", resp.Code)
	}
}

func TestControllers_UpdateTask_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	gormDB, mock := setupMockDB(t)
	repo := repositories.New(gormDB)
	svc := services.New(config.Config{}, repo, nil)
	ctrls := New(config.Config{}, svc)

	taskID := uuid.New()
	creatorID := uuid.New()
	teamID := uuid.New()
	logID := uuid.New()
	now := time.Now()
	version := 1

	// 1. Task found
	taskRows := sqlmock.NewRows([]string{"id", "title", "description", "status", "creator_id", "assignee_id", "team_id", "version", "created_at", "updated_at"}).
		AddRow(taskID, "Old Title", "Old Desc", "todo", creatorID, nil, teamID, version, now, now)
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "tasks" WHERE id = $1 AND "tasks"."deleted_at" IS NULL ORDER BY "tasks"."id" LIMIT $2`)).
		WithArgs(taskID, 1).
		WillReturnRows(taskRows)

	// 2. Update task & log
	mock.ExpectBegin()
	mock.ExpectExec(regexp.QuoteMeta(`UPDATE "tasks"`)).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectQuery(regexp.QuoteMeta(`INSERT INTO "task_logs"`)).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(logID))
	mock.ExpectCommit()

	newTitle := "Brand New Title"
	newStatus := constants.TaskStatusInProgress
	reqBody := dtos.UpdateTaskRequest{
		Version: version,
		Title:   &newTitle,
		Status:  &newStatus,
	}
	bodyBytes, _ := json.Marshal(reqBody)

	w := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(w)
	ctx.Params = gin.Params{{Key: "id", Value: taskID.String()}}
	req := httptest.NewRequest(http.MethodPut, "/v1/tasks/"+taskID.String(), bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	req = req.WithContext(ctxmeta.WithAuthUser(req.Context(), ctxmeta.AuthUser{
		UserID: creatorID,
		TeamID: teamID,
	}))
	ctx.Request = req

	ctrls.UpdateTask(ctx)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d. Body: %s", w.Code, w.Body.String())
	}
	var resp dtos.APIResponse[*dtos.TaskResponse]
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if resp.Status != constants.ResponseStatusSuccess || resp.Code != constants.ResponseCodeSuccess {
		t.Fatalf("expected success response, got: %+v", resp)
	}
	if resp.Data == nil || resp.Data.Title != newTitle || resp.Data.Status != newStatus || resp.Data.Version != 2 {
		t.Fatalf("unexpected task data: %+v", resp.Data)
	}
}

func TestControllers_UpdateTask_StaleVersionConflict(t *testing.T) {
	gin.SetMode(gin.TestMode)

	gormDB, mock := setupMockDB(t)
	repo := repositories.New(gormDB)
	svc := services.New(config.Config{}, repo, nil)
	ctrls := New(config.Config{}, svc)

	taskID := uuid.New()
	creatorID := uuid.New()
	teamID := uuid.New()
	now := time.Now()
	serverVersion := 2
	clientVersion := 1

	taskRows := sqlmock.NewRows([]string{"id", "title", "description", "status", "creator_id", "assignee_id", "team_id", "version", "created_at", "updated_at"}).
		AddRow(taskID, "Title", "Desc", "todo", creatorID, nil, teamID, serverVersion, now, now)
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "tasks" WHERE id = $1 AND "tasks"."deleted_at" IS NULL ORDER BY "tasks"."id" LIMIT $2`)).
		WithArgs(taskID, 1).
		WillReturnRows(taskRows)

	newTitle := "New Title"
	bodyBytes, _ := json.Marshal(dtos.UpdateTaskRequest{
		Version: clientVersion,
		Title:   &newTitle,
	})

	w := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(w)
	ctx.Params = gin.Params{{Key: "id", Value: taskID.String()}}
	req := httptest.NewRequest(http.MethodPut, "/v1/tasks/"+taskID.String(), bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	req = req.WithContext(ctxmeta.WithAuthUser(req.Context(), ctxmeta.AuthUser{
		UserID: creatorID,
		TeamID: teamID,
	}))
	ctx.Request = req

	ctrls.UpdateTask(ctx)

	if w.Code != http.StatusConflict {
		t.Fatalf("expected status 409 Conflict, got %d", w.Code)
	}
	var resp dtos.BaseResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if resp.Code != constants.ResponseCodeConflict || resp.Message != constants.ErrStaleVersion.Error() {
		t.Fatalf("expected conflict code and stale version message, got: %+v", resp)
	}
}

func TestControllers_UpdateTask_InvalidTaskID(t *testing.T) {
	gin.SetMode(gin.TestMode)

	ctrls := New(config.Config{}, nil)

	w := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(w)
	ctx.Params = gin.Params{{Key: "id", Value: "invalid-uuid"}}
	req := httptest.NewRequest(http.MethodPut, "/v1/tasks/invalid-uuid", bytes.NewReader([]byte(`{"version":1,"title":"New"}`)))
	req.Header.Set("Content-Type", "application/json")
	ctx.Request = req

	ctrls.UpdateTask(ctx)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", w.Code)
	}
	var resp dtos.BaseResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if resp.Code != constants.ResponseCodeBadRequest {
		t.Fatalf("expected BAD_REQUEST code, got %s", resp.Code)
	}
}

func TestControllers_UpdateTask_InvalidJSON(t *testing.T) {
	gin.SetMode(gin.TestMode)

	ctrls := New(config.Config{}, nil)
	taskID := uuid.New()

	w := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(w)
	ctx.Params = gin.Params{{Key: "id", Value: taskID.String()}}
	req := httptest.NewRequest(http.MethodPut, "/v1/tasks/"+taskID.String(), bytes.NewReader([]byte(`invalid-json`)))
	req.Header.Set("Content-Type", "application/json")
	ctx.Request = req

	ctrls.UpdateTask(ctx)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", w.Code)
	}
}

func TestControllers_UpdateTask_ValidationError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	ctrls := New(config.Config{}, nil)
	taskID := uuid.New()

	// Empty request (missing version)
	w := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(w)
	ctx.Params = gin.Params{{Key: "id", Value: taskID.String()}}
	req := httptest.NewRequest(http.MethodPut, "/v1/tasks/"+taskID.String(), bytes.NewReader([]byte(`{}`)))
	req.Header.Set("Content-Type", "application/json")
	ctx.Request = req

	ctrls.UpdateTask(ctx)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", w.Code)
	}
	var resp dtos.BaseResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if resp.Code != constants.ResponseCodeBadRequest {
		t.Fatalf("expected BAD_REQUEST code, got %s", resp.Code)
	}
}

func TestControllers_UpdateTask_NotFound(t *testing.T) {
	gin.SetMode(gin.TestMode)

	gormDB, mock := setupMockDB(t)
	repo := repositories.New(gormDB)
	svc := services.New(config.Config{}, repo, nil)
	ctrls := New(config.Config{}, svc)

	taskID := uuid.New()
	creatorID := uuid.New()
	teamID := uuid.New()
	version := 1

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "tasks" WHERE id = $1 AND "tasks"."deleted_at" IS NULL ORDER BY "tasks"."id" LIMIT $2`)).
		WithArgs(taskID, 1).
		WillReturnRows(sqlmock.NewRows([]string{"id", "title"}))

	newTitle := "New Title"
	bodyBytes, _ := json.Marshal(dtos.UpdateTaskRequest{
		Version: version,
		Title:   &newTitle,
	})

	w := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(w)
	ctx.Params = gin.Params{{Key: "id", Value: taskID.String()}}
	req := httptest.NewRequest(http.MethodPut, "/v1/tasks/"+taskID.String(), bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	req = req.WithContext(ctxmeta.WithAuthUser(req.Context(), ctxmeta.AuthUser{
		UserID: creatorID,
		TeamID: teamID,
	}))
	ctx.Request = req

	ctrls.UpdateTask(ctx)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected status 404, got %d", w.Code)
	}
	var resp dtos.BaseResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if resp.Code != constants.ResponseCodeNotFound {
		t.Fatalf("expected NOT_FOUND code, got %s", resp.Code)
	}
}

func TestControllers_UpdateTask_AssigneeNotInTeam(t *testing.T) {
	gin.SetMode(gin.TestMode)

	gormDB, mock := setupMockDB(t)
	repo := repositories.New(gormDB)
	svc := services.New(config.Config{}, repo, nil)
	ctrls := New(config.Config{}, svc)

	taskID := uuid.New()
	creatorID := uuid.New()
	teamID := uuid.New()
	foreignTeamID := uuid.New()
	foreignAssigneeID := uuid.New()
	now := time.Now()
	version := 1

	taskRows := sqlmock.NewRows([]string{"id", "title", "description", "status", "creator_id", "assignee_id", "team_id", "version", "created_at", "updated_at"}).
		AddRow(taskID, "Title", "Desc", "todo", creatorID, nil, teamID, version, now, now)
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "tasks" WHERE id = $1 AND "tasks"."deleted_at" IS NULL ORDER BY "tasks"."id" LIMIT $2`)).
		WithArgs(taskID, 1).
		WillReturnRows(taskRows)

	assigneeRows := sqlmock.NewRows([]string{"id", "name", "email", "team_id", "created_at", "updated_at"}).
		AddRow(foreignAssigneeID, "Foreign", "foreign@example.com", foreignTeamID, now, now)
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "users" WHERE id = $1 ORDER BY "users"."id" LIMIT $2`)).
		WithArgs(foreignAssigneeID, 1).
		WillReturnRows(assigneeRows)

	bodyBytes, _ := json.Marshal(dtos.UpdateTaskRequest{
		Version:    version,
		AssigneeID: &foreignAssigneeID,
	})

	w := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(w)
	ctx.Params = gin.Params{{Key: "id", Value: taskID.String()}}
	req := httptest.NewRequest(http.MethodPut, "/v1/tasks/"+taskID.String(), bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	req = req.WithContext(ctxmeta.WithAuthUser(req.Context(), ctxmeta.AuthUser{
		UserID: creatorID,
		TeamID: teamID,
	}))
	ctx.Request = req

	ctrls.UpdateTask(ctx)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400 on assignee not in team, got %d", w.Code)
	}
	var resp dtos.BaseResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if resp.Code != constants.ResponseCodeBadRequest || resp.Message != constants.ErrAssigneeNotInTeam.Error() {
		t.Fatalf("expected assignee not in team error, got: %+v", resp)
	}
}

func TestControllers_UpdateTask_InternalError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	gormDB, mock := setupMockDB(t)
	repo := repositories.New(gormDB)
	svc := services.New(config.Config{}, repo, nil)
	ctrls := New(config.Config{}, svc)

	taskID := uuid.New()
	creatorID := uuid.New()
	teamID := uuid.New()
	version := 1

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "tasks" WHERE id = $1 AND "tasks"."deleted_at" IS NULL ORDER BY "tasks"."id" LIMIT $2`)).
		WithArgs(taskID, 1).
		WillReturnError(errors.New("db connection crashed"))

	newTitle := "New Title"
	bodyBytes, _ := json.Marshal(dtos.UpdateTaskRequest{
		Version: version,
		Title:   &newTitle,
	})

	w := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(w)
	ctx.Params = gin.Params{{Key: "id", Value: taskID.String()}}
	req := httptest.NewRequest(http.MethodPut, "/v1/tasks/"+taskID.String(), bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	req = req.WithContext(ctxmeta.WithAuthUser(req.Context(), ctxmeta.AuthUser{
		UserID: creatorID,
		TeamID: teamID,
	}))
	ctx.Request = req

	ctrls.UpdateTask(ctx)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected status 500, got %d", w.Code)
	}
	var resp dtos.BaseResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if resp.Code != constants.ResponseCodeInternalError {
		t.Fatalf("expected INTERNAL_ERROR code, got %s", resp.Code)
	}
}

func TestControllers_AssignTask_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)
	gormDB, mock := setupMockDB(t)
	repo := repositories.New(gormDB)
	svc := services.New(config.Config{}, repo, nil)
	ctrls := New(config.Config{}, svc)

	taskID := uuid.New()
	creatorID := uuid.New()
	assigneeID := uuid.New()
	teamID := uuid.New()
	logID := uuid.New()
	now := time.Now()
	version := 1

	// 1. Task query
	taskRows := sqlmock.NewRows([]string{"id", "title", "description", "status", "creator_id", "assignee_id", "team_id", "version", "created_at", "updated_at"}).
		AddRow(taskID, "Task Title", "Desc", "todo", creatorID, nil, teamID, version, now, now)
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "tasks" WHERE id = $1 AND "tasks"."deleted_at" IS NULL ORDER BY "tasks"."id" LIMIT $2`)).
		WithArgs(taskID, 1).
		WillReturnRows(taskRows)

	// 2. Assignee query
	assigneeRows := sqlmock.NewRows([]string{"id", "name", "email", "team_id", "created_at", "updated_at"}).
		AddRow(assigneeID, "Assignee User", "assignee@example.com", teamID, now, now)
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "users" WHERE id = $1 ORDER BY "users"."id" LIMIT $2`)).
		WithArgs(assigneeID, 1).
		WillReturnRows(assigneeRows)

	// 3. Transaction
	mock.ExpectBegin()
	mock.ExpectExec(regexp.QuoteMeta(`UPDATE "tasks"`)).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectQuery(regexp.QuoteMeta(`INSERT INTO "task_logs"`)).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(logID))
	mock.ExpectCommit()

	bodyBytes, _ := json.Marshal(dtos.AssignTaskRequest{
		AssigneeID: assigneeID,
		Version:    version,
	})

	w := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(w)
	ctx.Params = gin.Params{{Key: "id", Value: taskID.String()}}
	req := httptest.NewRequest(http.MethodPost, "/v1/tasks/"+taskID.String()+"/assign", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	req = req.WithContext(ctxmeta.WithAuthUser(req.Context(), ctxmeta.AuthUser{
		UserID: creatorID,
		TeamID: teamID,
	}))
	ctx.Request = req

	ctrls.AssignTask(ctx)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d. Body: %s", w.Code, w.Body.String())
	}
	var resp dtos.APIResponse[*dtos.TaskResponse]
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if resp.Status != constants.ResponseStatusSuccess || resp.Data == nil || *resp.Data.AssigneeID != assigneeID || resp.Data.Version != 2 {
		t.Fatalf("unexpected response payload: %+v", resp)
	}
}

func TestControllers_AssignTask_InvalidTaskID(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctrls := New(config.Config{}, nil)

	w := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(w)
	ctx.Params = gin.Params{{Key: "id", Value: "invalid-uuid"}}
	req := httptest.NewRequest(http.MethodPost, "/v1/tasks/invalid-uuid/assign", bytes.NewReader([]byte("{}")))
	ctx.Request = req

	ctrls.AssignTask(ctx)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", w.Code)
	}
}

func TestControllers_AssignTask_InvalidJSON(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctrls := New(config.Config{}, nil)

	taskID := uuid.New()
	w := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(w)
	ctx.Params = gin.Params{{Key: "id", Value: taskID.String()}}
	req := httptest.NewRequest(http.MethodPost, "/v1/tasks/"+taskID.String()+"/assign", bytes.NewReader([]byte("invalid json")))
	ctx.Request = req

	ctrls.AssignTask(ctx)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", w.Code)
	}
}

func TestControllers_AssignTask_ValidationError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctrls := New(config.Config{}, nil)

	taskID := uuid.New()
	bodyBytes, _ := json.Marshal(dtos.AssignTaskRequest{
		AssigneeID: uuid.Nil,
	})

	w := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(w)
	ctx.Params = gin.Params{{Key: "id", Value: taskID.String()}}
	req := httptest.NewRequest(http.MethodPost, "/v1/tasks/"+taskID.String()+"/assign", bytes.NewReader(bodyBytes))
	ctx.Request = req

	ctrls.AssignTask(ctx)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", w.Code)
	}
}

func TestControllers_AssignTask_DomainErrors(t *testing.T) {
	gin.SetMode(gin.TestMode)
	gormDB, mock := setupMockDB(t)
	repo := repositories.New(gormDB)
	svc := services.New(config.Config{}, repo, nil)
	ctrls := New(config.Config{}, svc)

	taskID := uuid.New()
	creatorID := uuid.New()
	assigneeID := uuid.New()
	teamID := uuid.New()
	otherTeamID := uuid.New()
	version := 1
	staleVersion := 2
	now := time.Now()

	// 1. Assignee not in team -> 400 Bad Request
	taskRows := sqlmock.NewRows([]string{"id", "title", "description", "status", "creator_id", "assignee_id", "team_id", "version", "created_at", "updated_at"}).
		AddRow(taskID, "Task Title", "Desc", "todo", creatorID, nil, teamID, version, now, now)
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "tasks" WHERE id = $1 AND "tasks"."deleted_at" IS NULL ORDER BY "tasks"."id" LIMIT $2`)).
		WithArgs(taskID, 1).
		WillReturnRows(taskRows)

	assigneeRows := sqlmock.NewRows([]string{"id", "name", "email", "team_id", "created_at", "updated_at"}).
		AddRow(assigneeID, "Foreign Assignee", "foreign@example.com", otherTeamID, now, now)
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "users" WHERE id = $1 ORDER BY "users"."id" LIMIT $2`)).
		WithArgs(assigneeID, 1).
		WillReturnRows(assigneeRows)

	bodyBytes, _ := json.Marshal(dtos.AssignTaskRequest{
		AssigneeID: assigneeID,
		Version:    version,
	})

	w := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(w)
	ctx.Params = gin.Params{{Key: "id", Value: taskID.String()}}
	req := httptest.NewRequest(http.MethodPost, "/v1/tasks/"+taskID.String()+"/assign", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	req = req.WithContext(ctxmeta.WithAuthUser(req.Context(), ctxmeta.AuthUser{
		UserID: creatorID,
		TeamID: teamID,
	}))
	ctx.Request = req

	ctrls.AssignTask(ctx)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400 for assignee not in team, got %d", w.Code)
	}

	// 2. Stale version conflict -> 409 Conflict
	taskRows2 := sqlmock.NewRows([]string{"id", "title", "description", "status", "creator_id", "assignee_id", "team_id", "version", "created_at", "updated_at"}).
		AddRow(taskID, "Task Title", "Desc", "todo", creatorID, nil, teamID, staleVersion, now, now)
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "tasks" WHERE id = $1 AND "tasks"."deleted_at" IS NULL ORDER BY "tasks"."id" LIMIT $2`)).
		WithArgs(taskID, 1).
		WillReturnRows(taskRows2)

	bodyBytes2, _ := json.Marshal(dtos.AssignTaskRequest{
		AssigneeID: assigneeID,
		Version:    version,
	})

	w2 := httptest.NewRecorder()
	ctx2, _ := gin.CreateTestContext(w2)
	ctx2.Params = gin.Params{{Key: "id", Value: taskID.String()}}
	req2 := httptest.NewRequest(http.MethodPost, "/v1/tasks/"+taskID.String()+"/assign", bytes.NewReader(bodyBytes2))
	req2.Header.Set("Content-Type", "application/json")
	req2 = req2.WithContext(ctxmeta.WithAuthUser(req2.Context(), ctxmeta.AuthUser{
		UserID: creatorID,
		TeamID: teamID,
	}))
	ctx2.Request = req2

	ctrls.AssignTask(ctx2)

	if w2.Code != http.StatusConflict {
		t.Fatalf("expected status 409 for stale version, got %d", w2.Code)
	}

	var conflictResp dtos.BaseResponse
	if err := json.Unmarshal(w2.Body.Bytes(), &conflictResp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if conflictResp.Code != constants.ResponseCodeConflict {
		t.Fatalf("expected CONFLICT response code, got %s", conflictResp.Code)
	}
}



