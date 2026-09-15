package router

import (
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"

	_ "study-gin-clerk/docs"
	"study-gin-clerk/internal/auth"
	"study-gin-clerk/internal/chat"
	"study-gin-clerk/internal/config"
	"study-gin-clerk/internal/health"
	"study-gin-clerk/internal/user"
)

type Dependencies struct {
	Config        config.Config
	HealthHandler *health.Handler
	UserHandler   *user.Handler
	ChatHandler   *chat.Handler
}

func New(deps Dependencies) *gin.Engine {
	engine := gin.Default()

	// CORS 設定 (フロントエンドからの通信を許可)
	allowedOrigins := deps.Config.CORSAllowedOrigins
	if len(allowedOrigins) == 0 && deps.Config.AppURL != "" {
		allowedOrigins = []string{deps.Config.AppURL}
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
	api.Use(auth.RequireAuth())

	// ユーザープロファイル・設定関連エンドポイント
	me := api.Group("/me")
	{
		me.GET("", deps.UserHandler.GetMe)
		me.PUT("/system-prompt", deps.UserHandler.UpdateSystemPrompt)

		apiKeys := me.Group("/api-keys")
		{
			apiKeys.POST("", deps.UserHandler.RegisterKey)
			apiKeys.GET("", deps.UserHandler.ListKeys)
			apiKeys.DELETE("/:provider", deps.UserHandler.DeleteKey)
		}
	}

	// AI モデル一覧
	api.GET("/ai/models", deps.ChatHandler.ListModels)

	// チャット・メッセージ
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
