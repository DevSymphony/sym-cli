package validator

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/DevSymphony/sym-cli/internal/engine/core"
	"github.com/DevSymphony/sym-cli/internal/engine/registry"
	"github.com/DevSymphony/sym-cli/pkg/schema"
)

// Validator validates code against policy
type Validator struct {
	policy    *schema.CodePolicy
	verbose   bool
	registry  *registry.Registry
	workDir   string
	selector  *FileSelector
	ctx       context.Context
	ctxCancel context.CancelFunc
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
		policy:    policy,
		verbose:   verbose,
		registry:  registry.Global(),
		workDir:   workDir,
		selector:  NewFileSelector(workDir),
		ctx:       ctx,
		ctxCancel: cancel,
	}
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

// Result represents validation result
type Result struct {
	Violations []Violation
	Passed     bool
}

// Validate validates the given path
func (v *Validator) Validate(path string) (*Result, error) {
	if v.policy == nil {
		return nil, fmt.Errorf("policy is not loaded")
	}

	result := &Result{
		Violations: make([]Violation, 0),
		Passed:     true,
	}

	if v.verbose {
		fmt.Printf("🔍 Validating %s against %d rule(s)...\n", path, len(v.policy.Rules))
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

		// Get files that match this rule's selector
		files, err := v.selectFilesForRule(path, &rule)
		if err != nil {
			fmt.Printf("⚠️  Failed to select files for rule %s: %v\n", rule.ID, err)
			continue
		}

		if len(files) == 0 {
			if v.verbose {
				fmt.Printf("   Rule %s: no matching files\n", rule.ID)
			}
			continue
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

		// Execute validation
		if v.verbose {
			fmt.Printf("   Rule %s (%s): checking %d file(s)...\n", rule.ID, engineName, len(files))
		}

		validationResult, err := engine.Validate(v.ctx, coreRule, files)
		if err != nil {
			fmt.Printf("⚠️  Validation failed for rule %s: %v\n", rule.ID, err)
			continue
		}

		// Collect violations
		if validationResult != nil && len(validationResult.Violations) > 0 {
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

			if v.verbose {
				fmt.Printf("   ❌ Found %d violation(s)\n", len(validationResult.Violations))
			}
		} else if v.verbose {
			fmt.Printf("   ✓ Passed\n")
		}
	}

	// Determine overall pass/fail
	result.Passed = len(result.Violations) == 0

	if v.verbose {
		if result.Passed {
			fmt.Printf("\n✅ Validation passed: no violations found\n")
		} else {
			fmt.Printf("\n❌ Validation failed: %d violation(s) found\n", len(result.Violations))
		}
	}

	return result, nil
}

// CanAutoFix checks if violations can be auto-fixed
func (v *Result) CanAutoFix() bool {
	for _, violation := range v.Violations {
		// Check if rule has autofix enabled
		_ = violation
	}
	return false
}

// AutoFix attempts to automatically fix violations
func (v *Validator) AutoFix(result *Result) error {
	// TODO: Implement auto-fix logic
	return fmt.Errorf("auto-fix not implemented yet")
}

// Helper functions

// getEngineName extracts the engine name from a rule
func getEngineName(rule schema.PolicyRule) string {
	if engine, ok := rule.Check["engine"].(string); ok {
		return engine
	}
	return ""
}

// selectFilesForRule selects files that match the rule's selector
func (v *Validator) selectFilesForRule(basePath string, rule *schema.PolicyRule) ([]string, error) {
	// If path is a specific file, check if it matches the selector
	fileInfo, err := os.Stat(basePath)
	if err != nil {
		return nil, err
	}

	if !fileInfo.IsDir() {
		// Single file - check if it matches selector
		if rule.When != nil {
			lang := GetLanguageFromFile(basePath)
			if len(rule.When.Languages) > 0 {
				matched := false
				for _, ruleLang := range rule.When.Languages {
					if ruleLang == lang {
						matched = true
						break
					}
				}
				if !matched {
					return []string{}, nil
				}
			}
		}
		return []string{basePath}, nil
	}

	// Directory - use selector to find files
	if rule.When == nil {
		// No selector, use all files in directory
		return v.selector.SelectFiles(nil)
	}

	return v.selector.SelectFiles(rule.When)
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
