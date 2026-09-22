package controllers

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"

	"task-management/internal/config"
	"task-management/internal/constants"
	"task-management/internal/dtos"
	"task-management/internal/pkg/apperror"
)

func TestControllers_wrapError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctrls := New(config.Config{}, nil)

	t.Run("nil error does nothing", func(t *testing.T) {
		w := httptest.NewRecorder()
		ctx, _ := gin.CreateTestContext(w)
		ctrls.wrapError(ctx, nil)
		if w.Code != http.StatusOK {
			t.Fatalf("expected default 200 on nil error, got %d", w.Code)
		}
	})

	t.Run("AppError renders exact HTTP status and code", func(t *testing.T) {
		w := httptest.NewRecorder()
		ctx, _ := gin.CreateTestContext(w)
		appErr := apperror.New(http.StatusConflict, "CONFLICT", "resource conflict")
		ctrls.wrapError(ctx, appErr)

		if w.Code != http.StatusConflict {
			t.Fatalf("expected 409, got %d", w.Code)
		}
		var resp dtos.BaseResponse
		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}
		if resp.Success || resp.Code != "CONFLICT" || resp.Message != "resource conflict" {
			t.Fatalf("unexpected response payload: %+v", resp)
		}
	})

	t.Run("domain constants errors from constants/error.go", func(t *testing.T) {
		cases := []struct {
			name       string
			err        error
			wantStatus int
			wantCode   string
		}{
			{"unauthorized", constants.ErrUnauthorized, http.StatusUnauthorized, constants.ResponseCodeUnauthorized},
			{"task not found", constants.ErrTaskNotFound, http.StatusNotFound, constants.ResponseCodeNotFound},
			{"invalid idempotency key", constants.ErrInvalidIdempotencyKey, http.StatusBadRequest, constants.ResponseCodeBadRequest},
			{"team not found", constants.ErrTeamNotFound, http.StatusBadRequest, constants.ResponseCodeBadRequest},
		}

		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				w := httptest.NewRecorder()
				ctx, _ := gin.CreateTestContext(w)
				ctrls.wrapError(ctx, tc.err)

				if w.Code != tc.wantStatus {
					t.Fatalf("expected status %d, got %d", tc.wantStatus, w.Code)
				}
				var resp dtos.BaseResponse
				if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
					t.Fatalf("failed to decode response: %v", err)
				}
				if resp.Code != tc.wantCode {
					t.Fatalf("expected code %s, got %s", tc.wantCode, resp.Code)
				}
			})
		}
	})

	t.Run("wrapped constant error preserves code and uses custom error message", func(t *testing.T) {
		w := httptest.NewRecorder()
		ctx, _ := gin.CreateTestContext(w)
		ctrls.wrapError(ctx, constants.ErrBadRequest.Wrap(errors.New("custom validation failure")))

		if w.Code != http.StatusBadRequest {
			t.Fatalf("expected status 400, got %d", w.Code)
		}
		var resp dtos.BaseResponse
		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}
		if resp.Code != constants.ResponseCodeBadRequest || resp.Message != "custom validation failure" {
			t.Fatalf("unexpected response: %+v", resp)
		}
	})

	t.Run("raw unclassified error falls back to ErrInternalServerError", func(t *testing.T) {
		w := httptest.NewRecorder()
		ctx, _ := gin.CreateTestContext(w)
		ctrls.wrapError(ctx, errors.New("unhandled database failure"))

		if w.Code != http.StatusInternalServerError {
			t.Fatalf("expected status 500, got %d", w.Code)
		}
		var resp dtos.BaseResponse
		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Fatalf("failed to decode: %v", err)
		}
		if resp.Code != constants.ResponseCodeInternalError || resp.Message != "internal server error" {
			t.Fatalf("expected code %s and safe message 'internal server error', got code %s message %s",
				constants.ResponseCodeInternalError, resp.Code, resp.Message)
		}
	})
}
