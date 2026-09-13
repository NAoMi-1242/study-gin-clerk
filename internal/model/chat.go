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

// Role represents the sender role of a chat message.
type Role string

const (
	RoleUser      Role = "user"
	RoleAssistant Role = "assistant"
	RoleSystem    Role = "system"
)

// IsValid reports whether the role is supported.
func (r Role) IsValid() bool {
	switch r {
	case RoleUser, RoleAssistant, RoleSystem:
		return true
	default:
		return false
	}
}

// String returns the string representation of the role.
func (r Role) String() string {
	return string(r)
}

// Message represents a single message in a chat conversation.
type Message struct {
	ID        uint      `gorm:"primaryKey;autoIncrement" json:"id"`
	ChatID    uint      `gorm:"index;not null" json:"chat_id"`
	Role      Role      `gorm:"size:20;not null" json:"role"` // "user", "assistant", "system"
	Content   string    `gorm:"type:text;not null" json:"content"`
	CreatedAt time.Time `gorm:"autoCreateTime" json:"created_at"`
}

