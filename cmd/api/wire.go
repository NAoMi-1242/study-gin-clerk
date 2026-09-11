//go:build wireinject
// +build wireinject

package main

import (
	"github.com/gin-gonic/gin"
	"github.com/google/wire"

	"study-gin-clerk/internal/handler"
	"study-gin-clerk/internal/router"
)

func InitializeApp() (*gin.Engine, error) {
	wire.Build(
		handler.Set,
		router.Set,
	)
	return nil, nil
}

