package ai

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/zendev-sh/goai"
	"github.com/zendev-sh/goai/provider"
	"github.com/zendev-sh/goai/provider/anthropic"
	"github.com/zendev-sh/goai/provider/google"
	"github.com/zendev-sh/goai/provider/openai"
	"github.com/zendev-sh/goai/provider/openrouter"

	"study-gin-clerk/internal/config"
	"study-gin-clerk/internal/types"
)

// ChatMessage represents a generic message passed to the AI client.
type ChatMessage struct {
	Role    string
	Content string
}

type Client struct {
	cfg config.Config
}

func NewClient(cfg config.Config) *Client {
	return &Client{cfg: cfg}
}

// CreateModel builds the appropriate GoAI provider.LanguageModel.
func (c *Client) CreateModel(providerName types.Provider, modelID, apiKey string) (provider.LanguageModel, error) {
	modelID = strings.TrimSpace(modelID)
	apiKey = strings.TrimSpace(apiKey)

	if apiKey == "" {
		return nil, fmt.Errorf("API key is required for provider '%s'", providerName)
	}
	if modelID == "" {
		return nil, fmt.Errorf("model ID is required for provider '%s'", providerName)
	}

	switch providerName {
	case types.ProviderOpenRouter:
		referer := c.cfg.AppURL
		if referer == "" {
			referer = "http://localhost:3000"
		}
		return openrouter.Chat(
			modelID,
			openrouter.WithAPIKey(apiKey),
			openrouter.WithHeaders(map[string]string{
				"HTTP-Referer": referer,
				"X-Title":      "Study Gin Clerk",
			}),
		), nil
	case types.ProviderOpenAI:
		return openai.Chat(modelID, openai.WithAPIKey(apiKey)), nil
	case types.ProviderAnthropic:
		return anthropic.Chat(modelID, anthropic.WithAPIKey(apiKey)), nil
	case types.ProviderGoogle:
		return google.Chat(modelID, google.WithAPIKey(apiKey)), nil
	default:
		return nil, fmt.Errorf("unsupported AI provider: '%s'", providerName)
	}
}

// buildMessages converts system prompt, past DB messages, and the new user prompt into GoAI provider.Message slice.
func (c *Client) buildMessages(systemPrompt string, history []ChatMessage, prompt string) []provider.Message {
	var msgs []provider.Message
	if trimmed := strings.TrimSpace(systemPrompt); trimmed != "" {
		msgs = append(msgs, goai.SystemMessage(trimmed))
	}
	for _, m := range history {
		switch m.Role {
		case "user":
			msgs = append(msgs, goai.UserMessage(m.Content))
		case "assistant":
			msgs = append(msgs, goai.AssistantMessage(m.Content))
		case "system":
			msgs = append(msgs, goai.SystemMessage(m.Content))
		}
	}
	if prompt != "" {
		msgs = append(msgs, goai.UserMessage(prompt))
	}
	return msgs
}

func formatAIError(providerName types.Provider, modelID string, err error) error {
	var apiErr *goai.APIError
	if errors.As(err, &apiErr) {
		var details []string
		if apiErr.StatusCode > 0 {
			details = append(details, fmt.Sprintf("HTTP %d", apiErr.StatusCode))
		}
		if apiErr.Message != "" {
			details = append(details, apiErr.Message)
		}
		if apiErr.ResponseBody != "" && apiErr.ResponseBody != apiErr.Message {
			details = append(details, fmt.Sprintf("body: %s", apiErr.ResponseBody))
		}
		return fmt.Errorf("AI generation failed (%s/%s): %s", providerName, modelID, strings.Join(details, " - "))
	}
	return fmt.Errorf("AI generation failed (%s/%s): %w", providerName, modelID, err)
}

// GenerateReply generates a complete AI response synchronously.
func (c *Client) GenerateReply(
	ctx context.Context,
	providerName types.Provider,
	modelID, apiKey, systemPrompt string,
	history []ChatMessage,
	prompt string,
) (string, error) {
	langModel, err := c.CreateModel(providerName, modelID, apiKey)
	if err != nil {
		return "", err
	}

	msgs := c.buildMessages(systemPrompt, history, prompt)
	res, err := goai.GenerateText(ctx, langModel, goai.WithMessages(msgs...))
	if err != nil {
		return "", formatAIError(providerName, modelID, err)
	}

	return res.Text, nil
}

// StreamReply starts a streaming text generation using GoAI StreamText.
func (c *Client) StreamReply(
	ctx context.Context,
	providerName types.Provider,
	modelID, apiKey, systemPrompt string,
	history []ChatMessage,
	prompt string,
) (*goai.TextStream, error) {
	langModel, err := c.CreateModel(providerName, modelID, apiKey)
	if err != nil {
		return nil, err
	}

	msgs := c.buildMessages(systemPrompt, history, prompt)
	stream, err := goai.StreamText(ctx, langModel, goai.WithMessages(msgs...))
	if err != nil {
		return nil, formatAIError(providerName, modelID, err)
	}

	return stream, nil
}
