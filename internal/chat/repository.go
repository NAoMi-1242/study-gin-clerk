package chat

import (
	"context"
	"errors"
	"time"

	"gorm.io/gorm"
)

type Repository struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) *Repository {
	return &Repository{db: db}
}

// Create creates a new chat session for a user.
func (r *Repository) Create(ctx context.Context, userID, title string) (*Chat, error) {
	c := &Chat{
		UserID: userID,
		Title:  title,
	}
	if err := r.db.WithContext(ctx).Create(c).Error; err != nil {
		return nil, err
	}
	return c, nil
}

// ListByUserID retrieves all chats owned by the specified user.
func (r *Repository) ListByUserID(ctx context.Context, userID string) ([]Chat, error) {
	var chats []Chat
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
func (r *Repository) GetWithMessages(ctx context.Context, chatID uint, userID string) (*Chat, error) {
	var c Chat
	err := r.db.WithContext(ctx).
		Preload("Messages", func(db *gorm.DB) *gorm.DB {
			return db.Order("messages.created_at ASC")
		}).
		Where("id = ? AND user_id = ?", chatID, userID).
		First(&c).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &c, nil
}

// CreateMessage adds a new message to a chat and updates the chat's updated_at timestamp.
func (r *Repository) CreateMessage(ctx context.Context, chatID uint, role Role, content string) (*Message, error) {
	msg := &Message{
		ChatID:  chatID,
		Role:    role,
		Content: content,
	}

	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(msg).Error; err != nil {
			return err
		}
		return tx.Model(&Chat{}).
			Where("id = ?", chatID).
			Update("updated_at", time.Now()).Error
	})
	if err != nil {
		return nil, err
	}

	return msg, nil
}

// CreateMessagePair persists both a user message and an assistant reply in a single atomic transaction.
func (r *Repository) CreateMessagePair(ctx context.Context, chatID uint, userContent, aiContent string) (*Message, *Message, error) {
	userMsg := &Message{
		ChatID:  chatID,
		Role:    RoleUser,
		Content: userContent,
	}
	aiMsg := &Message{
		ChatID:  chatID,
		Role:    RoleAssistant,
		Content: aiContent,
	}

	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(userMsg).Error; err != nil {
			return err
		}
		if err := tx.Create(aiMsg).Error; err != nil {
			return err
		}
		return tx.Model(&Chat{}).
			Where("id = ?", chatID).
			Update("updated_at", time.Now()).Error
	})
	if err != nil {
		return nil, nil, err
	}

	return userMsg, aiMsg, nil
}
