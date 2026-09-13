package service

import (
	"context"

	"study-gin-clerk/internal/infra/repository"
	"study-gin-clerk/internal/model"
)

type UserService struct {
	userProfileRepo *repository.UserProfileRepository
}

func NewUserService(userProfileRepo *repository.UserProfileRepository) *UserService {
	return &UserService{userProfileRepo: userProfileRepo}
}

// GetProfile はユーザーの保存されたシステムプロンプト設定を取得します。
func (s *UserService) GetProfile(ctx context.Context, userID string) (*model.UserProfile, error) {
	return s.userProfileRepo.GetProfile(ctx, userID)
}

// UpdateSystemPrompt はユーザー共通のシステムプロンプトを更新・保存します。
func (s *UserService) UpdateSystemPrompt(ctx context.Context, userID, systemPrompt string) (*model.UserProfile, error) {
	return s.userProfileRepo.UpsertSystemPrompt(ctx, userID, systemPrompt)
}
