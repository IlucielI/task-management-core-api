package ctxmeta

import "context"

type contextKey string

const (
	clientIPKey  contextKey = "client_ip"
	userAgentKey contextKey = "user_agent"
)

// WithClientMeta injects client IP and User-Agent metadata into the context.
func WithClientMeta(ctx context.Context, clientIP, userAgent string) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	ctx = context.WithValue(ctx, clientIPKey, clientIP)
	return context.WithValue(ctx, userAgentKey, userAgent)
}

// GetClientIP retrieves the client IP address from context.
func GetClientIP(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	if ip, ok := ctx.Value(clientIPKey).(string); ok {
		return ip
	}
	return ""
}

// GetUserAgent retrieves the User-Agent header from context.
func GetUserAgent(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	if ua, ok := ctx.Value(userAgentKey).(string); ok {
		return ua
	}
	return ""
}
