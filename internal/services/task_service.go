package services

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"

	"task-management/internal/constants"
	"task-management/internal/dtos"
	"task-management/internal/models"
	"task-management/internal/pkg/ctxmeta"
)

// CreateTask handles task creation with strict 24h idempotency and concurrency locking.
func (s *Service) CreateTask(ctx context.Context, req dtos.CreateTaskRequest, idempotencyKey string) (*dtos.TaskResponse, error) {
	// 1. Validate Idempotency-Key format (must be valid non-nil UUID)
	parsedKey, err := uuid.Parse(strings.TrimSpace(idempotencyKey))
	if err != nil || parsedKey == uuid.Nil {
		return nil, constants.ErrInvalidIdempotencyKey
	}
	cleanKey := parsedKey.String()

	// 2. Check Redis idempotency cache for existing response (sequential duplicate)
	cached, err := s.repo.GetIdempotencyResponse(ctx, cleanKey)
	if err != nil {
		return nil, fmt.Errorf("failed to check idempotency cache: %w", err)
	}
	if cached != nil && cached.Data != nil {
		return cached.Data, nil
	}

	// 3. Acquire atomic idempotency lock (30-second TTL to prevent deadlocks)
	acquired, err := s.repo.AcquireIdempotencyLock(ctx, cleanKey, 30*time.Second)
	if err != nil {
		return nil, fmt.Errorf("failed to acquire idempotency lock: %w", err)
	}

	// If another goroutine/node already holds the lock, wait for it to finish and cache response
	if !acquired {
		ticker := time.NewTicker(25 * time.Millisecond)
		defer ticker.Stop()
		timeout := time.After(5 * time.Second)

		for {
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-timeout:
				return nil, errors.New("timeout waiting for concurrent task creation")
			case <-ticker.C:
				cached, err := s.repo.GetIdempotencyResponse(ctx, cleanKey)
				if err == nil && cached != nil && cached.Data != nil {
					return cached.Data, nil
				}
			}
		}
	}

	// Lock is acquired by this goroutine: ensure lock is released on return
	defer s.repo.ReleaseIdempotencyLock(ctx, cleanKey)

	// Double-check cache in case previous holder finished just before lock acquisition
	cached, err = s.repo.GetIdempotencyResponse(ctx, cleanKey)
	if err == nil && cached != nil && cached.Data != nil {
		return cached.Data, nil
	}

	// 4. Extract authenticated user from context
	authUser, ok := ctxmeta.GetAuthUser(ctx)
	if !ok || authUser.UserID == uuid.Nil {
		return nil, constants.ErrUnauthorized
	}

	// 5. Multi-tenancy check: If AssigneeID is specified, assignee must exist and belong to the same team
	if req.AssigneeID != nil {
		assignee, err := s.repo.FindUserByID(ctx, *req.AssigneeID)
		if err != nil {
			return nil, fmt.Errorf("failed to check assignee: %w", err)
		}
		if assignee == nil || assignee.TeamID != authUser.TeamID {
			return nil, constants.ErrAssigneeNotInTeam
		}
	}

	// 6. Default status to 'todo' if empty
	status := strings.TrimSpace(req.Status)
	if status == "" {
		status = constants.TaskStatusTodo
	}

	now := time.Now()
	taskID := uuid.New()

	task := &models.Task{
		ID:          taskID,
		Title:       strings.TrimSpace(req.Title),
		Description: strings.TrimSpace(req.Description),
		Status:      status,
		CreatorID:   authUser.UserID,
		AssigneeID:  req.AssigneeID,
		TeamID:      authUser.TeamID,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	notes := "Task created"
	log := &models.TaskLog{
		ID:           uuid.New(),
		TaskID:       taskID,
		Action:       constants.TaskActionCreate,
		ActorID:      &authUser.UserID,
		ToAssigneeID: req.AssigneeID,
		ToStatus:     &status,
		Notes:        &notes,
		Metadata:     models.JSONMap{"source": "api"},
		CreatedAt:    now,
	}

	// 7. Persist Task and TaskLog inside a single DB transaction
	if err := s.repo.CreateTaskWithLog(ctx, task, log); err != nil {
		return nil, fmt.Errorf("failed to create task: %w", err)
	}

	taskResp := &dtos.TaskResponse{
		ID:          task.ID,
		Title:       task.Title,
		Description: task.Description,
		Status:      task.Status,
		CreatorID:   task.CreatorID,
		AssigneeID:  task.AssigneeID,
		TeamID:      task.TeamID,
		CreatedAt:   task.CreatedAt,
		UpdatedAt:   task.UpdatedAt,
	}

	// 8. Cache response in Redis for 24 hours
	cachedResp := &dtos.CachedIdempotentResponse{
		StatusCode: http.StatusCreated,
		Data:       taskResp,
		Message:    "Task created successfully",
		Timestamp:  now,
	}
	if err := s.repo.SaveIdempotencyResponse(ctx, cleanKey, cachedResp, 24*time.Hour); err != nil {
		return nil, fmt.Errorf("failed to save idempotency cache: %w", err)
	}

	return taskResp, nil
}

