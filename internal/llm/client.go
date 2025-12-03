package llm

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/DevSymphony/sym-cli/internal/envutil"
	"github.com/DevSymphony/sym-cli/internal/llm/engine"
	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"

	// Import providers for side-effect registration
	_ "github.com/DevSymphony/sym-cli/internal/llm/claudecode"
	_ "github.com/DevSymphony/sym-cli/internal/llm/geminicli"
	_ "github.com/DevSymphony/sym-cli/internal/llm/mcp"
	_ "github.com/DevSymphony/sym-cli/internal/llm/openai"
)

const (
	defaultMaxTokens   = 1000
	defaultTemperature = 1.0
	defaultTimeout     = 60 * time.Second
)

const (
	// ModeAPI uses OpenAI API.
	ModeAPI = engine.ModeAPI
	// ModeMCP uses MCP sampling.
	ModeMCP = engine.ModeMCP
	// ModeCLI uses CLI engine.
	ModeCLI = engine.ModeCLI
	// ModeAuto automatically selects the best available engine.
	ModeAuto = engine.ModeAuto
)

// Client represents an LLM client with fallback chain support.
type Client struct {
	// Engine configuration
	config     *LLMConfig
	mode       engine.Mode
	engines    []engine.LLMEngine
	mcpSession *mcpsdk.ServerSession

	// Default request parameters
	maxTokens   int
	temperature float64
	verbose     bool
}

// ClientOption is a functional option for configuring the client.
type ClientOption func(*Client)

// WithMaxTokens sets the default max tokens.
func WithMaxTokens(maxTokens int) ClientOption {
	return func(c *Client) { c.maxTokens = maxTokens }
}

// WithTemperature sets the default temperature.
func WithTemperature(temperature float64) ClientOption {
	return func(c *Client) { c.temperature = temperature }
}

// WithTimeout sets the HTTP client timeout (for API engine).
func WithTimeout(_ time.Duration) ClientOption {
	// Note: This is handled by individual engines now
	return func(_ *Client) {}
}

// WithVerbose enables verbose logging.
func WithVerbose(verbose bool) ClientOption {
	return func(c *Client) { c.verbose = verbose }
}

// WithMCPSession sets the MCP session for MCP mode.
func WithMCPSession(session *mcpsdk.ServerSession) ClientOption {
	return func(c *Client) {
		c.mcpSession = session
		c.mode = engine.ModeMCP
	}
}

// WithConfig sets a custom LLM configuration.
func WithConfig(cfg *LLMConfig) ClientOption {
	return func(c *Client) {
		if cfg == nil {
			return
		}
		c.config = cfg
		if mode := cfg.GetEffectiveBackend(); mode != "" {
			c.mode = mode
		}
	}
}

// WithMode sets the preferred engine mode.
func WithMode(mode engine.Mode) ClientOption {
	return func(c *Client) {
		c.mode = mode
	}
}

// NewClient creates a new LLM client.
func NewClient(opts ...ClientOption) *Client {
	// Load default config
	config := LoadLLMConfig()

	apiKey := envutil.GetAPIKey("OPENAI_API_KEY")
	config.APIKey = apiKey

	client := &Client{
		config:      config,
		mode:        config.GetEffectiveBackend(),
		maxTokens:   defaultMaxTokens,
		temperature: defaultTemperature,
		verbose:     false,
	}

	// Apply options
	for _, opt := range opts {
		opt(client)
	}

	// Initialize engine chain
	client.initEngines()

	return client
}

// initEngines initializes the engine fallback chain using the registry.
func (c *Client) initEngines() {
	c.engines = []engine.LLMEngine{}

	cfg := &engine.EngineConfig{
		APIKey:     c.config.GetAPIKey(),
		Model:      c.config.Model,
		LargeModel: c.config.LargeModel,
		CLIPath:    c.config.CLIPath,
		Verbose:    c.verbose,
		MCPSession: c.mcpSession,
	}

	// Get all registered providers sorted by priority
	for _, reg := range engine.GetAllRegistrations() {
		// Filter by mode if not auto
		if !c.shouldUseProvider(reg.Name) {
			continue
		}

		eng, err := reg.Factory(cfg)
		if err != nil || eng == nil {
			continue
		}

		if eng.IsAvailable() {
			c.engines = append(c.engines, eng)
		}
	}
}

