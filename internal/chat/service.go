package chat

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/zendev-sh/goai"

	"study-gin-clerk/internal/infra/ai"
	"study-gin-clerk/internal/types"
	"study-gin-clerk/internal/user"
)

// ChatRepository defines the storage interface for chats and messages.
type ChatRepository interface {
	Create(ctx context.Context, userID, title string) (*Chat, error)
	ListByUserID(ctx context.Context, userID string) ([]Chat, error)
	GetWithMessages(ctx context.Context, chatID uint, userID string) (*Chat, error)
	CreateMessage(ctx context.Context, chatID uint, role Role, content string) (*Message, error)
	CreateMessagePair(ctx context.Context, chatID uint, userContent, aiContent string) (*Message, *Message, error)
}

// ChatClient defines text generation and streaming capabilities for AI chat.
type ChatClient interface {
	GenerateReply(
		ctx context.Context,
		providerName types.Provider,
		modelID, apiKey, systemPrompt string,
		history []ai.ChatMessage,
		prompt string,
	) (string, error)

	StreamReply(
		ctx context.Context,
		providerName types.Provider,
		modelID, apiKey, systemPrompt string,
		history []ai.ChatMessage,
		prompt string,
	) (*goai.TextStream, error)
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

// StreamMessageResult holds the stream instance and message persistence hook.
type StreamMessageResult struct {
	UserMessage *Message
	TextStream  *goai.TextStream
	OnComplete  func(fullText string) (*Message, error)
}

type Service struct {
	repo     ChatRepository
	aiClient ChatClient
	fetcher  ModelFetcher
	cache    ModelCache
}

func NewService(
	repo ChatRepository,
	aiClient ChatClient,
	fetcher ModelFetcher,
	cache ModelCache,
) *Service {
	return &Service{
		repo:     repo,
		aiClient: aiClient,
		fetcher:  fetcher,
		cache:    cache,
	}
}

// CreateChat creates a new conversation for the user.
func (s *Service) CreateChat(ctx context.Context, userID, title string) (*Chat, error) {
	title = strings.TrimSpace(title)
	if title == "" {
		title = "新規チャット"
	}
	return s.repo.Create(ctx, userID, title)
}

// ListChats retrieves all conversations for the user.
func (s *Service) ListChats(ctx context.Context, userID string) ([]Chat, error) {
	return s.repo.ListByUserID(ctx, userID)
}

// GetChat retrieves conversation details and message history, enforcing ownership.
func (s *Service) GetChat(ctx context.Context, chatID uint, userID string) (*Chat, error) {
	return s.repo.GetWithMessages(ctx, chatID, userID)
}

func toAIChatMessages(messages []Message) []ai.ChatMessage {
	aiMsgs := make([]ai.ChatMessage, 0, len(messages))
	for _, m := range messages {
		aiMsgs = append(aiMsgs, ai.ChatMessage{
			Role:    string(m.Role),
			Content: m.Content,
		})
	}
	return aiMsgs
}

// SendMessage handles saving user message, generating AI reply with GoAI, and saving it to DB.
func (s *Service) SendMessage(
	ctx context.Context,
	chatID uint,
	userID, content string,
	providerName types.Provider,
	modelID string,
	apiKey string,
	systemPrompt string,
) (*Message, *Message, error) {
	content = strings.TrimSpace(content)
	if content == "" {
		return nil, nil, fmt.Errorf("%w: content cannot be empty", ErrValidationFailed)
	}
	modelID = strings.TrimSpace(modelID)
	if modelID == "" {
		return nil, nil, fmt.Errorf("%w: model is required", ErrValidationFailed)
	}
	if !providerName.IsValid() {
		return nil, nil, fmt.Errorf("%w: unsupported provider '%s'", ErrValidationFailed, providerName)
	}
	if apiKey == "" {
		return nil, nil, fmt.Errorf("%w: api key is required", ErrValidationFailed)
	}

	// 1. チャットの所有権と過去メッセージ履歴を取得
	chat, err := s.repo.GetWithMessages(ctx, chatID, userID)
	if err != nil {
		return nil, nil, err
	}

	history := toAIChatMessages(chat.Messages)

	// 2. GoAI SDK を用いて AI の返答を生成
	aiContent, err := s.aiClient.GenerateReply(
		ctx,
		providerName,
		modelID,
		apiKey,
		systemPrompt,
		history,
		content,
	)
	if err != nil {
		return nil, nil, fmt.Errorf("%w: %s", ErrAIProvider, err.Error())
	}

	// 3. AI 生成成功後にユーザー発言と AI 返答を単一トランザクションでアトミックに DB 保存
	userMsg, aiMsg, err := s.repo.CreateMessagePair(ctx, chatID, content, aiContent)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to save messages: %w", err)
	}

	return userMsg, aiMsg, nil
}

// StreamMessage prepares a streaming response using GoAI StreamText.
func (s *Service) StreamMessage(
	ctx context.Context,
	chatID uint,
	userID, content string,
	providerName types.Provider,
	modelID string,
	apiKey string,
	systemPrompt string,
) (*StreamMessageResult, error) {
	content = strings.TrimSpace(content)
	if content == "" {
		return nil, fmt.Errorf("%w: content cannot be empty", ErrValidationFailed)
	}
	modelID = strings.TrimSpace(modelID)
	if modelID == "" {
		return nil, fmt.Errorf("%w: model is required", ErrValidationFailed)
	}
	if !providerName.IsValid() {
		return nil, fmt.Errorf("%w: unsupported provider '%s'", ErrValidationFailed, providerName)
	}
	if apiKey == "" {
		return nil, fmt.Errorf("%w: api key is required", ErrValidationFailed)
	}

	// 1. チャットの所有権と過去メッセージ履歴を取得
	chat, err := s.repo.GetWithMessages(ctx, chatID, userID)
	if err != nil {
		return nil, err
	}

	history := toAIChatMessages(chat.Messages)

	// 2. GoAI StreamText を呼び出し
	stream, err := s.aiClient.StreamReply(
		ctx,
		providerName,
		modelID,
		apiKey,
		systemPrompt,
		history,
		content,
	)
	if err != nil {
		return nil, fmt.Errorf("%w: %s", ErrAIProvider, err.Error())
	}

	// 3. ストリーム接続確立後にユーザーメッセージを保存
	userMsg, err := s.repo.CreateMessage(ctx, chatID, RoleUser, content)
	if err != nil {
		return nil, fmt.Errorf("failed to save user message: %w", err)
	}

	// 4. ストリーム完了時に呼び出す DB 保存コールバック (10秒の安全なタイムアウト付き)
	onComplete := func(fullText string) (*Message, error) {
		saveCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		return s.repo.CreateMessage(saveCtx, chatID, RoleAssistant, fullText)
	}

	return &StreamMessageResult{
		UserMessage: userMsg,
		TextStream:  stream,
		OnComplete:  onComplete,
	}, nil
}

// GetAvailableModels retrieves dynamically discovered models for all active providers of the user, using cache.
func (s *Service) GetAvailableModels(
	ctx context.Context,
	userID string,
	keys []user.DecryptedKey,
	refresh bool,
) ([]types.AIModel, []types.Provider, error) {
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

		if s.fetcher == nil {
			continue
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
