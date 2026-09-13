package model

import (
	"fmt"
	"strings"
)

// Provider represents a supported AI service provider.
type Provider string

const (
	ProviderOpenRouter Provider = "openrouter"
	ProviderOpenAI     Provider = "openai"
	ProviderAnthropic  Provider = "anthropic"
	ProviderGoogle     Provider = "google"
)

// AllProviders lists all supported AI providers.
var AllProviders = []Provider{
	ProviderOpenRouter,
	ProviderOpenAI,
	ProviderAnthropic,
	ProviderGoogle,
}

// IsValid reports whether the provider is one of the supported providers.
func (p Provider) IsValid() bool {
	switch p {
	case ProviderOpenRouter, ProviderOpenAI, ProviderAnthropic, ProviderGoogle:
		return true
	default:
		return false
	}
}

// String returns the string representation of the provider.
func (p Provider) String() string {
	return string(p)
}

// ParseProvider converts a string into a validated Provider, case-insensitively.
func ParseProvider(s string) (Provider, error) {
	normalized := Provider(strings.ToLower(strings.TrimSpace(s)))
	if !normalized.IsValid() {
		return "", fmt.Errorf("unsupported provider '%s' (supported: %v)", s, AllProviders)
	}
	return normalized, nil
}
