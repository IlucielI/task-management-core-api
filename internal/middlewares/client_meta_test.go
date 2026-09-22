package middlewares

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"

	"task-management/internal/pkg/ctxmeta"
)

func TestClientMetaMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	router.Use(ClientMeta())

	var capturedIP, capturedUA string

	router.GET("/test", func(c *gin.Context) {
		capturedIP = ctxmeta.GetClientIP(c.Request.Context())
		capturedUA = ctxmeta.GetUserAgent(c.Request.Context())
		c.Status(http.StatusOK)
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("User-Agent", "CustomBrowser/1.0")

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}
	if capturedUA != "CustomBrowser/1.0" {
		t.Errorf("expected User-Agent CustomBrowser/1.0, got %s", capturedUA)
	}
	if capturedIP == "" {
		t.Error("expected non-empty client IP")
	}
}
