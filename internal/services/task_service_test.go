package services

import (
	"context"
	"encoding/json"
	"errors"
	"regexp"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/alicebob/miniredis/v2"
	"github.com/go-redis/redismock/v9"
	"github.com/google/uuid"
	goRedis "github.com/redis/go-redis/v9"
	"gorm.io/gorm"

	redisAdapter "task-management/internal/adapters/redis"
	"task-management/internal/config"
	"task-management/internal/constants"
	"task-management/internal/dtos"
	"task-management/internal/pkg/ctxmeta"
	"task-management/internal/repositories"
)

func TestService_CreateTask_InvalidIdempotencyKey(t *testing.T) {
	svc := New(config.Config{}, nil, nil)
	ctx := context.Background()

	cases := []struct {
		name string
		key  string
	}{
		{"empty key", ""},
		{"non-uuid", "not-a-uuid"},
		{"nil uuid", uuid.Nil.String()},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := svc.CreateTask(ctx, dtos.CreateTaskRequest{Title: "Task"}, tc.key)
			if !errors.Is(err, constants.ErrInvalidIdempotencyKey) {
				t.Fatalf("expected ErrInvalidIdempotencyKey, got: %v", err)
			}
		})
	}
}

func TestService_CreateTask_CachedIdempotent(t *testing.T) {
	client, rmock := redismock.NewClientMock()
	rdb := redisAdapter.NewWithClient(client)
	repo := repositories.New(nil, rdb)
	svc := New(config.Config{}, repo, nil)

	key := uuid.New().String()
	taskID := uuid.New()
	cachedResp := &dtos.CachedIdempotentResponse{
		StatusCode: 201,
		Data: &dtos.TaskResponse{
			ID:    taskID,
			Title: "Cached Task",
		},
		Message:   "Task created successfully",
		Timestamp: time.Now(),
	}
	cachedBytes, _ := json.Marshal(cachedResp)

	rmock.ExpectGet("idempotency:" + key).SetVal(string(cachedBytes))

	resp, err := svc.CreateTask(context.Background(), dtos.CreateTaskRequest{Title: "New Task"}, key)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp == nil || resp.ID != taskID || resp.Title != "Cached Task" {
		t.Fatalf("unexpected task response: %+v", resp)
	}
}

func TestService_CreateTask_Unauthorized(t *testing.T) {
	client, rmock := redismock.NewClientMock()
	rdb := redisAdapter.NewWithClient(client)
	repo := repositories.New(nil, rdb)
	svc := New(config.Config{}, repo, nil)

	key := uuid.New().String()

	// Cache miss
	rmock.ExpectGet("idempotency:" + key).RedisNil()
	// Acquire lock
	rmock.ExpectSetNX("lock:idempotency:"+key, "locked", 30*time.Second).SetVal(true)
	// Double check cache
	rmock.ExpectGet("idempotency:" + key).RedisNil()
	// Release lock on exit
	rmock.ExpectDel("lock:idempotency:" + key).SetVal(1)

	// Context has no auth user
	_, err := svc.CreateTask(context.Background(), dtos.CreateTaskRequest{Title: "Task"}, key)
	if !errors.Is(err, constants.ErrUnauthorized) {
		t.Fatalf("expected ErrUnauthorized, got: %v", err)
	}
}

func TestService_CreateTask_AssigneeNotInTeam(t *testing.T) {
	gormDB, mock := setupMockDB(t)
	client, rmock := redismock.NewClientMock()
	rdb := redisAdapter.NewWithClient(client)
	repo := repositories.New(gormDB, rdb)
	svc := New(config.Config{}, repo, nil)

	creatorTeamID := uuid.New()
	creatorUserID := uuid.New()
	assigneeID := uuid.New()
	otherTeamID := uuid.New()
	now := time.Now()

	ctx := ctxmeta.WithAuthUser(context.Background(), ctxmeta.AuthUser{
		UserID: creatorUserID,
		TeamID: creatorTeamID,
	})

	key := uuid.New().String()

	// 1. Assignee has different team
	rmock.ExpectGet("idempotency:" + key).RedisNil()
	rmock.ExpectSetNX("lock:idempotency:"+key, "locked", 30*time.Second).SetVal(true)
	rmock.ExpectGet("idempotency:" + key).RedisNil()
	rmock.ExpectDel("lock:idempotency:" + key).SetVal(1)

	assigneeRows := sqlmock.NewRows([]string{"id", "name", "email", "password", "team_id", "created_at", "updated_at"}).
		AddRow(assigneeID, "Bob", "bob@example.com", "pass", otherTeamID, now, now)
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "users" WHERE id = $1 ORDER BY "users"."id" LIMIT $2`)).
		WithArgs(assigneeID, 1).
		WillReturnRows(assigneeRows)

	_, err := svc.CreateTask(ctx, dtos.CreateTaskRequest{
		Title:      "Task title",
		AssigneeID: &assigneeID,
	}, key)
	if !errors.Is(err, constants.ErrAssigneeNotInTeam) {
		t.Fatalf("expected ErrAssigneeNotInTeam, got: %v", err)
	}

	// 2. Assignee user does not exist
	key2 := uuid.New().String()
	rmock.ExpectGet("idempotency:" + key2).RedisNil()
	rmock.ExpectSetNX("lock:idempotency:"+key2, "locked", 30*time.Second).SetVal(true)
	rmock.ExpectGet("idempotency:" + key2).RedisNil()
	rmock.ExpectDel("lock:idempotency:" + key2).SetVal(1)

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "users" WHERE id = $1 ORDER BY "users"."id" LIMIT $2`)).
		WithArgs(assigneeID, 1).
		WillReturnError(gorm.ErrRecordNotFound)

	_, err = svc.CreateTask(ctx, dtos.CreateTaskRequest{
		Title:      "Task title",
		AssigneeID: &assigneeID,
	}, key2)
	if !errors.Is(err, constants.ErrAssigneeNotInTeam) {
		t.Fatalf("expected ErrAssigneeNotInTeam, got: %v", err)
	}
}

