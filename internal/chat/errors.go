package chat

import "errors"

var (
	// ErrNotFound indicates that the requested chat session does not exist or does not belong to the user.
	ErrNotFound = errors.New("chat not found")

	// ErrValidationFailed indicates that request inputs failed validation.
	ErrValidationFailed = errors.New("validation failed")

	// ErrAIProvider indicates an upstream error from an external AI service provider.
	ErrAIProvider = errors.New("AI provider error")
)
