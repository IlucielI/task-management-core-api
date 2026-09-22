package routes_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"task-management/internal/config"
	"task-management/internal/routes"
)

func TestRouter_HealthCheck(t *testing.T) {
	cfg := config.Load()
	router := routes.NewRouter(cfg)

	w := httptest.NewRecorder()
	req, err := http.NewRequest(http.MethodGet, "/v1/health", nil)
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if resp["version"] != cfg.Version {
		t.Errorf("expected version %q, got %v", cfg.Version, resp["version"])
	}
	if resp["git_hash"] != cfg.GitHash {
		t.Errorf("expected git_hash %q, got %v", cfg.GitHash, resp["git_hash"])
	}
	if resp["uptime"] == nil || resp["uptime"] == "" {
		t.Error("expected non-empty uptime in response")
	}
}
