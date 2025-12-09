package validator

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/DevSymphony/sym-cli/internal/git"
	"github.com/DevSymphony/sym-cli/internal/linter"
	"github.com/DevSymphony/sym-cli/internal/llm"
	"github.com/DevSymphony/sym-cli/pkg/schema"
)

// executionUnit represents a unit of work for validation
// Each unit can be executed independently and in parallel with other units
// This interface is internal - polymorphism works within the same package regardless of visibility
type executionUnit interface {
	// Execute runs the validation and returns violations
	Execute(ctx context.Context) ([]Violation, error)
	// GetRuleIDs returns the IDs of rules in this execution unit
	GetRuleIDs() []string
	// GetEngineName returns the engine name (e.g., "eslint", "pylint", "llm-validator")
	GetEngineName() string
	// GetFiles returns the files to be validated
	GetFiles() []string
}

// linterExecutionUnit groups multiple rules for the same linter
// All rules are validated in a single linter execution
type linterExecutionUnit struct {
	engineName string
	rules      []schema.PolicyRule
	files      []string
	registry   *linter.Registry
	symDir     string
	verbose    bool
}

// Execute runs the linter once with all rules and files
func (u *linterExecutionUnit) Execute(ctx context.Context) ([]Violation, error) {
	if len(u.files) == 0 {
		return nil, nil
	}

	// Get linter from registry
	lntr, err := u.registry.GetLinter(u.engineName)
	if err != nil {
		return nil, fmt.Errorf("linter not found: %s: %w", u.engineName, err)
	}

	// Check availability and install if needed
	if err := lntr.CheckAvailability(ctx); err != nil {
		if u.verbose {
			fmt.Printf("   📦 Installing %s...\n", lntr.Name())
		}
		if err := lntr.Install(ctx, linter.InstallConfig{
			ToolsDir: filepath.Join(os.Getenv("HOME"), ".sym", "tools"),
		}); err != nil {
			return nil, fmt.Errorf("failed to install %s: %w", lntr.Name(), err)
		}
	}

	// Get config from .sym directory (already contains all rules for this linter)
	config, err := u.getConfig(lntr)
	if err != nil {
		return nil, fmt.Errorf("failed to get config: %w", err)
	}

	// Execute linter ONCE with ALL files
	startTime := time.Now()
	output, err := lntr.Execute(ctx, config, u.files)
	execMs := time.Since(startTime).Milliseconds()

	if err != nil {
		return nil, fmt.Errorf("linter execution failed: %w", err)
	}

	// Parse output to violations
	linterViolations, err := lntr.ParseOutput(output)
	if err != nil {
		return nil, fmt.Errorf("failed to parse output: %w", err)
	}

	// Map linter violations to our Violation type
	violations := u.mapViolationsToRules(linterViolations, output, execMs)

	if u.verbose && output.Stdout != "" {
		fmt.Printf("   📋 %s output (%dms): %d violation(s)\n", u.engineName, execMs, len(violations))
	}

	return violations, nil
}

// getConfig retrieves the linter configuration
func (u *linterExecutionUnit) getConfig(lntr linter.Linter) ([]byte, error) {
	// Check for existing config in .sym directory
	configFile := u.registry.GetConfigFile(u.engineName)
	if configFile != "" {
		configPath := filepath.Join(u.symDir, configFile)
		if data, err := os.ReadFile(configPath); err == nil {
			if u.verbose {
				fmt.Printf("   📄 Using config from %s\n", configPath)
			}
			return data, nil
		}
	}

	// Fall back to generating config from first rule
	if len(u.rules) > 0 {
		config := make(map[string]interface{})
		for k, val := range u.rules[0].Check {
			if k != "engine" && k != "desc" {
				config[k] = val
			}
		}
		if u.rules[0].Desc != "" {
			config["description"] = u.rules[0].Desc
		}
		return json.Marshal(config)
	}

	return nil, fmt.Errorf("no config available for %s", u.engineName)
}

