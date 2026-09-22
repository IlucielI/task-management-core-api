package repositories

import (
	"context"
	"strings"

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
