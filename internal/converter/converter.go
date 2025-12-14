package converter

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"

	"github.com/DevSymphony/sym-cli/internal/linter"
	"github.com/DevSymphony/sym-cli/internal/llm"
	"github.com/DevSymphony/sym-cli/pkg/schema"
)

// llmValidatorEngine is the special linter for rules that don't fit any specific external linter
const llmValidatorEngine = "llm-validator"

// Converter is the main converter with language-based routing
type Converter struct {
	llmProvider llm.Provider
	outputDir   string
}

// NewConverter creates a new converter instance
func NewConverter(provider llm.Provider, outputDir string) *Converter {
	if outputDir == "" {
		outputDir = ".sym"
	}
	return &Converter{
		llmProvider: provider,
		outputDir:   outputDir,
	}
}

// ConvertResult represents the result of conversion
type ConvertResult struct {
	GeneratedFiles []string           // List of generated file paths (including code-policy.json)
	CodePolicy     *schema.CodePolicy // Generated code policy
	Errors         map[string]error   // Errors per linter
	Warnings       []string           // Conversion warnings
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

	// Step 4: Convert all (linter, rule) pairs in parallel with single semaphore
	// This is a flat parallelization - no nested goroutines
	result := &ConvertResult{
		GeneratedFiles: []string{},
		CodePolicy:     codePolicy,
		Errors:         make(map[string]error),
		Warnings:       []string{},
	}

	// Track failed rules per linter for fallback to llm-validator
	failedRulesPerLinter := make(map[string][]string)

	// Collect all (linter, rule) pairs as tasks
	var tasks []conversionTask
	for linterName, rules := range linterRules {
		if linterName == llmValidatorEngine {
			continue // Skip llm-validator - handled in CodePolicy only
		}
		for _, rule := range rules {
			tasks = append(tasks, conversionTask{linterName: linterName, rule: rule})
		}
	}

	// Convert all tasks in parallel with single semaphore
	successResults, failedResults := c.convertAllTasks(ctx, tasks)

	// Update failedRulesPerLinter from conversion results
	for linterName, ruleIDs := range failedResults {
		failedRulesPerLinter[linterName] = append(failedRulesPerLinter[linterName], ruleIDs...)
	}

	// Build configs and write files for each linter (sequential - no LLM calls)
	for linterName, linterResults := range successResults {
		converter := c.getLinterConverter(linterName)
		if converter == nil {
			result.Errors[linterName] = fmt.Errorf("unsupported linter: %s", linterName)
			continue
		}

		config, err := converter.BuildConfig(linterResults)
		if err != nil {
			result.Errors[linterName] = fmt.Errorf("failed to build config: %w", err)
			continue
		}

		if config == nil {
			continue
		}

		outputPath := filepath.Join(c.outputDir, config.Filename)
		if err := os.WriteFile(outputPath, config.Content, 0644); err != nil {
			result.Errors[linterName] = fmt.Errorf("failed to write file: %w", err)
			continue
		}

		result.GeneratedFiles = append(result.GeneratedFiles, outputPath)
		fmt.Fprintf(os.Stderr, "✓ Generated %s configuration: %s\n", linterName, outputPath)
	}

	// Step 4.1: Update ruleToLinters mapping - remove failed linters and add llm-validator fallback
	for linter, failedRuleIDs := range failedRulesPerLinter {
		for _, ruleID := range failedRuleIDs {
			// Remove failed linter from this rule's linters
			currentLinters := ruleToLinters[ruleID]
			updatedLinters := []string{}
			for _, l := range currentLinters {
				if l != linter {
					updatedLinters = append(updatedLinters, l)
				}
			}

			// Add llm-validator as fallback if not already present
			hasLLMValidator := false
			for _, l := range updatedLinters {
				if l == llmValidatorEngine {
					hasLLMValidator = true
					break
				}
			}
			if !hasLLMValidator {
				updatedLinters = append(updatedLinters, llmValidatorEngine)
			}

			ruleToLinters[ruleID] = updatedLinters
		}
	}

	// Log fallback info
	totalFallbacks := 0
	for _, failedRuleIDs := range failedRulesPerLinter {
		totalFallbacks += len(failedRuleIDs)
	}
	if totalFallbacks > 0 {
		fmt.Fprintf(os.Stderr, "ℹ️  %d rule(s) fell back to llm-validator due to conversion failures\n", totalFallbacks)
	}

	// Check if we have any successful conversions (excluding llm-validator rules)
	// Note: We don't fail if all rules went to llm-validator
	if len(result.GeneratedFiles) == 0 && len(result.Errors) > 0 && totalFallbacks == 0 {
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
				// Filter languages to only those supported by this linter
				filteredLanguages := userRule.Languages
				if conv, ok := linter.Global().GetConverter(linterName); ok {
					supportedLangs := conv.SupportedLanguages()
					if len(supportedLangs) > 0 && len(userRule.Languages) > 0 {
						filteredLanguages = intersectLanguages(userRule.Languages, supportedLangs)
					}
				}
				policyRule.When = &schema.Selector{
					Languages: filteredLanguages,
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
	fmt.Fprintf(os.Stderr, "✓ Generated code policy: %s\n", codePolicyPath)

	return result, nil
}

// routeRulesWithLLM uses LLM to determine which linters are appropriate for each rule
// Rules are processed in parallel with concurrency limited to CPU count
func (c *Converter) routeRulesWithLLM(ctx context.Context, userPolicy *schema.UserPolicy) map[string][]schema.UserRule {
	type routeResult struct {
		rule    schema.UserRule
		linters []string
	}

	results := make(chan routeResult, len(userPolicy.Rules))
	var wg sync.WaitGroup

	// Limit concurrent LLM calls: CPU/2 (minimum 1)
	maxConcurrent := runtime.NumCPU() / 2
	if maxConcurrent < 1 {
		maxConcurrent = 1
	}
	sem := make(chan struct{}, maxConcurrent)

	// Build category name -> description map
	categoryMap := make(map[string]string)
	for _, cat := range userPolicy.Category {
		categoryMap[cat.Name] = cat.Description
	}

	// Process rules in parallel with concurrency limit
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
			select {
			case results <- routeResult{rule: rule, linters: []string{llmValidatorEngine}}:
			case <-ctx.Done():
				continue
			}
			continue
		}

		wg.Add(1)
		go func(r schema.UserRule, linters []string, catMap map[string]string) {
			defer wg.Done()

			// Acquire semaphore with context check
			select {
			case sem <- struct{}{}:
			case <-ctx.Done():
				return
			}
			defer func() { <-sem }()

			// Ask LLM which linters are appropriate for this rule
			selectedLinters := c.selectLintersForRule(ctx, r, linters, catMap)

			// Send result with context check to prevent deadlock
			if len(selectedLinters) == 0 {
				// LLM couldn't map to any linter, use llm-validator
				select {
				case results <- routeResult{rule: r, linters: []string{llmValidatorEngine}}:
				case <-ctx.Done():
					return
				}
			} else {
				select {
				case results <- routeResult{rule: r, linters: selectedLinters}:
				case <-ctx.Done():
					return
				}
			}
		}(rule, availableLinters, categoryMap)
	}

	// Close results channel after all goroutines complete
	go func() {
		wg.Wait()
		close(results)
	}()

	// Collect results
	linterRules := make(map[string][]schema.UserRule)
	for result := range results {
		for _, linter := range result.linters {
			linterRules[linter] = append(linterRules[linter], result.rule)
		}
	}

	return linterRules
}

