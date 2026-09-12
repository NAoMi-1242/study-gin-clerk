package main

import (
    "log"
    "net/http"
    "time"

    "github.com/clerk/clerk-sdk-go/v2"

    "study-gin-clerk/internal/config"
)

// @title Study Gin Clerk & AI Chat API
// @version 1.0
// @description Go + Gin + Clerk + GORM による AI チャット基盤 API
// @host localhost:8080
// @BasePath /

// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
// @description Clerk から取得した JWT トークン (Bearer <TOKEN>)
func main() {
    cfg, err := config.Load()
    if err != nil {
        log.Fatal(err)
    }

    clerk.SetKey(cfg.ClerkSecretKey)

    engine, err := InitializeApp(cfg)
    if err != nil {
        log.Fatal(err)
    }

    server := &http.Server{
        Addr:              ":" + cfg.Port,
        Handler:           engine,
        ReadHeaderTimeout: 5 * time.Second,
    }

    log.Printf("server listening on :%s", cfg.Port)

    if err := server.ListenAndServe(); err != nil &&
        err != http.ErrServerClosed {
        log.Fatal(err)
    }
}