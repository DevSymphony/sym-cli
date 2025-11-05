package converter

import (
	"context"
	"fmt"
	"reflect"
	"strings"

	"github.com/DevSymphony/sym-cli/internal/converter/linters"
	"github.com/DevSymphony/sym-cli/internal/llm"
	"github.com/DevSymphony/sym-cli/pkg/schema"
)

// Converter converts user policy (A schema) to code policy (B schema)
type Converter struct {
	llmClient  *llm.Client
	inferencer *llm.Inferencer
}

// ConverterOption is a functional option for configuring the converter
type ConverterOption func(*Converter)

// WithLLMClient sets the LLM client for inference
func WithLLMClient(client *llm.Client) ConverterOption {
	return func(c *Converter) {
		c.llmClient = client
		c.inferencer = llm.NewInferencer(client)
	}
}

// NewConverter creates a new converter
func NewConverter(opts ...ConverterOption) *Converter {
	c := &Converter{}

	for _, opt := range opts {
		opt(c)
	}

	return c
}

// Convert converts user policy to code policy
func (c *Converter) Convert(userPolicy *schema.UserPolicy) (*schema.CodePolicy, error) {
	if userPolicy == nil {
		return nil, fmt.Errorf("user policy is nil")
	}

	codePolicy := &schema.CodePolicy{
		Version: userPolicy.Version,
		Rules:   make([]schema.PolicyRule, 0, len(userPolicy.Rules)),
		Enforce: schema.EnforceSettings{
			Stages: []string{"pre-commit"},
			FailOn: []string{"error"},
		},
	}

	if codePolicy.Version == "" {
		codePolicy.Version = "1.0.0"
	}

	// Convert RBAC
	if userPolicy.RBAC != nil {
		codePolicy.RBAC = c.convertRBAC(userPolicy.RBAC)
	}

	// Convert rules
	for i, userRule := range userPolicy.Rules {
		policyRule, err := c.convertRule(&userRule, userPolicy.Defaults, i)
		if err != nil {
			return nil, fmt.Errorf("failed to convert rule %d: %w", i, err)
		}
		codePolicy.Rules = append(codePolicy.Rules, *policyRule)
	}

	return codePolicy, nil
}

// convertWithEngines converts user policy to code policy with engine mappings
func (c *Converter) convertWithEngines(userPolicy *schema.UserPolicy, engineMap map[string]string) (*schema.CodePolicy, error) {
	if userPolicy == nil {
		return nil, fmt.Errorf("user policy is nil")
	}

	codePolicy := &schema.CodePolicy{
		Version: userPolicy.Version,
		Rules:   make([]schema.PolicyRule, 0, len(userPolicy.Rules)),
		Enforce: schema.EnforceSettings{
			Stages: []string{"pre-commit"},
			FailOn: []string{"error"},
		},
	}

	if codePolicy.Version == "" {
		codePolicy.Version = "1.0.0"
	}

	// Convert RBAC
	if userPolicy.RBAC != nil {
		codePolicy.RBAC = c.convertRBAC(userPolicy.RBAC)
	}

	// Convert rules with engine information
	for i, userRule := range userPolicy.Rules {
		policyRule, err := c.convertRule(&userRule, userPolicy.Defaults, i)
		if err != nil {
			return nil, fmt.Errorf("failed to convert rule %d: %w", i, err)
		}

		// Update engine field based on mapping
		if engine, ok := engineMap[userRule.ID]; ok {
			policyRule.Check["engine"] = engine
		} else {
			// Fallback to llm-validator if no mapping found
			policyRule.Check["engine"] = "llm-validator"
		}

		codePolicy.Rules = append(codePolicy.Rules, *policyRule)
	}

	return codePolicy, nil
}

