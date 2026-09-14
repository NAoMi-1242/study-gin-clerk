package ai

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"study-gin-clerk/internal/types"
)

// ProviderFetcher defines the strategy for discovering and validating models for a specific AI provider.
type ProviderFetcher interface {
	FetchModels(ctx context.Context, apiKey string) ([]types.AIModel, error)
	ValidateKey(ctx context.Context, apiKey string) error
}

// ModelFetcher represents the registry that coordinates model fetching across providers.
type ModelFetcher interface {
	FetchModels(ctx context.Context, providerName types.Provider, apiKey string) ([]types.AIModel, error)
	ValidateKey(ctx context.Context, providerName types.Provider, apiKey string) error
}

// ModelRegistry dispatches model discovery requests to individual provider fetchers.
type ModelRegistry struct {
	fetchers map[types.Provider]ProviderFetcher
}

// NewModelRegistry creates a new ModelRegistry with standard HTTP-based provider fetchers.
func NewModelRegistry() *ModelRegistry {
	httpClient := &http.Client{
		Timeout: 20 * time.Second,
	}

	return &ModelRegistry{
		fetchers: map[types.Provider]ProviderFetcher{
			types.ProviderOpenRouter: newOpenRouterFetcher(httpClient),
			types.ProviderOpenAI:     newOpenAIFetcher(httpClient),
			types.ProviderAnthropic:  newAnthropicFetcher(httpClient),
			types.ProviderGoogle:     newGoogleFetcher(httpClient),
		},
	}
}

// ValidateKey tests whether the given API key is valid for the specified provider.
func (r *ModelRegistry) ValidateKey(ctx context.Context, providerName types.Provider, apiKey string) error {
	apiKey = strings.TrimSpace(apiKey)
	if apiKey == "" {
		return errors.New("API key is required")
	}

	fetcher, ok := r.fetchers[providerName]
	if !ok {
		return fmt.Errorf("unsupported provider: '%s'", providerName)
	}

	return fetcher.ValidateKey(ctx, apiKey)
}

// FetchModels dynamically fetches available models from the specified provider's API.
func (r *ModelRegistry) FetchModels(ctx context.Context, providerName types.Provider, apiKey string) ([]types.AIModel, error) {
	apiKey = strings.TrimSpace(apiKey)
	if apiKey == "" {
		return nil, errors.New("API key is required")
	}

	fetcher, ok := r.fetchers[providerName]
	if !ok {
		return nil, fmt.Errorf("unsupported provider: '%s'", providerName)
	}

	return fetcher.FetchModels(ctx, apiKey)
}

// --- Provider Fetcher Implementations (Strategy Pattern) ---

type openRouterFetcher struct {
	httpClient *http.Client
}

func newOpenRouterFetcher(c *http.Client) *openRouterFetcher {
	return &openRouterFetcher{httpClient: c}
}

