package cliprovider

import "strings"

func newClaudeProvider() *Provider {
	return &Provider{
		Type:         TypeClaude,
		DisplayName:  "Claude CLI",
		Command:      "claude",
		DefaultModel: "claude-sonnet-4-20250514",
		LargeModel:   "claude-sonnet-4-20250514",
		BuildArgs: func(model string, prompt string) []string {
			args := []string{
				"-p", prompt,
				"--output-format", "text",
			}
			if model != "" {
				args = append(args, "--model", model)
			}
			return args
		},
		ParseResponse: func(output []byte) (string, error) {
			return strings.TrimSpace(string(output)), nil
		},
		SupportsMaxTokens:   true,
		MaxTokensFlag:       "--max-tokens",
		SupportsTemperature: false,
		TemperatureFlag:     "",
	}
}
