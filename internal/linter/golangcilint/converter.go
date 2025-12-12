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
	Name        string                 `json:"name"`
	IsFormatter bool                   `json:"is_formatter"`
	Settings    map[string]interface{} `json:"settings"`
}

// ConvertSingleRule converts ONE user rule to golangci-lint linter/formatter configuration.
// Returns (result, nil) on success,
//
//	(nil, nil) if rule cannot be converted by golangci-lint (skip),
//	(nil, error) on actual conversion error.
func (c *Converter) ConvertSingleRule(ctx context.Context, rule schema.UserRule, provider llm.Provider) (*linter.SingleRuleResult, error) {
	if provider == nil {
		return nil, fmt.Errorf("LLM provider is required")
	}

	name, isFormatter, settings, err := c.convertToGolangciLinter(ctx, rule, provider)
	if err != nil {
		return nil, err
	}

	// If name is empty, this rule cannot be converted by golangci-lint
	if name == "" {
		return nil, nil
	}

	return &linter.SingleRuleResult{
		RuleID: rule.ID,
		Data: golangciLinterData{
			Name:        name,
			IsFormatter: isFormatter,
			Settings:    settings,
		},
	}, nil
}

// BuildConfig assembles golangci-lint configuration from successful rule conversions.
func (c *Converter) BuildConfig(results []*linter.SingleRuleResult) (*linter.LinterConfig, error) {
	if len(results) == 0 {
		return nil, nil
	}

	// Collect enabled linters/formatters and their settings separately
	enabledLinters := make(map[string]bool)
	enabledFormatters := make(map[string]bool)
	linterSettings := make(map[string]map[string]interface{})
	formatterSettings := make(map[string]map[string]interface{})

	for _, r := range results {
		data, ok := r.Data.(golangciLinterData)
		if !ok {
			continue
		}

		if data.IsFormatter {
			// Add to formatters
			enabledFormatters[data.Name] = true
			if len(data.Settings) > 0 {
				if _, exists := formatterSettings[data.Name]; !exists {
					formatterSettings[data.Name] = make(map[string]interface{})
				}
				for k, v := range data.Settings {
					formatterSettings[data.Name][k] = v
				}
			}
		} else {
			// Add to linters
			enabledLinters[data.Name] = true
			if len(data.Settings) > 0 {
				if _, exists := linterSettings[data.Name]; !exists {
					linterSettings[data.Name] = make(map[string]interface{})
				}
				for k, v := range data.Settings {
					linterSettings[data.Name][k] = v
				}
			}
		}
	}

	if len(enabledLinters) == 0 && len(enabledFormatters) == 0 {
		return nil, nil
	}

	// Build golangci-lint config structure
	config := golangciConfig{
		Version: "2",
	}

	// Add linters section if any
	if len(enabledLinters) > 0 {
		enabledLintersList := make([]string, 0, len(enabledLinters))
		for name := range enabledLinters {
			enabledLintersList = append(enabledLintersList, name)
		}
		config.Linters = golangciLinters{
			Enable: enabledLintersList,
		}
		if len(linterSettings) > 0 {
			config.LintersSettings = linterSettings
		}
	}

	// Add formatters section if any
	if len(enabledFormatters) > 0 {
		enabledFormattersList := make([]string, 0, len(enabledFormatters))
		for name := range enabledFormatters {
			enabledFormattersList = append(enabledFormattersList, name)
		}
		config.Formatters = golangciFormatters{
			Enable: enabledFormattersList,
		}
		if len(formatterSettings) > 0 {
			config.FormattersSettings = formatterSettings
		}
	}

	// Validate config
	if err := validateConfig(&config); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	// Marshal to YAML
	content, err := yaml.Marshal(config)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal YAML config: %w", err)
	}

	return &linter.LinterConfig{
		Filename: ".golangci.yml",
		Content:  content,
		Format:   "yaml",
	}, nil
}

