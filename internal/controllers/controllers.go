package controllers

import (
	"time"

	"task-management/internal/adapters/database"
	"task-management/internal/adapters/redis"
	"task-management/internal/adapters/s3"
	"task-management/internal/config"
	"task-management/internal/services"
)

// Controllers is the central container for all handler methods.
type Controllers struct {
	cfg       config.Config
	db        *database.Postgres
	rdb       *redis.Redis
	storage   *s3.S3
	svc       *services.Service
	startedAt time.Time
}

// New initializes the central controllers container with application dependencies.
func New(cfg config.Config, db *database.Postgres, rdb *redis.Redis, storage *s3.S3) *Controllers {
	return &Controllers{
		cfg:       cfg,
		db:        db,
		rdb:       rdb,
		storage:   storage,
		svc:       services.New(cfg, db, rdb, storage),
		startedAt: time.Now(),
	}
}

// SetService allows overriding or injecting custom / mock Service for tests.
func (c *Controllers) SetService(s *services.Service) {
	c.svc = s
}
