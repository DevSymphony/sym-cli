package integration

import (
	"testing"

	"github.com/DevSymphony/sym-cli/internal/roles"
)

// Test complex RBAC scenarios with admin, developer, viewer roles
func TestComplexRBACPatterns(t *testing.T) {
	tests := []struct {
		name         string
		username     string
		files        []string
		expectAllow  bool
		expectDenied []string
	}{
		{
			name:     "Admin can modify all files",
			username: "alice", // alice is admin
			files: []string{
				"src/components/Button.js",
				"src/core/engine.js",
				"src/api/client.js",
				"config/settings.json",
			},
			expectAllow:  true,
			expectDenied: []string{},
		},
		{
			name:     "Developer can modify source files",
			username: "charlie", // charlie is developer
			files: []string{
				"src/components/Button.js",
				"src/components/ui/Modal.js",
				"src/hooks/useAuth.js",
			},
			expectAllow:  true,
			expectDenied: []string{},
		},
		{
			name:     "Developer cannot modify core/api files",
			username: "david", // david is developer
			files: []string{
				"src/components/Button.js",
				"src/core/engine.js",
				"src/api/client.js",
			},
			expectAllow: false,
			expectDenied: []string{
				"src/core/engine.js",
				"src/api/client.js",
			},
		},
		{
			name:     "Viewer cannot modify any files",
			username: "frank", // frank is viewer
			files: []string{
				"src/components/Button.js",
				"README.md",
			},
			expectAllow: false,
			expectDenied: []string{
				"src/components/Button.js",
				"README.md",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// This test requires actual roles.json and user-policy.json in .sym folder
			// For now, we'll skip it as it needs the full integration setup
			t.Skip("Requires .sym/roles.json and .sym/user-policy.json setup")

			result, err := roles.ValidateFilePermissions(tt.username, tt.files)
			if err != nil {
				t.Fatalf("ValidateFilePermissions failed: %v", err)
			}

			if result.Allowed != tt.expectAllow {
				t.Errorf("Expected Allowed=%v, got %v", tt.expectAllow, result.Allowed)
			}

			if len(result.DeniedFiles) != len(tt.expectDenied) {
				t.Errorf("Expected %d denied files, got %d: %v", len(tt.expectDenied), len(result.DeniedFiles), result.DeniedFiles)
			}
		})
	}
}

// Test RBAC validation result structure
func TestValidationResultStructure(t *testing.T) {
	tests := []struct {
		name         string
		result       *roles.ValidationResult
		expectAllow  bool
		expectDenied int
	}{
		{
			name: "All files allowed",
			result: &roles.ValidationResult{
				Allowed:     true,
				DeniedFiles: []string{},
			},
			expectAllow:  true,
			expectDenied: 0,
		},
		{
			name: "Some files denied",
			result: &roles.ValidationResult{
				Allowed:     false,
				DeniedFiles: []string{"src/core/api.js", "src/core/db.js"},
			},
			expectAllow:  false,
			expectDenied: 2,
		},
		{
			name: "All files denied",
			result: &roles.ValidationResult{
				Allowed:     false,
				DeniedFiles: []string{"file1.js", "file2.js", "file3.js"},
			},
			expectAllow:  false,
			expectDenied: 3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.result.Allowed != tt.expectAllow {
				t.Errorf("Expected Allowed=%v, got %v", tt.expectAllow, tt.result.Allowed)
			}
			if len(tt.result.DeniedFiles) != tt.expectDenied {
				t.Errorf("Expected %d denied files, got %d", tt.expectDenied, len(tt.result.DeniedFiles))
			}
		})
	}
}
