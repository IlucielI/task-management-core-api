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
	svc := services.New(config.Config{}, repo, nil)
	ctrls := New(config.Config{}, svc)

	teamID := uuid.New()
	now := time.Now()

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT count(*) FROM "teams" WHERE LOWER(name) LIKE LOWER($1)`)).
		WithArgs("%Engineering%").
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))

	rows := sqlmock.NewRows([]string{"id", "name", "created_at", "updated_at"}).
		AddRow(teamID, "Engineering", now, now)

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "teams" WHERE LOWER(name) LIKE LOWER($1) ORDER BY name ASC LIMIT $2`)).
		WithArgs("%Engineering%", 10).
		WillReturnRows(rows)

	w := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(w)
	ctx.Request = httptest.NewRequest(http.MethodGet, "/v1/teams?name=Engineering&page=1&limit=10", nil)

	ctrls.GetTeams(ctx)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	var resp dtos.APIResponse[dtos.ListTeamsData]
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if resp.Status != constants.ResponseStatusSuccess || resp.Code != constants.ResponseCodeSuccess {
		t.Fatalf("expected success response, got: %+v", resp)
	}
	if len(resp.Data.Items) != 1 || resp.Data.Items[0].Name != "Engineering" {
		t.Fatalf("unexpected items: %+v", resp.Data.Items)
	}
	if resp.Data.Metadata.Count != 1 || resp.Data.Metadata.Page != 1 || resp.Data.Metadata.Limit != 10 {
		t.Fatalf("unexpected metadata: %+v", resp.Data.Metadata)
	}
}

func TestControllers_GetTeams_RepositoryError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	gormDB, mock := setupMockDB(t)
	repo := repositories.New(gormDB)
	svc := services.New(config.Config{}, repo, nil)
	ctrls := New(config.Config{}, svc)

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT count(*) FROM "teams" WHERE LOWER(name) LIKE LOWER($1)`)).
		WithArgs("%Engineering%").
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
	if resp.Status != constants.ResponseStatusError || resp.Code != constants.ResponseCodeInternalError {
		t.Fatalf("expected internal error response, got: %+v", resp)
	}
}

func TestControllers_GetTeams_ValidationError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	ctrls := New(config.Config{}, nil)

	t.Run("name_too_long", func(t *testing.T) {
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
		if resp.Status != constants.ResponseStatusFail || resp.Code != constants.ResponseCodeBadRequest {
			t.Fatalf("expected bad request response, got: %+v", resp)
		}
	})

	t.Run("invalid_order_by", func(t *testing.T) {
		w := httptest.NewRecorder()
		ctx, _ := gin.CreateTestContext(w)
		ctx.Request = httptest.NewRequest(http.MethodGet, "/v1/teams?order_by=invalid_sort", nil)

		ctrls.GetTeams(ctx)

		if w.Code != http.StatusBadRequest {
			t.Fatalf("expected status %d, got %d", http.StatusBadRequest, w.Code)
		}
	})
}
