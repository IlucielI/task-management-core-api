package controllers

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"task-management/internal/constants"
	"task-management/internal/dtos"
)

// HealthCheck handles health status inquiries.
func (c *Controllers) HealthCheck(ctx *gin.Context) {
	dbStatus := constants.IntegrationStatusConnected
	if c.db == nil {
		dbStatus = constants.IntegrationStatusDisconnected
	} else {
		pingCtx, cancel := context.WithTimeout(ctx.Request.Context(), 2*time.Second)
		defer cancel()
		if err := c.db.Ping(pingCtx); err != nil {
			dbStatus = constants.IntegrationStatusDisconnected
		}
	}

	redisStatus := constants.IntegrationStatusConnected
	if c.rdb == nil {
		redisStatus = constants.IntegrationStatusDisconnected
	} else {
		pingCtx, cancel := context.WithTimeout(ctx.Request.Context(), 2*time.Second)
		defer cancel()
		if err := c.rdb.Ping(pingCtx); err != nil {
			redisStatus = constants.IntegrationStatusDisconnected
		}
	}

	ctx.JSON(http.StatusOK, dtos.APIResponse[dtos.HealthData]{
		Success: true,
		Code:    constants.ResponseCodeSuccess,
		Message: constants.ResponseMessageSuccess,
		Data: dtos.HealthData{
			Version:  c.cfg.Version,
			GitHash:  c.cfg.GitHash,
			Uptime:   time.Since(c.startedAt).String(),
			Services: dtos.HealthServices{
				Database: dbStatus,
				Redis:    redisStatus,
			},
		},
	})
}
