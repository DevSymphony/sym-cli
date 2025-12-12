// Package geminicli provides the Gemini CLI LLM provider.
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
	providerName   = "geminicli"
	displayName    = "Gemini CLI"
	command        = "gemini"
	defaultModel   = "gemini-2.5-flash"
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
		Models: []llm.ModelInfo{
			{ID: "gemini-2.5-flash", DisplayName: "2.5 flash", Description: "Fast and efficient", Recommended: true},
			{ID: "gemini-2.5-pro", DisplayName: "2.5 pro", Description: "Higher capability", Recommended: false},
		},
		APIKey: llm.APIKeyConfig{Required: false},
		Mode:   llm.ModeAgenticSingle,
		Profile: llm.ProviderProfile{
			MaxPromptChars:    100000,
			DefaultTimeoutSec: 300,
			MaxRetries:        1,
		},
	})
}

// Provider implements llm.RawProvider for Gemini CLI.
type Provider struct {
	model   string
	timeout time.Duration
	verbose bool
	cliPath string
}

// Compile-time check: Provider must implement RawProvider interface
var _ llm.RawProvider = (*Provider)(nil)

// newProvider creates a new Gemini CLI provider.
// Returns error if Gemini CLI is not installed.
func newProvider(cfg llm.Config) (llm.RawProvider, error) {
	path, err := exec.LookPath(command)
	if err != nil {
		return nil, fmt.Errorf("gemini CLI not installed: run 'npm install -g @anthropic-ai/gemini-cli' to install")
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

func (p *Provider) ExecuteRaw(ctx context.Context, prompt string, format llm.ResponseFormat) (string, error) {
	args := []string{"prompt", "-m", p.model, prompt}

	if p.verbose {
		fmt.Fprintf(os.Stderr, "[geminicli] Model: %s, Prompt: %d chars\n", p.model, len(prompt))
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
			return "", fmt.Errorf("gemini CLI timed out after %v", p.timeout)
		}
		return "", fmt.Errorf("gemini CLI failed: %w\nstderr: %s", err, stderr.String())
	}

	response := strings.TrimSpace(stdout.String())

	if p.verbose {
		fmt.Fprintf(os.Stderr, "[geminicli] Response: %d chars\n", len(response))
	}

	return response, nil
}

// Close is a no-op for CLI-based providers.
func (p *Provider) Close() error {
	return nil
}
