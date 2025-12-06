package validator

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/DevSymphony/sym-cli/internal/adapter"
	adapterRegistry "github.com/DevSymphony/sym-cli/internal/adapter/registry"
	"github.com/DevSymphony/sym-cli/internal/git"
	"github.com/DevSymphony/sym-cli/internal/llm"
	"github.com/DevSymphony/sym-cli/internal/roles"
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
	adapterRegistry *adapterRegistry.Registry
	workDir         string
	symDir          string // .sym directory for config files
	selector        *FileSelector
	ctx             context.Context
	ctxCancel       context.CancelFunc
	llmClient       *llm.Client
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
		policy:          policy,
		verbose:         verbose,
		adapterRegistry: adapterRegistry.Global(),
		workDir:         workDir,
		symDir:          symDir,
		selector:        NewFileSelector(workDir),
		ctx:             ctx,
		ctxCancel:       cancel,
		llmClient:       nil,
	}
}

// NewValidatorWithWorkDir creates a validator with a custom working directory
// symDir is automatically set to workDir/.sym
func NewValidatorWithWorkDir(policy *schema.CodePolicy, verbose bool, workDir string) *Validator {
	symDir := filepath.Join(workDir, ".sym")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)

	return &Validator{
		policy:          policy,
		verbose:         verbose,
		adapterRegistry: adapterRegistry.Global(),
		workDir:         workDir,
		symDir:          symDir,
		selector:        NewFileSelector(workDir),
		ctx:             ctx,
		ctxCancel:       cancel,
		llmClient:       nil,
	}
}

// SetLLMClient sets the LLM client for this validator
func (v *Validator) SetLLMClient(client *llm.Client) {
	v.llmClient = client
}

// executeRule executes a rule using the appropriate adapter
func (v *Validator) executeRule(engineName string, rule schema.PolicyRule, files []string) ([]Violation, error) {
	// Special case: LLM validator
	if engineName == "llm-validator" {
		return v.executeLLMRule(rule, files)
	}

	// Get adapter directly by tool name (e.g., "eslint", "prettier", "tsc")
	adp, err := v.adapterRegistry.GetAdapter(engineName)
	if err != nil {
		return nil, fmt.Errorf("adapter not found: %s: %w", engineName, err)
	}

	// Check if adapter is available
	if err := adp.CheckAvailability(v.ctx); err != nil {
		if v.verbose {
			fmt.Printf("   📦 Installing %s...\n", adp.Name())
		}
		if err := adp.Install(v.ctx, adapter.InstallConfig{
			ToolsDir: filepath.Join(os.Getenv("HOME"), ".sym", "tools"),
		}); err != nil {
			return nil, fmt.Errorf("failed to install %s: %w", adp.Name(), err)
		}
	}

	// Generate config from rule or use existing .sym config
	config, err := v.getAdapterConfig(adp.Name(), rule)
	if err != nil {
		return nil, fmt.Errorf("failed to generate config: %w", err)
	}

	// Execute adapter
	output, err := adp.Execute(v.ctx, config, files)
	if err != nil {
		return nil, fmt.Errorf("adapter execution failed: %w", err)
	}

	// Parse execution duration
	var execMs int64
	if output.Duration != "" {
		if duration, parseErr := time.ParseDuration(output.Duration); parseErr == nil {
			execMs = duration.Milliseconds()
		}
	}

	// Parse output to violations
	adapterViolations, err := adp.ParseOutput(output)
	if err != nil {
		return nil, fmt.Errorf("failed to parse output: %w", err)
	}

	// Convert adapter violations to validator violations
	violations := make([]Violation, 0, len(adapterViolations))
	for _, av := range adapterViolations {
		violations = append(violations, Violation{
			RuleID:      rule.ID,
			Severity:    rule.Severity,
			Message:     av.Message,
			File:        av.File,
			Line:        av.Line,
			Column:      av.Column,
			RawOutput:   output.Stdout,
			RawError:    output.Stderr,
			ToolName:    adp.Name(),
			ExecutionMs: execMs,
		})
	}

	// If verbose, log the raw output
	if v.verbose && output.Stdout != "" {
		fmt.Printf("   📋 Raw output (%dms):\n", execMs)
		if len(output.Stdout) > 500 {
			fmt.Printf("   %s...\n", output.Stdout[:500])
		} else {
			fmt.Printf("   %s\n", output.Stdout)
		}
	}

	return violations, nil
}

