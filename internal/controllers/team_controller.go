package controllers

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"task-management/internal/constants"
	"task-management/internal/dtos"
	"task-management/internal/validations"
)

// GetTeams handles the HTTP request to fetch all teams with optional query filter.
func (c *Controllers) GetTeams(ctx *gin.Context) {
	var filter dtos.TeamFilterQuery
	if err := ctx.ShouldBindQuery(&filter); err != nil {
		ctx.JSON(http.StatusBadRequest, dtos.BaseResponse{
			Success: false,
			Code:    constants.ResponseCodeBadRequest,
			Message: err.Error(),
		})
		return
	}

	if err := validations.ValidateTeamFilter(filter); err != nil {
		ctx.JSON(http.StatusBadRequest, dtos.BaseResponse{
			Success: false,
			Code:    constants.ResponseCodeBadRequest,
			Message: err.Error(),
		})
		return
	}

	teams, err := c.svc.GetTeams(ctx.Request.Context(), filter)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, dtos.BaseResponse{
			Success: false,
			Code:    constants.ResponseCodeInternalError,
			Message: "Failed to retrieve teams",
		})
		return
	}

	ctx.JSON(http.StatusOK, dtos.APIResponse[[]dtos.TeamResponse]{
		Success: true,
		Code:    constants.ResponseCodeSuccess,
		Message: "Teams retrieved successfully",
		Data:    teams,
	})
}
