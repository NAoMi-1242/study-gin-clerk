//go:build wireinject
// +build wireinject

package main

import (
	"github.com/gin-gonic/gin"
	"github.com/google/wire"

	"study-gin-clerk/internal/ai"
	"study-gin-clerk/internal/config"
	"study-gin-clerk/internal/db"
	"study-gin-clerk/internal/handler"
	"study-gin-clerk/internal/repository"
	"study-gin-clerk/internal/router"
	"study-gin-clerk/internal/service"
)

func InitializeApp(cfg config.Config) (*gin.Engine, error) {
	wire.Build(
		db.Set,
		ai.Set,
		repository.Set,
		service.Set,
		handler.Set,
		router.Set,
	)
	return nil, nil
}

