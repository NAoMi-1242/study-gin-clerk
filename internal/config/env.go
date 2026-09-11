package config

import (
    "fmt"
    "os"
)

type Config struct {
    Port           string
    ClerkSecretKey string
}

func Load() (Config, error) {
    cfg := Config{
        Port:           getEnv("PORT", "8080"),
        ClerkSecretKey: os.Getenv("CLERK_SECRET_KEY"),
    }

    if cfg.ClerkSecretKey == "" {
        return Config{}, fmt.Errorf("CLERK_SECRET_KEY is required")
    }

    return cfg, nil
}

func getEnv(key, defaultValue string) string {
    value := os.Getenv(key)
    if value == "" {
        return defaultValue
    }

    return value
}