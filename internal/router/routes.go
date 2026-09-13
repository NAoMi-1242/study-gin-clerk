package router

import (
	"strings"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"

	_ "study-gin-clerk/docs"
	"study-gin-clerk/internal/aimodel"
	"study-gin-clerk/internal/apikey"
	"study-gin-clerk/internal/chat"
	"study-gin-clerk/internal/config"
	"study-gin-clerk/internal/health"
	"study-gin-clerk/internal/middleware"
	"study-gin-clerk/internal/profile"
)

type Dependencies struct {
	Config         config.Config
	HealthHandler  *health.Handler
	ProfileHandler *profile.Handler
	APIKeyHandler  *apikey.Handler
	AIModelHandler *aimodel.Handler
	ChatHandler    *chat.Handler
}

func New(deps Dependencies) *gin.Engine {
	engine := gin.Default()

	// CORS 設定 (フロントエンドからの通信を許可)
	var allowedOrigins []string
	for _, origin := range strings.Split(deps.Config.AppURL, ",") {
		if trimmed := strings.TrimSpace(origin); trimmed != "" {
			allowedOrigins = append(allowedOrigins, trimmed)
		}
	}

	engine.Use(cors.New(cors.Config{
		AllowOrigins:     allowedOrigins,
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "HEAD", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization", "X-Requested-With"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))

	engine.GET("/health", deps.HealthHandler.Get)

	// Swagger UI
	engine.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	api := engine.Group("/api/v1")
	api.Use(middleware.ClerkAuthMiddleware())

	// 自身のアカウント・設定関連エンドポイント
	me := api.Group("/me")
	{
		me.GET("", deps.ProfileHandler.GetMe)
		me.PUT("/system-prompt", deps.ProfileHandler.UpdateSystemPrompt)

		apiKeys := me.Group("/api-keys")
		{
			apiKeys.POST("", deps.APIKeyHandler.RegisterKey)
			apiKeys.GET("", deps.APIKeyHandler.ListKeys)
			apiKeys.DELETE("/:provider", deps.APIKeyHandler.DeleteKey)
		}
	}

	// AI モデル関連エンドポイント
	api.GET("/ai/models", deps.AIModelHandler.ListModels)

	chats := api.Group("/chats")
	{
		chats.POST("", deps.ChatHandler.CreateChat)
		chats.GET("", deps.ChatHandler.ListChats)

		chats.GET("/:id", deps.ChatHandler.GetChat)

		chats.POST("/:id/messages", deps.ChatHandler.SendMessage)
		chats.POST("/:id/messages/stream", deps.ChatHandler.StreamMessage)
	}

	return engine
}
