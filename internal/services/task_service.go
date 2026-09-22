package services

import (
	"context"
	"errors"
	"fmt"
	stdlog "log"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"

	"task-management/internal/constants"
	"task-management/internal/dtos"
	"task-management/internal/models"
	"task-management/internal/repositories"
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
	authUser, err := s.getAuthUser(ctx)
	if err != nil {
		return nil, s.wrapError(ctx, err)
	}

	// 5. Multi-tenancy check: If AssigneeID is specified, assignee must exist and belong to the same team
	if req.AssigneeID != nil {
		assignee, err := s.repo.FindUserByID(ctx, *req.AssigneeID)
		if err != nil {
			return nil, s.wrapError(ctx, fmt.Errorf("failed to check assignee: %w", err))
		}
		if assignee == nil || assignee.TeamID != authUser.TeamID {
			return nil, constants.ErrAssigneeNotInTeam
		}
	}

	// 6. Default status to 'todo' if empty
	status := constants.TaskStatus(strings.TrimSpace(string(req.Status)))
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
		Version:     1,
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
		return nil, s.wrapError(ctx, fmt.Errorf("failed to create task: %w", err))
	}

	taskResp := composeTaskResponse(task)

	// 8. Cache response in Redis for 24 hours
	cachedResp := &dtos.CachedIdempotentResponse{
		StatusCode: http.StatusCreated,
		Data:       taskResp,
		Message:    "Task created successfully",
		Timestamp:  now,
	}
	if err := s.repo.SaveIdempotencyResponse(ctx, cleanKey, cachedResp, 24*time.Hour); err != nil {
		return nil, s.wrapError(ctx, fmt.Errorf("failed to save idempotency cache: %w", err))
	}

	return taskResp, nil
}

// GetTaskByID retrieves a task by ID with multi-tenant team boundary verification.
func (s *Service) GetTaskByID(ctx context.Context, taskID uuid.UUID) (*dtos.TaskResponse, error) {
	authUser, err := s.getAuthUser(ctx)
	if err != nil {
		return nil, s.wrapError(ctx, err)
	}

	task, err := s.repo.FindTaskDetailByID(ctx, taskID)
	if err != nil {
		return nil, s.wrapError(ctx, fmt.Errorf("failed to find task: %w", err))
	}
	if task == nil || task.TeamID != authUser.TeamID {
		return nil, constants.ErrTaskNotFound
	}

	return composeTaskResponse(task), nil
}

// DeleteTask soft-deletes a task and records a DELETE audit log within the caller's team boundary.
func (s *Service) DeleteTask(ctx context.Context, taskID uuid.UUID) error {
	authUser, err := s.getAuthUser(ctx)
	if err != nil {
		return s.wrapError(ctx, err)
	}

	task, err := s.repo.FindTaskByID(ctx, taskID)
	if err != nil {
		return s.wrapError(ctx, fmt.Errorf("failed to find task: %w", err))
	}
	if task == nil || task.TeamID != authUser.TeamID {
		return constants.ErrTaskNotFound
	}

	now := time.Now()
	notes := "Task deleted"
	log := &models.TaskLog{
		ID:             uuid.New(),
		TaskID:         task.ID,
		Action:         constants.TaskActionDelete,
		ActorID:        &authUser.UserID,
		FromAssigneeID: task.AssigneeID,
		FromStatus:     &task.Status,
		Notes:          &notes,
		Metadata:       models.JSONMap{"source": "api"},
		CreatedAt:      now,
	}

	if err := s.repo.DeleteTaskWithLog(ctx, task.ID, log); err != nil {
		return s.wrapError(ctx, fmt.Errorf("failed to delete task: %w", err))
	}

	return nil
}

