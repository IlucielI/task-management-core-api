package repositories

import (
	"context"
	"errors"
	"strings"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"task-management/internal/models"
)

// FindUserByEmail retrieves a user by their case-insensitive email. Returns (nil, nil) if not found.
func (r *Repositories) FindUserByEmail(ctx context.Context, email string) (*models.User, error) {
	var user models.User
	cleanEmail := strings.ToLower(strings.TrimSpace(email))

	if err := r.db.WithContext(ctx).First(&user, "LOWER(email) = ?", cleanEmail).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}

	return &user, nil
}

// CreateUser persists a new user into the database.
func (r *Repositories) CreateUser(ctx context.Context, user *models.User) error {
	return r.db.WithContext(ctx).Create(user).Error
}

// FindUserByID retrieves a user by their UUID. Returns (nil, nil) if not found.
func (r *Repositories) FindUserByID(ctx context.Context, id uuid.UUID) (*models.User, error) {
	var user models.User
	if err := r.db.WithContext(ctx).First(&user, "id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &user, nil
}

// UserFilter represents search, filter, and pagination criteria for users.
type UserFilter struct {
	TeamID *uuid.UUID
	Name   string
	Email  string
	Offset int
	Limit  int
}

// FindUsers retrieves a paginated slice of users and total count matching filter criteria.
func (r *Repositories) FindUsers(ctx context.Context, filter UserFilter) ([]models.User, int64, error) {
	query := r.db.WithContext(ctx).Model(&models.User{})

	if filter.TeamID != nil {
		query = query.Where("team_id = ?", *filter.TeamID)
	}

	if filter.Name != "" {
		query = query.Where("LOWER(name) LIKE LOWER(?)", "%"+filter.Name+"%")
	}

	if filter.Email != "" {
		query = query.Where("LOWER(email) LIKE LOWER(?)", "%"+filter.Email+"%")
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var users []models.User
	if err := query.Order("name ASC").Offset(filter.Offset).Limit(filter.Limit).Find(&users).Error; err != nil {
		return nil, 0, err
	}

	return users, total, nil
}

