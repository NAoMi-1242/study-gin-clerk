package ai

// ModelInfo represents a normalized AI model available from a provider.
type ModelInfo struct {
	ID          string `json:"id"`           // e.g. "anthropic/claude-3.5-sonnet", "gpt-4o"
	Name        string `json:"name"`         // e.g. "Anthropic: Claude 3.5 Sonnet"
	Provider    string `json:"provider"`     // "openrouter", "openai", "anthropic", "google"
	Description string `json:"description"`  // Brief description or tagline
	ContextLen  int    `json:"context_len"`  // Context window length in tokens
}

