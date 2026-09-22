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

// DeleteTaskWithLog soft-deletes a task by ID and inserts an audit log within a single database transaction.
func (r *Repositories) DeleteTaskWithLog(ctx context.Context, id uuid.UUID, log *models.TaskLog) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Delete(&models.Task{}, "id = ?", id).Error; err != nil {
			return err
		}
		if log != nil {
			log.TaskID = id
			if err := tx.Create(log).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

// TaskFilter represents search, filter, and pagination criteria for tasks.
type TaskFilter struct {
	TeamID     *uuid.UUID
	CreatorID  *uuid.UUID
	AssigneeID *uuid.UUID
	Status     string
	Title      string
	Offset     int
	Limit      int
}

// FindTasks retrieves a paginated slice of tasks and total count matching filter criteria.
func (r *Repositories) FindTasks(ctx context.Context, filter TaskFilter) ([]models.Task, int64, error) {
	query := r.db.WithContext(ctx).Model(&models.Task{})

	if filter.TeamID != nil {
		query = query.Where("team_id = ?", *filter.TeamID)
	}

	if filter.CreatorID != nil {
		query = query.Where("creator_id = ?", *filter.CreatorID)
	}

	if filter.AssigneeID != nil {
		query = query.Where("assignee_id = ?", *filter.AssigneeID)
	}

	if filter.Status != "" {
		query = query.Where("status = ?", filter.Status)
	}

	if filter.Title != "" {
		query = query.Where("LOWER(title) LIKE LOWER(?)", "%"+filter.Title+"%")
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var tasks []models.Task
	if err := query.Order("created_at DESC").Offset(filter.Offset).Limit(filter.Limit).Find(&tasks).Error; err != nil {
		return nil, 0, err
	}

	return tasks, total, nil
}

