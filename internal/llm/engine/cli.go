package engine

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"time"

	"github.com/DevSymphony/sym-cli/internal/llm/engine/cliprovider"
)

const (
	defaultCLITimeout = 120 * time.Second
)

// Re-export CLI provider types for backward compatibility.
type CLIProviderType = cliprovider.Type

const (
	// ProviderClaude is the Claude CLI provider.
	ProviderClaude CLIProviderType = cliprovider.TypeClaude
	// ProviderGemini is the Gemini CLI provider.
	ProviderGemini CLIProviderType = cliprovider.TypeGemini
)

// CLIProvider is an alias to cliprovider.Provider.
type CLIProvider = cliprovider.Provider

// CLIInfo is an alias to cliprovider.Info.
type CLIInfo = cliprovider.Info

// SupportedProviders returns all supported CLI providers.
func SupportedProviders() map[CLIProviderType]*CLIProvider {
	return cliprovider.Supported()
}

// GetProvider returns the provider for the given type.
func GetProvider(providerType CLIProviderType) (*CLIProvider, error) {
	return cliprovider.Get(providerType)
}

// DetectAvailableCLIs scans for installed CLI tools.
func DetectAvailableCLIs() []CLIInfo {
	return cliprovider.Detect()
}

// GetProviderByCommand finds a provider by its command name.
func GetProviderByCommand(command string) (*CLIProvider, error) {
	return cliprovider.GetByCommand(command)
}

// CLIEngine implements LLMEngine interface for CLI-based LLM tools.
type CLIEngine struct {
	provider   *cliprovider.Provider
	model      string
	largeModel string
	timeout    time.Duration
	verbose    bool
	customPath string
}

// CLIEngineOption is a functional option for CLIEngine.
type CLIEngineOption func(*CLIEngine)

// WithCLIModel sets the default model.
func WithCLIModel(model string) CLIEngineOption {
	return func(e *CLIEngine) { e.model = model }
}

// WithCLILargeModel sets the model for high complexity tasks.
func WithCLILargeModel(model string) CLIEngineOption {
	return func(e *CLIEngine) { e.largeModel = model }
}

// WithCLITimeout sets the execution timeout.
func WithCLITimeout(timeout time.Duration) CLIEngineOption {
	return func(e *CLIEngine) { e.timeout = timeout }
}

// WithCLIVerbose enables verbose logging.
func WithCLIVerbose(verbose bool) CLIEngineOption {
	return func(e *CLIEngine) { e.verbose = verbose }
}

// WithCLIPath sets a custom path to the CLI executable.
func WithCLIPath(path string) CLIEngineOption {
	return func(e *CLIEngine) { e.customPath = path }
}

// NewCLIEngine creates a new CLI engine for the given provider.
func NewCLIEngine(providerType CLIProviderType, opts ...CLIEngineOption) (*CLIEngine, error) {
	provider, err := cliprovider.Get(providerType)
	if err != nil {
		return nil, err
	}

	e := &CLIEngine{
		provider:   provider,
		model:      provider.DefaultModel,
		largeModel: provider.LargeModel,
		timeout:    defaultCLITimeout,
		verbose:    false,
	}

	for _, opt := range opts {
		opt(e)
	}

	return e, nil
}

// Name returns the engine identifier.
func (e *CLIEngine) Name() string {
	return fmt.Sprintf("cli-%s", e.provider.Type)
}

// IsAvailable checks if the engine can be used.
func (e *CLIEngine) IsAvailable() bool {
	cmdPath := e.getCommandPath()
	_, err := exec.LookPath(cmdPath)
	return err == nil
}

// Capabilities returns engine capabilities.
func (e *CLIEngine) Capabilities() Capabilities {
	models := []string{e.model}
	if e.largeModel != "" && e.largeModel != e.model {
		models = append(models, e.largeModel)
	}

	return Capabilities{
		SupportsTemperature: e.provider.SupportsTemperature,
		SupportsMaxTokens:   e.provider.SupportsMaxTokens,
		SupportsComplexity:  e.largeModel != "",
		SupportsStreaming:   false,
		MaxContextLength:    0,
		Models:              models,
	}
}

// Execute sends the request via CLI.
func (e *CLIEngine) Execute(ctx context.Context, req *Request) (string, error) {
	model := e.model
	if req.Complexity >= ComplexityHigh && e.largeModel != "" {
		model = e.largeModel
	}

	prompt := req.CombinedPrompt()
	args := e.provider.BuildArgs(model, prompt)
	args = e.appendOptionalFlags(args, req)

	if e.verbose {
		fmt.Fprintf(os.Stderr, "CLI Engine (%s) request:\n  Model: %s\n  Complexity: %s\n  Prompt length: %d chars\n",
			e.provider.Type, model, req.Complexity, len(prompt))
	}

	cmdCtx, cancel := context.WithTimeout(ctx, e.timeout)
	defer cancel()

	cmdPath := e.getCommandPath()
	cmd := exec.CommandContext(cmdCtx, cmdPath, args...)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		if cmdCtx.Err() == context.DeadlineExceeded {
			return "", fmt.Errorf("CLI command timed out after %v", e.timeout)
		}
		return "", fmt.Errorf("CLI command failed: %w\nstdout: %s\nstderr: %s", err, stdout.String(), stderr.String())
	}

	response, err := e.provider.ParseResponse(stdout.Bytes())
	if err != nil {
		return "", fmt.Errorf("failed to parse CLI response: %w", err)
	}

	if e.verbose {
		fmt.Fprintf(os.Stderr, "CLI Engine (%s) response:\n  Content length: %d chars\n",
			e.provider.Type, len(response))
	}

	return response, nil
}

// getCommandPath returns the path to the CLI executable.
func (e *CLIEngine) getCommandPath() string {
	if e.customPath != "" {
		return e.customPath
	}
	return e.provider.Command
}

// appendOptionalFlags adds optional flags based on request parameters.
func (e *CLIEngine) appendOptionalFlags(args []string, req *Request) []string {
	if e.provider.SupportsMaxTokens && e.provider.MaxTokensFlag != "" && req.MaxTokens > 0 {
		args = append(args, e.provider.MaxTokensFlag, fmt.Sprintf("%d", req.MaxTokens))
	}

	if e.provider.SupportsTemperature && e.provider.TemperatureFlag != "" && req.Temperature > 0 {
		args = append(args, e.provider.TemperatureFlag, fmt.Sprintf("%.2f", req.Temperature))
	}

	return args
}

// GetProvider returns the underlying provider.
func (e *CLIEngine) GetProvider() *CLIProvider {
	return e.provider
}

// GetModel returns the current model.
func (e *CLIEngine) GetModel() string {
	return e.model
}

// SetVerbose sets verbose mode.
func (e *CLIEngine) SetVerbose(verbose bool) {
	e.verbose = verbose
}
