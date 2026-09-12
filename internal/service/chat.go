package service

import (
	"context"
	"fmt"

	"study-gin-clerk/internal/model"
	"study-gin-clerk/internal/repository"
)

type ChatService struct {
	chatRepo *repository.ChatRepository
}

func NewChatService(chatRepo *repository.ChatRepository) *ChatService {
	return &ChatService{chatRepo: chatRepo}
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

// SendMessage handles saving the user's message and generating/saving the AI assistant's reply.
func (s *ChatService) SendMessage(ctx context.Context, chatID uint, userID, content string) (*model.Message, *model.Message, error) {
	// 1. チャットの所有権を確認
	if _, err := s.chatRepo.GetChatWithMessages(ctx, chatID, userID); err != nil {
		return nil, nil, fmt.Errorf("chat not found or unauthorized: %w", err)
	}

	// 2. ユーザーのメッセージを保存
	userMsg, err := s.chatRepo.CreateMessage(ctx, chatID, "user", content)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to save user message: %w", err)
	}

	// 3. AI の返答を生成（将来ここに Go AI / Gemini SDK の呼び出しが入ります）
	aiContent := fmt.Sprintf("【AI応答モック】「%s」について承知いたしました。何か他にご質問はありますか？", content)

	// 4. AI のメッセージを保存
	aiMsg, err := s.chatRepo.CreateMessage(ctx, chatID, "assistant", aiContent)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to save assistant message: %w", err)
	}

	return userMsg, aiMsg, nil
}

