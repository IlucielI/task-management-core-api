package controllers

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

// HealthCheck handles health status inquiries.
func (c *Controllers) HealthCheck(ctx *gin.Context) {
	dbStatus := "connected"
	if c.db == nil {
		dbStatus = "disconnected"
	} else {
		pingCtx, cancel := context.WithTimeout(ctx.Request.Context(), 2*time.Second)
		defer cancel()
		if err := c.db.Ping(pingCtx); err != nil {
			dbStatus = "disconnected"
		}
	}

	ctx.JSON(http.StatusOK, gin.H{
		"status":   "ok",
		"version":  c.cfg.Version,
		"uptime":   time.Since(c.startedAt).String(),
		"git_hash": c.cfg.GitHash,
		"services": gin.H{
			"database": dbStatus,
		},
	})
}
