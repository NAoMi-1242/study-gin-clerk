package chat

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/zendev-sh/goai"

	"study-gin-clerk/internal/infra/ai"
	"study-gin-clerk/internal/types"
	"study-gin-clerk/internal/user"
)

// Mock implementations for chat.Service testing

type mockChatRepository struct {
	chats    map[uint]*Chat
	messages map[uint][]Message
	nextID   uint
}

func newMockChatRepository() *mockChatRepository {
	return &mockChatRepository{
		chats:    make(map[uint]*Chat),
		messages: make(map[uint][]Message),
		nextID:   1,
	}
}

func (m *mockChatRepository) Create(ctx context.Context, userID, title string) (*Chat, error) {
	c := &Chat{
		ID:        m.nextID,
		UserID:    userID,
		Title:     title,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	m.chats[c.ID] = c
	m.nextID++
	return c, nil
}

func (m *mockChatRepository) ListByUserID(ctx context.Context, userID string) ([]Chat, error) {
	var list []Chat
	for _, c := range m.chats {
		if c.UserID == userID {
			list = append(list, *c)
		}
	}
	return list, nil
}

func (m *mockChatRepository) GetWithMessages(ctx context.Context, chatID uint, userID string) (*Chat, error) {
	c, ok := m.chats[chatID]
	if !ok || c.UserID != userID {
		return nil, ErrNotFound
	}
	res := *c
	res.Messages = m.messages[chatID]
	return &res, nil
}

func (m *mockChatRepository) CreateMessage(ctx context.Context, chatID uint, role Role, content string) (*Message, error) {
	msg := Message{
		ID:        uint(len(m.messages[chatID]) + 1),
		ChatID:    chatID,
		Role:      role,
		Content:   content,
		CreatedAt: time.Now(),
	}
	m.messages[chatID] = append(m.messages[chatID], msg)
	return &msg, nil
}

func (m *mockChatRepository) CreateMessagePair(ctx context.Context, chatID uint, userContent, aiContent string) (*Message, *Message, error) {
	uMsg, _ := m.CreateMessage(ctx, chatID, RoleUser, userContent)
	aMsg, _ := m.CreateMessage(ctx, chatID, RoleAssistant, aiContent)
	return uMsg, aMsg, nil
}

type mockChatClient struct {
	replyText string
	err       error
}

func (m *mockChatClient) GenerateReply(
	ctx context.Context,
	providerName types.Provider,
	modelID, apiKey, systemPrompt string,
	history []ai.ChatMessage,
	prompt string,
) (string, error) {
	if m.err != nil {
		return "", m.err
	}
	return m.replyText, nil
}

func (m *mockChatClient) StreamReply(
	ctx context.Context,
	providerName types.Provider,
	modelID, apiKey, systemPrompt string,
	history []ai.ChatMessage,
	prompt string,
) (*goai.TextStream, error) {
	if m.err != nil {
		return nil, m.err
	}
	return nil, nil
}

type mockModelFetcher struct {
	models map[types.Provider][]types.AIModel
	err    error
}

func (m *mockModelFetcher) FetchModels(ctx context.Context, providerName types.Provider, apiKey string) ([]types.AIModel, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.models[providerName], nil
}

type mockModelCache struct {
	store map[string][]types.AIModel
}

func newMockModelCache() *mockModelCache {
	return &mockModelCache{store: make(map[string][]types.AIModel)}
}

func (m *mockModelCache) Get(userID string, provider types.Provider) ([]types.AIModel, bool) {
	models, ok := m.store[userID+":"+string(provider)]
	return models, ok
}

func (m *mockModelCache) Set(userID string, provider types.Provider, models []types.AIModel) {
	m.store[userID+":"+string(provider)] = models
}

func TestChatService_CreateChat(t *testing.T) {
	repo := newMockChatRepository()
	svc := NewService(repo, &mockChatClient{}, nil, nil)

	ctx := context.Background()

	// 1. With title
	chat1, err := svc.CreateChat(ctx, "user_1", "テスト会話")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if chat1.Title != "テスト会話" || chat1.UserID != "user_1" {
		t.Errorf("unexpected chat: %+v", chat1)
	}

	// 2. Empty title defaults to "新規チャット"
	chat2, err := svc.CreateChat(ctx, "user_1", "   ")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if chat2.Title != "新規チャット" {
		t.Errorf("got title %q, want 新規チャット", chat2.Title)
	}
}

func TestChatService_SendMessage_Success(t *testing.T) {
	repo := newMockChatRepository()
	client := &mockChatClient{replyText: "AIからの回答です"}

	svc := NewService(repo, client, nil, nil)
	ctx := context.Background()

	// Create chat first
	c, _ := svc.CreateChat(ctx, "user_1", "新規会話")

	userMsg, aiMsg, err := svc.SendMessage(
		ctx,
		c.ID,
		"user_1",
		"こんにちは",
		types.ProviderOpenRouter,
		"anthropic/claude-3.5-sonnet",
		"test-api-key",
		"あなたは親切なAIです",
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if userMsg.Role != RoleUser || userMsg.Content != "こんにちは" {
		t.Errorf("unexpected user message: %+v", userMsg)
	}
	if aiMsg.Role != RoleAssistant || aiMsg.Content != "AIからの回答です" {
		t.Errorf("unexpected AI message: %+v", aiMsg)
	}

	// Verify chat now contains messages
	history, err := svc.GetChat(ctx, c.ID, "user_1")
	if err != nil {
		t.Fatalf("failed to get chat: %v", err)
	}
	if len(history.Messages) != 2 {
		t.Fatalf("expected 2 messages, got %d", len(history.Messages))
	}
}

func TestChatService_SendMessage_ValidationFailed(t *testing.T) {
	repo := newMockChatRepository()
	client := &mockChatClient{replyText: "AI"}

	svc := NewService(repo, client, nil, nil)
	ctx := context.Background()

	c, _ := svc.CreateChat(ctx, "user_1", "新規会話")

	// Missing content
	_, _, err := svc.SendMessage(ctx, c.ID, "user_1", "", types.ProviderOpenRouter, "model", "key", "")
	if !errors.Is(err, ErrValidationFailed) {
		t.Errorf("expected ErrValidationFailed for empty content, got %v", err)
	}

	// Missing apiKey
	_, _, err = svc.SendMessage(ctx, c.ID, "user_1", "hello", types.ProviderOpenRouter, "model", "", "")
	if !errors.Is(err, ErrValidationFailed) {
		t.Errorf("expected ErrValidationFailed for empty apiKey, got %v", err)
	}
}

func TestChatService_GetAvailableModels(t *testing.T) {
	fetcher := &mockModelFetcher{
		models: map[types.Provider][]types.AIModel{
			types.ProviderOpenRouter: {
				{ID: "openrouter/auto", Name: "Auto", Provider: types.ProviderOpenRouter},
			},
		},
	}
	cache := newMockModelCache()
	svc := NewService(nil, nil, fetcher, cache)

	ctx := context.Background()
	keys := []user.DecryptedKey{
		{Provider: types.ProviderOpenRouter, RawKey: "test-or-key"},
	}

	// 1. First call fetches from provider and populates cache
	models, providers, err := svc.GetAvailableModels(ctx, "user_1", keys, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(models) != 1 || models[0].ID != "openrouter/auto" {
		t.Errorf("unexpected models: %+v", models)
	}
	if len(providers) != 1 || providers[0] != types.ProviderOpenRouter {
		t.Errorf("unexpected providers: %+v", providers)
	}

	// 2. Second call should use cache
	cachedModels, found := cache.Get("user_1", types.ProviderOpenRouter)
	if !found || len(cachedModels) != 1 {
		t.Errorf("expected models in cache, got %+v", cachedModels)
	}
}
