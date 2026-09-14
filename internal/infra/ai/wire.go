package ai

import "github.com/google/wire"

// Set provides all AI components for dependency injection.
var Set = wire.NewSet(
	NewMemoryCacheDefault,
	wire.Bind(new(ModelCacher), new(*MemoryCache)),
	NewModelRegistry,
	wire.Bind(new(ModelFetcher), new(*ModelRegistry)),
	NewClient,
	wire.Bind(new(ChatClient), new(*Client)),
)
