package repositories

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/go-redis/redismock/v9"
	"github.com/google/uuid"

	"task-management/internal/adapters/redis"
	"task-management/internal/dtos"
)

func TestRepositories_Session_WithMockRedis(t *testing.T) {
	client, mock := redismock.NewClientMock()
	rdb := redis.NewWithClient(client)
	repo := New(nil, rdb)
	ctx := context.Background()

	userID := uuid.New()
	sessionID := uuid.New().String()
	now := time.Now()

	session := dtos.SessionData{
		SessionID:      sessionID,
		UserID:         userID,
		Name:           "Test User",
		Email:          "test@example.com",
		TeamID:         uuid.New(),
		CreatedAt:      now,
		LastActivityAt: now,
		ExpiresAt:      now.Add(24 * time.Hour),
	}

	sessionKey := fmt.Sprintf("auth:session:%s", sessionID)
	sessionBytes, _ := json.Marshal(session)

	// 1. SaveSession
	mock.ExpectSet(sessionKey, sessionBytes, 24*time.Hour).SetVal("OK")
	if err := repo.SaveSession(ctx, session, 24*time.Hour); err != nil {
		t.Fatalf("unexpected error on SaveSession: %v", err)
	}

	// 2. GetSession
	mock.ExpectGet(sessionKey).SetVal(string(sessionBytes))
	got, err := repo.GetSession(ctx, sessionID)
	if err != nil {
		t.Fatalf("unexpected error on GetSession: %v", err)
	}
	if got == nil || got.SessionID != sessionID {
		t.Fatalf("unexpected session data: %+v", got)
	}

	// 3. DeleteSession
	mock.ExpectDel(sessionKey).SetVal(1)
	if err := repo.DeleteSession(ctx, sessionID); err != nil {
		t.Fatalf("unexpected error on DeleteSession: %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet redis expectations: %v", err)
	}
}
