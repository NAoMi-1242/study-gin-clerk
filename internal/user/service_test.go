package user

import (
	"context"
	"errors"
	"testing"

	"study-gin-clerk/internal/types"
)

// Mock implementations for user.Service testing

type mockUserRepository struct {
	profiles map[string]*Profile
	keys     map[string]*Key // key = userID:provider
}

func newMockUserRepository() *mockUserRepository {
	return &mockUserRepository{
		profiles: make(map[string]*Profile),
		keys:     make(map[string]*Key),
	}
}

func (m *mockUserRepository) GetProfile(ctx context.Context, userID string) (*Profile, error) {
	p, ok := m.profiles[userID]
	if !ok {
		return nil, ErrNotFound
	}
	return p, nil
}

func (m *mockUserRepository) UpsertProfile(ctx context.Context, p *Profile) error {
	m.profiles[p.UserID] = p
	return nil
}

func (m *mockUserRepository) UpsertKey(ctx context.Context, key *Key) error {
	id := key.UserID + ":" + string(key.Provider)
	m.keys[id] = key
	return nil
}

func (m *mockUserRepository) GetKeyByProvider(ctx context.Context, userID string, provider types.Provider) (*Key, error) {
	id := userID + ":" + string(provider)
	k, ok := m.keys[id]
	if !ok {
		return nil, ErrNotFound
	}
	return k, nil
}

func (m *mockUserRepository) ListKeysByUserID(ctx context.Context, userID string) ([]Key, error) {
	var result []Key
	for _, k := range m.keys {
		if k.UserID == userID {
			result = append(result, *k)
		}
	}
	return result, nil
}

func (m *mockUserRepository) DeleteKeyByProvider(ctx context.Context, userID string, provider types.Provider) error {
	id := userID + ":" + string(provider)
	if _, ok := m.keys[id]; !ok {
		return ErrNotFound
	}
	delete(m.keys, id)
	return nil
}

type mockValidator struct {
	valid bool
	err   error
}

func (m *mockValidator) ValidateKey(ctx context.Context, provider types.Provider, apiKey string) error {
	if m.err != nil {
		return m.err
	}
	if !m.valid {
		return errors.New("probe failed")
	}
	return nil
}

type mockCacheInvalidator struct {
	purgedKeys []string
}

func (m *mockCacheInvalidator) Purge(userID string, provider types.Provider) {
	m.purgedKeys = append(m.purgedKeys, userID+":"+string(provider))
}

type mockCipher struct {
	prefix string
}

func (m *mockCipher) Encrypt(plainText string) (string, error) {
	return m.prefix + plainText, nil
}

func (m *mockCipher) Decrypt(cipherTextBase64 string) (string, error) {
	if len(cipherTextBase64) < len(m.prefix) {
		return "", errors.New("invalid ciphertext")
	}
	return cipherTextBase64[len(m.prefix):], nil
}

func TestService_GetProfile_Default(t *testing.T) {
	repo := newMockUserRepository()
	svc := NewService(repo, nil, nil, nil)

	p, err := svc.GetProfile(context.Background(), "user_new")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if p.UserID != "user_new" || p.SystemPrompt != "" {
		t.Errorf("unexpected profile: %+v", p)
	}
}

func TestService_UpdateSystemPrompt(t *testing.T) {
	repo := newMockUserRepository()
	svc := NewService(repo, nil, nil, nil)

	ctx := context.Background()
	p, err := svc.UpdateSystemPrompt(ctx, "user_1", "You are a helpful assistant.")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if p.SystemPrompt != "You are a helpful assistant." {
		t.Errorf("got %q, want 'You are a helpful assistant.'", p.SystemPrompt)
	}

	retrieved, err := svc.GetProfile(ctx, "user_1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if retrieved.SystemPrompt != "You are a helpful assistant." {
		t.Errorf("got %q, want 'You are a helpful assistant.'", retrieved.SystemPrompt)
	}
}

