package controllers

import (
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
		c.wrapError(ctx, constants.ErrInvalidIdempotencyKey)
		return
	}

	var req dtos.CreateTaskRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		c.wrapError(ctx, constants.ErrBadRequest.Wrap(err))
		return
	}

	if err := validations.ValidateCreateTaskRequest(req); err != nil {
		c.wrapError(ctx, constants.ErrBadRequest.Wrap(err))
		return
	}

	task, err := c.svc.CreateTask(ctx.Request.Context(), req, idempotencyKey)
	if err != nil {
		c.wrapError(ctx, err)
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
		c.wrapError(ctx, constants.ErrBadRequest.Wrap(err))
		return
	}

	task, err := c.svc.GetTaskByID(ctx.Request.Context(), taskID)
	if err != nil {
		c.wrapError(ctx, err)
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

// DeleteTask handles soft-deleting a single task by its UUID.
func (c *Controllers) DeleteTask(ctx *gin.Context) {
	taskID, err := validations.ValidateTaskID(ctx.Param("id"))
	if err != nil {
		c.wrapError(ctx, constants.ErrBadRequest.Wrap(err))
		return
	}

	if err := c.svc.DeleteTask(ctx.Request.Context(), taskID); err != nil {
		c.wrapError(ctx, err)
		return
	}

	ctx.JSON(http.StatusOK, dtos.BaseResponse{
		Success:   true,
		Code:      constants.ResponseCodeSuccess,
		Message:   "Task deleted successfully",
		Timestamp: time.Now(),
	})
}


