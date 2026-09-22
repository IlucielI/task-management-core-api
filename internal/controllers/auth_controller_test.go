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
	"github.com/go-redis/redismock/v9"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"

	"task-management/internal/adapters/redis"
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
	svc := services.New(config.Config{}, repo, nil)
	ctrls := New(config.Config{}, svc)

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

	ctrls := New(config.Config{}, nil)

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

	ctrls := New(config.Config{}, nil)

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
	svc := services.New(config.Config{}, repo, nil)
	ctrls := New(config.Config{}, svc)

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

func TestControllers_Login_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	gormDB, mock := setupMockDB(t)
	rClient, rMock := redismock.NewClientMock()
	rdb := redis.NewWithClient(rClient)
	repo := repositories.New(gormDB, rdb)
	cfg := config.Config{
		AppName:              "task-management-test",
		JWTSecret:            "test-jwt-secret-key",
		JWTAccessExpiration:  24 * time.Hour,
		JWTRefreshExpiration: 7 * 24 * time.Hour,
	}
	svc := services.New(cfg, repo, nil)
	ctrls := New(cfg, svc)

	userID := uuid.New()
	teamID := uuid.New()
	now := time.Now()

	rawPassword := "testPassword123"
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(rawPassword), bcrypt.DefaultCost)
	if err != nil {
		t.Fatalf("failed to hash password: %v", err)
	}

	userRows := sqlmock.NewRows([]string{"id", "name", "email", "password", "team_id", "created_at", "updated_at"}).
		AddRow(userID, "Jane Doe", "jane@example.com", string(hashedPassword), teamID, now, now)

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "users" WHERE LOWER(email) = $1 ORDER BY "users"."id" LIMIT $2`)).
		WithArgs("jane@example.com", 1).
		WillReturnRows(userRows)

	rMock.Regexp().ExpectSet(`^auth:session:.*`, `.*`, 7*24*time.Hour).SetVal("OK")

	reqPayload := dtos.LoginRequest{
		Email:    "jane@example.com",
		Password: rawPassword,
	}

	bodyBytes, _ := json.Marshal(reqPayload)
	w := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(w)
	ctx.Request = httptest.NewRequest(http.MethodPost, "/v1/auth/login", bytes.NewReader(bodyBytes))
	ctx.Request.Header.Set("Content-Type", "application/json")
	ctx.Request.Header.Set("User-Agent", "TestAgent")

	ctrls.Login(ctx)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d. Body: %s", http.StatusOK, w.Code, w.Body.String())
	}

	var resp dtos.APIResponse[*dtos.LoginResponse]
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if !resp.Success || resp.Code != constants.ResponseCodeSuccess {
		t.Fatalf("expected success response, got: %+v", resp)
	}
	if resp.Data == nil || resp.Data.Tokens.AccessToken == "" || resp.Data.Tokens.RefreshToken == "" {
		t.Fatalf("expected access and refresh tokens, got: %+v", resp.Data)
	}
	if resp.Data.User.Email != "jane@example.com" {
		t.Fatalf("expected email jane@example.com, got %s", resp.Data.User.Email)
	}
}

func TestControllers_Login_InvalidJSON(t *testing.T) {
	gin.SetMode(gin.TestMode)

	ctrls := New(config.Config{}, nil)

	w := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(w)
	ctx.Request = httptest.NewRequest(http.MethodPost, "/v1/auth/login", bytes.NewReader([]byte("not valid json")))
	ctx.Request.Header.Set("Content-Type", "application/json")

	ctrls.Login(ctx)

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

func TestControllers_Login_ValidationError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	ctrls := New(config.Config{}, nil)

	reqPayload := dtos.LoginRequest{
		Email:    "invalid-email-format",
		Password: "short",
	}

	bodyBytes, _ := json.Marshal(reqPayload)
	w := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(w)
	ctx.Request = httptest.NewRequest(http.MethodPost, "/v1/auth/login", bytes.NewReader(bodyBytes))
	ctx.Request.Header.Set("Content-Type", "application/json")

	ctrls.Login(ctx)

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

func TestControllers_Login_InvalidCredentials(t *testing.T) {
	gin.SetMode(gin.TestMode)

	gormDB, mock := setupMockDB(t)
	repo := repositories.New(gormDB)
	cfg := config.Config{
		JWTSecret: "test-jwt-secret-key",
	}
	svc := services.New(cfg, repo, nil)
	ctrls := New(cfg, svc)

	// User not found in DB
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "users" WHERE LOWER(email) = $1 ORDER BY "users"."id" LIMIT $2`)).
		WithArgs("notfound@example.com", 1).
		WillReturnError(gorm.ErrRecordNotFound)

	reqPayload := dtos.LoginRequest{
		Email:    "notfound@example.com",
		Password: "testPassword123",
	}

	bodyBytes, _ := json.Marshal(reqPayload)
	w := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(w)
	ctx.Request = httptest.NewRequest(http.MethodPost, "/v1/auth/login", bytes.NewReader(bodyBytes))
	ctx.Request.Header.Set("Content-Type", "application/json")

	ctrls.Login(ctx)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected status %d on invalid credentials, got %d", http.StatusUnauthorized, w.Code)
	}

	var resp dtos.BaseResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if resp.Success || resp.Code != constants.ResponseCodeUnauthorized || resp.Message != constants.ErrInvalidCredentials.Error() {
		t.Fatalf("expected unauthorized response with ErrInvalidCredentials, got: %+v", resp)
	}
}

func TestControllers_Login_InternalError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	gormDB, mock := setupMockDB(t)
	repo := repositories.New(gormDB)
	cfg := config.Config{}
	svc := services.New(cfg, repo, nil)
	ctrls := New(cfg, svc)

	// Database failure
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "users" WHERE LOWER(email) = $1 ORDER BY "users"."id" LIMIT $2`)).
		WithArgs("jane@example.com", 1).
		WillReturnError(errors.New("db crash"))

	reqPayload := dtos.LoginRequest{
		Email:    "jane@example.com",
		Password: "testPassword123",
	}

	bodyBytes, _ := json.Marshal(reqPayload)
	w := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(w)
	ctx.Request = httptest.NewRequest(http.MethodPost, "/v1/auth/login", bytes.NewReader(bodyBytes))
	ctx.Request.Header.Set("Content-Type", "application/json")

	ctrls.Login(ctx)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected status %d on db error, got %d", http.StatusInternalServerError, w.Code)
	}
}

