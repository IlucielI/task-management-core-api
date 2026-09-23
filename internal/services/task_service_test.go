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
	"task-management/internal/models"
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
	svc := New(config.Config{IdempotencyTTL: 24 * time.Hour}, repo, nil)

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
	svc := New(config.Config{IdempotencyTTL: 24 * time.Hour}, repo, nil)

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

func TestService_GetTaskByID(t *testing.T) {
	taskID := uuid.New()
	creatorID := uuid.New()
	teamID := uuid.New()
	otherTeamID := uuid.New()
	now := time.Now()

	cfg := config.Config{}

	t.Run("success_same_team", func(t *testing.T) {
		gormDB, mock := setupMockDB(t)
		repo := repositories.New(gormDB)
		svc := New(cfg, repo, nil)

		ctx := ctxmeta.WithAuthUser(context.Background(), ctxmeta.AuthUser{
			UserID: creatorID,
			TeamID: teamID,
		})

		assigneeID := uuid.New()
		logID := uuid.New()

		rows := sqlmock.NewRows([]string{"id", "title", "description", "status", "creator_id", "assignee_id", "team_id", "created_at", "updated_at"}).
			AddRow(taskID, "Task Title", "Description", "todo", creatorID, &assigneeID, teamID, now, now)
		mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "tasks" WHERE id = $1 AND "tasks"."deleted_at" IS NULL ORDER BY "tasks"."id" LIMIT $2`)).
			WithArgs(taskID, 1).
			WillReturnRows(rows)

		assigneeRows := sqlmock.NewRows([]string{"id", "name", "email", "team_id", "created_at", "updated_at"}).
			AddRow(assigneeID, "Assignee User", "assignee@example.com", teamID, now, now)
		mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "users" WHERE "users"."id" = $1`)).
			WithArgs(assigneeID).
			WillReturnRows(assigneeRows)

		creatorRows := sqlmock.NewRows([]string{"id", "name", "email", "team_id", "created_at", "updated_at"}).
			AddRow(creatorID, "Creator User", "creator@example.com", teamID, now, now)
		mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "users" WHERE "users"."id" = $1`)).
			WithArgs(creatorID).
			WillReturnRows(creatorRows)

		logRows := sqlmock.NewRows([]string{"id", "task_id", "action", "created_at"}).
			AddRow(logID, taskID, "CREATE", now)
		mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "task_logs" WHERE "task_logs"."task_id" = $1 ORDER BY created_at desc`)).
			WithArgs(taskID).
			WillReturnRows(logRows)

		teamRows := sqlmock.NewRows([]string{"id", "name", "created_at", "updated_at"}).
			AddRow(teamID, "Engineering", now, now)
		mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "teams" WHERE "teams"."id" = $1`)).
			WithArgs(teamID).
			WillReturnRows(teamRows)

		resp, err := svc.GetTaskByID(ctx, taskID)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if resp == nil || resp.ID != taskID || resp.Title != "Task Title" || resp.TeamID != teamID {
			t.Fatalf("unexpected task response: %+v", resp)
		}
		if resp.Creator == nil || resp.Creator.Name != "Creator User" {
			t.Fatalf("expected creator to be mapped, got: %+v", resp.Creator)
		}
		if resp.Assignee == nil || resp.Assignee.Name != "Assignee User" {
			t.Fatalf("expected assignee to be mapped, got: %+v", resp.Assignee)
		}
		if resp.Team == nil || resp.Team.Name != "Engineering" {
			t.Fatalf("expected team to be mapped, got: %+v", resp.Team)
		}
		if len(resp.Logs) != 1 || resp.Logs[0].Action != "CREATE" {
			t.Fatalf("expected 1 log with action CREATE, got: %+v", resp.Logs)
		}
	})

	t.Run("not_found_in_db", func(t *testing.T) {
		gormDB, mock := setupMockDB(t)
		repo := repositories.New(gormDB)
		svc := New(cfg, repo, nil)

		ctx := ctxmeta.WithAuthUser(context.Background(), ctxmeta.AuthUser{
			UserID: creatorID,
			TeamID: teamID,
		})

		mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "tasks" WHERE id = $1 AND "tasks"."deleted_at" IS NULL ORDER BY "tasks"."id" LIMIT $2`)).
			WithArgs(taskID, 1).
			WillReturnRows(sqlmock.NewRows([]string{"id", "title"}))

		resp, err := svc.GetTaskByID(ctx, taskID)
		if !errors.Is(err, constants.ErrTaskNotFound) {
			t.Fatalf("expected ErrTaskNotFound, got: %v", err)
		}
		if resp != nil {
			t.Fatalf("expected nil response on not found, got: %+v", resp)
		}
	})

	t.Run("cross_team_isolation_returns_not_found", func(t *testing.T) {
		gormDB, mock := setupMockDB(t)
		repo := repositories.New(gormDB)
		svc := New(cfg, repo, nil)

		// User belongs to teamID, but task belongs to otherTeamID
		ctx := ctxmeta.WithAuthUser(context.Background(), ctxmeta.AuthUser{
			UserID: creatorID,
			TeamID: teamID,
		})

		rows := sqlmock.NewRows([]string{"id", "title", "description", "status", "creator_id", "team_id", "created_at", "updated_at"}).
			AddRow(taskID, "Private Task", "Secret", "todo", creatorID, otherTeamID, now, now)
		mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "tasks" WHERE id = $1 AND "tasks"."deleted_at" IS NULL ORDER BY "tasks"."id" LIMIT $2`)).
			WithArgs(taskID, 1).
			WillReturnRows(rows)

		creatorRows := sqlmock.NewRows([]string{"id", "name", "email", "team_id", "created_at", "updated_at"}).
			AddRow(creatorID, "Creator", "creator@example.com", otherTeamID, now, now)
		mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "users" WHERE "users"."id" = $1`)).
			WithArgs(creatorID).
			WillReturnRows(creatorRows)

		mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "task_logs" WHERE "task_logs"."task_id" = $1 ORDER BY created_at desc`)).
			WithArgs(taskID).
			WillReturnRows(sqlmock.NewRows([]string{"id", "task_id"}))

		teamRows := sqlmock.NewRows([]string{"id", "name", "created_at", "updated_at"}).
			AddRow(otherTeamID, "Other Team", now, now)
		mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "teams" WHERE "teams"."id" = $1`)).
			WithArgs(otherTeamID).
			WillReturnRows(teamRows)

		resp, err := svc.GetTaskByID(ctx, taskID)
		if !errors.Is(err, constants.ErrTaskNotFound) {
			t.Fatalf("expected ErrTaskNotFound on cross-team access, got: %v", err)
		}
		if resp != nil {
			t.Fatalf("expected nil response on cross-team access, got: %+v", resp)
		}
	})

	t.Run("unauthorized_missing_auth_user", func(t *testing.T) {
		svc := New(cfg, nil, nil)

		resp, err := svc.GetTaskByID(context.Background(), taskID)
		if !errors.Is(err, constants.ErrUnauthorized) {
			t.Fatalf("expected ErrUnauthorized, got: %v", err)
		}
		if resp != nil {
			t.Fatalf("expected nil response, got: %+v", resp)
		}
	})

	t.Run("db_error", func(t *testing.T) {
		gormDB, mock := setupMockDB(t)
		repo := repositories.New(gormDB)
		svc := New(cfg, repo, nil)

		ctx := ctxmeta.WithAuthUser(context.Background(), ctxmeta.AuthUser{
			UserID: creatorID,
			TeamID: teamID,
		})

		mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "tasks" WHERE id = $1 AND "tasks"."deleted_at" IS NULL ORDER BY "tasks"."id" LIMIT $2`)).
			WithArgs(taskID, 1).
			WillReturnError(errors.New("connection failed"))

		resp, err := svc.GetTaskByID(ctx, taskID)
		if err == nil {
			t.Fatal("expected error on db failure, got nil")
		}
		if resp != nil {
			t.Fatalf("expected nil response on db failure, got: %+v", resp)
		}
	})
}

func TestService_DeleteTask(t *testing.T) {
	cfg := config.Config{}
	taskID := uuid.New()
	creatorID := uuid.New()
	assigneeID := uuid.New()
	teamID := uuid.New()
	otherTeamID := uuid.New()
	now := time.Now()

	t.Run("success", func(t *testing.T) {
		gormDB, mock := setupMockDB(t)
		repo := repositories.New(gormDB)
		svc := New(cfg, repo, nil)

		ctx := ctxmeta.WithAuthUser(context.Background(), ctxmeta.AuthUser{
			UserID: creatorID,
			TeamID: teamID,
		})

		// 1. FindTaskByID
		rows := sqlmock.NewRows([]string{"id", "title", "description", "status", "creator_id", "assignee_id", "team_id", "created_at", "updated_at"}).
			AddRow(taskID, "Task Title", "Description", "in_progress", creatorID, assigneeID, teamID, now, now)
		mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "tasks" WHERE id = $1 AND "tasks"."deleted_at" IS NULL ORDER BY "tasks"."id" LIMIT $2`)).
			WithArgs(taskID, 1).
			WillReturnRows(rows)

		// 2. DeleteTaskWithLog in transaction
		mock.ExpectBegin()
		mock.ExpectExec(regexp.QuoteMeta(`UPDATE "tasks" SET "deleted_at"=$1 WHERE id = $2 AND "tasks"."deleted_at" IS NULL`)).
			WithArgs(sqlmock.AnyArg(), taskID).
			WillReturnResult(sqlmock.NewResult(1, 1))
		mock.ExpectQuery(regexp.QuoteMeta(`INSERT INTO "task_logs"`)).
			WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(uuid.New()))
		mock.ExpectCommit()

		err := svc.DeleteTask(ctx, taskID)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("not_found_in_db", func(t *testing.T) {
		gormDB, mock := setupMockDB(t)
		repo := repositories.New(gormDB)
		svc := New(cfg, repo, nil)

		ctx := ctxmeta.WithAuthUser(context.Background(), ctxmeta.AuthUser{
			UserID: creatorID,
			TeamID: teamID,
		})

		mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "tasks" WHERE id = $1 AND "tasks"."deleted_at" IS NULL ORDER BY "tasks"."id" LIMIT $2`)).
			WithArgs(taskID, 1).
			WillReturnRows(sqlmock.NewRows([]string{"id", "title"}))

		err := svc.DeleteTask(ctx, taskID)
		if !errors.Is(err, constants.ErrTaskNotFound) {
			t.Fatalf("expected ErrTaskNotFound, got: %v", err)
		}
	})

	t.Run("cross_team_isolation_returns_not_found", func(t *testing.T) {
		gormDB, mock := setupMockDB(t)
		repo := repositories.New(gormDB)
		svc := New(cfg, repo, nil)

		ctx := ctxmeta.WithAuthUser(context.Background(), ctxmeta.AuthUser{
			UserID: creatorID,
			TeamID: teamID,
		})

		rows := sqlmock.NewRows([]string{"id", "title", "description", "status", "creator_id", "assignee_id", "team_id", "created_at", "updated_at"}).
			AddRow(taskID, "Secret Task", "Desc", "todo", creatorID, nil, otherTeamID, now, now)
		mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "tasks" WHERE id = $1 AND "tasks"."deleted_at" IS NULL ORDER BY "tasks"."id" LIMIT $2`)).
			WithArgs(taskID, 1).
			WillReturnRows(rows)

		err := svc.DeleteTask(ctx, taskID)
		if !errors.Is(err, constants.ErrTaskNotFound) {
			t.Fatalf("expected ErrTaskNotFound on cross-team delete, got: %v", err)
		}
	})

	t.Run("unauthorized_missing_auth_user", func(t *testing.T) {
		svc := New(cfg, nil, nil)

		err := svc.DeleteTask(context.Background(), taskID)
		if !errors.Is(err, constants.ErrUnauthorized) {
			t.Fatalf("expected ErrUnauthorized, got: %v", err)
		}
	})

	t.Run("db_error_on_find", func(t *testing.T) {
		gormDB, mock := setupMockDB(t)
		repo := repositories.New(gormDB)
		svc := New(cfg, repo, nil)

		ctx := ctxmeta.WithAuthUser(context.Background(), ctxmeta.AuthUser{
			UserID: creatorID,
			TeamID: teamID,
		})

		mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "tasks" WHERE id = $1 AND "tasks"."deleted_at" IS NULL ORDER BY "tasks"."id" LIMIT $2`)).
			WithArgs(taskID, 1).
			WillReturnError(errors.New("db query error"))

		err := svc.DeleteTask(ctx, taskID)
		if err == nil {
			t.Fatal("expected error on db query failure, got nil")
		}
	})

	t.Run("db_error_on_delete", func(t *testing.T) {
		gormDB, mock := setupMockDB(t)
		repo := repositories.New(gormDB)
		svc := New(cfg, repo, nil)

		ctx := ctxmeta.WithAuthUser(context.Background(), ctxmeta.AuthUser{
			UserID: creatorID,
			TeamID: teamID,
		})

		rows := sqlmock.NewRows([]string{"id", "title", "description", "status", "creator_id", "assignee_id", "team_id", "created_at", "updated_at"}).
			AddRow(taskID, "Task Title", "Description", "todo", creatorID, nil, teamID, now, now)
		mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "tasks" WHERE id = $1 AND "tasks"."deleted_at" IS NULL ORDER BY "tasks"."id" LIMIT $2`)).
			WithArgs(taskID, 1).
			WillReturnRows(rows)

		mock.ExpectBegin()
		mock.ExpectExec(regexp.QuoteMeta(`UPDATE "tasks" SET "deleted_at"=$1 WHERE id = $2 AND "tasks"."deleted_at" IS NULL`)).
			WithArgs(sqlmock.AnyArg(), taskID).
			WillReturnError(errors.New("db delete error"))
		mock.ExpectRollback()

		err := svc.DeleteTask(ctx, taskID)
		if err == nil {
			t.Fatal("expected error on db delete failure, got nil")
		}
	})
}

