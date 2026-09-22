package redis_test

import (
	"context"
	"testing"
	"time"

	"task-management/internal/adapters/redis"
	"task-management/internal/config"
)

func TestRedis_New_Unreachable(t *testing.T) {
	cfg := config.Config{
		RedisHost:        "127.0.0.1",
		RedisPort:        "59999", // non-existent port
		RedisDialTimeout: 50 * time.Millisecond,
	}

	client, err := redis.New(cfg)
	if err == nil {
		t.Fatal("expected error when connecting to unreachable redis, got nil")
	}
	if client != nil {
		t.Fatal("expected client to be nil on failed connection")
	}
}

func TestRedis_NilReceiver(t *testing.T) {
	var r *redis.Redis
	ctx := context.Background()

	if err := r.Ping(ctx); err == nil {
		t.Error("expected error calling Ping on nil Redis, got nil")
	}

	if _, err := r.Get(ctx, "key"); err == nil {
		t.Error("expected error calling Get on nil Redis, got nil")
	}

	if err := r.Set(ctx, "key", "val", 0); err == nil {
		t.Error("expected error calling Set on nil Redis, got nil")
	}

	if err := r.Delete(ctx, "key"); err == nil {
		t.Error("expected error calling Delete on nil Redis, got nil")
	}

	if err := r.Close(); err != nil {
		t.Errorf("expected nil error calling Close on nil Redis, got %v", err)
	}

	if r.Client() != nil {
		t.Error("expected nil *redis.Client from nil Redis")
	}
}
