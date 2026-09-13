package profile

import (
	"context"
	"errors"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type Repository struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) *Repository {
	return &Repository{db: db}
}

// GetByUserID retrieves user profile by userID from DB, returning ErrNotFound if missing.
func (r *Repository) GetByUserID(ctx context.Context, userID string) (*Profile, error) {
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

// Upsert updates or creates the user's profile.
func (r *Repository) Upsert(ctx context.Context, p *Profile) error {
	return r.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "user_id"}},
		DoUpdates: clause.AssignmentColumns([]string{"system_prompt", "updated_at"}),
	}).Create(p).Error
}
