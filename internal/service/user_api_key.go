package service

import (
	"context"
	"fmt"
	"strings"

	"study-gin-clerk/internal/ai"
	"study-gin-clerk/internal/config"
	"study-gin-clerk/internal/crypto"
	"study-gin-clerk/internal/model"
	"study-gin-clerk/internal/repository"
)

type UserAPIKeyService struct {
	keyRepo  *repository.UserAPIKeyRepository
	registry *ai.ModelRegistry
	cache    *ai.MemoryCache
	cfg      config.Config
}

func NewUserAPIKeyService(
	keyRepo *repository.UserAPIKeyRepository,
	registry *ai.ModelRegistry,
	cache *ai.MemoryCache,
	cfg config.Config,
) *UserAPIKeyService {
	return &UserAPIKeyService{
		keyRepo:  keyRepo,
		registry: registry,
		cache:    cache,
		cfg:      cfg,
	}
}

// RegisterKey validates the key with the provider, encrypts it, saves it in DB, and purges any stale cache.
func (s *UserAPIKeyService) RegisterKey(ctx context.Context, userID, providerName, apiKey string) (*model.UserAPIKey, error) {
	providerName = strings.ToLower(strings.TrimSpace(providerName))
	apiKey = strings.TrimSpace(apiKey)

	if providerName == "" {
		return nil, fmt.Errorf("provider is required")
	}
	if apiKey == "" {
		return nil, fmt.Errorf("api_key is required")
	}

	// 1. Probe the provider API to verify key validity
	if err := s.registry.ValidateKey(ctx, providerName, apiKey); err != nil {
		return nil, fmt.Errorf("API key validation failed for '%s': %w", providerName, err)
	}

	// 2. Encrypt the key using AES-256-GCM
	encrypted, err := crypto.Encrypt(apiKey, s.cfg.EncryptionKey)
	if err != nil {
		return nil, fmt.Errorf("failed to encrypt API key: %w", err)
	}

	keyHint := crypto.MaskAPIKey(apiKey)

	record := &model.UserAPIKey{
		UserID:       userID,
		Provider:     providerName,
		EncryptedKey: encrypted,
		KeyHint:      keyHint,
	}

	if err := s.keyRepo.UpsertUserAPIKey(ctx, record); err != nil {
		return nil, fmt.Errorf("failed to save API key: %w", err)
	}

	// 3. Immediately invalidate/purge any cached models for this user & provider
	s.cache.Purge(userID, providerName)

	return record, nil
}

// ListKeys returns all registered API keys for the user (masked hints only).
func (s *UserAPIKeyService) ListKeys(ctx context.Context, userID string) ([]model.UserAPIKey, error) {
	return s.keyRepo.ListUserAPIKeys(ctx, userID)
}

// DeleteKey removes an API key and purges the associated cache.
func (s *UserAPIKeyService) DeleteKey(ctx context.Context, userID, providerName string) error {
	providerName = strings.ToLower(strings.TrimSpace(providerName))
	if err := s.keyRepo.DeleteUserAPIKey(ctx, userID, providerName); err != nil {
		return err
	}
	s.cache.Purge(userID, providerName)
	return nil
}

// GetDecryptedKey retrieves and decrypts the user's API key for the specified provider.
func (s *UserAPIKeyService) GetDecryptedKey(ctx context.Context, userID, providerName string) (string, error) {
	providerName = strings.ToLower(strings.TrimSpace(providerName))
	record, err := s.keyRepo.GetUserAPIKey(ctx, userID, providerName)
	if err != nil {
		return "", err
	}
	if record == nil {
		return "", fmt.Errorf("API key for provider '%s' is not registered. Please register it in API Key Settings", providerName)
	}

	decrypted, err := crypto.Decrypt(record.EncryptedKey, s.cfg.EncryptionKey)
	if err != nil {
		return "", fmt.Errorf("failed to decrypt API key: %w", err)
	}

	return decrypted, nil
}

// GetAvailableModels retrieves dynamically discovered models for all active providers of the user, using cache.
func (s *UserAPIKeyService) GetAvailableModels(ctx context.Context, userID string, refresh bool) ([]ai.ModelInfo, []string, error) {
	keys, err := s.keyRepo.ListUserAPIKeys(ctx, userID)
	if err != nil {
		return nil, nil, err
	}

	var allModels []ai.ModelInfo
	var activeProviders []string

	for _, k := range keys {
		activeProviders = append(activeProviders, k.Provider)

		if !refresh {
			if cached, found := s.cache.Get(userID, k.Provider); found {
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

		s.cache.Set(userID, k.Provider, models)
		allModels = append(allModels, models...)
	}

	return allModels, activeProviders, nil
}

