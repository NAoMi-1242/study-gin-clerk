package ai

import "github.com/google/wire"

// Set provides all AI components for dependency injection.
var Set = wire.NewSet(
	NewMemoryCacheDefault,
	NewModelRegistry,
	NewClient,
)
