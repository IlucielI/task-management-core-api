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
)

