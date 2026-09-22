package validations

import (
	"strings"
	"testing"

	"task-management/internal/dtos"
)

func TestValidateTeamFilter(t *testing.T) {
	// Valid short name
	if err := ValidateTeamFilter(dtos.TeamFilterQuery{Name: "Engineering"}); err != nil {
		t.Errorf("expected nil error, got: %v", err)
	}

	// Empty name (optional filter)
	if err := ValidateTeamFilter(dtos.TeamFilterQuery{Name: ""}); err != nil {
		t.Errorf("expected nil error, got: %v", err)
	}

	// Name exceeding 100 characters
	longName := strings.Repeat("a", 101)
	if err := ValidateTeamFilter(dtos.TeamFilterQuery{Name: longName}); err == nil {
		t.Error("expected validation error for name exceeding 100 characters, got nil")
	}
}
