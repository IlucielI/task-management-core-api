package repositories

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/go-redis/redismock/v9"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"

	redisAdapter "task-management/internal/adapters/redis"
	"task-management/internal/dtos"
)

func TestRepositories_Idempotency_WithMockRedis(t *testing.T) {
	client, mock := redismock.NewClientMock()
	rdb := redisAdapter.NewWithClient(client)
	repo := New(nil, rdb)
	ctx := context.Background()

	key := uuid.New().String()
	lockKey := "lock:idempotency:" + key
	respKey := "idempotency:" + key

	// 1. AcquireIdempotencyLock - Success (acquired)
	mock.ExpectSetNX(lockKey, "locked", 30*time.Second).SetVal(true)
	acquired, err := repo.AcquireIdempotencyLock(ctx, key, 30*time.Second)
	if err != nil {
		t.Fatalf("unexpected error on AcquireIdempotencyLock: %v", err)
	}
	if !acquired {
		t.Fatal("expected lock to be acquired")
	}

	// 2. AcquireIdempotencyLock - Failed (already held)
	mock.ExpectSetNX(lockKey, "locked", 30*time.Second).SetVal(false)
	acquired, err = repo.AcquireIdempotencyLock(ctx, key, 30*time.Second)
	if err != nil {
		t.Fatalf("unexpected error on AcquireIdempotencyLock: %v", err)
	}
	if acquired {
		t.Fatal("expected lock NOT to be acquired")
	}

	// 3. ReleaseIdempotencyLock
	mock.ExpectDel(lockKey).SetVal(1)
	if err := repo.ReleaseIdempotencyLock(ctx, key); err != nil {
		t.Fatalf("unexpected error on ReleaseIdempotencyLock: %v", err)
	}

	// 4. SaveIdempotencyResponse
	cachedResp := &dtos.CachedIdempotentResponse{
		StatusCode: 201,
		Message:    "Task created successfully",
		Timestamp:  time.Now(),
		Data: &dtos.TaskResponse{
			ID:    uuid.New(),
			Title: "Test Task",
		},
	}
	respBytes, _ := json.Marshal(cachedResp)
	mock.ExpectSet(respKey, respBytes, 24*time.Hour).SetVal("OK")
	if err := repo.SaveIdempotencyResponse(ctx, key, cachedResp, 24*time.Hour); err != nil {
		t.Fatalf("unexpected error on SaveIdempotencyResponse: %v", err)
	}

	// 5. GetIdempotencyResponse - Found
	mock.ExpectGet(respKey).SetVal(string(respBytes))
	got, err := repo.GetIdempotencyResponse(ctx, key)
	if err != nil {
		t.Fatalf("unexpected error on GetIdempotencyResponse: %v", err)
	}
	if got == nil || got.StatusCode != 201 || got.Data == nil || got.Data.Title != "Test Task" {
		t.Fatalf("unexpected response data: %+v", got)
	}

	// 6. GetIdempotencyResponse - Not Found (redis.Nil)
	mock.ExpectGet(respKey).SetErr(redis.Nil)
	got, err = repo.GetIdempotencyResponse(ctx, key)
	if err != nil {
		t.Fatalf("expected nil error on redis.Nil, got: %v", err)
	}
	if got != nil {
		t.Fatalf("expected nil response on cache miss, got: %+v", got)
	}

	// 7. GetIdempotencyResponse - DB Error
	mock.ExpectGet(respKey).SetErr(errors.New("redis failure"))
	got, err = repo.GetIdempotencyResponse(ctx, key)
	if err == nil {
		t.Fatal("expected error on redis failure, got nil")
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet redis expectations: %v", err)
	}
}
