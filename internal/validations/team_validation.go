package validations

import (
	"strings"

	validation "github.com/go-ozzo/ozzo-validation/v4"

	"task-management/internal/constants"
	"task-management/internal/dtos"
)

var validTeamSortOrders = []interface{}{
	constants.SortByNameAsc,
	constants.SortByNameDesc,
	constants.SortByLatest,
	constants.SortByEarliest,
}

// ValidateListTeamsQuery validates and normalizes pagination and filter query parameters for listing teams.
func ValidateListTeamsQuery(query *dtos.ListTeamsQuery) error {
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

	// 3. Normalize Name query
	query.Name = strings.TrimSpace(query.Name)
	if len(query.Name) > 100 {
		return validation.NewError("validation_invalid", "name cannot exceed 100 characters")
	}

	// 4. Validate OrderBy if provided; default to SortByNameAsc
	if query.OrderBy == "" {
		query.OrderBy = constants.SortByNameAsc
	} else {
		if err := validation.Validate(query.OrderBy, validation.In(validTeamSortOrders...).Error("order_by must be a valid sort order")); err != nil {
			return err
		}
	}

	return nil
}

