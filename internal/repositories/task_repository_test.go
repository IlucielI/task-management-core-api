package repositories

import (
	"context"
	"errors"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"

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


