package controllers

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

// HealthCheck handles health status inquiries.
func (c *Controllers) HealthCheck(ctx *gin.Context) {
	ctx.JSON(http.StatusOK, gin.H{
		"version":  c.cfg.Version,
		"uptime":   time.Since(c.startedAt).String(),
		"git_hash": c.cfg.GitHash,
	})
}
