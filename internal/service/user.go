package service

import (
	"context"

	"study-gin-clerk/internal/infra/repository"
)

type UserProfile struct {
	UserID       string `json:"user_id"`
	Plan         string `json:"plan"`
	Status       string `json:"status"`
	SystemPrompt string `json:"system_prompt"`
}

type UserService struct {
	userProfileRepo *repository.UserProfileRepository
}

func NewUserService(userProfileRepo *repository.UserProfileRepository) *UserService {
	return &UserService{userProfileRepo: userProfileRepo}
}

// GetProfile はユーザーの業務情報および保存されたシステムプロンプトを取得します。
func (s *UserService) GetProfile(ctx context.Context, userID string) (*UserProfile, error) {
	profile, err := s.userProfileRepo.GetProfile(ctx, userID)
	if err != nil {
		return nil, err
	}
	return &UserProfile{
		UserID:       userID,
		Plan:         "free",
		Status:       "active",
		SystemPrompt: profile.SystemPrompt,
	}, nil
}

// UpdateSystemPrompt はユーザー共通のシステムプロンプトを更新・保存します。
func (s *UserService) UpdateSystemPrompt(ctx context.Context, userID, systemPrompt string) (*UserProfile, error) {
	profile, err := s.userProfileRepo.UpsertSystemPrompt(ctx, userID, systemPrompt)
	if err != nil {
		return nil, err
	}
	return &UserProfile{
		UserID:       userID,
		Plan:         "free",
		Status:       "active",
		SystemPrompt: profile.SystemPrompt,
	}, nil
}

