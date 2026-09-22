package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"task-management/internal/constants"
)

// Task represents a work item tracked within a team.
type Task struct {
	ID          uuid.UUID            `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	Title       string               `gorm:"type:varchar(255);not null" json:"title"`
	Description string               `gorm:"type:text" json:"description,omitempty"`
	Status      constants.TaskStatus `gorm:"type:varchar(50);not null;default:'todo';index" json:"status"`
	CreatorID   uuid.UUID            `gorm:"type:uuid;not null;index" json:"creator_id"`
	AssigneeID  *uuid.UUID           `gorm:"type:uuid;index" json:"assignee_id,omitempty"`
	TeamID      uuid.UUID            `gorm:"type:uuid;not null;index" json:"team_id"`
	Version     int                  `gorm:"type:int;not null;default:1" json:"version"`
	CreatedAt   time.Time            `gorm:"not null;default:now()" json:"created_at"`
	UpdatedAt   time.Time            `gorm:"not null;default:now()" json:"updated_at"`
	DeletedAt   gorm.DeletedAt       `gorm:"index" json:"-"`

	StatusRef *TaskStatus `gorm:"foreignKey:Status;references:Code" json:"status_ref,omitempty"`
	Creator   *User       `gorm:"foreignKey:CreatorID" json:"creator,omitempty"`
	Assignee  *User       `gorm:"foreignKey:AssigneeID" json:"assignee,omitempty"`
	Team      *Team       `gorm:"foreignKey:TeamID" json:"team,omitempty"`
	Logs      []TaskLog   `gorm:"foreignKey:TaskID" json:"logs,omitempty"`
}

// TableName returns the table name for GORM.
func (Task) TableName() string {
	return "tasks"
}
