package golangcilint

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/DevSymphony/sym-cli/internal/linter"
	"github.com/DevSymphony/sym-cli/internal/llm"
	"github.com/DevSymphony/sym-cli/pkg/schema"
)

// Compile-time interface check
var _ linter.Converter = (*Converter)(nil)

// Converter converts rules to golangci-lint configuration using LLM
type Converter struct{}

// NewConverter creates a new golangci-lint converter
func NewConverter() *Converter {
	return &Converter{}
}

func (c *Converter) Name() string {
	return "golangci-lint"
}

func (c *Converter) SupportedLanguages() []string {
	return []string{"go"}
}

// GetLLMDescription returns a description of golangci-lint's capabilities for LLM routing
func (c *Converter) GetLLMDescription() string {
	return `golangci-lint meta-linter for Go - runs 50+ linters in parallel
  - CAN: Error checking, code quality, security, performance, complexity, style, naming
  - Includes: errcheck, govet, staticcheck, gosec, ineffassign, unused, goconst, gocyclo, and many more
  - ALWAYS use for Go code rules`
}

// GetRoutingHints returns routing rules for LLM to decide when to use golangci-lint
func (c *Converter) GetRoutingHints() []string {
	return []string{
		"For Go code quality (errors, bugs, complexity) → use golangci-lint",
		"For Go security analysis → use golangci-lint",
		"For Go style and formatting → use golangci-lint",
		"For Go performance issues → use golangci-lint",
		"ALWAYS use golangci-lint for Go language rules",
	}
}

// golangciLinterData holds golangci-lint-specific conversion data
type golangciLinterData struct {
	Linter   string                 `json:"linter"`
	Settings map[string]interface{} `json:"settings"`
}

// ConvertSingleRule converts ONE user rule to golangci-lint linter configuration.
// Returns (result, nil) on success,
//
//	(nil, nil) if rule cannot be converted by golangci-lint (skip),
//	(nil, error) on actual conversion error.
func (c *Converter) ConvertSingleRule(ctx context.Context, rule schema.UserRule, provider llm.Provider) (*linter.SingleRuleResult, error) {
	if provider == nil {
		return nil, fmt.Errorf("LLM provider is required")
	}

	linterName, settings, err := c.convertToGolangciLinter(ctx, rule, provider)
	if err != nil {
		return nil, err
	}

	// If linter name is empty, this rule cannot be converted by golangci-lint
	if linterName == "" {
		return nil, nil
	}

	return &linter.SingleRuleResult{
		RuleID: rule.ID,
		Data: golangciLinterData{
			Linter:   linterName,
			Settings: settings,
		},
	}, nil
}

// BuildConfig assembles golangci-lint configuration from successful rule conversions.
func (c *Converter) BuildConfig(results []*linter.SingleRuleResult) (*linter.LinterConfig, error) {
	if len(results) == 0 {
		return nil, nil
	}

	// Collect enabled linters and their settings
	enabledLinters := make(map[string]bool)
	linterSettings := make(map[string]map[string]interface{})

	for _, r := range results {
		data, ok := r.Data.(golangciLinterData)
		if !ok {
			continue
		}

		// Add to enabled linters
		enabledLinters[data.Linter] = true

		// Merge settings
		if len(data.Settings) > 0 {
			if _, exists := linterSettings[data.Linter]; !exists {
				linterSettings[data.Linter] = make(map[string]interface{})
			}
			for k, v := range data.Settings {
				linterSettings[data.Linter][k] = v
			}
		}
	}

	if len(enabledLinters) == 0 {
		return nil, nil
	}

	// Convert map to slice for YAML
	enabledLintersList := make([]string, 0, len(enabledLinters))
	for linterName := range enabledLinters {
		enabledLintersList = append(enabledLintersList, linterName)
	}

	// Build golangci-lint config structure
	config := golangciConfig{
		Version: "2",
		Linters: golangciLinters{
			Enable: enabledLintersList,
		},
	}

	// Add linter settings if any
	if len(linterSettings) > 0 {
		config.LintersSettings = linterSettings
	}

	// Marshal to YAML
	content, err := yaml.Marshal(config)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal YAML config: %w", err)
	}

	// Validate config
	if err := validateConfig(&config); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	return &linter.LinterConfig{
		Filename: ".golangci.yml",
		Content:  content,
		Format:   "yaml",
	}, nil
}

// golangciConfig represents the .golangci.yml configuration structure
type golangciConfig struct {
	Version         string                            `yaml:"version"`
	Linters         golangciLinters                   `yaml:"linters"`
	LintersSettings map[string]map[string]interface{} `yaml:"linters-settings,omitempty"`
}

// golangciLinters represents the linters section
type golangciLinters struct {
	Enable []string `yaml:"enable"`
}

