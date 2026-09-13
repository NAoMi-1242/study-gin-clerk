package service

import (
	"context"
	"fmt"

	"github.com/zendev-sh/goai"

	"study-gin-clerk/internal/infra/ai"
	"study-gin-clerk/internal/model"
	"study-gin-clerk/internal/infra/repository"
)

type ChatService struct {
	chatRepo        *repository.ChatRepository
	apiKeyService   *UserAPIKeyService
	aiClient        *ai.Client
	userProfileRepo *repository.UserProfileRepository
}

func NewChatService(
	chatRepo *repository.ChatRepository,
	apiKeyService *UserAPIKeyService,
	aiClient *ai.Client,
	userProfileRepo *repository.UserProfileRepository,
) *ChatService {
	return &ChatService{
		chatRepo:        chatRepo,
		apiKeyService:   apiKeyService,
		aiClient:        aiClient,
		userProfileRepo: userProfileRepo,
	}
}

// CreateChat creates a new conversation for the user.
func (s *ChatService) CreateChat(ctx context.Context, userID, title string) (*model.Chat, error) {
	if title == "" {
		title = "新規チャット"
	}
	return s.chatRepo.CreateChat(ctx, userID, title)
}

// ListChats retrieves all conversations for the user.
func (s *ChatService) ListChats(ctx context.Context, userID string) ([]model.Chat, error) {
	return s.chatRepo.ListChatsByUserID(ctx, userID)
}

// GetChat retrieves conversation details and message history, enforcing ownership.
func (s *ChatService) GetChat(ctx context.Context, chatID uint, userID string) (*model.Chat, error) {
	return s.chatRepo.GetChatWithMessages(ctx, chatID, userID)
}

// SendMessage handles saving the user's message, generating an AI reply with GoAI, and saving it to DB.
func (s *ChatService) SendMessage(
	ctx context.Context,
	chatID uint,
	userID, content, providerName, modelID string,
) (*model.Message, *model.Message, error) {
	if providerName == "" {
		return nil, nil, fmt.Errorf("provider is required")
	}
	if modelID == "" {
		return nil, nil, fmt.Errorf("model is required")
	}

	// 1. チャットの所有権と過去メッセージを取得
	chat, err := s.chatRepo.GetChatWithMessages(ctx, chatID, userID)
	if err != nil {
		return nil, nil, fmt.Errorf("chat not found or unauthorized: %w", err)
	}

	// 2. ユーザーの API キーを復号して取得
	apiKey, err := s.apiKeyService.GetDecryptedKey(ctx, userID, providerName)
	if err != nil {
		return nil, nil, err
	}

	// 3. ユーザーのメッセージを DB 保存
	userMsg, err := s.chatRepo.CreateMessage(ctx, chatID, "user", content)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to save user message: %w", err)
	}

	// 4. ユーザーの最新システムプロンプトを取得 (常に最新の共通設定を動的適用)
	userProfile, _ := s.userProfileRepo.GetProfile(ctx, userID)
	systemPrompt := ""
	if userProfile != nil {
		systemPrompt = userProfile.SystemPrompt
	}

	// 5. GoAI SDK を用いて AI の返答を生成
	aiContent, err := s.aiClient.GenerateReply(ctx, providerName, modelID, apiKey, systemPrompt, chat.Messages, content)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to generate AI reply: %w", err)
	}

	// 6. AI の返答メッセージを DB 保存
	aiMsg, err := s.chatRepo.CreateMessage(ctx, chatID, "assistant", aiContent)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to save assistant message: %w", err)
	}

	return userMsg, aiMsg, nil
}

// StreamMessage prepares a streaming response using GoAI StreamText.
// Returns the saved user message, the active TextStream, and a completion callback to persist the assistant reply.
func (s *ChatService) StreamMessage(
	ctx context.Context,
	chatID uint,
	userID, content, providerName, modelID string,
) (*model.Message, *goai.TextStream, func(fullText string) (*model.Message, error), error) {
	if providerName == "" {
		return nil, nil, nil, fmt.Errorf("provider is required")
	}
	if modelID == "" {
		return nil, nil, nil, fmt.Errorf("model is required")
	}

	// 1. チャットの所有権と過去履歴を取得
	chat, err := s.chatRepo.GetChatWithMessages(ctx, chatID, userID)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("chat not found or unauthorized: %w", err)
	}

	// 2. ユーザーの API キーを復号して取得
	apiKey, err := s.apiKeyService.GetDecryptedKey(ctx, userID, providerName)
	if err != nil {
		return nil, nil, nil, err
	}

	// 3. ユーザーのメッセージを DB 保存
	userMsg, err := s.chatRepo.CreateMessage(ctx, chatID, "user", content)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to save user message: %w", err)
	}

	// 4. ユーザーの最新システムプロンプトを取得 (常に最新の共通設定を動的適用)
	userProfile, _ := s.userProfileRepo.GetProfile(ctx, userID)
	systemPrompt := ""
	if userProfile != nil {
		systemPrompt = userProfile.SystemPrompt
	}

	// 5. GoAI StreamText を呼び出し
	stream, err := s.aiClient.StreamReply(ctx, providerName, modelID, apiKey, systemPrompt, chat.Messages, content)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to initiate AI stream: %w", err)
	}

	// 5. ストリーム完了時に呼び出す DB 保存コールバック
	onComplete := func(fullText string) (*model.Message, error) {
		return s.chatRepo.CreateMessage(context.Background(), chatID, "assistant", fullText)
	}

	return userMsg, stream, onComplete, nil
}
