package validator

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/DevSymphony/sym-cli/internal/linter"
	"github.com/DevSymphony/sym-cli/internal/llm"
	"github.com/DevSymphony/sym-cli/internal/roles"
	"github.com/DevSymphony/sym-cli/internal/util/git"
	"github.com/DevSymphony/sym-cli/pkg/schema"
)

// Violation represents a policy violation
type Violation struct {
	RuleID   string
	Severity string
	Message  string
	File     string
	Line     int
	Column   int
	// Raw output from linter/validator
	RawOutput   string // stdout from adapter execution
	RawError    string // stderr from adapter execution
	ToolName    string // which tool detected this (eslint, prettier, llm-validator, etc.)
	ExecutionMs int64  // execution time in milliseconds
}

// Validator validates code against policy using adapters directly
// This replaces the old engine-based architecture
type Validator struct {
	policy          *schema.CodePolicy
	verbose         bool
	linterRegistry  *linter.Registry
	workDir         string
	symDir          string // .sym directory for config files
	ctx             context.Context
	ctxCancel       context.CancelFunc
	llmProvider     llm.Provider
	llmProviderInfo *llm.ProviderInfo // Provider metadata including mode
}

// NewValidator creates a new adapter-based validator
func NewValidator(policy *schema.CodePolicy, verbose bool) *Validator {
	workDir, err := os.Getwd()
	if err != nil {
		workDir = "."
	}

	symDir := filepath.Join(workDir, ".sym")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)

	return &Validator{
		policy:         policy,
		verbose:        verbose,
		linterRegistry: linter.Global(),
		workDir:        workDir,
		symDir:         symDir,
		ctx:            ctx,
		ctxCancel:      cancel,
		llmProvider:    nil,
	}
}

// NewValidatorWithWorkDir creates a validator with a custom working directory
// symDir is automatically set to workDir/.sym
func NewValidatorWithWorkDir(policy *schema.CodePolicy, verbose bool, workDir string) *Validator {
	symDir := filepath.Join(workDir, ".sym")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)

	return &Validator{
		policy:         policy,
		verbose:        verbose,
		linterRegistry: linter.Global(),
		workDir:        workDir,
		symDir:         symDir,
		ctx:            ctx,
		ctxCancel:      cancel,
		llmProvider:    nil,
	}
}

// SetLLMProvider sets the LLM provider for this validator
func (v *Validator) SetLLMProvider(provider llm.Provider) {
	v.llmProvider = provider
	// Also store provider info for mode-based execution decisions
	if provider != nil {
		v.llmProviderInfo = llm.GetProviderInfo(provider.Name())
	}
}

// getEngineName extracts the engine name from a rule
func getEngineName(rule schema.PolicyRule) string {
	if engine, ok := rule.Check["engine"].(string); ok {
		return engine
	}
	return ""
}

// ruleGroup groups rules by engine for batch execution
type ruleGroup struct {
	engineName string
	rules      []schema.PolicyRule
	files      map[string]bool // Use set for deduplication
	changes    []git.Change    // Original changes for LLM rules
}

// getDefaultConcurrency returns the default concurrency level (CPU/2, min 1, max 8)
func getDefaultConcurrency() int {
	concurrency := runtime.NumCPU() / 2
	if concurrency < 1 {
		concurrency = 1
	}
	if concurrency > 8 {
		concurrency = 8
	}
	return concurrency
}

// groupRulesByEngine groups rules by their engine name
func (v *Validator) groupRulesByEngine(rules []schema.PolicyRule, changes []git.Change) map[string]*ruleGroup {
	groups := make(map[string]*ruleGroup)

	for _, rule := range rules {
		if !rule.Enabled {
			continue
		}

		engineName := getEngineName(rule)
		if engineName == "" {
			continue
		}

		// Filter changes relevant to this rule
		relevantChanges := v.filterChangesForRule(changes, &rule)
		if len(relevantChanges) == 0 {
			continue
		}

		if groups[engineName] == nil {
			groups[engineName] = &ruleGroup{
				engineName: engineName,
				rules:      []schema.PolicyRule{},
				files:      make(map[string]bool),
				changes:    []git.Change{},
			}
		}

		groups[engineName].rules = append(groups[engineName].rules, rule)

		// Collect unique files and changes
		for _, change := range relevantChanges {
			if change.Status != "D" {
				groups[engineName].files[change.FilePath] = true
				// Track changes for LLM rules (they need the diff)
				if engineName == "llm-validator" {
					// Check if this change is already added
					found := false
					for _, existing := range groups[engineName].changes {
						if existing.FilePath == change.FilePath {
							found = true
							break
						}
					}
					if !found {
						groups[engineName].changes = append(groups[engineName].changes, change)
					}
				}
			}
		}
	}

	return groups
}

