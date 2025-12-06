package engine

import (
	"context"
	"fmt"
	"os"

	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

// MCPEngine implements LLMEngine interface for MCP sampling.
// It delegates LLM calls to the host application via MCP's CreateMessage.
type MCPEngine struct {
	session *mcpsdk.ServerSession
	verbose bool
}

// MCPEngineOption is a functional option for MCPEngine.
type MCPEngineOption func(*MCPEngine)

// WithMCPVerbose enables verbose logging.
func WithMCPVerbose(verbose bool) MCPEngineOption {
	return func(e *MCPEngine) { e.verbose = verbose }
}

// NewMCPEngine creates a new MCP sampling engine.
func NewMCPEngine(session *mcpsdk.ServerSession, opts ...MCPEngineOption) *MCPEngine {
	e := &MCPEngine{
		session: session,
		verbose: false,
	}

	for _, opt := range opts {
		opt(e)
	}

	return e
}

// Name returns the engine identifier.
func (e *MCPEngine) Name() string {
	return "mcp-sampling"
}

// IsAvailable checks if the engine can be used.
func (e *MCPEngine) IsAvailable() bool {
	return e.session != nil
}

// Capabilities returns engine capabilities.
// MCP sampling capabilities depend on the host LLM, so we're conservative here.
func (e *MCPEngine) Capabilities() Capabilities {
	return Capabilities{
		SupportsTemperature: false, // Host decides
		SupportsMaxTokens:   true,  // Passed to CreateMessage
		SupportsComplexity:  false, // Host decides model
		SupportsStreaming:   false, // Not implemented
		MaxContextLength:    0,     // Unknown
		Models:              nil,   // Host decides
	}
}

// Execute sends the request via MCP sampling.
func (e *MCPEngine) Execute(ctx context.Context, req *Request) (string, error) {
	if e.session == nil {
		return "", fmt.Errorf("MCP session not available")
	}

	if e.verbose {
		fmt.Fprintf(os.Stderr, "MCP Sampling request:\n  MaxTokens: %d\n  Prompt length: %d chars\n",
			req.MaxTokens, len(req.UserPrompt))
	}

	maxTokens := req.MaxTokens
	if maxTokens == 0 {
		maxTokens = defaultAPIMaxTokens
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

// GetSession returns the underlying MCP session.
func (e *MCPEngine) GetSession() *mcpsdk.ServerSession {
	return e.session
}

// SetVerbose sets verbose mode.
func (e *MCPEngine) SetVerbose(verbose bool) {
	e.verbose = verbose
}

