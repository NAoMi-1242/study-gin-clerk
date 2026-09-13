package apikey

import (
	"context"
	"errors"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"study-gin-clerk/internal/types"
)

type Repository struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) *Repository {
	return &Repository{db: db}
}

// Upsert creates or updates an API key for a user and provider.
func (r *Repository) Upsert(ctx context.Context, key *Key) error {
	return r.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "user_id"}, {Name: "provider"}},
		DoUpdates: clause.AssignmentColumns([]string{"encrypted_key", "key_hint", "updated_at"}),
	}).Create(key).Error
}

// GetByProvider retrieves a user's API key for a given provider, returning ErrNotFound if missing.
func (r *Repository) GetByProvider(ctx context.Context, userID string, provider types.Provider) (*Key, error) {
	var key Key
	if err := r.db.WithContext(ctx).
		Where("user_id = ? AND provider = ?", userID, provider).
		First(&key).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &key, nil
}

// ListByUserID retrieves all registered API keys for a user (without decrypted secret).
func (r *Repository) ListByUserID(ctx context.Context, userID string) ([]Key, error) {
	var keys []Key
	if err := r.db.WithContext(ctx).
		Where("user_id = ?", userID).
		Order("provider ASC").
		Find(&keys).Error; err != nil {
		return nil, err
	}
	return keys, nil
}

// DeleteByProvider deletes an API key for a user and provider, returning ErrNotFound if not present.
func (r *Repository) DeleteByProvider(ctx context.Context, userID string, provider types.Provider) error {
	res := r.db.WithContext(ctx).
		Where("user_id = ? AND provider = ?", userID, provider).
		Delete(&Key{})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}
