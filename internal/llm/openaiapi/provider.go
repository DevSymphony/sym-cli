// Package openaiapi provides the OpenAI API LLM provider.
package openaiapi

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/DevSymphony/sym-cli/internal/llm"
	"github.com/DevSymphony/sym-cli/internal/util/env"
)

const (
	providerName       = "openaiapi"
	displayName        = "OpenAI API"
	apiURL             = "https://api.openai.com/v1/chat/completions"
	defaultModel       = "gpt-4o-mini"
	defaultTimeout     = 60 * time.Second
	defaultMaxTokens   = 1000
	defaultTemperature = 1.0
)

// ErrAPIKeyRequired is returned when OpenAI API key is not provided.
var ErrAPIKeyRequired = errors.New("openaiapi: API key is required (set OPENAI_API_KEY environment variable)")

func init() {
	// OpenAI availability depends on API key (from env vars or .sym/.env)
	llm.RegisterProvider(providerName, newProvider, llm.ProviderInfo{
		Name:         providerName,
		DisplayName:  displayName,
		DefaultModel: defaultModel,
		Available:    env.GetAPIKey("OPENAI_API_KEY") != "",
		Path:         "",
		Models: []llm.ModelInfo{
			{ID: "gpt-4o-mini", DisplayName: "gpt-4o-mini", Description: "Fast and efficient", Recommended: true},
			{ID: "gpt-5-mini", DisplayName: "gpt-5-mini", Description: "Next generation model", Recommended: false},
		},
		APIKey: llm.APIKeyConfig{
			Required:   true,
			EnvVarName: "OPENAI_API_KEY",
			Prefix:     "sk-",
		},
		Mode: llm.ModeParallelAPI,
		Profile: llm.ProviderProfile{
			MaxPromptChars:    8000,
			DefaultTimeoutSec: 60,
			MaxRetries:        2,
		},
	})
}

// Provider implements llm.RawProvider for OpenAI API.
type Provider struct {
	apiKey      string
	model       string
	httpClient  *http.Client
	maxTokens   int
	temperature float64
	verbose     bool
}

// Compile-time check: Provider must implement RawProvider interface
var _ llm.RawProvider = (*Provider)(nil)

// newProvider creates a new OpenAI API provider.
// Returns ErrAPIKeyRequired if API key is not provided.
func newProvider(cfg llm.Config) (llm.RawProvider, error) {
	// Provider handles its own API key loading from env vars and .sym/.env
	apiKey := env.GetAPIKey("OPENAI_API_KEY")
	if apiKey == "" {
		return nil, ErrAPIKeyRequired
	}

	model := cfg.Model
	if model == "" {
		model = defaultModel
	}

	return &Provider{
		apiKey:      apiKey,
		model:       model,
		httpClient:  &http.Client{Timeout: defaultTimeout},
		maxTokens:   defaultMaxTokens,
		temperature: defaultTemperature,
		verbose:     cfg.Verbose,
	}, nil
}

func (p *Provider) Name() string {
	return providerName
}

func (p *Provider) ExecuteRaw(ctx context.Context, prompt string, format llm.ResponseFormat) (string, error) {
	apiReq := apiRequest{
		Model: p.model,
		Messages: []apiMessage{
			{Role: "user", Content: prompt},
		},
	}

	// Model-based parameter switching:
	// - Reasoning models (gpt-5, o1, o3, o4): use max_completion_tokens, reasoning_effort
	// - Standard models (gpt-4o): use max_tokens, temperature
	if p.isReasoningModel() {
		apiReq.MaxCompletionTokens = p.maxTokens
		apiReq.ReasoningEffort = "medium"
	} else {
		apiReq.MaxTokens = p.maxTokens
		apiReq.Temperature = p.temperature
	}

	jsonData, err := json.Marshal(apiReq)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, apiURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+p.apiKey)

	if p.verbose {
		fmt.Fprintf(os.Stderr, "[openaiapi] Model: %s, Prompt: %d chars\n", p.model, len(prompt))
	}

	resp, err := p.httpClient.Do(httpReq)
	if err != nil {
		return "", fmt.Errorf("failed to send request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response body: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("OpenAI API error (status %d): %s", resp.StatusCode, string(body))
	}

	var apiResp apiResponse
	if err := json.Unmarshal(body, &apiResp); err != nil {
		return "", fmt.Errorf("failed to unmarshal response: %w", err)
	}

	if apiResp.Error != nil {
		return "", fmt.Errorf("OpenAI API error: %s (type: %s, code: %s)",
			apiResp.Error.Message, apiResp.Error.Type, apiResp.Error.Code)
	}

	if len(apiResp.Choices) == 0 {
		return "", fmt.Errorf("no choices in response")
	}

	content := apiResp.Choices[0].Message.Content

	if p.verbose {
		fmt.Fprintf(os.Stderr, "[openaiapi] Response: %d chars, Tokens: %d\n", len(content), apiResp.Usage.TotalTokens)
	}

	return content, nil
}

// isReasoningModel returns true if the model is a reasoning model (gpt-5, o1, o3, o4).
func (p *Provider) isReasoningModel() bool {
	return strings.HasPrefix(p.model, "gpt-5") ||
		strings.HasPrefix(p.model, "o1") ||
		strings.HasPrefix(p.model, "o3") ||
		strings.HasPrefix(p.model, "o4")
}

type apiRequest struct {
	Model               string       `json:"model"`
	Messages            []apiMessage `json:"messages"`
	MaxTokens           int          `json:"max_tokens,omitempty"`
	MaxCompletionTokens int          `json:"max_completion_tokens,omitempty"`
	Temperature         float64      `json:"temperature,omitempty"`
	ReasoningEffort     string       `json:"reasoning_effort,omitempty"`
}

type apiMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type apiResponse struct {
	ID      string `json:"id"`
	Object  string `json:"object"`
	Created int64  `json:"created"`
	Model   string `json:"model"`
	Choices []struct {
		Index   int `json:"index"`
		Message struct {
			Role    string `json:"role"`
			Content string `json:"content"`
		} `json:"message"`
		FinishReason string `json:"finish_reason"`
	} `json:"choices"`
	Usage struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
		TotalTokens      int `json:"total_tokens"`
	} `json:"usage"`
	Error *struct {
		Message string `json:"message"`
		Type    string `json:"type"`
		Code    string `json:"code"`
	} `json:"error,omitempty"`
}

// Close releases HTTP client resources.
func (p *Provider) Close() error {
	if p.httpClient != nil {
		p.httpClient.CloseIdleConnections()
	}
	return nil
}
