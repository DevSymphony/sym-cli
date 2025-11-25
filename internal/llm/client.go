package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/DevSymphony/sym-cli/internal/envutil"
)

const (
	openAIAPIURL       = "https://api.openai.com/v1/chat/completions"
	defaultModel       = "gpt-5-mini"
	defaultMaxTokens   = 1000
	defaultTemperature = 1.0
	defaultTimeout     = 60 * time.Second
)

// ReasoningEffort defines the reasoning effort level for the model
type ReasoningEffort string

const (
	// ReasoningMinimal uses minimal reasoning - fastest, for simple tasks
	ReasoningMinimal ReasoningEffort = "minimal"
	// ReasoningLow uses low reasoning - for straightforward tasks
	ReasoningLow ReasoningEffort = "low"
	// ReasoningMedium uses medium reasoning - balanced
	ReasoningMedium ReasoningEffort = "medium"
	// ReasoningHigh uses high reasoning - for complex tasks
	ReasoningHigh ReasoningEffort = "high"
)

// Client represents an OpenAI API client
type Client struct {
	apiKey          string
	model           string
	httpClient      *http.Client
	maxTokens       int
	temperature     float64
	reasoningEffort ReasoningEffort
	verbose         bool
}

// ClientOption is a functional option for configuring the client
type ClientOption func(*Client)

// WithModel sets the OpenAI model to use
func WithModel(model string) ClientOption {
	return func(c *Client) {
		c.model = model
	}
}

// WithMaxTokens sets the maximum tokens for responses
func WithMaxTokens(maxTokens int) ClientOption {
	return func(c *Client) {
		c.maxTokens = maxTokens
	}
}

// WithTemperature sets the sampling temperature
func WithTemperature(temperature float64) ClientOption {
	return func(c *Client) {
		c.temperature = temperature
	}
}

// WithReasoningEffort sets the default reasoning effort level
func WithReasoningEffort(effort ReasoningEffort) ClientOption {
	return func(c *Client) {
		c.reasoningEffort = effort
	}
}

// WithTimeout sets the HTTP client timeout
func WithTimeout(timeout time.Duration) ClientOption {
	return func(c *Client) {
		c.httpClient.Timeout = timeout
	}
}

// WithVerbose enables verbose logging
func WithVerbose(verbose bool) ClientOption {
	return func(c *Client) {
		c.verbose = verbose
	}
}

// NewClient creates a new OpenAI API client
func NewClient(apiKey string, opts ...ClientOption) *Client {
	if apiKey == "" {
		apiKey = envutil.GetAPIKey("OPENAI_API_KEY")
	}

	client := &Client{
		apiKey: apiKey,
		model:  defaultModel,
		httpClient: &http.Client{
			Timeout: defaultTimeout,
		},
		maxTokens:       defaultMaxTokens,
		temperature:     defaultTemperature,
		reasoningEffort: ReasoningLow, // Default to low for general use
		verbose:         false,
	}

	for _, opt := range opts {
		opt(client)
	}

	return client
}

// openAIRequest represents the OpenAI API request structure
type openAIRequest struct {
	Model           string          `json:"model"`
	Messages        []openAIMessage `json:"messages"`
	MaxTokens       int             `json:"max_completion_tokens,omitempty"`
	Temperature     float64         `json:"temperature,omitempty"`
	ReasoningEffort string          `json:"reasoning_effort,omitempty"`
}

// openAIMessage represents a message in the conversation
type openAIMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// openAIResponse represents the OpenAI API response structure
type openAIResponse struct {
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

// CompleteOptions contains options for a single completion request
type CompleteOptions struct {
	ReasoningEffort ReasoningEffort
	MaxTokens       int
	Temperature     float64
}

// Complete sends a chat completion request using the client's default reasoning effort
func (c *Client) Complete(ctx context.Context, systemPrompt, userPrompt string) (string, error) {
	return c.CompleteWithOptions(ctx, systemPrompt, userPrompt, CompleteOptions{
		ReasoningEffort: c.reasoningEffort,
		MaxTokens:       c.maxTokens,
		Temperature:     c.temperature,
	})
}

// CompleteMinimal sends a completion request with minimal reasoning (fastest)
// Use this for simple, repetitive tasks in goroutines
func (c *Client) CompleteMinimal(ctx context.Context, systemPrompt, userPrompt string) (string, error) {
	return c.CompleteWithOptions(ctx, systemPrompt, userPrompt, CompleteOptions{
		ReasoningEffort: ReasoningMinimal,
		MaxTokens:       c.maxTokens,
		Temperature:     c.temperature,
	})
}

// CompleteLow sends a completion request with low reasoning
// Use this for straightforward tasks that need some thought
func (c *Client) CompleteLow(ctx context.Context, systemPrompt, userPrompt string) (string, error) {
	return c.CompleteWithOptions(ctx, systemPrompt, userPrompt, CompleteOptions{
		ReasoningEffort: ReasoningLow,
		MaxTokens:       c.maxTokens,
		Temperature:     c.temperature,
	})
}

// CompleteWithOptions sends a chat completion request with specific options
func (c *Client) CompleteWithOptions(ctx context.Context, systemPrompt, userPrompt string, opts CompleteOptions) (string, error) {
	if c.apiKey == "" {
		return "", fmt.Errorf("OpenAI API key not configured")
	}

	// Use default values if not specified
	if opts.ReasoningEffort == "" {
		opts.ReasoningEffort = c.reasoningEffort
	}
	if opts.MaxTokens == 0 {
		opts.MaxTokens = c.maxTokens
	}
	if opts.Temperature == 0 {
		opts.Temperature = c.temperature
	}

	reqBody := openAIRequest{
		Model: c.model,
		Messages: []openAIMessage{
			{Role: "user", Content: systemPrompt + "\n\n" + userPrompt},
		},
		MaxTokens:       opts.MaxTokens,
		Temperature:     opts.Temperature,
		ReasoningEffort: string(opts.ReasoningEffort),
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", openAIAPIURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.apiKey)

	if c.verbose {
		fmt.Printf("OpenAI API request:\n  Model: %s\n  Reasoning: %s\n  Temperature: %.1f\n  Prompt length: %d chars\n",
			c.model, opts.ReasoningEffort, opts.Temperature, len(userPrompt))
	}

	resp, err := c.httpClient.Do(req)
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

	var apiResp openAIResponse
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

	if c.verbose {
		fmt.Printf("OpenAI API response:\n  Tokens: %d\n  Content length: %d chars\n",
			apiResp.Usage.TotalTokens, len(content))
	}

	return content, nil
}

// CheckAvailability checks if the OpenAI API is available
func (c *Client) CheckAvailability(ctx context.Context) error {
	if c.apiKey == "" {
		return fmt.Errorf("OPENAI_API_KEY environment variable not set")
	}

	// Simple test request with minimal reasoning
	_, err := c.CompleteMinimal(ctx, "You are a test assistant.", "Say 'OK'")
	if err != nil {
		return fmt.Errorf("OpenAI API not available: %w", err)
	}

	return nil
}
