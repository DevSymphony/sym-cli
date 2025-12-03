package claudecode

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/DevSymphony/sym-cli/internal/llm/engine"
)

const (
	command           = "claude"
	defaultModel      = "claude-haiku-4-5-20251001"
	defaultLargeModel = "claude-sonnet-4-5-20250929"
	defaultTimeout    = 120 * time.Second
)

// Engine implements engine.LLMEngine for Claude Code CLI.
type Engine struct {
	model      string
	largeModel string
	timeout    time.Duration
	verbose    bool
	customPath string
}

// New creates a new Claude Code engine from configuration.
func New(cfg *engine.EngineConfig) (engine.LLMEngine, error) {
	e := &Engine{
		model:      defaultModel,
		largeModel: defaultLargeModel,
		timeout:    defaultTimeout,
		verbose:    cfg.Verbose,
		customPath: cfg.CLIPath,
	}

	if cfg.Model != "" {
		e.model = cfg.Model
	}
	if cfg.LargeModel != "" {
		e.largeModel = cfg.LargeModel
	}

	return e, nil
}

func (e *Engine) Name() string {
	return "claudecode"
}

func (e *Engine) IsAvailable() bool {
	cmdPath := e.getCommandPath()
	_, err := exec.LookPath(cmdPath)
	return err == nil
}

func (e *Engine) Capabilities() engine.Capabilities {
	models := []string{e.model}
	if e.largeModel != "" && e.largeModel != e.model {
		models = append(models, e.largeModel)
	}

	return engine.Capabilities{
		SupportsTemperature: false,
		SupportsMaxTokens:   false,
		SupportsComplexity:  e.largeModel != "",
		SupportsStreaming:   false,
		MaxContextLength:    0,
		Models:              models,
	}
}

func (e *Engine) Execute(ctx context.Context, req *engine.Request) (string, error) {
	model := e.model
	if req.Complexity >= engine.ComplexityHigh && e.largeModel != "" {
		model = e.largeModel
	}

	prompt := req.CombinedPrompt()
	args := []string{"-p", prompt, "--output-format", "text"}
	if model != "" {
		args = append(args, "--model", model)
	}

	if e.verbose {
		fmt.Fprintf(os.Stderr, "Claude Code request:\n  Model: %s\n  Complexity: %s\n  Prompt length: %d chars\n",
			model, req.Complexity, len(prompt))
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

	response := strings.TrimSpace(stdout.String())

	if e.verbose {
		fmt.Fprintf(os.Stderr, "Claude Code response:\n  Content length: %d chars\n", len(response))
	}

	return response, nil
}

func (e *Engine) getCommandPath() string {
	if e.customPath != "" {
		return e.customPath
	}
	return command
}
