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
		Status:    constants.ResponseStatusSuccess,
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
		Status:    constants.ResponseStatusSuccess,
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
		Status:    constants.ResponseStatusSuccess,
		Code:      constants.ResponseCodeSuccess,
		Message:   "Task deleted successfully",
		Timestamp: time.Now(),
	})
}

// ListTasks handles fetching a paginated list of tasks matching filter criteria.
func (c *Controllers) ListTasks(ctx *gin.Context) {
	var query dtos.ListTasksQuery
	if err := ctx.ShouldBindQuery(&query); err != nil {
		c.wrapError(ctx, constants.ErrBadRequest.Wrap(err))
		return
	}

	if err := validations.ValidateListTasksQuery(&query); err != nil {
		c.wrapError(ctx, constants.ErrBadRequest.Wrap(err))
		return
	}

	result, err := c.svc.ListTasks(ctx.Request.Context(), query)
	if err != nil {
		c.wrapError(ctx, err)
		return
	}

	ctx.JSON(http.StatusOK, dtos.APIResponse[*dtos.ListTasksData]{
		Status:    constants.ResponseStatusSuccess,
		Code:      constants.ResponseCodeSuccess,
		Message:   "Tasks retrieved successfully",
		Data:      result,
		Timestamp: time.Now(),
	})
}

// UpdateTask handles updating an existing task by its UUID.
func (c *Controllers) UpdateTask(ctx *gin.Context) {
	taskID, err := validations.ValidateTaskID(ctx.Param("id"))
	if err != nil {
		c.wrapError(ctx, constants.ErrBadRequest.Wrap(err))
		return
	}

	var req dtos.UpdateTaskRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		c.wrapError(ctx, constants.ErrBadRequest.Wrap(err))
		return
	}

	if err := validations.ValidateUpdateTaskRequest(req); err != nil {
		c.wrapError(ctx, constants.ErrBadRequest.Wrap(err))
		return
	}

	task, err := c.svc.UpdateTask(ctx.Request.Context(), taskID, req)
	if err != nil {
		c.wrapError(ctx, err)
		return
	}

	ctx.JSON(http.StatusOK, dtos.APIResponse[*dtos.TaskResponse]{
		Status:    constants.ResponseStatusSuccess,
		Code:      constants.ResponseCodeSuccess,
		Message:   "Task updated successfully",
		Data:      task,
		Timestamp: time.Now(),
	})
}

// AssignTask handles assigning a task to another user within the team.
func (c *Controllers) AssignTask(ctx *gin.Context) {
	taskID, err := validations.ValidateTaskID(ctx.Param("id"))
	if err != nil {
		c.wrapError(ctx, constants.ErrBadRequest.Wrap(err))
		return
	}

	var req dtos.AssignTaskRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		c.wrapError(ctx, constants.ErrBadRequest.Wrap(err))
		return
	}

	if err := validations.ValidateAssignTaskRequest(req); err != nil {
		c.wrapError(ctx, constants.ErrBadRequest.Wrap(err))
		return
	}

	task, err := c.svc.AssignTask(ctx.Request.Context(), taskID, req)
	if err != nil {
		c.wrapError(ctx, err)
		return
	}

	ctx.JSON(http.StatusOK, dtos.APIResponse[*dtos.TaskResponse]{
		Status:    constants.ResponseStatusSuccess,
		Code:      constants.ResponseCodeSuccess,
		Message:   "Task assigned successfully",
		Data:      task,
		Timestamp: time.Now(),
	})
}


