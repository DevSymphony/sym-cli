package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/DevSymphony/sym-cli/internal/converter"
	"github.com/DevSymphony/sym-cli/internal/importer"
	"github.com/DevSymphony/sym-cli/internal/llm"
	"github.com/DevSymphony/sym-cli/internal/policy"
	"github.com/DevSymphony/sym-cli/internal/roles"
	"github.com/DevSymphony/sym-cli/internal/util/config"
	"github.com/DevSymphony/sym-cli/internal/util/git"
	"github.com/DevSymphony/sym-cli/internal/validator"
	"github.com/DevSymphony/sym-cli/pkg/schema"
	sdkmcp "github.com/modelcontextprotocol/go-sdk/mcp"
)

// ConvertPolicyWithLLM converts user policy to code policy using LLM.
// This is extracted from cmd/mcp.go's autoConvertPolicy for reuse.
func ConvertPolicyWithLLM(userPolicyPath, codePolicyPath string) error {
	// Load user policy
	data, err := os.ReadFile(userPolicyPath)
	if err != nil {
		return fmt.Errorf("failed to read user policy: %w", err)
	}

	var userPolicy schema.UserPolicy
	if err := json.Unmarshal(data, &userPolicy); err != nil {
		return fmt.Errorf("failed to parse user policy: %w", err)
	}

	// Setup LLM provider
	cfg := llm.LoadConfig()
	llmProvider, err := llm.New(cfg)
	if err != nil {
		return fmt.Errorf("failed to create LLM provider: %w", err)
	}
	defer func() { _ = llmProvider.Close() }()

	// Create converter with output directory
	outputDir := filepath.Dir(codePolicyPath)
	conv := converter.NewConverter(llmProvider, outputDir)

	// Setup context with timeout (10 minutes to match validator)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	fmt.Fprintf(os.Stderr, "Converting %d rules...\n", len(userPolicy.Rules))

	// Convert using new API
	result, err := conv.Convert(ctx, &userPolicy)
	if err != nil {
		return fmt.Errorf("conversion failed: %w", err)
	}

	// Files are already written by converter
	for _, filePath := range result.GeneratedFiles {
		fmt.Fprintf(os.Stderr, "  ✓ Generated: %s\n", filePath)
	}

	// Report any errors
	if len(result.Errors) > 0 {
		for linter, err := range result.Errors {
			fmt.Fprintf(os.Stderr, "  ✗ %s: %v\n", linter, err)
		}
	}

	return nil
}

// Server is a MCP (Model Context Protocol) server.
// It communicates via JSON-RPC over stdio.
type Server struct {
	configPath string
	userPolicy *schema.UserPolicy
	codePolicy *schema.CodePolicy
	loader     *policy.Loader
}

// NewServer creates a new MCP server instance.
func NewServer(configPath string) *Server {
	return &Server{
		configPath: configPath,
		loader:     policy.NewLoader(false), // verbose = false for MCP
	}
}

// Start starts the MCP server.
// It communicates via JSON-RPC over stdio.
func (s *Server) Start() error {
	// Determine the directory to look for policy files
	var dir string

	if s.configPath != "" {
		// If configPath is provided, check if it's a directory or file
		fileInfo, err := os.Stat(s.configPath)
		if err == nil && fileInfo.IsDir() {
			// If it's a directory, use it directly
			dir = s.configPath
		} else {
			// If it's a file, use its parent directory
			dir = filepath.Dir(s.configPath)
		}
	} else {
		// No configPath provided, auto-detect .sym folder from git root
		repoRoot, err := git.GetRepoRoot()
		if err != nil {
			fmt.Fprintf(os.Stderr, "⚠️  Warning: Not in a git repository, MCP server starting without policies\n")
		} else {
			dir = filepath.Join(repoRoot, ".sym")
			// Change working directory to project root for git operations
			if err := os.Chdir(repoRoot); err != nil {
				fmt.Fprintf(os.Stderr, "⚠️  Warning: Failed to change to project root: %v\n", err)
			} else {
				fmt.Fprintf(os.Stderr, "✓ Working directory set to project root: %s\n", repoRoot)
			}
		}
	}

	// Only try to load policies if we have a directory
	if dir != "" {
		// Try to load user-policy.json for natural language descriptions
		// Get policy path from config.json
		projectCfg, _ := config.LoadProjectConfig()
		userPolicyPath := projectCfg.PolicyPath
		if userPolicyPath == "" {
			userPolicyPath = filepath.Join(dir, "user-policy.json")
		} else if !filepath.IsAbs(userPolicyPath) {
			// Make relative path absolute based on repo root
			if repoRoot, err := git.GetRepoRoot(); err == nil {
				userPolicyPath = filepath.Join(repoRoot, userPolicyPath)
			}
		}

		if userPolicy, err := s.loader.LoadUserPolicy(userPolicyPath); err == nil {
			s.userPolicy = userPolicy
			fmt.Fprintf(os.Stderr, "✓ User policy loaded: %s (%d rules)\n", userPolicyPath, len(userPolicy.Rules))
		}

		// Try to load code-policy.json for validation (in same directory as user policy)
		codePolicyPath := filepath.Join(filepath.Dir(userPolicyPath), "code-policy.json")
		if codePolicy, err := s.loader.LoadCodePolicy(codePolicyPath); err == nil {
			s.codePolicy = codePolicy
			fmt.Fprintf(os.Stderr, "✓ Code policy loaded: %s (%d rules)\n", codePolicyPath, len(codePolicy.Rules))
		}

		// Check if conversion is needed
		if s.userPolicy != nil {
			needsConversion := s.needsConversion(codePolicyPath)
			if needsConversion {
				fmt.Fprintf(os.Stderr, "⚙️  User policy has been updated. Converting to code policy...\n")
				if err := s.convertUserPolicy(userPolicyPath, codePolicyPath); err != nil {
					fmt.Fprintf(os.Stderr, "⚠️  Warning: Failed to convert policy: %v\n", err)
					fmt.Fprintf(os.Stderr, "   Continuing with existing policies...\n")
				} else {
					// Reload code policy after conversion
					if codePolicy, err := s.loader.LoadCodePolicy(codePolicyPath); err == nil {
						s.codePolicy = codePolicy
						fmt.Fprintf(os.Stderr, "✓ Code policy updated: %s (%d rules)\n", codePolicyPath, len(codePolicy.Rules))
					}
				}
			}
		}

		// At least one policy must be loaded
		if s.userPolicy == nil && s.codePolicy == nil {
			return fmt.Errorf("no policy found in %s", dir)
		}
	}

	fmt.Fprintln(os.Stderr, "Symphony MCP server started (stdio mode)")
	fmt.Fprintln(os.Stderr, "Available tools: list_convention, add_convention, edit_convention, remove_convention, validate_code, list_category, add_category, edit_category, remove_category, import_convention, convert")

	// Use official MCP go-sdk for stdio to ensure spec-compliant framing and lifecycle
	return s.runStdioWithSDK(context.Background())
}

// RPCError is an error type used for internal error handling.
type RPCError struct {
	Code    int
	Message string
}

// ListConventionInput represents the input schema for the list_convention tool (go-sdk).
type ListConventionInput struct {
	Categories []string `json:"categories,omitempty" jsonschema:"Filter by categories (optional). Leave empty or use [\"all\"] to fetch all categories. Example: [\"security\", \"style\"]"`
	Languages  []string `json:"languages,omitempty" jsonschema:"Programming languages to filter by (optional). Leave empty to get conventions for all languages. Examples: go, javascript, typescript, python, java"`
}

// ValidateCodeInput represents the input schema for the validate_code tool (go-sdk).
type ValidateCodeInput struct {
	Role string `json:"role,omitempty" jsonschema:"RBAC role for validation (optional)"`
}

// ListCategoryInput represents the input schema for the list_category tool (go-sdk).
type ListCategoryInput struct {
	// No parameters - returns all categories
}

// CategoryItem represents a single category for batch operations.
type CategoryItem struct {
	Name        string `json:"name" jsonschema:"Category name"`
	Description string `json:"description" jsonschema:"Category description"`
}

// CategoryEditItem represents a single category edit for batch operations.
type CategoryEditItem struct {
	Name        string `json:"name" jsonschema:"Current category name"`
	NewName     string `json:"new_name,omitempty" jsonschema:"New name (optional)"`
	Description string `json:"description,omitempty" jsonschema:"New description (optional)"`
}

