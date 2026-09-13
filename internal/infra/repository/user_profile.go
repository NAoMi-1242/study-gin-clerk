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

// GetByUserID retrieves user profile by userID, returning a default profile if not found.
func (r *UserProfileRepository) GetByUserID(ctx context.Context, userID string) (*model.UserProfile, error) {
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

// Upsert updates or creates the user's profile.
func (r *UserProfileRepository) Upsert(ctx context.Context, profile *model.UserProfile) error {
	return r.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "user_id"}},
		DoUpdates: clause.AssignmentColumns([]string{"system_prompt", "updated_at"}),
	}).Create(profile).Error
}
