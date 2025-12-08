package llm

import "context"

// parsedProvider wraps a RawProvider with automatic response parsing.
type parsedProvider struct {
	raw RawProvider
}

// wrapWithParser creates a Provider that automatically parses responses.
func wrapWithParser(raw RawProvider) Provider {
	return &parsedProvider{raw: raw}
}

// Execute sends a prompt and returns the parsed response.
func (p *parsedProvider) Execute(ctx context.Context, prompt string, format ResponseFormat) (string, error) {
	response, err := p.raw.ExecuteRaw(ctx, prompt, format)
	if err != nil {
		return "", err
	}
	return parse(response, format)
}

// Name returns the provider name.
func (p *parsedProvider) Name() string {
	return p.raw.Name()
}

// Close releases any resources held by the provider.
func (p *parsedProvider) Close() error {
	return p.raw.Close()
}
