package services

import (
	"context"
	"fmt"
	"strings"

	"golang.org/x/crypto/bcrypt"

	"task-management/internal/constants"
	"task-management/internal/dtos"
	"task-management/internal/models"
)

// Register creates a new user account after validating team existence, email uniqueness, and hashing password.
func (s *Service) Register(ctx context.Context, req dtos.RegisterRequest) (*dtos.UserResponse, error) {
	// 1. Validate team exists
	team, err := s.repo.FindTeamByID(ctx, req.TeamID)
	if err != nil {
		return nil, fmt.Errorf("failed to check team: %w", err)
	}
	if team == nil {
		return nil, constants.ErrTeamNotFound
	}

	// 2. Validate email uniqueness
	existingUser, err := s.repo.FindUserByEmail(ctx, req.Email)
	if err != nil {
		return nil, fmt.Errorf("failed to check email: %w", err)
	}
	if existingUser != nil {
		return nil, constants.ErrEmailAlreadyExists
	}

	// 3. Hash password using bcrypt
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("failed to hash password: %w", err)
	}

	// 4. Create user record
	user := &models.User{
		Name:     strings.TrimSpace(req.Name),
		Email:    strings.ToLower(strings.TrimSpace(req.Email)),
		Password: string(hashedPassword),
		TeamID:   req.TeamID,
	}

	if err := s.repo.CreateUser(ctx, user); err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	return &dtos.UserResponse{
		ID:        user.ID,
		Name:      user.Name,
		Email:     user.Email,
		TeamID:    user.TeamID,
		CreatedAt: user.CreatedAt,
		UpdatedAt: user.UpdatedAt,
	}, nil
}