// convertRBAC converts user RBAC to policy RBAC
func (c *Converter) convertRBAC(userRBAC *schema.UserRBAC) *schema.PolicyRBAC {
	policyRBAC := &schema.PolicyRBAC{
		Roles: make(map[string]schema.PolicyRole),
	}

	for roleName, userRole := range userRBAC.Roles {
		permissions := make([]schema.Permission, 0)

		// Convert allowWrite
		for _, path := range userRole.AllowWrite {
			permissions = append(permissions, schema.Permission{
				Path:    path,
				Read:    true,
				Write:   true,
				Execute: false,
			})
		}

		// Convert denyWrite
		for _, path := range userRole.DenyWrite {
			permissions = append(permissions, schema.Permission{
				Path:    path,
				Read:    true,
				Write:   false,
				Execute: false,
			})
		}

		// Convert allowExec
		for _, path := range userRole.AllowExec {
			permissions = append(permissions, schema.Permission{
				Path:    path,
				Read:    true,
				Write:   false,
				Execute: true,
			})
		}

		policyRBAC.Roles[roleName] = schema.PolicyRole{
			Permissions: permissions,
		}
	}

	return policyRBAC
}

// convertRule converts a user rule to policy rule
func (c *Converter) convertRule(userRule *schema.UserRule, defaults *schema.UserDefaults, index int) (*schema.PolicyRule, error) {
	// Generate ID if not provided
	id := userRule.ID
	if id == "" {
		id = fmt.Sprintf("RULE-%d", index+1)
	}

	// Determine severity
	severity := userRule.Severity
	if severity == "" && defaults != nil {
		severity = defaults.Severity
	}
	if severity == "" {
		severity = "error"
	}

	// Build selector
	var selector *schema.Selector
	if len(userRule.Languages) > 0 || len(userRule.Include) > 0 || len(userRule.Exclude) > 0 {
		selector = &schema.Selector{
			Languages: userRule.Languages,
			Include:   userRule.Include,
			Exclude:   userRule.Exclude,
		}
	} else if defaults != nil && (len(defaults.Languages) > 0 || len(defaults.Include) > 0 || len(defaults.Exclude) > 0) {
		selector = &schema.Selector{
			Languages: defaults.Languages,
			Include:   defaults.Include,
			Exclude:   defaults.Exclude,
		}
	}

	// TODO: Implement intelligent rule inference based on userRule.Say
	// For now, create a basic check structure
	check := map[string]any{
		"engine": "custom",
		"desc":   userRule.Say,
	}

	// Merge params if provided
	for k, v := range userRule.Params {
		check[k] = v
	}

	// Build remedy
	var remedy *schema.Remedy
	autofix := userRule.Autofix
	if !autofix && defaults != nil {
		autofix = defaults.Autofix
	}
	if autofix {
		remedy = &schema.Remedy{
			Autofix: true,
		}
	}

	policyRule := &schema.PolicyRule{
		ID:       id,
		Enabled:  true,
		Category: userRule.Category,
		Severity: severity,
		Desc:     userRule.Say,
		When:     selector,
		Check:    check,
		Remedy:   remedy,
		Message:  userRule.Message,
	}

	if policyRule.Category == "" {
		policyRule.Category = "custom"
	}

	return policyRule, nil
}

// MultiTargetConvertOptions represents options for multi-target conversion
type MultiTargetConvertOptions struct {
	Targets             []string // Linter targets (e.g., "eslint", "checkstyle", "pmd")
	OutputDir           string   // Output directory for generated files
	ConfidenceThreshold float64  // Minimum confidence for LLM inference
}

// MultiTargetConvertResult represents the result of multi-target conversion
type MultiTargetConvertResult struct {
	CodePolicy   *schema.CodePolicy                  // Internal policy
	LinterConfigs map[string]*linters.LinterConfig   // Linter-specific configs
	Results      map[string]*linters.ConversionResult // Detailed results per linter
	Warnings     []string                             // Overall warnings
}

