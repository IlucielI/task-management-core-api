package validations

import (
	"strings"

	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/google/uuid"

	"task-management/internal/dtos"
)

// ValidateListUsersQuery validates and normalizes pagination and filter query parameters for listing users.
func ValidateListUsersQuery(query *dtos.ListUsersQuery) error {
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

	// 3. Normalize Name & Email queries
	query.Name = strings.TrimSpace(query.Name)
	query.Email = strings.TrimSpace(query.Email)

	// 4. Validate TeamID if provided
	if query.TeamID != nil && *query.TeamID == uuid.Nil {
		return validation.NewError("validation_invalid", "team_id cannot be nil uuid")
	}

	return nil
}
