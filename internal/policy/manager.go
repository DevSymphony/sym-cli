package policy

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/DevSymphony/sym-cli/internal/util/git"
	"github.com/DevSymphony/sym-cli/pkg/schema"
)

var defaultPolicyPath = ".sym/user-policy.json"

// GetPolicyPath returns the configured or default policy file path
func GetPolicyPath(customPath string) (string, error) {
	repoRoot, err := git.GetRepoRoot()
	if err != nil {
		return "", err
	}

	policyPath := defaultPolicyPath
	if customPath != "" {
		policyPath = customPath
	}

	return filepath.Join(repoRoot, policyPath), nil
}

// LoadPolicy loads the policy from the configured path
func LoadPolicy(customPath string) (*schema.UserPolicy, error) {
	policyPath, err := GetPolicyPath(customPath)
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(policyPath)
	if err != nil {
		if os.IsNotExist(err) {
			// Return empty policy if file doesn't exist
			return &schema.UserPolicy{
				Version: "1.0.0",
				Rules:   []schema.UserRule{},
			}, nil
		}
		return nil, err
	}

	var policy schema.UserPolicy
	if err := json.Unmarshal(data, &policy); err != nil {
		return nil, fmt.Errorf("invalid policy file: %w", err)
	}

	return &policy, nil
}

// SavePolicy saves the policy to the configured path
func SavePolicy(policy *schema.UserPolicy, customPath string) error {
	policyPath, err := GetPolicyPath(customPath)
	if err != nil {
		return err
	}

	// Ensure directory exists
	dir := filepath.Dir(policyPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	// Validate policy before saving
	if err := ValidatePolicy(policy); err != nil {
		return fmt.Errorf("policy validation failed: %w", err)
	}

	data, err := json.MarshalIndent(policy, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(policyPath, data, 0644)
}

// ValidatePolicy validates the policy structure
func ValidatePolicy(policy *schema.UserPolicy) error {
	if policy == nil {
		return fmt.Errorf("policy cannot be nil")
	}

	if policy.Version == "" {
		return fmt.Errorf("policy version is required")
	}

	// Validate rules
	ruleIDs := make(map[string]bool)
	for i, rule := range policy.Rules {
		if rule.Say == "" {
			return fmt.Errorf("rule %d: 'say' field is required", i+1)
		}
		if rule.ID == "" {
			return fmt.Errorf("rule %d: 'id' field is required", i+1)
		}
		if ruleIDs[rule.ID] {
			return fmt.Errorf("duplicate rule id: %s", rule.ID)
		}
		ruleIDs[rule.ID] = true
	}

	// Validate RBAC roles
	if policy.RBAC != nil {
		hasAtLeastOneEditor := false
		for roleName, role := range policy.RBAC.Roles {
			if roleName == "" {
				return fmt.Errorf("RBAC role name cannot be empty")
			}
			if len(role.AllowWrite) == 0 && len(role.DenyWrite) == 0 {
				return fmt.Errorf("RBAC role '%s' must have at least one allowWrite or denyWrite rule", roleName)
			}
			if role.CanEditPolicy {
				hasAtLeastOneEditor = true
			}
		}

		// Ensure at least one role can edit policy (safety check)
		if len(policy.RBAC.Roles) > 0 && !hasAtLeastOneEditor {
			return fmt.Errorf("at least one role must have 'canEditPolicy' permission")
		}
	}

	return nil
}

// PolicyExists checks if the policy file exists
func PolicyExists(customPath string) (bool, error) {
	policyPath, err := GetPolicyPath(customPath)
	if err != nil {
		return false, err
	}

	_, err = os.Stat(policyPath)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
}