// ConvertMultiTarget converts user policy to multiple linter configurations
func (c *Converter) ConvertMultiTarget(ctx context.Context, userPolicy *schema.UserPolicy, opts MultiTargetConvertOptions) (*MultiTargetConvertResult, error) {
	if userPolicy == nil {
		return nil, fmt.Errorf("user policy is nil")
	}

	// Default options
	if opts.ConfidenceThreshold == 0 {
		opts.ConfidenceThreshold = 0.7
	}

	if len(opts.Targets) == 0 {
		opts.Targets = []string{"all"}
	}

	result := &MultiTargetConvertResult{
		CodePolicy:    nil, // Will be generated after linter conversion
		LinterConfigs: make(map[string]*linters.LinterConfig),
		Results:       make(map[string]*linters.ConversionResult),
		Warnings:      []string{},
	}

	// Resolve target linters
	targetConverters, err := c.resolveTargets(opts.Targets)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve targets: %w", err)
	}

	// Aggregate engine mappings: ruleID -> engine name
	engineMap := make(map[string]string)

	// Convert rules for each target linter
	for _, converter := range targetConverters {
		linterName := converter.Name()

		convResult, err := c.convertForLinter(ctx, userPolicy, converter, opts.ConfidenceThreshold)
		if err != nil {
			result.Warnings = append(result.Warnings, fmt.Sprintf("%s: conversion failed: %v", linterName, err))
			continue
		}

		result.Results[linterName] = convResult
		result.LinterConfigs[linterName] = convResult.Config

		// Aggregate engine mappings
		// Priority: first linter that successfully converts wins
		for ruleID, engine := range convResult.RuleEngineMap {
			if _, exists := engineMap[ruleID]; !exists || engine != "llm-validator" {
				engineMap[ruleID] = engine
			}
		}

		// Collect warnings
		for _, warning := range convResult.Warnings {
			result.Warnings = append(result.Warnings, fmt.Sprintf("%s: %s", linterName, warning))
		}
	}

	// Generate internal code policy with engine mappings
	codePolicy, err := c.convertWithEngines(userPolicy, engineMap)
	if err != nil {
		return nil, fmt.Errorf("failed to convert to code policy: %w", err)
	}

	result.CodePolicy = codePolicy

	return result, nil
}

// convertForLinter converts rules for a specific linter
func (c *Converter) convertForLinter(ctx context.Context, userPolicy *schema.UserPolicy, converter linters.LinterConverter, confidenceThreshold float64) (*linters.ConversionResult, error) {
	result := &linters.ConversionResult{
		LinterName: converter.Name(),
		Rules:      []*linters.LinterRule{},
		Warnings:   []string{},
		Errors:     []error{},
		RuleEngineMap: make(map[string]string), // Track which engine handles each rule
	}

	// Filter rules by language if needed
	supportedLangs := converter.SupportedLanguages()

	// Collect applicable rules
	type ruleWithIndex struct {
		rule  schema.UserRule
		index int
	}
	var applicableRules []ruleWithIndex

	for i, userRule := range userPolicy.Rules {
		if c.ruleAppliesToLanguages(userRule, supportedLangs, userPolicy.Defaults) {
			applicableRules = append(applicableRules, ruleWithIndex{rule: userRule, index: i})
		}
	}

	if len(applicableRules) == 0 {
		// No applicable rules, return empty result
		config, err := converter.GenerateConfig(result.Rules)
		if err != nil {
			return nil, fmt.Errorf("failed to generate config: %w", err)
		}
		result.Config = config
		return result, nil
	}

	// Infer rule intents in parallel
	if c.inferencer == nil {
		return nil, fmt.Errorf("LLM client not configured")
	}

	type inferenceJob struct {
		ruleWithIndex
		intent  *llm.RuleIntent
		err     error
		warning string
	}

	// Create worker pool with concurrency limit
	maxWorkers := 5 // Limit concurrent LLM API calls
	jobs := make(chan ruleWithIndex, len(applicableRules))
	results := make(chan inferenceJob, len(applicableRules))

	// Start workers
	for w := 0; w < maxWorkers; w++ {
		go func() {
			for job := range jobs {
				inferResult, err := c.inferencer.InferFromUserRule(ctx, &job.rule)

				jobResult := inferenceJob{
					ruleWithIndex: job,
				}

				if err != nil {
					jobResult.err = err
					jobResult.warning = fmt.Sprintf("Rule %d (%s): %v", job.index+1, job.rule.ID, err)
				} else {
					jobResult.intent = inferResult.Intent

					// Check confidence threshold
					if inferResult.Intent.Confidence < confidenceThreshold {
						jobResult.warning = fmt.Sprintf("Rule %d (%s): low confidence %.2f",
							job.index+1, job.rule.ID, inferResult.Intent.Confidence)
					}
				}

				results <- jobResult
			}
		}()
	}

	// Send jobs
	for _, rule := range applicableRules {
		jobs <- rule
	}
	close(jobs)

	// Collect results
	inferenceResults := make(map[int]inferenceJob)
	for i := 0; i < len(applicableRules); i++ {
		jobResult := <-results
		inferenceResults[jobResult.index] = jobResult
	}
	close(results)

	// Process results in original order
	for _, ruleInfo := range applicableRules {
		jobResult := inferenceResults[ruleInfo.index]

		if jobResult.err != nil {
			result.Warnings = append(result.Warnings, jobResult.warning)
			continue
		}

		if jobResult.warning != "" {
			result.Warnings = append(result.Warnings, jobResult.warning)
		}

		// Convert to linter-specific rule
		linterRule, err := converter.Convert(&ruleInfo.rule, jobResult.intent)
		if err != nil {
			result.Errors = append(result.Errors, fmt.Errorf("rule %d: %w", ruleInfo.index+1, err))
			continue
		}

		result.Rules = append(result.Rules, linterRule)

		// Track which engine will validate this rule
		// Check if the rule has meaningful configuration content
		hasContent := false
		if len(linterRule.Config) > 0 {
			// Check if config has actual rule content (not just empty nested structures)
			for key, value := range linterRule.Config {
				if key == "modules" && value != nil {
					// For Checkstyle, check if modules slice is not empty
					v := reflect.ValueOf(value)
					if v.Kind() == reflect.Slice && v.Len() > 0 {
						hasContent = true
						break
					}
				} else if key == "rules" && value != nil {
					// For PMD, check if rules array/slice is not empty
					v := reflect.ValueOf(value)
					if v.Kind() == reflect.Slice && v.Len() > 0 {
						hasContent = true
						break
					}
				} else if key != "" && value != nil {
					// For ESLint and other formats with direct config
					hasContent = true
					break
				}
			}
		}

		if hasContent {
			result.RuleEngineMap[ruleInfo.rule.ID] = converter.Name()
		} else {
			result.RuleEngineMap[ruleInfo.rule.ID] = "llm-validator"
		}
	}

	// Generate final configuration
	config, err := converter.GenerateConfig(result.Rules)
	if err != nil {
		return nil, fmt.Errorf("failed to generate config: %w", err)
	}

	result.Config = config

	return result, nil
}

