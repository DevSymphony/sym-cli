package policy

import "github.com/DevSymphony/sym-cli/pkg/schema"

// UpdateDefaultsLanguages adds new languages from rules to defaults.languages.
// This function is used by the importer and convention add/edit operations
// to automatically track languages used in the project.
func UpdateDefaultsLanguages(p *schema.UserPolicy, rules []schema.UserRule) {
	if p.Defaults == nil {
		p.Defaults = &schema.UserDefaults{}
	}

	// Build set of existing default languages
	existingLangs := make(map[string]bool)
	for _, lang := range p.Defaults.Languages {
		existingLangs[lang] = true
	}

	// Collect new languages from rules
	for _, rule := range rules {
		for _, lang := range rule.Languages {
			if !existingLangs[lang] {
				p.Defaults.Languages = append(p.Defaults.Languages, lang)
				existingLangs[lang] = true
			}
		}
	}
}
