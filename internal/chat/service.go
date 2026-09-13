package chat

import (
	"context"
	"fmt"

	"github.com/zendev-sh/goai"

	"study-gin-clerk/internal/apikey"
	"study-gin-clerk/internal/infra/ai"
	"study-gin-clerk/internal/profile"
	"study-gin-clerk/internal/types"
)

// StreamMessageResult holds the stream instance and message persistence hook.
type StreamMessageResult struct {
	UserMessage *Message
	TextStream  *goai.TextStream
	OnComplete  func(fullText string) (*Message, error)
}

type Service struct {
	repo           *Repository
	apiKeyService  *apikey.Service
	aiClient       *ai.Client
	profileService *profile.Service
}

func NewService(
	repo *Repository,
	apiKeyService *apikey.Service,
	aiClient *ai.Client,
	profileService *profile.Service,
) *Service {
	return &Service{
		repo:           repo,
		apiKeyService:  apiKeyService,
		aiClient:       aiClient,
		profileService: profileService,
	}
}

// CreateChat creates a new conversation for the user.
func (s *Service) CreateChat(ctx context.Context, userID, title string) (*Chat, error) {
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

type chatSessionContext struct {
	chat         *Chat
	apiKey       string
	userMsg      *Message
	systemPrompt string
}

func (s *Service) prepareSessionContext(
	ctx context.Context,
	chatID uint,
	userID, content string,
	providerName types.Provider,
	modelID string,
) (*chatSessionContext, error) {
	if !providerName.IsValid() {
		return nil, fmt.Errorf("%w: unsupported provider '%s'", ErrValidationFailed, providerName)
	}
	if modelID == "" {
		return nil, fmt.Errorf("%w: model is required", ErrValidationFailed)
	}

	// 1. チャットの所有権と過去メッセージ履歴を取得
	c, err := s.repo.GetWithMessages(ctx, chatID, userID)
	if err != nil {
		return nil, err
	}

	// 2. ユーザーの API キーを復号して取得
	apiKey, err := s.apiKeyService.GetDecryptedKey(ctx, userID, providerName)
	if err != nil {
		return nil, err
	}

	// 3. ユーザーのメッセージを DB 保存
	userMsg, err := s.repo.CreateMessage(ctx, chatID, RoleUser, content)
	if err != nil {
		return nil, fmt.Errorf("failed to save user message: %w", err)
	}

	// 4. ユーザーの最新システムプロンプトを取得 (動的適用)
	userProfile, _ := s.profileService.GetProfile(ctx, userID)
	systemPrompt := ""
	if userProfile != nil {
		systemPrompt = userProfile.SystemPrompt
	}

	return &chatSessionContext{
		chat:         c,
		apiKey:       apiKey,
		userMsg:      userMsg,
		systemPrompt: systemPrompt,
	}, nil
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
) (*Message, *Message, error) {
	session, err := s.prepareSessionContext(ctx, chatID, userID, content, providerName, modelID)
	if err != nil {
		return nil, nil, err
	}

	history := toAIChatMessages(session.chat.Messages)

	// GoAI SDK を用いて AI の返答を生成
	aiContent, err := s.aiClient.GenerateReply(
		ctx,
		providerName,
		modelID,
		session.apiKey,
		session.systemPrompt,
		history,
		content,
	)
	if err != nil {
		return nil, nil, fmt.Errorf("%w: %s", ErrAIProvider, err.Error())
	}

	// AI の返答メッセージを DB 保存
	aiMsg, err := s.repo.CreateMessage(ctx, chatID, RoleAssistant, aiContent)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to save assistant message: %w", err)
	}

	return session.userMsg, aiMsg, nil
}

// StreamMessage prepares a streaming response using GoAI StreamText.
func (s *Service) StreamMessage(
	ctx context.Context,
	chatID uint,
	userID, content string,
	providerName types.Provider,
	modelID string,
) (*StreamMessageResult, error) {
	session, err := s.prepareSessionContext(ctx, chatID, userID, content, providerName, modelID)
	if err != nil {
		return nil, err
	}

	history := toAIChatMessages(session.chat.Messages)

	// GoAI StreamText を呼び出し
	stream, err := s.aiClient.StreamReply(
		ctx,
		providerName,
		modelID,
		session.apiKey,
		session.systemPrompt,
		history,
		content,
	)
	if err != nil {
		return nil, fmt.Errorf("%w: %s", ErrAIProvider, err.Error())
	}

	// ストリーム完了時に呼び出す DB 保存コールバック
	onComplete := func(fullText string) (*Message, error) {
		return s.repo.CreateMessage(context.Background(), chatID, RoleAssistant, fullText)
	}

	return &StreamMessageResult{
		UserMessage: session.userMsg,
		TextStream:  stream,
		OnComplete:  onComplete,
	}, nil
}

