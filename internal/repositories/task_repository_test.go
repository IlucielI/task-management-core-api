package repositories

import (
	"context"
	"errors"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"

	"task-management/internal/constants"
	"task-management/internal/models"
)

func TestRepositories_CreateTaskWithLog(t *testing.T) {
	gormDB, mock := setupMockDB(t)
	repo := New(gormDB)

	taskID := uuid.New()
	creatorID := uuid.New()
	teamID := uuid.New()
	logID := uuid.New()
	now := time.Now()

	task := &models.Task{
		ID:          taskID,
		Title:       "Test Task",
		Description: "Task description",
		Status:      "todo",
		CreatorID:   creatorID,
		TeamID:      teamID,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	log := &models.TaskLog{
		ID:        logID,
		Action:    "CREATE",
		CreatedAt: now,
	}

	// 1. Success with task and log inside transaction
	mock.ExpectBegin()
	mock.ExpectQuery(regexp.QuoteMeta(`INSERT INTO "tasks"`)).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(taskID))
	mock.ExpectQuery(regexp.QuoteMeta(`INSERT INTO "task_logs"`)).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(logID))
	mock.ExpectCommit()

	err := repo.CreateTaskWithLog(context.Background(), task, log)
	if err != nil {
		t.Fatalf("unexpected error creating task with log: %v", err)
	}

	// 2. Task insert fails, transaction rolls back
	mock.ExpectBegin()
	mock.ExpectQuery(regexp.QuoteMeta(`INSERT INTO "tasks"`)).
		WillReturnError(errors.New("db task insert error"))
	mock.ExpectRollback()

	err = repo.CreateTaskWithLog(context.Background(), task, log)
	if err == nil {
		t.Fatal("expected error on task insert failure, got nil")
	}

	// 3. Log insert fails, transaction rolls back
	mock.ExpectBegin()
	mock.ExpectQuery(regexp.QuoteMeta(`INSERT INTO "tasks"`)).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(taskID))
	mock.ExpectQuery(regexp.QuoteMeta(`INSERT INTO "task_logs"`)).
		WillReturnError(errors.New("db log insert error"))
	mock.ExpectRollback()

	err = repo.CreateTaskWithLog(context.Background(), task, log)
	if err == nil {
		t.Fatal("expected error on log insert failure, got nil")
	}

	// 4. Success without log
	mock.ExpectBegin()
	mock.ExpectQuery(regexp.QuoteMeta(`INSERT INTO "tasks"`)).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(taskID))
	mock.ExpectCommit()

	err = repo.CreateTaskWithLog(context.Background(), task, nil)
	if err != nil {
		t.Fatalf("unexpected error creating task without log: %v", err)
	}
}

