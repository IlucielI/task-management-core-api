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

