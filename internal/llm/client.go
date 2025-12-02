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
	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

const (
	openAIAPIURL       = "https://api.openai.com/v1/chat/completions"
	defaultFastModel   = "gpt-4o-mini"
	defaultPowerModel  = "gpt-5-mini"
	defaultMaxTokens   = 1000
	defaultTemperature = 1.0
	defaultTimeout     = 60 * time.Second
)

// Mode defines the LLM client mode
type Mode string

const (
	ModeAPI Mode = "api"
	ModeMCP Mode = "mcp"
)

// ReasoningEffort defines the reasoning effort level for o3-mini
type ReasoningEffort string

const (
	ReasoningMinimal ReasoningEffort = "minimal"
	ReasoningLow     ReasoningEffort = "low"
	ReasoningMedium  ReasoningEffort = "medium"
	ReasoningHigh    ReasoningEffort = "high"
)

// Client represents an LLM client
type Client struct {
	mode        Mode
	apiKey      string
	fastModel   string
	powerModel  string
	httpClient  *http.Client
	mcpSession  *mcpsdk.ServerSession
	maxTokens   int
	temperature float64
	verbose     bool
}

// ClientOption is a functional option for configuring the client
type ClientOption func(*Client)

// WithMaxTokens sets the default max tokens
func WithMaxTokens(maxTokens int) ClientOption {
	return func(c *Client) { c.maxTokens = maxTokens }
}

// WithTemperature sets the default temperature
func WithTemperature(temperature float64) ClientOption {
	return func(c *Client) { c.temperature = temperature }
}

// WithTimeout sets the HTTP client timeout
func WithTimeout(timeout time.Duration) ClientOption {
	return func(c *Client) { c.httpClient.Timeout = timeout }
}

// WithVerbose enables verbose logging
func WithVerbose(verbose bool) ClientOption {
	return func(c *Client) { c.verbose = verbose }
}

// WithMCPSession sets the MCP session for MCP mode
func WithMCPSession(session *mcpsdk.ServerSession) ClientOption {
	return func(c *Client) {
		c.mcpSession = session
		c.mode = ModeMCP
	}
}

// NewClient creates a new LLM client
func NewClient(apiKey string, opts ...ClientOption) *Client {
	if apiKey == "" {
		apiKey = envutil.GetAPIKey("OPENAI_API_KEY")
	}

	client := &Client{
		mode:        ModeAPI,
		apiKey:      apiKey,
		fastModel:   defaultFastModel,
		powerModel:  defaultPowerModel,
		httpClient:  &http.Client{Timeout: defaultTimeout},
		maxTokens:   defaultMaxTokens,
		temperature: defaultTemperature,
		verbose:     false,
	}

	for _, opt := range opts {
		opt(client)
	}

	return client
}

// Request creates a new request builder
//
// Usage:
//
//	client.Request(system, user).Execute(ctx)                                    // fast model (gpt-4o-mini)
//	client.Request(system, user).WithPower(llm.ReasoningMedium).Execute(ctx)     // power model (o3-mini)
//	client.Request(system, user).WithMaxTokens(2000).Execute(ctx)                // custom tokens
func (c *Client) Request(systemPrompt, userPrompt string) *RequestBuilder {
	return &RequestBuilder{
		client:      c,
		system:      systemPrompt,
		user:        userPrompt,
		maxTokens:   c.maxTokens,
		temperature: c.temperature,
		usePower:    false,
	}
}

// RequestBuilder builds and executes LLM requests with chain methods
type RequestBuilder struct {
	client      *Client
	system      string
	user        string
	maxTokens   int
	temperature float64
	usePower    bool
	effort      ReasoningEffort
}

// WithPower enables power model (o3-mini) with specified reasoning effort
func (r *RequestBuilder) WithPower(effort ReasoningEffort) *RequestBuilder {
	r.usePower = true
	r.effort = effort
	return r
}

