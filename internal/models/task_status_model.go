package models

import "time"

// TaskStatus represents a lookup entity for valid task workflow statuses.
type TaskStatus struct {
	Code        string    `gorm:"type:varchar(50);primaryKey" json:"code"`
	Name        string    `gorm:"type:varchar(100);not null" json:"name"`
	Description string    `gorm:"type:text" json:"description,omitempty"`
	CreatedAt   time.Time `gorm:"not null;default:now()" json:"created_at"`
}

// TableName returns the table name for GORM.
func (TaskStatus) TableName() string {
	return "task_statuses"
}
