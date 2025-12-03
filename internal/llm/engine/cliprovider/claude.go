package cliprovider

import "strings"

func newClaudeProvider() *Provider {
	return &Provider{
		Type:         TypeClaude,
		DisplayName:  "Claude CLI",
		Command:      "claude",
		DefaultModel: "claude-haiku-4-5-20251001",
		LargeModel:   "claude-sonnet-4-5-20250929",
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
		SupportsMaxTokens:   false,
		MaxTokensFlag:       "",
		SupportsTemperature: false,
		TemperatureFlag:     "",
	}
}
