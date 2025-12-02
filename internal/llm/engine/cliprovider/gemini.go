package cliprovider

import "strings"

func newGeminiProvider() *Provider {
	return &Provider{
		Type:         TypeGemini,
		DisplayName:  "Gemini CLI",
		Command:      "gemini",
		DefaultModel: "gemini-2.0-flash",
		LargeModel:   "gemini-2.5-pro-preview-06-05",
		BuildArgs: func(model string, prompt string) []string {
			return []string{
				"prompt",
				"-m", model,
				prompt,
			}
		},
		ParseResponse: func(output []byte) (string, error) {
			return strings.TrimSpace(string(output)), nil
		},
		SupportsMaxTokens:   true,
		MaxTokensFlag:       "--max-tokens",
		SupportsTemperature: true,
		TemperatureFlag:     "--temperature",
	}
}
