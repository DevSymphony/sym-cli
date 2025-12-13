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
- FORMATTERS: Tools that format code (gofmt, goimports, gofumpt, gci, golines)
- LINTERS: Tools that analyze code for issues (errcheck, govet, gosec, etc.)

Return ONLY a JSON object (no markdown fences) with this structure:
{
  "name": "tool_name",
  "is_formatter": true/false,
  "settings": {}
}

=== FORMATTERS (is_formatter: true) ===
- gofmt: Standard Go formatting
- goimports: Formats imports and adds missing ones
- gofumpt: Stricter formatting than gofmt
- gci: Import ordering (settings: {"sections": ["standard", "default", "prefix(github.com/myorg)"]})
- golines: Line length limiting (settings: {"max-len": 120})

=== LINTERS (is_formatter: false) ===

ERROR HANDLING:
- errcheck: Unchecked errors (settings: {"check-type-assertions": true, "check-blank": true})
- wrapcheck: Errors from external packages must be wrapped
- nilerr: Returns nil even when error is not nil
- err113: Error handling best practices
- errorlint: Error wrapping issues (settings: {"errorf": true})

CODE QUALITY:
- govet: Suspicious constructs
- staticcheck: Advanced static analysis (includes style checks formerly in stylecheck)
- ineffassign: Ineffectual assignments
- unused: Unused code detection
- revive: Configurable linter (replacement for golint, can check style/naming)

COMPLEXITY:
- gocyclo: Cyclomatic complexity (settings: {"min-complexity": 10})
- gocognit: Cognitive complexity (settings: {"min-complexity": 15})
- funlen: Function length (settings: {"lines": 60, "statements": 40})
- nestif: Nested if depth (settings: {"min-complexity": 4})
- cyclop: Package complexity

PERFORMANCE:
- prealloc: Slice preallocation opportunities
- bodyclose: HTTP response body closure
- perfsprint: fmt.Sprintf optimization

SECURITY:
- gosec: Security vulnerabilities (settings: {"severity": "medium", "confidence": "medium"})

STYLE & NAMING:
- goconst: Repeated strings for constants (settings: {"min-len": 3, "min-occurrences": 3})
- misspell: Spelling errors (settings: {"locale": "US"})
- godot: Comments end with period
- nlreturn: Blank line before return
- varnamelen: Variable name length (settings: {"min-name-length": 3})
- lll: Line length limit (settings: {"line-length": 120})

DOCUMENTATION:
- godoclint: Godoc comment validation

OTHER:
- dupl: Code duplication (settings: {"threshold": 100})
- gocritic: Bugs, performance, style issues
- noctx: HTTP requests without context
- unconvert: Unnecessary type conversions
- mnd: Magic number detection (settings: {"checks": ["argument", "case", "condition", "operation", "return", "assign"]})

IMPORTANT - REMOVED/CHANGED IN v2:
1. stylecheck is REMOVED - use staticcheck or revive instead
2. gosimple is MERGED into staticcheck
3. typecheck is now built-in (don't use)
4. exportloopref is REMOVED (Go 1.22+ handles this)
5. For style rules, prefer revive (most configurable) or staticcheck

DECISION GUIDE:
- Error must be checked → errcheck
- Error wrapping → wrapcheck or errorlint
- Style/naming rules → revive (most flexible) or staticcheck
- Line length → lll (linter) or golines (formatter)
- Variable naming → varnamelen or revive
- Security issues → gosec
- Cannot be enforced → return empty name ""

Examples:

Input: "Code should follow Go formatting standards"
Output:
{
  "name": "gofmt",
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

Input: "Follow Go style guide"
Output:
{
  "name": "staticcheck",
  "is_formatter": false,
  "settings": {}
}

Input: "Variable names should be descriptive"
Output:
{
  "name": "varnamelen",
  "is_formatter": false,
  "settings": {"min-name-length": 3}
}

Input: "Line length should not exceed 120 characters"
Output:
{
  "name": "lll",
  "is_formatter": false,
  "settings": {"line-length": 120}
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

// isValidLinter checks if the name is a known golangci-lint v2 linter (NOT formatter)
func isValidLinter(name string) bool {
	validLinters := map[string]bool{
		// Error Handling
		"errcheck": true, "wrapcheck": true, "nilerr": true, "nilnil": true,
		"err113": true, "errorlint": true,
		// Code Quality
		"govet": true, "staticcheck": true, "ineffassign": true, "unused": true,
		"revive": true,
		// Complexity
		"gocyclo": true, "gocognit": true, "nestif": true, "funlen": true, "cyclop": true,
		// Performance
		"prealloc": true, "bodyclose": true, "perfsprint": true,
		// Security
		"gosec": true,
		// Style & Naming
		"goconst": true, "misspell": true, "godot": true, "nlreturn": true,
		"varnamelen": true, "lll": true,
		// Documentation
		"godoclint": true,
		// Other
		"dupl": true, "gocritic": true, "noctx": true,
		"unconvert": true, "goprintffuncname": true,
		"rowserrcheck": true, "sqlclosecheck": true, "mnd": true,
	}

	return validLinters[strings.ToLower(name)]
}

// isValidFormatter checks if the name is a known golangci-lint formatter
func isValidFormatter(name string) bool {
	validFormatters := map[string]bool{
		"gci":       true,
		"gofmt":     true,
		"gofumpt":   true,
		"goimports": true,
		"golines":   true,
		"swaggo":    true,
	}

	return validFormatters[strings.ToLower(name)]
}