// shouldUseProvider checks if provider should be used based on mode and config.
func (c *Client) shouldUseProvider(name string) bool {
	switch c.mode {
	case engine.ModeMCP:
		return name == "mcp"
	case engine.ModeAPI:
		return name == "openai"
	case engine.ModeCLI:
		// Use configured CLI provider or all CLI providers
		if c.config.CLI != "" {
			return name == c.config.CLI || name == c.config.CLI+"code" || name == c.config.CLI+"cli"
		}
		return name == "claudecode" || name == "geminicli"
	case engine.ModeAuto:
		fallthrough
	default:
		return true
	}
}

// Request creates a new request builder.
//
// Usage:
//
//	client.Request(system, user).Execute(ctx)                                    // default complexity
//	client.Request(system, user).WithComplexity(llm.ComplexityMedium).Execute(ctx) // higher complexity
//	client.Request(system, user).WithComplexity(engine.ComplexityHigh).Execute(ctx) // explicit complexity
//	client.Request(system, user).WithMaxTokens(2000).Execute(ctx)                // custom tokens
func (c *Client) Request(systemPrompt, userPrompt string) *RequestBuilder {
	return &RequestBuilder{
		client:      c,
		system:      systemPrompt,
		user:        userPrompt,
		maxTokens:   c.maxTokens,
		temperature: c.temperature,
		complexity:  engine.ComplexityLow,
	}
}

// GetActiveEngine returns the first available engine.
func (c *Client) GetActiveEngine() engine.LLMEngine {
	for _, e := range c.engines {
		if e.IsAvailable() {
			return e
		}
	}
	return nil
}

// GetEngines returns all configured engines.
func (c *Client) GetEngines() []engine.LLMEngine {
	return c.engines
}

// GetConfig returns the LLM configuration.
func (c *Client) GetConfig() *LLMConfig {
	return c.config
}

// CheckAvailability checks if any LLM engine is available.
func (c *Client) CheckAvailability(ctx context.Context) error {
	eng := c.GetActiveEngine()
	if eng == nil {
		return fmt.Errorf("no available LLM engine")
	}

	// For API engine, do a simple test request
	if eng.Name() == "openai-api" {
		_, err := c.Request("You are a test assistant.", "Say 'OK'").Execute(ctx)
		if err != nil {
			return fmt.Errorf("OpenAI API not available: %w", err)
		}
	}

	return nil
}

// RequestBuilder builds and executes LLM requests with chain methods.
type RequestBuilder struct {
	client      *Client
	system      string
	user        string
	maxTokens   int
	temperature float64
	complexity  engine.Complexity
}

// WithComplexity sets the task complexity hint (engine-agnostic).
func (r *RequestBuilder) WithComplexity(c engine.Complexity) *RequestBuilder {
	r.complexity = c
	return r
}

// WithMaxTokens sets max tokens for this request.
func (r *RequestBuilder) WithMaxTokens(tokens int) *RequestBuilder {
	r.maxTokens = tokens
	return r
}

// WithTemperature sets temperature for this request.
func (r *RequestBuilder) WithTemperature(temp float64) *RequestBuilder {
	r.temperature = temp
	return r
}

// Execute sends the request and returns the response.
func (r *RequestBuilder) Execute(ctx context.Context) (string, error) {
	req := &engine.Request{
		SystemPrompt: r.system,
		UserPrompt:   r.user,
		MaxTokens:    r.maxTokens,
		Temperature:  r.temperature,
		Complexity:   r.complexity,
	}

	return r.client.executeWithFallback(ctx, req)
}

// executeWithFallback tries engines in priority order.
func (c *Client) executeWithFallback(ctx context.Context, req *engine.Request) (string, error) {
	var lastErr error

	for _, eng := range c.engines {
		if !eng.IsAvailable() {
			continue
		}

		result, err := eng.Execute(ctx, req)
		if err == nil {
			return result, nil
		}

		lastErr = err
		if c.verbose {
			fmt.Fprintf(os.Stderr, "⚠️  %s failed: %v, trying next engine...\n", eng.Name(), err)
		}
	}

	if lastErr != nil {
		return "", fmt.Errorf("all engines failed, last error: %w", lastErr)
	}

	return "", fmt.Errorf("no available LLM engine configured")
}

// ExecuteDirect executes request on a specific engine without fallback.
func (c *Client) ExecuteDirect(ctx context.Context, eng engine.LLMEngine, req *engine.Request) (string, error) {
	if !eng.IsAvailable() {
		return "", fmt.Errorf("engine %s is not available", eng.Name())
	}
	return eng.Execute(ctx, req)
}
