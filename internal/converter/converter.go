package converter

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/DevSymphony/sym-cli/internal/converter/linters"
	"github.com/DevSymphony/sym-cli/internal/llm"
	"github.com/DevSymphony/sym-cli/pkg/schema"
)

// languageLinterMapping defines which linters are available for each language
var languageLinterMapping = map[string][]string{
	"javascript": {"eslint", "prettier"},
	"js":         {"eslint", "prettier"},
	"typescript": {"tsc", "eslint", "prettier"},
	"ts":         {"tsc", "eslint", "prettier"},
	"tsx":        {"tsc", "eslint", "prettier"},
	"jsx":        {"eslint", "prettier"},
	"java":       {"checkstyle", "pmd"},
}

// llmValidatorEngine is the special linter for rules that don't fit any specific external linter
const llmValidatorEngine = "llm-validator"

// Converter is the main converter with language-based routing
type Converter struct {
	llmClient *llm.Client
	outputDir string
}

// NewConverter creates a new converter instance
func NewConverter(llmClient *llm.Client, outputDir string) *Converter {
	if outputDir == "" {
		outputDir = ".sym"
	}
	return &Converter{
		llmClient: llmClient,
		outputDir: outputDir,
	}
}

// ConvertResult represents the result of conversion
type ConvertResult struct {
	GeneratedFiles []string          // List of generated file paths (including code-policy.json)
	CodePolicy     *schema.CodePolicy // Generated code policy
	Errors         map[string]error  // Errors per linter
	Warnings       []string          // Conversion warnings
}