// FailedItem represents a failed operation in batch processing.
type FailedItem struct {
	Name   string `json:"name"`
	Reason string `json:"reason"`
}

// AddCategoryInput represents the input schema for the add_category tool (batch mode).
type AddCategoryInput struct {
	Categories []CategoryItem `json:"categories" jsonschema:"Array of categories to add"`
}

// EditCategoryInput represents the input schema for the edit_category tool (batch mode).
type EditCategoryInput struct {
	Edits []CategoryEditItem `json:"edits" jsonschema:"Array of category edits"`
}

// RemoveCategoryInput represents the input schema for the remove_category tool (batch mode).
type RemoveCategoryInput struct {
	Names []string `json:"names" jsonschema:"Array of category names to remove"`
}

// ImportConventionsInput represents the input schema for the import_convention tool.
type ImportConventionsInput struct {
	Path string `json:"path" jsonschema:"File path to import conventions from"`
	Mode string `json:"mode,omitempty" jsonschema:"Import mode: 'append' (default) keeps existing, 'clear' removes existing first"`
}

// ConventionInput represents a single convention for add operations.
type ConventionInput struct {
	ID        string   `json:"id" jsonschema:"Rule ID (required)"`
	Say       string   `json:"say" jsonschema:"Rule description in natural language (required)"`
	Category  string   `json:"category,omitempty" jsonschema:"Category name"`
	Languages []string `json:"languages,omitempty" jsonschema:"Programming languages"`
	Severity  string   `json:"severity,omitempty" jsonschema:"error, warning, or info"`
	Autofix   bool     `json:"autofix,omitempty" jsonschema:"Enable auto-fix"`
	Message   string   `json:"message,omitempty" jsonschema:"Message to display on violation"`
	Example   string   `json:"example,omitempty" jsonschema:"Code example"`
	Include   []string `json:"include,omitempty" jsonschema:"File patterns to include"`
	Exclude   []string `json:"exclude,omitempty" jsonschema:"File patterns to exclude"`
}

// ConventionEditInput represents a single convention edit for batch operations.
type ConventionEditInput struct {
	ID        string   `json:"id" jsonschema:"Current rule ID (required)"`
	NewID     string   `json:"new_id,omitempty" jsonschema:"New rule ID"`
	Say       string   `json:"say,omitempty" jsonschema:"New description"`
	Category  string   `json:"category,omitempty" jsonschema:"New category name"`
	Languages []string `json:"languages,omitempty" jsonschema:"New programming languages"`
	Severity  string   `json:"severity,omitempty" jsonschema:"New severity level"`
	Autofix   *bool    `json:"autofix,omitempty" jsonschema:"Enable auto-fix"`
	Message   string   `json:"message,omitempty" jsonschema:"New message"`
	Example   string   `json:"example,omitempty" jsonschema:"New code example"`
	Include   []string `json:"include,omitempty" jsonschema:"New file patterns to include"`
	Exclude   []string `json:"exclude,omitempty" jsonschema:"New file patterns to exclude"`
}

// AddConventionInput represents the input schema for the add_convention tool (batch mode).
type AddConventionInput struct {
	Conventions []ConventionInput `json:"conventions" jsonschema:"Array of conventions to add"`
}

// EditConventionInput represents the input schema for the edit_convention tool (batch mode).
type EditConventionInput struct {
	Edits []ConventionEditInput `json:"edits" jsonschema:"Array of convention edits"`
}

// RemoveConventionInput represents the input schema for the remove_convention tool (batch mode).
type RemoveConventionInput struct {
	IDs []string `json:"ids" jsonschema:"Array of convention IDs to remove"`
}

// ConvertPolicyInput represents the input schema for the convert tool.
type ConvertPolicyInput struct {
	InputPath string `json:"input_path,omitempty" jsonschema:"Path to user-policy.json file (optional, defaults to config or .sym/user-policy.json)"`
	OutputDir string `json:"output_dir,omitempty" jsonschema:"Output directory for generated configs (optional, defaults to .sym)"`
}

// runStdioWithSDK runs a spec-compliant MCP server over stdio using the official go-sdk.
func (s *Server) runStdioWithSDK(ctx context.Context) error {
	server := sdkmcp.NewServer(&sdkmcp.Implementation{
		Name:    "symphony",
		Version: "1.0.0",
	}, nil)

	// Tool: list_convention
	sdkmcp.AddTool(server, &sdkmcp.Tool{
		Name:        "list_convention",
		Description: "[MANDATORY BEFORE CODING] List project conventions BEFORE writing any code to ensure compliance from the start. Filter by categories or languages.",
	}, func(ctx context.Context, req *sdkmcp.CallToolRequest, input ListConventionInput) (*sdkmcp.CallToolResult, map[string]any, error) {
		params := map[string]any{
			"categories": input.Categories,
			"languages":  input.Languages,
		}
		result, rpcErr := s.handleListConvention(params)
		if rpcErr != nil {
			return &sdkmcp.CallToolResult{IsError: true}, nil, fmt.Errorf("%s", rpcErr.Message)
		}
		// result is already MCP-shaped: { content: [{type:"text", text:"..."}] }
		return nil, result.(map[string]any), nil
	})

	// Tool: validate_code
	sdkmcp.AddTool(server, &sdkmcp.Tool{
		Name:        "validate_code",
		Description: "Validate git changes (staged + unstaged) against all project conventions.",
	}, func(ctx context.Context, req *sdkmcp.CallToolRequest, input ValidateCodeInput) (*sdkmcp.CallToolResult, map[string]any, error) {
		params := map[string]any{
			"role": input.Role,
		}
		result, rpcErr := s.handleValidateCode(ctx, req.Session, params)
		if rpcErr != nil {
			return &sdkmcp.CallToolResult{IsError: true}, nil, fmt.Errorf("%s", rpcErr.Message)
		}
		return nil, result.(map[string]any), nil
	})

	// Tool: list_category
	sdkmcp.AddTool(server, &sdkmcp.Tool{
		Name:        "list_category",
		Description: "List all available convention categories with descriptions. Use this to discover what categories exist before querying conventions.",
	}, func(ctx context.Context, req *sdkmcp.CallToolRequest, input ListCategoryInput) (*sdkmcp.CallToolResult, map[string]any, error) {
		result, rpcErr := s.handleListCategory()
		if rpcErr != nil {
			return &sdkmcp.CallToolResult{IsError: true}, nil, fmt.Errorf("%s", rpcErr.Message)
		}
		return nil, result.(map[string]any), nil
	})

	// Tool: add_category (batch mode)
	sdkmcp.AddTool(server, &sdkmcp.Tool{
		Name:        "add_category",
		Description: "Add convention categories. Pass array of {name, description} objects in 'categories' field.",
	}, func(ctx context.Context, req *sdkmcp.CallToolRequest, input AddCategoryInput) (*sdkmcp.CallToolResult, map[string]any, error) {
		result, rpcErr := s.handleAddCategory(input)
		if rpcErr != nil {
			return &sdkmcp.CallToolResult{IsError: true}, nil, fmt.Errorf("%s", rpcErr.Message)
		}
		return nil, result.(map[string]any), nil
	})

	// Tool: edit_category (batch mode)
	sdkmcp.AddTool(server, &sdkmcp.Tool{
		Name:        "edit_category",
		Description: "Edit convention categories. Pass array of {name, new_name?, description?} objects in 'edits' field.",
	}, func(ctx context.Context, req *sdkmcp.CallToolRequest, input EditCategoryInput) (*sdkmcp.CallToolResult, map[string]any, error) {
		result, rpcErr := s.handleEditCategory(input)
		if rpcErr != nil {
			return &sdkmcp.CallToolResult{IsError: true}, nil, fmt.Errorf("%s", rpcErr.Message)
		}
		return nil, result.(map[string]any), nil
	})

	// Tool: remove_category (batch mode)
	sdkmcp.AddTool(server, &sdkmcp.Tool{
		Name:        "remove_category",
		Description: "Remove convention categories. Pass array of category names in 'names' field.",
	}, func(ctx context.Context, req *sdkmcp.CallToolRequest, input RemoveCategoryInput) (*sdkmcp.CallToolResult, map[string]any, error) {
		result, rpcErr := s.handleRemoveCategory(input)
		if rpcErr != nil {
			return &sdkmcp.CallToolResult{IsError: true}, nil, fmt.Errorf("%s", rpcErr.Message)
		}
		return nil, result.(map[string]any), nil
	})

	// Tool: import_convention
	sdkmcp.AddTool(server, &sdkmcp.Tool{
		Name:        "import_convention",
		Description: "Import coding conventions from external documents (txt, md, code files) into user-policy.json. Uses LLM to extract categories and rules from document content. After importing, run 'convert' to generate linter configurations.",
	}, func(ctx context.Context, req *sdkmcp.CallToolRequest, input ImportConventionsInput) (*sdkmcp.CallToolResult, map[string]any, error) {
		result, rpcErr := s.handleImportConventions(ctx, input)
		if rpcErr != nil {
			return &sdkmcp.CallToolResult{IsError: true}, nil, fmt.Errorf("%s", rpcErr.Message)
		}
		return nil, result.(map[string]any), nil
	})

	// Tool: add_convention (batch mode)
	sdkmcp.AddTool(server, &sdkmcp.Tool{
		Name:        "add_convention",
		Description: "Add conventions (rules). Pass array of {id, say, category?, languages?, severity?, autofix?, message?, example?, include?, exclude?} objects in 'conventions' field. After adding rules, run 'convert' to generate linter configurations.",
	}, func(ctx context.Context, req *sdkmcp.CallToolRequest, input AddConventionInput) (*sdkmcp.CallToolResult, map[string]any, error) {
		result, rpcErr := s.handleAddConvention(input)
		if rpcErr != nil {
			return &sdkmcp.CallToolResult{IsError: true}, nil, fmt.Errorf("%s", rpcErr.Message)
		}
		return nil, result.(map[string]any), nil
	})

	// Tool: edit_convention (batch mode)
	sdkmcp.AddTool(server, &sdkmcp.Tool{
		Name:        "edit_convention",
		Description: "Edit conventions (rules). Pass array of {id, new_id?, say?, category?, languages?, severity?, autofix?, message?, example?, include?, exclude?} objects in 'edits' field. After editing rules, run 'convert' to regenerate linter configurations.",
	}, func(ctx context.Context, req *sdkmcp.CallToolRequest, input EditConventionInput) (*sdkmcp.CallToolResult, map[string]any, error) {
		result, rpcErr := s.handleEditConvention(input)
		if rpcErr != nil {
			return &sdkmcp.CallToolResult{IsError: true}, nil, fmt.Errorf("%s", rpcErr.Message)
		}
		return nil, result.(map[string]any), nil
	})

	// Tool: remove_convention (batch mode)
	sdkmcp.AddTool(server, &sdkmcp.Tool{
		Name:        "remove_convention",
		Description: "Remove conventions (rules). Pass array of convention IDs in 'ids' field. After removing rules, run 'convert' to regenerate linter configurations.",
	}, func(ctx context.Context, req *sdkmcp.CallToolRequest, input RemoveConventionInput) (*sdkmcp.CallToolResult, map[string]any, error) {
		result, rpcErr := s.handleRemoveConvention(input)
		if rpcErr != nil {
			return &sdkmcp.CallToolResult{IsError: true}, nil, fmt.Errorf("%s", rpcErr.Message)
		}
		return nil, result.(map[string]any), nil
	})

	// Tool: convert
	sdkmcp.AddTool(server, &sdkmcp.Tool{
		Name:        "convert",
		Description: "Convert user-policy.json (natural language rules) into linter-specific configurations and code-policy.json. Uses LLM to route rules to appropriate linters (ESLint, Prettier, Pylint, TSC, Checkstyle, PMD, golangci-lint).",
	}, func(ctx context.Context, req *sdkmcp.CallToolRequest, input ConvertPolicyInput) (*sdkmcp.CallToolResult, map[string]any, error) {
		result, rpcErr := s.handleConvertPolicy(ctx, input)
		if rpcErr != nil {
			return &sdkmcp.CallToolResult{IsError: true}, nil, fmt.Errorf("%s", rpcErr.Message)
		}
		return nil, result.(map[string]any), nil
	})

	// Run the server over stdio until the client disconnects
	return server.Run(ctx, &sdkmcp.StdioTransport{})
}

