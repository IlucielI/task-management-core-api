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
	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"task-management/internal/config"
	"task-management/internal/constants"
	"task-management/internal/dtos"
	"task-management/internal/repositories"
	"task-management/internal/services"
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

func TestControllers_GetTeams_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	gormDB, mock := setupMockDB(t)
	repo := repositories.New(gormDB)
	svc := services.New(config.Config{}, nil, nil, nil)
	svc.SetRepositories(repo)

	ctrls := New(config.Config{}, nil, nil, nil)
	ctrls.SetService(svc)

	teamID := uuid.New()
	now := time.Now()

	rows := sqlmock.NewRows([]string{"id", "name", "created_at", "updated_at"}).
		AddRow(teamID, "Engineering", now, now)

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "teams" WHERE LOWER(name) LIKE $1 ORDER BY name ASC`)).
		WithArgs("%engineering%").
		WillReturnRows(rows)

	w := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(w)
	ctx.Request = httptest.NewRequest(http.MethodGet, "/v1/teams?name=Engineering", nil)

	ctrls.GetTeams(ctx)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	var resp dtos.APIResponse[[]dtos.TeamResponse]
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if !resp.Success || resp.Code != constants.ResponseCodeSuccess {
		t.Fatalf("expected success response, got: %+v", resp)
	}
	if len(resp.Data) != 1 || resp.Data[0].Name != "Engineering" {
		t.Fatalf("unexpected data: %+v", resp.Data)
	}
}

func TestControllers_GetTeams_RepositoryError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	gormDB, mock := setupMockDB(t)
	repo := repositories.New(gormDB)
	svc := services.New(config.Config{}, nil, nil, nil)
	svc.SetRepositories(repo)

	ctrls := New(config.Config{}, nil, nil, nil)
	ctrls.SetService(svc)

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "teams" WHERE LOWER(name) LIKE $1 ORDER BY name ASC`)).
		WithArgs("%engineering%").
		WillReturnError(errors.New("db connection failure"))

	w := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(w)
	ctx.Request = httptest.NewRequest(http.MethodGet, "/v1/teams?name=Engineering", nil)

	ctrls.GetTeams(ctx)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected status %d, got %d", http.StatusInternalServerError, w.Code)
	}

	var resp dtos.BaseResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if resp.Success || resp.Code != constants.ResponseCodeInternalError {
		t.Fatalf("expected internal error response, got: %+v", resp)
	}
}

func TestControllers_GetTeams_ValidationError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	ctrls := New(config.Config{}, nil, nil, nil)

	// Generate string longer than 100 characters
	longName := ""
	for i := 0; i < 110; i++ {
		longName += "a"
	}

	w := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(w)
	ctx.Request = httptest.NewRequest(http.MethodGet, "/v1/teams?name="+longName, nil)

	ctrls.GetTeams(ctx)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, w.Code)
	}

	var resp dtos.BaseResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if resp.Success || resp.Code != constants.ResponseCodeBadRequest {
		t.Fatalf("expected bad request response, got: %+v", resp)
	}
}