// Convert is the main entry point for converting user policy to linter configs
func (c *Converter) Convert(ctx context.Context, userPolicy *schema.UserPolicy) (*ConvertResult, error) {
	if userPolicy == nil {
		return nil, fmt.Errorf("user policy is nil")
	}

	// Step 1: Route rules by asking LLM which linters are appropriate
	linterRules := c.routeRulesWithLLM(ctx, userPolicy)

	// Step 2: Create output directory
	if err := os.MkdirAll(c.outputDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create output directory: %w", err)
	}

	// Step 3: Build CodePolicy with linter mappings
	codePolicy := &schema.CodePolicy{
		Version: "1.0",
		Rules:   []schema.PolicyRule{},
		Enforce: schema.EnforceSettings{
			Stages: []string{"pre-commit", "pre-push"},
			FailOn: []string{"error"},
		},
	}

	// Step 3.1: Convert RBAC if present
	if userPolicy.RBAC != nil {
		codePolicy.RBAC = c.convertRBAC(userPolicy.RBAC)

		// Enable RBAC enforcement
		codePolicy.Enforce.RBACConfig = &schema.RBACEnforce{
			Enabled:     true,
			Stages:      []string{"pre-commit", "pre-push"},
			OnViolation: "block",
		}
	}

	// Track which linters each rule maps to
	ruleToLinters := make(map[string][]string) // rule ID -> linter names

	for linterName, rules := range linterRules {
		for _, rule := range rules {
			ruleToLinters[rule.ID] = append(ruleToLinters[rule.ID], linterName)
		}
	}

	// Step 4: Convert rules for each linter in parallel using goroutines
	result := &ConvertResult{
		GeneratedFiles: []string{},
		CodePolicy:     codePolicy,
		Errors:         make(map[string]error),
		Warnings:       []string{},
	}

	var wg sync.WaitGroup
	var mu sync.Mutex

	for linterName, rules := range linterRules {
		if len(rules) == 0 {
			continue
		}

		// Skip llm-validator - it will be handled in CodePolicy only
		if linterName == llmValidatorEngine {
			continue
		}

		wg.Add(1)
		go func(linter string, ruleSet []schema.UserRule) {
			defer wg.Done()

			// Get linter converter
			converter := c.getLinterConverter(linter)
			if converter == nil {
				mu.Lock()
				result.Errors[linter] = fmt.Errorf("unsupported linter: %s", linter)
				mu.Unlock()
				return
			}

			// Convert rules using LLM
			configFile, err := converter.ConvertRules(ctx, ruleSet, c.llmClient)
			if err != nil {
				mu.Lock()
				result.Errors[linter] = fmt.Errorf("conversion failed: %w", err)
				mu.Unlock()
				return
			}

			// Write config file to .sym directory
			outputPath := filepath.Join(c.outputDir, configFile.Filename)
			if err := os.WriteFile(outputPath, configFile.Content, 0644); err != nil {
				mu.Lock()
				result.Errors[linter] = fmt.Errorf("failed to write file: %w", err)
				mu.Unlock()
				return
			}

			mu.Lock()
			result.GeneratedFiles = append(result.GeneratedFiles, outputPath)
			mu.Unlock()

			fmt.Printf("✓ Generated %s configuration: %s\n", linter, outputPath)
		}(linterName, rules)
	}

	// Wait for all goroutines to complete
	wg.Wait()

	// Check if we have any successful conversions
	if len(result.GeneratedFiles) == 0 && len(result.Errors) > 0 {
		return result, fmt.Errorf("all conversions failed")
	}

	// Step 5: Generate CodePolicy rules from UserPolicy
	for _, userRule := range userPolicy.Rules {
		linters := ruleToLinters[userRule.ID]
		if len(linters) == 0 {
			continue // Skip rules that didn't map to any linter
		}

		// Create a PolicyRule for each linter this rule applies to
		for _, linterName := range linters {
			policyRule := schema.PolicyRule{
				ID:       fmt.Sprintf("%s-%s", userRule.ID, linterName),
				Enabled:  true,
				Category: userRule.Category,
				Severity: userRule.Severity,
				Desc:     userRule.Say,
				Message:  userRule.Message,
				Check: map[string]any{
					"engine": linterName, // External linter name
					"desc":   userRule.Say,
				},
			}

			// Special handling for LLM validator - ensure required fields
			if linterName == llmValidatorEngine {
				// LLM validator MUST have 'when' selector for file filtering
				if policyRule.When == nil {
					// Use languages from rule or defaults
					languages := userRule.Languages
					if len(languages) == 0 && userPolicy.Defaults != nil {
						languages = userPolicy.Defaults.Languages
					}
					if len(languages) == 0 {
						// Default to common languages if none specified
						languages = []string{"javascript", "typescript"}
					}

					policyRule.When = &schema.Selector{
						Languages: languages,
					}
				}

				// Ensure desc is not empty (required for LLM prompt)
				if policyRule.Desc == "" {
					policyRule.Desc = "Code quality check"
				}
			}

			// Add selector if languages are specified (for non-LLM linters)
			if linterName != llmValidatorEngine && (len(userRule.Languages) > 0 || len(userRule.Include) > 0 || len(userRule.Exclude) > 0) {
				policyRule.When = &schema.Selector{
					Languages: userRule.Languages,
					Include:   userRule.Include,
					Exclude:   userRule.Exclude,
				}
			}

			// Add remedy if autofix is enabled
			if userRule.Autofix {
				policyRule.Remedy = &schema.Remedy{
					Autofix: true,
					Tool:    linterName,
				}
			}

			// Use defaults if not specified
			if policyRule.Severity == "" && userPolicy.Defaults != nil {
				policyRule.Severity = userPolicy.Defaults.Severity
			}
			if policyRule.Severity == "" {
				policyRule.Severity = "error"
			}

			codePolicy.Rules = append(codePolicy.Rules, policyRule)
		}
	}

	// Step 6: Write code-policy.json
	codePolicyPath := filepath.Join(c.outputDir, "code-policy.json")
	codePolicyJSON, err := json.MarshalIndent(codePolicy, "", "  ")
	if err != nil {
		return result, fmt.Errorf("failed to marshal code policy: %w", err)
	}

	if err := os.WriteFile(codePolicyPath, codePolicyJSON, 0644); err != nil {
		return result, fmt.Errorf("failed to write code policy: %w", err)
	}

	result.GeneratedFiles = append(result.GeneratedFiles, codePolicyPath)
	fmt.Printf("✓ Generated code policy: %s\n", codePolicyPath)

	return result, nil
}

