package ctxmeta

import (
	"context"

	"github.com/google/uuid"
)

type contextKey string

const (
	clientIPKey  contextKey = "client_ip"
	userAgentKey contextKey = "user_agent"
	authUserKey  contextKey = "auth_user"
)

// AuthUser represents authenticated user identity extracted from JWT and session.
type AuthUser struct {
	UserID    uuid.UUID
	Email     string
	TeamID    uuid.UUID
	SessionID string
}

// WithAuthUser injects the authenticated user into the context.
func WithAuthUser(ctx context.Context, user AuthUser) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	return context.WithValue(ctx, authUserKey, user)
}

// GetAuthUser retrieves the authenticated user from the context.
func GetAuthUser(ctx context.Context) (AuthUser, bool) {
	if ctx == nil {
		return AuthUser{}, false
	}
	user, ok := ctx.Value(authUserKey).(AuthUser)
	return user, ok
}

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
