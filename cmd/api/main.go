package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
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
	// 構造化ロガー (slog JSON) をデフォルト設定
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	slog.SetDefault(logger)

	cfg, err := config.Load()
	if err != nil {
		slog.Error("failed to load configuration", "error", err)
		os.Exit(1)
	}

	clerk.SetKey(cfg.ClerkSecretKey)

	// Wire による依存関係解決 (DB コネクションプール切断等の cleanup 関数を受け取る)
	engine, cleanup, err := InitializeApp(cfg)
	if err != nil {
		slog.Error("failed to initialize application", "error", err)
		os.Exit(1)
	}
	defer cleanup()

	server := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           engine,
		ReadHeaderTimeout: 5 * time.Second,
	}

	// SIGINT / SIGTERM によるシャットダウンシグナルを待機
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	go func() {
		slog.Info("server starting", "port", cfg.Port, "app_url", cfg.AppURL)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("server listen error", "error", err)
			os.Exit(1)
		}
	}()

	<-ctx.Done()
	slog.Info("shutdown signal received, draining requests gracefully...")

	// 最大 10 秒間、進行中のリクエストが完了するのを待機
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		slog.Error("server forced to shutdown", "error", err)
	} else {
		slog.Info("server shutdown completed successfully")
	}
}