package controllers

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"task-management/internal/constants"
	"task-management/internal/dtos"
	"task-management/internal/validations"
)

// GetTeams handles the HTTP request to fetch all teams with optional query filter.
func (c *Controllers) GetTeams(ctx *gin.Context) {
	var filter dtos.TeamFilterQuery
	if err := ctx.ShouldBindQuery(&filter); err != nil {
		c.wrapError(ctx, constants.ErrBadRequest.Wrap(err))
		return
	}

	if err := validations.ValidateTeamFilter(filter); err != nil {
		c.wrapError(ctx, constants.ErrBadRequest.Wrap(err))
		return
	}

	teams, err := c.svc.GetTeams(ctx.Request.Context(), filter)
	if err != nil {
		c.wrapError(ctx, err)
		return
	}

	ctx.JSON(http.StatusOK, dtos.APIResponse[[]dtos.TeamResponse]{
		Success:   true,
		Code:      constants.ResponseCodeSuccess,
		Message:   "Teams retrieved successfully",
		Data:      teams,
		Timestamp: time.Now(),
	})
}

