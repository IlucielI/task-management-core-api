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

