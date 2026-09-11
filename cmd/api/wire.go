//go:build wireinject
// +build wireinject

package main

import (
	"github.com/gin-gonic/gin"
	"github.com/google/wire"

	"study-gin-clerk/internal/handler"
	"study-gin-clerk/internal/router"
	"study-gin-clerk/internal/service"
)

func InitializeApp() (*gin.Engine, error) {
	wire.Build(
		service.Set,
		handler.Set,
		router.Set,
	)
	return nil, nil
}

