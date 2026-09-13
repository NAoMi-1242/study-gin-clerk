package service

import (
	"context"

	"study-gin-clerk/internal/config"
	"study-gin-clerk/internal/infra/ai"
	"study-gin-clerk/internal/infra/crypto"
	"study-gin-clerk/internal/infra/repository"
	"study-gin-clerk/internal/model"
)

type AIModelService struct {
	keyRepo  *repository.UserAPIKeyRepository
	registry *ai.ModelRegistry
	cache    *ai.MemoryCache
	cfg      config.Config
}

func NewAIModelService(
	keyRepo *repository.UserAPIKeyRepository,
	registry *ai.ModelRegistry,
	cache *ai.MemoryCache,
	cfg config.Config,
) *AIModelService {
	return &AIModelService{
		keyRepo:  keyRepo,
		registry: registry,
		cache:    cache,
		cfg:      cfg,
	}
}

// GetAvailableModels retrieves dynamically discovered models for all active providers of the user, using cache.
func (s *AIModelService) GetAvailableModels(ctx context.Context, userID string, refresh bool) ([]model.AIModelInfo, []model.Provider, error) {
	keys, err := s.keyRepo.ListUserAPIKeys(ctx, userID)
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

		// Cache miss or refresh requested: Decrypt and fetch from provider API
		rawKey, err := crypto.Decrypt(k.EncryptedKey, s.cfg.EncryptionKey)
		if err != nil {
			continue
		}

		models, err := s.registry.FetchModels(ctx, k.Provider, rawKey)
		if err != nil {
			continue
		}

		s.cache.Set(userID, string(k.Provider), models)
		allModels = append(allModels, models...)
	}

	return allModels, activeProviders, nil
}

