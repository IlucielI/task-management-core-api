package controllers

import (
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"task-management/internal/constants"
	"task-management/internal/dtos"
	"task-management/internal/validations"
)

// CreateTask handles task creation with Idempotency-Key and multi-tenancy validation.
func (c *Controllers) CreateTask(ctx *gin.Context) {
	idempotencyKey := ctx.GetHeader("Idempotency-Key")
	if strings.TrimSpace(idempotencyKey) == "" {
		ctx.JSON(http.StatusBadRequest, dtos.BaseResponse{
			Success:   false,
			Code:      constants.ResponseCodeBadRequest,
			Message:   constants.ErrInvalidIdempotencyKey.Error(),
			Timestamp: time.Now(),
		})
		return
	}

	var req dtos.CreateTaskRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, dtos.BaseResponse{
			Success:   false,
			Code:      constants.ResponseCodeBadRequest,
			Message:   err.Error(),
			Timestamp: time.Now(),
		})
		return
	}

	if err := validations.ValidateCreateTaskRequest(req); err != nil {
		ctx.JSON(http.StatusBadRequest, dtos.BaseResponse{
			Success:   false,
			Code:      constants.ResponseCodeBadRequest,
			Message:   err.Error(),
			Timestamp: time.Now(),
		})
		return
	}

	task, err := c.svc.CreateTask(ctx.Request.Context(), req, idempotencyKey)
	if err != nil {
		now := time.Now()
		if errors.Is(err, constants.ErrInvalidIdempotencyKey) || errors.Is(err, constants.ErrAssigneeNotInTeam) {
			ctx.JSON(http.StatusBadRequest, dtos.BaseResponse{
				Success:   false,
				Code:      constants.ResponseCodeBadRequest,
				Message:   err.Error(),
				Timestamp: now,
			})
			return
		}
		if errors.Is(err, constants.ErrUnauthorized) {
			ctx.JSON(http.StatusUnauthorized, dtos.BaseResponse{
				Success:   false,
				Code:      constants.ResponseCodeUnauthorized,
				Message:   err.Error(),
				Timestamp: now,
			})
			return
		}

		ctx.JSON(http.StatusInternalServerError, dtos.BaseResponse{
			Success:   false,
			Code:      constants.ResponseCodeInternalError,
			Message:   "Failed to create task",
			Timestamp: now,
		})
		return
	}

	ctx.JSON(http.StatusCreated, dtos.APIResponse[*dtos.TaskResponse]{
		Success:   true,
		Code:      constants.ResponseCodeSuccess,
		Message:   "Task created successfully",
		Data:      task,
		Timestamp: time.Now(),
	})
}

// GetTaskByID handles fetching a single task by its UUID.
func (c *Controllers) GetTaskByID(ctx *gin.Context) {
	taskID, err := validations.ValidateTaskID(ctx.Param("id"))
	if err != nil {
		ctx.JSON(http.StatusBadRequest, dtos.BaseResponse{
			Success:   false,
			Code:      constants.ResponseCodeBadRequest,
			Message:   err.Error(),
			Timestamp: time.Now(),
		})
		return
	}

	task, err := c.svc.GetTaskByID(ctx.Request.Context(), taskID)
	if err != nil {
		now := time.Now()
		if errors.Is(err, constants.ErrTaskNotFound) {
			ctx.JSON(http.StatusNotFound, dtos.BaseResponse{
				Success:   false,
				Code:      constants.ResponseCodeNotFound,
				Message:   err.Error(),
				Timestamp: now,
			})
			return
		}
		if errors.Is(err, constants.ErrUnauthorized) {
			ctx.JSON(http.StatusUnauthorized, dtos.BaseResponse{
				Success:   false,
				Code:      constants.ResponseCodeUnauthorized,
				Message:   err.Error(),
				Timestamp: now,
			})
			return
		}

		ctx.JSON(http.StatusInternalServerError, dtos.BaseResponse{
			Success:   false,
			Code:      constants.ResponseCodeInternalError,
			Message:   "Failed to retrieve task",
			Timestamp: now,
		})
		return
	}

	ctx.JSON(http.StatusOK, dtos.APIResponse[*dtos.TaskResponse]{
		Success:   true,
		Code:      constants.ResponseCodeSuccess,
		Message:   "Task retrieved successfully",
		Data:      task,
		Timestamp: time.Now(),
	})
}

