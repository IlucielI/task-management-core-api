package middlewares

import (
	"crypto/subtle"
	"encoding/base64"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"task-management/internal/constants"
	"task-management/internal/dtos"
)

// BasicAuth returns a Gin middleware that validates HTTP Basic Authentication credentials.
// It verifies the Authorization header format ('Basic <base64>') and matches the credentials
// in constant time against expectedUser and expectedPass.
func BasicAuth(expectedUser, expectedPass string) gin.HandlerFunc {
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
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Basic") {
			c.AbortWithStatusJSON(http.StatusUnauthorized, dtos.BaseResponse{
				Status:    constants.ResponseStatusFail,
				Code:      constants.ResponseCodeUnauthorized,
				Message:   "invalid authorization header format, expected 'Basic <credentials>'",
				Timestamp: time.Now(),
			})
			return
		}

		payload, err := base64.StdEncoding.DecodeString(parts[1])
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, dtos.BaseResponse{
				Status:    constants.ResponseStatusFail,
				Code:      constants.ResponseCodeUnauthorized,
				Message:   "invalid basic auth encoding",
				Timestamp: time.Now(),
			})
			return
		}

		pair := strings.SplitN(string(payload), ":", 2)
		if len(pair) != 2 {
			c.AbortWithStatusJSON(http.StatusUnauthorized, dtos.BaseResponse{
				Status:    constants.ResponseStatusFail,
				Code:      constants.ResponseCodeUnauthorized,
				Message:   "invalid basic auth credentials format, expected 'username:password'",
				Timestamp: time.Now(),
			})
			return
		}

		userMatch := subtle.ConstantTimeCompare([]byte(pair[0]), []byte(expectedUser)) == 1
		passMatch := subtle.ConstantTimeCompare([]byte(pair[1]), []byte(expectedPass)) == 1

		if !userMatch || !passMatch {
			c.AbortWithStatusJSON(http.StatusUnauthorized, dtos.BaseResponse{
				Status:    constants.ResponseStatusFail,
				Code:      constants.ResponseCodeUnauthorized,
				Message:   "invalid basic auth credentials",
				Timestamp: time.Now(),
			})
			return
		}

		c.Next()
	}
}
