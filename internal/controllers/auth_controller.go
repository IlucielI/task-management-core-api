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

// Login handles user authentication and session creation.
func (c *Controllers) Login(ctx *gin.Context) {
	var req dtos.LoginRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, dtos.BaseResponse{
			Success: false,
			Code:    constants.ResponseCodeBadRequest,
			Message: err.Error(),
		})
		return
	}

	if err := validations.ValidateLoginRequest(req); err != nil {
		ctx.JSON(http.StatusBadRequest, dtos.BaseResponse{
			Success: false,
			Code:    constants.ResponseCodeBadRequest,
			Message: err.Error(),
		})
		return
	}

	resp, err := c.svc.Login(ctx.Request.Context(), req)
	if err != nil {
		if errors.Is(err, constants.ErrInvalidCredentials) {
			ctx.JSON(http.StatusUnauthorized, dtos.BaseResponse{
				Success: false,
				Code:    constants.ResponseCodeUnauthorized,
				Message: err.Error(),
			})
			return
		}

		ctx.JSON(http.StatusInternalServerError, dtos.BaseResponse{
			Success: false,
			Code:    constants.ResponseCodeInternalError,
			Message: "Failed to authenticate user",
		})
		return
	}

	ctx.JSON(http.StatusOK, dtos.APIResponse[*dtos.LoginResponse]{
		Success: true,
		Code:    constants.ResponseCodeSuccess,
		Message: "User authenticated successfully",
		Data:    resp,
	})
}


