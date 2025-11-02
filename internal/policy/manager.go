package policy

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"symphony/internal/git"
)

// Policy represents the user-policy.json structure
type Policy struct {
	Version  string            `json:"version,omitempty"`
	RBAC     *RBAC             `json:"rbac,omitempty"`
	Defaults *Defaults         `json:"defaults,omitempty"`
	Rules    []Rule            `json:"rules"`
}

// RBAC represents role-based access control settings
type RBAC struct {
	Roles map[string]RBACRole `json:"roles"`
}

// RBACRole represents a single RBAC role
type RBACRole struct {
	AllowWrite    []string `json:"allowWrite,omitempty"`
	DenyWrite     []string `json:"denyWrite,omitempty"`
	CanEditPolicy bool     `json:"canEditPolicy,omitempty"` // Permission to edit policy
	CanEditRoles  bool     `json:"canEditRoles,omitempty"`  // Permission to edit roles
}

// Defaults represents global default settings
type Defaults struct {
	Languages []string `json:"languages,omitempty"`
	Severity  string   `json:"severity,omitempty"`
	Autofix   bool     `json:"autofix,omitempty"`
}

// Rule represents a single coding rule
type Rule struct {
	No        int      `json:"no"`
	Say       string   `json:"say"`
	Category  string   `json:"category,omitempty"`
	Languages []string `json:"languages,omitempty"`
	Example   string   `json:"example,omitempty"` // Optional example code
}

var defaultPolicyPath = ".github/user-policy.json"

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
func LoadPolicy(customPath string) (*Policy, error) {
	policyPath, err := GetPolicyPath(customPath)
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(policyPath)
	if err != nil {
		if os.IsNotExist(err) {
			// Return empty policy if file doesn't exist
			return &Policy{
				Version: "1.0.0",
				Rules:   []Rule{},
			}, nil
		}
		return nil, err
	}

	var policy Policy
	if err := json.Unmarshal(data, &policy); err != nil {
		return nil, fmt.Errorf("invalid policy file: %w", err)
	}

	return &policy, nil
}

// SavePolicy saves the policy to the configured path
func SavePolicy(policy *Policy, customPath string) error {
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
func ValidatePolicy(policy *Policy) error {
	if policy == nil {
		return fmt.Errorf("policy cannot be nil")
	}

	if policy.Version == "" {
		return fmt.Errorf("policy version is required")
	}

	// Validate rules
	ruleNumbers := make(map[int]bool)
	for i, rule := range policy.Rules {
		if rule.Say == "" {
			return fmt.Errorf("rule %d: 'say' field is required", i+1)
		}
		if rule.No <= 0 {
			return fmt.Errorf("rule %d: 'no' field must be positive", i+1)
		}
		if ruleNumbers[rule.No] {
			return fmt.Errorf("duplicate rule number: %d", rule.No)
		}
		ruleNumbers[rule.No] = true
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
