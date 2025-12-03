package mcp

import (
	"context"
	"fmt"
	"os"

	"github.com/DevSymphony/sym-cli/internal/llm/engine"
	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

const defaultMaxTokens = 1000

// Engine implements engine.LLMEngine for MCP sampling.
type Engine struct {
	session *mcpsdk.ServerSession
	verbose bool
}

// New creates a new MCP engine from configuration.
func New(cfg *engine.EngineConfig) (engine.LLMEngine, error) {
	session, ok := cfg.MCPSession.(*mcpsdk.ServerSession)
	if !ok || session == nil {
		return nil, nil
	}

	return &Engine{
		session: session,
		verbose: cfg.Verbose,
	}, nil
}

func (e *Engine) Name() string {
	return "mcp"
}

func (e *Engine) IsAvailable() bool {
	return e.session != nil
}

func (e *Engine) Capabilities() engine.Capabilities {
	return engine.Capabilities{
		SupportsTemperature: false,
		SupportsMaxTokens:   true,
		SupportsComplexity:  false,
		SupportsStreaming:   false,
		MaxContextLength:    0,
		Models:              nil,
	}
}

func (e *Engine) Execute(ctx context.Context, req *engine.Request) (string, error) {
	if e.session == nil {
		return "", fmt.Errorf("MCP session not available")
	}

	if e.verbose {
		fmt.Fprintf(os.Stderr, "MCP Sampling request:\n  MaxTokens: %d\n  Prompt length: %d chars\n",
			req.MaxTokens, len(req.UserPrompt))
	}

	maxTokens := req.MaxTokens
	if maxTokens == 0 {
		maxTokens = defaultMaxTokens
	}

	result, err := e.session.CreateMessage(ctx, &mcpsdk.CreateMessageParams{
		Messages: []*mcpsdk.SamplingMessage{
			{
				Role:    "user",
				Content: &mcpsdk.TextContent{Text: req.CombinedPrompt()},
			},
		},
		MaxTokens: int64(maxTokens),
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

	if e.verbose {
		fmt.Fprintf(os.Stderr, "MCP Sampling response:\n  Model: %s\n  Content length: %d chars\n",
			result.Model, len(response))
	}

	return response, nil
}
