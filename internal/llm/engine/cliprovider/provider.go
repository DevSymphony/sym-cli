package cliprovider

import (
	"fmt"
	"os/exec"
	"strings"
)

// Type represents supported CLI provider types.
type Type string

const (
	// TypeClaude is the Claude CLI provider.
	TypeClaude Type = "claude"
	// TypeGemini is the Gemini CLI provider.
	TypeGemini Type = "gemini"
)

// IsValid checks if the provider type is valid.
func (t Type) IsValid() bool {
	switch t {
	case TypeClaude, TypeGemini:
		return true
	default:
		return false
	}
}

// Provider defines how to interact with a specific CLI tool.
type Provider struct {
	// Type is the provider identifier.
	Type Type

	// DisplayName is the human-readable name.
	DisplayName string

	// Command is the executable name or path.
	Command string

	// DefaultModel is the default model to use.
	DefaultModel string

	// LargeModel is the model for high complexity tasks (optional).
	LargeModel string

	// BuildArgs constructs CLI arguments for the given request.
	BuildArgs func(model string, prompt string) []string

	// ParseResponse extracts text from CLI output.
	ParseResponse func(output []byte) (string, error)

	// SupportsMaxTokens indicates if --max-tokens or similar is supported.
	SupportsMaxTokens bool

	// MaxTokensFlag is the flag name for max tokens (e.g., "--max-tokens").
	MaxTokensFlag string

	// SupportsTemperature indicates if temperature is supported.
	SupportsTemperature bool

	// TemperatureFlag is the flag name for temperature.
	TemperatureFlag string
}

// Info represents detected CLI information.
type Info struct {
	Provider  Type
	Name      string
	Path      string
	Version   string
	Available bool
}

// Supported returns all supported CLI providers.
func Supported() map[Type]*Provider {
	return map[Type]*Provider{
		TypeClaude: newClaudeProvider(),
		TypeGemini: newGeminiProvider(),
	}
}

// Get returns the provider for the given type.
func Get(providerType Type) (*Provider, error) {
	providers := Supported()
	provider, ok := providers[providerType]
	if !ok {
		return nil, fmt.Errorf("unsupported CLI provider: %s", providerType)
	}
	return provider, nil
}

// Detect scans for installed CLI tools.
func Detect() []Info {
	var results []Info

	providers := Supported()
	for providerType, provider := range providers {
		info := Info{
			Provider:  providerType,
			Name:      provider.DisplayName,
			Available: false,
		}

		path, err := exec.LookPath(provider.Command)
		if err == nil {
			info.Path = path
			info.Available = true
			info.Version = getProviderVersion(provider)
		}

		results = append(results, info)
	}

	return results
}

// GetByCommand finds a provider by its command name.
func GetByCommand(command string) (*Provider, error) {
	providers := Supported()
	for _, provider := range providers {
		if provider.Command == command {
			return provider, nil
		}
	}
	return nil, fmt.Errorf("no provider found for command: %s", command)
}

func getProviderVersion(provider *Provider) string {
	cmd := exec.Command(provider.Command, "--version") // #nosec G204
	output, err := cmd.Output()
	if err != nil {
		return ""
	}

	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	if len(lines) > 0 {
		return strings.TrimSpace(lines[0])
	}

	return ""
}
