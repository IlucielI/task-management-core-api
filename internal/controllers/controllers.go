package controllers

import (
	"time"

	"task-management/internal/adapters/database"
	"task-management/internal/adapters/redis"
	"task-management/internal/adapters/s3"
	"task-management/internal/config"
)

// Controllers is the central container for all handler methods.
type Controllers struct {
	cfg       config.Config
	db        *database.Postgres
	rdb       *redis.Redis
	storage   *s3.S3
	startedAt time.Time
}

// New initializes the central controllers container with application dependencies.
func New(cfg config.Config, db *database.Postgres, rdb *redis.Redis, storage *s3.S3) *Controllers {
	return &Controllers{
		cfg:       cfg,
		db:        db,
		rdb:       rdb,
		storage:   storage,
		startedAt: time.Now(),
	}
}
