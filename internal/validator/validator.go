package validator

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/DevSymphony/sym-cli/internal/engine/core"
	"github.com/DevSymphony/sym-cli/internal/engine/registry"
	"github.com/DevSymphony/sym-cli/internal/git"
	"github.com/DevSymphony/sym-cli/internal/llm"
	"github.com/DevSymphony/sym-cli/internal/roles"
	"github.com/DevSymphony/sym-cli/pkg/schema"
)

// Validator validates code against policy
type Validator struct {
	policy     *schema.CodePolicy
	verbose    bool
	registry   *registry.Registry
	workDir    string
	selector   *FileSelector
	ctx        context.Context
	ctxCancel  context.CancelFunc
	llmClient  *llm.Client // Optional: for LLM-based validation
}

// NewValidator creates a new validator
func NewValidator(policy *schema.CodePolicy, verbose bool) *Validator {
	// Determine working directory
	workDir, err := os.Getwd()
	if err != nil {
		workDir = "."
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)

	return &Validator{
		policy:     policy,
		verbose:    verbose,
		registry:   registry.Global(),
		workDir:    workDir,
		selector:   NewFileSelector(workDir),
		ctx:        ctx,
		ctxCancel:  cancel,
		llmClient:  nil, // Will be set via SetLLMClient if needed
	}
}

// SetLLMClient sets the LLM client for this validator
func (v *Validator) SetLLMClient(client *llm.Client) {
	v.llmClient = client
}

// Violation represents a policy violation
type Violation struct {
	RuleID   string
	Severity string
	Message  string
	File     string
	Line     int
	Column   int
}

// Helper functions

// getEngineName extracts the engine name from a rule
func getEngineName(rule schema.PolicyRule) string {
	if engine, ok := rule.Check["engine"].(string); ok {
		return engine
	}
	return ""
}

// initializeEngine initializes an engine if not already initialized
func (v *Validator) initializeEngine(engine core.Engine) error {
	// Create engine config
	config := core.EngineConfig{
		WorkDir:     v.workDir,
		ToolsDir:    filepath.Join(os.Getenv("HOME"), ".sym", "tools"),
		CacheDir:    filepath.Join(os.Getenv("HOME"), ".sym", "cache"),
		Timeout:     5 * time.Minute,
		Parallelism: 0,
		Debug:       v.verbose,
	}

	// Initialize engine
	return engine.Init(v.ctx, config)
}

// convertToCoreRule converts schema.PolicyRule to core.Rule
func convertToCoreRule(rule schema.PolicyRule) core.Rule {
	var when *core.Selector
	if rule.When != nil {
		when = &core.Selector{
			Languages: rule.When.Languages,
			Include:   rule.When.Include,
			Exclude:   rule.When.Exclude,
			Branches:  rule.When.Branches,
			Roles:     rule.When.Roles,
			Tags:      rule.When.Tags,
		}
	}

	var remedy *core.Remedy
	if rule.Remedy != nil {
		remedy = &core.Remedy{
			Autofix: rule.Remedy.Autofix,
			Tool:    rule.Remedy.Tool,
			Config:  rule.Remedy.Config,
		}
	}

	return core.Rule{
		ID:       rule.ID,
		Enabled:  rule.Enabled,
		Category: rule.Category,
		Severity: rule.Severity,
		Desc:     rule.Desc,
		When:     when,
		Check:    rule.Check,
		Remedy:   remedy,
		Message:  rule.Message,
	}
}

// Close cleans up validator resources
func (v *Validator) Close() error {
	if v.ctxCancel != nil {
		v.ctxCancel()
	}
	return nil
}

