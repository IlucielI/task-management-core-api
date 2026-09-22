package redis

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"

	"task-management/internal/config"
)

// Redis wraps the go-redis client with common helpers and lifecycle control.
type Redis struct {
	client *redis.Client
}

// New initializes and verifies a connection to Redis based on the provided configuration.
func New(cfg config.Config) (*Redis, error) {
	rdb := redis.NewClient(&redis.Options{
		Addr:         cfg.RedisAddr(),
		Password:     cfg.RedisPassword,
		DB:           cfg.RedisDB,
		PoolSize:     cfg.RedisPoolSize,
		DialTimeout:  cfg.RedisDialTimeout,
		ReadTimeout:  cfg.RedisReadTimeout,
		WriteTimeout: cfg.RedisWriteTimeout,
	})

	// Verify connection with timeout
	timeout := cfg.RedisDialTimeout
	if timeout <= 0 {
		timeout = 3 * time.Second
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	if err := rdb.Ping(ctx).Err(); err != nil {
		_ = rdb.Close()
		return nil, fmt.Errorf("failed to ping redis at %s: %w", cfg.RedisAddr(), err)
	}

	return &Redis{client: rdb}, nil
}

// Client returns the underlying *redis.Client for custom/advanced operations.
func (r *Redis) Client() *redis.Client {
	if r == nil {
		return nil
	}
	return r.client
}

// Ping checks if Redis server is reachable.
func (r *Redis) Ping(ctx context.Context) error {
	if r == nil || r.client == nil {
		return fmt.Errorf("redis client is nil")
	}
	return r.client.Ping(ctx).Err()
}

// Get retrieves the value of a string key.
func (r *Redis) Get(ctx context.Context, key string) (string, error) {
	if r == nil || r.client == nil {
		return "", fmt.Errorf("redis client is nil")
	}
	return r.client.Get(ctx, key).Result()
}

// Set stores a key-value pair with an expiration duration (0 for no expiration).
func (r *Redis) Set(ctx context.Context, key string, value any, expiration time.Duration) error {
	if r == nil || r.client == nil {
		return fmt.Errorf("redis client is nil")
	}
	return r.client.Set(ctx, key, value, expiration).Err()
}

// Delete removes one or more keys.
func (r *Redis) Delete(ctx context.Context, keys ...string) error {
	if r == nil || r.client == nil {
		return fmt.Errorf("redis client is nil")
	}
	if len(keys) == 0 {
		return nil
	}
	return r.client.Del(ctx, keys...).Err()
}

// GetJSON retrieves a JSON-serialized string and unmarshals it into dest.
func (r *Redis) GetJSON(ctx context.Context, key string, dest any) error {
	val, err := r.Get(ctx, key)
	if err != nil {
		return err
	}
	return json.Unmarshal([]byte(val), dest)
}

// SetJSON serializes value into JSON and stores it with an expiration duration.
func (r *Redis) SetJSON(ctx context.Context, key string, value any, expiration time.Duration) error {
	bytes, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("failed to marshal json: %w", err)
	}
	return r.Set(ctx, key, bytes, expiration)
}

// Close gracefully closes the Redis client connection pool.
func (r *Redis) Close() error {
	if r == nil || r.client == nil {
		return nil
	}
	return r.client.Close()
}
