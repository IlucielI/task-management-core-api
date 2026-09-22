package services

import (
	"context"
	"fmt"
	"strings"

	"task-management/internal/constants"
	"task-management/internal/dtos"
	"task-management/internal/repositories"
)

// ListUsers retrieves a paginated list of users matching filter criteria within the caller's team boundary.
func (s *Service) ListUsers(ctx context.Context, query dtos.ListUsersQuery) (*dtos.ListUsersData, error) {
	authUser, err := s.getAuthUser(ctx)
	if err != nil {
		return nil, s.wrapError(ctx, err)
	}

	// Calculate offset from Page & Limit
	offset := (query.Page - 1) * query.Limit

	// Multi-tenant boundary isolation: ensure caller cannot access another team's users
	if query.TeamID != nil && *query.TeamID != authUser.TeamID {
		return nil, constants.ErrTeamNotFound
	}

	targetTeamID := authUser.TeamID

	filter := repositories.UserFilter{
		TeamID: &targetTeamID,
		Name:   strings.TrimSpace(query.Name),
		Email:  strings.TrimSpace(query.Email),
		Offset: offset,
		Limit:  query.Limit,
	}

	users, total, err := s.repo.FindUsers(ctx, filter)
	if err != nil {
		return nil, s.wrapError(ctx, fmt.Errorf("failed to list users: %w", err))
	}

	totalPages := 0
	if total > 0 && query.Limit > 0 {
		totalPages = int((total + int64(query.Limit) - 1) / int64(query.Limit))
	}

	items := make([]*dtos.UserResponse, len(users))
	for i, user := range users {
		items[i] = &dtos.UserResponse{
			ID:        user.ID,
			Name:      user.Name,
			Email:     user.Email,
			TeamID:    user.TeamID,
			CreatedAt: user.CreatedAt,
			UpdatedAt: user.UpdatedAt,
		}
	}

	return &dtos.ListUsersData{
		Items: items,
		Metadata: dtos.ListMetadata{
			Count:      total,
			Limit:      int64(query.Limit),
			Page:       query.Page,
			TotalPages: totalPages,
		},
	}, nil
}
