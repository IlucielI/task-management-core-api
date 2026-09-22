package validations

import (
	"strings"
	"testing"

	"task-management/internal/constants"
	"task-management/internal/dtos"
)

func TestValidateListTeamsQuery(t *testing.T) {
	t.Run("empty_defaults_applied", func(t *testing.T) {
		query := dtos.ListTeamsQuery{}
		err := ValidateListTeamsQuery(&query)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if query.Page != 1 {
			t.Errorf("expected default page=1, got %d", query.Page)
		}
		if query.Limit != 10 {
			t.Errorf("expected default limit=10, got %d", query.Limit)
		}
		if query.OrderBy != constants.SortByNameAsc {
			t.Errorf("expected default order_by=name-asc, got %v", query.OrderBy)
		}
	})

	t.Run("limit_capped_at_100", func(t *testing.T) {
		query := dtos.ListTeamsQuery{
			Limit: 150,
		}
		err := ValidateListTeamsQuery(&query)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if query.Limit != 100 {
			t.Errorf("expected limit capped at 100, got %d", query.Limit)
		}
	})

	t.Run("name_trim_whitespace", func(t *testing.T) {
		query := dtos.ListTeamsQuery{
			Name: "   Backend Engineering   ",
		}
		err := ValidateListTeamsQuery(&query)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if query.Name != "Backend Engineering" {
			t.Errorf("expected trimmed name, got %q", query.Name)
		}
	})

	t.Run("name_too_long", func(t *testing.T) {
		query := dtos.ListTeamsQuery{
			Name: strings.Repeat("a", 101),
		}
		err := ValidateListTeamsQuery(&query)
		if err == nil {
			t.Fatal("expected error on name exceeding 100 chars, got nil")
		}
	})

	t.Run("valid_sort_orders", func(t *testing.T) {
		validOrders := []constants.SortOrder{
			constants.SortByNameAsc,
			constants.SortByNameDesc,
			constants.SortByLatest,
			constants.SortByEarliest,
		}

		for _, order := range validOrders {
			query := dtos.ListTeamsQuery{
				OrderBy: order,
			}
			err := ValidateListTeamsQuery(&query)
			if err != nil {
				t.Errorf("expected valid sort order %v, got error: %v", order, err)
			}
			if query.OrderBy != order {
				t.Errorf("expected order %v, got %v", order, query.OrderBy)
			}
		}
	})

	t.Run("invalid_sort_order", func(t *testing.T) {
		query := dtos.ListTeamsQuery{
			OrderBy: "invalid_order",
		}
		err := ValidateListTeamsQuery(&query)
		if err == nil {
			t.Fatal("expected error on invalid order_by, got nil")
		}
	})
}

