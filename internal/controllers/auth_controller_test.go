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
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"

	"task-management/internal/config"
	"task-management/internal/constants"
	"task-management/internal/dtos"
	"task-management/internal/repositories"
	"task-management/internal/services"
)

func TestControllers_Register_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	gormDB, mock := setupMockDB(t)
	repo := repositories.New(gormDB)
	svc := services.New(config.Config{}, nil, nil, nil)
	svc.SetRepositories(repo)

	ctrls := New(config.Config{}, nil, nil, nil)
	ctrls.SetService(svc)

	teamID := uuid.New()
	userID := uuid.New()
	now := time.Now()

	reqPayload := dtos.RegisterRequest{
		Name:     "Alice",
		Email:    "alice@example.com",
		Password: "password123",
		TeamID:   teamID,
	}

	// 1. Team found
	teamRows := sqlmock.NewRows([]string{"id", "name", "created_at", "updated_at"}).
		AddRow(teamID, "Engineering", now, now)
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "teams" WHERE id = $1 ORDER BY "teams"."id" LIMIT $2`)).
		WithArgs(teamID, 1).
		WillReturnRows(teamRows)

	// 2. Email unique
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "users" WHERE LOWER(email) = $1 ORDER BY "users"."id" LIMIT $2`)).
		WithArgs("alice@example.com", 1).
		WillReturnError(gorm.ErrRecordNotFound)

	// 3. Create user
	mock.ExpectBegin()
	mock.ExpectQuery(regexp.QuoteMeta(`INSERT INTO "users"`)).
		WillReturnRows(sqlmock.NewRows([]string{"id", "created_at", "updated_at"}).AddRow(userID, now, now))
	mock.ExpectCommit()

	bodyBytes, _ := json.Marshal(reqPayload)
	w := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(w)
	ctx.Request = httptest.NewRequest(http.MethodPost, "/v1/auth/register", bytes.NewReader(bodyBytes))
	ctx.Request.Header.Set("Content-Type", "application/json")

	ctrls.Register(ctx)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected status %d, got %d. Body: %s", http.StatusCreated, w.Code, w.Body.String())
	}

	var resp dtos.APIResponse[*dtos.UserResponse]
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if !resp.Success || resp.Code != constants.ResponseCodeSuccess {
		t.Fatalf("expected success response, got: %+v", resp)
	}
	if resp.Data == nil || resp.Data.Email != "alice@example.com" {
		t.Fatalf("unexpected user response data: %+v", resp.Data)
	}
}

func TestControllers_Register_InvalidJSON(t *testing.T) {
	gin.SetMode(gin.TestMode)

	ctrls := New(config.Config{}, nil, nil, nil)

	w := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(w)
	ctx.Request = httptest.NewRequest(http.MethodPost, "/v1/auth/register", bytes.NewReader([]byte("invalid json")))
	ctx.Request.Header.Set("Content-Type", "application/json")

	ctrls.Register(ctx)

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

func TestControllers_Register_ValidationError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	ctrls := New(config.Config{}, nil, nil, nil)

	invalidReq := dtos.RegisterRequest{
		Name:     "",
		Email:    "invalid-email",
		Password: "short",
		TeamID:   uuid.Nil,
	}

	bodyBytes, _ := json.Marshal(invalidReq)
	w := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(w)
	ctx.Request = httptest.NewRequest(http.MethodPost, "/v1/auth/register", bytes.NewReader(bodyBytes))
	ctx.Request.Header.Set("Content-Type", "application/json")

	ctrls.Register(ctx)

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

func TestControllers_Register_DomainErrors(t *testing.T) {
	gin.SetMode(gin.TestMode)

	gormDB, mock := setupMockDB(t)
	repo := repositories.New(gormDB)
	svc := services.New(config.Config{}, nil, nil, nil)
	svc.SetRepositories(repo)

	ctrls := New(config.Config{}, nil, nil, nil)
	ctrls.SetService(svc)

	teamID := uuid.New()
	reqPayload := dtos.RegisterRequest{
		Name:     "Bob",
		Email:    "bob@example.com",
		Password: "password123",
		TeamID:   teamID,
	}

	// 1. Team not found -> 400 Bad Request
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "teams" WHERE id = $1 ORDER BY "teams"."id" LIMIT $2`)).
		WithArgs(teamID, 1).
		WillReturnError(gorm.ErrRecordNotFound)

	bodyBytes, _ := json.Marshal(reqPayload)
	w := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(w)
	ctx.Request = httptest.NewRequest(http.MethodPost, "/v1/auth/register", bytes.NewReader(bodyBytes))
	ctx.Request.Header.Set("Content-Type", "application/json")

	ctrls.Register(ctx)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d on team not found, got %d", http.StatusBadRequest, w.Code)
	}

	var resp dtos.BaseResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if resp.Success || resp.Code != constants.ResponseCodeBadRequest || resp.Message != constants.ErrTeamNotFound.Error() {
		t.Fatalf("expected team not found message, got: %+v", resp)
	}

	// 2. Email already exists -> 400 Bad Request
	now := time.Now()
	teamRows := sqlmock.NewRows([]string{"id", "name", "created_at", "updated_at"}).
		AddRow(teamID, "Product", now, now)
	userRows := sqlmock.NewRows([]string{"id", "name", "email", "password", "team_id", "created_at", "updated_at"}).
		AddRow(uuid.New(), "Existing Bob", "bob@example.com", "hash", teamID, now, now)

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "teams" WHERE id = $1 ORDER BY "teams"."id" LIMIT $2`)).
		WithArgs(teamID, 1).
		WillReturnRows(teamRows)
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "users" WHERE LOWER(email) = $1 ORDER BY "users"."id" LIMIT $2`)).
		WithArgs("bob@example.com", 1).
		WillReturnRows(userRows)

	w2 := httptest.NewRecorder()
	ctx2, _ := gin.CreateTestContext(w2)
	ctx2.Request = httptest.NewRequest(http.MethodPost, "/v1/auth/register", bytes.NewReader(bodyBytes))
	ctx2.Request.Header.Set("Content-Type", "application/json")

	ctrls.Register(ctx2)

	if w2.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d on email already exists, got %d", http.StatusBadRequest, w2.Code)
	}

	var resp2 dtos.BaseResponse
	if err := json.Unmarshal(w2.Body.Bytes(), &resp2); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if resp2.Success || resp2.Code != constants.ResponseCodeBadRequest || resp2.Message != constants.ErrEmailAlreadyExists.Error() {
		t.Fatalf("expected email already registered message, got: %+v", resp2)
	}

	// 3. Unexpected internal error -> 500
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "teams" WHERE id = $1 ORDER BY "teams"."id" LIMIT $2`)).
		WithArgs(teamID, 1).
		WillReturnError(errors.New("fatal connection crash"))

	w3 := httptest.NewRecorder()
	ctx3, _ := gin.CreateTestContext(w3)
	ctx3.Request = httptest.NewRequest(http.MethodPost, "/v1/auth/register", bytes.NewReader(bodyBytes))
	ctx3.Request.Header.Set("Content-Type", "application/json")

	ctrls.Register(ctx3)

	if w3.Code != http.StatusInternalServerError {
		t.Fatalf("expected status %d on internal server error, got %d", http.StatusInternalServerError, w3.Code)
	}
}