// WithMaxTokens sets max tokens for this request
func (r *RequestBuilder) WithMaxTokens(tokens int) *RequestBuilder {
	r.maxTokens = tokens
	return r
}

// WithTemperature sets temperature for this request
func (r *RequestBuilder) WithTemperature(temp float64) *RequestBuilder {
	r.temperature = temp
	return r
}

// Execute sends the request and returns the response
func (r *RequestBuilder) Execute(ctx context.Context) (string, error) {
	if r.client.mode == ModeMCP {
		return r.client.executeViaMCP(ctx, r)
	}
	return r.client.executeViaAPI(ctx, r)
}

// openAIRequest represents the OpenAI API request structure
type openAIRequest struct {
	Model           string          `json:"model"`
	Messages        []openAIMessage `json:"messages"`
	MaxTokens       int             `json:"max_completion_tokens,omitempty"`
	Temperature     float64         `json:"temperature,omitempty"`
	ReasoningEffort string          `json:"reasoning_effort,omitempty"`
}

type openAIMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

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

// executeViaAPI sends request via OpenAI API
func (c *Client) executeViaAPI(ctx context.Context, r *RequestBuilder) (string, error) {
	if c.apiKey == "" {
		return "", fmt.Errorf("OpenAI API key not configured")
	}

	model := c.fastModel
	if r.usePower {
		model = c.powerModel
	}

	reqBody := openAIRequest{
		Model: model,
		Messages: []openAIMessage{
			{Role: "user", Content: r.system + "\n\n" + r.user},
		},
		MaxTokens:   r.maxTokens,
		Temperature: r.temperature,
	}

	if r.usePower {
		reqBody.ReasoningEffort = string(r.effort)
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
		if r.usePower {
			fmt.Printf("OpenAI API request:\n  Model: %s\n  Reasoning: %s\n  Prompt length: %d chars\n",
				model, r.effort, len(r.user))
		} else {
			fmt.Printf("OpenAI API request:\n  Model: %s\n  Prompt length: %d chars\n",
				model, len(r.user))
		}
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

// executeViaMCP sends request via MCP sampling
func (c *Client) executeViaMCP(ctx context.Context, r *RequestBuilder) (string, error) {
	if c.mcpSession == nil {
		return "", fmt.Errorf("MCP session not available")
	}

	if c.verbose {
		fmt.Printf("MCP Sampling request:\n  MaxTokens: %d\n  Prompt length: %d chars\n",
			r.maxTokens, len(r.user))
	}

	combinedPrompt := r.system + "\n\n" + r.user

	result, err := c.mcpSession.CreateMessage(ctx, &mcpsdk.CreateMessageParams{
		Messages: []*mcpsdk.SamplingMessage{
			{
				Role:    "user",
				Content: &mcpsdk.TextContent{Text: combinedPrompt},
			},
		},
		MaxTokens: int64(r.maxTokens),
	})
	if err != nil {
		return "", fmt.Errorf("MCP sampling failed: %w", err)
	}

	var response string
	if textContent, ok := result.Content.(*mcpsdk.TextContent); ok {
		response = textContent.Text
	} else {
		return "", fmt.Errorf("unexpected content type from MCP sampling")
	}

	if c.verbose {
		fmt.Printf("MCP Sampling response:\n  Model: %s\n  Content length: %d chars\n",
			result.Model, len(response))
	}

	return response, nil
}

// CheckAvailability checks if the LLM is available
func (c *Client) CheckAvailability(ctx context.Context) error {
	if c.mode == ModeMCP {
		if c.mcpSession == nil {
			return fmt.Errorf("MCP session not available")
		}
		return nil
	}

	if c.apiKey == "" {
		return fmt.Errorf("OPENAI_API_KEY environment variable not set")
	}

	_, err := c.Request("You are a test assistant.", "Say 'OK'").Execute(ctx)
	if err != nil {
		return fmt.Errorf("OpenAI API not available: %w", err)
	}

	return nil
}
