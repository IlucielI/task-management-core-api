package routes_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/gin-gonic/gin"
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

	// 1. Without Basic Auth -> 401 Unauthorized
	wUnauthorized := httptest.NewRecorder()
	reqUnauthorized, err := http.NewRequest(http.MethodGet, "/v1/teams?name=Engineering", nil)
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}
	router.ServeHTTP(wUnauthorized, reqUnauthorized)
	if wUnauthorized.Code != http.StatusUnauthorized {
		t.Fatalf("expected status 401 on missing basic auth, got %d", wUnauthorized.Code)
	}

	// 2. With valid Basic Auth -> 200 OK
	w := httptest.NewRecorder()
	req, err := http.NewRequest(http.MethodGet, "/v1/teams?name=Engineering", nil)
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}
	req.SetBasicAuth(cfg.BasicAuthUsername, cfg.BasicAuthPassword)

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}

	if reqID := w.Header().Get("X-Request-ID"); reqID == "" {
		t.Fatal("expected X-Request-ID header in response")
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

func TestRouter_PanicRecovery(t *testing.T) {
	cfg := config.Load()
	ctrls := controllers.New(cfg, nil)
	router := routes.NewRouter(cfg, ctrls)

	// Register an endpoint that panics
	router.GET("/v1/panic-test", func(c *gin.Context) {
		panic("database connection string contains password123")
	})

	w := httptest.NewRecorder()
	req, err := http.NewRequest(http.MethodGet, "/v1/panic-test", nil)
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}

	router.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected status 500, got %d", w.Code)
	}

	reqID := w.Header().Get("X-Request-ID")
	if reqID == "" {
		t.Error("expected X-Request-ID header in response")
	}

	var resp dtos.BaseResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to unmarshal JSON response: %v, body: %s", err, w.Body.String())
	}

	if resp.Success {
		t.Errorf("expected success false, got true")
	}
	if resp.Code != constants.ResponseCodeInternalError {
		t.Errorf("expected code %q, got %q", constants.ResponseCodeInternalError, resp.Code)
	}
	if resp.Message != "An internal server error occurred" {
		t.Errorf("expected message 'An internal server error occurred', got %q", resp.Message)
	}
	if resp.Timestamp.IsZero() {
		t.Errorf("expected non-zero timestamp")
	}

	bodyStr := w.Body.String()
	if strings.Contains(bodyStr, "password123") {
		t.Errorf("security violation: panic message leaked to response body: %s", bodyStr)
	}
}

func TestRouter_DocsEndpoints(t *testing.T) {
	cfg := config.Load()
	ctrls := controllers.New(cfg, nil)
	router := routes.NewRouter(cfg, ctrls)

	tests := []struct {
		path        string
		contentType string
		mustContain string
	}{
		{"/openapi.yaml", "application/x-yaml", "openapi: 3.0.3"},
		{"/docs", "text/html", "@scalar/api-reference"},
	}

	for _, tc := range tests {
		t.Run(tc.path, func(t *testing.T) {
			w := httptest.NewRecorder()
			req, err := http.NewRequest(http.MethodGet, tc.path, nil)
			if err != nil {
				t.Fatalf("failed to create request for %s: %v", tc.path, err)
			}

			router.ServeHTTP(w, req)

			if w.Code != http.StatusOK {
				t.Fatalf("expected status 200 for %s, got %d", tc.path, w.Code)
			}

			ct := w.Header().Get("Content-Type")
			if !strings.Contains(ct, tc.contentType) {
				t.Errorf("expected Content-Type %s, got %s", tc.contentType, ct)
			}

			if !strings.Contains(w.Body.String(), tc.mustContain) {
				t.Errorf("expected body to contain %q", tc.mustContain)
			}
		})
	}
}


