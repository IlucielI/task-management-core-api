package validations

import (
	"errors"
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

// ValidateTaskID validates that the path parameter is a valid non-nil UUID and returns the parsed UUID.
func ValidateTaskID(idStr string) (uuid.UUID, error) {
	trimmed := strings.TrimSpace(idStr)
	if trimmed == "" {
		return uuid.Nil, errors.New("invalid task id format")
	}

	taskID, err := uuid.Parse(trimmed)
	if err != nil || taskID == uuid.Nil {
		return uuid.Nil, errors.New("invalid task id format")
	}
	return taskID, nil
}

