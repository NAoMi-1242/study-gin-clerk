package repository

import (
	"context"
	"errors"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"study-gin-clerk/internal/model"
)

type UserAPIKeyRepository struct {
	db *gorm.DB
}

func NewUserAPIKeyRepository(db *gorm.DB) *UserAPIKeyRepository {
	return &UserAPIKeyRepository{db: db}
}

// Upsert creates or updates an API key for a user and provider.
func (r *UserAPIKeyRepository) Upsert(ctx context.Context, apiKey *model.UserAPIKey) error {
	return r.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "user_id"}, {Name: "provider"}},
		DoUpdates: clause.AssignmentColumns([]string{"encrypted_key", "key_hint", "updated_at"}),
	}).Create(apiKey).Error
}

// GetByProvider retrieves a user's API key for a given provider.
func (r *UserAPIKeyRepository) GetByProvider(ctx context.Context, userID string, provider model.Provider) (*model.UserAPIKey, error) {
	var key model.UserAPIKey
	if err := r.db.WithContext(ctx).
		Where("user_id = ? AND provider = ?", userID, provider).
		First(&key).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &key, nil
}

// ListByUserID retrieves all registered API keys for a user (without decrypted secret).
func (r *UserAPIKeyRepository) ListByUserID(ctx context.Context, userID string) ([]model.UserAPIKey, error) {
	var keys []model.UserAPIKey
	if err := r.db.WithContext(ctx).
		Where("user_id = ?", userID).
		Order("provider ASC").
		Find(&keys).Error; err != nil {
		return nil, err
	}
	return keys, nil
}

// DeleteByProvider deletes an API key for a user and provider.
func (r *UserAPIKeyRepository) DeleteByProvider(ctx context.Context, userID string, provider model.Provider) error {
	res := r.db.WithContext(ctx).
		Where("user_id = ? AND provider = ?", userID, provider).
		Delete(&model.UserAPIKey{})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

