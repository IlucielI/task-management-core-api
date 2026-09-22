package dtos

import (
	"time"

	"github.com/google/uuid"

	"task-management/internal/constants"
)

// ListTeamsQuery contains URL query parameters for filtering, sorting, and paginating teams.
type ListTeamsQuery struct {
	Page    int                 `form:"page"`
	Limit   int                 `form:"limit"`
	Name    string              `form:"name"`
	OrderBy constants.SortOrder `form:"order_by"`
}

// ListTeamsData represents the paginated team list payload containing items and pagination metadata.
type ListTeamsData struct {
	Items    []TeamResponse `json:"items"`
	Metadata ListMetadata   `json:"metadata"`
}

// TeamResponse represents the response payload for a team.
type TeamResponse struct {
	ID        uuid.UUID `json:"id"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}