// QueryConventionsRequest is a request to query conventions.
type QueryConventionsRequest struct {
	Categories []string `json:"categories"` // optional; use ["all"] or empty to fetch all categories
	Languages  []string `json:"languages"`  // optional; empty means all languages
}

// ConventionItem is a convention item.
type ConventionItem struct {
	ID          string `json:"id"`
	Category    string `json:"category"`
	Description string `json:"description"`
	Message     string `json:"message"`
	Severity    string `json:"severity"`
}

// handleListConvention handles convention list/query requests.
// It finds and returns relevant conventions by category.
func (s *Server) handleListConvention(params map[string]interface{}) (interface{}, *RPCError) {
	if s.userPolicy == nil && s.codePolicy == nil {
		return map[string]interface{}{
			"conventions": []ConventionItem{},
			"message":     "policy not loaded",
		}, nil
	}

	var req QueryConventionsRequest
	paramBytes, _ := json.Marshal(params)
	if err := json.Unmarshal(paramBytes, &req); err != nil {
		return nil, &RPCError{
			Code:    -32602,
			Message: fmt.Sprintf("failed to parse parameters: %v", err),
		}
	}

	// Apply defaults for missing parameters
	// If categories contains "all" or is empty, return all categories
	req.Categories = normalizeCategories(req.Categories)

	// If languages is empty, return all languages
	// This is more user-friendly than requiring the parameter

	conventions := s.filterConventions(req)

	// Format conventions as readable text for MCP response
	var textContent string
	if len(conventions) == 0 {
		textContent = "No conventions found for the specified criteria."
	} else {
		textContent = fmt.Sprintf("Found %d convention(s):\n\n", len(conventions))
		for i, conv := range conventions {
			textContent += fmt.Sprintf("%d. [%s] %s\n", i+1, conv.Severity, conv.ID)
			textContent += fmt.Sprintf("   Category: %s\n", conv.Category)
			textContent += fmt.Sprintf("   Description: %s\n", conv.Description)
			if conv.Message != "" && conv.Message != conv.Description {
				textContent += fmt.Sprintf("   Message: %s\n", conv.Message)
			}
			textContent += "\n"
		}
	}

	// Add RBAC information if available
	rbacInfo := s.getRBACInfo()
	if rbacInfo != "" {
		textContent += "\n\n" + rbacInfo
	}

	textContent += "\n✓ Next Step: Implement your code following these conventions. After completion, MUST call validate_code to verify compliance."

	// Return MCP-compliant response with content array
	return map[string]interface{}{
		"content": []map[string]interface{}{
			{
				"type": "text",
				"text": textContent,
			},
		},
	}, nil
}

// filterConventions filters conventions that match the request.
func (s *Server) filterConventions(req QueryConventionsRequest) []ConventionItem {
	var conventions []ConventionItem

	// If UserPolicy is loaded, use natural language rules
	if s.userPolicy != nil {
		for _, rule := range s.userPolicy.Rules {
			// Check category filter
			if !matchesCategories(rule.Category, req.Categories) {
				continue
			}

			// Check language relevance
			// Only filter by language if both req.Languages and rule.Languages are specified
			if len(req.Languages) > 0 && len(rule.Languages) > 0 {
				if !containsAny(rule.Languages, req.Languages) {
					continue
				}
			}
			// If req.Languages is empty, include all rules (more user-friendly)

			severity := rule.Severity
			if severity == "" && s.userPolicy.Defaults.Severity != "" {
				severity = s.userPolicy.Defaults.Severity
			}
			if severity == "" {
				severity = "warning" // fallback default
			}

			message := rule.Message
			if message == "" {
				message = rule.Say // Use description as message if no explicit message
			}

			conventions = append(conventions, ConventionItem{
				ID:          rule.ID,
				Category:    rule.Category,
				Description: rule.Say, // Use natural language description
				Message:     message,
				Severity:    severity,
			})
		}
		return conventions
	}

	// Fallback to CodePolicy if UserPolicy not available
	if s.codePolicy != nil {
		for _, rule := range s.codePolicy.Rules {
			if !rule.Enabled {
				continue
			}

			// Check category filter (supports multiple categories)
			if !matchesCategories(rule.Category, req.Categories) {
				continue
			}

			if !s.isRuleRelevant(rule, req) {
				continue
			}

			conventions = append(conventions, ConventionItem{
				ID:          rule.ID,
				Category:    rule.Category,
				Description: rule.Desc,
				Message:     rule.Message,
				Severity:    rule.Severity,
			})
		}
	}

	return conventions
}

