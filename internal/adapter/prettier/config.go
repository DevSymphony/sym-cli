package prettier

import (
	"encoding/json"
	"fmt"

	"github.com/DevSymphony/sym-cli/internal/engine/core"
)

// PrettierConfig represents .prettierrc.json structure.
type PrettierConfig struct {
	TabWidth     int    `json:"tabWidth,omitempty"`
	UseTabs      bool   `json:"useTabs,omitempty"`
	Semi         bool   `json:"semi,omitempty"`
	SingleQuote  bool   `json:"singleQuote,omitempty"`
	TrailingComma string `json:"trailingComma,omitempty"` // "none", "es5", "all"
	PrintWidth   int    `json:"printWidth,omitempty"`
}

// generateConfig creates Prettier config from a Symphony rule.
func generateConfig(ruleInterface interface{}) ([]byte, error) {
	rule, ok := ruleInterface.(*core.Rule)
	if !ok {
		return nil, fmt.Errorf("expected *core.Rule, got %T", ruleInterface)
	}

	config := &PrettierConfig{}

	// Map Symphony style config to Prettier options
	if indent := rule.GetInt("indent"); indent > 0 {
		config.TabWidth = indent
		config.UseTabs = false // Default to spaces
	}

	if quote := rule.GetString("quote"); quote != "" {
		config.SingleQuote = (quote == "single")
	}

	// Semi is tricky - need to check if it exists in Check
	if _, ok := rule.Check["semi"]; ok {
		config.Semi = rule.GetBool("semi")
	}

	if trailingComma := rule.GetString("trailingComma"); trailingComma != "" {
		config.TrailingComma = trailingComma
	} else {
		config.TrailingComma = "es5" // Default
	}

	// Line length
	if printWidth := rule.GetInt("printWidth"); printWidth > 0 {
		config.PrintWidth = printWidth
	} else if maxLen := rule.GetInt("max"); maxLen > 0 {
		config.PrintWidth = maxLen
	} else {
		config.PrintWidth = 100 // Default
	}

	return json.MarshalIndent(config, "", "  ")
}
