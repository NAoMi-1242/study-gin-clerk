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

	"study-gin-clerk/internal/model"
)

type ModelRegistry struct {
	httpClient *http.Client
}

func NewModelRegistry() *ModelRegistry {
	return &ModelRegistry{
		httpClient: &http.Client{
			Timeout: 20 * time.Second,
		},
	}
}

// ValidateKey tests whether the given API key is valid for the specified provider by making a probe call.
func (r *ModelRegistry) ValidateKey(ctx context.Context, providerName model.Provider, apiKey string) error {
	models, err := r.FetchModels(ctx, providerName, apiKey)
	if err != nil {
		return err
	}
	if len(models) == 0 {
		return fmt.Errorf("no models available for provider '%s'", providerName)
	}
	return nil
}

// FetchModels dynamically fetches available models from the provider's API.
func (r *ModelRegistry) FetchModels(ctx context.Context, providerName model.Provider, apiKey string) ([]model.AIModel, error) {
	apiKey = strings.TrimSpace(apiKey)
	if apiKey == "" {
		return nil, errors.New("API key is required")
	}

	switch providerName {
	case model.ProviderOpenRouter:
		return r.fetchOpenRouterModels(ctx, apiKey)
	case model.ProviderOpenAI:
		return r.fetchOpenAIModels(ctx, apiKey)
	case model.ProviderAnthropic:
		return r.fetchAnthropicModels(ctx, apiKey)
	case model.ProviderGoogle:
		return r.fetchGoogleModels(ctx, apiKey)
	default:
		return nil, fmt.Errorf("unsupported provider: '%s'", providerName)
	}
}

func (r *ModelRegistry) fetchOpenRouterModels(ctx context.Context, apiKey string) ([]model.AIModel, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://openrouter.ai/api/v1/models", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+apiKey)

	resp, err := r.httpClient.Do(req)
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

	models := make([]model.AIModel, 0, len(res.Data))
	for _, m := range res.Data {
		name := m.Name
		if name == "" {
			name = m.ID
		}
		models = append(models, model.AIModel{
			ID:          m.ID,
			Name:        name,
			Provider:    model.ProviderOpenRouter,
			Description: m.Description,
			ContextLen:  m.ContextLength,
		})
	}
	return models, nil
}

func (r *ModelRegistry) fetchOpenAIModels(ctx context.Context, apiKey string) ([]model.AIModel, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://api.openai.com/v1/models", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+apiKey)

	resp, err := r.httpClient.Do(req)
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

	var models []model.AIModel
	for _, m := range res.Data {
		// Filter for chat completion models only (gpt-*, o1*, o3*, chatgpt-*)
		if strings.HasPrefix(m.ID, "gpt-") ||
			strings.HasPrefix(m.ID, "o1") ||
			strings.HasPrefix(m.ID, "o3") ||
			strings.HasPrefix(m.ID, "chatgpt-") {
			models = append(models, model.AIModel{
				ID:       m.ID,
				Name:     m.ID,
				Provider: model.ProviderOpenAI,
			})
		}
	}
	return models, nil
}

func (r *ModelRegistry) fetchAnthropicModels(ctx context.Context, apiKey string) ([]model.AIModel, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://api.anthropic.com/v1/models", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("x-api-key", apiKey)
	req.Header.Set("anthropic-version", "2023-06-01")

	resp, err := r.httpClient.Do(req)
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

	models := make([]model.AIModel, 0, len(res.Data))
	for _, m := range res.Data {
		name := m.DisplayName
		if name == "" {
			name = m.ID
		}
		models = append(models, model.AIModel{
			ID:       m.ID,
			Name:     name,
			Provider: model.ProviderAnthropic,
		})
	}
	return models, nil
}

func (r *ModelRegistry) fetchGoogleModels(ctx context.Context, apiKey string) ([]model.AIModel, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://generativelanguage.googleapis.com/v1beta/models", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("x-goog-api-key", apiKey)

	resp, err := r.httpClient.Do(req)
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

	var models []model.AIModel
	for _, m := range res.Models {
		// Filter for generateContent support
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

		models = append(models, model.AIModel{
			ID:          cleanID,
			Name:        displayName,
			Provider:    model.ProviderGoogle,
			Description: m.Description,
			ContextLen:  m.InputTokenLimit,
		})
	}
	return models, nil
}
