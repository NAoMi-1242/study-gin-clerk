package service

import (
	"context"
	"fmt"
	"strings"

	"study-gin-clerk/internal/config"
	"study-gin-clerk/internal/infra/ai"
	"study-gin-clerk/internal/infra/crypto"
	"study-gin-clerk/internal/infra/repository"
	"study-gin-clerk/internal/model"
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
func (s *UserAPIKeyService) RegisterKey(ctx context.Context, userID string, provider model.Provider, apiKey string) (*model.UserAPIKey, error) {
	apiKey = strings.TrimSpace(apiKey)

	if !provider.IsValid() {
		return nil, fmt.Errorf("unsupported provider: '%s'", provider)
	}
	if apiKey == "" {
		return nil, fmt.Errorf("api_key is required")
	}

	// 1. Probe the provider API to verify key validity
	if err := s.registry.ValidateKey(ctx, provider, apiKey); err != nil {
		return nil, fmt.Errorf("API key validation failed for '%s': %w", provider, err)
	}

	// 2. Encrypt the key using AES-256-GCM
	encrypted, err := crypto.Encrypt(apiKey, s.cfg.EncryptionKey)
	if err != nil {
		return nil, fmt.Errorf("failed to encrypt API key: %w", err)
	}

	keyHint := model.MaskAPIKey(apiKey)

	record := &model.UserAPIKey{
		UserID:       userID,
		Provider:     provider,
		EncryptedKey: encrypted,
		KeyHint:      keyHint,
	}

	if err := s.keyRepo.Upsert(ctx, record); err != nil {
		return nil, fmt.Errorf("failed to save API key: %w", err)
	}

	// 3. Immediately invalidate/purge any cached models for this user & provider
	s.cache.Purge(userID, string(provider))

	return record, nil
}

// ListKeys returns all registered API keys for the user (masked hints only).
func (s *UserAPIKeyService) ListKeys(ctx context.Context, userID string) ([]model.UserAPIKey, error) {
	return s.keyRepo.ListByUserID(ctx, userID)
}

// DeleteKey removes an API key and purges the associated cache.
func (s *UserAPIKeyService) DeleteKey(ctx context.Context, userID string, provider model.Provider) error {
	if !provider.IsValid() {
		return fmt.Errorf("unsupported provider: '%s'", provider)
	}
	if err := s.keyRepo.DeleteByProvider(ctx, userID, provider); err != nil {
		return err
	}
	s.cache.Purge(userID, string(provider))
	return nil
}

// GetDecryptedKey retrieves and decrypts the user's API key for the specified provider.
func (s *UserAPIKeyService) GetDecryptedKey(ctx context.Context, userID string, provider model.Provider) (string, error) {
	record, err := s.keyRepo.GetByProvider(ctx, userID, provider)
	if err != nil {
		return "", err
	}
	if record == nil {
		return "", fmt.Errorf("API key for provider '%s' is not registered. Please register it in API Key Settings", provider)
	}

	decrypted, err := crypto.Decrypt(record.EncryptedKey, s.cfg.EncryptionKey)
	if err != nil {
		return "", fmt.Errorf("failed to decrypt API key: %w", err)
	}

	return decrypted, nil
}

// DecryptedAPIKey represents a user's decrypted API key for a specific provider.
type DecryptedAPIKey struct {
	Provider model.Provider
	RawKey   string
}

// GetDecryptedKeys retrieves and decrypts all registered API keys for the user.
func (s *UserAPIKeyService) GetDecryptedKeys(ctx context.Context, userID string) ([]DecryptedAPIKey, error) {
	records, err := s.keyRepo.ListByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}

	var keys []DecryptedAPIKey
	for _, record := range records {
		rawKey, err := crypto.Decrypt(record.EncryptedKey, s.cfg.EncryptionKey)
		if err != nil {
			continue
		}
		keys = append(keys, DecryptedAPIKey{
			Provider: record.Provider,
			RawKey:   rawKey,
		})
	}

	return keys, nil
}