func TestService_RegisterKey_Success(t *testing.T) {
	repo := newMockUserRepository()
	validator := &mockValidator{valid: true}
	cache := &mockCacheInvalidator{}
	cipher := &mockCipher{prefix: "enc:"}

	svc := NewService(repo, validator, cache, cipher)

	ctx := context.Background()
	key, err := svc.RegisterKey(ctx, "user_1", types.ProviderOpenRouter, "sk-or-v1-testkey1234")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if key.UserID != "user_1" || key.Provider != types.ProviderOpenRouter {
		t.Errorf("unexpected key fields: %+v", key)
	}
	if key.EncryptedKey != "enc:sk-or-v1-testkey1234" {
		t.Errorf("got encrypted key %q, want enc:sk-or-v1-testkey1234", key.EncryptedKey)
	}
	if len(cache.purgedKeys) != 1 || cache.purgedKeys[0] != "user_1:openrouter" {
		t.Errorf("expected cache to be purged, got %v", cache.purgedKeys)
	}
}

func TestService_RegisterKey_ValidationFailed(t *testing.T) {
	repo := newMockUserRepository()
	validator := &mockValidator{valid: false}
	cache := &mockCacheInvalidator{}
	cipher := &mockCipher{prefix: "enc:"}

	svc := NewService(repo, validator, cache, cipher)

	ctx := context.Background()
	_, err := svc.RegisterKey(ctx, "user_1", types.ProviderOpenRouter, "invalid-key")
	if err == nil {
		t.Fatalf("expected validation error, got nil")
	}
	if !errors.Is(err, ErrValidationFailed) {
		t.Errorf("expected ErrValidationFailed, got %v", err)
	}
}

func TestService_GetDecryptedKey(t *testing.T) {
	repo := newMockUserRepository()
	validator := &mockValidator{valid: true}
	cache := &mockCacheInvalidator{}
	cipher := &mockCipher{prefix: "enc:"}

	svc := NewService(repo, validator, cache, cipher)
	ctx := context.Background()

	// 1. Not registered
	_, err := svc.GetDecryptedKey(ctx, "user_1", types.ProviderOpenRouter)
	if err == nil || !errors.Is(err, ErrNotRegistered) {
		t.Fatalf("expected ErrNotRegistered, got %v", err)
	}

	// 2. Register and then retrieve
	_, err = svc.RegisterKey(ctx, "user_1", types.ProviderOpenRouter, "my-secret-key")
	if err != nil {
		t.Fatalf("register failed: %v", err)
	}

	decrypted, err := svc.GetDecryptedKey(ctx, "user_1", types.ProviderOpenRouter)
	if err != nil {
		t.Fatalf("get decrypted key failed: %v", err)
	}
	if decrypted != "my-secret-key" {
		t.Errorf("got %q, want my-secret-key", decrypted)
	}
}

func TestService_DeleteKey(t *testing.T) {
	repo := newMockUserRepository()
	validator := &mockValidator{valid: true}
	cache := &mockCacheInvalidator{}
	cipher := &mockCipher{prefix: "enc:"}

	svc := NewService(repo, validator, cache, cipher)
	ctx := context.Background()

	_, _ = svc.RegisterKey(ctx, "user_1", types.ProviderOpenAI, "sk-123456789")

	err := svc.DeleteKey(ctx, "user_1", types.ProviderOpenAI)
	if err != nil {
		t.Fatalf("delete key failed: %v", err)
	}

	_, err = svc.GetDecryptedKey(ctx, "user_1", types.ProviderOpenAI)
	if !errors.Is(err, ErrNotRegistered) {
		t.Errorf("expected key to be deleted and return ErrNotRegistered, got %v", err)
	}
}

func TestMaskKey(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"1234", "****"},
		{"sk-or-v1-abcdef1234", "sk-or-v1-...1234"},
		{"sk-123456789", "sk-...6789"},
		{"AIzaSyCustomKeyGoogle1234", "AIzaSy...1234"},
		{"abcdefghij", "ab...ij"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := MaskKey(tt.input)
			if got != tt.want {
				t.Errorf("MaskKey(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

