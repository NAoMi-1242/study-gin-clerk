package user

import "errors"

var (
	// ErrNotFound is returned when a requested user resource does not exist.
	ErrNotFound = errors.New("user resource not found")

	// ErrValidationFailed is returned when an API key or profile validation fails.
	ErrValidationFailed = errors.New("validation failed")

	// ErrNotRegistered is returned when an API key is required but not configured by the user.
	ErrNotRegistered = errors.New("api key not registered")
)

