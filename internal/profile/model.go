package profile

import "time"

// Profile represents user-level profile and preferences including default system prompt.
type Profile struct {
	UserID       string    `gorm:"primaryKey;size:64" json:"user_id"`
	SystemPrompt string    `gorm:"type:text;not null;default:''" json:"system_prompt"`
	CreatedAt    time.Time `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt    time.Time `gorm:"autoUpdateTime" json:"updated_at"`
}

// TableName specifies the GORM table name for Profile.
func (Profile) TableName() string {
	return "user_profiles"
}