func TestRepositories_FindTaskByID(t *testing.T) {
	gormDB, mock := setupMockDB(t)
	repo := New(gormDB)

	taskID := uuid.New()
	creatorID := uuid.New()
	teamID := uuid.New()
	now := time.Now()

	// 1. Success - Task Found
	rows := sqlmock.NewRows([]string{"id", "title", "description", "status", "creator_id", "team_id", "created_at", "updated_at"}).
		AddRow(taskID, "Found Task", "Desc", "todo", creatorID, teamID, now, now)
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "tasks" WHERE id = $1 AND "tasks"."deleted_at" IS NULL ORDER BY "tasks"."id" LIMIT $2`)).
		WithArgs(taskID, 1).
		WillReturnRows(rows)

	got, err := repo.FindTaskByID(context.Background(), taskID)
	if err != nil {
		t.Fatalf("unexpected error finding task by ID: %v", err)
	}
	if got == nil || got.ID != taskID || got.Title != "Found Task" {
		t.Fatalf("unexpected task result: %+v", got)
	}

	// 2. Not Found - Returns (nil, nil)
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "tasks" WHERE id = $1 AND "tasks"."deleted_at" IS NULL ORDER BY "tasks"."id" LIMIT $2`)).
		WithArgs(taskID, 1).
		WillReturnRows(sqlmock.NewRows([]string{"id", "title"}))

	got, err = repo.FindTaskByID(context.Background(), taskID)
	if err != nil {
		t.Fatalf("expected nil error on not found, got: %v", err)
	}
	if got != nil {
		t.Fatalf("expected nil task on not found, got: %+v", got)
	}

	// 3. Database Error
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "tasks" WHERE id = $1 AND "tasks"."deleted_at" IS NULL ORDER BY "tasks"."id" LIMIT $2`)).
		WithArgs(taskID, 1).
		WillReturnError(errors.New("db query failure"))

	got, err = repo.FindTaskByID(context.Background(), taskID)
	if err == nil {
		t.Fatal("expected error on db failure, got nil")
	}
	if got != nil {
		t.Fatalf("expected nil task on db error, got: %+v", got)
	}
}

func TestRepositories_DeleteTaskWithLog(t *testing.T) {
	gormDB, mock := setupMockDB(t)
	repo := New(gormDB)

	taskID := uuid.New()
	logID := uuid.New()
	now := time.Now()

	log := &models.TaskLog{
		ID:        logID,
		Action:    "DELETE",
		CreatedAt: now,
	}

	// 1. Success soft delete with audit log
	mock.ExpectBegin()
	mock.ExpectExec(regexp.QuoteMeta(`UPDATE "tasks" SET "deleted_at"=$1 WHERE id = $2 AND "tasks"."deleted_at" IS NULL`)).
		WithArgs(sqlmock.AnyArg(), taskID).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectQuery(regexp.QuoteMeta(`INSERT INTO "task_logs"`)).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(logID))
	mock.ExpectCommit()

	err := repo.DeleteTaskWithLog(context.Background(), taskID, log)
	if err != nil {
		t.Fatalf("unexpected error deleting task with log: %v", err)
	}

	// 2. Task delete fails, transaction rolls back
	mock.ExpectBegin()
	mock.ExpectExec(regexp.QuoteMeta(`UPDATE "tasks" SET "deleted_at"=$1 WHERE id = $2 AND "tasks"."deleted_at" IS NULL`)).
		WithArgs(sqlmock.AnyArg(), taskID).
		WillReturnError(errors.New("db delete error"))
	mock.ExpectRollback()

	err = repo.DeleteTaskWithLog(context.Background(), taskID, log)
	if err == nil {
		t.Fatal("expected error on task delete failure, got nil")
	}

	// 3. Log insert fails, transaction rolls back
	mock.ExpectBegin()
	mock.ExpectExec(regexp.QuoteMeta(`UPDATE "tasks" SET "deleted_at"=$1 WHERE id = $2 AND "tasks"."deleted_at" IS NULL`)).
		WithArgs(sqlmock.AnyArg(), taskID).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectQuery(regexp.QuoteMeta(`INSERT INTO "task_logs"`)).
		WillReturnError(errors.New("db log insert error"))
	mock.ExpectRollback()

	err = repo.DeleteTaskWithLog(context.Background(), taskID, log)
	if err == nil {
		t.Fatal("expected error on log insert failure, got nil")
	}

	// 4. Success without log
	mock.ExpectBegin()
	mock.ExpectExec(regexp.QuoteMeta(`UPDATE "tasks" SET "deleted_at"=$1 WHERE id = $2 AND "tasks"."deleted_at" IS NULL`)).
		WithArgs(sqlmock.AnyArg(), taskID).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	err = repo.DeleteTaskWithLog(context.Background(), taskID, nil)
	if err != nil {
		t.Fatalf("unexpected error deleting task without log: %v", err)
	}
}

func TestRepositories_FindTasks(t *testing.T) {
	gormDB, mock := setupMockDB(t)
	repo := New(gormDB)

	teamID := uuid.New()
	task1ID := uuid.New()
	task2ID := uuid.New()
	now := time.Now()

	t.Run("success_with_all_filters", func(t *testing.T) {
		filter := TaskFilter{
			TeamID: &teamID,
			Status: "in_progress",
			Title:  "bug",
			Offset: 0,
			Limit:  10,
		}

		// 1. Count query
		mock.ExpectQuery(regexp.QuoteMeta(`SELECT count(*) FROM "tasks" WHERE team_id = $1 AND status = $2 AND LOWER(title) LIKE LOWER($3) AND "tasks"."deleted_at" IS NULL`)).
			WithArgs(teamID, "in_progress", "%bug%").
			WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(2))

		// 2. Find query
		rows := sqlmock.NewRows([]string{"id", "title", "description", "status", "team_id", "created_at", "updated_at"}).
			AddRow(task1ID, "Fix login bug", "desc", "in_progress", teamID, now, now).
			AddRow(task2ID, "Fix payment bug", "desc", "in_progress", teamID, now, now)
		mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "tasks" WHERE team_id = $1 AND status = $2 AND LOWER(title) LIKE LOWER($3) AND "tasks"."deleted_at" IS NULL ORDER BY created_at DESC LIMIT $4`)).
			WithArgs(teamID, "in_progress", "%bug%", 10).
			WillReturnRows(rows)

		tasks, total, err := repo.FindTasks(context.Background(), filter)
		if err != nil {
			t.Fatalf("unexpected error finding tasks: %v", err)
		}
		if total != 2 || len(tasks) != 2 {
			t.Fatalf("expected 2 tasks, got total=%d len=%d", total, len(tasks))
		}
	})

	t.Run("success_without_filters", func(t *testing.T) {
		filter := TaskFilter{
			TeamID: &teamID,
			Offset: 10,
			Limit:  10,
		}

		mock.ExpectQuery(regexp.QuoteMeta(`SELECT count(*) FROM "tasks" WHERE team_id = $1 AND "tasks"."deleted_at" IS NULL`)).
			WithArgs(teamID).
			WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(15))

		rows := sqlmock.NewRows([]string{"id", "title", "status", "team_id", "created_at", "updated_at"}).
			AddRow(task1ID, "Task 11", "todo", teamID, now, now)
		mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "tasks" WHERE team_id = $1 AND "tasks"."deleted_at" IS NULL ORDER BY created_at DESC LIMIT $2 OFFSET $3`)).
			WithArgs(teamID, 10, 10).
			WillReturnRows(rows)

		tasks, total, err := repo.FindTasks(context.Background(), filter)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if total != 15 || len(tasks) != 1 {
			t.Fatalf("expected total=15 len=1, got total=%d len=%d", total, len(tasks))
		}
	})

	t.Run("count_db_error", func(t *testing.T) {
		filter := TaskFilter{
			TeamID: &teamID,
			Limit:  10,
		}

		mock.ExpectQuery(regexp.QuoteMeta(`SELECT count(*) FROM "tasks" WHERE team_id = $1 AND "tasks"."deleted_at" IS NULL`)).
			WithArgs(teamID).
			WillReturnError(errors.New("db count error"))

		tasks, total, err := repo.FindTasks(context.Background(), filter)
		if err == nil {
			t.Fatal("expected error on count failure, got nil")
		}
		if tasks != nil || total != 0 {
			t.Fatalf("expected nil tasks and 0 total, got tasks=%v total=%d", tasks, total)
		}
	})

	t.Run("find_db_error", func(t *testing.T) {
		filter := TaskFilter{
			TeamID: &teamID,
			Limit:  10,
		}

		mock.ExpectQuery(regexp.QuoteMeta(`SELECT count(*) FROM "tasks" WHERE team_id = $1 AND "tasks"."deleted_at" IS NULL`)).
			WithArgs(teamID).
			WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(5))

		mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "tasks" WHERE team_id = $1 AND "tasks"."deleted_at" IS NULL ORDER BY created_at DESC LIMIT $2`)).
			WithArgs(teamID, 10).
			WillReturnError(errors.New("db find error"))

		tasks, total, err := repo.FindTasks(context.Background(), filter)
		if err == nil {
			t.Fatal("expected error on find failure, got nil")
		}
		if tasks != nil || total != 0 {
			t.Fatalf("expected nil tasks and 0 total, got tasks=%v total=%d", tasks, total)
		}
	})
}

