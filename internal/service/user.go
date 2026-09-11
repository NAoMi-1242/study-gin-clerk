package service

import "context"

type UserProfile struct {
	UserID string `json:"user_id"`
	Plan   string `json:"plan"`
	Status string `json:"status"`
}

type UserService struct{}

func NewUserService() *UserService {
	return &UserService{}
}

// GetProfile はユーザーの業務情報（プランやステータスなど）を取得します。
// （将来はここで GORM / Supabase などの DB を参照します）
func (s *UserService) GetProfile(ctx context.Context, userID string) (*UserProfile, error) {
	return &UserProfile{
		UserID: userID,
		Plan:   "free",
		Status: "active",
	}, nil
}

