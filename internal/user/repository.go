package user

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

// GetProfile retrieves user profile by userID, returning ErrNotFound if missing.
func (r *Repository) GetProfile(ctx context.Context, userID string) (*Profile, error) {
	var p Profile
	err := r.db.WithContext(ctx).Where("user_id = ?", userID).First(&p).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &p, nil
}

// UpsertProfile updates or creates the user's profile.
func (r *Repository) UpsertProfile(ctx context.Context, p *Profile) error {
	return r.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "user_id"}},
		DoUpdates: clause.AssignmentColumns([]string{"system_prompt", "updated_at"}),
	}).Create(p).Error
}

// UpsertKey creates or updates an API key for a user and provider.
func (r *Repository) UpsertKey(ctx context.Context, key *Key) error {
	return r.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "user_id"}, {Name: "provider"}},
		DoUpdates: clause.AssignmentColumns([]string{"encrypted_key", "key_hint", "updated_at"}),
	}).Create(key).Error
}

// GetKeyByProvider retrieves a user's API key for a given provider, returning ErrNotFound if missing.
func (r *Repository) GetKeyByProvider(ctx context.Context, userID string, provider types.Provider) (*Key, error) {
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

// ListKeysByUserID retrieves all registered API keys for a user (without decrypted secret).
func (r *Repository) ListKeysByUserID(ctx context.Context, userID string) ([]Key, error) {
	var keys []Key
	if err := r.db.WithContext(ctx).
		Where("user_id = ?", userID).
		Order("provider ASC").
		Find(&keys).Error; err != nil {
		return nil, err
	}
	return keys, nil
}

// DeleteKeyByProvider deletes an API key for a user and provider, returning ErrNotFound if not present.
func (r *Repository) DeleteKeyByProvider(ctx context.Context, userID string, provider types.Provider) error {
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

