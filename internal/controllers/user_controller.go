package controllers

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"task-management/internal/constants"
	"task-management/internal/dtos"
	"task-management/internal/validations"
)

// ListUsers handles fetching a paginated list of users matching filter criteria.
func (c *Controllers) ListUsers(ctx *gin.Context) {
	var query dtos.ListUsersQuery
	if err := ctx.ShouldBindQuery(&query); err != nil {
		c.wrapError(ctx, constants.ErrBadRequest.Wrap(err))
		return
	}

	if err := validations.ValidateListUsersQuery(&query); err != nil {
		c.wrapError(ctx, constants.ErrBadRequest.Wrap(err))
		return
	}

	res, err := c.svc.ListUsers(ctx.Request.Context(), query)
	if err != nil {
		c.wrapError(ctx, err)
		return
	}

	ctx.JSON(http.StatusOK, dtos.APIResponse[*dtos.ListUsersData]{
		Status:    constants.ResponseStatusSuccess,
		Code:      constants.ResponseCodeSuccess,
		Message:   "Users retrieved successfully",
		Data:      res,
		Timestamp: time.Now(),
	})
}