// isRuleRelevant checks if a rule is relevant to the request.
func (s *Server) isRuleRelevant(rule schema.PolicyRule, req QueryConventionsRequest) bool {
	if rule.When == nil {
		return true
	}

	if len(rule.When.Languages) > 0 && len(req.Languages) > 0 {
		if !containsAny(rule.When.Languages, req.Languages) {
			return false
		}
	}
	return true
}

// ValidateCodeRequest is a code validation request.
type ValidateCodeRequest struct {
	Role string `json:"role"` // RBAC role (optional)
}

// ViolationItem is a violation item.
type ViolationItem struct {
	RuleID   string `json:"rule_id"`
	Category string `json:"category"`
	Message  string `json:"message"`
	Severity string `json:"severity"`
	File     string `json:"file,omitempty"`
	Line     int    `json:"line,omitempty"`
	Column   int    `json:"column,omitempty"`
}

// handleValidateCode handles code validation requests.
// It validates git changes (diff) instead of entire files for efficiency.
func (s *Server) handleValidateCode(ctx context.Context, session *sdkmcp.ServerSession, params map[string]interface{}) (interface{}, *RPCError) {
	// Get policy for validation (convert UserPolicy if needed)
	validationPolicy, err := s.getValidationPolicy()
	if err != nil {
		return nil, &RPCError{
			Code:    -32000,
			Message: fmt.Sprintf("policy not available: %v", err),
		}
	}

	var req ValidateCodeRequest
	paramBytes, _ := json.Marshal(params)
	if err := json.Unmarshal(paramBytes, &req); err != nil {
		return nil, &RPCError{
			Code:    -32602,
			Message: fmt.Sprintf("failed to parse parameters: %v", err),
		}
	}

	var allViolations []ViolationItem
	var hasErrors bool

	// Get all git changes (staged + unstaged + untracked)
	// GetChanges already includes all types of changes
	changes, err := git.GetChanges()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to get git changes: %v\n", err)
		return map[string]interface{}{
			"content": []map[string]interface{}{
				{
					"type": "text",
					"text": "⚠️  Failed to get git changes. Make sure you're in a git repository.\n\nError: " + err.Error(),
				},
			},
			"isError": false,
		}, nil
	}

	if len(changes) == 0 {
		return map[string]interface{}{
			"content": []map[string]interface{}{
				{
					"type": "text",
					"text": "✓ No uncommitted changes to validate. Working directory is clean.",
				},
			},
			"isError": false,
		}, nil
	}

	// Use configured LLM provider
	llmCfg := llm.LoadConfig()
	llmProvider, err := llm.New(llmCfg)
	if err != nil {
		return nil, &RPCError{
			Code:    -32000,
			Message: fmt.Sprintf("failed to create LLM provider: %v", err),
		}
	}
	fmt.Fprintf(os.Stderr, "✓ Using LLM provider: %s\n", llmProvider.Name())

	// Create unified validator that handles all engines + RBAC
	v := validator.NewValidator(validationPolicy, false) // verbose=false for MCP
	v.SetLLMProvider(llmProvider)
	defer func() {
		_ = v.Close() // Ignore close error in MCP context
	}()

	// Validate git changes using unified validator
	result, err := v.ValidateChanges(ctx, changes)
	if err != nil {
		return nil, &RPCError{
			Code:    -32000,
			Message: fmt.Sprintf("validation failed: %v", err),
		}
	}

	// Convert violations
	for _, violation := range result.Violations {
		allViolations = append(allViolations, ViolationItem{
			RuleID:   violation.RuleID,
			Category: "",
			Message:  violation.Message,
			Severity: violation.Severity,
			File:     violation.File,
			Line:     violation.Line,
			Column:   violation.Column,
		})

		if violation.Severity == "error" {
			hasErrors = true
		}
	}

	// Save validation results to .sym/validation-results.json
	if err := s.saveValidationResults(result, allViolations, hasErrors); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to save validation results: %v\n", err)
	}

	// Format validation results as readable text for MCP response
	var textContent string
	if hasErrors {
		textContent = "VALIDATION FAILED: Found error-level violations. You MUST fix these issues and re-validate before proceeding.\n\n"
	} else if len(allViolations) > 0 {
		textContent = "VALIDATION WARNING: Found non-critical violations. Consider fixing these warnings for better code quality.\n\n"
	} else {
		textContent = "VALIDATION PASSED: Code complies with all conventions. Task can be marked as complete.\n\n"
	}

	if len(allViolations) > 0 {
		textContent += fmt.Sprintf("Total violations: %d\n\n", len(allViolations))
		for i, violation := range allViolations {
			textContent += fmt.Sprintf("%d. [%s] %s\n", i+1, violation.Severity, violation.RuleID)
			if violation.File != "" {
				textContent += fmt.Sprintf("   File: %s", violation.File)
				if violation.Line > 0 {
					textContent += fmt.Sprintf(":%d", violation.Line)
					if violation.Column > 0 {
						textContent += fmt.Sprintf(":%d", violation.Column)
					}
				}
				textContent += "\n"
			}
			textContent += fmt.Sprintf("   Message: %s\n\n", violation.Message)
		}
	}

	// Add engine errors if any (adapter execution failures)
	if len(result.Errors) > 0 {
		textContent += fmt.Sprintf("\n⚠️  Engine errors (%d):\n", len(result.Errors))
		for _, e := range result.Errors {
			textContent += fmt.Sprintf("   [%s] %s: %s\n", e.Engine, e.RuleID, e.Message)
		}
		textContent += "\n"
	}

	// Add note about saved results
	textContent += "\n💾 Validation results saved to .sym/validation-results.json\n"

	// Return MCP-compliant response with content array
	return map[string]interface{}{
		"content": []map[string]interface{}{
			{
				"type": "text",
				"text": textContent,
			},
		},
		"isError": hasErrors,
	}, nil
}

// containsAny checks if haystack contains any of the needles.
func containsAny(haystack, needles []string) bool {
	for _, needle := range needles {
		for _, hay := range haystack {
			if hay == needle {
				return true
			}
		}
	}
	return false
}

// normalizeCategories normalizes the categories input.
// Returns nil if categories is empty or contains "all" (meaning fetch all).
func normalizeCategories(categories []string) []string {
	if len(categories) == 0 {
		return nil
	}
	// Check if any category is "all"
	for _, cat := range categories {
		if strings.EqualFold(strings.TrimSpace(cat), "all") {
			return nil
		}
	}
	// Trim whitespace from each category
	result := make([]string, 0, len(categories))
	for _, cat := range categories {
		trimmed := strings.TrimSpace(cat)
		if trimmed != "" {
			result = append(result, trimmed)
		}
	}
	if len(result) == 0 {
		return nil
	}
	return result
}

// matchesCategories checks if the given category matches any of the requested categories.
// Returns true if categories is nil/empty (meaning all categories match).
func matchesCategories(ruleCategory string, requestedCategories []string) bool {
	if len(requestedCategories) == 0 {
		return true
	}
	for _, cat := range requestedCategories {
		if ruleCategory == cat {
			return true
		}
	}
	return false
}

// getValidationPolicy returns CodePolicy for validation.
func (s *Server) getValidationPolicy() (*schema.CodePolicy, error) {
	if s.codePolicy != nil {
		return s.codePolicy, nil
	}
	return nil, fmt.Errorf("no code policy loaded - validation requires code policy")
}

// needsConversion checks if user policy needs to be converted to code policy.
// Returns true if:
// 1. code-policy.json doesn't exist, OR
// 2. user policy has rule IDs that don't exist in code policy (after extracting source ID)
func (s *Server) needsConversion(codePolicyPath string) bool {
	// If no code policy exists, conversion is needed
	if s.codePolicy == nil {
		return true
	}

	// If no user policy, no conversion needed
	if s.userPolicy == nil {
		return false
	}

	// Extract source rule IDs from code policy
	// code-policy rules have IDs like "FMT-001-eslint", we extract "FMT-001"
	codePolicySourceIDs := make(map[string]bool)
	for _, rule := range s.codePolicy.Rules {
		sourceID := extractSourceRuleID(rule.ID)
		codePolicySourceIDs[sourceID] = true
	}

	// Check if all user policy rule IDs have corresponding code policy rules
	for _, userRule := range s.userPolicy.Rules {
		if !codePolicySourceIDs[userRule.ID] {
			// Found a user rule that doesn't exist in code policy
			return true
		}
	}

	return false
}

