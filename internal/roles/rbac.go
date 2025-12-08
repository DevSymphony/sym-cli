package roles

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/DevSymphony/sym-cli/internal/git"
	"github.com/DevSymphony/sym-cli/internal/policy"
	"github.com/DevSymphony/sym-cli/pkg/schema"
)

// ValidationResult represents the result of RBAC validation
type ValidationResult struct {
	Allowed     bool     // true if all files are allowed, false if any are denied
	DeniedFiles []string // list of files that are denied (empty if Allowed is true)
}

// GetUserPolicyPath returns the path to user-policy.json in the current repo
func GetUserPolicyPath() (string, error) {
	repoRoot, err := git.GetRepoRoot()
	if err != nil {
		return "", err
	}
	return filepath.Join(repoRoot, ".sym", "user-policy.json"), nil
}

// LoadUserPolicyFromRepo loads user-policy.json from the current repository
func LoadUserPolicyFromRepo() (*schema.UserPolicy, error) {
	policyPath, err := GetUserPolicyPath()
	if err != nil {
		return nil, err
	}

	// Check if file exists
	if _, err := os.Stat(policyPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("user-policy.json not found at %s. Run 'sym init' to create it", policyPath)
	}

	// Use existing loader
	loader := policy.NewLoader(false)
	return loader.LoadUserPolicy(policyPath)
}

// matchPattern checks if a file path matches a glob pattern
// Supports ** (match any directory level) and * (match within directory)
func matchPattern(pattern, path string) bool {
	// Normalize paths
	pattern = filepath.ToSlash(pattern)
	path = filepath.ToSlash(path)

	// Handle ** pattern (match any directory level)
	if strings.Contains(pattern, "**") {
		parts := strings.Split(pattern, "**")
		if len(parts) == 2 {
			prefix := strings.TrimSuffix(parts[0], "/")
			suffix := strings.TrimPrefix(parts[1], "/")

			// Check prefix
			if prefix != "" && !strings.HasPrefix(path, prefix) {
				return false
			}

			// Check suffix
			if suffix != "" {
				// Remove prefix from path
				remaining := path
				if prefix != "" {
					remaining = strings.TrimPrefix(path, prefix+"/")
				}

				// Check if suffix matches
				if suffix == "*" {
					return true
				}
				if strings.HasSuffix(suffix, "/*") {
					// Match directory and any file in it
					dir := strings.TrimSuffix(suffix, "/*")
					return strings.Contains(remaining, dir+"/") || strings.HasPrefix(remaining, dir+"/")
				}
				// Exact match or contains the path
				return strings.Contains(remaining, suffix) || strings.HasSuffix(remaining, suffix)
			}
			return true
		}
	}

	// Handle simple * pattern
	if strings.Contains(pattern, "*") {
		matched, _ := filepath.Match(pattern, path)
		return matched
	}

	// Exact match or prefix match
	if strings.HasSuffix(pattern, "/") {
		return strings.HasPrefix(path, pattern)
	}

	return path == pattern || strings.HasPrefix(path, pattern+"/")
}

// checkFilePermission checks if a single file is allowed for the given role
func checkFilePermission(filePath string, role *schema.UserRole) bool {
	// Check denyWrite first (deny takes precedence)
	for _, denyPattern := range role.DenyWrite {
		if matchPattern(denyPattern, filePath) {
			return false
		}
	}

	// If no allowWrite patterns, allow by default
	if len(role.AllowWrite) == 0 {
		return true
	}

	// Check allowWrite patterns
	for _, allowPattern := range role.AllowWrite {
		if matchPattern(allowPattern, filePath) {
			return true
		}
	}

	// Not explicitly allowed
	return false
}

// ValidateFilePermissions validates if a user can modify the given files
// Returns ValidationResult with Allowed=true if all files are permitted,
// or Allowed=false with a list of denied files
func ValidateFilePermissions(username string, files []string) (*ValidationResult, error) {
	// Get user's role (this internally loads roles.json)
	userRole, err := GetUserRole(username)
	if err != nil {
		return nil, fmt.Errorf("failed to get user role: %w", err)
	}

	return ValidateFilePermissionsForRole(userRole, files)
}

// ValidateFilePermissionsForRole validates if a role can modify the given files
// This is the local role-based version that takes a role name directly
// Returns ValidationResult with Allowed=true if all files are permitted,
// or Allowed=false with a list of denied files
func ValidateFilePermissionsForRole(role string, files []string) (*ValidationResult, error) {
	if role == "" || role == "none" {
		return &ValidationResult{
			Allowed:     false,
			DeniedFiles: files, // All files denied if no role selected
		}, nil
	}

	// Load user-policy.json
	userPolicy, err := LoadUserPolicyFromRepo()
	if err != nil {
		return nil, fmt.Errorf("failed to load user policy: %w", err)
	}

	// Check if RBAC is defined in policy
	if userPolicy.RBAC == nil || userPolicy.RBAC.Roles == nil {
		// No RBAC defined, allow all files
		return &ValidationResult{
			Allowed:     true,
			DeniedFiles: []string{},
		}, nil
	}

	// Get role configuration from policy
	roleConfig, exists := userPolicy.RBAC.Roles[role]
	if !exists {
		// Role not defined in policy, deny all
		return &ValidationResult{
			Allowed:     false,
			DeniedFiles: files,
		}, nil
	}

	// Check each file
	deniedFiles := []string{}
	for _, file := range files {
		if !checkFilePermission(file, &roleConfig) {
			deniedFiles = append(deniedFiles, file)
		}
	}

	// Return result
	if len(deniedFiles) == 0 {
		return &ValidationResult{
			Allowed:     true,
			DeniedFiles: []string{},
		}, nil
	}

	return &ValidationResult{
		Allowed:     false,
		DeniedFiles: deniedFiles,
	}, nil
}
