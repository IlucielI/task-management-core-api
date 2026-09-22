package models

import (
	"time"

	"github.com/google/uuid"
)

// Team represents an organizational team within the task management system.
type Team struct {
	ID        uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	Name      string    `gorm:"type:varchar(100);not null;uniqueIndex" json:"name"`
	CreatedAt time.Time `gorm:"not null;default:now()" json:"created_at"`
	UpdatedAt time.Time `gorm:"not null;default:now()" json:"updated_at"`

	Users []User `gorm:"foreignKey:TeamID" json:"users,omitempty"`
	Tasks []Task `gorm:"foreignKey:TeamID" json:"tasks,omitempty"`
}

// TableName returns the table name for GORM.
func (Team) TableName() string {
	return "teams"
}