// executeLLMRule executes an LLM-based rule
func (v *Validator) executeLLMRule(rule schema.PolicyRule, files []string) ([]Violation, error) {
	if v.llmClient == nil {
		return nil, fmt.Errorf("LLM client not configured")
	}

	// Validate required fields for LLM validator
	if rule.Desc == "" {
		return nil, fmt.Errorf("LLM validator requires 'desc' field in rule %s", rule.ID)
	}

	// Check if When selector exists for file filtering
	if rule.When == nil && len(files) == 0 {
		if v.verbose {
			fmt.Printf("⚠️  LLM rule %s has no 'when' selector and no files provided\n", rule.ID)
		}
	}

	// Create a consolidated ToolOutput for all files
	var allResponses strings.Builder
	var allErrors strings.Builder
	startTime := time.Now()
	totalViolations := 0

	violations := make([]Violation, 0)

	for fileIdx, file := range files {
		// Read file content
		content, err := os.ReadFile(file)
		if err != nil {
			allErrors.WriteString(fmt.Sprintf("[File %d/%d] %s: %v\n", fileIdx+1, len(files), file, err))
			continue
		}

		// Build LLM prompt
		systemPrompt := `You are a code reviewer. Check if the code violates the given coding convention.

Respond with JSON only:
{
  "violates": true/false,
  "description": "explanation of violation if any",
  "suggestion": "how to fix it if violated"
}`

		userPrompt := fmt.Sprintf(`File: %s

Coding Convention:
%s

Code:
%s

Does this code violate the convention?`, file, rule.Desc, string(content))

		// Call LLM
		fileStartTime := time.Now()
		response, err := v.llmClient.Request(systemPrompt, userPrompt).Execute(v.ctx)
		fileExecMs := time.Since(fileStartTime).Milliseconds()

		// Record response in consolidated output
		allResponses.WriteString(fmt.Sprintf("=== File %d/%d: %s (%dms) ===\n", fileIdx+1, len(files), file, fileExecMs))
		if err != nil {
			allErrors.WriteString(fmt.Sprintf("[File %d/%d] LLM error: %v\n", fileIdx+1, len(files), err))
			allResponses.WriteString(fmt.Sprintf("Error: %v\n\n", err))
			continue
		}

		allResponses.WriteString(response)
		allResponses.WriteString("\n\n")

		// Parse response
		result := parseValidationResponse(response)
		if result.Violates {
			totalViolations++
			message := result.Description
			if result.Suggestion != "" {
				message += fmt.Sprintf(" | Suggestion: %s", result.Suggestion)
			}

			// Store violation (will update with consolidated output later)
			violations = append(violations, Violation{
				RuleID:      rule.ID,
				Severity:    rule.Severity,
				Message:     message,
				File:        file,
				RawOutput:   response, // Individual LLM response for this file
				ToolName:    "llm-validator",
				ExecutionMs: fileExecMs,
			})
		}
	}

	// Calculate total execution time
	totalExecMs := time.Since(startTime).Milliseconds()

	// Create consolidated ToolOutput in linter format
	consolidatedStdout := allResponses.String()
	consolidatedStderr := allErrors.String()

	// Update all violations with consolidated output (like how linters include full output)
	// Each violation gets the full stdout/stderr, just like ESLint violations all share the same JSON output
	for i := range violations {
		// Keep individual response in RawOutput, but add consolidated info
		violations[i].RawOutput = fmt.Sprintf("=== Individual Response ===\n%s\n\n=== Consolidated Output ===\n%s",
			violations[i].RawOutput, consolidatedStdout)
		if consolidatedStderr != "" {
			violations[i].RawError = consolidatedStderr
		}
	}

	// If verbose, log the consolidated output (like adapter verbose output)
	if v.verbose && consolidatedStdout != "" {
		fmt.Printf("   📋 LLM consolidated output (%dms):\n", totalExecMs)
		fmt.Printf("   - Checked: %d file(s)\n", len(files))
		fmt.Printf("   - Violations: %d\n", totalViolations)
		// Show first 500 chars of consolidated output
		if len(consolidatedStdout) > 500 {
			fmt.Printf("   %s...\n", consolidatedStdout[:500])
		} else {
			fmt.Printf("   %s\n", consolidatedStdout)
		}
		if consolidatedStderr != "" {
			fmt.Printf("   ⚠️  Errors: %s\n", consolidatedStderr)
		}
	}

	return violations, nil
}

// getAdapterConfig gets config for an adapter
// First checks .sym directory for existing config files, then generates from rule
func (v *Validator) getAdapterConfig(adapterName string, rule schema.PolicyRule) ([]byte, error) {
	// Check for existing config in .sym directory (using registry)
	configFile := v.adapterRegistry.GetConfigFile(adapterName)
	if configFile != "" {
		configPath := filepath.Join(v.symDir, configFile)
		if data, err := os.ReadFile(configPath); err == nil {
			if v.verbose {
				fmt.Printf("   📄 Using config from %s\n", configPath)
			}
			return data, nil
		}
	}

	// Otherwise, generate config from rule
	config := make(map[string]interface{})

	// Copy rule.Check to config
	for k, val := range rule.Check {
		if k != "engine" && k != "desc" {
			config[k] = val
		}
	}

	// Add rule description
	if rule.Desc != "" {
		config["description"] = rule.Desc
	}

	return json.Marshal(config)
}

