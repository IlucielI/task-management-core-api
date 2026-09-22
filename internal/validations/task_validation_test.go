package validations

import (
	"strings"
	"testing"

	"github.com/google/uuid"

	"task-management/internal/constants"
	"task-management/internal/dtos"
)

func TestValidateCreateTaskRequest(t *testing.T) {
	validAssignee := uuid.New()
	nilUUID := uuid.Nil

	tests := []struct {
		name    string
		req     dtos.CreateTaskRequest
		wantErr bool
	}{
		{
			name: "valid - minimal required fields",
			req: dtos.CreateTaskRequest{
				Title: "Implement API",
			},
			wantErr: false,
		},
		{
			name: "valid - all fields",
			req: dtos.CreateTaskRequest{
				Title:       "Complete user onboarding",
				Description: "Build onboarding flow",
				Status:      constants.TaskStatusInProgress,
				AssigneeID:  &validAssignee,
			},
			wantErr: false,
		},
		{
			name: "invalid - empty title",
			req: dtos.CreateTaskRequest{
				Title: "",
			},
			wantErr: true,
		},
		{
			name: "invalid - title too long",
			req: dtos.CreateTaskRequest{
				Title: strings.Repeat("a", 256),
			},
			wantErr: true,
		},
		{
			name: "invalid - invalid status",
			req: dtos.CreateTaskRequest{
				Title:  "Test task",
				Status: "archived",
			},
			wantErr: true,
		},
		{
			name: "invalid - nil uuid assignee",
			req: dtos.CreateTaskRequest{
				Title:      "Test task",
				AssigneeID: &nilUUID,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateCreateTaskRequest(tt.req)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateCreateTaskRequest() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidateTaskID(t *testing.T) {
	validID := uuid.New()

	tests := []struct {
		name    string
		idStr   string
		wantID  uuid.UUID
		wantErr bool
	}{
		{
			name:    "valid uuid",
			idStr:   validID.String(),
			wantID:  validID,
			wantErr: false,
		},
		{
			name:    "valid uuid with spaces",
			idStr:   "  " + validID.String() + "  ",
			wantID:  validID,
			wantErr: false,
		},
		{
			name:    "empty string",
			idStr:   "",
			wantID:  uuid.Nil,
			wantErr: true,
		},
		{
			name:    "whitespace only",
			idStr:   "   ",
			wantID:  uuid.Nil,
			wantErr: true,
		},
		{
			name:    "malformed uuid",
			idStr:   "not-a-uuid",
			wantID:  uuid.Nil,
			wantErr: true,
		},
		{
			name:    "nil uuid",
			idStr:   uuid.Nil.String(),
			wantID:  uuid.Nil,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ValidateTaskID(tt.idStr)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateTaskID() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !tt.wantErr && got != tt.wantID {
				t.Errorf("ValidateTaskID() got = %v, want %v", got, tt.wantID)
			}
		})
	}
}

func TestValidateListTasksQuery(t *testing.T) {
	validUUID := uuid.New()
	nilUUID := uuid.Nil

	tests := []struct {
		name      string
		query     *dtos.ListTasksQuery
		wantErr   bool
		checkFunc func(t *testing.T, q *dtos.ListTasksQuery)
	}{
		{
			name:    "nil query",
			query:   nil,
			wantErr: true,
		},
		{
			name: "empty defaults applied",
			query: &dtos.ListTasksQuery{
				Page:  0,
				Limit: 0,
			},
			wantErr: false,
			checkFunc: func(t *testing.T, q *dtos.ListTasksQuery) {
				if q.Page != 1 || q.Limit != 10 {
					t.Errorf("expected page=1, limit=10, got page=%d, limit=%d", q.Page, q.Limit)
				}
			},
		},
		{
			name: "limit capped at 100",
			query: &dtos.ListTasksQuery{
				Page:  1,
				Limit: 500,
			},
			wantErr: false,
			checkFunc: func(t *testing.T, q *dtos.ListTasksQuery) {
				if q.Limit != 100 {
					t.Errorf("expected limit=100, got limit=%d", q.Limit)
				}
			},
		},
		{
			name: "title trim whitespace",
			query: &dtos.ListTasksQuery{
				Title: "  Bug Fix  ",
			},
			wantErr: false,
			checkFunc: func(t *testing.T, q *dtos.ListTasksQuery) {
				if q.Title != "Bug Fix" {
					t.Errorf("expected title='Bug Fix', got '%s'", q.Title)
				}
			},
		},
		{
			name: "valid status",
			query: &dtos.ListTasksQuery{
				Status: constants.TaskStatusInProgress,
			},
			wantErr: false,
		},
		{
			name: "invalid status",
			query: &dtos.ListTasksQuery{
				Status: "non_existent_status",
			},
			wantErr: true,
		},
		{
			name: "valid uuid filters",
			query: &dtos.ListTasksQuery{
				TeamID:     &validUUID,
				CreatorID:  &validUUID,
				AssigneeID: &validUUID,
			},
			wantErr: false,
		},
		{
			name: "invalid nil team_id",
			query: &dtos.ListTasksQuery{
				TeamID: &nilUUID,
			},
			wantErr: true,
		},
		{
			name: "invalid nil creator_id",
			query: &dtos.ListTasksQuery{
				CreatorID: &nilUUID,
			},
			wantErr: true,
		},
		{
			name: "invalid nil assignee_id",
			query: &dtos.ListTasksQuery{
				AssigneeID: &nilUUID,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateListTasksQuery(tt.query)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateListTasksQuery() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !tt.wantErr && tt.checkFunc != nil {
				tt.checkFunc(t, tt.query)
			}
		})
	}
}

func TestValidateUpdateTaskRequest(t *testing.T) {
	validVersion := 1
	zeroVersion := 0
	negativeVersion := -1
	validTitle := "Updated Title"
	emptyTitle := ""
	whitespaceTitle := "   "
	tooLongTitle := strings.Repeat("x", 256)
	validDesc := "Updated Description"
	validStatus := constants.TaskStatusInProgress
	invalidStatus := "invalid_status"
	validAssignee := uuid.New()
	nilAssignee := uuid.Nil

	tests := []struct {
		name    string
		req     dtos.UpdateTaskRequest
		wantErr bool
	}{
		{
			name:    "empty request - no fields provided and missing version",
			req:     dtos.UpdateTaskRequest{},
			wantErr: true,
		},
		{
			name: "missing version with valid title",
			req: dtos.UpdateTaskRequest{
				Title: &validTitle,
			},
			wantErr: true,
		},
		{
			name: "zero version with valid title",
			req: dtos.UpdateTaskRequest{
				Version: zeroVersion,
				Title:   &validTitle,
			},
			wantErr: true,
		},
		{
			name: "negative version with valid title",
			req: dtos.UpdateTaskRequest{
				Version: negativeVersion,
				Title:   &validTitle,
			},
			wantErr: true,
		},
		{
			name: "valid - version and title only",
			req: dtos.UpdateTaskRequest{
				Version: validVersion,
				Title:   &validTitle,
			},
			wantErr: false,
		},
		{
			name: "valid - all fields",
			req: dtos.UpdateTaskRequest{
				Version:     validVersion,
				Title:       &validTitle,
				Description: &validDesc,
				Status:      &validStatus,
				AssigneeID:  &validAssignee,
			},
			wantErr: false,
		},
		{
			name: "invalid - empty title",
			req: dtos.UpdateTaskRequest{
				Version: validVersion,
				Title:   &emptyTitle,
			},
			wantErr: true,
		},
		{
			name: "invalid - whitespace only title",
			req: dtos.UpdateTaskRequest{
				Version: validVersion,
				Title:   &whitespaceTitle,
			},
			wantErr: true,
		},
		{
			name: "invalid - title too long",
			req: dtos.UpdateTaskRequest{
				Version: validVersion,
				Title:   &tooLongTitle,
			},
			wantErr: true,
		},
		{
			name: "valid - version and status only",
			req: dtos.UpdateTaskRequest{
				Version: validVersion,
				Status:  &validStatus,
			},
			wantErr: false,
		},
		{
			name: "invalid - invalid status",
			req: dtos.UpdateTaskRequest{
				Version: validVersion,
				Status:  &invalidStatus,
			},
			wantErr: true,
		},
		{
			name: "valid - version and assignee only",
			req: dtos.UpdateTaskRequest{
				Version:    validVersion,
				AssigneeID: &validAssignee,
			},
			wantErr: false,
		},
		{
			name: "invalid - nil uuid assignee",
			req: dtos.UpdateTaskRequest{
				Version:    validVersion,
				AssigneeID: &nilAssignee,
			},
			wantErr: true,
		},
		{
			name: "invalid - version only without update fields",
			req: dtos.UpdateTaskRequest{
				Version: validVersion,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateUpdateTaskRequest(tt.req)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateUpdateTaskRequest() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidateAssignTaskRequest(t *testing.T) {
	validAssignee := uuid.New()
	nilAssignee := uuid.Nil

	tests := []struct {
		name    string
		req     dtos.AssignTaskRequest
		wantErr bool
	}{
		{
			name: "valid - assignee and version",
			req: dtos.AssignTaskRequest{
				AssigneeID: validAssignee,
				Version:    1,
			},
			wantErr: false,
		},
		{
			name: "invalid - missing / zero version",
			req: dtos.AssignTaskRequest{
				AssigneeID: validAssignee,
				Version:    0,
			},
			wantErr: true,
		},
		{
			name: "invalid - negative version",
			req: dtos.AssignTaskRequest{
				AssigneeID: validAssignee,
				Version:    -1,
			},
			wantErr: true,
		},
		{
			name: "invalid - nil assignee",
			req: dtos.AssignTaskRequest{
				AssigneeID: nilAssignee,
				Version:    1,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateAssignTaskRequest(tt.req)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateAssignTaskRequest() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

