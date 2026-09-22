package services

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"

	"task-management/internal/constants"
	"task-management/internal/dtos"
	"task-management/internal/models"
	"task-management/internal/pkg/ctxmeta"
	"task-management/internal/pkg/jwt"
)

// Register creates a new user account after validating team existence, email uniqueness, and hashing password.
func (s *Service) Register(ctx context.Context, req dtos.RegisterRequest) (*dtos.UserResponse, error) {
	// 1. Validate team exists
	team, err := s.repo.FindTeamByID(ctx, req.TeamID)
	if err != nil {
		return nil, s.wrapError(ctx, fmt.Errorf("failed to check team: %w", err))
	}
	if team == nil {
		return nil, constants.ErrTeamNotFound
	}

	// 2. Validate email uniqueness
	existingUser, err := s.repo.FindUserByEmail(ctx, req.Email)
	if err != nil {
		return nil, s.wrapError(ctx, fmt.Errorf("failed to check email: %w", err))
	}
	if existingUser != nil {
		return nil, constants.ErrEmailAlreadyExists
	}

	// 3. Hash password using bcrypt
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, s.wrapError(ctx, fmt.Errorf("failed to hash password: %w", err))
	}

	// 4. Create user record
	user := &models.User{
		Name:     strings.TrimSpace(req.Name),
		Email:    strings.ToLower(strings.TrimSpace(req.Email)),
		Password: string(hashedPassword),
		TeamID:   req.TeamID,
	}

	if err := s.repo.CreateUser(ctx, user); err != nil {
		return nil, s.wrapError(ctx, fmt.Errorf("failed to create user: %w", err))
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

// Login authenticates a user by email and password, creates an active session in Redis via session repository, and issues dual JWT tokens.
func (s *Service) Login(ctx context.Context, req dtos.LoginRequest) (*dtos.LoginResponse, error) {
	// 1. Find user by email
	user, err := s.repo.FindUserByEmail(ctx, req.Email)
	if err != nil {
		return nil, s.wrapError(ctx, fmt.Errorf("failed to check user: %w", err))
	}
	if user == nil {
		return nil, constants.ErrInvalidCredentials
	}

	// 2. Verify password hash
	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.Password)); err != nil {
		return nil, constants.ErrInvalidCredentials
	}

	// 3. Generate session ID and dual JWT tokens
	sessionID := uuid.New().String()
	tokenPair, err := jwt.GenerateTokenPair(s.cfg, user, sessionID)
	if err != nil {
		return nil, s.wrapError(ctx, fmt.Errorf("failed to generate tokens: %w", err))
	}

	// 4. Store stateful session in Redis via repository if configured
	now := time.Now()
	sessionData := dtos.SessionData{
		SessionID:      sessionID,
		UserID:         user.ID,
		Name:           user.Name,
		Email:          user.Email,
		TeamID:         user.TeamID,
		ClientIP:       ctxmeta.GetClientIP(ctx),
		UserAgent:      ctxmeta.GetUserAgent(ctx),
		CreatedAt:      now,
		LastActivityAt: now,
		ExpiresAt:      now.Add(s.cfg.JWTRefreshExpiration),
	}

	if err := s.repo.SaveSession(ctx, sessionData, s.cfg.JWTRefreshExpiration); err != nil {
		return nil, s.wrapError(ctx, fmt.Errorf("failed to save session: %w", err))
	}

	return &dtos.LoginResponse{
		Tokens: *tokenPair,
		User: dtos.UserResponse{
			ID:        user.ID,
			Name:      user.Name,
			Email:     user.Email,
			TeamID:    user.TeamID,
			CreatedAt: user.CreatedAt,
			UpdatedAt: user.UpdatedAt,
		},
	}, nil
}

// AuthValidator defines the contract for validating authentication tokens and active sessions.
type AuthValidator interface {
	Authenticate(ctx context.Context, tokenStr string) (*ctxmeta.AuthUser, error)
}

// Authenticate verifies the JWT Bearer token and checks stateful session in Redis if configured.
func (s *Service) Authenticate(ctx context.Context, tokenStr string) (*ctxmeta.AuthUser, error) {
	if strings.TrimSpace(tokenStr) == "" {
		return nil, constants.ErrInvalidToken
	}

	claims, err := jwt.ValidateToken(s.cfg, tokenStr)
	if err != nil || claims.TokenType != "access" {
		return nil, constants.ErrInvalidToken
	}

	// Verify stateful session in Redis if session repository and session ID are present
	if s.repo != nil && s.repo.Redis() != nil && claims.SessionID != "" {
		session, err := s.repo.GetSession(ctx, claims.SessionID)
		if err != nil || session == nil {
			return nil, constants.ErrInvalidToken
		}
	}

	return &ctxmeta.AuthUser{
		UserID:    claims.UserID,
		Email:     claims.Email,
		TeamID:    claims.TeamID,
		SessionID: claims.SessionID,
	}, nil
}