// createExecutionUnits creates execution units from rule groups
// - Linter: all files + all rules = 1 unit
// - LLM (agentic_single mode): all files + all rules = 1 unit (Claude Code, Gemini CLI)
// - LLM (parallel_api mode): 1 file × 1 rule = N units (OpenAI API)
func (v *Validator) createExecutionUnits(groups map[string]*ruleGroup) []executionUnit {
	var units []executionUnit

	for engineName, group := range groups {
		if engineName == "llm-validator" {
			units = append(units, v.createLLMExecutionUnits(group)...)
		} else {
			// Linter: all files + all rules = 1 unit
			files := make([]string, 0, len(group.files))
			for f := range group.files {
				files = append(files, f)
			}
			units = append(units, &linterExecutionUnit{
				engineName: engineName,
				rules:      group.rules,
				files:      files,
				registry:   v.linterRegistry,
				symDir:     v.symDir,
				verbose:    v.verbose,
			})
		}
	}

	return units
}

// createLLMExecutionUnits creates execution units for LLM validation
// based on the provider's mode (agentic_single vs parallel_api)
func (v *Validator) createLLMExecutionUnits(group *ruleGroup) []executionUnit {
	var units []executionUnit

	// Check provider mode (default to parallel for backward compatibility)
	mode := llm.ModeParallelAPI
	var profile llm.ProviderProfile

	if v.llmProviderInfo != nil {
		mode = v.llmProviderInfo.Mode
		profile = v.llmProviderInfo.Profile
	}

	switch mode {
	case llm.ModeAgenticSingle:
		// Agentic mode: all files + all rules = 1 unit
		// Agent-based CLIs (Claude Code, Gemini CLI) can handle complex tasks
		// internally, so we send everything in a single call
		if len(group.changes) > 0 && len(group.rules) > 0 {
			units = append(units, &agenticLLMExecutionUnit{
				rules:    group.rules,
				changes:  group.changes,
				provider: v.llmProvider,
				policy:   v.policy,
				profile:  profile,
				verbose:  v.verbose,
			})
		}

	default: // ModeParallelAPI
		// Parallel API mode: 1 file × 1 rule = separate units
		// Traditional APIs (OpenAI) process multiple parallel requests
		for _, rule := range group.rules {
			ruleChanges := v.filterChangesForRule(group.changes, &rule)
			for _, change := range ruleChanges {
				if change.Status == "D" {
					continue
				}
				units = append(units, &llmExecutionUnit{
					rule:     rule,
					change:   change,
					provider: v.llmProvider,
					policy:   v.policy,
					verbose:  v.verbose,
				})
			}
		}
	}

	return units
}

// executeUnitsParallel executes units in parallel with semaphore-based concurrency
func (v *Validator) executeUnitsParallel(ctx context.Context, units []executionUnit) ([]Violation, []ValidationError) {
	var wg sync.WaitGroup
	var mu sync.Mutex

	concurrency := getDefaultConcurrency()
	sem := make(chan struct{}, concurrency)

	var allViolations []Violation
	var allErrors []ValidationError

	for _, unit := range units {
		wg.Add(1)
		go func(u executionUnit) {
			defer wg.Done()

			// Acquire semaphore
			select {
			case sem <- struct{}{}:
			case <-ctx.Done():
				mu.Lock()
				allErrors = append(allErrors, ValidationError{
					RuleID:  strings.Join(u.GetRuleIDs(), ","),
					Engine:  u.GetEngineName(),
					Message: ctx.Err().Error(),
				})
				mu.Unlock()
				return
			}
			defer func() { <-sem }()

			violations, err := u.Execute(ctx)

			mu.Lock()
			defer mu.Unlock()

			if err != nil {
				allErrors = append(allErrors, ValidationError{
					RuleID:  strings.Join(u.GetRuleIDs(), ","),
					Engine:  u.GetEngineName(),
					Message: err.Error(),
				})
				return
			}

			allViolations = append(allViolations, violations...)
		}(unit)
	}

	wg.Wait()

	return allViolations, allErrors
}

