package repositories

import (
	"context"
	"errors"
	"strings"

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
