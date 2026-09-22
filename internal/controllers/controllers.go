package controllers

import (
	"time"

	"task-management/internal/config"
)

// Controllers is the central container for all handler methods.
type Controllers struct {
	cfg       config.Config
	startedAt time.Time
}

// New initializes the central controllers container with application dependencies.
func New(cfg config.Config) *Controllers {
	return &Controllers{
		cfg:       cfg,
		startedAt: time.Now(),
	}
}