func TestRepositories_UpdateTaskWithLog(t *testing.T) {
	gormDB, mock := setupMockDB(t)
	repo := New(gormDB)

	taskID := uuid.New()
	creatorID := uuid.New()
	teamID := uuid.New()
	logID := uuid.New()
	now := time.Now()

	task := &models.Task{
		ID:          taskID,
		Title:       "Updated Task Title",
		Description: "Updated Description",
		Status:      "in_progress",
		CreatorID:   creatorID,
		TeamID:      teamID,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	log := &models.TaskLog{
		ID:        logID,
		Action:    "UPDATE",
		CreatedAt: now,
	}

	// 1. Success update with log
	task.Version = 2
	mock.ExpectBegin()
	mock.ExpectExec(regexp.QuoteMeta(`UPDATE "tasks"`)).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectQuery(regexp.QuoteMeta(`INSERT INTO "task_logs"`)).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(logID))
	mock.ExpectCommit()

	err := repo.UpdateTaskWithLog(context.Background(), task, 1, log)
	if err != nil {
		t.Fatalf("unexpected error updating task with log: %v", err)
	}

	// 2. Success update without log
	task.Version = 3
	mock.ExpectBegin()
	mock.ExpectExec(regexp.QuoteMeta(`UPDATE "tasks"`)).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	err = repo.UpdateTaskWithLog(context.Background(), task, 2, nil)
	if err != nil {
		t.Fatalf("unexpected error updating task without log: %v", err)
	}

	// 3. Stale version (0 rows affected) -> returns ErrStaleVersion
	mock.ExpectBegin()
	mock.ExpectExec(regexp.QuoteMeta(`UPDATE "tasks"`)).
		WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectRollback()

	err = repo.UpdateTaskWithLog(context.Background(), task, 2, log)
	if !errors.Is(err, constants.ErrStaleVersion) {
		t.Fatalf("expected ErrStaleVersion, got: %v", err)
	}

	// 4. Task update error rolls back
	mock.ExpectBegin()
	mock.ExpectExec(regexp.QuoteMeta(`UPDATE "tasks"`)).
		WillReturnError(errors.New("db update error"))
	mock.ExpectRollback()

	err = repo.UpdateTaskWithLog(context.Background(), task, 3, log)
	if err == nil {
		t.Fatal("expected error on task update failure, got nil")
	}

	// 5. Log create error rolls back
	mock.ExpectBegin()
	mock.ExpectExec(regexp.QuoteMeta(`UPDATE "tasks"`)).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectQuery(regexp.QuoteMeta(`INSERT INTO "task_logs"`)).
		WillReturnError(errors.New("db log create error"))
	mock.ExpectRollback()

	err = repo.UpdateTaskWithLog(context.Background(), task, 3, log)
	if err == nil {
		t.Fatal("expected error on log create failure, got nil")
	}
}

