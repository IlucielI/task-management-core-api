package services

import (
	"context"
	"time"

	"task-management/internal/constants"
	"task-management/internal/dtos"
)

// CheckHealth inspects the connectivity of sub-systems (database, redis, s3 storage).
func (s *Service) CheckHealth(ctx context.Context) dtos.HealthServices {
	dbStatus := constants.IntegrationStatusConnected
	if s == nil || s.repo == nil {
		dbStatus = constants.IntegrationStatusDisconnected
	} else {
		pingCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
		defer cancel()
		if err := s.repo.PingDB(pingCtx); err != nil {
			dbStatus = constants.IntegrationStatusDisconnected
		}
	}

	redisStatus := constants.IntegrationStatusConnected
	if s == nil || s.repo == nil {
		redisStatus = constants.IntegrationStatusDisconnected
	} else {
		pingCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
		defer cancel()
		if err := s.repo.PingRedis(pingCtx); err != nil {
			redisStatus = constants.IntegrationStatusDisconnected
		}
	}

	s3Status := constants.IntegrationStatusConnected
	if s == nil || s.storage == nil {
		s3Status = constants.IntegrationStatusDisconnected
	} else {
		pingCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
		defer cancel()
		if err := s.storage.Ping(pingCtx); err != nil {
			s3Status = constants.IntegrationStatusDisconnected
		}
	}

	return dtos.HealthServices{
		Database: dbStatus,
		Redis:    redisStatus,
		S3:       s3Status,
	}
}