// extractSourceRuleID extracts the original user-policy rule ID from a code-policy rule ID.
// For example: "FMT-001-eslint" -> "FMT-001"
func extractSourceRuleID(codePolicyRuleID string) string {
	// Known linter suffixes that are appended during conversion (see converter.go:179)
	linterSuffixes := []string{"-eslint", "-prettier", "-tsc", "-pylint", "-checkstyle", "-pmd", "-llm-validator"}
	for _, suffix := range linterSuffixes {
		if strings.HasSuffix(codePolicyRuleID, suffix) {
			return strings.TrimSuffix(codePolicyRuleID, suffix)
		}
	}
	return codePolicyRuleID
}

// convertUserPolicy converts user policy to code policy using LLM.
// This is a wrapper that calls the shared conversion logic.
func (s *Server) convertUserPolicy(userPolicyPath, codePolicyPath string) error {
	return ConvertPolicyWithLLM(userPolicyPath, codePolicyPath)
}

// getRBACInfo returns RBAC information for the current user
func (s *Server) getRBACInfo() string {
	// Try to get current user
	username, err := git.GetCurrentUser()
	if err != nil {
		// Not in a git environment or user not configured
		return ""
	}

	// Get user's role
	userRole, err := roles.GetUserRole(username)
	if err != nil {
		// Roles not configured
		return ""
	}

	if userRole == "none" {
		return fmt.Sprintf("⚠️  RBAC: User '%s' has no assigned role. You may not have permission to modify files.", username)
	}

	// Load user policy to get RBAC details
	userPolicy, err := roles.LoadUserPolicyFromRepo()
	if err != nil {
		// User policy not available
		return fmt.Sprintf("🔐 RBAC: Current user '%s' has role '%s'", username, userRole)
	}

	// Check if RBAC is defined
	if userPolicy.RBAC == nil || userPolicy.RBAC.Roles == nil {
		return fmt.Sprintf("🔐 RBAC: Current user '%s' has role '%s' (no restrictions defined)", username, userRole)
	}

	// Get role configuration
	roleConfig, exists := userPolicy.RBAC.Roles[userRole]
	if !exists {
		return fmt.Sprintf("⚠️  RBAC: User '%s' has role '%s', but role is not defined in policy", username, userRole)
	}

	// Build RBAC info message
	var rbacMsg strings.Builder
	rbacMsg.WriteString("🔐 RBAC Information:\n")
	rbacMsg.WriteString(fmt.Sprintf("   User: %s\n", username))
	rbacMsg.WriteString(fmt.Sprintf("   Role: %s\n", userRole))

	if len(roleConfig.AllowWrite) > 0 {
		rbacMsg.WriteString(fmt.Sprintf("   Allowed paths: %s\n", strings.Join(roleConfig.AllowWrite, ", ")))
	} else {
		rbacMsg.WriteString("   Allowed paths: All files (no restrictions)\n")
	}

	if len(roleConfig.DenyWrite) > 0 {
		rbacMsg.WriteString(fmt.Sprintf("   Denied paths: %s\n", strings.Join(roleConfig.DenyWrite, ", ")))
	}

	if roleConfig.CanEditPolicy {
		rbacMsg.WriteString("   Can edit policy: Yes\n")
	}

	if roleConfig.CanEditRoles {
		rbacMsg.WriteString("   Can edit roles: Yes\n")
	}

	rbacMsg.WriteString("\n⚠️  Note: Modifications to denied paths will be blocked during validation.")

	return rbacMsg.String()
}

// ValidationResultRecord represents a single validation result with timestamp
type ValidationResultRecord struct {
	Timestamp    string          `json:"timestamp"`
	Status       string          `json:"status"` // "passed", "warning", "failed"
	TotalChecks  int             `json:"total_checks"`
	Passed       int             `json:"passed"`
	Failed       int             `json:"failed"`
	Violations   []ViolationItem `json:"violations"`
	FilesChecked []string        `json:"files_checked"`
}

// ValidationHistory represents the history of validation results
type ValidationHistory struct {
	Results []ValidationResultRecord `json:"results"`
}

// saveValidationResults saves validation results to .sym/validation-results.json
func (s *Server) saveValidationResults(result *validator.ValidationResult, violations []ViolationItem, hasErrors bool) error {
	// Get git root to find .sym directory
	repoRoot, err := git.GetRepoRoot()
	if err != nil {
		return fmt.Errorf("failed to get repository root: %w", err)
	}

	symDir := filepath.Join(repoRoot, ".sym")
	if err := os.MkdirAll(symDir, 0755); err != nil {
		return fmt.Errorf("failed to create .sym directory: %w", err)
	}

	resultsPath := filepath.Join(symDir, "validation-results.json")

	// Load existing history
	var history ValidationHistory
	if data, err := os.ReadFile(resultsPath); err == nil {
		if err := json.Unmarshal(data, &history); err != nil {
			// If unmarshal fails, start fresh
			history = ValidationHistory{Results: []ValidationResultRecord{}}
		}
	} else {
		history = ValidationHistory{Results: []ValidationResultRecord{}}
	}

	// Determine status
	status := "passed"
	if hasErrors {
		status = "failed"
	} else if len(violations) > 0 {
		status = "warning"
	}

	// Collect files checked (from violations)
	filesChecked := make(map[string]bool)
	for _, v := range violations {
		if v.File != "" {
			filesChecked[v.File] = true
		}
	}
	filesCheckedList := make([]string, 0, len(filesChecked))
	for file := range filesChecked {
		filesCheckedList = append(filesCheckedList, file)
	}

	// Create new record
	record := ValidationResultRecord{
		Timestamp:    time.Now().Format(time.RFC3339),
		Status:       status,
		TotalChecks:  result.Checked,
		Passed:       result.Passed,
		Failed:       result.Failed,
		Violations:   violations,
		FilesChecked: filesCheckedList,
	}

	// Add to history (keep last 50 results)
	history.Results = append(history.Results, record)
	if len(history.Results) > 50 {
		history.Results = history.Results[len(history.Results)-50:]
	}

	// Save to file
	data, err := json.MarshalIndent(history, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal validation results: %w", err)
	}

	if err := os.WriteFile(resultsPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write validation results: %w", err)
	}

	fmt.Fprintf(os.Stderr, "✓ Validation results saved to %s\n", resultsPath)
	return nil
}

// handleListCategory handles category listing requests.
func (s *Server) handleListCategory() (interface{}, *RPCError) {
	category := s.getCategory()

	var textContent string
	if len(category) == 0 {
		textContent = "No categories defined in user-policy.json.\n\nRun 'sym init' to create default categories."
	} else {
		textContent = fmt.Sprintf("Available categories (%d):\n\n", len(category))
		for _, cat := range category {
			textContent += fmt.Sprintf("• %s\n  %s\n\n", cat.Name, cat.Description)
		}
		textContent += "Use list_convention with a specific category to get rules for that category."
	}

	return map[string]interface{}{
		"content": []map[string]interface{}{
			{"type": "text", "text": textContent},
		},
	}, nil
}

// getCategory returns categories from user-policy.json.
func (s *Server) getCategory() []schema.CategoryDef {
	if s.userPolicy != nil {
		return s.userPolicy.Category
	}
	return nil
}

