package apikey

import "errors"

var (
	// ErrNotFound indicates that an API key for the requested provider does not exist.
	ErrNotFound = errors.New("API key not found")

	// ErrNotRegistered indicates that the user must register an API key before performing this action.
	ErrNotRegistered = errors.New("API key is not registered")

	// ErrValidationFailed indicates that request inputs or upstream API key probe failed.
	ErrValidationFailed = errors.New("validation failed")
)
