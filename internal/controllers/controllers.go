package controllers

import (
	"time"

	"task-management/internal/adapters/database"
	"task-management/internal/config"
)

// Controllers is the central container for all handler methods.
type Controllers struct {
	cfg       config.Config
	db        *database.Postgres
	startedAt time.Time
}

// New initializes the central controllers container with application dependencies.
func New(cfg config.Config, db *database.Postgres) *Controllers {
	return &Controllers{
		cfg:       cfg,
		db:        db,
		startedAt: time.Now(),
	}
}