// handleAddCategory handles adding categories (batch mode).
func (s *Server) handleAddCategory(input AddCategoryInput) (interface{}, *RPCError) {
	// Validate input
	if len(input.Categories) == 0 {
		return nil, &RPCError{Code: -32602, Message: "At least one category is required in 'categories' array"}
	}

	// Build existing names map
	existingNames := make(map[string]bool)
	for _, cat := range s.userPolicy.Category {
		existingNames[cat.Name] = true
	}

	var succeeded []string
	var failed []FailedItem

	// Process each category
	for _, cat := range input.Categories {
		// Validate
		if cat.Name == "" {
			failed = append(failed, FailedItem{Name: "(empty)", Reason: "Category name is required"})
			continue
		}
		if cat.Description == "" {
			failed = append(failed, FailedItem{Name: cat.Name, Reason: "Category description is required"})
			continue
		}

		// Check for duplicate
		if existingNames[cat.Name] {
			failed = append(failed, FailedItem{Name: cat.Name, Reason: fmt.Sprintf("Category '%s' already exists", cat.Name)})
			continue
		}

		// Add category
		s.userPolicy.Category = append(s.userPolicy.Category, schema.CategoryDef{
			Name:        cat.Name,
			Description: cat.Description,
		})
		existingNames[cat.Name] = true
		succeeded = append(succeeded, cat.Name)
	}

	// Save policy if any succeeded
	if len(succeeded) > 0 {
		if err := s.saveUserPolicy(); err != nil {
			return nil, &RPCError{Code: -32000, Message: fmt.Sprintf("Failed to save policy: %v", err)}
		}
	}

	// Build response
	return s.buildBatchResponse("Added", succeeded, failed), nil
}

// handleEditCategory handles editing categories (batch mode).
func (s *Server) handleEditCategory(input EditCategoryInput) (interface{}, *RPCError) {
	// Validate input
	if len(input.Edits) == 0 {
		return nil, &RPCError{Code: -32602, Message: "At least one edit is required in 'edits' array"}
	}

	// Build category index map
	categoryIndex := make(map[string]int)
	for i, cat := range s.userPolicy.Category {
		categoryIndex[cat.Name] = i
	}

	var succeeded []string
	var failed []FailedItem
	totalRulesUpdated := 0

	// Process each edit
	for _, edit := range input.Edits {
		// Validate
		if edit.Name == "" {
			failed = append(failed, FailedItem{Name: "(empty)", Reason: "Category name is required"})
			continue
		}
		if edit.NewName == "" && edit.Description == "" {
			failed = append(failed, FailedItem{Name: edit.Name, Reason: "At least one of new_name or description must be provided"})
			continue
		}

		// Find category
		idx, exists := categoryIndex[edit.Name]
		if !exists {
			failed = append(failed, FailedItem{Name: edit.Name, Reason: fmt.Sprintf("Category '%s' not found", edit.Name)})
			continue
		}

		rulesUpdated := 0
		resultText := edit.Name

		// If renaming
		if edit.NewName != "" && edit.NewName != edit.Name {
			// Check for duplicate
			if _, dupExists := categoryIndex[edit.NewName]; dupExists {
				failed = append(failed, FailedItem{Name: edit.Name, Reason: fmt.Sprintf("Category '%s' already exists", edit.NewName)})
				continue
			}

			// Update rule references
			for i := range s.userPolicy.Rules {
				if s.userPolicy.Rules[i].Category == edit.Name {
					s.userPolicy.Rules[i].Category = edit.NewName
					rulesUpdated++
				}
			}

			// Update index map
			delete(categoryIndex, edit.Name)
			categoryIndex[edit.NewName] = idx

			s.userPolicy.Category[idx].Name = edit.NewName
			resultText = fmt.Sprintf("%s → %s", edit.Name, edit.NewName)
		}

		// Update description if provided
		if edit.Description != "" {
			s.userPolicy.Category[idx].Description = edit.Description
			if edit.NewName == "" || edit.NewName == edit.Name {
				resultText = fmt.Sprintf("%s (description updated)", edit.Name)
			}
		}

		if rulesUpdated > 0 {
			resultText = fmt.Sprintf("%s (%d rules updated)", resultText, rulesUpdated)
			totalRulesUpdated += rulesUpdated
		}

		succeeded = append(succeeded, resultText)
	}

	// Save policy if any succeeded
	if len(succeeded) > 0 {
		if err := s.saveUserPolicy(); err != nil {
			return nil, &RPCError{Code: -32000, Message: fmt.Sprintf("Failed to save policy: %v", err)}
		}
	}

	// Build response
	return s.buildBatchResponse("Updated", succeeded, failed), nil
}

// handleRemoveCategory handles removing categories (batch mode).
func (s *Server) handleRemoveCategory(input RemoveCategoryInput) (interface{}, *RPCError) {
	// Validate input
	if len(input.Names) == 0 {
		return nil, &RPCError{Code: -32602, Message: "At least one category name is required in 'names' array"}
	}

	// Build category index map and rule count map
	categoryIndex := make(map[string]int)
	for i, cat := range s.userPolicy.Category {
		categoryIndex[cat.Name] = i
	}

	ruleCount := make(map[string]int)
	for _, rule := range s.userPolicy.Rules {
		ruleCount[rule.Category]++
	}

	var succeeded []string
	var failed []FailedItem
	toRemove := make(map[int]bool) // indices to remove

	// Process each name
	for _, name := range input.Names {
		// Validate
		if name == "" {
			failed = append(failed, FailedItem{Name: "(empty)", Reason: "Category name is required"})
			continue
		}

		// Find category
		idx, exists := categoryIndex[name]
		if !exists {
			failed = append(failed, FailedItem{Name: name, Reason: fmt.Sprintf("Category '%s' not found", name)})
			continue
		}

		// Check if rules reference this category
		if count := ruleCount[name]; count > 0 {
			failed = append(failed, FailedItem{Name: name, Reason: fmt.Sprintf("Category is used by %d rule(s)", count)})
			continue
		}

		toRemove[idx] = true
		succeeded = append(succeeded, name)
	}

	// Remove categories (in reverse order to preserve indices)
	if len(toRemove) > 0 {
		newCategories := make([]schema.CategoryDef, 0, len(s.userPolicy.Category)-len(toRemove))
		for i, cat := range s.userPolicy.Category {
			if !toRemove[i] {
				newCategories = append(newCategories, cat)
			}
		}
		s.userPolicy.Category = newCategories

		if err := s.saveUserPolicy(); err != nil {
			return nil, &RPCError{Code: -32000, Message: fmt.Sprintf("Failed to save policy: %v", err)}
		}
	}

	// Build response
	return s.buildBatchResponse("Removed", succeeded, failed), nil
}

// buildBatchResponse builds a standardized batch operation response.
func (s *Server) buildBatchResponse(action string, succeeded []string, failed []FailedItem) map[string]interface{} {
	var textContent string

	if len(failed) == 0 && len(succeeded) > 0 {
		// All succeeded
		textContent = fmt.Sprintf("%s %d category(ies) successfully:\n", action, len(succeeded))
		for _, name := range succeeded {
			textContent += fmt.Sprintf("  ✓ %s\n", name)
		}
	} else if len(succeeded) == 0 && len(failed) > 0 {
		// All failed
		textContent = fmt.Sprintf("Failed to %s any categories:\n", strings.ToLower(action))
		for _, f := range failed {
			textContent += fmt.Sprintf("  ✗ %s: %s\n", f.Name, f.Reason)
		}
	} else if len(succeeded) > 0 && len(failed) > 0 {
		// Partial success
		textContent = "Batch operation completed with errors:\n"
		textContent += fmt.Sprintf("  ✓ Succeeded (%d):\n", len(succeeded))
		for _, name := range succeeded {
			textContent += fmt.Sprintf("    - %s\n", name)
		}
		textContent += fmt.Sprintf("  ✗ Failed (%d):\n", len(failed))
		for _, f := range failed {
			textContent += fmt.Sprintf("    - %s: %s\n", f.Name, f.Reason)
		}
	} else {
		// Nothing to do
		textContent = "No categories to process."
	}

	return map[string]interface{}{
		"content": []map[string]interface{}{
			{"type": "text", "text": textContent},
		},
	}
}

