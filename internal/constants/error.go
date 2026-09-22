package constants

import (
	"net/http"

	"task-management/internal/pkg/apperror"
)

var (
	// ErrTeamNotFound is returned when the specified team ID does not exist during registration.
	ErrTeamNotFound = apperror.New(http.StatusBadRequest, ResponseCodeBadRequest, "team does not exist")

	// ErrEmailAlreadyExists is returned when attempting to register with an already registered email.
	ErrEmailAlreadyExists = apperror.New(http.StatusBadRequest, ResponseCodeBadRequest, "email is already registered")

	// ErrInvalidCredentials is returned when email or password is invalid.
	ErrInvalidCredentials = apperror.New(http.StatusUnauthorized, ResponseCodeUnauthorized, "invalid email or password")

	// ErrUserNotFound is returned when user cannot be found.
	ErrUserNotFound = apperror.New(http.StatusNotFound, ResponseCodeNotFound, "user not found")

	// ErrInvalidToken is returned when a JWT token is expired, malformed, or has an invalid signature.
	ErrInvalidToken = apperror.New(http.StatusUnauthorized, ResponseCodeUnauthorized, "invalid or expired token")

	// ErrAssigneeNotInTeam is returned when the specified assignee does not belong to the creator's team.
	ErrAssigneeNotInTeam = apperror.New(http.StatusBadRequest, ResponseCodeBadRequest, "assignee must belong to the same team")

	// ErrInvalidTaskStatus is returned when an invalid task status code is provided.
	ErrInvalidTaskStatus = apperror.New(http.StatusBadRequest, ResponseCodeBadRequest, "invalid task status")

	// ErrInvalidIdempotencyKey is returned when the Idempotency-Key header is missing or malformed.
	ErrInvalidIdempotencyKey = apperror.New(http.StatusBadRequest, ResponseCodeBadRequest, "invalid or missing idempotency key")

	// ErrTaskNotFound is returned when a task cannot be found.
	ErrTaskNotFound = apperror.New(http.StatusNotFound, ResponseCodeNotFound, "task not found")

	// ErrUnauthorized is returned when an operation is attempted without valid authentication.
	ErrUnauthorized = apperror.New(http.StatusUnauthorized, ResponseCodeUnauthorized, "unauthorized access")

	// ErrBadRequest is returned for generic or wrapped invalid client requests.
	ErrBadRequest = apperror.New(http.StatusBadRequest, ResponseCodeBadRequest, "bad request")

	// ErrInternalServerError is returned when an unexpected system error occurs.
	ErrInternalServerError = apperror.New(http.StatusInternalServerError, ResponseCodeInternalError, "internal server error")
)