func (f *openRouterFetcher) FetchModels(ctx context.Context, apiKey string) ([]types.AIModel, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://openrouter.ai/api/v1/models", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+apiKey)

	resp, err := f.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("openrouter request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("openrouter API error (%d): %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	var res struct {
		Data []struct {
			ID            string `json:"id"`
			Name          string `json:"name"`
			Description   string `json:"description"`
			ContextLength int    `json:"context_length"`
		} `json:"data"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return nil, fmt.Errorf("failed to parse openrouter response: %w", err)
	}

	models := make([]types.AIModel, 0, len(res.Data))
	for _, m := range res.Data {
		name := m.Name
		if name == "" {
			name = m.ID
		}
		models = append(models, types.AIModel{
			ID:          m.ID,
			Name:        name,
			Provider:    types.ProviderOpenRouter,
			Description: m.Description,
			ContextLen:  m.ContextLength,
		})
	}
	return models, nil
}

func (f *openRouterFetcher) ValidateKey(ctx context.Context, apiKey string) error {
	models, err := f.FetchModels(ctx, apiKey)
	if err != nil {
		return err
	}
	if len(models) == 0 {
		return fmt.Errorf("no models available for provider '%s'", types.ProviderOpenRouter)
	}
	return nil
}

type openAIFetcher struct {
	httpClient *http.Client
}

func newOpenAIFetcher(c *http.Client) *openAIFetcher {
	return &openAIFetcher{httpClient: c}
}

func (f *openAIFetcher) FetchModels(ctx context.Context, apiKey string) ([]types.AIModel, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://api.openai.com/v1/models", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+apiKey)

	resp, err := f.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("openai request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("openai API error (%d): %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	var res struct {
		Data []struct {
			ID string `json:"id"`
		} `json:"data"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return nil, fmt.Errorf("failed to parse openai response: %w", err)
	}

	var models []types.AIModel
	for _, m := range res.Data {
		if strings.HasPrefix(m.ID, "gpt-") ||
			strings.HasPrefix(m.ID, "o1") ||
			strings.HasPrefix(m.ID, "o3") ||
			strings.HasPrefix(m.ID, "chatgpt-") {
			models = append(models, types.AIModel{
				ID:       m.ID,
				Name:     m.ID,
				Provider: types.ProviderOpenAI,
			})
		}
	}
	return models, nil
}

func (f *openAIFetcher) ValidateKey(ctx context.Context, apiKey string) error {
	models, err := f.FetchModels(ctx, apiKey)
	if err != nil {
		return err
	}
	if len(models) == 0 {
		return fmt.Errorf("no models available for provider '%s'", types.ProviderOpenAI)
	}
	return nil
}

type anthropicFetcher struct {
	httpClient *http.Client
}

func newAnthropicFetcher(c *http.Client) *anthropicFetcher {
	return &anthropicFetcher{httpClient: c}
}

func (f *anthropicFetcher) FetchModels(ctx context.Context, apiKey string) ([]types.AIModel, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://api.anthropic.com/v1/models", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("x-api-key", apiKey)
	req.Header.Set("anthropic-version", "2023-06-01")

	resp, err := f.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("anthropic request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("anthropic API error (%d): %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	var res struct {
		Data []struct {
			ID          string `json:"id"`
			DisplayName string `json:"display_name"`
		} `json:"data"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return nil, fmt.Errorf("failed to parse anthropic response: %w", err)
	}

	models := make([]types.AIModel, 0, len(res.Data))
	for _, m := range res.Data {
		name := m.DisplayName
		if name == "" {
			name = m.ID
		}
		models = append(models, types.AIModel{
			ID:       m.ID,
			Name:     name,
			Provider: types.ProviderAnthropic,
		})
	}
	return models, nil
}

func (f *anthropicFetcher) ValidateKey(ctx context.Context, apiKey string) error {
	models, err := f.FetchModels(ctx, apiKey)
	if err != nil {
		return err
	}
	if len(models) == 0 {
		return fmt.Errorf("no models available for provider '%s'", types.ProviderAnthropic)
	}
	return nil
}

type googleFetcher struct {
	httpClient *http.Client
}

func newGoogleFetcher(c *http.Client) *googleFetcher {
	return &googleFetcher{httpClient: c}
}

func (f *googleFetcher) FetchModels(ctx context.Context, apiKey string) ([]types.AIModel, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://generativelanguage.googleapis.com/v1beta/models", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("x-goog-api-key", apiKey)

	resp, err := f.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("google request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("google API error (%d): %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	var res struct {
		Models []struct {
			Name                       string   `json:"name"`
			DisplayName                string   `json:"displayName"`
			Description                string   `json:"description"`
			InputTokenLimit            int      `json:"inputTokenLimit"`
			SupportedGenerationMethods []string `json:"supportedGenerationMethods"`
		} `json:"models"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return nil, fmt.Errorf("failed to parse google response: %w", err)
	}

	var models []types.AIModel
	for _, m := range res.Models {
		supportsGenerate := false
		for _, method := range m.SupportedGenerationMethods {
			if method == "generateContent" {
				supportsGenerate = true
				break
			}
		}
		if !supportsGenerate {
			continue
		}

		cleanID := strings.TrimPrefix(m.Name, "models/")
		displayName := m.DisplayName
		if displayName == "" {
			displayName = cleanID
		}

		models = append(models, types.AIModel{
			ID:          cleanID,
			Name:        displayName,
			Provider:    types.ProviderGoogle,
			Description: m.Description,
			ContextLen:  m.InputTokenLimit,
		})
	}
	return models, nil
}

func (f *googleFetcher) ValidateKey(ctx context.Context, apiKey string) error {
	models, err := f.FetchModels(ctx, apiKey)
	if err != nil {
		return err
	}
	if len(models) == 0 {
		return fmt.Errorf("no models available for provider '%s'", types.ProviderGoogle)
	}
	return nil
}