// ValidateChanges validates git changes using adapters directly
// Rules are grouped by engine for efficient batch execution:
// - Linter rules (eslint, pylint, etc.) are batched into single executions per linter
// - LLM rules are executed as individual units
// Execution units run in parallel with CPU/2 concurrency (min 1, max 8)
func (v *Validator) ValidateChanges(ctx context.Context, changes []git.Change) (*ValidationResult, error) {
	if v.policy == nil {
		return nil, fmt.Errorf("policy is not loaded")
	}

	result := &ValidationResult{
		Violations: make([]Violation, 0),
		Checked:    0,
		Passed:     0,
		Failed:     0,
	}

	// Phase 1: Check RBAC permissions first
	if v.policy.Enforce.RBACConfig != nil && v.policy.Enforce.RBACConfig.Enabled {
		currentRole, err := roles.GetCurrentRole()
		if err == nil && currentRole != "" {
			if v.verbose {
				fmt.Printf("🔐 Checking RBAC permissions for role: %s\n", currentRole)
			}

			changedFiles := make([]string, 0, len(changes))
			for _, change := range changes {
				if change.Status != "D" {
					changedFiles = append(changedFiles, change.FilePath)
				}
			}

			if len(changedFiles) > 0 {
				rbacResult, err := roles.ValidateFilePermissionsForRole(currentRole, changedFiles)
				if err == nil && !rbacResult.Allowed {
					for _, deniedFile := range rbacResult.DeniedFiles {
						result.Violations = append(result.Violations, Violation{
							RuleID:   "rbac-permission-denied",
							Severity: "error",
							Message:  fmt.Sprintf("Role '%s' does not have permission to modify this file", currentRole),
							File:     deniedFile,
							Line:     0,
							Column:   0,
						})
						result.Failed++
					}
				}
			}
		}
	}

	// Phase 2: Group rules by engine
	groups := v.groupRulesByEngine(v.policy.Rules, changes)

	// Phase 3: Create execution units
	units := v.createExecutionUnits(groups)

	if v.verbose {
		// Count files to check
		totalFiles := 0
		for _, change := range changes {
			if change.Status != "D" {
				totalFiles++
			}
		}
		fmt.Printf("🔍 Validating %d file(s) against %d rule(s) in %d execution unit(s) (concurrency: %d)...\n",
			totalFiles, len(v.policy.Rules), len(units), getDefaultConcurrency())
		for _, unit := range units {
			fmt.Printf("   - %s: %d rule(s), %d file(s)\n",
				unit.GetEngineName(), len(unit.GetRuleIDs()), len(unit.GetFiles()))
		}
	}

	// Phase 4: Execute units in parallel
	violations, errors := v.executeUnitsParallel(ctx, units)

	// Aggregate results
	result.Violations = append(result.Violations, violations...)
	result.Errors = append(result.Errors, errors...)

	// Calculate statistics
	// Count unique files that were checked
	checkedFiles := make(map[string]bool)
	for _, unit := range units {
		for _, file := range unit.GetFiles() {
			checkedFiles[file] = true
		}
	}
	result.Checked = len(checkedFiles)

	// Count failures based on violations
	failedFiles := make(map[string]bool)
	for _, v := range result.Violations {
		failedFiles[v.File] = true
	}
	result.Failed = len(failedFiles)
	result.Passed = result.Checked - result.Failed

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
func (v *Validator) filterChangesForRule(changes []git.Change, rule *schema.PolicyRule) []git.Change {
	if rule.When == nil {
		return changes
	}

	var filtered []git.Change
	for _, change := range changes {
		if len(rule.When.Languages) > 0 {
			lang := getLanguageFromFile(change.FilePath)
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

		filtered = append(filtered, change)
	}

	return filtered
}

// Close cleans up validator resources
func (v *Validator) Close() error {
	if v.ctxCancel != nil {
		v.ctxCancel()
	}
	return nil
}

// getLanguageFromFile determines the programming language from a file path
func getLanguageFromFile(filePath string) string {
	ext := filepath.Ext(filePath)

	switch ext {
	case ".js", ".mjs", ".cjs":
		return "javascript"
	case ".ts", ".mts", ".cts":
		return "typescript"
	case ".jsx":
		return "jsx"
	case ".tsx":
		return "tsx"
	case ".go":
		return "go"
	case ".py":
		return "python"
	case ".java":
		return "java"
	case ".c", ".h":
		return "c"
	case ".cpp", ".cc", ".cxx", ".hpp", ".hh", ".hxx":
		return "cpp"
	case ".rs":
		return "rust"
	case ".rb":
		return "ruby"
	case ".php":
		return "php"
	case ".sh", ".bash":
		return "shell"
	default:
		return ""
	}
}
