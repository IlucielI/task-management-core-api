package routes_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"task-management/internal/config"
	"task-management/internal/constants"
	"task-management/internal/controllers"
	"task-management/internal/dtos"
	"task-management/internal/repositories"
	"task-management/internal/routes"
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

func TestRouter_HealthCheck(t *testing.T) {
	cfg := config.Load()
	ctrls := controllers.New(cfg, nil)
	router := routes.NewRouter(cfg, ctrls)

	w := httptest.NewRecorder()
	req, err := http.NewRequest(http.MethodGet, "/v1/health", nil)
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}

	var resp dtos.APIResponse[dtos.HealthData]
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if !resp.Success {
		t.Errorf("expected success true, got %v", resp.Success)
	}
	if resp.Code != constants.ResponseCodeSuccess {
		t.Errorf("expected code %q, got %q", constants.ResponseCodeSuccess, resp.Code)
	}
	if resp.Message != constants.ResponseMessageSuccess {
		t.Errorf("expected message %q, got %q", constants.ResponseMessageSuccess, resp.Message)
	}
	if resp.Data.Version != cfg.Version {
		t.Errorf("expected version %q, got %q", cfg.Version, resp.Data.Version)
	}
	if resp.Data.GitHash != cfg.GitHash {
		t.Errorf("expected git_hash %q, got %q", cfg.GitHash, resp.Data.GitHash)
	}
	if resp.Data.Uptime == "" {
		t.Error("expected non-empty uptime in response")
	}
	if resp.Data.Services.Database != constants.IntegrationStatusDisconnected {
		t.Errorf("expected database status %q when db is nil, got %q", constants.IntegrationStatusDisconnected, resp.Data.Services.Database)
	}
	if resp.Data.Services.Redis != constants.IntegrationStatusDisconnected {
		t.Errorf("expected redis status %q when rdb is nil, got %q", constants.IntegrationStatusDisconnected, resp.Data.Services.Redis)
	}
	if resp.Data.Services.S3 != constants.IntegrationStatusDisconnected {
		t.Errorf("expected s3 status %q when storage is nil, got %q", constants.IntegrationStatusDisconnected, resp.Data.Services.S3)
	}
}

func TestRouter_GetTeams_RouteRegistered(t *testing.T) {
	cfg := config.Load()
	gormDB, mock := setupMockDB(t)
	repo := repositories.New(gormDB)
	svc := services.New(cfg, repo, nil)
	ctrls := controllers.New(cfg, svc)

	router := routes.NewRouter(cfg, ctrls)

	teamID := uuid.New()
	now := time.Now()

	rows := sqlmock.NewRows([]string{"id", "name", "created_at", "updated_at"}).
		AddRow(teamID, "Engineering", now, now)

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT count(*) FROM "teams" WHERE LOWER(name) LIKE LOWER($1)`)).
		WithArgs("%Engineering%").
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "teams" WHERE LOWER(name) LIKE LOWER($1) ORDER BY name ASC LIMIT $2`)).
		WithArgs("%Engineering%", 10).
		WillReturnRows(rows)

	w := httptest.NewRecorder()
	req, err := http.NewRequest(http.MethodGet, "/v1/teams?name=Engineering", nil)
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}

	var resp dtos.APIResponse[dtos.ListTeamsData]
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if !resp.Success || resp.Code != constants.ResponseCodeSuccess {
		t.Fatalf("expected success envelope response, got: %+v", resp)
	}
	if len(resp.Data.Items) != 1 || resp.Data.Items[0].Name != "Engineering" {
		t.Fatalf("unexpected items: %+v", resp.Data.Items)
	}
	if resp.Data.Metadata.Count != 1 || resp.Data.Metadata.Page != 1 {
		t.Fatalf("unexpected metadata: %+v", resp.Data.Metadata)
	}
}
