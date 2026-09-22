package dtos

import (
	"time"

	"github.com/google/uuid"
)

// TeamFilterQuery contains URL query parameters for filtering teams.
type TeamFilterQuery struct {
	Name string `form:"name"`
}

// TeamResponse represents the response payload for a team.
type TeamResponse struct {
	ID        uuid.UUID `json:"id"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}