// golangciConfig represents the .golangci.yml configuration structure
type golangciConfig struct {
	Version            string                            `yaml:"version"`
	Linters            golangciLinters                   `yaml:"linters,omitempty"`
	LintersSettings    map[string]map[string]interface{} `yaml:"linters-settings,omitempty"`
	Formatters         golangciFormatters                `yaml:"formatters,omitempty"`
	FormattersSettings map[string]map[string]interface{} `yaml:"formatters-settings,omitempty"`
}

// golangciLinters represents the linters section
type golangciLinters struct {
	Enable []string `yaml:"enable,omitempty"`
}

// golangciFormatters represents the formatters section
type golangciFormatters struct {
	Enable []string `yaml:"enable,omitempty"`
}

// convertToGolangciLinter converts a single user rule to golangci-lint linter/formatter using LLM
func (c *Converter) convertToGolangciLinter(ctx context.Context, rule schema.UserRule, provider llm.Provider) (string, bool, map[string]interface{}, error) {
	systemPrompt := `You are a golangci-lint v2 configuration expert. Convert natural language Go coding rules to golangci-lint linter or formatter names and settings.

CRITICAL: In golangci-lint v2, FORMATTERS and LINTERS are DIFFERENT categories.
- FORMATTERS: Tools that format code (gofmt, goimports, gofumpt, gci, golines, swaggo)
- LINTERS: Tools that analyze code for issues (errcheck, govet, gosec, etc.)

Return ONLY a JSON object (no markdown fences) with this structure:
{
  "name": "tool_name",
  "is_formatter": true/false,
  "settings": {}
}

=== FORMATTERS (is_formatter: true) ===
Use these for code formatting rules:
- gofmt: Formats code according to standard Go formatting
- goimports: Formats imports and adds missing imports
- gofumpt: Stricter formatting than gofmt
- gci: Import statement formatting with custom ordering
- golines: Fixes long lines by wrapping them
- swaggo: Formats swaggo comments

=== LINTERS (is_formatter: false) ===

Error Handling:
- errcheck: Check for unchecked errors
- wrapcheck: Checks that errors from external packages are wrapped
- nilerr: Finds code that returns nil even if it checks that the error is not nil

Code Quality:
- govet: Examines Go source code and reports suspicious constructs
- staticcheck: Advanced static analysis (find bugs, performance issues)
- ineffassign: Detects ineffectual assignments
- unused: Checks for unused code
- revive: Fast, configurable linter (replacement for golint)

Complexity:
- gocyclo: Computes cyclomatic complexity
- gocognit: Computes cognitive complexity
- nestif: Reports deeply nested if statements
- funlen: Detects long functions
- cyclop: Checks function and package cyclomatic complexity

Performance:
- prealloc: Finds slice declarations that could be preallocated
- bodyclose: Checks whether HTTP response body is closed

Security:
- gosec: Inspects source code for security problems

Style:
- stylecheck: Enforces Go style guide
- goconst: Finds repeated strings for constants
- misspell: Finds misspelled English words
- godot: Checks if comments end in a period
- nlreturn: Checks for blank lines before return

Other Linters:
- dupl: Code clone detection
- gocritic: Checks for bugs, performance and style issues
- gosimple: Simplifies code
- noctx: Finds HTTP requests without context
- unconvert: Removes unnecessary type conversions
- exportloopref: Checks for pointers to loop variables

Settings examples:
- gocyclo: {"min-complexity": 10}
- gocognit: {"min-complexity": 15}
- funlen: {"lines": 60, "statements": 40}
- gosec: {"severity": "medium", "confidence": "medium"}
- errcheck: {"check-type-assertions": true}
- gofmt: {"simplify": true}
- goimports: {"local-prefixes": "github.com/myorg"}

CRITICAL RULES:
1. ONLY use actual golangci-lint tools - do NOT invent names
2. For formatting rules (gofmt, goimports, etc.) → set is_formatter: true
3. For analysis/linting rules → set is_formatter: false
4. If no tool can enforce this requirement, return name as empty string ""
5. Settings are optional - only include if the rule specifies parameters

Examples:

Input: "Code should follow Go formatting standards"
Output:
{
  "name": "gofmt",
  "is_formatter": true,
  "settings": {}
}

Input: "Imports should be properly organized"
Output:
{
  "name": "goimports",
  "is_formatter": true,
  "settings": {}
}

Input: "Check for unchecked errors"
Output:
{
  "name": "errcheck",
  "is_formatter": false,
  "settings": {}
}

Input: "Cyclomatic complexity should not exceed 15"
Output:
{
  "name": "gocyclo",
  "is_formatter": false,
  "settings": {"min-complexity": 15}
}

Input: "Detect security vulnerabilities"
Output:
{
  "name": "gosec",
  "is_formatter": false,
  "settings": {}
}

Input: "File names must be snake_case"
Output:
{
  "name": "",
  "is_formatter": false,
  "settings": {}
}
(Reason: golangci-lint does not check file names)`

	userPrompt := fmt.Sprintf("Convert this Go coding rule to golangci-lint linter or formatter:\n\n%s", rule.Say)

	// Call LLM
	prompt := systemPrompt + "\n\n" + userPrompt
	response, err := provider.Execute(ctx, prompt, llm.JSON)
	if err != nil {
		return "", false, nil, fmt.Errorf("LLM call failed: %w", err)
	}

	// Parse response
	response = linter.CleanJSONResponse(response)

	if response == "" {
		return "", false, nil, fmt.Errorf("LLM returned empty response")
	}

	var result struct {
		Name        string                 `json:"name"`
		IsFormatter bool                   `json:"is_formatter"`
		Settings    map[string]interface{} `json:"settings"`
	}

	if err := json.Unmarshal([]byte(response), &result); err != nil {
		return "", false, nil, fmt.Errorf("failed to parse LLM response: %w (response: %.100s)", err, response)
	}

	// If name is empty, this rule cannot be converted
	if result.Name == "" {
		return "", false, nil, nil
	}

	// Validate tool name based on type
	if result.IsFormatter {
		if !isValidFormatter(result.Name) {
			return "", false, nil, fmt.Errorf("invalid formatter name: %s", result.Name)
		}
	} else {
		if !isValidLinter(result.Name) {
			return "", false, nil, fmt.Errorf("invalid linter name: %s", result.Name)
		}
	}

	return result.Name, result.IsFormatter, result.Settings, nil
}

