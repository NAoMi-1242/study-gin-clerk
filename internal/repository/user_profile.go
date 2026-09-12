package repository

import (
	"context"
	"errors"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"study-gin-clerk/internal/model"
)

type UserProfileRepository struct {
	db *gorm.DB
}

func NewUserProfileRepository(db *gorm.DB) *UserProfileRepository {
	return &UserProfileRepository{db: db}
}

// GetProfile retrieves user profile by userID, returning a default profile if not found.
func (r *UserProfileRepository) GetProfile(ctx context.Context, userID string) (*model.UserProfile, error) {
	var profile model.UserProfile
	err := r.db.WithContext(ctx).Where("user_id = ?", userID).First(&profile).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return &model.UserProfile{UserID: userID, SystemPrompt: ""}, nil
		}
		return nil, err
	}
	return &profile, nil
}

// UpsertSystemPrompt updates or creates the user's system prompt.
func (r *UserProfileRepository) UpsertSystemPrompt(ctx context.Context, userID, systemPrompt string) (*model.UserProfile, error) {
	profile := &model.UserProfile{
		UserID:       userID,
		SystemPrompt: systemPrompt,
	}
	err := r.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "user_id"}},
		DoUpdates: clause.AssignmentColumns([]string{"system_prompt", "updated_at"}),
	}).Create(profile).Error
	if err != nil {
		return nil, err
	}
	return profile, nil
}

