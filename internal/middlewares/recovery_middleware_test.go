package middlewares_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	"task-management/internal/constants"
	"task-management/internal/dtos"
	"task-management/internal/middlewares"
	"task-management/internal/pkg/ctxmeta"
)

func init() {
	gin.SetMode(gin.TestMode)
}

func TestRecovery_NormalExecution(t *testing.T) {
	buf := &bytes.Buffer{}
	r := gin.New()
	r.Use(middlewares.Recovery(middlewares.WithRecoveryWriter(buf)))

	r.GET("/ok", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	w := httptest.NewRecorder()
	req, err := http.NewRequest(http.MethodGet, "/ok", nil)
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}

	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}
	if buf.Len() > 0 {
		t.Errorf("expected no recovery log on normal execution, got: %s", buf.String())
	}
}

func TestRecovery_CatchesPanicAndReturnsUnifiedEnvelope(t *testing.T) {
	buf := &bytes.Buffer{}
	r := gin.New()
	r.Use(middlewares.Recovery(middlewares.WithRecoveryWriter(buf)))

	secretError := "database connection unexpectedly dropped: secret_token_xyz"
	r.GET("/panic", func(c *gin.Context) {
		panic(secretError)
	})

	w := httptest.NewRecorder()
	req, err := http.NewRequest(http.MethodGet, "/panic", nil)
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}

	r.ServeHTTP(w, req)

	// 1. Must return HTTP 500
	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected status 500, got %d", w.Code)
	}

	// 2. Must return standard JSON error envelope
	var resp dtos.BaseResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse JSON response: %v, body: %s", err, w.Body.String())
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
	if time.Since(resp.Timestamp) > 10*time.Second {
		t.Errorf("expected recent timestamp, got %v", resp.Timestamp)
	}

	// 3. Must NOT leak internal panic details to client
	bodyStr := w.Body.String()
	if strings.Contains(bodyStr, "secret_token_xyz") {
		t.Errorf("security violation: response body leaks secret details: %s", bodyStr)
	}
	if strings.Contains(bodyStr, "panic.go") || strings.Contains(bodyStr, "goroutine") {
		t.Errorf("security violation: response body leaks stack trace: %s", bodyStr)
	}

	// 4. Must log stack trace and request details server-side
	logStr := buf.String()
	if !strings.Contains(logStr, "[PANIC RECOVERED]") {
		t.Errorf("expected recovery log to contain '[PANIC RECOVERED]', got: %s", logStr)
	}
	if !strings.Contains(logStr, "secret_token_xyz") {
		t.Errorf("expected server-side log to contain error details, got: %s", logStr)
	}
	if !strings.Contains(logStr, "method=GET path=/panic") {
		t.Errorf("expected server-side log to contain method and path, got: %s", logStr)
	}
}

func TestRecovery_WithRequestIDInContext(t *testing.T) {
	buf := &bytes.Buffer{}
	r := gin.New()
	r.Use(middlewares.Recovery(middlewares.WithRecoveryWriter(buf)))

	r.GET("/panic-with-id", func(c *gin.Context) {
		panic("nil pointer dereference")
	})

	w := httptest.NewRecorder()
	req, err := http.NewRequest(http.MethodGet, "/panic-with-id", nil)
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}

	testReqID := "custom-req-id-12345"
	ctx := ctxmeta.WithRequestID(req.Context(), testReqID)
	req = req.WithContext(ctx)

	r.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected status 500, got %d", w.Code)
	}

	logStr := buf.String()
	if !strings.Contains(logStr, "request_id="+testReqID) {
		t.Errorf("expected server log to include request_id=%s, got: %s", testReqID, logStr)
	}
}

func TestRecovery_WithRequestIDInHeaderOrContextSet(t *testing.T) {
	buf := &bytes.Buffer{}
	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set("request_id", "gin-context-req-id")
		c.Next()
	})
	r.Use(middlewares.Recovery(middlewares.WithRecoveryWriter(buf)))

	r.GET("/panic-header", func(c *gin.Context) {
		panic("boom")
	})

	w := httptest.NewRecorder()
	req, err := http.NewRequest(http.MethodGet, "/panic-header", nil)
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}
	req.Header.Set("X-Request-ID", "header-req-id")

	r.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected status 500, got %d", w.Code)
	}

	logStr := buf.String()
	if !strings.Contains(logStr, "request_id=gin-context-req-id") {
		t.Errorf("expected server log to have request_id from gin context, got: %s", logStr)
	}
}
