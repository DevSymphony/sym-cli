package openai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/DevSymphony/sym-cli/internal/llm/engine"
)

const (
	apiURL          = "https://api.openai.com/v1/chat/completions"
	defaultFastModel   = "gpt-4o-mini"
	defaultPowerModel  = "gpt-5-mini"
	defaultTimeout     = 60 * time.Second
	defaultMaxTokens   = 1000
	defaultTemperature = 1.0
)

// Engine implements engine.LLMEngine for OpenAI API.
type Engine struct {
	apiKey      string
	fastModel   string
	powerModel  string
	httpClient  *http.Client
	maxTokens   int
	temperature float64
	verbose     bool
}

// New creates a new OpenAI engine from configuration.
func New(cfg *engine.EngineConfig) (engine.LLMEngine, error) {
	if cfg.APIKey == "" {
		return nil, nil
	}

	e := &Engine{
		apiKey:      cfg.APIKey,
		fastModel:   defaultFastModel,
		powerModel:  defaultPowerModel,
		httpClient:  &http.Client{Timeout: defaultTimeout},
		maxTokens:   defaultMaxTokens,
		temperature: defaultTemperature,
		verbose:     cfg.Verbose,
	}

	if cfg.Model != "" {
		e.fastModel = cfg.Model
	}
	if cfg.LargeModel != "" {
		e.powerModel = cfg.LargeModel
	}

	return e, nil
}

func (e *Engine) Name() string {
	return "openai"
}

func (e *Engine) IsAvailable() bool {
	return e.apiKey != ""
}

func (e *Engine) Capabilities() engine.Capabilities {
	return engine.Capabilities{
		SupportsTemperature: true,
		SupportsMaxTokens:   true,
		SupportsComplexity:  true,
		SupportsStreaming:   true,
		MaxContextLength:    128000,
		Models:              []string{e.fastModel, e.powerModel},
	}
}

func (e *Engine) Execute(ctx context.Context, req *engine.Request) (string, error) {
	if e.apiKey == "" {
		return "", fmt.Errorf("OpenAI API key not configured")
	}

	model := e.fastModel
	var reasoningEffort string

	switch req.Complexity {
	case engine.ComplexityMinimal:
		model = e.fastModel
		reasoningEffort = "minimal"
	case engine.ComplexityLow:
		model = e.fastModel
	case engine.ComplexityMedium:
		model = e.powerModel
		reasoningEffort = "low"
	case engine.ComplexityHigh:
		model = e.powerModel
		reasoningEffort = "medium"
	}

	maxTokens := req.MaxTokens
	if maxTokens == 0 {
		maxTokens = e.maxTokens
	}

	temperature := req.Temperature
	if temperature == 0 {
		temperature = e.temperature
	}

	apiReq := apiRequest{
		Model: model,
		Messages: []apiMessage{
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

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, apiURL, bytes.NewBuffer(jsonData))
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

	if e.verbose {
		fmt.Fprintf(os.Stderr, "OpenAI API response:\n  Tokens: %d\n  Content length: %d chars\n",
			apiResp.Usage.TotalTokens, len(content))
	}

	return content, nil
}

type apiRequest struct {
	Model           string       `json:"model"`
	Messages        []apiMessage `json:"messages"`
	MaxTokens       int          `json:"max_completion_tokens,omitempty"`
	Temperature     float64      `json:"temperature,omitempty"`
	ReasoningEffort string       `json:"reasoning_effort,omitempty"`
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
