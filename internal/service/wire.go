package service

import "github.com/google/wire"

// Set provides all services for dependency injection.
var Set = wire.NewSet(
	NewUserService,
	NewChatService,
	NewUserAPIKeyService,
)


