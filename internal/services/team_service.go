package services

import (
	"context"
	"fmt"

	"task-management/internal/dtos"
	"task-management/internal/models"
)

// GetTeams retrieves all teams matching optional filter criteria.
func (s *Service) GetTeams(ctx context.Context, filter dtos.TeamFilterQuery) ([]dtos.TeamResponse, error) {
	teams, err := s.repo.FindAllTeams(ctx, filter)
	if err != nil {
		return nil, s.wrapError(ctx, fmt.Errorf("failed to fetch teams: %w", err))
	}

	result := make([]dtos.TeamResponse, len(teams))
	for i := range teams {
		result[i] = composeTeamResponse(&teams[i])
	}

	return result, nil
}

// composeTeamResponse maps a models.Team entity into a dtos.TeamResponse.
func composeTeamResponse(team *models.Team) dtos.TeamResponse {
	if team == nil {
		return dtos.TeamResponse{}
	}
	return dtos.TeamResponse{
		ID:        team.ID,
		Name:      team.Name,
		CreatedAt: team.CreatedAt,
		UpdatedAt: team.UpdatedAt,
	}
}
