package chat

import (
	"github.com/google/wire"

	"study-gin-clerk/internal/infra/ai"
)

// ProvideService adapts concrete dependencies to chat.Service.
func ProvideService(
	repo *Repository,
	aiClient *ai.Client,
	registry *ai.ModelRegistry,
	cache *ai.MemoryCache,
) *Service {
	return NewService(repo, aiClient, registry, cache)
}

var Set = wire.NewSet(
	NewRepository,
	ProvideService,
	NewHandler,
)