// mapViolationsToRules maps linter violations back to policy rules
func (u *linterExecutionUnit) mapViolationsToRules(
	linterViolations []linter.Violation,
	output *linter.ToolOutput,
	execMs int64,
) []Violation {
	var violations []Violation

	for _, lv := range linterViolations {
		// Find the matching policy rule by linter rule ID
		policyRule := u.findPolicyRule(lv.RuleID)

		var policyRuleID string
		var severity string

		if policyRule != nil {
			policyRuleID = policyRule.ID
			severity = policyRule.Severity
		} else {
			// If no mapping found, use a generic ID based on the linter and rule
			policyRuleID = fmt.Sprintf("%s-%s", u.engineName, lv.RuleID)
			// Fall back to linter severity if no policy rule found
			severity = lv.Severity
		}

		if severity == "" {
			severity = "error"
		}

		violations = append(violations, Violation{
			RuleID:      policyRuleID,
			Severity:    severity,
			Message:     lv.Message,
			File:        lv.File,
			Line:        lv.Line,
			Column:      lv.Column,
			RawOutput:   output.Stdout,
			RawError:    output.Stderr,
			ToolName:    u.engineName,
			ExecutionMs: execMs,
		})
	}

	return violations
}

// findPolicyRule finds the policy rule that corresponds to a linter rule ID
func (u *linterExecutionUnit) findPolicyRule(linterRuleID string) *schema.PolicyRule {
	for i, rule := range u.rules {
		// Check if this policy rule's check config matches the linter rule
		if checkID, ok := rule.Check["ruleId"].(string); ok && checkID == linterRuleID {
			return &u.rules[i]
		}
		// Also check if the rule ID contains the linter rule ID
		if strings.Contains(rule.ID, linterRuleID) {
			return &u.rules[i]
		}
	}

	// If no direct mapping, return the first rule (legacy behavior)
	if len(u.rules) > 0 {
		return &u.rules[0]
	}

	return nil
}

// GetRuleIDs returns the IDs of all rules in this execution unit
func (u *linterExecutionUnit) GetRuleIDs() []string {
	ids := make([]string, len(u.rules))
	for i, rule := range u.rules {
		ids[i] = rule.ID
	}
	return ids
}

// GetEngineName returns the engine name
func (u *linterExecutionUnit) GetEngineName() string {
	return u.engineName
}

// GetFiles returns the files to be validated
func (u *linterExecutionUnit) GetFiles() []string {
	return u.files
}

// llmExecutionUnit represents a single (file, rule) pair for LLM validation
type llmExecutionUnit struct {
	rule     schema.PolicyRule
	change   git.Change // 단일 파일
	provider llm.Provider
	policy   *schema.CodePolicy
	verbose  bool
}

// Execute runs the LLM validation for a single (file, rule) pair
func (u *llmExecutionUnit) Execute(ctx context.Context) ([]Violation, error) {
	if u.provider == nil {
		return nil, fmt.Errorf("LLM provider not configured")
	}

	if u.change.Status == "D" {
		return nil, nil
	}

	addedLines := git.ExtractAddedLines(u.change.Diff)
	if len(addedLines) == 0 && strings.TrimSpace(u.change.Diff) != "" {
		addedLines = strings.Split(u.change.Diff, "\n")
	}

	if len(addedLines) == 0 {
		return nil, nil
	}

	llmValidator := newLLMValidator(u.provider, u.policy)
	violation, err := llmValidator.checkRule(ctx, u.change, addedLines, u.rule)
	if err != nil {
		return nil, err
	}

	if violation != nil {
		return []Violation{*violation}, nil
	}
	return nil, nil
}

// GetRuleIDs returns the ID of this rule
func (u *llmExecutionUnit) GetRuleIDs() []string {
	return []string{u.rule.ID}
}

// GetEngineName returns "llm-validator"
func (u *llmExecutionUnit) GetEngineName() string {
	return "llm-validator"
}

// GetFiles returns the single file
func (u *llmExecutionUnit) GetFiles() []string {
	if u.change.Status != "D" {
		return []string{u.change.FilePath}
	}
	return nil
}
