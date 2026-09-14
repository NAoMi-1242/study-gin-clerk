package aimodel

import (
	"context"
	"log/slog"

	"study-gin-clerk/internal/apikey"
	"study-gin-clerk/internal/types"
)

// KeyProvider retrieves user decrypted API keys.
type KeyProvider interface {
	GetDecryptedKeys(ctx context.Context, userID string) ([]apikey.DecryptedKey, error)
}

// ModelFetcher discovers available models from an AI provider.
type ModelFetcher interface {
	FetchModels(ctx context.Context, providerName types.Provider, apiKey string) ([]types.AIModel, error)
}

// ModelCache caches AI models in memory.
type ModelCache interface {
	Get(userID string, provider types.Provider) ([]types.AIModel, bool)
	Set(userID string, provider types.Provider, models []types.AIModel)
}

type Service struct {
	keyProvider KeyProvider
	fetcher     ModelFetcher
	cache       ModelCache
}

func NewService(
	keyProvider KeyProvider,
	fetcher ModelFetcher,
	cache ModelCache,
) *Service {
	return &Service{
		keyProvider: keyProvider,
		fetcher:     fetcher,
		cache:       cache,
	}
}

// GetAvailableModels retrieves dynamically discovered models for all active providers of the user, using cache.
func (s *Service) GetAvailableModels(ctx context.Context, userID string, refresh bool) ([]types.AIModel, []types.Provider, error) {
	keys, err := s.keyProvider.GetDecryptedKeys(ctx, userID)
	if err != nil {
		return nil, nil, err
	}

	allModels := make([]types.AIModel, 0)
	activeProviders := make([]types.Provider, 0)

	for _, k := range keys {
		activeProviders = append(activeProviders, k.Provider)

		if !refresh && s.cache != nil {
			if cached, found := s.cache.Get(userID, k.Provider); found {
				allModels = append(allModels, cached...)
				continue
			}
		}

		models, err := s.fetcher.FetchModels(ctx, k.Provider, k.RawKey)
		if err != nil {
			slog.WarnContext(ctx, "failed to fetch models from provider", "provider", k.Provider, "user_id", userID, "error", err)
			continue
		}

		if s.cache != nil {
			s.cache.Set(userID, k.Provider, models)
		}
		allModels = append(allModels, models...)
	}

	return allModels, activeProviders, nil
}
