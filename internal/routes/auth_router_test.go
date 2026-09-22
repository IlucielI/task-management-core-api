package routes_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
	"gorm.io/gorm"

	"task-management/internal/config"
	"task-management/internal/constants"
	"task-management/internal/controllers"
	"task-management/internal/dtos"
	"task-management/internal/repositories"
	"task-management/internal/routes"
	"task-management/internal/services"
)

func TestRouter_Register_RouteRegistered(t *testing.T) {
	cfg := config.Load()
	gormDB, mock := setupMockDB(t)
	repo := repositories.New(gormDB)
	svc := services.New(cfg, nil, nil, nil)
	svc.SetRepositories(repo)

	ctrls := controllers.New(cfg, nil, nil, nil)
	ctrls.SetService(svc)

	router := routes.NewRouter(cfg, ctrls)

	teamID := uuid.New()
	userID := uuid.New()
	now := time.Now()

	reqPayload := dtos.RegisterRequest{
		Name:     "Bob",
		Email:    "bob@example.com",
		Password: "password123",
		TeamID:   teamID,
	}

	// 1. Team check
	teamRows := sqlmock.NewRows([]string{"id", "name", "created_at", "updated_at"}).
		AddRow(teamID, "Engineering", now, now)
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "teams" WHERE id = $1 ORDER BY "teams"."id" LIMIT $2`)).
		WithArgs(teamID, 1).
		WillReturnRows(teamRows)

	// 2. Email uniqueness check
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "users" WHERE LOWER(email) = $1 ORDER BY "users"."id" LIMIT $2`)).
		WithArgs("bob@example.com", 1).
		WillReturnError(gorm.ErrRecordNotFound)

	// 3. User creation
	mock.ExpectBegin()
	mock.ExpectQuery(regexp.QuoteMeta(`INSERT INTO "users"`)).
		WillReturnRows(sqlmock.NewRows([]string{"id", "created_at", "updated_at"}).AddRow(userID, now, now))
	mock.ExpectCommit()

	bodyBytes, _ := json.Marshal(reqPayload)
	w := httptest.NewRecorder()
	req, err := http.NewRequest(http.MethodPost, "/v1/auth/register", bytes.NewReader(bodyBytes))
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")

	router.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected status 201, got %d. Body: %s", w.Code, w.Body.String())
	}

	var resp dtos.APIResponse[*dtos.UserResponse]
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if !resp.Success || resp.Code != constants.ResponseCodeSuccess {
		t.Fatalf("expected success envelope response, got: %+v", resp)
	}
	if resp.Data == nil || resp.Data.Email != "bob@example.com" {
		t.Fatalf("unexpected data: %+v", resp.Data)
	}
}
