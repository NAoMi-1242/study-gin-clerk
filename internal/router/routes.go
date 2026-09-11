package router

import (
	"github.com/gin-gonic/gin"

	"study-gin-clerk/internal/handler"
	"study-gin-clerk/internal/middleware"
)

type Dependencies struct {
	HealthHandler *handler.HealthHandler
	UserHandler   *handler.UserHandler
}

func New(deps Dependencies) *gin.Engine {
    engine := gin.Default()

    engine.GET("/", func(c *gin.Context) {
        c.File("./web/index.html")
    })

    engine.GET("/health", deps.HealthHandler.Get)

    api := engine.Group("/api/v1")
    api.Use(middleware.ClerkAuthMiddleware())

    api.GET("/me", deps.UserHandler.GetMe)

    return engine
}