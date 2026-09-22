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
