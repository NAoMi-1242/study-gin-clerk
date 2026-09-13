package chat

import (
	"github.com/google/wire"

	"study-gin-clerk/internal/apikey"
	"study-gin-clerk/internal/infra/ai"
	"study-gin-clerk/internal/profile"
)

// ProvideService adapts concrete apikey.Service and profile.Service to ChatService interfaces for Wire.
func ProvideService(
	repo *Repository,
	keyService *apikey.Service,
	aiClient *ai.Client,
	profileService *profile.Service,
) *Service {
	return NewService(repo, keyService, aiClient, profileService)
}

var Set = wire.NewSet(
	NewRepository,
	ProvideService,
	NewHandler,
)
