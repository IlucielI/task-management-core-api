package models

import (
	"time"

	"github.com/google/uuid"
)

// User represents a system user belonging to a team.
type User struct {
	ID        uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	Name      string    `gorm:"type:varchar(100);not null" json:"name"`
	Email     string    `gorm:"type:varchar(255);not null;uniqueIndex:idx_users_email_lower" json:"email"`
	Password  string    `gorm:"type:varchar(255);not null" json:"-"`
	TeamID    uuid.UUID `gorm:"type:uuid;not null;index" json:"team_id"`
	CreatedAt time.Time `gorm:"not null;default:now()" json:"created_at"`
	UpdatedAt time.Time `gorm:"not null;default:now()" json:"updated_at"`

	Team          *Team     `gorm:"foreignKey:TeamID" json:"team,omitempty"`
	CreatedTasks  []Task    `gorm:"foreignKey:CreatorID" json:"created_tasks,omitempty"`
	AssignedTasks []Task    `gorm:"foreignKey:AssigneeID" json:"assigned_tasks,omitempty"`
	ActionLogs    []TaskLog `gorm:"foreignKey:ActorID" json:"action_logs,omitempty"`
}

// TableName returns the table name for GORM.
func (User) TableName() string {
	return "users"
}
