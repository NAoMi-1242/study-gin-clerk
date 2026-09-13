package router

import (
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"

	_ "study-gin-clerk/docs"
	"study-gin-clerk/internal/config"
	"study-gin-clerk/internal/handler"
	"study-gin-clerk/internal/middleware"
)

type Dependencies struct {
	Config            config.Config
	HealthHandler     *handler.HealthHandler
	UserHandler       *handler.UserHandler
	ChatHandler       *handler.ChatHandler
	UserAPIKeyHandler *handler.UserAPIKeyHandler
	AIHandler         *handler.AIHandler
}

func New(deps Dependencies) *gin.Engine {
	engine := gin.Default()

	// CORS 設定 (フロントエンドからの通信を許可)
	allowedOrigins := []string{deps.Config.AppURL}
	if deps.Config.AppURL != "http://localhost:8080" {
		allowedOrigins = append(allowedOrigins, "http://localhost:8080")
	}
	if deps.Config.AppURL == "http://localhost:3000" {
		allowedOrigins = append(allowedOrigins, "http://127.0.0.1:3000")
	}

	engine.Use(cors.New(cors.Config{
		AllowOrigins:     allowedOrigins,
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))

	engine.GET("/health", deps.HealthHandler.Get)

	// Swagger UI
	engine.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	api := engine.Group("/api/v1")
	api.Use(middleware.ClerkAuthMiddleware())

	api.GET("/me", deps.UserHandler.GetMe)
	api.PUT("/me/system-prompt", deps.UserHandler.UpdateSystemPrompt)

	// AI 関連エンドポイント
	api.GET("/ai/models", deps.AIHandler.ListModels)

	// ユーザー API キー管理エンドポイント
	apiKeys := api.Group("/user/api-keys")
	{
		apiKeys.POST("", deps.UserAPIKeyHandler.RegisterKey)
		apiKeys.GET("", deps.UserAPIKeyHandler.ListKeys)
		apiKeys.DELETE("/:provider", deps.UserAPIKeyHandler.DeleteKey)
	}

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