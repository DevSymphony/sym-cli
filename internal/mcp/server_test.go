package mcp

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/DevSymphony/sym-cli/internal/policy"
	"github.com/DevSymphony/sym-cli/pkg/schema"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// runGitInit initializes a git repository in the given directory
func runGitInit(dir string) error {
	cmd := exec.Command("git", "init")
	cmd.Dir = dir
	return cmd.Run()
}

func TestQueryConventions(t *testing.T) {
	// Setup: Create a temporary user policy
	tmpDir := t.TempDir()
	userPolicyPath := filepath.Join(tmpDir, "user-policy.json")

	userPolicyJSON := `{
  "version": "1.0.0",
  "defaults": {
    "languages": ["javascript", "typescript"],
    "severity": "error",
    "autofix": true
  },
  "rules": [
    {
      "id": "DOC-001",
      "say": "주석에는 항상 순서 번호가 달려있어야 함",
      "category": "documentation",
      "languages": ["javascript"],
      "severity": "warning",
      "message": "Comments must include sequence numbers"
    },
    {
      "id": "SEC-001",
      "say": "환경변수를 사용해서 API 키를 관리해야 함",
      "category": "security",
      "languages": ["javascript", "typescript"],
      "severity": "error",
      "message": "Use environment variables for API keys"
    },
    {
      "id": "STYLE-001",
      "say": "함수는 camelCase를 사용해야 함",
      "category": "style",
      "languages": ["javascript", "typescript"]
    }
  ]
}`

	err := os.WriteFile(userPolicyPath, []byte(userPolicyJSON), 0644)
	require.NoError(t, err)

	// Create server
	server := &Server{
		configPath: userPolicyPath,
		loader:     policy.NewLoader(false),
	}

	// Load user policy
	userPolicy, err := server.loader.LoadUserPolicy(userPolicyPath)
	require.NoError(t, err)
	server.userPolicy = userPolicy

	t.Run("query all categories for javascript", func(t *testing.T) {
		params := map[string]interface{}{
			"category":  "all",
			"languages": []interface{}{"javascript"},
		}

		result, rpcErr := server.handleQueryConventions(params)
		require.Nil(t, rpcErr)
		require.NotNil(t, result)

		resultMap := result.(map[string]interface{})
		content := resultMap["content"].([]map[string]interface{})
		text := content[0]["text"].(string)

		t.Logf("Result: %s", text)

		// Should find conventions
		assert.NotContains(t, text, "No conventions found")
		assert.Contains(t, text, "DOC-001")
		assert.Contains(t, text, "SEC-001")
		assert.Contains(t, text, "STYLE-001")
	})

	t.Run("query documentation category for javascript", func(t *testing.T) {
		params := map[string]interface{}{
			"category":  "documentation",
			"languages": []interface{}{"javascript"},
		}

		result, rpcErr := server.handleQueryConventions(params)
		require.Nil(t, rpcErr)
		require.NotNil(t, result)

		resultMap := result.(map[string]interface{})
		content := resultMap["content"].([]map[string]interface{})
		text := content[0]["text"].(string)

		t.Logf("Result: %s", text)

		// Should find only documentation conventions
		assert.Contains(t, text, "DOC-001")
		assert.NotContains(t, text, "SEC-001")
		assert.NotContains(t, text, "STYLE-001")
	})

	t.Run("query security category for typescript", func(t *testing.T) {
		params := map[string]interface{}{
			"category":  "security",
			"languages": []interface{}{"typescript"},
		}

		result, rpcErr := server.handleQueryConventions(params)
		require.Nil(t, rpcErr)
		require.NotNil(t, result)

		resultMap := result.(map[string]interface{})
		content := resultMap["content"].([]map[string]interface{})
		text := content[0]["text"].(string)

		t.Logf("Result: %s", text)

		// Should find SEC-001 (supports typescript)
		assert.Contains(t, text, "SEC-001")
		assert.NotContains(t, text, "DOC-001") // javascript only
	})

	t.Run("query with unsupported language", func(t *testing.T) {
		params := map[string]interface{}{
			"category":  "all",
			"languages": []interface{}{"python"},
		}

		result, rpcErr := server.handleQueryConventions(params)
		require.Nil(t, rpcErr)
		require.NotNil(t, result)

		resultMap := result.(map[string]interface{})
		content := resultMap["content"].([]map[string]interface{})
		text := content[0]["text"].(string)

		t.Logf("Result: %s", text)

		// Should return no conventions
		assert.Contains(t, text, "No conventions found")
	})

	t.Run("rule without severity uses defaults", func(t *testing.T) {
		params := map[string]interface{}{
			"category":  "style",
			"languages": []interface{}{"javascript"},
		}

		result, rpcErr := server.handleQueryConventions(params)
		require.Nil(t, rpcErr)
		require.NotNil(t, result)

		resultMap := result.(map[string]interface{})
		content := resultMap["content"].([]map[string]interface{})
		text := content[0]["text"].(string)

		t.Logf("Result: %s", text)

		// STYLE-001 doesn't have explicit severity, should use default "error"
		assert.Contains(t, text, "STYLE-001")
		assert.Contains(t, text, "[error]") // Should use default from policy
	})

	t.Run("empty parameters returns all conventions", func(t *testing.T) {
		params := map[string]interface{}{}

		result, rpcErr := server.handleQueryConventions(params)
		require.Nil(t, rpcErr)
		require.NotNil(t, result)

		resultMap := result.(map[string]interface{})
		content := resultMap["content"].([]map[string]interface{})
		text := content[0]["text"].(string)

		t.Logf("Result: %s", text)

		// Should return all conventions when no filters specified
		assert.Contains(t, text, "DOC-001")
		assert.Contains(t, text, "SEC-001")
		assert.Contains(t, text, "STYLE-001")
	})

	t.Run("only category specified", func(t *testing.T) {
		params := map[string]interface{}{
			"category": "security",
		}

		result, rpcErr := server.handleQueryConventions(params)
		require.Nil(t, rpcErr)
		require.NotNil(t, result)

		resultMap := result.(map[string]interface{})
		content := resultMap["content"].([]map[string]interface{})
		text := content[0]["text"].(string)

		t.Logf("Result: %s", text)

		// Should return only security conventions
		assert.Contains(t, text, "SEC-001")
		assert.NotContains(t, text, "DOC-001")
	})
}

