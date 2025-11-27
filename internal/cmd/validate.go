package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/DevSymphony/sym-cli/internal/llm"
	"github.com/DevSymphony/sym-cli/internal/validator"
	"github.com/DevSymphony/sym-cli/pkg/schema"
	"github.com/spf13/cobra"
)

var (
	validatePolicyFile string
	validateStaged     bool
	validateModel      string
	validateTimeout    int
)

var validateCmd = &cobra.Command{
	Use:   "validate",
	Short: "Validate git changes against coding conventions",
	Long: `Validate git changes against coding conventions using LLM.

This command checks git changes (diff) against rules in code-policy.json
that use "llm-validator" as the engine. These are typically complex rules
that cannot be checked by traditional linters (e.g., security, architecture).

By default, this validates ALL uncommitted changes including:
  - Staged changes (git add)
  - Unstaged changes (modified but not staged)
  - Untracked files (new files not yet added)

Examples:
  # Validate all uncommitted changes (default)
  sym validate

  # Validate only staged changes
  sym validate --staged

  # Use custom policy file
  sym validate --policy custom-policy.json`,
	RunE: runValidate,
}

func init() {
	validateCmd.Flags().StringVarP(&validatePolicyFile, "policy", "p", "", "Path to code-policy.json (default: .sym/code-policy.json)")
	validateCmd.Flags().BoolVar(&validateStaged, "staged", false, "Validate only staged changes (default: all uncommitted changes)")
	validateCmd.Flags().StringVar(&validateModel, "model", "gpt-4o", "OpenAI model to use")
	validateCmd.Flags().IntVar(&validateTimeout, "timeout", 30, "Timeout per rule check in seconds")
}

func runValidate(cmd *cobra.Command, args []string) error {
	// Load code policy
	policyPath := validatePolicyFile
	if policyPath == "" {
		symDir, err := getSymDir()
		if err != nil {
			return fmt.Errorf("failed to find .sym directory: %w", err)
		}
		policyPath = filepath.Join(symDir, "code-policy.json")
	}

	policyData, err := os.ReadFile(policyPath)
	if err != nil {
		return fmt.Errorf("failed to read policy file: %w", err)
	}

	var policy schema.CodePolicy
	if err := json.Unmarshal(policyData, &policy); err != nil {
		return fmt.Errorf("failed to parse policy: %w", err)
	}

	// Get OpenAI API key
	apiKey, err := getAPIKey()
	if err != nil {
		return fmt.Errorf("OpenAI API key not configured: %w\nTip: Run 'sym init' or set OPENAI_API_KEY in .sym/.env", err)
	}

	// Create LLM client
	llmClient := llm.NewClient(
		apiKey,
		llm.WithModel(validateModel),
		llm.WithTimeout(time.Duration(validateTimeout)*time.Second),
	)

	var changes []validator.GitChange
	if validateStaged {
		changes, err = validator.GetStagedChanges()
		if err != nil {
			return fmt.Errorf("failed to get staged changes: %w", err)
		}
		fmt.Println("Validating staged changes...")
	} else {
		changes, err = validator.GetGitChanges()
		if err != nil {
			return fmt.Errorf("failed to get git changes: %w", err)
		}
		fmt.Println("Validating all uncommitted changes (staged + unstaged + untracked)...")
	}

	if len(changes) == 0 {
		fmt.Println("No changes to validate")
		return nil
	}

	fmt.Printf("Found %d changed file(s)\n", len(changes))

	// Create unified validator that handles all engines + RBAC
	v := validator.NewValidator(&policy, true) // verbose=true for CLI
	v.SetLLMClient(llmClient)
	defer func() {
		if err := v.Close(); err != nil {
			fmt.Printf("Warning: failed to close validator: %v\n", err)
		}
	}()

	// Validate changes
	ctx := context.Background()
	result, err := v.ValidateChanges(ctx, changes)
	if err != nil {
		return fmt.Errorf("validation failed: %w", err)
	}

	printValidationResult(result)

	// Exit with error if violations found
	if len(result.Violations) > 0 {
		return fmt.Errorf("found %d violation(s)", len(result.Violations))
	}

	return nil
}

func printValidationResult(result *validator.ValidationResult) {
	fmt.Printf("\n=== Validation Results ===\n")
	fmt.Printf("Checked: %d\n", result.Checked)
	fmt.Printf("Passed:  %d\n", result.Passed)
	fmt.Printf("Failed:  %d\n\n", result.Failed)

	if len(result.Violations) == 0 {
		fmt.Println("✓ All checks passed!")
		return
	}

	fmt.Printf("Found %d violation(s):\n\n", len(result.Violations))

	for i, v := range result.Violations {
		fmt.Printf("%d. [%s] %s\n", i+1, v.Severity, v.RuleID)
		fmt.Printf("   File: %s\n", v.File)
		fmt.Printf("   %s\n", v.Message)
		fmt.Println()
	}
}
