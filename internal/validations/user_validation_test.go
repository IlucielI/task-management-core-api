package validations

import (
	"testing"

	"github.com/google/uuid"

	"task-management/internal/constants"
	"task-management/internal/dtos"
)

func TestValidateListUsersQuery(t *testing.T) {
	validUUID := uuid.New()
	nilUUID := uuid.Nil

	tests := []struct {
		name      string
		query     *dtos.ListUsersQuery
		wantErr   bool
		checkFunc func(t *testing.T, q *dtos.ListUsersQuery)
	}{
		{
			name: "empty defaults applied",
			query: &dtos.ListUsersQuery{
				Page:  0,
				Limit: 0,
			},
			wantErr: false,
			checkFunc: func(t *testing.T, q *dtos.ListUsersQuery) {
				if q.Page != 1 || q.Limit != 10 {
					t.Errorf("expected page=1, limit=10, got page=%d, limit=%d", q.Page, q.Limit)
				}
				if q.OrderBy != constants.SortByNameAsc {
					t.Errorf("expected default order_by=name-asc, got %v", q.OrderBy)
				}
			},
		},
		{
			name: "limit capped at 100",
			query: &dtos.ListUsersQuery{
				Page:  1,
				Limit: 500,
			},
			wantErr: false,
			checkFunc: func(t *testing.T, q *dtos.ListUsersQuery) {
				if q.Limit != 100 {
					t.Errorf("expected limit=100, got limit=%d", q.Limit)
				}
			},
		},
		{
			name: "name and email trim whitespace",
			query: &dtos.ListUsersQuery{
				Name:  "  Alice Smith  ",
				Email: "  alice@example.com  ",
			},
			wantErr: false,
			checkFunc: func(t *testing.T, q *dtos.ListUsersQuery) {
				if q.Name != "Alice Smith" {
					t.Errorf("expected name='Alice Smith', got '%s'", q.Name)
				}
				if q.Email != "alice@example.com" {
					t.Errorf("expected email='alice@example.com', got '%s'", q.Email)
				}
			},
		},
		{
			name: "valid uuid team_id",
			query: &dtos.ListUsersQuery{
				TeamID: &validUUID,
			},
			wantErr: false,
		},
		{
			name: "invalid nil team_id",
			query: &dtos.ListUsersQuery{
				TeamID: &nilUUID,
			},
			wantErr: true,
		},
		{
			name: "valid sort order name-desc",
			query: &dtos.ListUsersQuery{
				OrderBy: constants.SortByNameDesc,
			},
			wantErr: false,
			checkFunc: func(t *testing.T, q *dtos.ListUsersQuery) {
				if q.OrderBy != constants.SortByNameDesc {
					t.Errorf("expected order_by=name-desc, got %v", q.OrderBy)
				}
			},
		},
		{
			name: "valid sort order email-asc",
			query: &dtos.ListUsersQuery{
				OrderBy: constants.SortByEmailAsc,
			},
			wantErr: false,
			checkFunc: func(t *testing.T, q *dtos.ListUsersQuery) {
				if q.OrderBy != constants.SortByEmailAsc {
					t.Errorf("expected order_by=email-asc, got %v", q.OrderBy)
				}
			},
		},
		{
			name: "valid sort order email-desc",
			query: &dtos.ListUsersQuery{
				OrderBy: constants.SortByEmailDesc,
			},
			wantErr: false,
			checkFunc: func(t *testing.T, q *dtos.ListUsersQuery) {
				if q.OrderBy != constants.SortByEmailDesc {
					t.Errorf("expected order_by=email-desc, got %v", q.OrderBy)
				}
			},
		},
		{
			name: "valid sort order latest",
			query: &dtos.ListUsersQuery{
				OrderBy: constants.SortByLatest,
			},
			wantErr: false,
			checkFunc: func(t *testing.T, q *dtos.ListUsersQuery) {
				if q.OrderBy != constants.SortByLatest {
					t.Errorf("expected order_by=latest, got %v", q.OrderBy)
				}
			},
		},
		{
			name: "valid sort order earliest",
			query: &dtos.ListUsersQuery{
				OrderBy: constants.SortByEarliest,
			},
			wantErr: false,
			checkFunc: func(t *testing.T, q *dtos.ListUsersQuery) {
				if q.OrderBy != constants.SortByEarliest {
					t.Errorf("expected order_by=earliest, got %v", q.OrderBy)
				}
			},
		},
		{
			name: "invalid sort order",
			query: &dtos.ListUsersQuery{
				OrderBy: "invalid-sort",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateListUsersQuery(tt.query)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateListUsersQuery() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !tt.wantErr && tt.checkFunc != nil {
				tt.checkFunc(t, tt.query)
			}
		})
	}
}