func TestService_ListTasks(t *testing.T) {
	cfg := config.Config{}
	creatorID := uuid.New()
	teamID := uuid.New()
	task1ID := uuid.New()
	task2ID := uuid.New()
	now := time.Now()

	t.Run("success_with_page_and_limit", func(t *testing.T) {
		gormDB, mock := setupMockDB(t)
		repo := repositories.New(gormDB)
		svc := New(cfg, repo, nil)

		ctx := ctxmeta.WithAuthUser(context.Background(), ctxmeta.AuthUser{
			UserID: creatorID,
			TeamID: teamID,
		})

		query := dtos.ListTasksQuery{
			Page:  2,
			Limit: 2,
		}

		mock.ExpectQuery(regexp.QuoteMeta(`SELECT count(*) FROM "tasks" WHERE team_id = $1 AND "tasks"."deleted_at" IS NULL`)).
			WithArgs(teamID).
			WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(5))

		rows := sqlmock.NewRows([]string{"id", "title", "description", "status", "creator_id", "team_id", "created_at", "updated_at"}).
			AddRow(task1ID, "Task 3", "desc", "todo", creatorID, teamID, now, now).
			AddRow(task2ID, "Task 4", "desc", "todo", creatorID, teamID, now, now)
		mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "tasks" WHERE team_id = $1 AND "tasks"."deleted_at" IS NULL ORDER BY created_at DESC LIMIT $2 OFFSET $3`)).
			WithArgs(teamID, 2, 2).
			WillReturnRows(rows)

		resp, err := svc.ListTasks(ctx, query)
		if err != nil {
			t.Fatalf("unexpected error listing tasks: %v", err)
		}
		if resp == nil || len(resp.Items) != 2 {
			t.Fatalf("expected 2 items, got %+v", resp)
		}
		if resp.Metadata.Count != 5 || resp.Metadata.Limit != 2 || resp.Metadata.Page != 2 || resp.Metadata.TotalPages != 3 {
			t.Fatalf("unexpected metadata: %+v", resp.Metadata)
		}
	})

	t.Run("success_with_filter_and_search", func(t *testing.T) {
		gormDB, mock := setupMockDB(t)
		repo := repositories.New(gormDB)
		svc := New(cfg, repo, nil)

		ctx := ctxmeta.WithAuthUser(context.Background(), ctxmeta.AuthUser{
			UserID: creatorID,
			TeamID: teamID,
		})

		query := dtos.ListTasksQuery{
			Page:   1,
			Limit:  10,
			Status: "in_progress",
			Title:  "Payment",
		}

		mock.ExpectQuery(regexp.QuoteMeta(`SELECT count(*) FROM "tasks" WHERE team_id = $1 AND status = $2 AND LOWER(title) LIKE LOWER($3) AND "tasks"."deleted_at" IS NULL`)).
			WithArgs(teamID, "in_progress", "%Payment%").
			WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))

		rows := sqlmock.NewRows([]string{"id", "title", "description", "status", "creator_id", "team_id", "created_at", "updated_at"}).
			AddRow(task1ID, "Fix Payment Gateway", "desc", "in_progress", creatorID, teamID, now, now)
		mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "tasks" WHERE team_id = $1 AND status = $2 AND LOWER(title) LIKE LOWER($3) AND "tasks"."deleted_at" IS NULL ORDER BY created_at DESC LIMIT $4`)).
			WithArgs(teamID, "in_progress", "%Payment%", 10).
			WillReturnRows(rows)

		resp, err := svc.ListTasks(ctx, query)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if resp == nil || len(resp.Items) != 1 || resp.Items[0].Title != "Fix Payment Gateway" {
			t.Fatalf("unexpected response: %+v", resp)
		}
	})

	t.Run("unauthorized_missing_auth_user", func(t *testing.T) {
		svc := New(cfg, nil, nil)

		_, err := svc.ListTasks(context.Background(), dtos.ListTasksQuery{Page: 1, Limit: 10})
		if !errors.Is(err, constants.ErrUnauthorized) {
			t.Fatalf("expected ErrUnauthorized, got: %v", err)
		}
	})

	t.Run("cross_team_query_returns_not_found", func(t *testing.T) {
		svc := New(cfg, nil, nil)

		ctx := ctxmeta.WithAuthUser(context.Background(), ctxmeta.AuthUser{
			UserID: creatorID,
			TeamID: teamID,
		})

		otherTeamID := uuid.New()
		_, err := svc.ListTasks(ctx, dtos.ListTasksQuery{
			Page:   1,
			Limit:  10,
			TeamID: &otherTeamID,
		})
		if !errors.Is(err, constants.ErrTaskNotFound) {
			t.Fatalf("expected ErrTaskNotFound on cross-team query, got: %v", err)
		}
	})

	t.Run("success_with_creator_and_assignee_filters", func(t *testing.T) {
		gormDB, mock := setupMockDB(t)
		repo := repositories.New(gormDB)
		svc := New(cfg, repo, nil)

		ctx := ctxmeta.WithAuthUser(context.Background(), ctxmeta.AuthUser{
			UserID: creatorID,
			TeamID: teamID,
		})

		assigneeID := uuid.New()
		query := dtos.ListTasksQuery{
			Page:       1,
			Limit:      10,
			TeamID:     &teamID,
			CreatorID:  &creatorID,
			AssigneeID: &assigneeID,
		}

		mock.ExpectQuery(regexp.QuoteMeta(`SELECT count(*) FROM "tasks" WHERE team_id = $1 AND creator_id = $2 AND assignee_id = $3 AND "tasks"."deleted_at" IS NULL`)).
			WithArgs(teamID, creatorID, assigneeID).
			WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))

		rows := sqlmock.NewRows([]string{"id", "title", "description", "status", "creator_id", "assignee_id", "team_id", "created_at", "updated_at"}).
			AddRow(task1ID, "Assigned Task", "desc", "todo", creatorID, assigneeID, teamID, now, now)
		mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "tasks" WHERE team_id = $1 AND creator_id = $2 AND assignee_id = $3 AND "tasks"."deleted_at" IS NULL ORDER BY created_at DESC LIMIT $4`)).
			WithArgs(teamID, creatorID, assigneeID, 10).
			WillReturnRows(rows)

		resp, err := svc.ListTasks(ctx, query)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if resp == nil || len(resp.Items) != 1 || *resp.Items[0].AssigneeID != assigneeID {
			t.Fatalf("unexpected response: %+v", resp)
		}
	})

	t.Run("db_error", func(t *testing.T) {
		gormDB, mock := setupMockDB(t)
		repo := repositories.New(gormDB)
		svc := New(cfg, repo, nil)

		ctx := ctxmeta.WithAuthUser(context.Background(), ctxmeta.AuthUser{
			UserID: creatorID,
			TeamID: teamID,
		})

		mock.ExpectQuery(regexp.QuoteMeta(`SELECT count(*) FROM "tasks" WHERE team_id = $1 AND "tasks"."deleted_at" IS NULL`)).
			WithArgs(teamID).
			WillReturnError(errors.New("db error"))

		resp, err := svc.ListTasks(ctx, dtos.ListTasksQuery{Page: 1, Limit: 10})
		if err == nil {
			t.Fatal("expected error on db failure, got nil")
		}
		if resp != nil {
			t.Fatalf("expected nil response, got %+v", resp)
		}
	})
}

func TestService_UpdateTask(t *testing.T) {
	taskID := uuid.New()
	creatorID := uuid.New()
	assigneeID := uuid.New()
	newAssigneeID := uuid.New()
	teamID := uuid.New()
	otherTeamID := uuid.New()
	logID := uuid.New()
	now := time.Now()

	cfg := config.Config{}

	newTitle := "Updated Task Title"
	newDesc := "Updated Task Description"
	newStatus := constants.TaskStatusInProgress

	version := 1
	staleVersion := 2

	t.Run("success_update_all_fields", func(t *testing.T) {
		gormDB, mock := setupMockDB(t)
		repo := repositories.New(gormDB)
		svc := New(cfg, repo, nil)

		ctx := ctxmeta.WithAuthUser(context.Background(), ctxmeta.AuthUser{
			UserID: creatorID,
			TeamID: teamID,
		})

		// 1. Find task
		taskRows := sqlmock.NewRows([]string{"id", "title", "description", "status", "creator_id", "assignee_id", "team_id", "version", "created_at", "updated_at"}).
			AddRow(taskID, "Old Title", "Old Desc", "todo", creatorID, &assigneeID, teamID, version, now, now)
		mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "tasks" WHERE id = $1 AND "tasks"."deleted_at" IS NULL ORDER BY "tasks"."id" LIMIT $2`)).
			WithArgs(taskID, 1).
			WillReturnRows(taskRows)

		// 2. Find new assignee in same team
		assigneeRows := sqlmock.NewRows([]string{"id", "name", "email", "team_id", "created_at", "updated_at"}).
			AddRow(newAssigneeID, "New Assignee", "assignee@example.com", teamID, now, now)
		mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "users" WHERE id = $1 ORDER BY "users"."id" LIMIT $2`)).
			WithArgs(newAssigneeID, 1).
			WillReturnRows(assigneeRows)

		// 3. Update task and insert log in transaction
		mock.ExpectBegin()
		mock.ExpectExec(regexp.QuoteMeta(`UPDATE "tasks"`)).
			WillReturnResult(sqlmock.NewResult(1, 1))
		mock.ExpectQuery(regexp.QuoteMeta(`INSERT INTO "task_logs"`)).
			WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(logID))
		mock.ExpectCommit()

		req := dtos.UpdateTaskRequest{
			Version:     version,
			Title:       &newTitle,
			Description: &newDesc,
			Status:      &newStatus,
			AssigneeID:  &newAssigneeID,
		}

		resp, err := svc.UpdateTask(ctx, taskID, req)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if resp == nil || resp.ID != taskID || resp.Title != newTitle || resp.Status != newStatus || *resp.AssigneeID != newAssigneeID || resp.Version != 2 {
			t.Fatalf("unexpected updated task response: %+v", resp)
		}
	})

	t.Run("stale_task_version_conflict", func(t *testing.T) {
		gormDB, mock := setupMockDB(t)
		repo := repositories.New(gormDB)
		svc := New(cfg, repo, nil)

		ctx := ctxmeta.WithAuthUser(context.Background(), ctxmeta.AuthUser{
			UserID: creatorID,
			TeamID: teamID,
		})

		// Task in DB is at version 2
		taskRows := sqlmock.NewRows([]string{"id", "title", "description", "status", "creator_id", "assignee_id", "team_id", "version", "created_at", "updated_at"}).
			AddRow(taskID, "Title", "Desc", "todo", creatorID, nil, teamID, staleVersion, now, now)
		mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "tasks" WHERE id = $1 AND "tasks"."deleted_at" IS NULL ORDER BY "tasks"."id" LIMIT $2`)).
			WithArgs(taskID, 1).
			WillReturnRows(taskRows)

		// Client sends update with stale version 1
		req := dtos.UpdateTaskRequest{
			Version: version,
			Title:   &newTitle,
		}

		resp, err := svc.UpdateTask(ctx, taskID, req)
		if !errors.Is(err, constants.ErrStaleVersion) {
			t.Fatalf("expected ErrStaleVersion, got: %v", err)
		}
		if resp != nil {
			t.Fatalf("expected nil response on conflict, got: %+v", resp)
		}
	})

	t.Run("success_status_only_update", func(t *testing.T) {
		gormDB, mock := setupMockDB(t)
		repo := repositories.New(gormDB)
		svc := New(cfg, repo, nil)

		ctx := ctxmeta.WithAuthUser(context.Background(), ctxmeta.AuthUser{
			UserID: creatorID,
			TeamID: teamID,
		})

		taskRows := sqlmock.NewRows([]string{"id", "title", "description", "status", "creator_id", "assignee_id", "team_id", "version", "created_at", "updated_at"}).
			AddRow(taskID, "Title", "Desc", "todo", creatorID, nil, teamID, version, now, now)
		mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "tasks" WHERE id = $1 AND "tasks"."deleted_at" IS NULL ORDER BY "tasks"."id" LIMIT $2`)).
			WithArgs(taskID, 1).
			WillReturnRows(taskRows)

		mock.ExpectBegin()
		mock.ExpectExec(regexp.QuoteMeta(`UPDATE "tasks"`)).
			WillReturnResult(sqlmock.NewResult(1, 1))
		mock.ExpectQuery(regexp.QuoteMeta(`INSERT INTO "task_logs"`)).
			WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(logID))
		mock.ExpectCommit()

		status := constants.TaskStatusDone
		req := dtos.UpdateTaskRequest{
			Version: version,
			Status:  &status,
		}

		resp, err := svc.UpdateTask(ctx, taskID, req)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if resp == nil || resp.Status != constants.TaskStatusDone || resp.Version != 2 {
			t.Fatalf("unexpected task response: %+v", resp)
		}
	})

	t.Run("success_assignee_only_update", func(t *testing.T) {
		gormDB, mock := setupMockDB(t)
		repo := repositories.New(gormDB)
		svc := New(cfg, repo, nil)

		ctx := ctxmeta.WithAuthUser(context.Background(), ctxmeta.AuthUser{
			UserID: creatorID,
			TeamID: teamID,
		})

		taskRows := sqlmock.NewRows([]string{"id", "title", "description", "status", "creator_id", "assignee_id", "team_id", "version", "created_at", "updated_at"}).
			AddRow(taskID, "Title", "Desc", "todo", creatorID, nil, teamID, version, now, now)
		mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "tasks" WHERE id = $1 AND "tasks"."deleted_at" IS NULL ORDER BY "tasks"."id" LIMIT $2`)).
			WithArgs(taskID, 1).
			WillReturnRows(taskRows)

		assigneeRows := sqlmock.NewRows([]string{"id", "name", "email", "team_id", "created_at", "updated_at"}).
			AddRow(newAssigneeID, "Assignee", "assignee@example.com", teamID, now, now)
		mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "users" WHERE id = $1 ORDER BY "users"."id" LIMIT $2`)).
			WithArgs(newAssigneeID, 1).
			WillReturnRows(assigneeRows)

		mock.ExpectBegin()
		mock.ExpectExec(regexp.QuoteMeta(`UPDATE "tasks"`)).
			WillReturnResult(sqlmock.NewResult(1, 1))
		mock.ExpectQuery(regexp.QuoteMeta(`INSERT INTO "task_logs"`)).
			WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(logID))
		mock.ExpectCommit()

		req := dtos.UpdateTaskRequest{
			Version:    version,
			AssigneeID: &newAssigneeID,
		}

		resp, err := svc.UpdateTask(ctx, taskID, req)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if resp == nil || *resp.AssigneeID != newAssigneeID || resp.Version != 2 {
			t.Fatalf("unexpected task response: %+v", resp)
		}
	})

	t.Run("task_not_found_in_db", func(t *testing.T) {
		gormDB, mock := setupMockDB(t)
		repo := repositories.New(gormDB)
		svc := New(cfg, repo, nil)

		ctx := ctxmeta.WithAuthUser(context.Background(), ctxmeta.AuthUser{
			UserID: creatorID,
			TeamID: teamID,
		})

		mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "tasks" WHERE id = $1 AND "tasks"."deleted_at" IS NULL ORDER BY "tasks"."id" LIMIT $2`)).
			WithArgs(taskID, 1).
			WillReturnRows(sqlmock.NewRows([]string{"id"}))

		req := dtos.UpdateTaskRequest{
			Version: version,
			Title:   &newTitle,
		}

		resp, err := svc.UpdateTask(ctx, taskID, req)
		if !errors.Is(err, constants.ErrTaskNotFound) {
			t.Fatalf("expected ErrTaskNotFound, got: %v", err)
		}
		if resp != nil {
			t.Fatalf("expected nil response, got: %+v", resp)
		}
	})

	t.Run("cross_team_task_isolation", func(t *testing.T) {
		gormDB, mock := setupMockDB(t)
		repo := repositories.New(gormDB)
		svc := New(cfg, repo, nil)

		ctx := ctxmeta.WithAuthUser(context.Background(), ctxmeta.AuthUser{
			UserID: creatorID,
			TeamID: teamID,
		})

		taskRows := sqlmock.NewRows([]string{"id", "title", "description", "status", "creator_id", "assignee_id", "team_id", "version", "created_at", "updated_at"}).
			AddRow(taskID, "Title", "Desc", "todo", creatorID, nil, otherTeamID, version, now, now)
		mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "tasks" WHERE id = $1 AND "tasks"."deleted_at" IS NULL ORDER BY "tasks"."id" LIMIT $2`)).
			WithArgs(taskID, 1).
			WillReturnRows(taskRows)

		req := dtos.UpdateTaskRequest{
			Version: version,
			Title:   &newTitle,
		}

		resp, err := svc.UpdateTask(ctx, taskID, req)
		if !errors.Is(err, constants.ErrTaskNotFound) {
			t.Fatalf("expected ErrTaskNotFound on cross-team update, got: %v", err)
		}
		if resp != nil {
			t.Fatalf("expected nil response, got: %+v", resp)
		}
	})

	t.Run("unauthorized_missing_auth_user", func(t *testing.T) {
		gormDB, _ := setupMockDB(t)
		repo := repositories.New(gormDB)
		svc := New(cfg, repo, nil)

		req := dtos.UpdateTaskRequest{
			Version: version,
			Title:   &newTitle,
		}

		resp, err := svc.UpdateTask(context.Background(), taskID, req)
		if err == nil {
			t.Fatal("expected error on missing auth user, got nil")
		}
		if resp != nil {
			t.Fatalf("expected nil response, got: %+v", resp)
		}
	})

	t.Run("assignee_not_found_or_different_team", func(t *testing.T) {
		gormDB, mock := setupMockDB(t)
		repo := repositories.New(gormDB)
		svc := New(cfg, repo, nil)

		ctx := ctxmeta.WithAuthUser(context.Background(), ctxmeta.AuthUser{
			UserID: creatorID,
			TeamID: teamID,
		})

		taskRows := sqlmock.NewRows([]string{"id", "title", "description", "status", "creator_id", "assignee_id", "team_id", "version", "created_at", "updated_at"}).
			AddRow(taskID, "Title", "Desc", "todo", creatorID, nil, teamID, version, now, now)
		mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "tasks" WHERE id = $1 AND "tasks"."deleted_at" IS NULL ORDER BY "tasks"."id" LIMIT $2`)).
			WithArgs(taskID, 1).
			WillReturnRows(taskRows)

		// Assignee in other team
		assigneeRows := sqlmock.NewRows([]string{"id", "name", "email", "team_id", "created_at", "updated_at"}).
			AddRow(newAssigneeID, "Foreign Assignee", "foreign@example.com", otherTeamID, now, now)
		mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "users" WHERE id = $1 ORDER BY "users"."id" LIMIT $2`)).
			WithArgs(newAssigneeID, 1).
			WillReturnRows(assigneeRows)

		req := dtos.UpdateTaskRequest{
			Version:    version,
			AssigneeID: &newAssigneeID,
		}

		resp, err := svc.UpdateTask(ctx, taskID, req)
		if !errors.Is(err, constants.ErrAssigneeNotInTeam) {
			t.Fatalf("expected ErrAssigneeNotInTeam, got: %v", err)
		}
		if resp != nil {
			t.Fatalf("expected nil response, got: %+v", resp)
		}
	})

	t.Run("db_find_task_error", func(t *testing.T) {
		gormDB, mock := setupMockDB(t)
		repo := repositories.New(gormDB)
		svc := New(cfg, repo, nil)

		ctx := ctxmeta.WithAuthUser(context.Background(), ctxmeta.AuthUser{
			UserID: creatorID,
			TeamID: teamID,
		})

		mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "tasks" WHERE id = $1 AND "tasks"."deleted_at" IS NULL ORDER BY "tasks"."id" LIMIT $2`)).
			WithArgs(taskID, 1).
			WillReturnError(errors.New("db find error"))

		req := dtos.UpdateTaskRequest{
			Version: version,
			Title:   &newTitle,
		}

		resp, err := svc.UpdateTask(ctx, taskID, req)
		if err == nil {
			t.Fatal("expected error on find failure, got nil")
		}
		if resp != nil {
			t.Fatalf("expected nil response, got: %+v", resp)
		}
	})

	t.Run("db_update_task_error", func(t *testing.T) {
		gormDB, mock := setupMockDB(t)
		repo := repositories.New(gormDB)
		svc := New(cfg, repo, nil)

		ctx := ctxmeta.WithAuthUser(context.Background(), ctxmeta.AuthUser{
			UserID: creatorID,
			TeamID: teamID,
		})

		taskRows := sqlmock.NewRows([]string{"id", "title", "description", "status", "creator_id", "assignee_id", "team_id", "version", "created_at", "updated_at"}).
			AddRow(taskID, "Title", "Desc", "todo", creatorID, nil, teamID, version, now, now)
		mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "tasks" WHERE id = $1 AND "tasks"."deleted_at" IS NULL ORDER BY "tasks"."id" LIMIT $2`)).
			WithArgs(taskID, 1).
			WillReturnRows(taskRows)

		mock.ExpectBegin()
		mock.ExpectExec(regexp.QuoteMeta(`UPDATE "tasks"`)).
			WillReturnError(errors.New("db save error"))
		mock.ExpectRollback()

		req := dtos.UpdateTaskRequest{
			Version: version,
			Title:   &newTitle,
		}

		resp, err := svc.UpdateTask(ctx, taskID, req)
		if err == nil {
			t.Fatal("expected error on update failure, got nil")
		}
		if resp != nil {
			t.Fatalf("expected nil response, got: %+v", resp)
		}
	})
}

type testNotifier struct {
	notifyErr error
	done      chan struct{}
}

func (m *testNotifier) SendTaskAssignedNotification(ctx context.Context, task *models.Task, assignee *models.User) error {
	if m.done != nil {
		select {
		case m.done <- struct{}{}:
		default:
		}
	}
	return m.notifyErr
}

func TestService_AssignTask(t *testing.T) {
	taskID := uuid.New()
	creatorID := uuid.New()
	oldAssigneeID := uuid.New()
	newAssigneeID := uuid.New()
	teamID := uuid.New()
	otherTeamID := uuid.New()
	logID := uuid.New()
	now := time.Now()
	version := 1
	staleVersion := 2

	cfg := config.Config{}

	t.Run("success_assign_task", func(t *testing.T) {
		gormDB, mock := setupMockDB(t)
		repo := repositories.New(gormDB)
		notif := &testNotifier{done: make(chan struct{}, 1)}
		svc := New(cfg, repo, nil)
		svc.SetNotifier(notif)

		ctx := ctxmeta.WithAuthUser(context.Background(), ctxmeta.AuthUser{
			UserID: creatorID,
			TeamID: teamID,
		})

		// 1. Find task
		taskRows := sqlmock.NewRows([]string{"id", "title", "description", "status", "creator_id", "assignee_id", "team_id", "version", "created_at", "updated_at"}).
			AddRow(taskID, "Task Title", "Desc", "todo", creatorID, &oldAssigneeID, teamID, version, now, now)
		mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "tasks" WHERE id = $1 AND "tasks"."deleted_at" IS NULL ORDER BY "tasks"."id" LIMIT $2`)).
			WithArgs(taskID, 1).
			WillReturnRows(taskRows)

		// 2. Find new assignee in same team
		assigneeRows := sqlmock.NewRows([]string{"id", "name", "email", "team_id", "created_at", "updated_at"}).
			AddRow(newAssigneeID, "Target Assignee", "target@example.com", teamID, now, now)
		mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "users" WHERE id = $1 ORDER BY "users"."id" LIMIT $2`)).
			WithArgs(newAssigneeID, 1).
			WillReturnRows(assigneeRows)

		// 3. Transaction update & log
		mock.ExpectBegin()
		mock.ExpectExec(regexp.QuoteMeta(`UPDATE "tasks"`)).
			WillReturnResult(sqlmock.NewResult(1, 1))
		mock.ExpectQuery(regexp.QuoteMeta(`INSERT INTO "task_logs"`)).
			WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(logID))
		mock.ExpectCommit()

		req := dtos.AssignTaskRequest{
			AssigneeID: newAssigneeID,
			Version:    version,
		}

		resp, err := svc.AssignTask(ctx, taskID, req)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if resp == nil || resp.ID != taskID || *resp.AssigneeID != newAssigneeID || resp.Version != 2 {
			t.Fatalf("unexpected assigned task response: %+v", resp)
		}

		select {
		case <-notif.done:
		case <-time.After(100 * time.Millisecond):
			t.Fatal("expected notification to be dispatched asynchronously")
		}
	})

	t.Run("db_assign_task_error", func(t *testing.T) {
		gormDB, mock := setupMockDB(t)
		repo := repositories.New(gormDB)
		svc := New(cfg, repo, nil)

		ctx := ctxmeta.WithAuthUser(context.Background(), ctxmeta.AuthUser{
			UserID: creatorID,
			TeamID: teamID,
		})

		taskRows := sqlmock.NewRows([]string{"id", "title", "description", "status", "creator_id", "assignee_id", "team_id", "version", "created_at", "updated_at"}).
			AddRow(taskID, "Task Title", "Desc", "todo", creatorID, nil, teamID, version, now, now)
		mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "tasks" WHERE id = $1 AND "tasks"."deleted_at" IS NULL ORDER BY "tasks"."id" LIMIT $2`)).
			WithArgs(taskID, 1).
			WillReturnRows(taskRows)

		assigneeRows := sqlmock.NewRows([]string{"id", "name", "email", "team_id", "created_at", "updated_at"}).
			AddRow(newAssigneeID, "Target Assignee", "target@example.com", teamID, now, now)
		mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "users" WHERE id = $1 ORDER BY "users"."id" LIMIT $2`)).
			WithArgs(newAssigneeID, 1).
			WillReturnRows(assigneeRows)

		mock.ExpectBegin()
		mock.ExpectExec(regexp.QuoteMeta(`UPDATE "tasks"`)).
			WillReturnError(errors.New("db assign error"))
		mock.ExpectRollback()

		req := dtos.AssignTaskRequest{
			AssigneeID: newAssigneeID,
			Version:    version,
		}

		resp, err := svc.AssignTask(ctx, taskID, req)
		if err == nil {
			t.Fatal("expected error on db assign failure, got nil")
		}
		if resp != nil {
			t.Fatalf("expected nil response, got: %+v", resp)
		}
	})

	t.Run("stale_task_version_conflict", func(t *testing.T) {
		gormDB, mock := setupMockDB(t)
		repo := repositories.New(gormDB)
		svc := New(cfg, repo, nil)

		ctx := ctxmeta.WithAuthUser(context.Background(), ctxmeta.AuthUser{
			UserID: creatorID,
			TeamID: teamID,
		})

		taskRows := sqlmock.NewRows([]string{"id", "title", "description", "status", "creator_id", "assignee_id", "team_id", "version", "created_at", "updated_at"}).
			AddRow(taskID, "Title", "Desc", "todo", creatorID, nil, teamID, staleVersion, now, now)
		mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "tasks" WHERE id = $1 AND "tasks"."deleted_at" IS NULL ORDER BY "tasks"."id" LIMIT $2`)).
			WithArgs(taskID, 1).
			WillReturnRows(taskRows)

		req := dtos.AssignTaskRequest{
			AssigneeID: newAssigneeID,
			Version:    version,
		}

		resp, err := svc.AssignTask(ctx, taskID, req)
		if !errors.Is(err, constants.ErrStaleVersion) {
			t.Fatalf("expected ErrStaleVersion, got: %v", err)
		}
		if resp != nil {
			t.Fatalf("expected nil response, got: %+v", resp)
		}
	})

	t.Run("cross_team_task_isolation", func(t *testing.T) {
		gormDB, mock := setupMockDB(t)
		repo := repositories.New(gormDB)
		svc := New(cfg, repo, nil)

		ctx := ctxmeta.WithAuthUser(context.Background(), ctxmeta.AuthUser{
			UserID: creatorID,
			TeamID: teamID,
		})

		taskRows := sqlmock.NewRows([]string{"id", "title", "description", "status", "creator_id", "assignee_id", "team_id", "version", "created_at", "updated_at"}).
			AddRow(taskID, "Other Task", "Desc", "todo", creatorID, nil, otherTeamID, version, now, now)
		mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "tasks" WHERE id = $1 AND "tasks"."deleted_at" IS NULL ORDER BY "tasks"."id" LIMIT $2`)).
			WithArgs(taskID, 1).
			WillReturnRows(taskRows)

		req := dtos.AssignTaskRequest{
			AssigneeID: newAssigneeID,
			Version:    version,
		}

		resp, err := svc.AssignTask(ctx, taskID, req)
		if !errors.Is(err, constants.ErrTaskNotFound) {
			t.Fatalf("expected ErrTaskNotFound, got: %v", err)
		}
		if resp != nil {
			t.Fatalf("expected nil response, got: %+v", resp)
		}
	})

	t.Run("assignee_not_found_or_different_team", func(t *testing.T) {
		gormDB, mock := setupMockDB(t)
		repo := repositories.New(gormDB)
		svc := New(cfg, repo, nil)

		ctx := ctxmeta.WithAuthUser(context.Background(), ctxmeta.AuthUser{
			UserID: creatorID,
			TeamID: teamID,
		})

		taskRows := sqlmock.NewRows([]string{"id", "title", "description", "status", "creator_id", "assignee_id", "team_id", "version", "created_at", "updated_at"}).
			AddRow(taskID, "Task", "Desc", "todo", creatorID, nil, teamID, version, now, now)
		mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "tasks" WHERE id = $1 AND "tasks"."deleted_at" IS NULL ORDER BY "tasks"."id" LIMIT $2`)).
			WithArgs(taskID, 1).
			WillReturnRows(taskRows)

		// Assignee in other team
		assigneeRows := sqlmock.NewRows([]string{"id", "name", "email", "team_id", "created_at", "updated_at"}).
			AddRow(newAssigneeID, "Foreign Assignee", "foreign@example.com", otherTeamID, now, now)
		mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "users" WHERE id = $1 ORDER BY "users"."id" LIMIT $2`)).
			WithArgs(newAssigneeID, 1).
			WillReturnRows(assigneeRows)

		req := dtos.AssignTaskRequest{
			AssigneeID: newAssigneeID,
			Version:    version,
		}

		resp, err := svc.AssignTask(ctx, taskID, req)
		if !errors.Is(err, constants.ErrAssigneeNotInTeam) {
			t.Fatalf("expected ErrAssigneeNotInTeam, got: %v", err)
		}
		if resp != nil {
			t.Fatalf("expected nil response, got: %+v", resp)
		}
	})

	t.Run("unauthorized_missing_auth_user", func(t *testing.T) {
		svc := New(cfg, nil, nil)
		req := dtos.AssignTaskRequest{
			AssigneeID: newAssigneeID,
			Version:    version,
		}

		resp, err := svc.AssignTask(context.Background(), taskID, req)
		if !errors.Is(err, constants.ErrUnauthorized) {
			t.Fatalf("expected ErrUnauthorized, got: %v", err)
		}
		if resp != nil {
			t.Fatalf("expected nil response, got: %+v", resp)
		}
	})
}