// getEngineName extracts the engine name from a rule
func getEngineName(rule schema.PolicyRule) string {
	if engine, ok := rule.Check["engine"].(string); ok {
		return engine
	}
	return ""
}

// ValidateChanges validates git changes using adapters directly
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

	// Check RBAC permissions first
	if v.policy.Enforce.RBACConfig != nil && v.policy.Enforce.RBACConfig.Enabled {
		username, err := git.GetCurrentUser()
		if err == nil {
			if v.verbose {
				fmt.Printf("🔐 Checking RBAC permissions for user: %s\n", username)
			}

			changedFiles := make([]string, 0, len(changes))
			for _, change := range changes {
				if change.Status != "D" {
					changedFiles = append(changedFiles, change.FilePath)
				}
			}

			if len(changedFiles) > 0 {
				rbacResult, err := roles.ValidateFilePermissions(username, changedFiles)
				if err == nil && !rbacResult.Allowed {
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
				}
			}
		}
	}

	if v.verbose {
		fmt.Printf("🔍 Validating %d change(s) against %d rule(s)...\n", len(changes), len(v.policy.Rules))
	}

	// Validate each enabled rule
	for _, rule := range v.policy.Rules {
		if !rule.Enabled {
			continue
		}

		engineName := getEngineName(rule)
		if engineName == "" {
			continue
		}

		// Filter changes that match this rule's selector
		relevantChanges := v.filterChangesForRule(changes, &rule)
		if len(relevantChanges) == 0 {
			continue
		}

		if v.verbose {
			fmt.Printf("   Rule %s (%s): checking %d change(s)...\n", rule.ID, engineName, len(relevantChanges))
		}

		// For LLM engine, use parallel processing
		if engineName == "llm-validator" {
			v.validateLLMChanges(ctx, relevantChanges, rule, result)
		} else {
			// For adapter-based engines, validate files
			for _, change := range relevantChanges {
				if change.Status == "D" {
					continue
				}

				result.Checked++

				violations, err := v.executeRule(engineName, rule, []string{change.FilePath})
				if err != nil {
					// Always log errors to stderr (not just in verbose mode)
					fmt.Fprintf(os.Stderr, "⚠️  Validation failed for rule %s (%s): %v\n", rule.ID, engineName, err)
					// Track error in result for MCP response
					result.Errors = append(result.Errors, ValidationError{
						RuleID:  rule.ID,
						Engine:  engineName,
						Message: err.Error(),
					})
					continue
				}

				if len(violations) > 0 {
					result.Failed++
					result.Violations = append(result.Violations, violations...)
				} else {
					result.Passed++
				}
			}
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

// validateLLMChanges validates changes using LLM in parallel
func (v *Validator) validateLLMChanges(ctx context.Context, changes []GitChange, rule schema.PolicyRule, result *ValidationResult) {
	if v.llmClient == nil {
		return
	}

	var wg sync.WaitGroup
	var mu sync.Mutex
	llmValidator := NewLLMValidator(v.llmClient, v.policy)

	for _, change := range changes {
		if change.Status == "D" {
			continue
		}

		addedLines := ExtractAddedLines(change.Diff)
		if len(addedLines) == 0 && strings.TrimSpace(change.Diff) != "" {
			addedLines = strings.Split(change.Diff, "\n")
		}

		if len(addedLines) == 0 {
			mu.Lock()
			result.Checked++
			result.Passed++
			mu.Unlock()
			continue
		}

		mu.Lock()
		result.Checked++
		mu.Unlock()

		wg.Add(1)
		go func(ch GitChange, lines []string, r schema.PolicyRule) {
			defer wg.Done()

			violation, err := llmValidator.CheckRule(ctx, ch, lines, r)
			if err != nil {
				return
			}

			mu.Lock()
			defer mu.Unlock()
			if violation != nil {
				result.Failed++
				result.Violations = append(result.Violations, *violation)
			} else {
				result.Passed++
			}
		}(change, addedLines, rule)
	}

	wg.Wait()
}

// filterChangesForRule filters git changes that match the rule's selector
func (v *Validator) filterChangesForRule(changes []GitChange, rule *schema.PolicyRule) []GitChange {
	if rule.When == nil {
		return changes
	}

	var filtered []GitChange
	for _, change := range changes {
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
