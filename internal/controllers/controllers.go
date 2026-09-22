package controllers

import (
	"time"

	"task-management/internal/config"
	"task-management/internal/services"
)

// Controllers is the central container for all handler methods.
type Controllers struct {
	cfg       config.Config
	svc       *services.Service
	startedAt time.Time
}

// New initializes the central controllers container with application dependencies.
func New(cfg config.Config, svc *services.Service) *Controllers {
	return &Controllers{
		cfg:       cfg,
		svc:       svc,
		startedAt: time.Now(),
	}
}

// SetService allows overriding or injecting custom / mock Service for tests.
func (c *Controllers) SetService(s *services.Service) {
	c.svc = s
}
