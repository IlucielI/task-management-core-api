package validations

import (
	"strings"
	"testing"

	"github.com/google/uuid"

	"task-management/internal/dtos"
)

func TestValidateRegisterRequest(t *testing.T) {
	validReq := dtos.RegisterRequest{
		Name:     "John Doe",
		Email:    "john.doe@example.com",
		Password: "password123",
		TeamID:   uuid.New(),
	}

	// 1. Valid request
	if err := ValidateRegisterRequest(validReq); err != nil {
		t.Fatalf("expected valid registration request, got error: %v", err)
	}

	// 2. Missing Name
	req := validReq
	req.Name = ""
	if err := ValidateRegisterRequest(req); err == nil {
		t.Fatal("expected error on empty name, got nil")
	}

	// 3. Name too long (> 100)
	req = validReq
	req.Name = strings.Repeat("a", 101)
	if err := ValidateRegisterRequest(req); err == nil {
		t.Fatal("expected error on name exceeding 100 characters, got nil")
	}

	// 4. Invalid Email
	req = validReq
	req.Email = "not-an-email"
	if err := ValidateRegisterRequest(req); err == nil {
		t.Fatal("expected error on invalid email, got nil")
	}

	// 5. Password too short (< 8)
	req = validReq
	req.Password = "1234567"
	if err := ValidateRegisterRequest(req); err == nil {
		t.Fatal("expected error on short password, got nil")
	}

	// 6. Password too long (> 72)
	req = validReq
	req.Password = strings.Repeat("a", 73)
	if err := ValidateRegisterRequest(req); err == nil {
		t.Fatal("expected error on long password, got nil")
	}

	// 7. Nil TeamID
	req = validReq
	req.TeamID = uuid.Nil
	if err := ValidateRegisterRequest(req); err == nil {
		t.Fatal("expected error on nil team ID, got nil")
	}
}

func TestValidateLoginRequest(t *testing.T) {
	validReq := dtos.LoginRequest{
		Email:    "john.doe@example.com",
		Password: "password123",
	}

	// 1. Valid request
	if err := ValidateLoginRequest(validReq); err != nil {
		t.Fatalf("expected valid login request, got error: %v", err)
	}

	// 2. Missing Email
	req := validReq
	req.Email = ""
	if err := ValidateLoginRequest(req); err == nil {
		t.Fatal("expected error on empty email, got nil")
	}

	// 3. Invalid Email Format
	req = validReq
	req.Email = "invalid-email"
	if err := ValidateLoginRequest(req); err == nil {
		t.Fatal("expected error on invalid email, got nil")
	}

	// 4. Missing Password
	req = validReq
	req.Password = ""
	if err := ValidateLoginRequest(req); err == nil {
		t.Fatal("expected error on empty password, got nil")
	}

	// 5. Password too short (< 8)
	req = validReq
	req.Password = "short"
	if err := ValidateLoginRequest(req); err == nil {
		t.Fatal("expected error on short password, got nil")
	}

	// 6. Password too long (> 72)
	req = validReq
	req.Password = strings.Repeat("a", 73)
	if err := ValidateLoginRequest(req); err == nil {
		t.Fatal("expected error on long password, got nil")
	}
}

