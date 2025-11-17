package tsc

import (
	"encoding/json"
	"testing"

	"github.com/DevSymphony/sym-cli/internal/engine/core"
)

func TestGenerateConfig_Default(t *testing.T) {
	adapter := NewAdapter("", "/test/project")

	rule := &core.Rule{
		ID:    "test-default",
		Check: map[string]interface{}{},
	}

	config, err := adapter.GenerateConfig(rule)
	if err != nil {
		t.Fatalf("GenerateConfig() error = %v", err)
	}

	var tsconfig TSConfig
	if err := json.Unmarshal(config, &tsconfig); err != nil {
		t.Fatalf("Failed to unmarshal config: %v", err)
	}

	// Verify default options
	if !tsconfig.CompilerOptions.Strict {
		t.Error("Expected strict mode to be enabled by default")
	}

	if !tsconfig.CompilerOptions.NoImplicitAny {
		t.Error("Expected noImplicitAny to be enabled by default")
	}

	if !tsconfig.CompilerOptions.StrictNullChecks {
		t.Error("Expected strictNullChecks to be enabled by default")
	}

	if tsconfig.CompilerOptions.Target != "ES2020" {
		t.Errorf("Target = %q, want %q", tsconfig.CompilerOptions.Target, "ES2020")
	}

	if tsconfig.CompilerOptions.AllowJS {
		t.Error("Expected allowJs to be false by default")
	}
}

func TestGenerateConfig_WithRuleOptions(t *testing.T) {
	adapter := NewAdapter("", "/test/project")

	rule := &core.Rule{
		ID: "test-with-options",
		Check: map[string]interface{}{
			"strict":           false,
			"noImplicitAny":    false,
			"allowJs":          true,
			"checkJs":          true,
			"strictNullChecks": false,
		},
	}

	config, err := adapter.GenerateConfig(rule)
	if err != nil {
		t.Fatalf("GenerateConfig() error = %v", err)
	}

	var tsconfig TSConfig
	if err := json.Unmarshal(config, &tsconfig); err != nil {
		t.Fatalf("Failed to unmarshal config: %v", err)
	}

	// Verify custom options
	if tsconfig.CompilerOptions.Strict {
		t.Error("Expected strict mode to be disabled")
	}

	if tsconfig.CompilerOptions.NoImplicitAny {
		t.Error("Expected noImplicitAny to be disabled")
	}

	if !tsconfig.CompilerOptions.AllowJS {
		t.Error("Expected allowJs to be enabled")
	}

	if !tsconfig.CompilerOptions.CheckJS {
		t.Error("Expected checkJs to be enabled")
	}

	if tsconfig.CompilerOptions.StrictNullChecks {
		t.Error("Expected strictNullChecks to be disabled")
	}
}

func TestGenerateConfig_WithIncludeExclude(t *testing.T) {
	adapter := NewAdapter("", "/test/project")

	rule := &core.Rule{
		ID: "test-include-exclude",
		Check: map[string]interface{}{
			"include": []interface{}{"src/**/*.ts", "lib/**/*.ts"},
			"exclude": []interface{}{"**/*.test.ts", "dist/**"},
		},
	}

	config, err := adapter.GenerateConfig(rule)
	if err != nil {
		t.Fatalf("GenerateConfig() error = %v", err)
	}

	var tsconfig TSConfig
	if err := json.Unmarshal(config, &tsconfig); err != nil {
		t.Fatalf("Failed to unmarshal config: %v", err)
	}

	// Verify include patterns
	if len(tsconfig.Include) != 2 {
		t.Errorf("Include has %d patterns, want 2", len(tsconfig.Include))
	}

	if tsconfig.Include[0] != "src/**/*.ts" {
		t.Errorf("Include[0] = %q, want %q", tsconfig.Include[0], "src/**/*.ts")
	}

	// Verify exclude patterns
	if len(tsconfig.Exclude) != 2 {
		t.Errorf("Exclude has %d patterns, want 2", len(tsconfig.Exclude))
	}

	if tsconfig.Exclude[0] != "**/*.test.ts" {
		t.Errorf("Exclude[0] = %q, want %q", tsconfig.Exclude[0], "**/*.test.ts")
	}
}

func TestGenerateConfig_ValidJSON(t *testing.T) {
	adapter := NewAdapter("", "/test/project")

	rule := &core.Rule{
		ID:    "test-valid-json",
		Check: map[string]interface{}{},
	}

	config, err := adapter.GenerateConfig(rule)
	if err != nil {
		t.Fatalf("GenerateConfig() error = %v", err)
	}

	// Verify it's valid JSON
	var result map[string]interface{}
	if err := json.Unmarshal(config, &result); err != nil {
		t.Errorf("Generated config is not valid JSON: %v", err)
	}

	// Verify it has compilerOptions
	if _, ok := result["compilerOptions"]; !ok {
		t.Error("Config missing compilerOptions field")
	}
}

func TestMarshalConfig(t *testing.T) {
	config := map[string]interface{}{
		"compilerOptions": map[string]interface{}{
			"target": "ES2020",
			"strict": true,
		},
	}

	data, err := MarshalConfig(config)
	if err != nil {
		t.Fatalf("MarshalConfig() error = %v", err)
	}

	// Verify it's valid JSON
	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		t.Errorf("Marshaled config is not valid JSON: %v", err)
	}
}

func TestApplyRuleConfig_AllOptions(t *testing.T) {
	config := &TSConfig{
		CompilerOptions: CompilerOptions{
			Strict:           true,
			NoImplicitAny:    true,
			StrictNullChecks: true,
			AllowJS:          false,
			CheckJS:          false,
		},
	}

	check := map[string]interface{}{
		"strict":           false,
		"noImplicitAny":    false,
		"strictNullChecks": false,
		"allowJs":          true,
		"checkJs":          true,
		"include":          []interface{}{"src/**"},
		"exclude":          []interface{}{"dist/**"},
	}

	applyRuleConfig(config, check)

	// Verify all options were applied
	if config.CompilerOptions.Strict {
		t.Error("strict should be false")
	}
	if config.CompilerOptions.NoImplicitAny {
		t.Error("noImplicitAny should be false")
	}
	if config.CompilerOptions.StrictNullChecks {
		t.Error("strictNullChecks should be false")
	}
	if !config.CompilerOptions.AllowJS {
		t.Error("allowJs should be true")
	}
	if !config.CompilerOptions.CheckJS {
		t.Error("checkJs should be true")
	}
	if len(config.Include) != 1 || config.Include[0] != "src/**" {
		t.Error("include pattern not applied correctly")
	}
	if len(config.Exclude) != 1 || config.Exclude[0] != "dist/**" {
		t.Error("exclude pattern not applied correctly")
	}
}
