package s3_test

import (
	"context"
	"strings"
	"testing"

	"task-management/internal/adapters/s3"
	"task-management/internal/config"
)

func TestS3_New_ValidationErrors(t *testing.T) {
	// Empty endpoint
	_, err := s3.New(config.Config{
		S3Endpoint:  "",
		S3AccessKey: "key",
		S3SecretKey: "secret",
	})
	if err == nil {
		t.Error("expected error for empty endpoint, got nil")
	}

	// Empty access key
	_, err = s3.New(config.Config{
		S3Endpoint:  "localhost:9000",
		S3AccessKey: "",
		S3SecretKey: "secret",
	})
	if err == nil {
		t.Error("expected error for empty access key, got nil")
	}

	// Empty secret key
	_, err = s3.New(config.Config{
		S3Endpoint:  "localhost:9000",
		S3AccessKey: "key",
		S3SecretKey: "",
	})
	if err == nil {
		t.Error("expected error for empty secret key, got nil")
	}
}

func TestS3_New_UnreachableEndpoint(t *testing.T) {
	cfg := config.Config{
		S3Endpoint:   "127.0.0.1:59998",
		S3AccessKey:  "minioadmin",
		S3SecretKey:  "minioadmin",
		S3BucketName: "test-bucket",
	}

	client, err := s3.New(cfg)
	if err == nil {
		t.Fatal("expected error connecting to unreachable S3 endpoint, got nil")
	}
	if client != nil {
		t.Fatal("expected client to be nil on failed connection")
	}
}

func TestS3_NilReceiver(t *testing.T) {
	var s *s3.S3
	ctx := context.Background()

	if s.Client() != nil {
		t.Error("expected nil *minio.Client from nil S3")
	}

	if s.BucketName() != "" {
		t.Error("expected empty bucket name from nil S3")
	}

	if err := s.Ping(ctx); err == nil {
		t.Error("expected error calling Ping on nil S3, got nil")
	}

	if err := s.EnsureBucketExists(ctx, "bucket"); err == nil {
		t.Error("expected error calling EnsureBucketExists on nil S3, got nil")
	}

	if err := s.Upload(ctx, "bucket", "obj", strings.NewReader("data"), 4, "text/plain"); err == nil {
		t.Error("expected error calling Upload on nil S3, got nil")
	}

	if _, err := s.Download(ctx, "bucket", "obj"); err == nil {
		t.Error("expected error calling Download on nil S3, got nil")
	}

	if err := s.Delete(ctx, "bucket", "obj"); err == nil {
		t.Error("expected error calling Delete on nil S3, got nil")
	}

	if _, err := s.PresignGetObject(ctx, "bucket", "obj", 0); err == nil {
		t.Error("expected error calling PresignGetObject on nil S3, got nil")
	}

	if _, err := s.PresignPutObject(ctx, "bucket", "obj", 0); err == nil {
		t.Error("expected error calling PresignPutObject on nil S3, got nil")
	}
}
