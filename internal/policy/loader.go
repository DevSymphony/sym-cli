package policy

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/DevSymphony/sym-cli/pkg/schema"
)

// Loader handles loading policy files
type Loader struct {
}

// NewLoader creates a new policy loader
func NewLoader(verbose bool) *Loader {
	return &Loader{}
}

// LoadUserPolicy loads user-friendly policy (A schema)
func (l *Loader) LoadUserPolicy(path string) (*schema.UserPolicy, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read policy file: %w", err)
	}

	var policy schema.UserPolicy
	if err := json.Unmarshal(data, &policy); err != nil {
		return nil, fmt.Errorf("failed to parse policy: %w", err)
	}

	return &policy, nil
}

// LoadCodePolicy loads formal validation policy (B schema)
func (l *Loader) LoadCodePolicy(path string) (*schema.CodePolicy, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read policy file: %w", err)
	}

	var policy schema.CodePolicy
	if err := json.Unmarshal(data, &policy); err != nil {
		return nil, fmt.Errorf("failed to parse policy: %w", err)
	}

	return &policy, nil
}
