package roles

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"github.com/DevSymphony/sym-cli/internal/git"
)

// Roles represents a map of role names to lists of usernames
// This allows dynamic role creation instead of hardcoded admin/developer/viewer
type Roles map[string][]string

// GetRolesPath returns the path to the roles.json file in the current repo
func GetRolesPath() (string, error) {
	repoRoot, err := git.GetRepoRoot()
	if err != nil {
		return "", err
	}
	return filepath.Join(repoRoot, ".github", "roles.json"), nil
}

// LoadRoles loads the roles from the .github/roles.json file
func LoadRoles() (Roles, error) {
	rolesPath, err := GetRolesPath()
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(rolesPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("roles.json not found. Run 'symphony init' to create it")
		}
		return nil, err
	}

	var roles Roles
	if err := json.Unmarshal(data, &roles); err != nil {
		return nil, fmt.Errorf("invalid roles.json: %w", err)
	}

	return roles, nil
}

// SaveRoles saves the roles to the .github/roles.json file
func SaveRoles(roles Roles) error {
	rolesPath, err := GetRolesPath()
	if err != nil {
		return err
	}

	// Ensure .github directory exists
	githubDir := filepath.Dir(rolesPath)
	if err := os.MkdirAll(githubDir, 0755); err != nil {
		return err
	}

	data, err := json.MarshalIndent(roles, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(rolesPath, data, 0644)
}

// GetUserRole returns the role of a specific user
func GetUserRole(username string) (string, error) {
	roles, err := LoadRoles()
	if err != nil {
		return "", err
	}

	// Iterate through all roles dynamically
	for roleName, usernames := range roles {
		for _, user := range usernames {
			if user == username {
				return roleName, nil
			}
		}
	}

	return "none", nil
}

// IsAdmin checks if a user is in the "admin" role
// NOTE: This is deprecated. Use RBAC canEditPolicy permission instead.
func IsAdmin(username string) (bool, error) {
	roles, err := LoadRoles()
	if err != nil {
		return false, err
	}

	// Check if admin role exists and contains the user
	if adminUsers, exists := roles["admin"]; exists {
		for _, admin := range adminUsers {
			if admin == username {
				return true, nil
			}
		}
	}

	return false, nil
}

// RolesExists checks if roles.json file exists
func RolesExists() (bool, error) {
	rolesPath, err := GetRolesPath()
	if err != nil {
		return false, err
	}

	_, err = os.Stat(rolesPath)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
}
