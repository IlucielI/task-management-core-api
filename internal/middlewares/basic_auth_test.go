package middlewares

import (
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"

	"task-management/internal/constants"
	"task-management/internal/dtos"
)

func TestBasicAuthMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)

	expectedUser := "test-client"
	expectedPass := "secret-key-123"

	newTestRouter := func() *gin.Engine {
		r := gin.New()
		r.Use(BasicAuth(expectedUser, expectedPass))
		r.GET("/secure", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"status": "ok"})
		})
		return r
	}

	// 1. Missing Authorization header -> 401
	t.Run("missing authorization header", func(t *testing.T) {
		r := newTestRouter()
		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/secure", nil)

		r.ServeHTTP(w, req)

		if w.Code != http.StatusUnauthorized {
			t.Fatalf("expected status 401, got %d", w.Code)
		}

		var resp dtos.BaseResponse
		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}
		if resp.Success || resp.Code != constants.ResponseCodeUnauthorized {
			t.Fatalf("expected unauthorized response envelope, got: %+v", resp)
		}
	})

	// 2. Malformed scheme (not Basic) -> 401
	t.Run("invalid scheme prefix", func(t *testing.T) {
		r := newTestRouter()
		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/secure", nil)
		req.Header.Set("Authorization", "Bearer some-token")

		r.ServeHTTP(w, req)

		if w.Code != http.StatusUnauthorized {
			t.Fatalf("expected status 401, got %d", w.Code)
		}
	})

	// 3. Invalid base64 encoding -> 401
	t.Run("invalid base64 encoding", func(t *testing.T) {
		r := newTestRouter()
		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/secure", nil)
		req.Header.Set("Authorization", "Basic !!!invalid-base64!!!")

		r.ServeHTTP(w, req)

		if w.Code != http.StatusUnauthorized {
			t.Fatalf("expected status 401, got %d", w.Code)
		}
	})

	// 4. Missing colon delimiter -> 401
	t.Run("invalid credential format without colon", func(t *testing.T) {
		r := newTestRouter()
		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/secure", nil)
		encoded := base64.StdEncoding.EncodeToString([]byte("useronlywithoutcolon"))
		req.Header.Set("Authorization", "Basic "+encoded)

		r.ServeHTTP(w, req)

		if w.Code != http.StatusUnauthorized {
			t.Fatalf("expected status 401, got %d", w.Code)
		}
	})

	// 5. Wrong username or password -> 401
	t.Run("wrong credentials", func(t *testing.T) {
		r := newTestRouter()
		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/secure", nil)
		req.SetBasicAuth(expectedUser, "wrong-password")

		r.ServeHTTP(w, req)

		if w.Code != http.StatusUnauthorized {
			t.Fatalf("expected status 401, got %d", w.Code)
		}
	})

	// 6. Correct credentials -> 200
	t.Run("valid credentials", func(t *testing.T) {
		r := newTestRouter()
		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/secure", nil)
		req.SetBasicAuth(expectedUser, expectedPass)

		r.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected status 200, got %d, body: %s", w.Code, w.Body.String())
		}
	})
}