// routeRulesWithLLM uses LLM to determine which linters are appropriate for each rule
func (c *Converter) routeRulesWithLLM(ctx context.Context, userPolicy *schema.UserPolicy) map[string][]schema.UserRule {
	linterRules := make(map[string][]schema.UserRule)

	for _, rule := range userPolicy.Rules {
		// Get languages for this rule
		languages := rule.Languages
		if len(languages) == 0 && userPolicy.Defaults != nil {
			languages = userPolicy.Defaults.Languages
		}

		// Get available linters for these languages
		availableLinters := c.getAvailableLinters(languages)
		if len(availableLinters) == 0 {
			// No language-specific linters, use llm-validator
			linterRules[llmValidatorEngine] = append(linterRules[llmValidatorEngine], rule)
			continue
		}

		// Ask LLM which linters are appropriate for this rule
		selectedLinters := c.selectLintersForRule(ctx, rule, availableLinters)

		if len(selectedLinters) == 0 {
			// LLM couldn't map to any linter, use llm-validator
			linterRules[llmValidatorEngine] = append(linterRules[llmValidatorEngine], rule)
		} else {
			// Add rule to selected linters
			for _, linter := range selectedLinters {
				linterRules[linter] = append(linterRules[linter], rule)
			}
		}
	}

	return linterRules
}

// getAvailableLinters returns available linters for given languages
func (c *Converter) getAvailableLinters(languages []string) []string {
	if len(languages) == 0 {
		// If no languages specified, include all linters
		languages = []string{"javascript", "typescript", "java"}
	}

	linterSet := make(map[string]bool)
	for _, lang := range languages {
		if linters, ok := languageLinterMapping[lang]; ok {
			for _, linter := range linters {
				linterSet[linter] = true
			}
		}
	}

	result := []string{}
	for linter := range linterSet {
		result = append(result, linter)
	}
	return result
}

// selectLintersForRule uses LLM to determine which linters are appropriate for a rule
func (c *Converter) selectLintersForRule(ctx context.Context, rule schema.UserRule, availableLinters []string) []string {
	systemPrompt := fmt.Sprintf(`You are a code quality expert. Analyze the given coding rule and determine which linters can ACTUALLY enforce it using their NATIVE rules (without plugins).

Available linters and NATIVE capabilities:
- eslint: ONLY native ESLint rules (no-console, no-unused-vars, eqeqeq, no-var, camelcase, new-cap, max-len, max-lines, no-eval, etc.)
  - CAN: Simple syntax checks, variable naming, console usage, basic patterns
  - CANNOT: Complex business logic, context-aware rules, file naming, advanced async patterns
- prettier: Code formatting ONLY (quotes, semicolons, indentation, line length, trailing commas)
- tsc: TypeScript type checking ONLY (strict modes, noImplicitAny, strictNullChecks, type inference)
- checkstyle: Java style checks (naming, whitespace, imports, line length, complexity)
- pmd: Java code quality (unused code, empty blocks, naming conventions, design issues)

STRICT Rules for selection:
1. ONLY select if the linter has a NATIVE rule that can enforce this
2. If the rule requires understanding business logic or context → return []
3. If the rule requires custom plugins → return []
4. If the rule is about file naming → return []
5. If the rule requires deep semantic analysis → return []
6. When in doubt, return [] (better to use llm-validator than fail)

Available linters for this rule: %s

Return ONLY a JSON array of linter names (no markdown):
["linter1", "linter2"] or []

Examples:

Input: "Use single quotes for strings"
Output: ["prettier"]

Input: "No console.log allowed"
Output: ["eslint"]

Input: "Classes start with capital letter"
Output: ["eslint"]

Input: "Maximum line length is 120"
Output: ["prettier"]

Input: "No implicit any types"
Output: ["tsc"]

Input: "All async functions must have try-catch"
Output: []
Reason: Requires semantic understanding of error handling

Input: "File names must be kebab-case"
Output: []
Reason: File naming requires plugin

Input: "API handlers must return proper status codes"
Output: []
Reason: Requires business logic understanding

Input: "Database queries must use parameterized queries"
Output: []
Reason: Requires understanding SQL injection context

Input: "No hardcoded API keys or passwords"
Output: []
Reason: Requires semantic analysis of what constitutes secrets

Input: "Imports from large packages must be specific"
Output: []
Reason: Requires knowing which packages are "large"`, availableLinters)

	userPrompt := fmt.Sprintf("Rule: %s\nCategory: %s", rule.Say, rule.Category)

	// Call LLM
	response, err := c.llmClient.Complete(ctx, systemPrompt, userPrompt)
	if err != nil {
		fmt.Printf("Warning: LLM routing failed for rule %s: %v\n", rule.ID, err)
		return []string{} // Will fall back to llm-validator
	}

	// Parse response
	response = strings.TrimSpace(response)
	response = strings.TrimPrefix(response, "```json")
	response = strings.TrimPrefix(response, "```")
	response = strings.TrimSuffix(response, "```")
	response = strings.TrimSpace(response)

	var selectedLinters []string
	if err := json.Unmarshal([]byte(response), &selectedLinters); err != nil {
		fmt.Printf("Warning: Failed to parse LLM response for rule %s: %v\n", rule.ID, err)
		return []string{} // Will fall back to llm-validator
	}

	return selectedLinters
}

