package middlewares

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"task-management/internal/constants"
	"task-management/internal/pkg/ctxmeta"
)

type mockAuthValidator struct {
	authenticateFn func(ctx context.Context, tokenStr string) (*ctxmeta.AuthUser, error)
}

func (m *mockAuthValidator) Authenticate(ctx context.Context, tokenStr string) (*ctxmeta.AuthUser, error) {
	if m.authenticateFn != nil {
		return m.authenticateFn(ctx, tokenStr)
	}
	return nil, constants.ErrInvalidToken
}

func TestAuthMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)

	userID := uuid.New()
	teamID := uuid.New()
	validUser := &ctxmeta.AuthUser{
		UserID:    userID,
		Email:     "user@example.com",
		TeamID:    teamID,
		SessionID: "sess-123",
	}

	// 1. Missing Authorization header
	{
		validator := &mockAuthValidator{}
		w := httptest.NewRecorder()
		c, r := gin.CreateTestContext(w)
		r.Use(Auth(validator))
		r.GET("/protected", func(ctx *gin.Context) {
			ctx.Status(http.StatusOK)
		})
		c.Request = httptest.NewRequest(http.MethodGet, "/protected", nil)
		r.ServeHTTP(w, c.Request)

		if w.Code != http.StatusUnauthorized {
			t.Fatalf("expected 401 on missing auth header, got %d", w.Code)
		}
	}

	// 2. Invalid Authorization header format
	{
		validator := &mockAuthValidator{}
		w := httptest.NewRecorder()
		c, r := gin.CreateTestContext(w)
		r.Use(Auth(validator))
		r.GET("/protected", func(ctx *gin.Context) {
			ctx.Status(http.StatusOK)
		})
		c.Request = httptest.NewRequest(http.MethodGet, "/protected", nil)
		c.Request.Header.Set("Authorization", "Basic 123456")
		r.ServeHTTP(w, c.Request)

		if w.Code != http.StatusUnauthorized {
			t.Fatalf("expected 401 on invalid format, got %d", w.Code)
		}
	}

	// 3. Nil validator or empty token
	{
		w := httptest.NewRecorder()
		c, r := gin.CreateTestContext(w)
		r.Use(Auth(nil))
		r.GET("/protected", func(ctx *gin.Context) {
			ctx.Status(http.StatusOK)
		})
		c.Request = httptest.NewRequest(http.MethodGet, "/protected", nil)
		c.Request.Header.Set("Authorization", "Bearer valid-token")
		r.ServeHTTP(w, c.Request)

		if w.Code != http.StatusUnauthorized {
			t.Fatalf("expected 401 on nil validator, got %d", w.Code)
		}
	}

	// 4. Validator returns error
	{
		validator := &mockAuthValidator{
			authenticateFn: func(ctx context.Context, tokenStr string) (*ctxmeta.AuthUser, error) {
				return nil, errors.New("invalid or expired token")
			},
		}
		w := httptest.NewRecorder()
		c, r := gin.CreateTestContext(w)
		r.Use(Auth(validator))
		r.GET("/protected", func(ctx *gin.Context) {
			ctx.Status(http.StatusOK)
		})
		c.Request = httptest.NewRequest(http.MethodGet, "/protected", nil)
		c.Request.Header.Set("Authorization", "Bearer bad-token")
		r.ServeHTTP(w, c.Request)

		if w.Code != http.StatusUnauthorized {
			t.Fatalf("expected 401 on validation error, got %d", w.Code)
		}
	}

	// 5. Success authentication
	{
		validator := &mockAuthValidator{
			authenticateFn: func(ctx context.Context, tokenStr string) (*ctxmeta.AuthUser, error) {
				if tokenStr == "good-token" {
					return validUser, nil
				}
				return nil, constants.ErrInvalidToken
			},
		}
		var capturedUser ctxmeta.AuthUser
		w := httptest.NewRecorder()
		c, r := gin.CreateTestContext(w)
		r.Use(Auth(validator))
		r.GET("/protected", func(ctx *gin.Context) {
			u, _ := ctxmeta.GetAuthUser(ctx.Request.Context())
			capturedUser = u
			ctx.Status(http.StatusOK)
		})
		c.Request = httptest.NewRequest(http.MethodGet, "/protected", nil)
		c.Request.Header.Set("Authorization", "Bearer good-token")
		r.ServeHTTP(w, c.Request)

		if w.Code != http.StatusOK {
			t.Fatalf("expected 200 on valid token, got %d", w.Code)
		}
		if capturedUser.UserID != userID || capturedUser.Email != "user@example.com" {
			t.Fatalf("unexpected captured user: %+v", capturedUser)
		}
	}
}
