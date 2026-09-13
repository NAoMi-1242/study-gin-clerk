package profile

import (
	"context"
	"errors"
)

type Service struct {
	repo *Repository
}

func NewService(repo *Repository) *Service {
	return &Service{repo: repo}
}

// GetProfile retrieves the user's profile and system prompt, returning a default profile if not yet configured.
func (s *Service) GetProfile(ctx context.Context, userID string) (*Profile, error) {
	p, err := s.repo.GetByUserID(ctx, userID)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			// 未設定ユーザーにはデフォルトプロファイルを返却
			return &Profile{
				UserID:       userID,
				SystemPrompt: "",
			}, nil
		}
		return nil, err
	}
	return p, nil
}

// UpdateSystemPrompt updates and persists the user's shared system prompt.
func (s *Service) UpdateSystemPrompt(ctx context.Context, userID, systemPrompt string) (*Profile, error) {
	p := &Profile{
		UserID:       userID,
		SystemPrompt: systemPrompt,
	}
	if err := s.repo.Upsert(ctx, p); err != nil {
		return nil, err
	}
	return p, nil
}