// convertToGolangciLinter converts a single user rule to golangci-lint linter using LLM
func (c *Converter) convertToGolangciLinter(ctx context.Context, rule schema.UserRule, provider llm.Provider) (string, map[string]interface{}, error) {
	systemPrompt := `You are a golangci-lint configuration expert. Convert natural language Go coding rules to golangci-lint linter names and settings.

Return ONLY a JSON object (no markdown fences) with this structure:
{
  "linter": "linter_name",
  "settings": {
    "key": "value"
  }
}

Available golangci-lint linters and their purposes:

Error Handling:
- errcheck: Check for unchecked errors
- wrapcheck: Checks that errors from external packages are wrapped

Code Quality:
- govet: Vet examines Go source code and reports suspicious constructs
- staticcheck: Advanced static analysis (find bugs, performance issues, etc.)
- ineffassign: Detects ineffectual assignments
- unused: Checks for unused constants, variables, functions, and types
- deadcode: Finds unused code

Complexity:
- gocyclo: Computes cyclomatic complexities (default threshold: 30)
- gocognit: Computes cognitive complexities (more accurate than cyclomatic)
- nestif: Reports deeply nested if statements
- funlen: Tool for detecting long functions

Performance:
- prealloc: Finds slice declarations that could potentially be preallocated

Security:
- gosec: Inspects source code for security problems

Style & Formatting:
- gofmt: Checks whether code was gofmt-ed
- goimports: Checks import formatting and missing imports
- stylecheck: Replacement for golint (enforces Go style guide)
- revive: Fast, configurable, extensible Go linter

Naming & Best Practices:
- goconst: Finds repeated strings that could be replaced by a constant
- misspell: Finds commonly misspelled English words in comments
- unconvert: Removes unnecessary type conversions

Other Useful Linters:
- bodyclose: Checks whether HTTP response body is closed successfully
- dupl: Tool for code clone detection
- exportloopref: Checks for pointers to enclosing loop variables
- gocritic: Provides diagnostics that check for bugs, performance and style issues
- godot: Checks if comments end in a period
- goprintffuncname: Checks that printf-like functions are named with f at the end
- gosimple: Linter for Go source code that specializes in simplifying code
- noctx: Finds sending http request without context.Context
- rowserrcheck: Checks whether Err of rows is checked successfully
- sqlclosecheck: Checks that sql.Rows and sql.Stmt are closed
- typecheck: Like the front-end of a Go compiler, parses and type-checks Go code

Settings examples:
- gocyclo: {"min-complexity": 10}
- gocognit: {"min-complexity": 15}
- nestif: {"min-complexity": 4}
- gosec: {"severity": "medium", "confidence": "medium"}
- errcheck: {"check-type-assertions": true, "check-blank": true}
- goconst: {"min-len": 3, "min-occurrences": 3}
- funlen: {"lines": 60, "statements": 40}
- revive: {"severity": "warning"}

CRITICAL RULES:
1. ONLY use actual golangci-lint linters - do NOT invent linter names
2. If no linter can enforce this requirement, return linter as empty string ""
3. When in doubt, return empty linter name - better to skip than use wrong linter
4. Settings are optional - only include if the rule specifies parameters

Examples:

Input: "Check for unchecked errors"
Output:
{
  "linter": "errcheck",
  "settings": {}
}

Input: "Cyclomatic complexity should not exceed 15"
Output:
{
  "linter": "gocyclo",
  "settings": {"min-complexity": 15}
}

Input: "Detect security vulnerabilities"
Output:
{
  "linter": "gosec",
  "settings": {}
}

Input: "Find unused variables and functions"
Output:
{
  "linter": "unused",
  "settings": {}
}

Input: "Code should follow Go formatting standards"
Output:
{
  "linter": "gofmt",
  "settings": {}
}

Input: "File names must be snake_case"
Output:
{
  "linter": "",
  "settings": {}
}
(Reason: golangci-lint does not check file names)

Input: "Maximum 3 database connections"
Output:
{
  "linter": "",
  "settings": {}
}
(Reason: No linter for runtime resource limits)`

	userPrompt := fmt.Sprintf("Convert this Go coding rule to golangci-lint linter:\n\n%s", rule.Say)

	// Call LLM
	prompt := systemPrompt + "\n\n" + userPrompt
	response, err := provider.Execute(ctx, prompt, llm.JSON)
	if err != nil {
		return "", nil, fmt.Errorf("LLM call failed: %w", err)
	}

	// Parse response
	response = linter.CleanJSONResponse(response)

	if response == "" {
		return "", nil, fmt.Errorf("LLM returned empty response")
	}

	var result struct {
		Linter   string                 `json:"linter"`
		Settings map[string]interface{} `json:"settings"`
	}

	if err := json.Unmarshal([]byte(response), &result); err != nil {
		return "", nil, fmt.Errorf("failed to parse LLM response: %w (response: %.100s)", err, response)
	}

	// If linter is empty, this rule cannot be converted
	if result.Linter == "" {
		return "", nil, nil
	}

	// Validate linter name
	if !isValidLinter(result.Linter) {
		return "", nil, fmt.Errorf("invalid linter name: %s", result.Linter)
	}

	return result.Linter, result.Settings, nil
}

// validateConfig validates the golangci-lint configuration
func validateConfig(config *golangciConfig) error {
	if config.Version == "" {
		return fmt.Errorf("version is required")
	}

	if len(config.Linters.Enable) == 0 {
		return fmt.Errorf("no linters enabled")
	}

	return nil
}

// isValidLinter checks if the linter name is a known golangci-lint linter
func isValidLinter(name string) bool {
	validLinters := map[string]bool{
		"errcheck": true, "govet": true, "staticcheck": true, "gosec": true,
		"ineffassign": true, "unused": true, "deadcode": true,
		"goconst": true, "gocyclo": true, "gocognit": true, "nestif": true, "funlen": true,
		"prealloc": true, "gofmt": true, "goimports": true, "stylecheck": true, "revive": true,
		"misspell": true, "unconvert": true, "bodyclose": true, "dupl": true,
		"exportloopref": true, "gocritic": true, "godot": true, "goprintffuncname": true,
		"gosimple": true, "noctx": true, "rowserrcheck": true, "sqlclosecheck": true,
		"typecheck": true, "wrapcheck": true,
		// Add more as needed
	}

	return validLinters[strings.ToLower(name)]
}