func TestAddCategory(t *testing.T) {
	t.Run("add category successfully", func(t *testing.T) {
		tmpDir := t.TempDir()

		// Change to temp dir first, before any git operations
		originalDir, _ := os.Getwd()
		require.NoError(t, os.Chdir(tmpDir))
		defer os.Chdir(originalDir)

		// Initialize real git repository and .sym directory
		require.NoError(t, runGitInit(tmpDir))
		require.NoError(t, os.MkdirAll(".sym", 0755))

		userPolicyPath := filepath.Join(tmpDir, ".sym", "user-policy.json")
		userPolicyJSON := `{
  "version": "1.0.0",
  "category": [
    {"name": "security", "description": "Security rules"}
  ],
  "rules": []
}`
		require.NoError(t, os.WriteFile(userPolicyPath, []byte(userPolicyJSON), 0644))

		server := &Server{
			configPath: userPolicyPath,
			loader:     policy.NewLoader(false),
		}
		userPolicy, err := server.loader.LoadUserPolicy(userPolicyPath)
		require.NoError(t, err)
		server.userPolicy = userPolicy

		result, rpcErr := server.handleAddCategory("testing", "Testing rules")
		require.Nil(t, rpcErr)
		require.NotNil(t, result)

		resultMap := result.(map[string]interface{})
		content := resultMap["content"].([]map[string]interface{})
		text := content[0]["text"].(string)

		assert.Contains(t, text, "added successfully")
		assert.Len(t, server.userPolicy.Category, 2)
		assert.Equal(t, "testing", server.userPolicy.Category[1].Name)
	})

	t.Run("reject duplicate category", func(t *testing.T) {
		server := &Server{
			loader: policy.NewLoader(false),
			userPolicy: &schema.UserPolicy{
				Version: "1.0.0",
				Category: []schema.CategoryDef{
					{Name: "security", Description: "Security rules"},
				},
			},
		}

		_, rpcErr := server.handleAddCategory("security", "Duplicate")
		require.NotNil(t, rpcErr)
		assert.Contains(t, rpcErr.Message, "already exists")
	})

	t.Run("reject empty name", func(t *testing.T) {
		server := &Server{
			loader:     policy.NewLoader(false),
			userPolicy: &schema.UserPolicy{Version: "1.0.0"},
		}

		_, rpcErr := server.handleAddCategory("", "Some description")
		require.NotNil(t, rpcErr)
		assert.Contains(t, rpcErr.Message, "name is required")
	})

	t.Run("reject empty description", func(t *testing.T) {
		server := &Server{
			loader:     policy.NewLoader(false),
			userPolicy: &schema.UserPolicy{Version: "1.0.0"},
		}

		_, rpcErr := server.handleAddCategory("testing", "")
		require.NotNil(t, rpcErr)
		assert.Contains(t, rpcErr.Message, "description is required")
	})
}

