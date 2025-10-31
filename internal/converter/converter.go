package converter

import (
	"context"
	"fmt"
	"strings"

	"github.com/DevSymphony/sym-cli/internal/converter/linters"
	"github.com/DevSymphony/sym-cli/internal/llm"
	"github.com/DevSymphony/sym-cli/pkg/schema"
)

// Converter converts user policy (A schema) to code policy (B schema)
type Converter struct {
	verbose    bool
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
func NewConverter(verbose bool, opts ...ConverterOption) *Converter {
	c := &Converter{
		verbose: verbose,
	}

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

	// Generate internal code policy first
	codePolicy, err := c.Convert(userPolicy)
	if err != nil {
		return nil, fmt.Errorf("failed to convert to code policy: %w", err)
	}

	result := &MultiTargetConvertResult{
		CodePolicy:    codePolicy,
		LinterConfigs: make(map[string]*linters.LinterConfig),
		Results:       make(map[string]*linters.ConversionResult),
		Warnings:      []string{},
	}

	// Resolve target linters
	targetConverters, err := c.resolveTargets(opts.Targets)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve targets: %w", err)
	}

	if c.verbose {
		fmt.Printf("Converting rules for %d linter(s): %v\n", len(targetConverters), c.getConverterNames(targetConverters))
	}

	// Convert rules for each target linter
	for _, converter := range targetConverters {
		linterName := converter.Name()

		if c.verbose {
			fmt.Printf("Converting rules for %s...\n", linterName)
		}

		convResult, err := c.convertForLinter(ctx, userPolicy, converter, opts.ConfidenceThreshold)
		if err != nil {
			result.Warnings = append(result.Warnings, fmt.Sprintf("%s: conversion failed: %v", linterName, err))
			continue
		}

		result.Results[linterName] = convResult
		result.LinterConfigs[linterName] = convResult.Config

		// Collect warnings
		for _, warning := range convResult.Warnings {
			result.Warnings = append(result.Warnings, fmt.Sprintf("%s: %s", linterName, warning))
		}
	}

	return result, nil
}

// convertForLinter converts rules for a specific linter
func (c *Converter) convertForLinter(ctx context.Context, userPolicy *schema.UserPolicy, converter linters.LinterConverter, confidenceThreshold float64) (*linters.ConversionResult, error) {
	result := &linters.ConversionResult{
		LinterName: converter.Name(),
		Rules:      []*linters.LinterRule{},
		Warnings:   []string{},
		Errors:     []error{},
	}

	// Filter rules by language if needed
	supportedLangs := converter.SupportedLanguages()

	for i, userRule := range userPolicy.Rules {
		// Check if rule applies to this linter's languages
		if !c.ruleAppliesToLanguages(userRule, supportedLangs, userPolicy.Defaults) {
			if c.verbose {
				fmt.Printf("  Skipping rule %d: not applicable to %s languages\n", i+1, converter.Name())
			}
			continue
		}

		// Infer rule intent using LLM
		var intent *llm.RuleIntent
		if c.inferencer != nil {
			inferResult, err := c.inferencer.InferFromUserRule(ctx, &userRule)
			if err != nil {
				result.Warnings = append(result.Warnings, fmt.Sprintf("Rule %d: inference failed: %v", i+1, err))
				// Use fallback intent
				intent = c.createFallbackIntent(&userRule)
			} else {
				intent = inferResult.Intent

				// Check confidence threshold
				if intent.Confidence < confidenceThreshold {
					result.Warnings = append(result.Warnings,
						fmt.Sprintf("Rule %d: low confidence (%.2f < %.2f): %s",
							i+1, intent.Confidence, confidenceThreshold, userRule.Say))
				}

				if c.verbose && inferResult.UsedCache {
					fmt.Printf("  Rule %d: using cached inference\n", i+1)
				}
			}
		} else {
			// No LLM client, use fallback
			intent = c.createFallbackIntent(&userRule)
			result.Warnings = append(result.Warnings,
				fmt.Sprintf("Rule %d: no LLM client, using fallback inference", i+1))
		}

		// Convert to linter-specific rule
		linterRule, err := converter.Convert(&userRule, intent)
		if err != nil {
			result.Errors = append(result.Errors, fmt.Errorf("rule %d: %w", i+1, err))
			continue
		}

		result.Rules = append(result.Rules, linterRule)

		if c.verbose {
			fmt.Printf("  Rule %d: converted successfully (engine: %s, confidence: %.2f)\n",
				i+1, intent.Engine, intent.Confidence)
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

// createFallbackIntent creates a fallback intent when LLM is unavailable
func (c *Converter) createFallbackIntent(userRule *schema.UserRule) *llm.RuleIntent {
	intent := &llm.RuleIntent{
		Engine:     "custom",
		Category:   userRule.Category,
		Params:     make(map[string]any),
		Confidence: 0.5,
		Reasoning:  "Fallback: no LLM inference available",
	}

	// Copy params
	for k, v := range userRule.Params {
		intent.Params[k] = v
	}

	// Use category as a hint for engine type
	if userRule.Category != "" {
		switch strings.ToLower(userRule.Category) {
		case "naming":
			intent.Engine = "pattern"
		case "formatting", "style":
			intent.Engine = "style"
		case "length":
			intent.Engine = "length"
		}
	}

	if intent.Category == "" {
		intent.Category = "custom"
	}

	return intent
}

// getConverterNames returns converter names for logging
func (c *Converter) getConverterNames(converters []linters.LinterConverter) []string {
	names := make([]string, len(converters))
	for i, converter := range converters {
		names[i] = converter.Name()
	}
	return names
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
