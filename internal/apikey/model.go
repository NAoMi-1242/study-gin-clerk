package apikey

import (
	"strings"
	"time"

	"study-gin-clerk/internal/types"
)

// Key represents an encrypted API key registered by a user for a specific AI provider.
type Key struct {
	ID           uint           `gorm:"primaryKey;autoIncrement" json:"id"`
	UserID       string         `gorm:"size:64;index;not null;uniqueIndex:idx_user_provider" json:"user_id"`
	Provider     types.Provider `gorm:"size:50;not null;uniqueIndex:idx_user_provider" json:"provider"` // ProviderOpenRouter, ProviderOpenAI, ProviderAnthropic, ProviderGoogle
	EncryptedKey string         `gorm:"type:text;not null" json:"-"`                                    // AES-256-GCM encrypted key (never serialized to JSON)
	KeyHint      string         `gorm:"size:50;not null" json:"key_hint"`                               // Masked representation for display (e.g. "sk-or-v1-...1234")
	CreatedAt    time.Time      `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt    time.Time      `gorm:"autoUpdateTime" json:"updated_at"`
}

// TableName specifies the GORM table name for Key.
func (Key) TableName() string {
	return "user_api_keys"
}

// MaskKey masks an API key for safe UI display (e.g., "sk-or-v1-...abcd").
func MaskKey(key string) string {
	key = strings.TrimSpace(key)
	if len(key) <= 8 {
		return "****"
	}
	if strings.HasPrefix(key, "sk-or-v1-") && len(key) > 13 {
		return "sk-or-v1-..." + key[len(key)-4:]
	}
	if strings.HasPrefix(key, "sk-") && len(key) > 8 {
		return "sk-..." + key[len(key)-4:]
	}
	if len(key) > 12 {
		return key[:6] + "..." + key[len(key)-4:]
	}
	return key[:2] + "..." + key[len(key)-2:]
}