// handleImportConventions handles the import_convention tool.
func (s *Server) handleImportConventions(ctx context.Context, input ImportConventionsInput) (interface{}, *RPCError) {
	// Validate input
	if input.Path == "" {
		return nil, &RPCError{Code: -32602, Message: "File path is required"}
	}

	// Default mode to append
	mode := importer.ImportModeAppend
	if input.Mode == "clear" {
		mode = importer.ImportModeClear
	}

	// Setup LLM provider
	llmCfg := llm.LoadConfig()
	llmProvider, err := llm.New(llmCfg)
	if err != nil {
		return nil, &RPCError{Code: -32000, Message: fmt.Sprintf("Failed to create LLM provider: %v", err)}
	}
	defer func() { _ = llmProvider.Close() }()

	// Create importer and execute
	imp := importer.NewImporter(llmProvider, false)
	importInput := &importer.ImportInput{
		Path: input.Path,
		Mode: mode,
	}

	result, err := imp.Import(ctx, importInput)
	if err != nil {
		// Build partial result response if available
		if result != nil {
			return s.buildImportResponse(result, err), nil
		}
		return nil, &RPCError{Code: -32000, Message: fmt.Sprintf("Import failed: %v", err)}
	}

	// Reload user policy after import
	if s.userPolicy != nil {
		userPolicyPath := s.getUserPolicyPath()
		if userPolicy, err := s.loader.LoadUserPolicy(userPolicyPath); err == nil {
			s.userPolicy = userPolicy
		}
	}

	return s.buildImportResponse(result, nil), nil
}

// buildImportResponse builds the MCP response for import operation.
func (s *Server) buildImportResponse(result *importer.ImportResult, importErr error) map[string]interface{} {
	var textContent strings.Builder
	textContent.WriteString("Convention Import ")

	if importErr != nil {
		textContent.WriteString("(completed with errors)\n\n")
	} else {
		textContent.WriteString("Complete\n\n")
	}

	if result.FileProcessed != "" {
		textContent.WriteString(fmt.Sprintf("Processed: %s\n\n", result.FileProcessed))
	}

	if result.CategoriesRemoved > 0 || result.RulesRemoved > 0 {
		textContent.WriteString(fmt.Sprintf("Removed: %d categories, %d rules (clear mode)\n\n",
			result.CategoriesRemoved, result.RulesRemoved))
	}

	if len(result.CategoriesAdded) > 0 {
		textContent.WriteString(fmt.Sprintf("Added %d categories:\n", len(result.CategoriesAdded)))
		for _, cat := range result.CategoriesAdded {
			textContent.WriteString(fmt.Sprintf("  • %s: %s\n", cat.Name, cat.Description))
		}
		textContent.WriteString("\n")
	}

	if len(result.RulesAdded) > 0 {
		textContent.WriteString(fmt.Sprintf("Added %d rules:\n", len(result.RulesAdded)))
		for _, rule := range result.RulesAdded {
			textContent.WriteString(fmt.Sprintf("  • [%s] %s (%s)\n", rule.ID, rule.Say, rule.Category))
		}
		textContent.WriteString("\n")
	}

	if len(result.Warnings) > 0 {
		textContent.WriteString(fmt.Sprintf("Warnings (%d):\n", len(result.Warnings)))
		for _, w := range result.Warnings {
			textContent.WriteString(fmt.Sprintf("  • %s\n", w))
		}
		textContent.WriteString("\n")
	}

	if importErr != nil {
		textContent.WriteString(fmt.Sprintf("Import Error: %v\n", importErr))
	}

	return map[string]interface{}{
		"content": []map[string]interface{}{
			{"type": "text", "text": textContent.String()},
		},
	}
}

// handleConvertPolicy handles the convert tool.
func (s *Server) handleConvertPolicy(ctx context.Context, input ConvertPolicyInput) (interface{}, *RPCError) {
	// 1. Determine input path
	inputPath := input.InputPath
	if inputPath == "" {
		inputPath = s.getUserPolicyPath()
	}

	outputDir := input.OutputDir
	if outputDir == "" {
		outputDir = filepath.Dir(inputPath)
	}

	// 2. Load user-policy.json
	data, err := os.ReadFile(inputPath)
	if err != nil {
		return nil, &RPCError{Code: -32000, Message: fmt.Sprintf("Failed to read policy: %v", err)}
	}
	var userPolicy schema.UserPolicy
	if err := json.Unmarshal(data, &userPolicy); err != nil {
		return nil, &RPCError{Code: -32000, Message: fmt.Sprintf("Failed to parse policy: %v", err)}
	}

	// 3. Setup LLM provider
	llmCfg := llm.LoadConfig()
	llmProvider, err := llm.New(llmCfg)
	if err != nil {
		return nil, &RPCError{Code: -32000, Message: fmt.Sprintf("Failed to create LLM provider: %v", err)}
	}
	defer func() { _ = llmProvider.Close() }()

	// 4. Create converter and execute
	conv := converter.NewConverter(llmProvider, outputDir)
	result, err := conv.Convert(ctx, &userPolicy)
	if err != nil {
		if result != nil {
			return s.buildConvertResponse(result, err), nil
		}
		return nil, &RPCError{Code: -32000, Message: fmt.Sprintf("Conversion failed: %v", err)}
	}

	// 5. Reload code policy after conversion
	codePolicyPath := filepath.Join(outputDir, "code-policy.json")
	if codePolicy, loadErr := s.loader.LoadCodePolicy(codePolicyPath); loadErr == nil {
		s.codePolicy = codePolicy
	}

	return s.buildConvertResponse(result, nil), nil
}

// buildConvertResponse builds the MCP response for convert operation.
func (s *Server) buildConvertResponse(result *converter.ConvertResult, convertErr error) map[string]interface{} {
	var textContent strings.Builder

	if convertErr != nil {
		textContent.WriteString("Conversion completed with errors\n\n")
	} else {
		textContent.WriteString("✓ Conversion completed successfully\n\n")
	}

	if len(result.GeneratedFiles) > 0 {
		textContent.WriteString(fmt.Sprintf("Generated %d file(s):\n", len(result.GeneratedFiles)))
		for _, file := range result.GeneratedFiles {
			textContent.WriteString(fmt.Sprintf("  • %s\n", file))
		}
		textContent.WriteString("\n")
	}

	if len(result.Errors) > 0 {
		textContent.WriteString(fmt.Sprintf("Errors (%d):\n", len(result.Errors)))
		for linter, err := range result.Errors {
			textContent.WriteString(fmt.Sprintf("  • %s: %v\n", linter, err))
		}
		textContent.WriteString("\n")
	}

	if len(result.Warnings) > 0 {
		textContent.WriteString(fmt.Sprintf("Warnings (%d):\n", len(result.Warnings)))
		for _, warning := range result.Warnings {
			textContent.WriteString(fmt.Sprintf("  • %s\n", warning))
		}
	}

	if convertErr != nil {
		textContent.WriteString(fmt.Sprintf("\nError: %v\n", convertErr))
	}

	return map[string]interface{}{
		"content": []map[string]interface{}{
			{"type": "text", "text": textContent.String()},
		},
	}
}

// getUserPolicyPath returns the user policy file path.
func (s *Server) getUserPolicyPath() string {
	projectCfg, _ := config.LoadProjectConfig()
	userPolicyPath := projectCfg.PolicyPath
	if userPolicyPath == "" {
		repoRoot, err := git.GetRepoRoot()
		if err != nil {
			return ".sym/user-policy.json"
		}
		return filepath.Join(repoRoot, ".sym", "user-policy.json")
	}
	if !filepath.IsAbs(userPolicyPath) {
		if repoRoot, err := git.GetRepoRoot(); err == nil {
			userPolicyPath = filepath.Join(repoRoot, userPolicyPath)
		}
	}
	return userPolicyPath
}

// saveUserPolicy saves the user policy to file.
func (s *Server) saveUserPolicy() error {
	return policy.SavePolicy(s.userPolicy, "")
}

