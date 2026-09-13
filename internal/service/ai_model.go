package service

import (
	"context"

	"study-gin-clerk/internal/infra/ai"
	"study-gin-clerk/internal/model"
)

type AIModelService struct {
	userAPIKeyService *UserAPIKeyService
	registry          *ai.ModelRegistry
	cache             *ai.MemoryCache
}

func NewAIModelService(
	userAPIKeyService *UserAPIKeyService,
	registry *ai.ModelRegistry,
	cache *ai.MemoryCache,
) *AIModelService {
	return &AIModelService{
		userAPIKeyService: userAPIKeyService,
		registry:          registry,
		cache:             cache,
	}
}

// GetAvailableModels retrieves dynamically discovered models for all active providers of the user, using cache.
func (s *AIModelService) GetAvailableModels(ctx context.Context, userID string, refresh bool) ([]model.AIModelInfo, []model.Provider, error) {
	keys, err := s.userAPIKeyService.GetDecryptedKeys(ctx, userID)
	if err != nil {
		return nil, nil, err
	}

	var allModels []model.AIModelInfo
	var activeProviders []model.Provider

	for _, k := range keys {
		activeProviders = append(activeProviders, k.Provider)

		if !refresh {
			if cached, found := s.cache.Get(userID, string(k.Provider)); found {
				allModels = append(allModels, cached...)
				continue
			}
		}

		models, err := s.registry.FetchModels(ctx, k.Provider, k.RawKey)
		if err != nil {
			continue
		}

		s.cache.Set(userID, string(k.Provider), models)
		allModels = append(allModels, models...)
	}

	return allModels, activeProviders, nil
}
