//go:build wireinject
// +build wireinject

package main

import (
	"github.com/gin-gonic/gin"
	"github.com/google/wire"

	"study-gin-clerk/internal/config"
	"study-gin-clerk/internal/handler"
	"study-gin-clerk/internal/infra/ai"
	"study-gin-clerk/internal/infra/db"
	"study-gin-clerk/internal/infra/repository"
	"study-gin-clerk/internal/router"
	"study-gin-clerk/internal/service"
)

func InitializeApp(cfg config.Config) (*gin.Engine, func(), error) {
	wire.Build(
		db.Set,
		ai.Set,
		repository.Set,
		service.Set,
		handler.Set,
		router.Set,
	)
	return nil, nil, nil
}

