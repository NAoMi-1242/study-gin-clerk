package repository

import (
	"context"
	"time"

	"gorm.io/gorm"

	"study-gin-clerk/internal/model"
)

type ChatRepository struct {
	db *gorm.DB
}

func NewChatRepository(db *gorm.DB) *ChatRepository {
	return &ChatRepository{db: db}
}

// Create creates a new chat session for a user.
func (r *ChatRepository) Create(ctx context.Context, userID, title string) (*model.Chat, error) {
	chat := &model.Chat{
		UserID: userID,
		Title:  title,
	}
	if err := r.db.WithContext(ctx).Create(chat).Error; err != nil {
		return nil, err
	}
	return chat, nil
}

// ListByUserID retrieves all chats owned by the specified user.
func (r *ChatRepository) ListByUserID(ctx context.Context, userID string) ([]model.Chat, error) {
	var chats []model.Chat
	err := r.db.WithContext(ctx).
		Where("user_id = ?", userID).
		Order("updated_at DESC").
		Find(&chats).Error
	if err != nil {
		return nil, err
	}
	return chats, nil
}

// GetWithMessages retrieves a single chat and its messages, enforcing user ownership.
func (r *ChatRepository) GetWithMessages(ctx context.Context, chatID uint, userID string) (*model.Chat, error) {
	var chat model.Chat
	err := r.db.WithContext(ctx).
		Preload("Messages", func(db *gorm.DB) *gorm.DB {
			return db.Order("messages.created_at ASC")
		}).
		Where("id = ? AND user_id = ?", chatID, userID).
		First(&chat).Error
	if err != nil {
		return nil, err
	}
	return &chat, nil
}

// CreateMessage adds a new message to a chat and updates the chat's updated_at timestamp.
func (r *ChatRepository) CreateMessage(ctx context.Context, chatID uint, role model.Role, content string) (*model.Message, error) {
	msg := &model.Message{
		ChatID:  chatID,
		Role:    role,
		Content: content,
	}

	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(msg).Error; err != nil {
			return err
		}
		return tx.Model(&model.Chat{}).
			Where("id = ?", chatID).
			Update("updated_at", time.Now()).Error
	})
	if err != nil {
		return nil, err
	}

	return msg, nil
}
