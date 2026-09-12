package model

import "time"

// UserAPIKey represents an encrypted API key registered by a user for a specific AI provider.
type UserAPIKey struct {
	ID           uint      `gorm:"primaryKey;autoIncrement" json:"id"`
	UserID       string    `gorm:"size:64;index;not null;uniqueIndex:idx_user_provider" json:"user_id"`
	Provider     string    `gorm:"size:50;not null;uniqueIndex:idx_user_provider" json:"provider"` // "openrouter", "openai", "anthropic", "google"
	EncryptedKey string    `gorm:"type:text;not null" json:"-"`                                    // AES-256-GCM encrypted key (never serialized to JSON)
	KeyHint      string    `gorm:"size:50;not null" json:"key_hint"`                               // Masked representation for display (e.g. "sk-or-v1-...1234")
	CreatedAt    time.Time `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt    time.Time `gorm:"autoUpdateTime" json:"updated_at"`
}

