package models

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"time"

	"github.com/google/uuid"
)

// JSONMap represents a JSONB map stored in PostgreSQL.
type JSONMap map[string]interface{}

// Value implements the database/sql/driver Valuer interface.
func (m JSONMap) Value() (driver.Value, error) {
	if m == nil {
		return "{}", nil
	}
	return json.Marshal(m)
}

// Scan implements the database/sql Scanner interface.
func (m *JSONMap) Scan(value interface{}) error {
	if value == nil {
		*m = JSONMap{}
		return nil
	}

	var bytes []byte
	switch v := value.(type) {
	case []byte:
		bytes = v
	case string:
		bytes = []byte(v)
	default:
		return errors.New("type assertion to []byte/string failed for JSONMap")
	}

	if len(bytes) == 0 {
		*m = JSONMap{}
		return nil
	}

	return json.Unmarshal(bytes, m)
}

// TaskLog represents an audit log entry for changes made to a task.
type TaskLog struct {
	ID             uuid.UUID   `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	TaskID         uuid.UUID   `gorm:"type:uuid;not null;index" json:"task_id"`
	Action         string      `gorm:"type:varchar(50);not null" json:"action"`
	ActorID        *uuid.UUID  `gorm:"type:uuid;index" json:"actor_id,omitempty"`
	FromAssigneeID *uuid.UUID  `gorm:"type:uuid" json:"from_assignee_id,omitempty"`
	ToAssigneeID   *uuid.UUID  `gorm:"type:uuid" json:"to_assignee_id,omitempty"`
	FromStatus     *string     `gorm:"type:varchar(50)" json:"from_status,omitempty"`
	ToStatus       *string     `gorm:"type:varchar(50)" json:"to_status,omitempty"`
	Notes          *string     `gorm:"type:text" json:"notes,omitempty"`
	Metadata       JSONMap     `gorm:"type:jsonb;not null;default:'{}'" json:"metadata"`
	CreatedAt      time.Time   `gorm:"not null;default:now()" json:"created_at"`

	Task          *Task       `gorm:"foreignKey:TaskID" json:"task,omitempty"`
	ActionRef     *TaskAction `gorm:"foreignKey:Action;references:Code" json:"action_ref,omitempty"`
	Actor         *User       `gorm:"foreignKey:ActorID" json:"actor,omitempty"`
	FromAssignee  *User       `gorm:"foreignKey:FromAssigneeID" json:"from_assignee,omitempty"`
	ToAssignee    *User       `gorm:"foreignKey:ToAssigneeID" json:"to_assignee,omitempty"`
	FromStatusRef *TaskStatus `gorm:"foreignKey:FromStatus;references:Code" json:"from_status_ref,omitempty"`
	ToStatusRef   *TaskStatus `gorm:"foreignKey:ToStatus;references:Code" json:"to_status_ref,omitempty"`
}

// TableName returns the table name for GORM.
func (TaskLog) TableName() string {
	return "task_logs"
}
