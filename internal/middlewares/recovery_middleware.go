package middlewares

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"runtime/debug"
	"time"

	"github.com/gin-gonic/gin"

	"task-management/internal/constants"
	"task-management/internal/dtos"
	"task-management/internal/pkg/ctxmeta"
)

type recoveryOptions struct {
	writer io.Writer
}

// RecoveryOption configures the Recovery middleware.
type RecoveryOption func(*recoveryOptions)

// WithRecoveryWriter sets a custom output destination for recovery stack trace logs (useful for testing).
func WithRecoveryWriter(w io.Writer) RecoveryOption {
	return func(o *recoveryOptions) {
		if w != nil {
			o.writer = w
		}
	}
}

// Recovery returns a Gin middleware that recovers from any panics, logs the stack trace server-side,
// and returns a unified HTTP 500 JSON error envelope without exposing internal details.
func Recovery(opts ...RecoveryOption) gin.HandlerFunc {
	cfg := &recoveryOptions{
		writer: os.Stderr,
	}
	for _, opt := range opts {
		opt(cfg)
	}

	return func(c *gin.Context) {
		defer func() {
			if r := recover(); r != nil {
				// 1. Get request context metadata
				reqID := ""
				if c.Request != nil {
					reqID = ctxmeta.GetRequestID(c.Request.Context())
				}
				if reqID == "" {
					if id, exists := c.Get("request_id"); exists {
						if idStr, ok := id.(string); ok {
							reqID = idStr
						}
					}
				}
				if reqID == "" {
					reqID = c.GetHeader("X-Request-ID")
				}

				method := ""
				path := ""
				if c.Request != nil {
					method = c.Request.Method
					if c.Request.URL != nil {
						path = c.Request.URL.Path
					}
				}

				stack := debug.Stack()

				// 2. Log full stack trace server-side with request context
				logMsg := fmt.Sprintf(
					"[PANIC RECOVERED] time=%s request_id=%s method=%s path=%s error=%v\nstack:\n%s\n",
					time.Now().UTC().Format(time.RFC3339),
					reqID,
					method,
					path,
					r,
					string(stack),
				)

				if n, writeErr := cfg.writer.Write([]byte(logMsg)); writeErr != nil {
					log.Printf("failed to write recovery log (bytes: %d): %v", n, writeErr)
				}

				// 3. Return unified JSON error response without exposing internal error or stack trace
				resp := dtos.BaseResponse{
					Success:   false,
					Code:      constants.ResponseCodeInternalError,
					Message:   "An internal server error occurred",
					Timestamp: time.Now().UTC(),
				}

				c.AbortWithStatusJSON(http.StatusInternalServerError, resp)
			}
		}()

		c.Next()
	}
}