// validateConfig validates the golangci-lint configuration
func validateConfig(config *golangciConfig) error {
	if config.Version == "" {
		return fmt.Errorf("version is required")
	}

	// At least one linter or formatter must be enabled
	if len(config.Linters.Enable) == 0 && len(config.Formatters.Enable) == 0 {
		return fmt.Errorf("no linters or formatters enabled")
	}

	return nil
}

// isValidLinter checks if the name is a known golangci-lint linter (NOT formatter)
func isValidLinter(name string) bool {
	validLinters := map[string]bool{
		// Error Handling
		"errcheck": true, "wrapcheck": true, "nilerr": true, "nilnil": true,
		// Code Quality
		"govet": true, "staticcheck": true, "ineffassign": true, "unused": true,
		"revive": true, "typecheck": true,
		// Complexity
		"gocyclo": true, "gocognit": true, "nestif": true, "funlen": true, "cyclop": true,
		// Performance
		"prealloc": true, "bodyclose": true,
		// Security
		"gosec": true,
		// Style (NOT formatters)
		"stylecheck": true, "goconst": true, "misspell": true, "godot": true, "nlreturn": true,
		// Other
		"dupl": true, "gocritic": true, "gosimple": true, "noctx": true,
		"unconvert": true, "exportloopref": true, "goprintffuncname": true,
		"rowserrcheck": true, "sqlclosecheck": true,
	}

	return validLinters[strings.ToLower(name)]
}

// isValidFormatter checks if the name is a known golangci-lint formatter
func isValidFormatter(name string) bool {
	validFormatters := map[string]bool{
		"gci":      true,
		"gofmt":    true,
		"gofumpt":  true,
		"goimports": true,
		"golines":  true,
		"swaggo":   true,
	}

	return validFormatters[strings.ToLower(name)]
}
