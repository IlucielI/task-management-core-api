package routes_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"regexp"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/alicebob/miniredis/v2"
	"github.com/google/uuid"
	goRedis "github.com/redis/go-redis/v9"

	redisAdapter "task-management/internal/adapters/redis"
	"task-management/internal/config"
	"task-management/internal/constants"
	"task-management/internal/controllers"
	"task-management/internal/dtos"
	"task-management/internal/models"
	"task-management/internal/pkg/jwt"
	"task-management/internal/repositories"
	"task-management/internal/routes"
	"task-management/internal/services"
)

func TestRouter_CreateTask_RequiresAuth(t *testing.T) {
	cfg := config.Load()
	ctrls := controllers.New(cfg, nil)
	router := routes.NewRouter(cfg, ctrls)

	req := httptest.NewRequest(http.MethodPost, "/v1/tasks", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected status 401 Unauthorized when missing token, got %d", w.Code)
	}
}

func TestRouter_CreateTask_Success(t *testing.T) {
	cfg := config.Load()
	cfg.JWTSecret = "test-secret-key-that-is-long-enough-32bytes"

	mr := miniredis.RunT(t)
	rawClient := goRedis.NewClient(&goRedis.Options{Addr: mr.Addr()})
	rdb := redisAdapter.NewWithClient(rawClient)
	gormDB, mock := setupMockDB(t)

	repo := repositories.New(gormDB, rdb)
	svc := services.New(cfg, repo, nil)
	ctrls := controllers.New(cfg, svc)
	router := routes.NewRouter(cfg, ctrls, svc)

	userID := uuid.New()
	teamID := uuid.New()
	taskID := uuid.New()
	user := &models.User{
		ID:     userID,
		Email:  "user@example.com",
		TeamID: teamID,
	}

	tokenPair, err := jwt.GenerateTokenPair(cfg, user, "")
	if err != nil {
		t.Fatalf("failed to generate token pair: %v", err)
	}

	// Mock DB transaction for task creation
	mock.ExpectBegin()
	mock.ExpectQuery(regexp.QuoteMeta(`INSERT INTO "tasks"`)).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(taskID))
	mock.ExpectQuery(regexp.QuoteMeta(`INSERT INTO "task_logs"`)).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(uuid.New()))
	mock.ExpectCommit()

	body, _ := json.Marshal(dtos.CreateTaskRequest{
		Title: "E2E Route Task",
	})
	req := httptest.NewRequest(http.MethodPost, "/v1/tasks", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+tokenPair.AccessToken)
	req.Header.Set("Idempotency-Key", uuid.New().String())
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected status 201, got %d: %s", w.Code, w.Body.String())
	}

	var resp dtos.APIResponse[*dtos.TaskResponse]
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if !resp.Success || resp.Data == nil || resp.Data.Title != "E2E Route Task" {
		t.Fatalf("unexpected response body: %+v", resp)
	}
	if resp.Data.CreatorID != userID || resp.Data.TeamID != teamID {
		t.Fatalf("mismatched user/team IDs: %+v", resp.Data)
	}
}

func TestRouter_GetTaskByID_RequiresAuth(t *testing.T) {
	cfg := config.Load()
	ctrls := controllers.New(cfg, nil)
	router := routes.NewRouter(cfg, ctrls)

	taskID := uuid.New()
	req := httptest.NewRequest(http.MethodGet, "/v1/tasks/"+taskID.String(), nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected status 401 Unauthorized when missing token, got %d", w.Code)
	}
}

