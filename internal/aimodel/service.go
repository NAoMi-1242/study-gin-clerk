package aimodel

import (
	"context"
	"log/slog"

	"study-gin-clerk/internal/apikey"
	"study-gin-clerk/internal/infra/ai"
	"study-gin-clerk/internal/types"
)

type Service struct {
	apiKeyService *apikey.Service
	registry      *ai.ModelRegistry
	cache         *ai.MemoryCache
}

func NewService(
	apiKeyService *apikey.Service,
	registry *ai.ModelRegistry,
	cache *ai.MemoryCache,
) *Service {
	return &Service{
		apiKeyService: apiKeyService,
		registry:      registry,
		cache:         cache,
	}
}

// GetAvailableModels retrieves dynamically discovered models for all active providers of the user, using cache.
func (s *Service) GetAvailableModels(ctx context.Context, userID string, refresh bool) ([]types.AIModel, []types.Provider, error) {
	keys, err := s.apiKeyService.GetDecryptedKeys(ctx, userID)
	if err != nil {
		return nil, nil, err
	}

	allModels := make([]types.AIModel, 0)
	activeProviders := make([]types.Provider, 0)

	for _, k := range keys {
		activeProviders = append(activeProviders, k.Provider)

		if !refresh {
			if cached, found := s.cache.Get(userID, k.Provider); found {
				allModels = append(allModels, cached...)
				continue
			}
		}

		models, err := s.registry.FetchModels(ctx, k.Provider, k.RawKey)
		if err != nil {
			slog.WarnContext(ctx, "failed to fetch models from provider", "provider", k.Provider, "user_id", userID, "error", err)
			continue
		}

		s.cache.Set(userID, k.Provider, models)
		allModels = append(allModels, models...)
	}

	return allModels, activeProviders, nil
}

