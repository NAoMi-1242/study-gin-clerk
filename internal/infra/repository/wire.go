package repository

import "github.com/google/wire"

// Set provides all repositories for dependency injection.
var Set = wire.NewSet(
	NewChatRepository,
	NewUserAPIKeyRepository,
	NewUserProfileRepository,
)
