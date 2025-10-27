package converter

import (
	"fmt"

	"github.com/DevSymphony/sym-cli/pkg/schema"
)

// Converter converts user policy (A schema) to code policy (B schema)
type Converter struct {
	verbose bool
}

// NewConverter creates a new converter
func NewConverter(verbose bool) *Converter {
	return &Converter{
		verbose: verbose,
	}
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
