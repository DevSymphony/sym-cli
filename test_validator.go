package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"

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
	defer func() {
		if err := v.Close(); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to close validator: %v\n", err)
		}
	}()

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

		// Create GitChange from file using git diff
		cmd := exec.Command("git", "diff", "--no-index", "/dev/null", file)
		diffOutput, err := cmd.CombinedOutput()
		// git diff --no-index returns exit code 1 when there are differences (expected)
		if err != nil {
			if exitErr, ok := err.(*exec.ExitError); ok && exitErr.ExitCode() == 1 {
				// Expected - continue
			} else {
				fmt.Printf("❌ Failed to generate diff: %v\n\n", err)
				continue
			}
		}

		changes := []validator.GitChange{{
			FilePath: file,
			Status:   "A",
			Diff:     string(diffOutput),
		}}

		ctx := context.Background()
		result, err := v.ValidateChanges(ctx, changes)
		if err != nil {
			fmt.Printf("❌ Validation error: %v\n\n", err)
			continue
		}

		if len(result.Violations) == 0 {
			fmt.Printf("\n✅ PASSED: No violations\n")
			fmt.Printf("   Checked: %d, Passed: %d, Failed: %d\n\n", result.Checked, result.Passed, result.Failed)
		} else {
			fmt.Printf("\n❌ FAILED: %d violation(s) found\n", len(result.Violations))
			fmt.Printf("   Checked: %d, Passed: %d, Failed: %d\n", result.Checked, result.Passed, result.Failed)
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
