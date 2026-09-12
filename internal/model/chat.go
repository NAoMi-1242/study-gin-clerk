package model

import "time"

// Chat represents a conversation session owned by a Clerk user.
type Chat struct {
	ID        uint      `gorm:"primaryKey;autoIncrement" json:"id"`
	UserID    string    `gorm:"size:64;index;not null" json:"user_id"`
	Title     string    `gorm:"size:255;not null" json:"title"`
	CreatedAt time.Time `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt time.Time `gorm:"autoUpdateTime" json:"updated_at"`

	Messages []Message `gorm:"foreignKey:ChatID;constraint:OnDelete:CASCADE" json:"messages,omitempty"`
}

// Message represents a single message in a chat conversation.
type Message struct {
	ID        uint      `gorm:"primaryKey;autoIncrement" json:"id"`
	ChatID    uint      `gorm:"index;not null" json:"chat_id"`
	Role      string    `gorm:"size:20;not null" json:"role"` // "user", "assistant", "system"
	Content   string    `gorm:"type:text;not null" json:"content"`
	CreatedAt time.Time `gorm:"autoCreateTime" json:"created_at"`
}

