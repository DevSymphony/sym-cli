package prettier

import (
	"encoding/json"
	"testing"

	"github.com/DevSymphony/sym-cli/internal/engine/core"
)

func TestGenerateConfig_Basic(t *testing.T) {
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

	var prettierConfig PrettierConfig
	if err := json.Unmarshal(config, &prettierConfig); err != nil {
		t.Fatalf("failed to parse config: %v", err)
	}

	if prettierConfig.TabWidth != 2 {
		t.Errorf("tabWidth = %d, want 2", prettierConfig.TabWidth)
	}

	if !prettierConfig.SingleQuote {
		t.Error("singleQuote = false, want true")
	}

	if !prettierConfig.Semi {
		t.Error("semi = false, want true")
	}
}

func TestGenerateConfig_DoubleQuotes(t *testing.T) {
	rule := &core.Rule{
		Check: map[string]interface{}{
			"quote": "double",
		},
	}

	config, err := generateConfig(rule)
	if err != nil {
		t.Fatalf("generateConfig failed: %v", err)
	}

	var prettierConfig PrettierConfig
	if err := json.Unmarshal(config, &prettierConfig); err != nil {
		t.Fatalf("failed to parse config: %v", err)
	}

	if prettierConfig.SingleQuote {
		t.Error("singleQuote = true, want false (for double quotes)")
	}
}

func TestGenerateConfig_PrintWidth(t *testing.T) {
	rule := &core.Rule{
		Check: map[string]interface{}{
			"printWidth": 120,
		},
	}

	config, err := generateConfig(rule)
	if err != nil {
		t.Fatalf("generateConfig failed: %v", err)
	}

	var prettierConfig PrettierConfig
	if err := json.Unmarshal(config, &prettierConfig); err != nil {
		t.Fatalf("failed to parse config: %v", err)
	}

	if prettierConfig.PrintWidth != 120 {
		t.Errorf("printWidth = %d, want 120", prettierConfig.PrintWidth)
	}
}
