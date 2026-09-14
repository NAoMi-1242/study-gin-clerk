package config

import (
	"os"
	"testing"
)

func TestLoad_Success(t *testing.T) {
	// Set required environment variables
	t.Setenv("PORT", "8080")
	t.Setenv("APP_URL", "http://localhost:3000")
	t.Setenv("CLERK_SECRET_KEY", "sk_test_123456789")
	t.Setenv("DATABASE_URL", "postgres://user:pass@localhost:5432/db")
	t.Setenv("ENCRYPTION_KEY", "12345678901234567890123456789012")
	t.Setenv("CORS_ALLOWED_ORIGINS", "http://localhost:3000, http://localhost:5173")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.Port != "8080" {
		t.Errorf("got port %q, want 8080", cfg.Port)
	}
	if cfg.AppURL != "http://localhost:3000" {
		t.Errorf("got app_url %q, want http://localhost:3000", cfg.AppURL)
	}
	if len(cfg.CORSAllowedOrigins) != 2 {
		t.Fatalf("expected 2 CORS origins, got %d", len(cfg.CORSAllowedOrigins))
	}
	if cfg.CORSAllowedOrigins[0] != "http://localhost:3000" || cfg.CORSAllowedOrigins[1] != "http://localhost:5173" {
		t.Errorf("unexpected CORS origins: %v", cfg.CORSAllowedOrigins)
	}
}

func TestLoad_CORSFallbackToAppURL(t *testing.T) {
	t.Setenv("PORT", "8080")
	t.Setenv("APP_URL", "http://localhost:3000")
	t.Setenv("CLERK_SECRET_KEY", "sk_test_123456789")
	t.Setenv("DATABASE_URL", "postgres://user:pass@localhost:5432/db")
	t.Setenv("ENCRYPTION_KEY", "12345678901234567890123456789012")
	_ = os.Unsetenv("CORS_ALLOWED_ORIGINS")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(cfg.CORSAllowedOrigins) != 1 || cfg.CORSAllowedOrigins[0] != "http://localhost:3000" {
		t.Errorf("expected CORS origins to fallback to [http://localhost:3000], got %v", cfg.CORSAllowedOrigins)
	}
}

func TestLoad_MissingRequiredEnv(t *testing.T) {
	tests := []struct {
		name    string
		unset   string
		setup   func()
	}{
		{
			name:  "missing PORT",
			unset: "PORT",
		},
		{
			name:  "missing APP_URL",
			unset: "APP_URL",
		},
		{
			name:  "missing CLERK_SECRET_KEY",
			unset: "CLERK_SECRET_KEY",
		},
		{
			name:  "missing DATABASE_URL",
			unset: "DATABASE_URL",
		},
		{
			name:  "missing ENCRYPTION_KEY",
			unset: "ENCRYPTION_KEY",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("PORT", "8080")
			t.Setenv("APP_URL", "http://localhost:3000")
			t.Setenv("CLERK_SECRET_KEY", "sk_test_123456789")
			t.Setenv("DATABASE_URL", "postgres://user:pass@localhost:5432/db")
			t.Setenv("ENCRYPTION_KEY", "12345678901234567890123456789012")

			_ = os.Unsetenv(tt.unset)

			_, err := Load()
			if err == nil {
				t.Errorf("expected error when %s is unset, got nil", tt.unset)
			}
		})
	}
}

func TestLoad_InvalidEncryptionKey(t *testing.T) {
	t.Setenv("PORT", "8080")
	t.Setenv("APP_URL", "http://localhost:3000")
	t.Setenv("CLERK_SECRET_KEY", "sk_test_123456789")
	t.Setenv("DATABASE_URL", "postgres://user:pass@localhost:5432/db")
	t.Setenv("ENCRYPTION_KEY", "short-key")

	_, err := Load()
	if err == nil {
		t.Errorf("expected error for short encryption key, got nil")
	}
}

