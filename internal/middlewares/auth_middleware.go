package middlewares

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"task-management/internal/constants"
	"task-management/internal/dtos"
	"task-management/internal/pkg/ctxmeta"
)

// AuthValidator defines the required interface for authenticating incoming requests.
type AuthValidator interface {
	Authenticate(ctx context.Context, tokenStr string) (*ctxmeta.AuthUser, error)
}

// Auth returns a Gin middleware that validates the JWT Bearer token using the provided AuthValidator service.
func Auth(validator AuthValidator) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, dtos.BaseResponse{
				Status:    constants.ResponseStatusFail,
				Code:      constants.ResponseCodeUnauthorized,
				Message:   "missing authorization header",
				Timestamp: time.Now(),
			})
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
			c.AbortWithStatusJSON(http.StatusUnauthorized, dtos.BaseResponse{
				Status:    constants.ResponseStatusFail,
				Code:      constants.ResponseCodeUnauthorized,
				Message:   "invalid authorization header format, expected 'Bearer <token>'",
				Timestamp: time.Now(),
			})
			return
		}

		tokenStr := strings.TrimSpace(parts[1])
		if tokenStr == "" || validator == nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, dtos.BaseResponse{
				Status:    constants.ResponseStatusFail,
				Code:      constants.ResponseCodeUnauthorized,
				Message:   "invalid or expired token",
				Timestamp: time.Now(),
			})
			return
		}

		authUser, err := validator.Authenticate(c.Request.Context(), tokenStr)
		if err != nil || authUser == nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, dtos.BaseResponse{
				Status:    constants.ResponseStatusFail,
				Code:      constants.ResponseCodeUnauthorized,
				Message:   "invalid or expired token",
				Timestamp: time.Now(),
			})
			return
		}

		// Inject authenticated user into standard context
		c.Request = c.Request.WithContext(ctxmeta.WithAuthUser(c.Request.Context(), *authUser))

		c.Next()
	}
}
