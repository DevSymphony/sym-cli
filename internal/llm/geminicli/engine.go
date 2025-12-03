package geminicli

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/DevSymphony/sym-cli/internal/llm"
)

const (
	command           = "gemini"
	defaultModel      = "gemini-2.0-flash"
	defaultLargeModel = "gemini-2.5-pro-preview-06-05"
	defaultTimeout    = 120 * time.Second
)

// Engine implements llm.LLMEngine for Gemini CLI.
type Engine struct {
	model      string
	largeModel string
	timeout    time.Duration
	verbose    bool
	customPath string
}

// New creates a new Gemini CLI engine from configuration.
func New(cfg *llm.EngineConfig) (llm.LLMEngine, error) {
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
	return "geminicli"
}

func (e *Engine) IsAvailable() bool {
	cmdPath := e.getCommandPath()
	_, err := exec.LookPath(cmdPath)
	return err == nil
}

func (e *Engine) Capabilities() llm.Capabilities {
	models := []string{e.model}
	if e.largeModel != "" && e.largeModel != e.model {
		models = append(models, e.largeModel)
	}

	return llm.Capabilities{
		SupportsTemperature: true,
		SupportsMaxTokens:   true,
		SupportsComplexity:  e.largeModel != "",
		SupportsStreaming:   false,
		MaxContextLength:    0,
		Models:              models,
	}
}

func (e *Engine) Execute(ctx context.Context, req *llm.Request) (string, error) {
	model := e.model
	if req.Complexity >= llm.ComplexityHigh && e.largeModel != "" {
		model = e.largeModel
	}

	prompt := req.CombinedPrompt()
	args := []string{"prompt", "-m", model, prompt}

	if req.MaxTokens > 0 {
		args = append(args, "--max-tokens", fmt.Sprintf("%d", req.MaxTokens))
	}
	if req.Temperature > 0 {
		args = append(args, "--temperature", fmt.Sprintf("%.2f", req.Temperature))
	}

	if e.verbose {
		fmt.Fprintf(os.Stderr, "Gemini CLI request:\n  Model: %s\n  Complexity: %s\n  Prompt length: %d chars\n",
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
		fmt.Fprintf(os.Stderr, "Gemini CLI response:\n  Content length: %d chars\n", len(response))
	}

	return response, nil
}

func (e *Engine) getCommandPath() string {
	if e.customPath != "" {
		return e.customPath
	}
	return command
}
