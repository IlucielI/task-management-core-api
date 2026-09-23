package controllers

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"task-management/internal/constants"
	"task-management/internal/dtos"
)

// HealthCheck handles health status inquiries.
func (c *Controllers) HealthCheck(ctx *gin.Context) {
	servicesStatus := c.svc.CheckHealth(ctx.Request.Context())

	ctx.JSON(http.StatusOK, dtos.APIResponse[dtos.HealthData]{
		Status:  constants.ResponseStatusSuccess,
		Code:    constants.ResponseCodeSuccess,
		Message: constants.ResponseMessageSuccess,
		Data: dtos.HealthData{
			Version:  c.cfg.Version,
			GitHash:  c.cfg.GitHash,
			Uptime:   time.Since(c.startedAt).String(),
			Services: servicesStatus,
		},
		Timestamp: time.Now(),
	})
}
