package chat

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/zendev-sh/goai"

	"study-gin-clerk/internal/infra/ai"
	"study-gin-clerk/internal/profile"
	"study-gin-clerk/internal/types"
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

type mockKeyProvider struct {
	keys map[string]string // key = userID:provider
}

func (m *mockKeyProvider) GetDecryptedKey(ctx context.Context, userID string, provider types.Provider) (string, error) {
	k, ok := m.keys[userID+":"+string(provider)]
	if !ok {
		return "", errors.New("key not found")
	}
	return k, nil
}

type mockProfileProvider struct {
	systemPrompt string
}

func (m *mockProfileProvider) GetProfile(ctx context.Context, userID string) (*profile.Profile, error) {
	return &profile.Profile{
		UserID:       userID,
		SystemPrompt: m.systemPrompt,
	}, nil
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
	// For testing StreamReply setup without real GoAI stream
	return nil, nil
}

func TestChatService_CreateChat(t *testing.T) {
	repo := newMockChatRepository()
	svc := NewService(repo, &mockKeyProvider{}, &mockChatClient{}, &mockProfileProvider{})

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
	keyProvider := &mockKeyProvider{
		keys: map[string]string{
			"user_1:openrouter": "test-key-123",
		},
	}
	client := &mockChatClient{replyText: "AIからの回答です"}
	profileProvider := &mockProfileProvider{systemPrompt: "あなたは丁寧なAIです"}

	svc := NewService(repo, keyProvider, client, profileProvider)
	ctx := context.Background()

	// Create chat first
	c, _ := svc.CreateChat(ctx, "user_1", "新規会話")

	userMsg, aiMsg, err := svc.SendMessage(ctx, c.ID, "user_1", "こんにちは", types.ProviderOpenRouter, "anthropic/claude-3.5-sonnet")
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

func TestChatService_SendMessage_APIKeyNotConfigured(t *testing.T) {
	repo := newMockChatRepository()
	keyProvider := &mockKeyProvider{keys: map[string]string{}} // No keys registered
	client := &mockChatClient{replyText: "AI"}
	profileProvider := &mockProfileProvider{}

	svc := NewService(repo, keyProvider, client, profileProvider)
	ctx := context.Background()

	c, _ := svc.CreateChat(ctx, "user_1", "新規会話")

	_, _, err := svc.SendMessage(ctx, c.ID, "user_1", "こんにちは", types.ProviderOpenRouter, "some-model")
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
	if !errors.Is(err, ErrAPIKeyNotConfigured) {
		t.Errorf("expected ErrAPIKeyNotConfigured, got %v", err)
	}
}

