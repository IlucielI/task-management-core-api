package dtos

import "github.com/google/uuid"

// ListUsersQuery represents URL query parameters for user listing, filtering, and pagination.
type ListUsersQuery struct {
	Page   int        `form:"page"`
	Limit  int        `form:"limit"`
	Name   string     `form:"name"`
	Email  string     `form:"email"`
	TeamID *uuid.UUID `form:"team_id"`
}

// ListUsersData represents the paginated user list payload containing items and metadata.
type ListUsersData struct {
	Items    []*UserResponse `json:"items"`
	Metadata ListMetadata    `json:"metadata"`
}