func TestRepositories_AssignTaskWithLog(t *testing.T) {
	gormDB, mock := setupMockDB(t)
	repo := New(gormDB)

	taskID := uuid.New()
	creatorID := uuid.New()
	assigneeID := uuid.New()
	teamID := uuid.New()
	logID := uuid.New()
	now := time.Now()
	version := 1
	nextVersion := 2

	task := &models.Task{
		ID:          taskID,
		Title:       "Assignment Task",
		Status:      "todo",
		CreatorID:   creatorID,
		AssigneeID:  &assigneeID,
		TeamID:      teamID,
		Version:     nextVersion,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	log := &models.TaskLog{
		ID:        logID,
		TaskID:    taskID,
		Action:    "ASSIGN",
		CreatedAt: now,
	}

	// 1. Success update and log
	mock.ExpectBegin()
	mock.ExpectExec(regexp.QuoteMeta(`UPDATE "tasks"`)).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectQuery(regexp.QuoteMeta(`INSERT INTO "task_logs"`)).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(logID))
	mock.ExpectCommit()

	err := repo.AssignTaskWithLog(context.Background(), task, version, log)
	if err != nil {
		t.Fatalf("unexpected error on assign task: %v", err)
	}

	// 2. Stale version (0 rows affected) rolls back transaction
	mock.ExpectBegin()
	mock.ExpectExec(regexp.QuoteMeta(`UPDATE "tasks"`)).
		WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectRollback()

	err = repo.AssignTaskWithLog(context.Background(), task, version, log)
	if !errors.Is(err, constants.ErrStaleVersion) {
		t.Fatalf("expected ErrStaleVersion, got: %v", err)
	}

	// 3. Task update failure rolls back transaction
	mock.ExpectBegin()
	mock.ExpectExec(regexp.QuoteMeta(`UPDATE "tasks"`)).
		WillReturnError(errors.New("db update error"))
	mock.ExpectRollback()

	err = repo.AssignTaskWithLog(context.Background(), task, version, log)
	if err == nil {
		t.Fatal("expected error on db update failure, got nil")
	}

	// 4. Task log create failure rolls back transaction
	mock.ExpectBegin()
	mock.ExpectExec(regexp.QuoteMeta(`UPDATE "tasks"`)).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectQuery(regexp.QuoteMeta(`INSERT INTO "task_logs"`)).
		WillReturnError(errors.New("db log create error"))
	mock.ExpectRollback()

	err = repo.AssignTaskWithLog(context.Background(), task, version, log)
	if err == nil {
		t.Fatal("expected error on log create failure, got nil")
	}
}


