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

// TaskResponse represents formatted task entity returned to API clients.
type TaskResponse struct {
	ID          uuid.UUID  `json:"id"`
	Title       string     `json:"title"`
	Description string     `json:"description,omitempty"`
	Status      string     `json:"status"`
	CreatorID   uuid.UUID  `json:"creator_id"`
	AssigneeID  *uuid.UUID `json:"assignee_id,omitempty"`
	TeamID      uuid.UUID  `json:"team_id"`
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