func TestService_CreateTask_Success_WithAssigneeAndCustomStatus(t *testing.T) {
	gormDB, mock := setupMockDB(t)
	client, rmock := redismock.NewClientMock()
	rdb := redisAdapter.NewWithClient(client)
	repo := repositories.New(gormDB, rdb)
	svc := New(config.Config{}, repo, nil)

	teamID := uuid.New()
	creatorID := uuid.New()
	assigneeID := uuid.New()
	now := time.Now()

	ctx := ctxmeta.WithAuthUser(context.Background(), ctxmeta.AuthUser{
		UserID: creatorID,
		TeamID: teamID,
	})

	key := uuid.New().String()

	rmock.ExpectGet("idempotency:" + key).RedisNil()
	rmock.ExpectSetNX("lock:idempotency:"+key, "locked", 30*time.Second).SetVal(true)
	rmock.ExpectGet("idempotency:" + key).RedisNil()

	// Assignee lookup
	assigneeRows := sqlmock.NewRows([]string{"id", "name", "email", "password", "team_id", "created_at", "updated_at"}).
		AddRow(assigneeID, "Bob", "bob@example.com", "pass", teamID, now, now)
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "users" WHERE id = $1 ORDER BY "users"."id" LIMIT $2`)).
		WithArgs(assigneeID, 1).
		WillReturnRows(assigneeRows)

	// DB Transaction
	mock.ExpectBegin()
	mock.ExpectQuery(regexp.QuoteMeta(`INSERT INTO "tasks"`)).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(uuid.New()))
	mock.ExpectQuery(regexp.QuoteMeta(`INSERT INTO "task_logs"`)).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(uuid.New()))
	mock.ExpectCommit()

	// Redis Cache response
	rmock.Regexp().ExpectSet(`^idempotency:`+key+`$`, `.*`, 24*time.Hour).SetVal("OK")
	rmock.ExpectDel("lock:idempotency:" + key).SetVal(1)

	resp, err := svc.CreateTask(ctx, dtos.CreateTaskRequest{
		Title:       "Integration Test",
		Description: "Detailed task description",
		Status:      constants.TaskStatusInProgress,
		AssigneeID:  &assigneeID,
	}, key)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp == nil {
		t.Fatal("expected non-nil response")
	}
	if resp.Title != "Integration Test" || resp.Status != constants.TaskStatusInProgress {
		t.Fatalf("unexpected task data: %+v", resp)
	}
	if resp.AssigneeID == nil || *resp.AssigneeID != assigneeID {
		t.Fatalf("expected assignee %v, got %v", assigneeID, resp.AssigneeID)
	}
	if resp.CreatorID != creatorID || resp.TeamID != teamID {
		t.Fatalf("mismatched creator/team ids: %+v", resp)
	}
}

func TestService_CreateTask_Success_DefaultStatusAndNoAssignee(t *testing.T) {
	gormDB, mock := setupMockDB(t)
	client, rmock := redismock.NewClientMock()
	rdb := redisAdapter.NewWithClient(client)
	repo := repositories.New(gormDB, rdb)
	svc := New(config.Config{}, repo, nil)

	teamID := uuid.New()
	creatorID := uuid.New()

	ctx := ctxmeta.WithAuthUser(context.Background(), ctxmeta.AuthUser{
		UserID: creatorID,
		TeamID: teamID,
	})

	key := uuid.New().String()

	rmock.ExpectGet("idempotency:" + key).RedisNil()
	rmock.ExpectSetNX("lock:idempotency:"+key, "locked", 30*time.Second).SetVal(true)
	rmock.ExpectGet("idempotency:" + key).RedisNil()

	// DB Transaction
	mock.ExpectBegin()
	mock.ExpectQuery(regexp.QuoteMeta(`INSERT INTO "tasks"`)).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(uuid.New()))
	mock.ExpectQuery(regexp.QuoteMeta(`INSERT INTO "task_logs"`)).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(uuid.New()))
	mock.ExpectCommit()

	rmock.Regexp().ExpectSet(`^idempotency:`+key+`$`, `.*`, 24*time.Hour).SetVal("OK")
	rmock.ExpectDel("lock:idempotency:" + key).SetVal(1)

	resp, err := svc.CreateTask(ctx, dtos.CreateTaskRequest{
		Title: "Task without status or assignee",
	}, key)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Status != constants.TaskStatusTodo {
		t.Fatalf("expected default status 'todo', got %s", resp.Status)
	}
	if resp.AssigneeID != nil {
		t.Fatalf("expected nil assignee, got %v", resp.AssigneeID)
	}
}

func TestService_CreateTask_DBError(t *testing.T) {
	gormDB, mock := setupMockDB(t)
	client, rmock := redismock.NewClientMock()
	rdb := redisAdapter.NewWithClient(client)
	repo := repositories.New(gormDB, rdb)
	svc := New(config.Config{}, repo, nil)

	ctx := ctxmeta.WithAuthUser(context.Background(), ctxmeta.AuthUser{
		UserID: uuid.New(),
		TeamID: uuid.New(),
	})

	key := uuid.New().String()

	rmock.ExpectGet("idempotency:" + key).RedisNil()
	rmock.ExpectSetNX("lock:idempotency:"+key, "locked", 30*time.Second).SetVal(true)
	rmock.ExpectGet("idempotency:" + key).RedisNil()

	mock.ExpectBegin()
	mock.ExpectQuery(regexp.QuoteMeta(`INSERT INTO "tasks"`)).
		WillReturnError(errors.New("db disk full"))
	mock.ExpectRollback()

	rmock.ExpectDel("lock:idempotency:" + key).SetVal(1)

	_, err := svc.CreateTask(ctx, dtos.CreateTaskRequest{Title: "Failing task"}, key)
	if err == nil {
		t.Fatal("expected error on DB failure, got nil")
	}
}

func TestService_CreateTask_Concurrency(t *testing.T) {
	// Use thread-safe in-memory miniredis to test real concurrent SetNX locking & caching
	mr := miniredis.RunT(t)
	rawClient := goRedis.NewClient(&goRedis.Options{
		Addr: mr.Addr(),
	})
	rdb := redisAdapter.NewWithClient(rawClient)
	gormDB, mock := setupMockDB(t)
	repo := repositories.New(gormDB, rdb)
	svc := New(config.Config{}, repo, nil)

	key := uuid.New().String()
	creatorID := uuid.New()
	teamID := uuid.New()

	ctx := ctxmeta.WithAuthUser(context.Background(), ctxmeta.AuthUser{
		UserID: creatorID,
		TeamID: teamID,
	})

	// Exactly ONE database transaction must be executed across all N goroutines
	mock.ExpectBegin()
	mock.ExpectQuery(regexp.QuoteMeta(`INSERT INTO "tasks"`)).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(uuid.New()))
	mock.ExpectQuery(regexp.QuoteMeta(`INSERT INTO "task_logs"`)).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(uuid.New()))
	mock.ExpectCommit()

	const numGoroutines = 10
	var wg sync.WaitGroup
	var successCount int32
	results := make([]*dtos.TaskResponse, numGoroutines)
	errorsList := make([]error, numGoroutines)

	wg.Add(numGoroutines)
	for i := 0; i < numGoroutines; i++ {
		idx := i
		go func() {
			defer wg.Done()
			resp, err := svc.CreateTask(ctx, dtos.CreateTaskRequest{Title: "Concurrent Task"}, key)
			results[idx] = resp
			errorsList[idx] = err
			if err == nil && resp != nil {
				atomic.AddInt32(&successCount, 1)
			}
		}()
	}
	wg.Wait()

	if successCount != int32(numGoroutines) {
		t.Fatalf("expected all %d goroutines to succeed, got %d successes. Errors: %v", numGoroutines, successCount, errorsList)
	}

	// Verify all returned responses are non-nil, identical title and identical task ID
	firstID := results[0].ID
	for i, r := range results {
		if r == nil {
			t.Fatalf("goroutine %d returned nil response", i)
		}
		if r.Title != "Concurrent Task" {
			t.Fatalf("goroutine %d unexpected title: %s", i, r.Title)
		}
		if r.ID != firstID {
			t.Fatalf("goroutine %d returned different task ID: %v vs %v", i, r.ID, firstID)
		}
	}
}

