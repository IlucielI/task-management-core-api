package services

import (
	"context"
	"errors"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/go-redis/redismock/v9"
	gormPostgres "gorm.io/driver/postgres"
	"gorm.io/gorm"

	"task-management/internal/adapters/redis"
	"task-management/internal/config"
	"task-management/internal/constants"
	"task-management/internal/repositories"
)

func TestService_CheckHealth_NilService(t *testing.T) {
	var s *Service
	status := s.CheckHealth(context.Background())
	if status.Database != constants.IntegrationStatusDisconnected {
		t.Errorf("expected db disconnected on nil service, got %s", status.Database)
	}
	if status.Redis != constants.IntegrationStatusDisconnected {
		t.Errorf("expected redis disconnected on nil service, got %s", status.Redis)
	}
	if status.S3 != constants.IntegrationStatusDisconnected {
		t.Errorf("expected s3 disconnected on nil service, got %s", status.S3)
	}
}

func TestService_CheckHealth_EmptyDependencies(t *testing.T) {
	s := New(config.Config{}, nil, nil)
	status := s.CheckHealth(context.Background())
	if status.Database != constants.IntegrationStatusDisconnected {
		t.Errorf("expected db disconnected, got %s", status.Database)
	}
	if status.Redis != constants.IntegrationStatusDisconnected {
		t.Errorf("expected redis disconnected, got %s", status.Redis)
	}
	if status.S3 != constants.IntegrationStatusDisconnected {
		t.Errorf("expected s3 disconnected, got %s", status.S3)
	}
}

func TestService_CheckHealth_Connected(t *testing.T) {
	sqlDB, mock, err := sqlmock.New(sqlmock.MonitorPingsOption(true))
	if err != nil {
		t.Fatalf("failed to open sqlmock: %v", err)
	}
	defer sqlDB.Close()

	mock.ExpectPing() // for gorm.Open

	gormDB, err := gorm.Open(gormPostgres.New(gormPostgres.Config{
		Conn: sqlDB,
	}), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to initialize gorm: %v", err)
	}

	rClient, rMock := redismock.NewClientMock()
	rdb := redis.NewWithClient(rClient)
	repo := repositories.New(gormDB, rdb)

	s := New(config.Config{}, repo, nil)

	mock.ExpectPing()
	rMock.ExpectPing().SetVal("PONG")

	status := s.CheckHealth(context.Background())
	if status.Database != constants.IntegrationStatusConnected {
		t.Errorf("expected db connected, got %s", status.Database)
	}
	if status.Redis != constants.IntegrationStatusConnected {
		t.Errorf("expected redis connected, got %s", status.Redis)
	}
	if status.S3 != constants.IntegrationStatusDisconnected {
		t.Errorf("expected s3 disconnected when nil, got %s", status.S3)
	}
}

func TestService_CheckHealth_PingErrors(t *testing.T) {
	sqlDB, mock, err := sqlmock.New(sqlmock.MonitorPingsOption(true))
	if err != nil {
		t.Fatalf("failed to open sqlmock: %v", err)
	}
	defer sqlDB.Close()

	mock.ExpectPing() // for gorm.Open

	gormDB, err := gorm.Open(gormPostgres.New(gormPostgres.Config{
		Conn: sqlDB,
	}), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to initialize gorm: %v", err)
	}

	rClient, rMock := redismock.NewClientMock()
	rdb := redis.NewWithClient(rClient)
	repo := repositories.New(gormDB, rdb)

	s := New(config.Config{}, repo, nil)

	mock.ExpectPing().WillReturnError(errors.New("db ping failed"))
	rMock.ExpectPing().SetErr(errors.New("redis ping failed"))

	status := s.CheckHealth(context.Background())
	if status.Database != constants.IntegrationStatusDisconnected {
		t.Errorf("expected db disconnected on ping error, got %s", status.Database)
	}
	if status.Redis != constants.IntegrationStatusDisconnected {
		t.Errorf("expected redis disconnected on ping error, got %s", status.Redis)
	}
}
