package repositories

import (
	"context"
	"errors"
	"time"

	"github.com/redis/go-redis/v9"

	"task-management/internal/dtos"
)

const (
	idempotencyKeyPrefix = "idempotency:"
	idempotencyLockPrefix = "lock:idempotency:"
)

// AcquireIdempotencyLock attempts to acquire an atomic distributed lock for the idempotency key using SetNX.
func (r *Repositories) AcquireIdempotencyLock(ctx context.Context, key string, ttl time.Duration) (bool, error) {
	return r.rdb.Client().SetNX(ctx, idempotencyLockPrefix+key, "locked", ttl).Result()
}

// ReleaseIdempotencyLock releases the atomic distributed lock for the idempotency key.
func (r *Repositories) ReleaseIdempotencyLock(ctx context.Context, key string) error {
	return r.rdb.Delete(ctx, idempotencyLockPrefix+key)
}

// GetIdempotencyResponse retrieves the cached response for the given idempotency key.
// Returns (nil, nil) if key is not found in cache.
func (r *Repositories) GetIdempotencyResponse(ctx context.Context, key string) (*dtos.CachedIdempotentResponse, error) {
	var resp dtos.CachedIdempotentResponse
	err := r.rdb.GetJSON(ctx, idempotencyKeyPrefix+key, &resp)
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return nil, nil
		}
		return nil, err
	}
	return &resp, nil
}

// SaveIdempotencyResponse caches the response for the given idempotency key with TTL (e.g. 24 hours).
func (r *Repositories) SaveIdempotencyResponse(ctx context.Context, key string, resp *dtos.CachedIdempotentResponse, ttl time.Duration) error {
	return r.rdb.SetJSON(ctx, idempotencyKeyPrefix+key, resp, ttl)
}
