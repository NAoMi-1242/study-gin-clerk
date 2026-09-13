package service

import (
	"context"
	"errors"
	"fmt"

	"github.com/zendev-sh/goai"
	"gorm.io/gorm"

	"study-gin-clerk/internal/infra/ai"
	"study-gin-clerk/internal/infra/repository"
	"study-gin-clerk/internal/model"
)

// StreamMessageResult holds the stream instance and message persistence hook.
type StreamMessageResult struct {
	UserMessage *model.Message
	TextStream  *goai.TextStream
	OnComplete  func(fullText string) (*model.Message, error)
}

type ChatService struct {
	chatRepo           *repository.ChatRepository
	userAPIKeyService  *UserAPIKeyService
	aiClient           *ai.Client
	userProfileService *UserProfileService
}

func NewChatService(
	chatRepo *repository.ChatRepository,
	userAPIKeyService *UserAPIKeyService,
	aiClient *ai.Client,
	userProfileService *UserProfileService,
) *ChatService {
	return &ChatService{
		chatRepo:           chatRepo,
		userAPIKeyService:  userAPIKeyService,
		aiClient:           aiClient,
		userProfileService: userProfileService,
	}
}

// CreateChat creates a new conversation for the user.
func (s *ChatService) CreateChat(ctx context.Context, userID, title string) (*model.Chat, error) {
	if title == "" {
		title = "新規チャット"
	}
	return s.chatRepo.Create(ctx, userID, title)
}

// ListChats retrieves all conversations for the user.
func (s *ChatService) ListChats(ctx context.Context, userID string) ([]model.Chat, error) {
	return s.chatRepo.ListByUserID(ctx, userID)
}

// GetChat retrieves conversation details and message history, enforcing ownership.
func (s *ChatService) GetChat(ctx context.Context, chatID uint, userID string) (*model.Chat, error) {
	chat, err := s.chatRepo.GetWithMessages(ctx, chatID, userID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrChatNotFound
		}
		return nil, err
	}
	return chat, nil
}

type chatSessionContext struct {
	chat         *model.Chat
	apiKey       string
	userMsg      *model.Message
	systemPrompt string
}

func (s *ChatService) prepareSessionContext(
	ctx context.Context,
	chatID uint,
	userID, content string,
	providerName model.Provider,
	modelID string,
) (*chatSessionContext, error) {
	if !providerName.IsValid() {
		return nil, fmt.Errorf("%w: unsupported provider '%s'", ErrValidationFailed, providerName)
	}
	if modelID == "" {
		return nil, fmt.Errorf("%w: model is required", ErrValidationFailed)
	}

	// 1. チャットの所有権と過去メッセージ履歴を取得
	chat, err := s.chatRepo.GetWithMessages(ctx, chatID, userID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrChatNotFound
		}
		return nil, fmt.Errorf("failed to retrieve chat: %w", err)
	}

	// 2. ユーザーの API キーを復号して取得
	apiKey, err := s.userAPIKeyService.GetDecryptedKey(ctx, userID, providerName)
	if err != nil {
		return nil, err
	}

	// 3. ユーザーのメッセージを DB 保存
	userMsg, err := s.chatRepo.CreateMessage(ctx, chatID, model.RoleUser, content)
	if err != nil {
		return nil, fmt.Errorf("failed to save user message: %w", err)
	}

	// 4. ユーザーの最新システムプロンプトを取得 (常に最新の共通設定を動的適用)
	userProfile, _ := s.userProfileService.GetProfile(ctx, userID)
	systemPrompt := ""
	if userProfile != nil {
		systemPrompt = userProfile.SystemPrompt
	}

	return &chatSessionContext{
		chat:         chat,
		apiKey:       apiKey,
		userMsg:      userMsg,
		systemPrompt: systemPrompt,
	}, nil
}

// SendMessage handles saving the user's message, generating an AI reply with GoAI, and saving it to DB.
func (s *ChatService) SendMessage(
	ctx context.Context,
	chatID uint,
	userID, content string,
	providerName model.Provider,
	modelID string,
) (*model.Message, *model.Message, error) {
	session, err := s.prepareSessionContext(ctx, chatID, userID, content, providerName, modelID)
	if err != nil {
		return nil, nil, err
	}

	// GoAI SDK を用いて AI の返答を生成
	aiContent, err := s.aiClient.GenerateReply(
		ctx,
		providerName,
		modelID,
		session.apiKey,
		session.systemPrompt,
		session.chat.Messages,
		content,
	)
	if err != nil {
		return nil, nil, fmt.Errorf("%w: %s", ErrAIProvider, err.Error())
	}

	// AI の返答メッセージを DB 保存
	aiMsg, err := s.chatRepo.CreateMessage(ctx, chatID, model.RoleAssistant, aiContent)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to save assistant message: %w", err)
	}

	return session.userMsg, aiMsg, nil
}

// StreamMessage prepares a streaming response using GoAI StreamText.
func (s *ChatService) StreamMessage(
	ctx context.Context,
	chatID uint,
	userID, content string,
	providerName model.Provider,
	modelID string,
) (*StreamMessageResult, error) {
	session, err := s.prepareSessionContext(ctx, chatID, userID, content, providerName, modelID)
	if err != nil {
		return nil, err
	}

	// GoAI StreamText を呼び出し
	stream, err := s.aiClient.StreamReply(
		ctx,
		providerName,
		modelID,
		session.apiKey,
		session.systemPrompt,
		session.chat.Messages,
		content,
	)
	if err != nil {
		return nil, fmt.Errorf("%w: %s", ErrAIProvider, err.Error())
	}

	// ストリーム完了時に呼び出す DB 保存コールバック
	onComplete := func(fullText string) (*model.Message, error) {
		return s.chatRepo.CreateMessage(context.Background(), chatID, model.RoleAssistant, fullText)
	}

	return &StreamMessageResult{
		UserMessage: session.userMsg,
		TextStream:  stream,
		OnComplete:  onComplete,
	}, nil
}
