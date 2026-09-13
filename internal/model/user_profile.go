package model

import "time"

// UserProfile represents user-level profile and preferences including default system prompt.
type UserProfile struct {
	UserID       string    `gorm:"primaryKey;size:64" json:"user_id"`
	SystemPrompt string    `gorm:"type:text;not null;default:''" json:"system_prompt"`
	CreatedAt    time.Time `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt    time.Time `gorm:"autoUpdateTime" json:"updated_at"`
}

