package repositories

import (
	"context"

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