// getAvailableLinters returns available linters for given languages
func (c *Converter) getAvailableLinters(languages []string) []string {
	// Build language mapping dynamically from registry
	languageLinterMapping := linter.Global().BuildLanguageMapping()

	if len(languages) == 0 {
		// If no languages specified, return all registered tools
		return linter.Global().GetAllToolNames()
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
func (c *Converter) selectLintersForRule(ctx context.Context, rule schema.UserRule, availableLinters []string, categoryMap map[string]string) []string {
	// Build linter descriptions dynamically from registry
	linterDescriptions := c.buildLinterDescriptions(availableLinters)

	// Build routing hints dynamically from converters
	routingHints := c.buildRoutingHints(availableLinters)

	systemPrompt := fmt.Sprintf(`You are a code quality expert. Analyze the given coding rule and determine which linters can ACTUALLY enforce it using their NATIVE rules (without plugins).

Available linters and NATIVE capabilities:
%s

STRICT Rules for selection:
1. ONLY select if the linter has a NATIVE rule that can enforce this
2. If the rule requires understanding business logic or context → return []
3. If the rule requires custom plugins → return []
4. If the rule is about file naming → return []
5. If the rule requires deep semantic analysis → return []
6. When in doubt, return [] (better to use llm-validator than fail)
%s

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
Reason: Requires knowing which packages are "large"`, linterDescriptions, routingHints, availableLinters)

	categoryInfo := rule.Category
	if desc, ok := categoryMap[rule.Category]; ok && desc != "" {
		categoryInfo = fmt.Sprintf("%s (%s)", rule.Category, desc)
	}
	userPrompt := fmt.Sprintf("Rule: %s\nCategory: %s", rule.Say, categoryInfo)

	// Call LLM
	prompt := systemPrompt + "\n\n" + userPrompt
	response, err := c.llmProvider.Execute(ctx, prompt, llm.JSON)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: LLM routing failed for rule %s: %v\n", rule.ID, err)
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
		fmt.Fprintf(os.Stderr, "Warning: Failed to parse LLM response for rule %s: %v\n", rule.ID, err)
		return []string{} // Will fall back to llm-validator
	}

	return selectedLinters
}

// getLinterConverter returns the appropriate converter for a linter
func (c *Converter) getLinterConverter(linterName string) linter.Converter {
	// Use registry to get converter (no hardcoding)
	converter, ok := linter.Global().GetConverter(linterName)
	if !ok {
		return nil
	}
	return converter
}

// buildLinterDescriptions builds linter capability descriptions from registry
func (c *Converter) buildLinterDescriptions(availableLinters []string) string {
	var descriptions []string

	for _, linterName := range availableLinters {
		converter, ok := linter.Global().GetConverter(linterName)
		if !ok || converter == nil {
			continue
		}

		desc := converter.GetLLMDescription()
		if desc != "" {
			descriptions = append(descriptions, fmt.Sprintf("- %s: %s", linterName, desc))
		}
	}

	if len(descriptions) == 0 {
		return "No linter descriptions available"
	}

	return strings.Join(descriptions, "\n")
}

// buildRoutingHints builds routing hints from all available converters
func (c *Converter) buildRoutingHints(availableLinters []string) string {
	var hints []string
	hintNumber := 7 // Start after the base rules (1-6)

	for _, linterName := range availableLinters {
		converter, ok := linter.Global().GetConverter(linterName)
		if !ok || converter == nil {
			continue
		}

		routingHints := converter.GetRoutingHints()
		for _, hint := range routingHints {
			hints = append(hints, fmt.Sprintf("%d. %s", hintNumber, hint))
			hintNumber++
		}
	}

	if len(hints) == 0 {
		return ""
	}

	return strings.Join(hints, "\n")
}

// conversionTask represents a single (linter, rule) pair to be converted
type conversionTask struct {
	linterName string
	rule       schema.UserRule
}

// convertAllTasks converts all (linter, rule) pairs in parallel with a single semaphore.
// Returns results grouped by linter name, and failed rule IDs grouped by linter name.
func (c *Converter) convertAllTasks(ctx context.Context, tasks []conversionTask) (map[string][]*linter.SingleRuleResult, map[string][]string) {
	if len(tasks) == 0 {
		return make(map[string][]*linter.SingleRuleResult), make(map[string][]string)
	}

	type taskResult struct {
		linterName string
		result     *linter.SingleRuleResult
		ruleID     string
		err        error
	}

	results := make(chan taskResult, len(tasks))
	var wg sync.WaitGroup

	// Single semaphore for all conversions: CPU/2 (minimum 1)
	maxConcurrent := runtime.NumCPU() / 2
	if maxConcurrent < 1 {
		maxConcurrent = 1
	}
	sem := make(chan struct{}, maxConcurrent)

	// Process all tasks in parallel
	for _, task := range tasks {
		wg.Add(1)
		go func(t conversionTask) {
			defer wg.Done()

			// Acquire semaphore with context check
			select {
			case sem <- struct{}{}:
			case <-ctx.Done():
				results <- taskResult{linterName: t.linterName, ruleID: t.rule.ID, err: ctx.Err()}
				return
			}
			defer func() { <-sem }()

			// Get converter and convert single rule
			converter := c.getLinterConverter(t.linterName)
			if converter == nil {
				results <- taskResult{linterName: t.linterName, ruleID: t.rule.ID, err: fmt.Errorf("unsupported linter: %s", t.linterName)}
				return
			}

			res, err := converter.ConvertSingleRule(ctx, t.rule, c.llmProvider)
			results <- taskResult{linterName: t.linterName, result: res, ruleID: t.rule.ID, err: err}
		}(task)
	}

	// Close results channel after all goroutines complete
	go func() {
		wg.Wait()
		close(results)
	}()

	// Collect and group results by linter
	successByLinter := make(map[string][]*linter.SingleRuleResult)
	failedByLinter := make(map[string][]string)

	for res := range results {
		if res.err != nil {
			failedByLinter[res.linterName] = append(failedByLinter[res.linterName], res.ruleID)
			continue
		}
		if res.result == nil {
			// Skip = cannot be enforced by this linter, fallback to llm-validator
			failedByLinter[res.linterName] = append(failedByLinter[res.linterName], res.ruleID)
			continue
		}
		successByLinter[res.linterName] = append(successByLinter[res.linterName], res.result)
	}

	return successByLinter, failedByLinter
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

// intersectLanguages returns the intersection of two language slices.
// It normalizes language names (e.g., "ts" -> "typescript") for comparison.
func intersectLanguages(langs1, langs2 []string) []string {
	// Normalization map for common language aliases
	normalize := func(lang string) string {
		switch strings.ToLower(lang) {
		case "ts", "tsx":
			return "typescript"
		case "js", "jsx":
			return "javascript"
		case "py":
			return "python"
		default:
			return strings.ToLower(lang)
		}
	}

	// Build a set of normalized languages from langs2
	supported := make(map[string]bool)
	for _, lang := range langs2 {
		supported[normalize(lang)] = true
	}

	// Find intersection - keep original language names from langs1
	var result []string
	seen := make(map[string]bool)
	for _, lang := range langs1 {
		normalized := normalize(lang)
		if supported[normalized] && !seen[normalized] {
			result = append(result, lang)
			seen[normalized] = true
		}
	}

	return result
}
