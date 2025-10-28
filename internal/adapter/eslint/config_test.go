package eslint

import (
	"encoding/json"
	"testing"

	"github.com/DevSymphony/sym-cli/internal/engine/core"
)

func TestGenerateConfig_Pattern(t *testing.T) {
	rule := &core.Rule{
		ID:       "TEST-PATTERN",
		Category: "naming",
		Severity: "error",
		Check: map[string]interface{}{
			"engine":  "pattern",
			"target":  "identifier",
			"pattern": "^[A-Z][a-zA-Z0-9]*$",
		},
	}

	config, err := generateConfig(rule)
	if err != nil {
		t.Fatalf("generateConfig failed: %v", err)
	}

	var eslintConfig ESLintConfig
	if err := json.Unmarshal(config, &eslintConfig); err != nil {
		t.Fatalf("failed to parse config: %v", err)
	}

	if _, ok := eslintConfig.Rules["id-match"]; !ok {
		t.Error("expected id-match rule to be set")
	}
}

func TestGenerateConfig_Length(t *testing.T) {
	rule := &core.Rule{
		ID:       "TEST-LENGTH",
		Category: "formatting",
		Severity: "error",
		Check: map[string]interface{}{
			"engine": "length",
			"scope":  "line",
			"max":    100,
		},
	}

	config, err := generateConfig(rule)
	if err != nil {
		t.Fatalf("generateConfig failed: %v", err)
	}

	var eslintConfig ESLintConfig
	if err := json.Unmarshal(config, &eslintConfig); err != nil {
		t.Fatalf("failed to parse config: %v", err)
	}

	if _, ok := eslintConfig.Rules["max-len"]; !ok {
		t.Error("expected max-len rule to be set")
	}
}

func TestGenerateConfig_Style(t *testing.T) {
	rule := &core.Rule{
		ID:       "TEST-STYLE",
		Category: "style",
		Severity: "error",
		Check: map[string]interface{}{
			"engine": "style",
			"indent": 2,
			"quote":  "single",
			"semi":   true,
		},
	}

	config, err := generateConfig(rule)
	if err != nil {
		t.Fatalf("generateConfig failed: %v", err)
	}

	var eslintConfig ESLintConfig
	if err := json.Unmarshal(config, &eslintConfig); err != nil {
		t.Fatalf("failed to parse config: %v", err)
	}

	if _, ok := eslintConfig.Rules["indent"]; !ok {
		t.Error("expected indent rule to be set")
	}

	if _, ok := eslintConfig.Rules["quotes"]; !ok {
		t.Error("expected quotes rule to be set")
	}

	if _, ok := eslintConfig.Rules["semi"]; !ok {
		t.Error("expected semi rule to be set")
	}
}
