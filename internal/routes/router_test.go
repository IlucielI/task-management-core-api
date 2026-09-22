package routes_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"task-management/internal/config"
	"task-management/internal/controllers"
	"task-management/internal/routes"
)

func TestRouter_HealthCheck(t *testing.T) {
	cfg := config.Load()
	ctrls := controllers.New(cfg, nil)
	router := routes.NewRouter(cfg, ctrls)

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

	if resp["status"] != "ok" {
		t.Errorf("expected status 'ok', got %v", resp["status"])
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

	services, ok := resp["services"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected services map in response, got %v", resp["services"])
	}

	if services["database"] != "disconnected" {
		t.Errorf("expected database status 'disconnected' when db is nil, got %v", services["database"])
	}
}
