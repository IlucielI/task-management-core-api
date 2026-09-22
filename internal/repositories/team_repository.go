package repositories

import (
	"context"
	"errors"
	"strings"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"task-management/internal/constants"
	"task-management/internal/models"
)

// TeamFilter represents filtering, sorting, and pagination criteria for querying teams.
type TeamFilter struct {
	Name    string
	OrderBy constants.SortOrder
	Offset  int
	Limit   int
}

// FindAllTeams retrieves teams from the database with filtering, sorting, and pagination.
func (r *Repositories) FindAllTeams(ctx context.Context, filter TeamFilter) ([]models.Team, int64, error) {
	query := r.db.WithContext(ctx).Model(&models.Team{})

	if name := strings.TrimSpace(filter.Name); name != "" {
		query = query.Where("LOWER(name) LIKE LOWER(?)", "%"+name+"%")
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var orderClause string
	switch filter.OrderBy {
	case constants.SortByNameDesc:
		orderClause = "name DESC"
	case constants.SortByLatest:
		orderClause = "created_at DESC"
	case constants.SortByEarliest:
		orderClause = "created_at ASC"
	default:
		orderClause = "name ASC"
	}

	var teams []models.Team
	if err := query.Order(orderClause).Offset(filter.Offset).Limit(filter.Limit).Find(&teams).Error; err != nil {
		return nil, 0, err
	}

	return teams, total, nil
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
