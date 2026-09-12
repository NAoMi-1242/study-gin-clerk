package router

import (
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"

	_ "study-gin-clerk/docs"
	"study-gin-clerk/internal/handler"
	"study-gin-clerk/internal/middleware"
)

type Dependencies struct {
	HealthHandler *handler.HealthHandler
	UserHandler   *handler.UserHandler
	ChatHandler   *handler.ChatHandler
}

func New(deps Dependencies) *gin.Engine {
	engine := gin.Default()

	// CORS 設定 (フロントエンドからの通信を許可)
	engine.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"http://localhost:3000", "http://localhost:8080", "http://127.0.0.1:3000"},
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))

	engine.GET("/health", deps.HealthHandler.Get)
	engine.GET("/api/v1/health", deps.HealthHandler.Get)

	// Swagger UI
	engine.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	api := engine.Group("/api/v1")
	api.Use(middleware.ClerkAuthMiddleware())

	api.GET("/me", deps.UserHandler.GetMe)

	chats := api.Group("/chats")
	{
		chats.POST("", deps.ChatHandler.CreateChat)
		chats.GET("", deps.ChatHandler.ListChats)

		chats.GET("/:id", deps.ChatHandler.GetChat)
		
		chats.POST("/:id/messages", deps.ChatHandler.SendMessage)
	}

	return engine
}