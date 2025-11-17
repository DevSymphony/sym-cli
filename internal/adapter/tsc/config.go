package tsc

import (
	"encoding/json"
	"fmt"

	"github.com/DevSymphony/sym-cli/internal/engine/core"
)

// TSConfig represents TypeScript compiler configuration.
type TSConfig struct {
	CompilerOptions CompilerOptions `json:"compilerOptions"`
	Include         []string        `json:"include,omitempty"`
	Exclude         []string        `json:"exclude,omitempty"`
}

// CompilerOptions represents TypeScript compiler options.
type CompilerOptions struct {
	Target                 string   `json:"target,omitempty"`
	Module                 string   `json:"module,omitempty"`
	Lib                    []string `json:"lib,omitempty"`
	Strict                 bool     `json:"strict,omitempty"`
	NoImplicitAny          bool     `json:"noImplicitAny,omitempty"`
	StrictNullChecks       bool     `json:"strictNullChecks,omitempty"`
	StrictFunctionTypes    bool     `json:"strictFunctionTypes,omitempty"`
	StrictBindCallApply    bool     `json:"strictBindCallApply,omitempty"`
	StrictPropertyInit     bool     `json:"strictPropertyInitialization,omitempty"`
	NoImplicitThis         bool     `json:"noImplicitThis,omitempty"`
	AlwaysStrict           bool     `json:"alwaysStrict,omitempty"`
	NoUnusedLocals         bool     `json:"noUnusedLocals,omitempty"`
	NoUnusedParameters     bool     `json:"noUnusedParameters,omitempty"`
	NoImplicitReturns      bool     `json:"noImplicitReturns,omitempty"`
	NoFallthroughCasesInSwitch bool `json:"noFallthroughCasesInSwitch,omitempty"`
	SkipLibCheck           bool     `json:"skipLibCheck,omitempty"`
	ESModuleInterop        bool     `json:"esModuleInterop,omitempty"`
	AllowJS                bool     `json:"allowJs,omitempty"`
	CheckJS                bool     `json:"checkJs,omitempty"`
}

// GenerateConfig generates a tsconfig.json from a rule.
func (a *Adapter) GenerateConfig(ruleInterface interface{}) ([]byte, error) {
	// Type assert to *core.Rule
	rule, ok := ruleInterface.(*core.Rule)
	if !ok {
		return nil, fmt.Errorf("expected *core.Rule, got %T", ruleInterface)
	}

	// Default configuration for type checking
	config := TSConfig{
		CompilerOptions: CompilerOptions{
			Target:                 "ES2020",
			Module:                 "commonjs",
			Lib:                    []string{"ES2020"},
			Strict:                 true,
			NoImplicitAny:          true,
			StrictNullChecks:       true,
			StrictFunctionTypes:    true,
			StrictBindCallApply:    true,
			StrictPropertyInit:     true,
			NoImplicitThis:         true,
			AlwaysStrict:           true,
			NoUnusedLocals:         false, // Don't fail on unused locals
			NoUnusedParameters:     false, // Don't fail on unused params
			NoImplicitReturns:      true,
			NoFallthroughCasesInSwitch: true,
			SkipLibCheck:           true, // Skip type checking of declaration files
			ESModuleInterop:        true,
			AllowJS:                false, // Only check TypeScript by default
			CheckJS:                false,
		},
	}

	// Apply rule-specific configuration from Check map
	applyRuleConfig(&config, rule.Check)

	return json.MarshalIndent(config, "", "  ")
}

// applyRuleConfig applies rule-specific configuration to TSConfig.
func applyRuleConfig(config *TSConfig, check map[string]interface{}) {
	// Extract compiler options from rule
	if strict, ok := check["strict"].(bool); ok {
		config.CompilerOptions.Strict = strict
	}

	if noImplicitAny, ok := check["noImplicitAny"].(bool); ok {
		config.CompilerOptions.NoImplicitAny = noImplicitAny
	}

	if strictNullChecks, ok := check["strictNullChecks"].(bool); ok {
		config.CompilerOptions.StrictNullChecks = strictNullChecks
	}

	if allowJS, ok := check["allowJs"].(bool); ok {
		config.CompilerOptions.AllowJS = allowJS
	}

	if checkJS, ok := check["checkJs"].(bool); ok {
		config.CompilerOptions.CheckJS = checkJS
	}

	// Extract file patterns
	if include, ok := check["include"].([]interface{}); ok {
		patterns := make([]string, 0, len(include))
		for _, p := range include {
			if str, ok := p.(string); ok {
				patterns = append(patterns, str)
			}
		}
		config.Include = patterns
	}

	if exclude, ok := check["exclude"].([]interface{}); ok {
		patterns := make([]string, 0, len(exclude))
		for _, p := range exclude {
			if str, ok := p.(string); ok {
				patterns = append(patterns, str)
			}
		}
		config.Exclude = patterns
	}
}

// MarshalConfig marshals a config map to JSON.
func MarshalConfig(config interface{}) ([]byte, error) {
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to marshal config: %w", err)
	}
	return data, nil
}
