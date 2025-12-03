package llm

import (
	"os/exec"
	"strings"
)

// CLIInfo represents detected CLI information.
type CLIInfo struct {
	Provider  string
	Name      string
	Path      string
	Version   string
	Available bool
}

// CLIProviderInfo holds default model information for a CLI provider.
type CLIProviderInfo struct {
	DefaultModel string
	LargeModel   string
}

var cliProviders = map[string]struct {
	DisplayName  string
	Command      string
	DefaultModel string
	LargeModel   string
}{
	"claude": {
		DisplayName:  "Claude CLI",
		Command:      "claude",
		DefaultModel: "claude-haiku-4-5-20251001",
		LargeModel:   "claude-sonnet-4-5-20250929",
	},
	"gemini": {
		DisplayName:  "Gemini CLI",
		Command:      "gemini",
		DefaultModel: "gemini-2.0-flash",
		LargeModel:   "gemini-2.5-pro-preview-06-05",
	},
}

// DetectAvailableCLIs scans for installed CLI tools.
func DetectAvailableCLIs() []CLIInfo {
	var results []CLIInfo

	for provider, info := range cliProviders {
		cli := CLIInfo{
			Provider:  provider,
			Name:      info.DisplayName,
			Available: false,
		}

		path, err := exec.LookPath(info.Command)
		if err == nil {
			cli.Path = path
			cli.Available = true
			cli.Version = getCLIVersion(info.Command)
		}

		results = append(results, cli)
	}

	return results
}

// GetCLIProviderInfo returns model information for a CLI provider.
func GetCLIProviderInfo(provider string) *CLIProviderInfo {
	if info, ok := cliProviders[provider]; ok {
		return &CLIProviderInfo{
			DefaultModel: info.DefaultModel,
			LargeModel:   info.LargeModel,
		}
	}
	return nil
}

func getCLIVersion(command string) string {
	cmd := exec.Command(command, "--version") // #nosec G204
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
