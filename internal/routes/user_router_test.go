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

func TestRouter_ListUsers_RequiresAuth(t *testing.T) {
	cfg := config.Config{}
	router := routes.NewRouter(cfg, nil, nil)

	req := httptest.NewRequest(http.MethodGet, "/v1/users", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected status 401 for unauthenticated request, got %d", w.Code)
	}
}

func TestRouter_ListUsers_Success(t *testing.T) {
	cfg := config.Load()
	cfg.JWTSecret = "test-secret-key-that-is-at-least-32-bytes-long"
	gormDB, mock := setupMockDB(t)
	repo := repositories.New(gormDB)
	svc := services.New(cfg, repo, nil)
	ctrls := controllers.New(cfg, svc)
	router := routes.NewRouter(cfg, ctrls, svc)

	userID := uuid.New()
	teamID := uuid.New()
	user := &models.User{
		ID:     userID,
		Name:   "Caller User",
		Email:  "caller@example.com",
		TeamID: teamID,
	}

	tokenPair, err := jwt.GenerateTokenPair(cfg, user, "")
	if err != nil {
		t.Fatalf("failed to generate token pair: %v", err)
	}

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT count(*) FROM "users" WHERE team_id = $1`)).
		WithArgs(teamID).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))

	rows := sqlmock.NewRows([]string{"id", "name", "email", "team_id", "created_at", "updated_at"}).
		AddRow(userID, "Caller User", "caller@example.com", teamID, time.Now(), time.Now())
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "users" WHERE team_id = $1 ORDER BY name ASC LIMIT $2`)).
		WithArgs(teamID, 10).
		WillReturnRows(rows)

	req := httptest.NewRequest(http.MethodGet, "/v1/users?page=1&limit=10", nil)
	req.Header.Set("Authorization", "Bearer "+tokenPair.AccessToken)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp dtos.APIResponse[dtos.ListUsersData]
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if resp.Status != constants.ResponseStatusSuccess || len(resp.Data.Items) != 1 || resp.Data.Items[0].Name != "Caller User" {
		t.Fatalf("unexpected items: %+v", resp.Data)
	}
	if resp.Data.Metadata.Count != 1 || resp.Data.Metadata.Limit != 10 || resp.Data.Metadata.Page != 1 || resp.Data.Metadata.TotalPages != 1 {
		t.Fatalf("unexpected metadata: %+v", resp.Data.Metadata)
	}
}