// ListTasks retrieves a paginated list of tasks matching filter criteria within the caller's team boundary.
func (s *Service) ListTasks(ctx context.Context, query dtos.ListTasksQuery) (*dtos.ListTasksData, error) {
	authUser, err := s.getAuthUser(ctx)
	if err != nil {
		return nil, s.wrapError(ctx, err)
	}

	// Calculate offset from Page & Limit
	offset := (query.Page - 1) * query.Limit

	// Multi-tenant boundary isolation: ensure caller cannot access another team's tasks
	if query.TeamID != nil && *query.TeamID != authUser.TeamID {
		return nil, constants.ErrTaskNotFound
	}

	targetTeamID := authUser.TeamID

	filter := repositories.TaskFilter{
		TeamID:     &targetTeamID,
		CreatorID:  query.CreatorID,
		AssigneeID: query.AssigneeID,
		Status:     constants.TaskStatus(strings.TrimSpace(string(query.Status))),
		Title:      strings.TrimSpace(query.Title),
		OrderBy:    query.OrderBy,
		Offset:     offset,
		Limit:      query.Limit,
	}

	tasks, total, err := s.repo.FindTasks(ctx, filter)
	if err != nil {
		return nil, s.wrapError(ctx, fmt.Errorf("failed to list tasks: %w", err))
	}

	totalPages := 0
	if total > 0 && query.Limit > 0 {
		totalPages = int((total + int64(query.Limit) - 1) / int64(query.Limit))
	}

	items := make([]*dtos.TaskResponse, len(tasks))
	for i := range tasks {
		items[i] = composeTaskResponse(&tasks[i])
	}

	return &dtos.ListTasksData{
		Items: items,
		Metadata: dtos.ListMetadata{
			Count:      total,
			Limit:      int64(query.Limit),
			Page:       query.Page,
			TotalPages: totalPages,
		},
	}, nil
}

// UpdateTask updates an existing task, enforces multi-tenant team boundaries, and records an audit log in a single transaction.
func (s *Service) UpdateTask(ctx context.Context, taskID uuid.UUID, req dtos.UpdateTaskRequest) (*dtos.TaskResponse, error) {
	authUser, err := s.getAuthUser(ctx)
	if err != nil {
		return nil, s.wrapError(ctx, err)
	}

	task, err := s.repo.FindTaskByID(ctx, taskID)
	if err != nil {
		return nil, s.wrapError(ctx, fmt.Errorf("failed to find task: %w", err))
	}
	if task == nil || task.TeamID != authUser.TeamID {
		return nil, constants.ErrTaskNotFound
	}

	// Optimistic locking: verify task version has not changed
	if task.Version != req.Version {
		return nil, constants.ErrStaleVersion
	}

	// Multi-tenancy check: If AssigneeID is updated and non-nil, assignee must exist and belong to the same team
	if req.AssigneeID != nil {
		assignee, err := s.repo.FindUserByID(ctx, *req.AssigneeID)
		if err != nil {
			return nil, s.wrapError(ctx, fmt.Errorf("failed to check assignee: %w", err))
		}
		if assignee == nil || assignee.TeamID != authUser.TeamID {
			return nil, constants.ErrAssigneeNotInTeam
		}
	}

	// Track changes for audit log
	oldStatus := task.Status
	oldAssigneeID := task.AssigneeID

	statusChanged := false
	if req.Status != nil {
		cleanStatus := constants.TaskStatus(strings.TrimSpace(string(*req.Status)))
		if cleanStatus != "" && cleanStatus != task.Status {
			statusChanged = true
			task.Status = cleanStatus
		}
	}

	assigneeChanged := false
	if req.AssigneeID != nil {
		if task.AssigneeID == nil || *task.AssigneeID != *req.AssigneeID {
			assigneeChanged = true
			task.AssigneeID = req.AssigneeID
		}
	}

	detailsChanged := false
	if req.Title != nil {
		cleanTitle := strings.TrimSpace(*req.Title)
		if cleanTitle != "" && cleanTitle != task.Title {
			detailsChanged = true
			task.Title = cleanTitle
		}
	}
	if req.Description != nil {
		cleanDesc := strings.TrimSpace(*req.Description)
		if cleanDesc != task.Description {
			detailsChanged = true
			task.Description = cleanDesc
		}
	}

	expectedVersion := req.Version
	now := time.Now()
	task.UpdatedAt = now
	task.Version = expectedVersion + 1

	// Determine audit log action
	action := constants.TaskActionUpdate
	if statusChanged && !assigneeChanged && !detailsChanged {
		action = constants.TaskActionStatusUpdate
	} else if assigneeChanged && !statusChanged && !detailsChanged {
		action = constants.TaskActionAssign
	}

	notes := "Task updated"
	log := &models.TaskLog{
		ID:        uuid.New(),
		TaskID:    task.ID,
		Action:    action,
		ActorID:   &authUser.UserID,
		Notes:     &notes,
		Metadata:  models.JSONMap{"source": "api"},
		CreatedAt: now,
	}

	if statusChanged {
		log.FromStatus = &oldStatus
		log.ToStatus = &task.Status
	}
	if assigneeChanged {
		log.FromAssigneeID = oldAssigneeID
		log.ToAssigneeID = task.AssigneeID
	}

	if err := s.repo.UpdateTaskWithLog(ctx, task, expectedVersion, log); err != nil {
		return nil, s.wrapError(ctx, fmt.Errorf("failed to update task: %w", err))
	}

	return composeTaskResponse(task), nil
}

