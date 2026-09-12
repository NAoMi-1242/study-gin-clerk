package config

import (
    "fmt"
    "os"
)

type Config struct {
	Port           string
	ClerkSecretKey string

	DatabaseURL string

	DBHost     string
	DBPort     string
	DBUser     string
	DBPassword string
	DBName     string
	DBSSLMode  string
}

func Load() (Config, error) {
	cfg := Config{
		Port:           getEnv("PORT", "8080"),
		ClerkSecretKey: os.Getenv("CLERK_SECRET_KEY"),

		DatabaseURL: os.Getenv("DATABASE_URL"),

		DBHost:     getEnv("DB_HOST", "localhost"),
		DBPort:     getEnv("DB_PORT", "5432"),
		DBUser:     getEnv("DB_USER", "postgres"),
		DBPassword: os.Getenv("DB_PASSWORD"),
		DBName:     getEnv("DB_NAME", "study_app"),
		DBSSLMode:  getEnv("DB_SSLMODE", "disable"),
	}

	if cfg.ClerkSecretKey == "" {
		return Config{}, fmt.Errorf("CLERK_SECRET_KEY is required")
	}

	// DATABASE_URL が未指定の場合は、DB_PASSWORD を必須とする (Fail Fast)
	if cfg.DatabaseURL == "" && cfg.DBPassword == "" {
		return Config{}, fmt.Errorf("DB_PASSWORD is required (or set DATABASE_URL)")
	}

	return cfg, nil
}

func (c Config) DSN() string {
	if c.DatabaseURL != "" {
		return c.DatabaseURL
	}
	return fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=%s TimeZone=UTC",
		c.DBHost, c.DBPort, c.DBUser, c.DBPassword, c.DBName, c.DBSSLMode,
	)
}

func getEnv(key, defaultValue string) string {
    value := os.Getenv(key)
    if value == "" {
        return defaultValue
    }

    return value
}