// resolveTargets resolves target names to converters
func (c *Converter) resolveTargets(targets []string) ([]linters.LinterConverter, error) {
	if len(targets) == 1 && strings.ToLower(targets[0]) == "all" {
		// Return all registered converters
		return linters.GetAll(), nil
	}

	converters := []linters.LinterConverter{}
	for _, target := range targets {
		converter, err := linters.Get(target)
		if err != nil {
			return nil, fmt.Errorf("target %s: %w", target, err)
		}
		converters = append(converters, converter)
	}

	return converters, nil
}

// ruleAppliesToLanguages checks if a rule applies to any of the supported languages
func (c *Converter) ruleAppliesToLanguages(rule schema.UserRule, supportedLangs []string, defaults *schema.UserDefaults) bool {
	// Get rule's target languages
	targetLangs := rule.Languages
	if len(targetLangs) == 0 && defaults != nil {
		targetLangs = defaults.Languages
	}

	// If no target languages specified, apply to all
	if len(targetLangs) == 0 {
		return true
	}

	// Check if any target language matches supported languages
	for _, targetLang := range targetLangs {
		targetLang = strings.ToLower(targetLang)
		for _, supportedLang := range supportedLangs {
			supportedLang = strings.ToLower(supportedLang)
			if targetLang == supportedLang || strings.Contains(supportedLang, targetLang) || strings.Contains(targetLang, supportedLang) {
				return true
			}
		}
	}

	return false
}

// GetAll is a helper to get all registered converters
func GetAll() []linters.LinterConverter {
	registry := linters.List()
	converters := make([]linters.LinterConverter, 0, len(registry))
	for _, name := range registry {
		converter, err := linters.Get(name)
		if err == nil {
			converters = append(converters, converter)
		}
	}
	return converters
}
