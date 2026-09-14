package aimodel

import (
	"context"
	"testing"

	"study-gin-clerk/internal/apikey"
	"study-gin-clerk/internal/types"
)

type mockKeyProvider struct {
	keys []apikey.DecryptedKey
}

func (m *mockKeyProvider) GetDecryptedKeys(ctx context.Context, userID string) ([]apikey.DecryptedKey, error) {
	return m.keys, nil
}

type mockModelFetcher struct {
	models map[types.Provider][]types.AIModel
	calls  int
}

func (m *mockModelFetcher) FetchModels(ctx context.Context, providerName types.Provider, apiKey string) ([]types.AIModel, error) {
	m.calls++
	return m.models[providerName], nil
}

type mockModelCache struct {
	items map[string][]types.AIModel
}

func (m *mockModelCache) Get(userID string, provider types.Provider) ([]types.AIModel, bool) {
	item, ok := m.items[userID+":"+string(provider)]
	return item, ok
}

func (m *mockModelCache) Set(userID string, provider types.Provider, models []types.AIModel) {
	m.items[userID+":"+string(provider)] = models
}

func TestAimodelService_GetAvailableModels_WithCache(t *testing.T) {
	keyProvider := &mockKeyProvider{
		keys: []apikey.DecryptedKey{
			{Provider: types.ProviderOpenRouter, RawKey: "test-or-key"},
		},
	}
	fetcher := &mockModelFetcher{
		models: map[types.Provider][]types.AIModel{
			types.ProviderOpenRouter: {
				{ID: "anthropic/claude-3.5-sonnet", Name: "Claude 3.5 Sonnet", Provider: types.ProviderOpenRouter},
			},
		},
	}
	cache := &mockModelCache{items: make(map[string][]types.AIModel)}

	svc := NewService(keyProvider, fetcher, cache)
	ctx := context.Background()

	// 1. First fetch (uncached) -> fetcher called
	models, providers, err := svc.GetAvailableModels(ctx, "user_1", false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(models) != 1 || len(providers) != 1 {
		t.Fatalf("expected 1 model and 1 provider, got %d and %d", len(models), len(providers))
	}
	if fetcher.calls != 1 {
		t.Errorf("expected fetcher calls = 1, got %d", fetcher.calls)
	}

	// 2. Second fetch (cached) -> fetcher should NOT be called again
	modelsCached, _, err := svc.GetAvailableModels(ctx, "user_1", false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(modelsCached) != 1 {
		t.Fatalf("expected 1 cached model, got %d", len(modelsCached))
	}
	if fetcher.calls != 1 {
		t.Errorf("expected fetcher calls to remain 1, got %d", fetcher.calls)
	}

	// 3. Force refresh -> fetcher should be called
	_, _, err = svc.GetAvailableModels(ctx, "user_1", true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if fetcher.calls != 2 {
		t.Errorf("expected fetcher calls = 2 on refresh, got %d", fetcher.calls)
	}
}

