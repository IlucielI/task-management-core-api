package validations

import (
	"strings"

	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/google/uuid"

	"task-management/internal/constants"
	"task-management/internal/dtos"
)

var validTaskStatuses = []interface{}{
	constants.TaskStatusTodo,
	constants.TaskStatusInProgress,
	constants.TaskStatusCodeReview,
	constants.TaskStatusReadyForQA,
	constants.TaskStatusDone,
}

// ValidateCreateTaskRequest validates the task creation payload.
func ValidateCreateTaskRequest(req dtos.CreateTaskRequest) error {
	return validation.ValidateStruct(&req,
		validation.Field(&req.Title,
			validation.Required.Error("title is required"),
			validation.Length(1, 255).Error("title must be between 1 and 255 characters"),
		),
		validation.Field(&req.Status,
			validation.In(validTaskStatuses...).Error("status must be a valid task status"),
		),
		validation.Field(&req.AssigneeID, validation.By(func(value interface{}) error {
			if ptr, ok := value.(*uuid.UUID); ok && ptr != nil {
				if *ptr == uuid.Nil {
					return validation.NewError("validation_invalid", "assignee_id cannot be nil UUID")
				}
			}
			return nil
		})),
	)
}

// ValidateUpdateTaskRequest validates the task update payload.
func ValidateUpdateTaskRequest(req dtos.UpdateTaskRequest) error {
	if req.Version == nil {
		return validation.NewError("validation_required", "version is required")
	}
	if *req.Version < 1 {
		return validation.NewError("validation_invalid", "version must be at least 1")
	}

	if req.Title == nil && req.Description == nil && req.Status == nil && req.AssigneeID == nil {
		return validation.NewError("validation_invalid", "at least one field must be provided for update")
	}

	return validation.ValidateStruct(&req,
		validation.Field(&req.Title, validation.By(func(value interface{}) error {
			if ptr, ok := value.(*string); ok && ptr != nil {
				trimmed := strings.TrimSpace(*ptr)
				if trimmed == "" {
					return validation.NewError("validation_invalid", "title cannot be empty")
				}
				if len(trimmed) > 255 {
					return validation.NewError("validation_invalid", "title must be between 1 and 255 characters")
				}
			}
			return nil
		})),
		validation.Field(&req.Status, validation.By(func(value interface{}) error {
			if ptr, ok := value.(*string); ok && ptr != nil {
				trimmed := strings.TrimSpace(*ptr)
				return validation.Validate(trimmed, validation.In(validTaskStatuses...).Error("status must be a valid task status"))
			}
			return nil
		})),
		validation.Field(&req.AssigneeID, validation.By(func(value interface{}) error {
			if ptr, ok := value.(*uuid.UUID); ok && ptr != nil {
				if *ptr == uuid.Nil {
					return validation.NewError("validation_invalid", "assignee_id cannot be nil UUID")
				}
			}
			return nil
		})),
	)
}

// ValidateTaskID validates that the path parameter is a valid non-nil UUID and returns the parsed UUID.
func ValidateTaskID(idStr string) (uuid.UUID, error) {
	trimmed := strings.TrimSpace(idStr)
	if trimmed == "" {
		return uuid.Nil, validation.NewError("validation_invalid", "invalid task id format")
	}

	taskID, err := uuid.Parse(trimmed)
	if err != nil || taskID == uuid.Nil {
		return uuid.Nil, validation.NewError("validation_invalid", "invalid task id format")
	}
	return taskID, nil
}

// ValidateListTasksQuery validates and normalizes pagination and filter query parameters for listing tasks.
func ValidateListTasksQuery(query *dtos.ListTasksQuery) error {
	if query == nil {
		return validation.NewError("validation_invalid", "query cannot be nil")
	}

	// 1. Normalize Limit: default to 10 if <= 0; max 100
	if query.Limit <= 0 {
		query.Limit = 10
	} else if query.Limit > 100 {
		query.Limit = 100
	}

	// 2. Normalize Pagination (page)
	if query.Page <= 0 {
		query.Page = 1
	}

	// 3. Normalize Title query
	query.Title = strings.TrimSpace(query.Title)

	// 4. Validate Status if provided
	query.Status = strings.TrimSpace(query.Status)
	if query.Status != "" {
		if err := validation.Validate(query.Status, validation.In(validTaskStatuses...).Error("status must be a valid task status")); err != nil {
			return err
		}
	}

	// 5. Validate UUID filters if provided
	if query.TeamID != nil && *query.TeamID == uuid.Nil {
		return validation.NewError("validation_invalid", "team_id cannot be nil uuid")
	}
	if query.CreatorID != nil && *query.CreatorID == uuid.Nil {
		return validation.NewError("validation_invalid", "creator_id cannot be nil uuid")
	}
	if query.AssigneeID != nil && *query.AssigneeID == uuid.Nil {
		return validation.NewError("validation_invalid", "assignee_id cannot be nil uuid")
	}

	return nil
}

