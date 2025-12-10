package mcp

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/DevSymphony/sym-cli/internal/policy"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

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
