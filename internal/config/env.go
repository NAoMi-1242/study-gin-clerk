package config

import (
	"encoding/hex"
	"fmt"
	"os"
	"strings"
)

type Config struct {
	Port           string
	AppURL         string
	ClerkSecretKey string
	DatabaseURL    string
	EncryptionKey  string
}

func Load() (Config, error) {
	cfg := Config{
		Port:           os.Getenv("PORT"),
		AppURL:         os.Getenv("APP_URL"),
		ClerkSecretKey: os.Getenv("CLERK_SECRET_KEY"),
		DatabaseURL:    os.Getenv("DATABASE_URL"),
		EncryptionKey:  os.Getenv("ENCRYPTION_KEY"),
	}

	if cfg.Port == "" {
		return Config{}, fmt.Errorf("PORT is required")
	}

	if cfg.AppURL == "" {
		return Config{}, fmt.Errorf("APP_URL is required")
	}

	if cfg.ClerkSecretKey == "" {
		return Config{}, fmt.Errorf("CLERK_SECRET_KEY is required")
	}

	if cfg.DatabaseURL == "" {
		return Config{}, fmt.Errorf("DATABASE_URL is required")
	}

	if cfg.EncryptionKey == "" {
		return Config{}, fmt.Errorf("ENCRYPTION_KEY is required")
	}

	// Fail-Fast: AES-256 に適合する32バイト長、または64文字の16進数文字列であることを検証
	trimmedKey := strings.TrimSpace(cfg.EncryptionKey)
	if len(trimmedKey) == 64 {
		if _, err := hex.DecodeString(trimmedKey); err != nil {
			return Config{}, fmt.Errorf("ENCRYPTION_KEY is 64 characters but contains invalid hex: %w", err)
		}
	} else if len(trimmedKey) != 32 {
		return Config{}, fmt.Errorf("ENCRYPTION_KEY must be exactly 32 bytes or 64 hex characters (got %d characters)", len(trimmedKey))
	}

	return cfg, nil
}

func (c Config) DSN() string {
	return c.DatabaseURL
}