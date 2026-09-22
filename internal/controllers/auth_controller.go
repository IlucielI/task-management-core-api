package controllers

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"task-management/internal/constants"
	"task-management/internal/dtos"
	"task-management/internal/validations"
)

// Register handles the HTTP request for user registration.
func (c *Controllers) Register(ctx *gin.Context) {
	var req dtos.RegisterRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, dtos.BaseResponse{
			Success: false,
			Code:    constants.ResponseCodeBadRequest,
			Message: err.Error(),
		})
		return
	}

	if err := validations.ValidateRegisterRequest(req); err != nil {
		ctx.JSON(http.StatusBadRequest, dtos.BaseResponse{
			Success: false,
			Code:    constants.ResponseCodeBadRequest,
			Message: err.Error(),
		})
		return
	}

	user, err := c.svc.Register(ctx.Request.Context(), req)
	if err != nil {
		if errors.Is(err, constants.ErrTeamNotFound) || errors.Is(err, constants.ErrEmailAlreadyExists) {
			ctx.JSON(http.StatusBadRequest, dtos.BaseResponse{
				Success: false,
				Code:    constants.ResponseCodeBadRequest,
				Message: err.Error(),
			})
			return
		}

		ctx.JSON(http.StatusInternalServerError, dtos.BaseResponse{
			Success: false,
			Code:    constants.ResponseCodeInternalError,
			Message: "Failed to register user",
		})
		return
	}

	ctx.JSON(http.StatusCreated, dtos.APIResponse[*dtos.UserResponse]{
		Success: true,
		Code:    constants.ResponseCodeSuccess,
		Message: "User registered successfully",
		Data:    user,
	})
}