// AssignTask assigns an existing task to another user within the same team, records an audit log,
// and sends a notification within a single database transaction.
func (s *Service) AssignTask(ctx context.Context, taskID uuid.UUID, req dtos.AssignTaskRequest) (*dtos.TaskResponse, error) {
	authUser, err := s.getAuthUser(ctx)
	if err != nil {
		return nil, s.wrapError(ctx, err)
	}

	task, err := s.repo.FindTaskByID(ctx, taskID)
	if err != nil {
		return nil, s.wrapError(ctx, fmt.Errorf("failed to find task: %w", err))
	}
	if task == nil || task.TeamID != authUser.TeamID {
		return nil, constants.ErrTaskNotFound
	}

	if task.Version != req.Version {
		return nil, constants.ErrStaleVersion
	}
	expectedVersion := req.Version

	// Validate assignee exists and belongs to the caller's team
	assignee, err := s.repo.FindUserByID(ctx, req.AssigneeID)
	if err != nil {
		return nil, s.wrapError(ctx, fmt.Errorf("failed to check assignee: %w", err))
	}
	if assignee == nil || assignee.TeamID != authUser.TeamID {
		return nil, constants.ErrAssigneeNotInTeam
	}

	oldAssigneeID := task.AssigneeID
	now := time.Now()
	task.AssigneeID = &req.AssigneeID
	task.UpdatedAt = now
	task.Version = expectedVersion + 1

	notes := fmt.Sprintf("Task assigned to %s", assignee.Name)
	log := &models.TaskLog{
		ID:             uuid.New(),
		TaskID:         task.ID,
		Action:         constants.TaskActionAssign,
		ActorID:        &authUser.UserID,
		FromAssigneeID: oldAssigneeID,
		ToAssigneeID:   task.AssigneeID,
		Notes:          &notes,
		Metadata:       models.JSONMap{"source": "api"},
		CreatedAt:      now,
	}

	if err := s.repo.AssignTaskWithLog(ctx, task, expectedVersion, log); err != nil {
		return nil, s.wrapError(ctx, fmt.Errorf("failed to assign task: %w", err))
	}

	// Dispatch notification asynchronously after successful database transaction commit
	go func() {
		if err := s.notifier.SendTaskAssignedNotification(context.Background(), task, assignee); err != nil {
			stdlog.Printf("[WARN] failed to send task assignment notification: %v", err)
		}
	}()

	return composeTaskResponse(task), nil
}

// composeTaskResponse maps a models.Task entity into a dtos.TaskResponse with preloaded relations.
func composeTaskResponse(task *models.Task) *dtos.TaskResponse {
	if task == nil {
		return nil
	}

	resp := &dtos.TaskResponse{
		ID:          task.ID,
		Title:       task.Title,
		Description: task.Description,
		Status:      task.Status,
		CreatorID:   task.CreatorID,
		AssigneeID:  task.AssigneeID,
		TeamID:      task.TeamID,
		Version:     task.Version,
		CreatedAt:   task.CreatedAt,
		UpdatedAt:   task.UpdatedAt,
	}

	if task.Creator != nil {
		resp.Creator = composeUserResponse(task.Creator)
	}

	if task.Assignee != nil {
		resp.Assignee = composeUserResponse(task.Assignee)
	}

	if task.Team != nil {
		teamResp := composeTeamResponse(task.Team)
		resp.Team = &teamResp
	}

	if len(task.Logs) > 0 {
		resp.Logs = make([]*dtos.TaskLogResponse, len(task.Logs))
		for i := range task.Logs {
			l := &task.Logs[i]
			resp.Logs[i] = &dtos.TaskLogResponse{
				ID:             l.ID,
				Action:         l.Action,
				ActorID:        l.ActorID,
				FromAssigneeID: l.FromAssigneeID,
				ToAssigneeID:   l.ToAssigneeID,
				FromStatus:     l.FromStatus,
				ToStatus:       l.ToStatus,
				Notes:          l.Notes,
				Metadata:       l.Metadata,
				CreatedAt:      l.CreatedAt,
			}
		}
	}

	return resp
}
