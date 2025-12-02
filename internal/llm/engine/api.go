package engine

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"
)

const (
	openAIAPIURL          = "https://api.openai.com/v1/chat/completions"
	defaultAPIFastModel   = "gpt-4o-mini"
	defaultAPIPowerModel  = "gpt-5-mini"
	defaultAPITimeout     = 60 * time.Second
	defaultAPIMaxTokens   = 1000
	defaultAPITemperature = 1.0
)

// APIEngine implements LLMEngine interface for OpenAI API.
type APIEngine struct {
	apiKey      string
	fastModel   string
	powerModel  string
	httpClient  *http.Client
	maxTokens   int
	temperature float64
	verbose     bool
}

// APIEngineOption is a functional option for APIEngine.
type APIEngineOption func(*APIEngine)

// WithAPIFastModel sets the fast model.
func WithAPIFastModel(model string) APIEngineOption {
	return func(e *APIEngine) { e.fastModel = model }
}

// WithAPIPowerModel sets the power model.
func WithAPIPowerModel(model string) APIEngineOption {
	return func(e *APIEngine) { e.powerModel = model }
}

// WithAPITimeout sets the HTTP client timeout.
func WithAPITimeout(timeout time.Duration) APIEngineOption {
	return func(e *APIEngine) { e.httpClient.Timeout = timeout }
}

// WithAPIVerbose enables verbose logging.
func WithAPIVerbose(verbose bool) APIEngineOption {
	return func(e *APIEngine) { e.verbose = verbose }
}

// NewAPIEngine creates a new OpenAI API engine.
func NewAPIEngine(apiKey string, opts ...APIEngineOption) *APIEngine {
	e := &APIEngine{
		apiKey:      apiKey,
		fastModel:   defaultAPIFastModel,
		powerModel:  defaultAPIPowerModel,
		httpClient:  &http.Client{Timeout: defaultAPITimeout},
		maxTokens:   defaultAPIMaxTokens,
		temperature: defaultAPITemperature,
		verbose:     false,
	}

	for _, opt := range opts {
		opt(e)
	}

	return e
}

// Name returns the engine identifier.
func (e *APIEngine) Name() string {
	return "openai-api"
}

// IsAvailable checks if the engine can be used.
func (e *APIEngine) IsAvailable() bool {
	return e.apiKey != ""
}

// Capabilities returns engine capabilities.
func (e *APIEngine) Capabilities() Capabilities {
	return Capabilities{
		SupportsTemperature: true,
		SupportsMaxTokens:   true,
		SupportsComplexity:  true,
		SupportsStreaming:   true,
		MaxContextLength:    128000,
		Models:              []string{e.fastModel, e.powerModel},
	}
}

// Execute sends the request via OpenAI API.
func (e *APIEngine) Execute(ctx context.Context, req *Request) (string, error) {
	if e.apiKey == "" {
		return "", fmt.Errorf("OpenAI API key not configured")
	}

	// Select model based on complexity
	model := e.fastModel
	var reasoningEffort string

	switch req.Complexity {
	case ComplexityMinimal:
		model = e.fastModel
		reasoningEffort = "minimal"
	case ComplexityLow:
		model = e.fastModel
	case ComplexityMedium:
		model = e.powerModel
		reasoningEffort = "low"
	case ComplexityHigh:
		model = e.powerModel
		reasoningEffort = "medium"
	}

	// Build request body
	maxTokens := req.MaxTokens
	if maxTokens == 0 {
		maxTokens = e.maxTokens
	}

	temperature := req.Temperature
	if temperature == 0 {
		temperature = e.temperature
	}

	apiReq := openAIAPIRequest{
		Model: model,
		Messages: []openAIAPIMessage{
			{Role: "user", Content: req.CombinedPrompt()},
		},
		MaxTokens:   maxTokens,
		Temperature: temperature,
	}

	if reasoningEffort != "" {
		apiReq.ReasoningEffort = reasoningEffort
	}

	jsonData, err := json.Marshal(apiReq)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, openAIAPIURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+e.apiKey)

	if e.verbose {
		fmt.Fprintf(os.Stderr, "OpenAI API request:\n  Model: %s\n  Complexity: %s\n  Prompt length: %d chars\n",
			model, req.Complexity, len(req.UserPrompt))
	}

	resp, err := e.httpClient.Do(httpReq)
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

	var apiResp openAIAPIResponse
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

	if e.verbose {
		fmt.Fprintf(os.Stderr, "OpenAI API response:\n  Tokens: %d\n  Content length: %d chars\n",
			apiResp.Usage.TotalTokens, len(content))
	}

	return content, nil
}

// SetVerbose sets verbose mode.
func (e *APIEngine) SetVerbose(verbose bool) {
	e.verbose = verbose
}

// openAIAPIRequest represents the OpenAI API request structure.
type openAIAPIRequest struct {
	Model           string             `json:"model"`
	Messages        []openAIAPIMessage `json:"messages"`
	MaxTokens       int                `json:"max_completion_tokens,omitempty"`
	Temperature     float64            `json:"temperature,omitempty"`
	ReasoningEffort string             `json:"reasoning_effort,omitempty"`
}

// openAIAPIMessage represents a message in the OpenAI API request.
type openAIAPIMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// openAIAPIResponse represents the OpenAI API response structure.
type openAIAPIResponse struct {
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
