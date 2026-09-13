package model

// AIModel represents a normalized AI model available from a provider.
type AIModel struct {
	ID          string   `json:"id"`          // e.g. "anthropic/claude-3.5-sonnet", "gpt-4o"
	Name        string   `json:"name"`        // e.g. "Anthropic: Claude 3.5 Sonnet"
	Provider    Provider `json:"provider"`    // ProviderOpenRouter, ProviderOpenAI, ProviderAnthropic, ProviderGoogle
	Description string   `json:"description"` // Brief description or tagline
	ContextLen  int      `json:"context_len"` // Context window length in tokens
}

