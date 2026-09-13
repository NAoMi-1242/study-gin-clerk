package handler

import "github.com/google/wire"

// Set provides all handlers for dependency injection.
var Set = wire.NewSet(
	NewHealthHandler,
	NewUserProfileHandler,
	NewUserAPIKeyHandler,
	NewAIModelHandler,
	NewChatHandler,
)


