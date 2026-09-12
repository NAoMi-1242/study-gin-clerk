package db

import (
	"fmt"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"study-gin-clerk/internal/config"
	"study-gin-clerk/internal/model"
)

// NewDB initializes PostgreSQL connection via GORM and runs AutoMigrate.
func NewDB(cfg config.Config) (*gorm.DB, error) {
	database, err := gorm.Open(postgres.Open(cfg.DSN()), &gorm.Config{})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	sqlDB, err := database.DB()
	if err != nil {
		return nil, fmt.Errorf("failed to get sql.DB: %w", err)
	}

	sqlDB.SetMaxIdleConns(10)
	sqlDB.SetMaxOpenConns(100)
	sqlDB.SetConnMaxLifetime(time.Hour)

	if err := database.AutoMigrate(
		&model.Chat{},
		&model.Message{},
		&model.UserAPIKey{},
	); err != nil {
		return nil, fmt.Errorf("failed to auto migrate: %w", err)
	}

	// 以前の仕様で存在していた chats テーブルの provider / model カラムを安全に削除
	if database.Migrator().HasColumn("chats", "provider") {
		_ = database.Migrator().DropColumn("chats", "provider")
	}
	if database.Migrator().HasColumn("chats", "model") {
		_ = database.Migrator().DropColumn("chats", "model")
	}

	return database, nil
}

