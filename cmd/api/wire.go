//go:build wireinject
// +build wireinject

package main

import (
	"github.com/gin-gonic/gin"
	"github.com/google/wire"

	"study-gin-clerk/internal/aimodel"
	"study-gin-clerk/internal/apikey"
	"study-gin-clerk/internal/chat"
	"study-gin-clerk/internal/config"
	"study-gin-clerk/internal/health"
	"study-gin-clerk/internal/infra/ai"
	"study-gin-clerk/internal/infra/crypto"
	"study-gin-clerk/internal/infra/db"
	"study-gin-clerk/internal/profile"
	"study-gin-clerk/internal/router"
)

func InitializeApp(cfg config.Config) (*gin.Engine, func(), error) {
	wire.Build(
		db.Set,
		crypto.Set,
		ai.Set,
		health.Set,
		profile.Set,
		apikey.Set,
		aimodel.Set,
		chat.Set,
		router.Set,
	)
	return nil, nil, nil
}
