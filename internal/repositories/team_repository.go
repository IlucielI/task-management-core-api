package repositories

import (
	"context"
	"errors"
	"strings"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"task-management/internal/dtos"
	"task-management/internal/models"
)

// FindAllTeams retrieves teams from the database with optional name filtering.
func (r *Repositories) FindAllTeams(ctx context.Context, filter dtos.TeamFilterQuery) ([]models.Team, error) {
	query := r.db.WithContext(ctx).Model(&models.Team{})

	if name := strings.TrimSpace(filter.Name); name != "" {
		query = query.Where("LOWER(name) LIKE ?", "%"+strings.ToLower(name)+"%")
	}

	var teams []models.Team
	if err := query.Order("name ASC").Find(&teams).Error; err != nil {
		return nil, err
	}

	return teams, nil
}

// FindTeamByID retrieves a team by its unique ID. Returns (nil, nil) if not found.
func (r *Repositories) FindTeamByID(ctx context.Context, id uuid.UUID) (*models.Team, error) {
	var team models.Team
	if err := r.db.WithContext(ctx).First(&team, "id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}

	return &team, nil
}
