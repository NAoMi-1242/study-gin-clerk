package service

import (
	"context"

	"study-gin-clerk/internal/infra/repository"
	"study-gin-clerk/internal/model"
)

type UserProfileService struct {
	userProfileRepo *repository.UserProfileRepository
}

func NewUserProfileService(userProfileRepo *repository.UserProfileRepository) *UserProfileService {
	return &UserProfileService{userProfileRepo: userProfileRepo}
}

// GetProfile はユーザーの保存されたシステムプロンプト設定を取得します。
func (s *UserProfileService) GetProfile(ctx context.Context, userID string) (*model.UserProfile, error) {
	return s.userProfileRepo.GetProfile(ctx, userID)
}

// UpdateSystemPrompt はユーザー共通のシステムプロンプトを更新・保存します。
func (s *UserProfileService) UpdateSystemPrompt(ctx context.Context, userID, systemPrompt string) (*model.UserProfile, error) {
	return s.userProfileRepo.UpsertSystemPrompt(ctx, userID, systemPrompt)
}

