package services

import (
	"context"
	"fmt"

	"task-management/internal/dtos"
)

// GetTeams retrieves all teams matching optional filter criteria.
func (s *Service) GetTeams(ctx context.Context, filter dtos.TeamFilterQuery) ([]dtos.TeamResponse, error) {
	teams, err := s.repo.FindAllTeams(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch teams: %w", err)
	}

	result := make([]dtos.TeamResponse, 0, len(teams))
	for _, t := range teams {
		result = append(result, dtos.TeamResponse{
			ID:        t.ID,
			Name:      t.Name,
			CreatedAt: t.CreatedAt,
			UpdatedAt: t.UpdatedAt,
		})
	}

	return result, nil
}
