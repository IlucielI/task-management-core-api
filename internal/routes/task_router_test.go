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

