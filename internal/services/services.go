package services

import (
	"context"

	"github.com/google/uuid"

	"task-management/internal/adapters/s3"
	"task-management/internal/config"
	"task-management/internal/constants"
	"task-management/internal/pkg/ctxmeta"
	"task-management/internal/repositories"
)

// Service is the unified application service container.
type Service struct {
	cfg     config.Config
	storage *s3.S3
	repo    *repositories.Repositories
}

// New creates a new unified service container.
func New(cfg config.Config, repo *repositories.Repositories, storage *s3.S3) *Service {
	return &Service{
		cfg:     cfg,
		storage: storage,
		repo:    repo,
	}
}

// SetRepositories allows injecting or overriding repositories (e.g. for testing).
func (s *Service) SetRepositories(repo *repositories.Repositories) {
	s.repo = repo
}

// Repositories returns the underlying repositories container.
func (s *Service) Repositories() *repositories.Repositories {
	if s == nil {
		return nil
	}
	return s.repo
}

// wrapError wraps unknown or system errors into structured AppError.
func (s *Service) wrapError(ctx context.Context, err error) error {
	if err == nil {
		return nil
	}
	return constants.ErrInternalServerError.Wrap(err)
}

// getAuthUser extracts and validates the authenticated user from context.
func (s *Service) getAuthUser(ctx context.Context) (ctxmeta.AuthUser, error) {
	authUser, ok := ctxmeta.GetAuthUser(ctx)
	if !ok || authUser.UserID == uuid.Nil {
		return ctxmeta.AuthUser{}, constants.ErrUnauthorized
	}
	return authUser, nil
}