// getLinterConverter returns the appropriate converter for a linter
func (c *Converter) getLinterConverter(linterName string) linters.LinterConverter {
	switch linterName {
	case "eslint":
		return linters.NewESLintLinterConverter()
	case "prettier":
		return linters.NewPrettierLinterConverter()
	case "tsc":
		return linters.NewTSCLinterConverter()
	case "checkstyle":
		return linters.NewCheckstyleLinterConverter()
	case "pmd":
		return linters.NewPMDLinterConverter()
	default:
		return nil
	}
}

// convertRBAC converts UserRBAC to PolicyRBAC
func (c *Converter) convertRBAC(userRBAC *schema.UserRBAC) *schema.PolicyRBAC {
	if userRBAC == nil || len(userRBAC.Roles) == 0 {
		return nil
	}

	policyRBAC := &schema.PolicyRBAC{
		Roles: make(map[string]schema.PolicyRole),
	}

	for roleName, userRole := range userRBAC.Roles {
		policyRole := schema.PolicyRole{
			Permissions: []schema.Permission{},
		}

		// Convert allowWrite to write permissions
		for _, path := range userRole.AllowWrite {
			policyRole.Permissions = append(policyRole.Permissions, schema.Permission{
				Path:    path,
				Read:    true,
				Write:   true,
				Execute: false,
			})
		}

		// Convert denyWrite to read-only permissions
		for _, path := range userRole.DenyWrite {
			policyRole.Permissions = append(policyRole.Permissions, schema.Permission{
				Path:    path,
				Read:    true,
				Write:   false,
				Execute: false,
			})
		}

		// Convert allowExec to execute permissions
		for _, path := range userRole.AllowExec {
			policyRole.Permissions = append(policyRole.Permissions, schema.Permission{
				Path:    path,
				Read:    true,
				Write:   false,
				Execute: true,
			})
		}

		// Add special permissions for policy/role editing
		if userRole.CanEditPolicy {
			policyRole.Permissions = append(policyRole.Permissions, schema.Permission{
				Path:    ".sym/**",
				Read:    true,
				Write:   true,
				Execute: false,
			})
		}

		if userRole.CanEditRoles {
			policyRole.Permissions = append(policyRole.Permissions, schema.Permission{
				Path:    ".sym/user-policy.json",
				Read:    true,
				Write:   true,
				Execute: false,
			})
		}

		policyRBAC.Roles[roleName] = policyRole
	}

	return policyRBAC
}
