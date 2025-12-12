package validator

import (
	"testing"

	"github.com/DevSymphony/sym-cli/internal/linter"
	"github.com/DevSymphony/sym-cli/internal/llm"
	"github.com/DevSymphony/sym-cli/internal/util/git"
	"github.com/DevSymphony/sym-cli/pkg/schema"
	"github.com/stretchr/testify/assert"
)

func TestGetEngineName(t *testing.T) {
	tests := []struct {
		name     string
		rule     schema.PolicyRule
		expected string
	}{
		{
			name: "valid engine",
			rule: schema.PolicyRule{
				Check: map[string]interface{}{"engine": "eslint"},
			},
			expected: "eslint",
		},
		{
			name: "missing engine",
			rule: schema.PolicyRule{
				Check: map[string]interface{}{"other": "value"},
			},
			expected: "",
		},
		{
			name: "non-string engine",
			rule: schema.PolicyRule{
				Check: map[string]interface{}{"engine": 123},
			},
			expected: "",
		},
		{
			name: "nil check map",
			rule: schema.PolicyRule{
				Check: nil,
			},
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getEngineName(tt.rule)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGetDefaultConcurrency(t *testing.T) {
	result := getDefaultConcurrency()
	assert.GreaterOrEqual(t, result, 1, "concurrency should be at least 1")
	assert.LessOrEqual(t, result, 8, "concurrency should be at most 8")
}

func TestGetLanguageFromFile(t *testing.T) {
	tests := []struct {
		name     string
		filePath string
		expected string
	}{
		// JavaScript
		{"js file", "app.js", "javascript"},
		{"mjs file", "module.mjs", "javascript"},
		{"cjs file", "common.cjs", "javascript"},
		// TypeScript
		{"ts file", "app.ts", "typescript"},
		{"mts file", "module.mts", "typescript"},
		{"cts file", "common.cts", "typescript"},
		// JSX/TSX
		{"jsx file", "component.jsx", "jsx"},
		{"tsx file", "component.tsx", "tsx"},
		// Go
		{"go file", "main.go", "go"},
		// Python
		{"py file", "script.py", "python"},
		// Java
		{"java file", "Main.java", "java"},
		// C
		{"c file", "main.c", "c"},
		{"h file", "header.h", "c"},
		// C++
		{"cpp file", "main.cpp", "cpp"},
		{"cc file", "main.cc", "cpp"},
		{"cxx file", "main.cxx", "cpp"},
		{"hpp file", "header.hpp", "cpp"},
		{"hh file", "header.hh", "cpp"},
		{"hxx file", "header.hxx", "cpp"},
		// Rust
		{"rs file", "main.rs", "rust"},
		// Ruby
		{"rb file", "script.rb", "ruby"},
		// PHP
		{"php file", "index.php", "php"},
		// Shell
		{"sh file", "script.sh", "shell"},
		{"bash file", "script.bash", "shell"},
		// Unknown/unsupported
		{"unknown extension", "file.xyz", ""},
		{"no extension", "Makefile", ""},
		// Path with directories
		{"nested path", "src/components/Button.tsx", "tsx"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getLanguageFromFile(tt.filePath)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestParseJSON(t *testing.T) {
	t.Run("valid JSON", func(t *testing.T) {
		var target jsonValidationResponse
		err := parseJSON(`{"violates": true, "confidence": "high"}`, &target)
		assert.NoError(t, err)
		assert.True(t, target.Violates)
		assert.Equal(t, "high", target.Confidence)
	})

	t.Run("invalid JSON", func(t *testing.T) {
		var target jsonValidationResponse
		err := parseJSON(`{invalid json}`, &target)
		assert.Error(t, err)
	})

	t.Run("empty JSON", func(t *testing.T) {
		var target jsonValidationResponse
		err := parseJSON(`{}`, &target)
		assert.NoError(t, err)
		assert.False(t, target.Violates)
	})
}

func TestParseValidationResponseFallback(t *testing.T) {
	tests := []struct {
		name             string
		response         string
		expectViolates   bool
		expectConfidence string
	}{
		{
			name:             "violates true",
			response:         `{"violates": true, "description": "test"}`,
			expectViolates:   true,
			expectConfidence: "medium",
		},
		{
			name:             "violates true no space",
			response:         `{"violates":true}`,
			expectViolates:   true,
			expectConfidence: "medium",
		},
		{
			name:             "violates false",
			response:         `{"violates": false}`,
			expectViolates:   false,
			expectConfidence: "low",
		},
		{
			name:             "does not violate text",
			response:         `The code does not violate the rule.`,
			expectViolates:   false,
			expectConfidence: "low",
		},
		{
			name:             "no violation indicators",
			response:         `Random text without any violation`,
			expectViolates:   false,
			expectConfidence: "low",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseValidationResponseFallback(tt.response)
			assert.Equal(t, tt.expectViolates, result.Violates)
			assert.Equal(t, tt.expectConfidence, result.Confidence)
		})
	}
}

func TestParseValidationResponseFallback_WithDescriptionExtraction(t *testing.T) {
	response := `{"violates": true, "description": "Found security issue", "suggestion": "Fix it"}`
	result := parseValidationResponseFallback(response)

	assert.True(t, result.Violates)
	assert.Equal(t, "Found security issue", result.Description)
	assert.Equal(t, "Fix it", result.Suggestion)
}

func TestFilterChangesForRule(t *testing.T) {
	changes := []git.Change{
		{FilePath: "main.go", Status: "M"},
		{FilePath: "app.py", Status: "M"},
		{FilePath: "script.js", Status: "A"},
	}

	v := &Validator{}

	t.Run("nil When selector returns all", func(t *testing.T) {
		rule := &schema.PolicyRule{When: nil}
		result := v.filterChangesForRule(changes, rule)
		assert.Len(t, result, 3)
	})

	t.Run("empty When selector returns all", func(t *testing.T) {
		rule := &schema.PolicyRule{When: &schema.Selector{}}
		result := v.filterChangesForRule(changes, rule)
		assert.Len(t, result, 3)
	})

	t.Run("filter by language - go only", func(t *testing.T) {
		rule := &schema.PolicyRule{
			When: &schema.Selector{Languages: []string{"go"}},
		}
		result := v.filterChangesForRule(changes, rule)
		assert.Len(t, result, 1)
		assert.Equal(t, "main.go", result[0].FilePath)
	})

	t.Run("filter by language - multiple languages", func(t *testing.T) {
		rule := &schema.PolicyRule{
			When: &schema.Selector{Languages: []string{"go", "python"}},
		}
		result := v.filterChangesForRule(changes, rule)
		assert.Len(t, result, 2)
	})

	t.Run("filter by language - no match", func(t *testing.T) {
		rule := &schema.PolicyRule{
			When: &schema.Selector{Languages: []string{"rust"}},
		}
		result := v.filterChangesForRule(changes, rule)
		assert.Len(t, result, 0)
	})
}

func TestLinterExecutionUnit_Getters(t *testing.T) {
	rules := []schema.PolicyRule{
		{ID: "rule-1"},
		{ID: "rule-2"},
	}
	files := []string{"file1.js", "file2.js"}

	unit := &linterExecutionUnit{
		engineName: "eslint",
		rules:      rules,
		files:      files,
	}

	t.Run("GetRuleIDs", func(t *testing.T) {
		ids := unit.GetRuleIDs()
		assert.Equal(t, []string{"rule-1", "rule-2"}, ids)
	})

	t.Run("GetEngineName", func(t *testing.T) {
		assert.Equal(t, "eslint", unit.GetEngineName())
	})

	t.Run("GetFiles", func(t *testing.T) {
		assert.Equal(t, files, unit.GetFiles())
	})
}

func TestLlmExecutionUnit_Getters(t *testing.T) {
	rule := schema.PolicyRule{ID: "llm-rule-1"}

	t.Run("normal file", func(t *testing.T) {
		unit := &llmExecutionUnit{
			rule:   rule,
			change: git.Change{FilePath: "test.py", Status: "M"},
		}
		assert.Equal(t, []string{"llm-rule-1"}, unit.GetRuleIDs())
		assert.Equal(t, "llm-validator", unit.GetEngineName())
		assert.Equal(t, []string{"test.py"}, unit.GetFiles())
	})

	t.Run("deleted file returns nil files", func(t *testing.T) {
		unit := &llmExecutionUnit{
			rule:   rule,
			change: git.Change{FilePath: "deleted.py", Status: "D"},
		}
		assert.Nil(t, unit.GetFiles())
	})
}

func TestFindPolicyRule(t *testing.T) {
	rules := []schema.PolicyRule{
		{ID: "rule-1", Check: map[string]interface{}{"ruleId": "no-console"}},
		{ID: "rule-2", Check: map[string]interface{}{"ruleId": "no-unused-vars"}},
		{ID: "eslint-no-debugger"},
	}
	unit := &linterExecutionUnit{rules: rules}

	t.Run("match by ruleId in Check", func(t *testing.T) {
		result := unit.findPolicyRule("no-console")
		assert.NotNil(t, result)
		assert.Equal(t, "rule-1", result.ID)
	})

	t.Run("match by rule ID contains", func(t *testing.T) {
		result := unit.findPolicyRule("no-debugger")
		assert.NotNil(t, result)
		assert.Equal(t, "eslint-no-debugger", result.ID)
	})

	t.Run("fallback to first rule", func(t *testing.T) {
		result := unit.findPolicyRule("unknown-rule")
		assert.NotNil(t, result)
		assert.Equal(t, "rule-1", result.ID)
	})

	t.Run("empty rules returns nil", func(t *testing.T) {
		emptyUnit := &linterExecutionUnit{rules: []schema.PolicyRule{}}
		result := emptyUnit.findPolicyRule("any-rule")
		assert.Nil(t, result)
	})
}

func TestMapViolationsToRules(t *testing.T) {
	rules := []schema.PolicyRule{
		{ID: "policy-rule-1", Severity: "warning", Check: map[string]interface{}{"ruleId": "no-console"}},
	}
	unit := &linterExecutionUnit{
		engineName: "eslint",
		rules:      rules,
	}
	output := &linter.ToolOutput{
		Stdout: "test output",
		Stderr: "",
	}

	t.Run("maps with matching policy rule", func(t *testing.T) {
		linterViolations := []linter.Violation{
			{File: "app.js", Line: 10, Column: 5, Message: "Unexpected console", RuleID: "no-console"},
		}
		result := unit.mapViolationsToRules(linterViolations, output, 100)

		assert.Len(t, result, 1)
		assert.Equal(t, "policy-rule-1", result[0].RuleID)
		assert.Equal(t, "warning", result[0].Severity)
		assert.Equal(t, "app.js", result[0].File)
		assert.Equal(t, 10, result[0].Line)
		assert.Equal(t, "eslint", result[0].ToolName)
		assert.Equal(t, int64(100), result[0].ExecutionMs)
	})

	t.Run("maps without matching rule uses fallback", func(t *testing.T) {
		linterViolations := []linter.Violation{
			{File: "app.js", Line: 1, Message: "Some error", RuleID: "unknown-rule", Severity: "error"},
		}
		result := unit.mapViolationsToRules(linterViolations, output, 50)

		assert.Len(t, result, 1)
		assert.Equal(t, "policy-rule-1", result[0].RuleID) // fallback to first rule
	})

	t.Run("empty severity defaults to error", func(t *testing.T) {
		emptyRulesUnit := &linterExecutionUnit{
			engineName: "eslint",
			rules:      []schema.PolicyRule{},
		}
		linterViolations := []linter.Violation{
			{File: "app.js", Line: 1, Message: "Error", RuleID: "some-rule", Severity: ""},
		}
		result := emptyRulesUnit.mapViolationsToRules(linterViolations, output, 10)

		assert.Len(t, result, 1)
		assert.Equal(t, "error", result[0].Severity)
	})
}

func TestGroupRulesByEngine(t *testing.T) {
	changes := []git.Change{
		{FilePath: "app.js", Status: "M"},
		{FilePath: "main.go", Status: "A"},
	}

	t.Run("groups rules by engine", func(t *testing.T) {
		policy := &schema.CodePolicy{
			Rules: []schema.PolicyRule{
				{ID: "r1", Enabled: true, Check: map[string]interface{}{"engine": "eslint"}},
				{ID: "r2", Enabled: true, Check: map[string]interface{}{"engine": "eslint"}},
				{ID: "r3", Enabled: true, Check: map[string]interface{}{"engine": "golint"}},
			},
		}
		v := &Validator{policy: policy}
		groups := v.groupRulesByEngine(policy.Rules, changes)

		assert.Len(t, groups, 2)
		assert.Len(t, groups["eslint"].rules, 2)
		assert.Len(t, groups["golint"].rules, 1)
	})

	t.Run("skips disabled rules", func(t *testing.T) {
		policy := &schema.CodePolicy{
			Rules: []schema.PolicyRule{
				{ID: "r1", Enabled: true, Check: map[string]interface{}{"engine": "eslint"}},
				{ID: "r2", Enabled: false, Check: map[string]interface{}{"engine": "eslint"}},
			},
		}
		v := &Validator{policy: policy}
		groups := v.groupRulesByEngine(policy.Rules, changes)

		assert.Len(t, groups["eslint"].rules, 1)
	})

	t.Run("skips rules without engine", func(t *testing.T) {
		policy := &schema.CodePolicy{
			Rules: []schema.PolicyRule{
				{ID: "r1", Enabled: true, Check: map[string]interface{}{"engine": "eslint"}},
				{ID: "r2", Enabled: true, Check: map[string]interface{}{}},
			},
		}
		v := &Validator{policy: policy}
		groups := v.groupRulesByEngine(policy.Rules, changes)

		assert.Len(t, groups, 1)
		assert.Len(t, groups["eslint"].rules, 1)
	})

	t.Run("deduplicates files", func(t *testing.T) {
		policy := &schema.CodePolicy{
			Rules: []schema.PolicyRule{
				{ID: "r1", Enabled: true, Check: map[string]interface{}{"engine": "eslint"}},
				{ID: "r2", Enabled: true, Check: map[string]interface{}{"engine": "eslint"}},
			},
		}
		v := &Validator{policy: policy}
		groups := v.groupRulesByEngine(policy.Rules, changes)

		// Both rules apply to same files, should be deduplicated
		assert.Len(t, groups["eslint"].files, 2) // app.js and main.go
	})

	t.Run("tracks changes for llm-validator", func(t *testing.T) {
		llmChanges := []git.Change{
			{FilePath: "app.js", Status: "M", Diff: "diff content"},
		}
		policy := &schema.CodePolicy{
			Rules: []schema.PolicyRule{
				{ID: "r1", Enabled: true, Check: map[string]interface{}{"engine": "llm-validator"}},
			},
		}
		v := &Validator{policy: policy}
		groups := v.groupRulesByEngine(policy.Rules, llmChanges)

		assert.Len(t, groups["llm-validator"].changes, 1)
		assert.Equal(t, "app.js", groups["llm-validator"].changes[0].FilePath)
	})

	t.Run("skips deleted files", func(t *testing.T) {
		deletedChanges := []git.Change{
			{FilePath: "app.js", Status: "D"},
			{FilePath: "main.go", Status: "M"},
		}
		policy := &schema.CodePolicy{
			Rules: []schema.PolicyRule{
				{ID: "r1", Enabled: true, Check: map[string]interface{}{"engine": "eslint"}},
			},
		}
		v := &Validator{policy: policy}
		groups := v.groupRulesByEngine(policy.Rules, deletedChanges)

		assert.Len(t, groups["eslint"].files, 1)
		assert.True(t, groups["eslint"].files["main.go"])
	})
}

func TestCreateLLMExecutionUnits_ModeBasedBranching(t *testing.T) {
	changes := []git.Change{
		{FilePath: "app.js", Status: "M", Diff: "+console.log('test')"},
		{FilePath: "main.py", Status: "A", Diff: "+print('hello')"},
	}
	rules := []schema.PolicyRule{
		{ID: "rule-1", Enabled: true, Severity: "error", Desc: "No console.log"},
		{ID: "rule-2", Enabled: true, Severity: "warning", Desc: "No print statements"},
	}
	group := &ruleGroup{
		engineName: "llm-validator",
		rules:      rules,
		changes:    changes,
		files:      map[string]bool{"app.js": true, "main.py": true},
	}

	t.Run("parallel_api mode creates multiple units (file x rule)", func(t *testing.T) {
		v := &Validator{
			llmProviderInfo: &llm.ProviderInfo{
				Mode: llm.ModeParallelAPI,
				Profile: llm.ProviderProfile{
					MaxPromptChars: 8000,
				},
			},
		}
		units := v.createLLMExecutionUnits(group)

		// 2 files x 2 rules = 4 units (parallel API mode)
		assert.Len(t, units, 4)
		for _, unit := range units {
			assert.Equal(t, "llm-validator", unit.GetEngineName())
			assert.Len(t, unit.GetRuleIDs(), 1) // Each unit has 1 rule
			assert.Len(t, unit.GetFiles(), 1)   // Each unit has 1 file
		}
	})

	t.Run("agentic_single mode creates single unit (all files, all rules)", func(t *testing.T) {
		v := &Validator{
			llmProviderInfo: &llm.ProviderInfo{
				Mode: llm.ModeAgenticSingle,
				Profile: llm.ProviderProfile{
					MaxPromptChars:    100000,
					DefaultTimeoutSec: 300,
				},
			},
		}
		units := v.createLLMExecutionUnits(group)

		// Agentic mode: 1 unit with all files and rules
		assert.Len(t, units, 1)
		assert.Equal(t, "llm-validator", units[0].GetEngineName())
		assert.Len(t, units[0].GetRuleIDs(), 2) // All rules in single unit
		assert.Len(t, units[0].GetFiles(), 2)   // All files in single unit
	})

	t.Run("nil provider info defaults to parallel_api mode", func(t *testing.T) {
		v := &Validator{
			llmProviderInfo: nil, // No provider info
		}
		units := v.createLLMExecutionUnits(group)

		// Default (parallel): 2 files x 2 rules = 4 units
		assert.Len(t, units, 4)
	})

	t.Run("empty changes returns no units", func(t *testing.T) {
		emptyGroup := &ruleGroup{
			engineName: "llm-validator",
			rules:      rules,
			changes:    []git.Change{},
			files:      map[string]bool{},
		}
		v := &Validator{
			llmProviderInfo: &llm.ProviderInfo{Mode: llm.ModeAgenticSingle},
		}
		units := v.createLLMExecutionUnits(emptyGroup)
		assert.Len(t, units, 0)
	})

	t.Run("empty rules returns no units", func(t *testing.T) {
		noRulesGroup := &ruleGroup{
			engineName: "llm-validator",
			rules:      []schema.PolicyRule{},
			changes:    changes,
			files:      map[string]bool{"app.js": true},
		}
		v := &Validator{
			llmProviderInfo: &llm.ProviderInfo{Mode: llm.ModeAgenticSingle},
		}
		units := v.createLLMExecutionUnits(noRulesGroup)
		assert.Len(t, units, 0)
	})
}

func TestAgenticLLMExecutionUnit_Getters(t *testing.T) {
	rules := []schema.PolicyRule{
		{ID: "rule-1", Severity: "error"},
		{ID: "rule-2", Severity: "warning"},
	}
	changes := []git.Change{
		{FilePath: "app.js", Status: "M"},
		{FilePath: "main.py", Status: "A"},
		{FilePath: "deleted.go", Status: "D"}, // Should be excluded from GetFiles
	}

	unit := &agenticLLMExecutionUnit{
		rules:   rules,
		changes: changes,
	}

	t.Run("GetRuleIDs returns all rule IDs", func(t *testing.T) {
		ids := unit.GetRuleIDs()
		assert.Equal(t, []string{"rule-1", "rule-2"}, ids)
	})

	t.Run("GetEngineName returns llm-validator", func(t *testing.T) {
		assert.Equal(t, "llm-validator", unit.GetEngineName())
	})

	t.Run("GetFiles excludes deleted files", func(t *testing.T) {
		files := unit.GetFiles()
		assert.Len(t, files, 2)
		assert.Contains(t, files, "app.js")
		assert.Contains(t, files, "main.py")
		assert.NotContains(t, files, "deleted.go")
	})
}

func TestAgenticLLMExecutionUnit_BuildPrompt(t *testing.T) {
	rules := []schema.PolicyRule{
		{ID: "security-1", Severity: "error", Desc: "No hardcoded secrets", Category: "security"},
		{ID: "style-1", Severity: "warning", Desc: "Use const instead of let", When: &schema.Selector{Languages: []string{"javascript"}}},
	}
	changes := []git.Change{
		{FilePath: "app.js", Status: "M", Diff: "+const API_KEY = 'secret123';"},
		{FilePath: "main.py", Status: "A", Diff: "+password = 'admin'"},
	}

	unit := &agenticLLMExecutionUnit{
		rules:   rules,
		changes: changes,
		profile: llm.ProviderProfile{MaxPromptChars: 100000},
	}

	prompt := unit.buildAgenticPrompt()

	// Check prompt contains key sections
	assert.Contains(t, prompt, "=== RULES TO CHECK ===")
	assert.Contains(t, prompt, "=== FILES AND CHANGES TO REVIEW ===")
	assert.Contains(t, prompt, "=== END OF FILES ===")

	// Check rules are included
	assert.Contains(t, prompt, "security-1")
	assert.Contains(t, prompt, "No hardcoded secrets")
	assert.Contains(t, prompt, "style-1")
	assert.Contains(t, prompt, "Use const instead of let")

	// Check files are included
	assert.Contains(t, prompt, "app.js")
	assert.Contains(t, prompt, "main.py")

	// Check JSON format instructions
	assert.Contains(t, prompt, "JSON array")
	assert.Contains(t, prompt, "rule_id")
}

func TestAgenticLLMExecutionUnit_ParseResponse(t *testing.T) {
	rules := []schema.PolicyRule{
		{ID: "security-1", Severity: "error"},
		{ID: "style-1", Severity: "warning"},
	}
	unit := &agenticLLMExecutionUnit{rules: rules}

	t.Run("parses valid JSON array response", func(t *testing.T) {
		response := `[
			{"rule_id": "security-1", "file": "app.js", "violates": true, "confidence": "high", "description": "Found hardcoded secret", "suggestion": "Use environment variables"},
			{"rule_id": "style-1", "file": "main.py", "violates": true, "confidence": "medium", "description": "Should use const", "suggestion": "Change let to const"}
		]`
		violations := unit.parseAgenticResponse(response)

		assert.Len(t, violations, 2)
		assert.Equal(t, "security-1", violations[0].RuleID)
		assert.Equal(t, "error", violations[0].Severity)
		assert.Equal(t, "app.js", violations[0].File)
		assert.Contains(t, violations[0].Message, "Found hardcoded secret")
		assert.Contains(t, violations[0].Message, "Use environment variables")
	})

	t.Run("filters out low confidence results", func(t *testing.T) {
		response := `[
			{"rule_id": "security-1", "file": "app.js", "violates": true, "confidence": "high", "description": "High confidence"},
			{"rule_id": "style-1", "file": "main.py", "violates": true, "confidence": "low", "description": "Low confidence - ignored"}
		]`
		violations := unit.parseAgenticResponse(response)

		assert.Len(t, violations, 1)
		assert.Equal(t, "security-1", violations[0].RuleID)
	})

	t.Run("filters out non-violations", func(t *testing.T) {
		response := `[
			{"rule_id": "security-1", "file": "app.js", "violates": true, "confidence": "high", "description": "Violation"},
			{"rule_id": "style-1", "file": "main.py", "violates": false, "confidence": "high", "description": "Not a violation"}
		]`
		violations := unit.parseAgenticResponse(response)

		assert.Len(t, violations, 1)
		assert.Equal(t, "security-1", violations[0].RuleID)
	})

	t.Run("handles empty array", func(t *testing.T) {
		response := `[]`
		violations := unit.parseAgenticResponse(response)
		assert.Len(t, violations, 0)
	})

	t.Run("handles markdown-wrapped JSON", func(t *testing.T) {
		response := "```json\n[{\"rule_id\": \"security-1\", \"file\": \"app.js\", \"violates\": true, \"confidence\": \"high\", \"description\": \"test\"}]\n```"
		violations := unit.parseAgenticResponse(response)
		assert.Len(t, violations, 1)
	})

	t.Run("handles invalid JSON gracefully", func(t *testing.T) {
		response := `not valid json`
		violations := unit.parseAgenticResponse(response)
		assert.Len(t, violations, 0)
	})

	t.Run("uses default severity for unknown rule", func(t *testing.T) {
		response := `[{"rule_id": "unknown-rule", "file": "app.js", "violates": true, "confidence": "high", "description": "test"}]`
		violations := unit.parseAgenticResponse(response)

		assert.Len(t, violations, 1)
		assert.Equal(t, "warning", violations[0].Severity) // Default severity
	})
}
