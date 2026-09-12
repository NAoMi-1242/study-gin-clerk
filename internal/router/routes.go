package router

import (
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

	engine.GET("/", func(c *gin.Context) {
		c.File("./web/index.html")
	})

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