func TestEditCategory(t *testing.T) {
	t.Run("edit description only", func(t *testing.T) {
		tmpDir := t.TempDir()

		originalDir, _ := os.Getwd()
		require.NoError(t, os.Chdir(tmpDir))
		defer os.Chdir(originalDir)

		require.NoError(t, runGitInit(tmpDir))
		require.NoError(t, os.MkdirAll(".sym", 0755))

		userPolicyPath := filepath.Join(tmpDir, ".sym", "user-policy.json")
		userPolicyJSON := `{
  "version": "1.0.0",
  "category": [
    {"name": "security", "description": "Old description"}
  ],
  "rules": []
}`
		require.NoError(t, os.WriteFile(userPolicyPath, []byte(userPolicyJSON), 0644))

		server := &Server{
			configPath: userPolicyPath,
			loader:     policy.NewLoader(false),
		}
		userPolicy, err := server.loader.LoadUserPolicy(userPolicyPath)
		require.NoError(t, err)
		server.userPolicy = userPolicy

		result, rpcErr := server.handleEditCategory("security", "", "New description")
		require.Nil(t, rpcErr)
		require.NotNil(t, result)

		assert.Equal(t, "New description", server.userPolicy.Category[0].Description)
		assert.Equal(t, "security", server.userPolicy.Category[0].Name)
	})

	t.Run("rename category and update rules", func(t *testing.T) {
		tmpDir := t.TempDir()

		originalDir, _ := os.Getwd()
		require.NoError(t, os.Chdir(tmpDir))
		defer os.Chdir(originalDir)

		require.NoError(t, runGitInit(tmpDir))
		require.NoError(t, os.MkdirAll(".sym", 0755))

		userPolicyPath := filepath.Join(tmpDir, ".sym", "user-policy.json")
		userPolicyJSON := `{
  "version": "1.0.0",
  "category": [
    {"name": "old-name", "description": "Description"}
  ],
  "rules": [
    {"id": "R1", "say": "Rule 1", "category": "old-name"},
    {"id": "R2", "say": "Rule 2", "category": "old-name"},
    {"id": "R3", "say": "Rule 3", "category": "other"}
  ]
}`
		require.NoError(t, os.WriteFile(userPolicyPath, []byte(userPolicyJSON), 0644))

		server := &Server{
			configPath: userPolicyPath,
			loader:     policy.NewLoader(false),
		}
		userPolicy, err := server.loader.LoadUserPolicy(userPolicyPath)
		require.NoError(t, err)
		server.userPolicy = userPolicy

		result, rpcErr := server.handleEditCategory("old-name", "new-name", "")
		require.Nil(t, rpcErr)
		require.NotNil(t, result)

		resultMap := result.(map[string]interface{})
		content := resultMap["content"].([]map[string]interface{})
		text := content[0]["text"].(string)

		assert.Contains(t, text, "2 rule(s) updated")
		assert.Equal(t, "new-name", server.userPolicy.Category[0].Name)
		assert.Equal(t, "new-name", server.userPolicy.Rules[0].Category)
		assert.Equal(t, "new-name", server.userPolicy.Rules[1].Category)
		assert.Equal(t, "other", server.userPolicy.Rules[2].Category)
	})

	t.Run("reject rename to existing name", func(t *testing.T) {
		server := &Server{
			loader: policy.NewLoader(false),
			userPolicy: &schema.UserPolicy{
				Version: "1.0.0",
				Category: []schema.CategoryDef{
					{Name: "security", Description: "Security"},
					{Name: "style", Description: "Style"},
				},
			},
		}

		_, rpcErr := server.handleEditCategory("security", "style", "")
		require.NotNil(t, rpcErr)
		assert.Contains(t, rpcErr.Message, "already exists")
	})

	t.Run("reject not found category", func(t *testing.T) {
		server := &Server{
			loader: policy.NewLoader(false),
			userPolicy: &schema.UserPolicy{
				Version:  "1.0.0",
				Category: []schema.CategoryDef{},
			},
		}

		_, rpcErr := server.handleEditCategory("nonexistent", "new-name", "")
		require.NotNil(t, rpcErr)
		assert.Contains(t, rpcErr.Message, "not found")
	})

	t.Run("reject no changes provided", func(t *testing.T) {
		server := &Server{
			loader: policy.NewLoader(false),
			userPolicy: &schema.UserPolicy{
				Version: "1.0.0",
				Category: []schema.CategoryDef{
					{Name: "security", Description: "Security"},
				},
			},
		}

		_, rpcErr := server.handleEditCategory("security", "", "")
		require.NotNil(t, rpcErr)
		assert.Contains(t, rpcErr.Message, "At least one")
	})
}

