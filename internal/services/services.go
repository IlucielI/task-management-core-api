package services

import (
	"task-management/internal/adapters/s3"
	"task-management/internal/config"
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
