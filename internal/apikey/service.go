package apikey

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"study-gin-clerk/internal/types"
)

// KeyValidator verifies whether an API key is valid for the specified AI provider.
type KeyValidator interface {
	ValidateKey(ctx context.Context, provider types.Provider, apiKey string) error
}

// CacheInvalidator handles purging cached data when an API key is modified or deleted.
type CacheInvalidator interface {
	Purge(userID string, provider types.Provider)
}

// Encryptor encrypts and decrypts secret data.
type Encryptor interface {
	Encrypt(plainText string) (string, error)
	Decrypt(cipherTextBase64 string) (string, error)
}

// KeyRepository defines data access methods for user API keys.
type KeyRepository interface {
	Upsert(ctx context.Context, key *Key) error
	GetByProvider(ctx context.Context, userID string, provider types.Provider) (*Key, error)
	ListByUserID(ctx context.Context, userID string) ([]Key, error)
	DeleteByProvider(ctx context.Context, userID string, provider types.Provider) error
}

type Service struct {
	repo      KeyRepository
	validator KeyValidator
	cache     CacheInvalidator
	cipher    Encryptor
}

func NewService(
	repo KeyRepository,
	validator KeyValidator,
	cache CacheInvalidator,
	cipher Encryptor,
) *Service {
	return &Service{
		repo:      repo,
		validator: validator,
		cache:     cache,
		cipher:    cipher,
	}
}

// RegisterKey validates the key with the provider, encrypts it, saves it in DB, and purges any stale cache.
func (s *Service) RegisterKey(ctx context.Context, userID string, provider types.Provider, rawKey string) (*Key, error) {
	rawKey = strings.TrimSpace(rawKey)

	if !provider.IsValid() {
		return nil, fmt.Errorf("%w: unsupported provider '%s'", ErrValidationFailed, provider)
	}
	if rawKey == "" {
		return nil, fmt.Errorf("%w: api_key is required", ErrValidationFailed)
	}

	// 1. Probe the provider API to verify key validity
	if err := s.validator.ValidateKey(ctx, provider, rawKey); err != nil {
		return nil, fmt.Errorf("%w: API key probe failed for '%s': %v", ErrValidationFailed, provider, err)
	}

	// 2. Encrypt the key using AES-256-GCM
	encrypted, err := s.cipher.Encrypt(rawKey)
	if err != nil {
		return nil, fmt.Errorf("failed to encrypt API key: %w", err)
	}

	keyHint := MaskKey(rawKey)

	record := &Key{
		UserID:       userID,
		Provider:     provider,
		EncryptedKey: encrypted,
		KeyHint:      keyHint,
	}

	if err := s.repo.Upsert(ctx, record); err != nil {
		return nil, fmt.Errorf("failed to save API key: %w", err)
	}

	// 3. Immediately invalidate/purge any cached models for this user & provider
	if s.cache != nil {
		s.cache.Purge(userID, provider)
	}

	return record, nil
}

// ListKeys returns all registered API keys for the user (masked hints only).
func (s *Service) ListKeys(ctx context.Context, userID string) ([]Key, error) {
	return s.repo.ListByUserID(ctx, userID)
}

// DeleteKey removes an API key and purges the associated cache.
func (s *Service) DeleteKey(ctx context.Context, userID string, provider types.Provider) error {
	if !provider.IsValid() {
		return fmt.Errorf("%w: unsupported provider '%s'", ErrValidationFailed, provider)
	}
	if err := s.repo.DeleteByProvider(ctx, userID, provider); err != nil {
		return err
	}
	if s.cache != nil {
		s.cache.Purge(userID, provider)
	}
	return nil
}

// GetDecryptedKey retrieves and decrypts the user's API key for the specified provider.
func (s *Service) GetDecryptedKey(ctx context.Context, userID string, provider types.Provider) (string, error) {
	record, err := s.repo.GetByProvider(ctx, userID, provider)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return "", fmt.Errorf("%w: provider '%s'", ErrNotRegistered, provider)
		}
		return "", err
	}

	decrypted, err := s.cipher.Decrypt(record.EncryptedKey)
	if err != nil {
		return "", fmt.Errorf("failed to decrypt API key: %w", err)
	}

	return decrypted, nil
}

// DecryptedKey represents a user's decrypted API key for a specific provider.
type DecryptedKey struct {
	Provider types.Provider
	RawKey   string
}

// GetDecryptedKeys retrieves and decrypts all registered API keys for the user.
func (s *Service) GetDecryptedKeys(ctx context.Context, userID string) ([]DecryptedKey, error) {
	records, err := s.repo.ListByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}

	var keys []DecryptedKey
	for _, record := range records {
		rawKey, err := s.cipher.Decrypt(record.EncryptedKey)
		if err != nil {
			continue
		}
		keys = append(keys, DecryptedKey{
			Provider: record.Provider,
			RawKey:   rawKey,
		})
	}

	return keys, nil
}
