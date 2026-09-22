package dtos

import (
	"time"

	"github.com/google/uuid"
)

// CreateTaskRequest defines payload for task creation.
type CreateTaskRequest struct {
	Title       string     `json:"title"`
	Description string     `json:"description,omitempty"`
	Status      string     `json:"status,omitempty"`
	AssigneeID  *uuid.UUID `json:"assignee_id,omitempty"`
}

// UpdateTaskRequest defines payload for updating an existing task.
type UpdateTaskRequest struct {
	Version     int        `json:"version"`
	Title       *string    `json:"title,omitempty"`
	Description *string    `json:"description,omitempty"`
	Status      *string    `json:"status,omitempty"`
	AssigneeID  *uuid.UUID `json:"assignee_id,omitempty"`
}

// AssignTaskRequest defines payload for assigning a task to a user.
type AssignTaskRequest struct {
	AssigneeID uuid.UUID `json:"assignee_id"`
	Version    int       `json:"version"`
}

// TaskResponse represents formatted task entity returned to API clients.
type TaskResponse struct {
	ID          uuid.UUID  `json:"id"`
	Title       string     `json:"title"`
	Description string     `json:"description,omitempty"`
	Status      string     `json:"status"`
	CreatorID   uuid.UUID  `json:"creator_id"`
	AssigneeID  *uuid.UUID `json:"assignee_id,omitempty"`
	TeamID      uuid.UUID  `json:"team_id"`
	Version     int        `json:"version"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

// CachedIdempotentResponse caches API response payload in Redis for 24h idempotency window.
type CachedIdempotentResponse struct {
	StatusCode int           `json:"status_code"`
	Data       *TaskResponse `json:"data"`
	Message    string        `json:"message"`
	Timestamp  time.Time     `json:"timestamp"`
}

// ListMetadata represents pagination metadata.
type ListMetadata struct {
	Count      int64 `json:"count"`
	Limit      int64 `json:"limit"`
	Page       int   `json:"page"`
	TotalPages int   `json:"total_pages"`
}

// ListTasksData represents the paginated task list payload containing items and metadata.
type ListTasksData struct {
	Items    []*TaskResponse `json:"items"`
	Metadata ListMetadata    `json:"metadata"`
}

// ListTasksQuery represents URL query parameters for task listing, filtering, and pagination.
type ListTasksQuery struct {
	Page       int        `form:"page"`
	Limit      int        `form:"limit"`
	Status     string     `form:"status"`
	Title      string     `form:"title"`
	TeamID     *uuid.UUID `form:"team_id"`
	CreatorID  *uuid.UUID `form:"creator_id"`
	AssigneeID *uuid.UUID `form:"assignee_id"`
}
