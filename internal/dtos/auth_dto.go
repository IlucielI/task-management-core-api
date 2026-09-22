package dtos

import (
	"time"

	"github.com/google/uuid"
)

// RegisterRequest represents payload data for new user registration.
type RegisterRequest struct {
	Name     string    `json:"name"`
	Email    string    `json:"email"`
	Password string    `json:"password"`
	TeamID   uuid.UUID `json:"team_id"`
}

// UserResponse represents payload data returned for a user profile.
type UserResponse struct {
	ID        uuid.UUID `json:"id"`
	Name      string    `json:"name"`
	Email     string    `json:"email"`
	TeamID    uuid.UUID `json:"team_id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// LoginRequest represents payload data for user login.
type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// TokenResponse contains JWT access and refresh token pair with expiration details.
type TokenResponse struct {
	AccessToken      string `json:"access_token"`
	RefreshToken     string `json:"refresh_token"`
	TokenType        string `json:"token_type"`
	ExpiresIn        int64  `json:"expires_in"`
	RefreshExpiresIn int64  `json:"refresh_expires_in"`
}

// LoginResponse represents token pair and user profile returned on successful login.
type LoginResponse struct {
	Tokens TokenResponse `json:"tokens"`
	User   UserResponse  `json:"user"`
}

// SessionData represents stateful session information stored in Redis.
type SessionData struct {
	SessionID      string    `json:"session_id"`
	UserID         uuid.UUID `json:"user_id"`
	Name           string    `json:"name"`
	Email          string    `json:"email"`
	TeamID         uuid.UUID `json:"team_id"`
	ClientIP       string    `json:"client_ip"`
	UserAgent      string    `json:"user_agent"`
	CreatedAt      time.Time `json:"created_at"`
	LastActivityAt time.Time `json:"last_activity_at"`
	ExpiresAt      time.Time `json:"expires_at"`
}


