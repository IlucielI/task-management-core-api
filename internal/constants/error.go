package constants

import "errors"

var (
	// ErrTeamNotFound is returned when the specified team ID does not exist.
	ErrTeamNotFound = errors.New("team does not exist")

	// ErrEmailAlreadyExists is returned when attempting to register with an already registered email.
	ErrEmailAlreadyExists = errors.New("email is already registered")

	// ErrInvalidCredentials is returned when email or password is invalid.
	ErrInvalidCredentials = errors.New("invalid email or password")

	// ErrUserNotFound is returned when user cannot be found.
	ErrUserNotFound = errors.New("user not found")

	// ErrInvalidToken is returned when a JWT token is expired, malformed, or has an invalid signature.
	ErrInvalidToken = errors.New("invalid or expired token")

	// ErrAssigneeNotInTeam is returned when the specified assignee does not belong to the creator's team.
	ErrAssigneeNotInTeam = errors.New("assignee must belong to the same team")

	// ErrInvalidTaskStatus is returned when an invalid task status code is provided.
	ErrInvalidTaskStatus = errors.New("invalid task status")

	// ErrInvalidIdempotencyKey is returned when the Idempotency-Key header is missing or malformed.
	ErrInvalidIdempotencyKey = errors.New("invalid or missing idempotency key")

	// ErrTaskNotFound is returned when a task cannot be found.
	ErrTaskNotFound = errors.New("task not found")

	// ErrUnauthorized is returned when an operation is attempted without valid authentication.
	ErrUnauthorized = errors.New("unauthorized access")
)

