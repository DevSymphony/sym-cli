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
		Models: []llm.ModelInfo{
			{ID: "sonnet", DisplayName: "sonnet", Description: "Balanced performance and speed", Recommended: true},
			{ID: "opus", DisplayName: "opus", Description: "Highest capability", Recommended: false},
			{ID: "haiku", DisplayName: "haiku", Description: "Fast and efficient", Recommended: false},
		},
		APIKey: llm.APIKeyConfig{Required: false},
		Mode: llm.ModeAgenticSingle,
		Profile: llm.ProviderProfile{
			MaxPromptChars:    100000,
			DefaultTimeoutSec: 300,
			MaxRetries:        1,
		},
	})
}

// Provider implements llm.RawProvider for Claude Code CLI.
type Provider struct {
	model   string
	timeout time.Duration
	verbose bool
	cliPath string
}

// Compile-time check: Provider must implement RawProvider interface
var _ llm.RawProvider = (*Provider)(nil)

// newProvider creates a new Claude Code provider.
// Returns error if Claude CLI is not installed.
func newProvider(cfg llm.Config) (llm.RawProvider, error) {
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

func (p *Provider) ExecuteRaw(ctx context.Context, prompt string, format llm.ResponseFormat) (string, error) {
	args := []string{"-p", prompt, "--output-format", "text"}

	// MCP 서버 로딩 비활성화: 재귀 호출 방지
	// Symphony가 claude CLI를 호출할 때 MCP가 다시 로드되면 무한 루프 발생
	// --mcp-config는 JSON 문자열도 지원함 (파일 경로 외에)
	args = append(args, "--strict-mcp-config", "--mcp-config", `{"mcpServers":{}}`)

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

	return response, nil
}

// Close is a no-op for CLI-based providers.
func (p *Provider) Close() error {
	return nil
}