func TestRouter_GetTaskByID_Success(t *testing.T) {
	cfg := config.Load()
	cfg.JWTSecret = "test-secret-key-that-is-long-enough-32bytes"

	gormDB, mock := setupMockDB(t)
	repo := repositories.New(gormDB)
	svc := services.New(cfg, repo, nil)
	ctrls := controllers.New(cfg, svc)
	router := routes.NewRouter(cfg, ctrls, svc)

	userID := uuid.New()
	teamID := uuid.New()
	taskID := uuid.New()
	user := &models.User{
		ID:     userID,
		Email:  "user@example.com",
		TeamID: teamID,
	}

	tokenPair, err := jwt.GenerateTokenPair(cfg, user, "")
	if err != nil {
		t.Fatalf("failed to generate token pair: %v", err)
	}

	rows := sqlmock.NewRows([]string{"id", "title", "description", "status", "creator_id", "team_id"}).
		AddRow(taskID, "E2E Detail Task", "Detail Description", "todo", userID, teamID)
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "tasks" WHERE id = $1 AND "tasks"."deleted_at" IS NULL ORDER BY "tasks"."id" LIMIT $2`)).
		WithArgs(taskID, 1).
		WillReturnRows(rows)

	req := httptest.NewRequest(http.MethodGet, "/v1/tasks/"+taskID.String(), nil)
	req.Header.Set("Authorization", "Bearer "+tokenPair.AccessToken)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp dtos.APIResponse[*dtos.TaskResponse]
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if !resp.Success || resp.Data == nil || resp.Data.ID != taskID || resp.Data.Title != "E2E Detail Task" {
		t.Fatalf("unexpected response body: %+v", resp)
	}
	if resp.Data.CreatorID != userID || resp.Data.TeamID != teamID {
		t.Fatalf("mismatched user/team IDs: %+v", resp.Data)
	}
}

func TestRouter_DeleteTask_RequiresAuth(t *testing.T) {
	cfg := config.Load()
	ctrls := controllers.New(cfg, nil)
	router := routes.NewRouter(cfg, ctrls)

	taskID := uuid.New()
	req := httptest.NewRequest(http.MethodDelete, "/v1/tasks/"+taskID.String(), nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected status 401 Unauthorized when missing token, got %d", w.Code)
	}
}

func TestRouter_DeleteTask_Success(t *testing.T) {
	cfg := config.Load()
	cfg.JWTSecret = "test-secret-key-that-is-long-enough-32bytes"

	gormDB, mock := setupMockDB(t)
	repo := repositories.New(gormDB)
	svc := services.New(cfg, repo, nil)
	ctrls := controllers.New(cfg, svc)
	router := routes.NewRouter(cfg, ctrls, svc)

	userID := uuid.New()
	teamID := uuid.New()
	taskID := uuid.New()
	user := &models.User{
		ID:     userID,
		Email:  "user@example.com",
		TeamID: teamID,
	}

	tokenPair, err := jwt.GenerateTokenPair(cfg, user, "")
	if err != nil {
		t.Fatalf("failed to generate token pair: %v", err)
	}

	// 1. FindTaskByID
	rows := sqlmock.NewRows([]string{"id", "title", "description", "status", "creator_id", "team_id"}).
		AddRow(taskID, "Task to Delete", "Desc", "todo", userID, teamID)
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

	req := httptest.NewRequest(http.MethodDelete, "/v1/tasks/"+taskID.String(), nil)
	req.Header.Set("Authorization", "Bearer "+tokenPair.AccessToken)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp dtos.BaseResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if !resp.Success || resp.Code != "OK" || resp.Message != "Task deleted successfully" {
		t.Fatalf("unexpected response body: %+v", resp)
	}
}

func TestRouter_ListTasks_RequiresAuth(t *testing.T) {
	cfg := config.Load()
	ctrls := controllers.New(cfg, nil)
	router := routes.NewRouter(cfg, ctrls)

	req := httptest.NewRequest(http.MethodGet, "/v1/tasks", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected status 401 Unauthorized when missing token, got %d", w.Code)
	}
}

func TestRouter_ListTasks_Success(t *testing.T) {
	cfg := config.Load()
	cfg.JWTSecret = "test-secret-key-that-is-long-enough-32bytes"

	gormDB, mock := setupMockDB(t)
	repo := repositories.New(gormDB)
	svc := services.New(cfg, repo, nil)
	ctrls := controllers.New(cfg, svc)
	router := routes.NewRouter(cfg, ctrls, svc)

	userID := uuid.New()
	teamID := uuid.New()
	taskID := uuid.New()
	user := &models.User{
		ID:     userID,
		Email:  "user@example.com",
		TeamID: teamID,
	}

	tokenPair, err := jwt.GenerateTokenPair(cfg, user, "")
	if err != nil {
		t.Fatalf("failed to generate token pair: %v", err)
	}

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT count(*) FROM "tasks" WHERE team_id = $1 AND status = $2 AND "tasks"."deleted_at" IS NULL`)).
		WithArgs(teamID, "todo").
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))

	rows := sqlmock.NewRows([]string{"id", "title", "description", "status", "creator_id", "team_id"}).
		AddRow(taskID, "E2E List Task", "Desc", "todo", userID, teamID)
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "tasks" WHERE team_id = $1 AND status = $2 AND "tasks"."deleted_at" IS NULL ORDER BY created_at DESC LIMIT $3 OFFSET $4`)).
		WithArgs(teamID, "todo", 10, 10).
		WillReturnRows(rows)

	req := httptest.NewRequest(http.MethodGet, "/v1/tasks?page=2&limit=10&status=todo", nil)
	req.Header.Set("Authorization", "Bearer "+tokenPair.AccessToken)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp dtos.APIResponse[dtos.ListTasksData]
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if !resp.Success || len(resp.Data.Items) != 1 || resp.Data.Items[0].Title != "E2E List Task" {
		t.Fatalf("unexpected items: %+v", resp.Data)
	}
	if resp.Data.Metadata.Count != 1 || resp.Data.Metadata.Limit != 10 || resp.Data.Metadata.Page != 2 || resp.Data.Metadata.TotalPages != 1 {
		t.Fatalf("unexpected metadata: %+v", resp.Data.Metadata)
	}
}

func TestRouter_UpdateTask_RequiresAuth(t *testing.T) {
	cfg := config.Load()
	ctrls := controllers.New(cfg, nil)
	router := routes.NewRouter(cfg, ctrls)

	taskID := uuid.New()
	req := httptest.NewRequest(http.MethodPut, "/v1/tasks/"+taskID.String(), bytes.NewReader([]byte(`{"title":"New"}`)))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected status 401 Unauthorized when missing token, got %d", w.Code)
	}
}

func TestRouter_UpdateTask_Success(t *testing.T) {
	cfg := config.Load()
	cfg.JWTSecret = "test-secret-key-that-is-long-enough-32bytes"

	gormDB, mock := setupMockDB(t)
	repo := repositories.New(gormDB)
	svc := services.New(cfg, repo, nil)
	ctrls := controllers.New(cfg, svc)
	router := routes.NewRouter(cfg, ctrls, svc)

	userID := uuid.New()
	teamID := uuid.New()
	taskID := uuid.New()
	logID := uuid.New()

	user := &models.User{
		ID:     userID,
		Email:  "user@example.com",
		TeamID: teamID,
	}
	tokenPair, err := jwt.GenerateTokenPair(cfg, user, "test-session-id")
	if err != nil {
		t.Fatalf("failed to generate token pair: %v", err)
	}

	version := 1
	taskRows := sqlmock.NewRows([]string{"id", "title", "description", "status", "creator_id", "team_id", "version"}).
		AddRow(taskID, "Old Task", "Desc", "todo", userID, teamID, version)
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "tasks" WHERE id = $1 AND "tasks"."deleted_at" IS NULL ORDER BY "tasks"."id" LIMIT $2`)).
		WithArgs(taskID, 1).
		WillReturnRows(taskRows)

	mock.ExpectBegin()
	mock.ExpectExec(regexp.QuoteMeta(`UPDATE "tasks"`)).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectQuery(regexp.QuoteMeta(`INSERT INTO "task_logs"`)).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(logID))
	mock.ExpectCommit()

	newTitle := "Updated Task Title"
	newStatus := constants.TaskStatusInProgress
	reqBody := dtos.UpdateTaskRequest{
		Version: &version,
		Title:   &newTitle,
		Status:  &newStatus,
	}
	bodyBytes, _ := json.Marshal(reqBody)

	req := httptest.NewRequest(http.MethodPut, "/v1/tasks/"+taskID.String(), bytes.NewReader(bodyBytes))
	req.Header.Set("Authorization", "Bearer "+tokenPair.AccessToken)
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp dtos.APIResponse[*dtos.TaskResponse]
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if !resp.Success || resp.Data == nil || resp.Data.Title != newTitle || resp.Data.Status != newStatus || resp.Data.Version != 2 {
		t.Fatalf("unexpected response data: %+v", resp.Data)
	}
}


