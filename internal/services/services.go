package services

import (
	"task-management/internal/adapters/database"
	"task-management/internal/adapters/redis"
	"task-management/internal/adapters/s3"
	"task-management/internal/config"
	"task-management/internal/repositories"
)

// Service is the unified application service container.
type Service struct {
	cfg     config.Config
	db      *database.Postgres
	rdb     *redis.Redis
	storage *s3.S3
	repo    *repositories.Repositories
}

// New creates a new unified service container.
func New(cfg config.Config, db *database.Postgres, rdb *redis.Redis, storage *s3.S3) *Service {
	var repo *repositories.Repositories
	if db != nil && db.DB() != nil {
		repo = repositories.New(db.DB())
	}

	return &Service{
		cfg:     cfg,
		db:      db,
		rdb:     rdb,
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
