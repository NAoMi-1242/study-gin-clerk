package service

import "errors"

var (
	// ErrChatNotFound indicates that the requested chat session does not exist or does not belong to the user.
	ErrChatNotFound = errors.New("chat not found")

	// ErrKeyNotFound indicates that an API key for the requested provider does not exist.
	ErrKeyNotFound = errors.New("API key not found")

	// ErrKeyNotRegistered indicates that the user must register an API key before performing this action.
	ErrKeyNotRegistered = errors.New("API key is not registered")

	// ErrValidationFailed indicates that request inputs or upstream API key validation failed.
	ErrValidationFailed = errors.New("validation failed")

	// ErrAIProvider indicates an upstream error from an external AI service provider.
	ErrAIProvider = errors.New("AI provider error")
)