func TestRemoveCategory(t *testing.T) {
	t.Run("remove unused category", func(t *testing.T) {
		tmpDir := t.TempDir()

		originalDir, _ := os.Getwd()
		require.NoError(t, os.Chdir(tmpDir))
		defer os.Chdir(originalDir)

		require.NoError(t, runGitInit(tmpDir))
		require.NoError(t, os.MkdirAll(".sym", 0755))

		userPolicyPath := filepath.Join(tmpDir, ".sym", "user-policy.json")
		userPolicyJSON := `{
  "version": "1.0.0",
  "category": [
    {"name": "security", "description": "Security"},
    {"name": "unused", "description": "Unused category"}
  ],
  "rules": [
    {"id": "R1", "say": "Rule 1", "category": "security"}
  ]
}`
		require.NoError(t, os.WriteFile(userPolicyPath, []byte(userPolicyJSON), 0644))

		server := &Server{
			configPath: userPolicyPath,
			loader:     policy.NewLoader(false),
		}
		userPolicy, err := server.loader.LoadUserPolicy(userPolicyPath)
		require.NoError(t, err)
		server.userPolicy = userPolicy

		result, rpcErr := server.handleRemoveCategory("unused")
		require.Nil(t, rpcErr)
		require.NotNil(t, result)

		resultMap := result.(map[string]interface{})
		content := resultMap["content"].([]map[string]interface{})
		text := content[0]["text"].(string)

		assert.Contains(t, text, "removed successfully")
		assert.Len(t, server.userPolicy.Category, 1)
		assert.Equal(t, "security", server.userPolicy.Category[0].Name)
	})

	t.Run("reject remove category with rules", func(t *testing.T) {
		server := &Server{
			loader: policy.NewLoader(false),
			userPolicy: &schema.UserPolicy{
				Version: "1.0.0",
				Category: []schema.CategoryDef{
					{Name: "security", Description: "Security"},
				},
				Rules: []schema.UserRule{
					{ID: "R1", Say: "Rule 1", Category: "security"},
					{ID: "R2", Say: "Rule 2", Category: "security"},
				},
			},
		}

		_, rpcErr := server.handleRemoveCategory("security")
		require.NotNil(t, rpcErr)
		assert.Contains(t, rpcErr.Message, "is used by 2 rule(s)")
	})

	t.Run("reject not found category", func(t *testing.T) {
		server := &Server{
			loader: policy.NewLoader(false),
			userPolicy: &schema.UserPolicy{
				Version:  "1.0.0",
				Category: []schema.CategoryDef{},
			},
		}

		_, rpcErr := server.handleRemoveCategory("nonexistent")
		require.NotNil(t, rpcErr)
		assert.Contains(t, rpcErr.Message, "not found")
	})
}

func TestListCategory(t *testing.T) {
	t.Run("returns no categories message when no user policy", func(t *testing.T) {
		server := &Server{
			loader: policy.NewLoader(false),
		}

		result, rpcErr := server.handleListCategory()
		require.Nil(t, rpcErr)
		require.NotNil(t, result)

		resultMap := result.(map[string]interface{})
		content := resultMap["content"].([]map[string]interface{})
		text := content[0]["text"].(string)

		t.Logf("Result: %s", text)

		// Should show no categories message (categories are now only from user-policy.json)
		assert.Contains(t, text, "No categories defined in user-policy.json")
		assert.Contains(t, text, "Run 'sym init' to create default categories")
	})

	t.Run("returns only user-defined categories from user-policy.json", func(t *testing.T) {
		// Setup: Create a temporary user policy with custom categories
		tmpDir := t.TempDir()
		userPolicyPath := filepath.Join(tmpDir, "user-policy.json")

		userPolicyJSON := `{
  "version": "1.0.0",
  "category": [
    {"name": "security", "description": "Custom security description"},
    {"name": "naming", "description": "Naming convention rules"}
  ],
  "rules": []
}`

		err := os.WriteFile(userPolicyPath, []byte(userPolicyJSON), 0644)
		require.NoError(t, err)

		server := &Server{
			configPath: userPolicyPath,
			loader:     policy.NewLoader(false),
		}

		// Load user policy
		userPolicy, err := server.loader.LoadUserPolicy(userPolicyPath)
		require.NoError(t, err)
		server.userPolicy = userPolicy

		result, rpcErr := server.handleListCategory()
		require.Nil(t, rpcErr)
		require.NotNil(t, result)

		resultMap := result.(map[string]interface{})
		content := resultMap["content"].([]map[string]interface{})
		text := content[0]["text"].(string)

		t.Logf("Result: %s", text)

		// Should include user-defined categories only (no merging with defaults)
		assert.Contains(t, text, "security")
		assert.Contains(t, text, "Custom security description")
		assert.Contains(t, text, "naming")
		assert.Contains(t, text, "Naming convention rules")
		// Should have only 2 categories (user-defined only)
		assert.Contains(t, text, "Available categories (2)")
	})
}
