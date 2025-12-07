// Package claudecode provides the Claude Code CLI LLM provider.
package claudecode

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
	providerName   = "claudecode"
	displayName    = "Claude Code CLI"
	command        = "claude"
	defaultModel   = "sonnet" // Claude CLI accepts short aliases: sonnet, opus, haiku
	defaultTimeout = 120 * time.Second
)

func init() {
	// Check if CLI is available
	path, _ := exec.LookPath(command)
	available := path != ""

	llm.RegisterProvider(providerName, newProvider, llm.ProviderInfo{
		Name:         providerName,
		DisplayName:  displayName,
		DefaultModel: defaultModel,
		Available:    available,
		Path:         path,
	})
}

// Provider implements llm.Provider for Claude Code CLI.
type Provider struct {
	model   string
	timeout time.Duration
	verbose bool
	cliPath string
}

// newProvider creates a new Claude Code provider.
// Returns error if Claude CLI is not installed.
func newProvider(cfg llm.Config) (llm.Provider, error) {
	path, err := exec.LookPath(command)
	if err != nil {
		return nil, fmt.Errorf("claude CLI not installed: run 'npm install -g @anthropic-ai/claude-cli' to install")
	}

	model := cfg.Model
	if model == "" {
		model = defaultModel
	}

	return &Provider{
		model:   model,
		timeout: defaultTimeout,
		verbose: cfg.Verbose,
		cliPath: path,
	}, nil
}

func (p *Provider) Name() string {
	return providerName
}

func (p *Provider) Execute(ctx context.Context, prompt string, format llm.ResponseFormat) (string, error) {
	args := []string{"-p", prompt, "--output-format", "text"}
	if p.model != "" {
		args = append(args, "--model", p.model)
	}

	if p.verbose {
		fmt.Fprintf(os.Stderr, "[claudecode] Model: %s, Prompt: %d chars\n", p.model, len(prompt))
	}

	cmdCtx, cancel := context.WithTimeout(ctx, p.timeout)
	defer cancel()

	cmd := exec.CommandContext(cmdCtx, p.cliPath, args...)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		if cmdCtx.Err() == context.DeadlineExceeded {
			return "", fmt.Errorf("claude CLI timed out after %v", p.timeout)
		}
		return "", fmt.Errorf("claude CLI failed: %w\nstderr: %s", err, stderr.String())
	}

	response := strings.TrimSpace(stdout.String())

	if p.verbose {
		fmt.Fprintf(os.Stderr, "[claudecode] Response: %d chars\n", len(response))
	}

	// Parse response based on format
	return llm.Parse(response, format)
}

// Close is a no-op for CLI-based providers.
func (p *Provider) Close() error {
	return nil
}