// ValidateChanges validates git changes against all enabled rules
// This is the unified entry point for diff-based validation used by both CLI and MCP
func (v *Validator) ValidateChanges(ctx context.Context, changes []GitChange) (*ValidationResult, error) {
	if v.policy == nil {
		return nil, fmt.Errorf("policy is not loaded")
	}

	result := &ValidationResult{
		Violations: make([]Violation, 0),
		Checked:    0,
		Passed:     0,
		Failed:     0,
	}

	// Check RBAC permissions first if enabled
	if v.policy.Enforce.RBACConfig != nil && v.policy.Enforce.RBACConfig.Enabled {
		// Get current git user
		username, err := git.GetCurrentUser()
		if err != nil {
			if v.verbose {
				fmt.Printf("⚠️  RBAC check skipped: %v\n", err)
			}
		} else {
			if v.verbose {
				fmt.Printf("🔐 Checking RBAC permissions for user: %s\n", username)
			}

			// Collect all changed files
			changedFiles := make([]string, 0, len(changes))
			for _, change := range changes {
				if change.Status != "D" { // Skip deleted files
					changedFiles = append(changedFiles, change.FilePath)
				}
			}

			if len(changedFiles) > 0 {
				// Validate file permissions
				rbacResult, err := roles.ValidateFilePermissions(username, changedFiles)
				if err != nil {
					if v.verbose {
						fmt.Printf("⚠️  RBAC validation failed: %v\n", err)
					}
				} else if !rbacResult.Allowed {
					// Add RBAC violations
					for _, deniedFile := range rbacResult.DeniedFiles {
						result.Violations = append(result.Violations, Violation{
							RuleID:   "rbac-permission-denied",
							Severity: "error",
							Message:  fmt.Sprintf("User '%s' does not have permission to modify this file", username),
							File:     deniedFile,
							Line:     0,
							Column:   0,
						})
						result.Failed++
					}

					if v.verbose {
						fmt.Printf("❌ RBAC: %d file(s) denied for user %s\n", len(rbacResult.DeniedFiles), username)
					}
				} else if v.verbose {
					fmt.Printf("✓ RBAC: User %s has permission to modify all files\n", username)
				}
			}
		}
	}

	if v.verbose {
		fmt.Printf("🔍 Validating %d change(s) against %d rule(s)...\n", len(changes), len(v.policy.Rules))
	}

	// For each enabled rule, execute validation
	for _, rule := range v.policy.Rules {
		if !rule.Enabled {
			continue
		}

		// Determine which engine to use
		engineName := getEngineName(rule)
		if engineName == "" {
			if v.verbose {
				fmt.Printf("⚠️  Rule %s has no engine specified, skipping\n", rule.ID)
			}
			continue
		}

		// Filter changes that match this rule's selector
		relevantChanges := v.filterChangesForRule(changes, &rule)
		if len(relevantChanges) == 0 {
			if v.verbose {
				fmt.Printf("   Rule %s: no matching changes\n", rule.ID)
			}
			continue
		}

		if v.verbose {
			fmt.Printf("   Rule %s (%s): checking %d change(s)...\n", rule.ID, engineName, len(relevantChanges))
		}

		// Get or create engine
		engine, err := v.registry.Get(engineName)
		if err != nil {
			fmt.Printf("⚠️  Engine %s not found for rule %s: %v\n", engineName, rule.ID, err)
			continue
		}

		// Initialize engine if needed
		if err := v.initializeEngine(engine); err != nil {
			fmt.Printf("⚠️  Failed to initialize engine %s: %v\n", engineName, err)
			continue
		}

		// Convert schema.PolicyRule to core.Rule
		coreRule := convertToCoreRule(rule)

		// Execute validation on each change
		for _, change := range relevantChanges {
			if change.Status == "D" {
				continue // Skip deleted files
			}

			result.Checked++

			// For LLM engine, validate the diff using LLMValidator
			if engineName == "llm-validator" {
				if v.llmClient == nil {
					fmt.Printf("⚠️  LLM client not configured for rule %s\n", rule.ID)
					continue
				}

				// Create LLMValidator to use its CheckRule method
				llmValidator := NewLLMValidator(v.llmClient, v.policy)

				// Extract added lines from diff
				addedLines := ExtractAddedLines(change.Diff)
				if len(addedLines) == 0 && strings.TrimSpace(change.Diff) != "" {
					addedLines = strings.Split(change.Diff, "\n")
				}

				if len(addedLines) == 0 {
					result.Passed++
					continue
				}

				violation, err := llmValidator.CheckRule(ctx, change, addedLines, rule)
				if err != nil {
					fmt.Printf("⚠️  Validation failed for rule %s: %v\n", rule.ID, err)
					continue
				}
				if violation != nil {
					result.Failed++
					result.Violations = append(result.Violations, *violation)
				} else {
					result.Passed++
				}
			} else {
				// For other engines, validate the file
				validationResult, err := engine.Validate(ctx, coreRule, []string{change.FilePath})
				if err != nil {
					fmt.Printf("⚠️  Validation failed for rule %s: %v\n", rule.ID, err)
					continue
				}

				// Collect violations
				if validationResult != nil && len(validationResult.Violations) > 0 {
					result.Failed++
					for _, coreViolation := range validationResult.Violations {
						violation := Violation{
							RuleID:   rule.ID,
							Severity: rule.Severity,
							Message:  coreViolation.Message,
							File:     coreViolation.File,
							Line:     coreViolation.Line,
							Column:   coreViolation.Column,
						}

						// Use custom message if provided
						if rule.Message != "" {
							violation.Message = rule.Message
						}

						result.Violations = append(result.Violations, violation)
					}
				} else {
					result.Passed++
				}
			}
		}

		if v.verbose && len(result.Violations) > 0 {
			fmt.Printf("   ❌ Found %d violation(s)\n", len(result.Violations))
		} else if v.verbose {
			fmt.Printf("   ✓ Passed\n")
		}
	}

	if v.verbose {
		if len(result.Violations) == 0 {
			fmt.Printf("\n✅ Validation passed: no violations found\n")
		} else {
			fmt.Printf("\n❌ Validation failed: %d violation(s) found\n", len(result.Violations))
		}
	}

	return result, nil
}

// filterChangesForRule filters git changes that match the rule's selector
func (v *Validator) filterChangesForRule(changes []GitChange, rule *schema.PolicyRule) []GitChange {
	if rule.When == nil {
		return changes // No selector, all changes match
	}

	var filtered []GitChange
	for _, change := range changes {
		// Check language match
		if len(rule.When.Languages) > 0 {
			lang := GetLanguageFromFile(change.FilePath)
			matched := false
			for _, ruleLang := range rule.When.Languages {
				if ruleLang == lang {
					matched = true
					break
				}
			}
			if !matched {
				continue
			}
		}

		// TODO: Check include/exclude patterns
		filtered = append(filtered, change)
	}

	return filtered
}
