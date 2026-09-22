package services

import (
	"context"
	"fmt"

	"task-management/internal/dtos"
	"task-management/internal/models"
	"task-management/internal/repositories"
)

// GetTeams retrieves all teams matching filter, sorting, and pagination criteria.
func (s *Service) GetTeams(ctx context.Context, query dtos.ListTeamsQuery) (*dtos.ListTeamsData, error) {
	offset := (query.Page - 1) * query.Limit
	filter := repositories.TeamFilter{
		Name:    query.Name,
		OrderBy: query.OrderBy,
		Offset:  offset,
		Limit:   query.Limit,
	}

	teams, total, err := s.repo.FindAllTeams(ctx, filter)
	if err != nil {
		return nil, s.wrapError(ctx, fmt.Errorf("failed to fetch teams: %w", err))
	}

	totalPages := 0
	if total > 0 && query.Limit > 0 {
		totalPages = int((total + int64(query.Limit) - 1) / int64(query.Limit))
	}

	items := make([]dtos.TeamResponse, len(teams))
	for i := range teams {
		items[i] = composeTeamResponse(&teams[i])
	}

	return &dtos.ListTeamsData{
		Items: items,
		Metadata: dtos.ListMetadata{
			Count:      total,
			Limit:      int64(query.Limit),
			Page:       query.Page,
			TotalPages: totalPages,
		},
	}, nil
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
