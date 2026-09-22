package middlewares

import (
	"github.com/gin-gonic/gin"

	"task-management/internal/pkg/ctxmeta"
)

// ClientMeta extracts client IP and User-Agent header from the Gin request
// and injects them into standard context.Context via ctxmeta.
func ClientMeta() gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.Request != nil {
			ctx := ctxmeta.WithClientMeta(c.Request.Context(), c.ClientIP(), c.Request.UserAgent())
			c.Request = c.Request.WithContext(ctx)
		}
		c.Next()
	}
}
