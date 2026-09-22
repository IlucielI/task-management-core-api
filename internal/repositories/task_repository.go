package repositories

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"task-management/internal/models"
)

// CreateTaskWithLog inserts a task and its initial audit log inside a single database transaction.
func (r *Repositories) CreateTaskWithLog(ctx context.Context, task *models.Task, log *models.TaskLog) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(task).Error; err != nil {
			return err
		}
		if log != nil {
			log.TaskID = task.ID
			if err := tx.Create(log).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

// FindTaskByID retrieves a task by its UUID. Returns (nil, nil) if not found.
func (r *Repositories) FindTaskByID(ctx context.Context, id uuid.UUID) (*models.Task, error) {
	var task models.Task
	if err := r.db.WithContext(ctx).First(&task, "id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &task, nil
}
