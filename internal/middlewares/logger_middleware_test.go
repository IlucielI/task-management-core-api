package middlewares

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"task-management/internal/pkg/ctxmeta"
)

func init() {
	gin.SetMode(gin.TestMode)
}

func TestStructuredLogger(t *testing.T) {
	t.Run("normal_200_info_log_and_generated_request_id", func(t *testing.T) {
		buf := &bytes.Buffer{}
		router := gin.New()
		router.Use(StructuredLogger(WithWriter(buf)))

		var capturedContextReqID string
		router.GET("/test-ok", func(c *gin.Context) {
			capturedContextReqID = ctxmeta.GetRequestID(c.Request.Context())
			c.String(http.StatusOK, "hello")
		})

		req := httptest.NewRequest(http.MethodGet, "/test-ok", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", w.Code)
		}

		resHeaderID := w.Header().Get("X-Request-ID")
		if resHeaderID == "" {
			t.Fatal("expected X-Request-ID in response header")
		}
		if _, err := uuid.Parse(resHeaderID); err != nil {
			t.Fatalf("expected valid UUID for X-Request-ID, got %s", resHeaderID)
		}
		if capturedContextReqID != resHeaderID {
			t.Fatalf("expected context request_id %s to match header %s", capturedContextReqID, resHeaderID)
		}

		var entry LogEntry
		if err := json.Unmarshal(buf.Bytes(), &entry); err != nil {
			t.Fatalf("failed to parse log entry as JSON: %v. Raw: %s", err, buf.String())
		}

		if entry.Level != "INFO" {
			t.Errorf("expected level INFO, got %s", entry.Level)
		}
		if entry.Method != http.MethodGet {
			t.Errorf("expected method GET, got %s", entry.Method)
		}
		if entry.Path != "/test-ok" {
			t.Errorf("expected path /test-ok, got %s", entry.Path)
		}
		if entry.StatusCode != http.StatusOK {
			t.Errorf("expected status_code 200, got %d", entry.StatusCode)
		}
		if entry.RequestID != resHeaderID {
			t.Errorf("expected request_id %s, got %s", resHeaderID, entry.RequestID)
		}
		if entry.Latency == "" {
			t.Error("expected non-empty latency string")
		}
	})

	t.Run("retains_valid_x_request_id_header", func(t *testing.T) {
		buf := &bytes.Buffer{}
		router := gin.New()
		router.Use(StructuredLogger(WithWriter(buf)))

		router.GET("/custom-id", func(c *gin.Context) {
			c.Status(http.StatusNoContent)
		})

		customID := uuid.New().String()
		req := httptest.NewRequest(http.MethodGet, "/custom-id", nil)
		req.Header.Set("X-Request-ID", customID)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Header().Get("X-Request-ID") != customID {
			t.Fatalf("expected echoed X-Request-ID %s, got %s", customID, w.Header().Get("X-Request-ID"))
		}

		var entry LogEntry
		if err := json.Unmarshal(buf.Bytes(), &entry); err != nil {
			t.Fatalf("failed to parse JSON log: %v", err)
		}
		if entry.RequestID != customID {
			t.Fatalf("expected log request_id %s, got %s", customID, entry.RequestID)
		}
	})

	t.Run("regenerates_on_invalid_x_request_id", func(t *testing.T) {
		buf := &bytes.Buffer{}
		router := gin.New()
		router.Use(StructuredLogger(WithWriter(buf)))

		router.GET("/invalid-id", func(c *gin.Context) {
			c.Status(http.StatusOK)
		})

		req := httptest.NewRequest(http.MethodGet, "/invalid-id", nil)
		req.Header.Set("X-Request-ID", "not-a-valid-uuid")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		newID := w.Header().Get("X-Request-ID")
		if newID == "not-a-valid-uuid" || newID == "" {
			t.Fatalf("expected newly generated valid UUID, got %s", newID)
		}
		if _, err := uuid.Parse(newID); err != nil {
			t.Fatalf("expected valid UUID, got %s", newID)
		}
	})

	t.Run("client_error_warn_level", func(t *testing.T) {
		buf := &bytes.Buffer{}
		router := gin.New()
		router.Use(StructuredLogger(WithWriter(buf)))

		router.GET("/client-err", func(c *gin.Context) {
			c.Status(http.StatusBadRequest)
		})

		req := httptest.NewRequest(http.MethodGet, "/client-err", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		var entry LogEntry
		if err := json.Unmarshal(buf.Bytes(), &entry); err != nil {
			t.Fatalf("failed to parse log: %v", err)
		}
		if entry.Level != "WARN" {
			t.Fatalf("expected level WARN for 400, got %s", entry.Level)
		}
	})

	t.Run("server_error_error_level", func(t *testing.T) {
		buf := &bytes.Buffer{}
		router := gin.New()
		router.Use(StructuredLogger(WithWriter(buf)))

		router.GET("/server-err", func(c *gin.Context) {
			c.Status(http.StatusInternalServerError)
		})

		req := httptest.NewRequest(http.MethodGet, "/server-err", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		var entry LogEntry
		if err := json.Unmarshal(buf.Bytes(), &entry); err != nil {
			t.Fatalf("failed to parse log: %v", err)
		}
		if entry.Level != "ERROR" {
			t.Fatalf("expected level ERROR for 500, got %s", entry.Level)
		}
	})
}
