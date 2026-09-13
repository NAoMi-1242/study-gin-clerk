package chat

import (
	"context"
	"fmt"
	"strings"

	"github.com/zendev-sh/goai"

	"study-gin-clerk/internal/infra/ai"
	"study-gin-clerk/internal/profile"
	"study-gin-clerk/internal/types"
)

// KeyProvider defines the interface required by ChatService to retrieve decrypted API keys.
type KeyProvider interface {
	GetDecryptedKey(ctx context.Context, userID string, provider types.Provider) (string, error)
}

// ProfileProvider defines the interface required by ChatService to retrieve user profile settings.
type ProfileProvider interface {
	GetProfile(ctx context.Context, userID string) (*profile.Profile, error)
}

// StreamMessageResult holds the stream instance and message persistence hook.
type StreamMessageResult struct {
	UserMessage *Message
	TextStream  *goai.TextStream
	OnComplete  func(fullText string) (*Message, error)
}

type Service struct {
	repo            *Repository
	keyProvider     KeyProvider
	aiClient        *ai.Client
	profileProvider ProfileProvider
}

func NewService(
	repo *Repository,
	keyProvider KeyProvider,
	aiClient *ai.Client,
	profileProvider ProfileProvider,
) *Service {
	return &Service{
		repo:            repo,
		keyProvider:     keyProvider,
		aiClient:        aiClient,
		profileProvider: profileProvider,
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

type sessionContext struct {
	chat         *Chat
	apiKey       string
	systemPrompt string
}

func (s *Service) prepareSession(
	ctx context.Context,
	chatID uint,
	userID string,
	providerName types.Provider,
	modelID string,
) (*sessionContext, error) {
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
	apiKey, err := s.keyProvider.GetDecryptedKey(ctx, userID, providerName)
	if err != nil {
		return nil, err
	}

	// 3. ユーザーの最新システムプロンプトを取得 (動的適用)
	userProfile, _ := s.profileProvider.GetProfile(ctx, userID)
	systemPrompt := ""
	if userProfile != nil {
		systemPrompt = userProfile.SystemPrompt
	}

	return &sessionContext{
		chat:         c,
		apiKey:       apiKey,
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
	content = strings.TrimSpace(content)
	if content == "" {
		return nil, nil, fmt.Errorf("%w: content cannot be empty", ErrValidationFailed)
	}
	modelID = strings.TrimSpace(modelID)

	session, err := s.prepareSession(ctx, chatID, userID, providerName, modelID)
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

	// AI 生成成功後にユーザー発言と AI 返答を単一トランザクションでアトミックに DB 保存
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
) (*StreamMessageResult, error) {
	content = strings.TrimSpace(content)
	if content == "" {
		return nil, fmt.Errorf("%w: content cannot be empty", ErrValidationFailed)
	}
	modelID = strings.TrimSpace(modelID)

	session, err := s.prepareSession(ctx, chatID, userID, providerName, modelID)
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

	// ストリーム接続確立後にユーザーメッセージを保存
	userMsg, err := s.repo.CreateMessage(ctx, chatID, RoleUser, content)
	if err != nil {
		return nil, fmt.Errorf("failed to save user message: %w", err)
	}

	// ストリーム完了時に呼び出す DB 保存コールバック
	onComplete := func(fullText string) (*Message, error) {
		return s.repo.CreateMessage(context.Background(), chatID, RoleAssistant, fullText)
	}

	return &StreamMessageResult{
		UserMessage: userMsg,
		TextStream:  stream,
		OnComplete:  onComplete,
	}, nil
}
