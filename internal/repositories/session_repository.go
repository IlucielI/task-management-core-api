package repositories

import (
	"context"
	"fmt"
	"time"

	"task-management/internal/dtos"
)

// SaveSession stores session metadata in Redis with the specified TTL.
func (r *Repositories) SaveSession(ctx context.Context, session dtos.SessionData, ttl time.Duration) error {
	key := fmt.Sprintf("auth:session:%s", session.SessionID)
	return r.rdb.SetJSON(ctx, key, session, ttl)
}

// GetSession retrieves session metadata by session ID from Redis.
func (r *Repositories) GetSession(ctx context.Context, sessionID string) (*dtos.SessionData, error) {
	key := fmt.Sprintf("auth:session:%s", sessionID)
	var session dtos.SessionData
	if err := r.rdb.GetJSON(ctx, key, &session); err != nil {
		return nil, err
	}
	return &session, nil
}

// DeleteSession removes a session key from Redis.
func (r *Repositories) DeleteSession(ctx context.Context, sessionID string) error {
	key := fmt.Sprintf("auth:session:%s", sessionID)
	return r.rdb.Delete(ctx, key)
}
