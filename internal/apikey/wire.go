package apikey

import (
	"github.com/google/wire"

	"study-gin-clerk/internal/infra/ai"
	"study-gin-clerk/internal/infra/crypto"
)

// ProvideService creates an apikey.Service by adapting concrete dependencies.
func ProvideService(
	repo *Repository,
	registry *ai.ModelRegistry,
	cache *ai.MemoryCache,
	cipher *crypto.AESCipher,
) *Service {
	return NewService(repo, registry, cache, cipher)
}

var Set = wire.NewSet(
	NewRepository,
	ProvideService,
	NewHandler,
)
