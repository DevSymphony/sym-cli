package importer

import (
	"context"
	"fmt"

	"github.com/DevSymphony/sym-cli/internal/llm"
	"github.com/DevSymphony/sym-cli/internal/policy"
	"github.com/DevSymphony/sym-cli/pkg/schema"
)

// Importer handles the complete import workflow
type Importer struct {
	reader    *Reader
	extractor *Extractor
	verbose   bool
}

// NewImporter creates a new Importer instance
func NewImporter(provider llm.Provider, verbose bool) *Importer {
	return &Importer{
		reader:    NewReader(verbose),
		extractor: NewExtractor(provider, verbose),
		verbose:   verbose,
	}
}

// Import executes the import workflow for a single file
func (i *Importer) Import(ctx context.Context, input *ImportInput) (*ImportResult, error) {
	result := &ImportResult{
		CategoriesAdded: []schema.CategoryDef{},
		RulesAdded:      []schema.UserRule{},
		Warnings:        []string{},
	}

	// Validate input
	if input.Path == "" {
		return nil, fmt.Errorf("file path is required")
	}

	// Step 1: Read the file
	doc, err := i.reader.ReadFile(ctx, input.Path)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}
	result.FileProcessed = doc.Path

	// Step 2: Extract conventions using LLM
	extracted, err := i.extractor.Extract(ctx, doc)
	if err != nil {
		return nil, fmt.Errorf("failed to extract conventions: %w", err)
	}

	if len(extracted.Categories) == 0 && len(extracted.Rules) == 0 {
		result.Warnings = append(result.Warnings, "No conventions found in the document")
		return result, nil
	}

	// Step 3: Load existing policy
	existingPolicy, err := policy.LoadPolicy("")
	if err != nil {
		return nil, fmt.Errorf("failed to load existing policy: %w", err)
	}

	// Step 4: Apply import mode
	if input.Mode == ImportModeClear {
		result.CategoriesRemoved = len(existingPolicy.Category)
		result.RulesRemoved = len(existingPolicy.Rules)
		existingPolicy.Category = []schema.CategoryDef{}
		existingPolicy.Rules = []schema.UserRule{}
		existingPolicy.Defaults.Languages = []string{}
	}

	// Step 5: Generate unique IDs and merge
	newCategories, newRules := i.assignUniqueIDs(existingPolicy, extracted, result)

	existingPolicy.Category = append(existingPolicy.Category, newCategories...)
	existingPolicy.Rules = append(existingPolicy.Rules, newRules...)

	result.CategoriesAdded = newCategories
	result.RulesAdded = newRules

	// Step 5.5: Update defaults.languages with new languages from rules
	i.updateDefaultsLanguages(existingPolicy, newRules)

	// Step 6: Save updated policy
	if err := policy.SavePolicy(existingPolicy, ""); err != nil {
		return result, fmt.Errorf("failed to save policy: %w", err)
	}

	return result, nil
}

// assignUniqueIDs generates unique IDs for all extracted items
func (i *Importer) assignUniqueIDs(
	existing *schema.UserPolicy,
	extracted *ExtractedConventions,
	result *ImportResult,
) ([]schema.CategoryDef, []schema.UserRule) {
	// Build map of existing category names
	existingCategoryNames := make(map[string]bool)
	for _, cat := range existing.Category {
		existingCategoryNames[cat.Name] = true
	}

	// Build map of existing rule IDs
	existingRuleIDs := make(map[string]bool)
	for _, rule := range existing.Rules {
		existingRuleIDs[rule.ID] = true
	}

	// Process categories: skip duplicates
	var newCategories []schema.CategoryDef
	for _, cat := range extracted.Categories {
		if existingCategoryNames[cat.Name] {
			result.Warnings = append(result.Warnings,
				fmt.Sprintf("Category '%s' already exists, skipped", cat.Name))
			continue
		}
		newCategories = append(newCategories, cat)
		existingCategoryNames[cat.Name] = true
	}

	// Process rules: generate unique IDs
	var newRules []schema.UserRule
	for _, rule := range extracted.Rules {
		uniqueID := i.generateUniqueID(rule.ID, existingRuleIDs)
		if uniqueID != rule.ID {
			result.Warnings = append(result.Warnings,
				fmt.Sprintf("Rule ID '%s' already exists, renamed to '%s'", rule.ID, uniqueID))
		}
		rule.ID = uniqueID
		existingRuleIDs[uniqueID] = true
		newRules = append(newRules, rule)
	}

	return newCategories, newRules
}

// generateUniqueID generates a unique rule ID
func (i *Importer) generateUniqueID(baseID string, existingIDs map[string]bool) string {
	if !existingIDs[baseID] {
		return baseID
	}

	counter := 1
	for {
		newID := fmt.Sprintf("%s-%d", baseID, counter)
		if !existingIDs[newID] {
			return newID
		}
		counter++
	}
}

// updateDefaultsLanguages adds new languages from rules to defaults.languages
func (i *Importer) updateDefaultsLanguages(p *schema.UserPolicy, newRules []schema.UserRule) {
	// Build set of existing default languages
	existingLangs := make(map[string]bool)
	for _, lang := range p.Defaults.Languages {
		existingLangs[lang] = true
	}

	// Collect new languages from rules
	for _, rule := range newRules {
		for _, lang := range rule.Languages {
			if !existingLangs[lang] {
				p.Defaults.Languages = append(p.Defaults.Languages, lang)
				existingLangs[lang] = true
			}
		}
	}
}
