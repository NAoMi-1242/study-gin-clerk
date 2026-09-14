package aimodel

import (
	"github.com/google/wire"

	"study-gin-clerk/internal/apikey"
	"study-gin-clerk/internal/infra/ai"
)

// ProvideService adapts concrete implementations for aimodel.Service.
func ProvideService(
	apiKeyService *apikey.Service,
	registry *ai.ModelRegistry,
	cache *ai.MemoryCache,
) *Service {
	return NewService(apiKeyService, registry, cache)
}

var Set = wire.NewSet(
	ProvideService,
	NewHandler,
)
