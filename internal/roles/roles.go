package roles

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/DevSymphony/sym-cli/internal/envutil"
)

// Roles represents a map of role names to lists of usernames
// This allows dynamic role creation instead of hardcoded admin/developer/viewer
type Roles map[string][]string

// Environment variable key for current role
const currentRoleKey = "CURRENT_ROLE"

// getSymDir returns the .sym directory path (current working directory based)
func getSymDir() (string, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	return filepath.Join(cwd, ".sym"), nil
}

// getEnvPath returns the path to .sym/.env file
func getEnvPath() (string, error) {
	symDir, err := getSymDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(symDir, ".env"), nil
}

// GetRolesPath returns the path to the roles.json file in the current directory
func GetRolesPath() (string, error) {
	symDir, err := getSymDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(symDir, "roles.json"), nil
}

// LoadRoles loads the roles from the .sym/roles.json file
func LoadRoles() (Roles, error) {
	rolesPath, err := GetRolesPath()
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(rolesPath)
	if err != nil {
		if os.IsNotExist(err) {
			// symphonyclient integration: symphony → sym command
			return nil, fmt.Errorf("roles.json not found. Run 'sym init' to create it")
		}
		return nil, err
	}

	var roles Roles
	if err := json.Unmarshal(data, &roles); err != nil {
		return nil, fmt.Errorf("invalid roles.json: %w", err)
	}

	return roles, nil
}

// SaveRoles saves the roles to the .sym/roles.json file
func SaveRoles(roles Roles) error {
	rolesPath, err := GetRolesPath()
	if err != nil {
		return err
	}

	// symphonyclient integration: Ensure .sym directory exists
	symDir := filepath.Dir(rolesPath)
	if err := os.MkdirAll(symDir, 0755); err != nil {
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

// GetCurrentRole returns the currently selected role from .sym/.env (CURRENT_ROLE key)
// If no role is selected, returns empty string and nil error
func GetCurrentRole() (string, error) {
	envPath, err := getEnvPath()
	if err != nil {
		return "", err
	}

	role := envutil.LoadKeyFromEnvFile(envPath, currentRoleKey)
	return role, nil
}

// SetCurrentRole saves the selected role to .sym/.env (CURRENT_ROLE key)
func SetCurrentRole(role string) error {
	envPath, err := getEnvPath()
	if err != nil {
		return err
	}

	return envutil.SaveKeyToEnvFile(envPath, currentRoleKey, role)
}

// CurrentRoleExists checks if CURRENT_ROLE is set in .sym/.env
func CurrentRoleExists() (bool, error) {
	role, err := GetCurrentRole()
	if err != nil {
		return false, err
	}
	return role != "", nil
}

// GetAvailableRoles returns all role names defined in roles.json
// Returns roles sorted alphabetically for consistent ordering
func GetAvailableRoles() ([]string, error) {
	roles, err := LoadRoles()
	if err != nil {
		return nil, err
	}

	roleNames := make([]string, 0, len(roles))
	for roleName := range roles {
		roleNames = append(roleNames, roleName)
	}
	sort.Strings(roleNames)
	return roleNames, nil
}

// IsValidRole checks if a role name exists in roles.json
func IsValidRole(role string) (bool, error) {
	roles, err := LoadRoles()
	if err != nil {
		return false, err
	}

	_, exists := roles[role]
	return exists, nil
}