// handleAddConvention handles adding conventions (batch mode).
func (s *Server) handleAddConvention(input AddConventionInput) (interface{}, *RPCError) {
	// Validate input
	if len(input.Conventions) == 0 {
		return nil, &RPCError{Code: -32602, Message: "At least one convention is required in 'conventions' array"}
	}

	// Build existing IDs map
	existingIDs := make(map[string]bool)
	for _, rule := range s.userPolicy.Rules {
		existingIDs[rule.ID] = true
	}

	var succeeded []string
	var failed []FailedItem
	var addedRules []schema.UserRule

	// Process each convention
	for _, conv := range input.Conventions {
		// Validate
		if conv.ID == "" {
			failed = append(failed, FailedItem{Name: "(empty)", Reason: "Convention ID is required"})
			continue
		}
		if conv.Say == "" {
			failed = append(failed, FailedItem{Name: conv.ID, Reason: "Convention 'say' description is required"})
			continue
		}

		// Check for duplicate
		if existingIDs[conv.ID] {
			failed = append(failed, FailedItem{Name: conv.ID, Reason: fmt.Sprintf("Convention '%s' already exists", conv.ID)})
			continue
		}

		// Add convention
		rule := schema.UserRule{
			ID:        conv.ID,
			Say:       conv.Say,
			Category:  conv.Category,
			Languages: conv.Languages,
			Severity:  conv.Severity,
			Autofix:   conv.Autofix,
			Message:   conv.Message,
			Example:   conv.Example,
			Include:   conv.Include,
			Exclude:   conv.Exclude,
		}
		s.userPolicy.Rules = append(s.userPolicy.Rules, rule)
		addedRules = append(addedRules, rule)
		existingIDs[conv.ID] = true
		succeeded = append(succeeded, conv.ID)
	}

	// Update defaults.languages with new languages from rules
	if len(addedRules) > 0 {
		policy.UpdateDefaultsLanguages(s.userPolicy, addedRules)
	}

	// Save policy if any succeeded
	if len(succeeded) > 0 {
		if err := s.saveUserPolicy(); err != nil {
			return nil, &RPCError{Code: -32000, Message: fmt.Sprintf("Failed to save policy: %v", err)}
		}
	}

	// Build response
	return s.buildConventionBatchResponse("Added", succeeded, failed), nil
}

// handleEditConvention handles editing conventions (batch mode).
func (s *Server) handleEditConvention(input EditConventionInput) (interface{}, *RPCError) {
	// Validate input
	if len(input.Edits) == 0 {
		return nil, &RPCError{Code: -32602, Message: "At least one edit is required in 'edits' array"}
	}

	// Build rule index map
	ruleIndex := make(map[string]int)
	for i, rule := range s.userPolicy.Rules {
		ruleIndex[rule.ID] = i
	}

	var succeeded []string
	var failed []FailedItem
	var editedRules []schema.UserRule

	// Process each edit
	for _, edit := range input.Edits {
		// Validate
		if edit.ID == "" {
			failed = append(failed, FailedItem{Name: "(empty)", Reason: "Convention ID is required"})
			continue
		}

		// Check if at least one field to edit
		hasEdit := edit.NewID != "" || edit.Say != "" || edit.Category != "" ||
			len(edit.Languages) > 0 || edit.Severity != "" || edit.Autofix != nil ||
			edit.Message != "" || edit.Example != "" || len(edit.Include) > 0 || len(edit.Exclude) > 0

		if !hasEdit {
			failed = append(failed, FailedItem{Name: edit.ID, Reason: "At least one field to edit must be provided"})
			continue
		}

		// Find convention
		idx, exists := ruleIndex[edit.ID]
		if !exists {
			failed = append(failed, FailedItem{Name: edit.ID, Reason: fmt.Sprintf("Convention '%s' not found", edit.ID)})
			continue
		}

		resultText := edit.ID

		// If renaming ID
		if edit.NewID != "" && edit.NewID != edit.ID {
			// Check for duplicate
			if _, dupExists := ruleIndex[edit.NewID]; dupExists {
				failed = append(failed, FailedItem{Name: edit.ID, Reason: fmt.Sprintf("Convention '%s' already exists", edit.NewID)})
				continue
			}

			// Update index map
			delete(ruleIndex, edit.ID)
			ruleIndex[edit.NewID] = idx

			s.userPolicy.Rules[idx].ID = edit.NewID
			resultText = fmt.Sprintf("%s → %s", edit.ID, edit.NewID)
		}

		// Update other fields
		if edit.Say != "" {
			s.userPolicy.Rules[idx].Say = edit.Say
		}
		if edit.Category != "" {
			s.userPolicy.Rules[idx].Category = edit.Category
		}
		if len(edit.Languages) > 0 {
			s.userPolicy.Rules[idx].Languages = edit.Languages
		}
		if edit.Severity != "" {
			s.userPolicy.Rules[idx].Severity = edit.Severity
		}
		if edit.Autofix != nil {
			s.userPolicy.Rules[idx].Autofix = *edit.Autofix
		}
		if edit.Message != "" {
			s.userPolicy.Rules[idx].Message = edit.Message
		}
		if edit.Example != "" {
			s.userPolicy.Rules[idx].Example = edit.Example
		}
		if len(edit.Include) > 0 {
			s.userPolicy.Rules[idx].Include = edit.Include
		}
		if len(edit.Exclude) > 0 {
			s.userPolicy.Rules[idx].Exclude = edit.Exclude
		}

		editedRules = append(editedRules, s.userPolicy.Rules[idx])
		succeeded = append(succeeded, resultText)
	}

	// Update defaults.languages with new languages from edited rules
	if len(editedRules) > 0 {
		policy.UpdateDefaultsLanguages(s.userPolicy, editedRules)
	}

	// Save policy if any succeeded
	if len(succeeded) > 0 {
		if err := s.saveUserPolicy(); err != nil {
			return nil, &RPCError{Code: -32000, Message: fmt.Sprintf("Failed to save policy: %v", err)}
		}
	}

	// Build response
	return s.buildConventionBatchResponse("Updated", succeeded, failed), nil
}

// handleRemoveConvention handles removing conventions (batch mode).
func (s *Server) handleRemoveConvention(input RemoveConventionInput) (interface{}, *RPCError) {
	// Validate input
	if len(input.IDs) == 0 {
		return nil, &RPCError{Code: -32602, Message: "At least one convention ID is required in 'ids' array"}
	}

	// Build rule index map
	ruleIndex := make(map[string]int)
	for i, rule := range s.userPolicy.Rules {
		ruleIndex[rule.ID] = i
	}

	var succeeded []string
	var failed []FailedItem
	toRemove := make(map[int]bool) // indices to remove

	// Process each ID
	for _, id := range input.IDs {
		// Validate
		if id == "" {
			failed = append(failed, FailedItem{Name: "(empty)", Reason: "Convention ID is required"})
			continue
		}

		// Find convention
		idx, exists := ruleIndex[id]
		if !exists {
			failed = append(failed, FailedItem{Name: id, Reason: fmt.Sprintf("Convention '%s' not found", id)})
			continue
		}

		toRemove[idx] = true
		succeeded = append(succeeded, id)
	}

	// Remove conventions
	if len(toRemove) > 0 {
		newRules := make([]schema.UserRule, 0, len(s.userPolicy.Rules)-len(toRemove))
		for i, rule := range s.userPolicy.Rules {
			if !toRemove[i] {
				newRules = append(newRules, rule)
			}
		}
		s.userPolicy.Rules = newRules

		if err := s.saveUserPolicy(); err != nil {
			return nil, &RPCError{Code: -32000, Message: fmt.Sprintf("Failed to save policy: %v", err)}
		}
	}

	// Build response
	return s.buildConventionBatchResponse("Removed", succeeded, failed), nil
}

// buildConventionBatchResponse builds a standardized batch operation response for conventions.
func (s *Server) buildConventionBatchResponse(action string, succeeded []string, failed []FailedItem) map[string]interface{} {
	var textContent string

	if len(failed) == 0 && len(succeeded) > 0 {
		// All succeeded
		textContent = fmt.Sprintf("%s %d convention(s) successfully:\n", action, len(succeeded))
		for _, id := range succeeded {
			textContent += fmt.Sprintf("  ✓ %s\n", id)
		}
	} else if len(succeeded) == 0 && len(failed) > 0 {
		// All failed
		textContent = fmt.Sprintf("Failed to %s any conventions:\n", strings.ToLower(action))
		for _, f := range failed {
			textContent += fmt.Sprintf("  ✗ %s: %s\n", f.Name, f.Reason)
		}
	} else if len(succeeded) > 0 && len(failed) > 0 {
		// Partial success
		textContent = "Batch operation completed with errors:\n"
		textContent += fmt.Sprintf("  ✓ Succeeded (%d):\n", len(succeeded))
		for _, id := range succeeded {
			textContent += fmt.Sprintf("    - %s\n", id)
		}
		textContent += fmt.Sprintf("  ✗ Failed (%d):\n", len(failed))
		for _, f := range failed {
			textContent += fmt.Sprintf("    - %s: %s\n", f.Name, f.Reason)
		}
	} else {
		// Nothing to do
		textContent = "No conventions to process."
	}

	return map[string]interface{}{
		"content": []map[string]interface{}{
			{"type": "text", "text": textContent},
		},
	}
}
