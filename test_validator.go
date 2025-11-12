package main

import (
	"fmt"
	"os"

	"github.com/DevSymphony/sym-cli/internal/validator"
	"github.com/DevSymphony/sym-cli/pkg/schema"
)

func main() {
	// Create a simple test policy inline with multiple engine types
	policy := &schema.CodePolicy{
		Version: "1.0.0",
		Rules: []schema.PolicyRule{
			{
				ID:       "test-max-len",
				Enabled:  true,
				Category: "style",
				Severity: "warning",
				Desc:     "Lines should not exceed 120 characters",
				When: &schema.Selector{
					Languages: []string{"javascript"},
				},
				Check: map[string]any{
					"engine": "length",
					"scope":  "line",
					"max":    120,
				},
				Message: "Line too long (max 120 characters)",
			},
			{
				ID:       "test-pattern",
				Enabled:  true,
				Category: "security",
				Severity: "error",
				Desc:     "No hardcoded API keys",
				When: &schema.Selector{
					Languages: []string{"javascript"},
				},
				Check: map[string]any{
					"engine":  "pattern",
					"pattern": "sk-[a-zA-Z0-9]{30,}",
					"target":  "content",
				},
				Message: "Hardcoded API key detected",
			},
		},
		Enforce: schema.EnforceSettings{
			Stages: []string{"pre-commit"},
			FailOn: []string{"error"},
		},
	}

	fmt.Printf("📋 Testing validator with %d rule(s)\n\n", len(policy.Rules))

	// Create validator
	v := validator.NewValidator(policy, true)
	defer v.Close()

	// Test files
	testFiles := []string{
		"tests/e2e/examples/bad-example.js",
		"tests/e2e/examples/good-example.js",
	}

	for _, file := range testFiles {
		// Check if file exists
		if _, err := os.Stat(file); os.IsNotExist(err) {
			fmt.Printf("⚠️  Skipping %s (file not found)\n\n", file)
			continue
		}

		fmt.Printf("═══════════════════════════════════════════════\n")
		fmt.Printf("Testing: %s\n", file)
		fmt.Printf("═══════════════════════════════════════════════\n\n")

		result, err := v.Validate(file)
		if err != nil {
			fmt.Printf("❌ Validation error: %v\n\n", err)
			continue
		}

		if result.Passed {
			fmt.Printf("\n✅ PASSED: No violations\n\n")
		} else {
			fmt.Printf("\n❌ FAILED: %d violation(s) found:\n", len(result.Violations))
			for i, violation := range result.Violations {
				fmt.Printf("\n%d. [%s] %s\n", i+1, violation.Severity, violation.RuleID)
				fmt.Printf("   File: %s:%d:%d\n", violation.File, violation.Line, violation.Column)
				fmt.Printf("   Message: %s\n", violation.Message)
			}
			fmt.Println()
		}
	}

	fmt.Printf("═══════════════════════════════════════════════\n")
	fmt.Printf("✅ Validator test complete!\n")
}
