package validations

import (
	validation "github.com/go-ozzo/ozzo-validation/v4"

	"task-management/internal/dtos"
)

// ValidateTeamFilter validates the team query filter parameters using ozzo-validation.
func ValidateTeamFilter(filter dtos.TeamFilterQuery) error {
	return validation.ValidateStruct(&filter,
		validation.Field(&filter.Name, validation.Length(0, 100)),
	)
}
