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
	"github.com/DevSymphony/sym-cli/internal/envutil"
	"github.com/DevSymphony/sym-cli/internal/git"
	"github.com/DevSymphony/sym-cli/internal/llm"
	"github.com/DevSymphony/sym-cli/internal/policy"
	"github.com/DevSymphony/sym-cli/internal/roles"
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

	// Setup LLM client (backend auto-selection via @llm)
	llmClient := llm.NewClient(
		llm.WithTimeout(30 * time.Second),
	)

	// Create converter with output directory
	outputDir := filepath.Dir(codePolicyPath)
	conv := converter.NewConverter(llmClient, outputDir)

	// Setup context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(30*10)*time.Second)
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
		// First check .env for POLICY_PATH, otherwise use default
		userPolicyPath := envutil.GetPolicyPath()
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
	fmt.Fprintln(os.Stderr, "Available tools: query_conventions, validate_code")

	// Use official MCP go-sdk for stdio to ensure spec-compliant framing and lifecycle
	return s.runStdioWithSDK(context.Background())
}

// RPCError is an error type used for internal error handling.
type RPCError struct {
	Code    int
	Message string
}

// QueryConventionsInput represents the input schema for the query_conventions tool (go-sdk).
type QueryConventionsInput struct {
	Category  string   `json:"category,omitempty" jsonschema:"Filter by category (optional). Use 'all' or leave empty to fetch all categories. Options: security, style, documentation, error_handling, architecture, performance, testing"`
	Languages []string `json:"languages,omitempty" jsonschema:"Programming languages to filter by (optional). Leave empty to get conventions for all languages. Examples: go, javascript, typescript, python, java"`
}

// ValidateCodeInput represents the input schema for the validate_code tool (go-sdk).
type ValidateCodeInput struct {
	Role string `json:"role,omitempty" jsonschema:"RBAC role for validation (optional)"`
}

// runStdioWithSDK runs a spec-compliant MCP server over stdio using the official go-sdk.
func (s *Server) runStdioWithSDK(ctx context.Context) error {
	server := sdkmcp.NewServer(&sdkmcp.Implementation{
		Name:    "symphony",
		Version: "1.0.0",
	}, nil)

	// Tool: query_conventions
	sdkmcp.AddTool(server, &sdkmcp.Tool{
		Name:        "query_conventions",
		Description: "[MANDATORY BEFORE CODING] Query project conventions BEFORE writing any code to ensure compliance from the start.",
	}, func(ctx context.Context, req *sdkmcp.CallToolRequest, input QueryConventionsInput) (*sdkmcp.CallToolResult, map[string]any, error) {
		params := map[string]any{
			"category":  input.Category,
			"languages": input.Languages,
		}
		result, rpcErr := s.handleQueryConventions(params)
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

	// Run the server over stdio until the client disconnects
	return server.Run(ctx, &sdkmcp.StdioTransport{})
}

// QueryConventionsRequest is a request to query conventions.
type QueryConventionsRequest struct {
	Category  string   `json:"category"`  // optional; use "all" or empty to fetch all categories
	Languages []string `json:"languages"` // optional; empty means all languages
}

// ConventionItem is a convention item.
type ConventionItem struct {
	ID          string `json:"id"`
	Category    string `json:"category"`
	Description string `json:"description"`
	Message     string `json:"message"`
	Severity    string `json:"severity"`
}

// handleQueryConventions handles convention query requests.
// It finds and returns relevant conventions by category.
func (s *Server) handleQueryConventions(params map[string]interface{}) (interface{}, *RPCError) {
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
	// If category is empty or "all", return all categories
	if strings.TrimSpace(req.Category) == "" || strings.EqualFold(req.Category, "all") {
		req.Category = ""
	}

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
			if req.Category != "" && rule.Category != req.Category {
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

			if req.Category != "" && rule.Category != req.Category {
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
	// GetGitChanges already includes all types of changes
	changes, err := validator.GetGitChanges()
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

	var llmClient *llm.Client
	if session != nil {
		// MCP mode: use host LLM via sampling
		llmClient = llm.NewClient(llm.WithMCPSession(session))
		fmt.Fprintf(os.Stderr, "✓ Using host LLM via MCP sampling\n")
	} else {
		// Auto mode: use configured LLM backend (CLI/API)
		llmClient = llm.NewClient()
		fmt.Fprintf(os.Stderr, "✓ Using configured LLM backend\n")
	}

	// Create unified validator that handles all engines + RBAC
	v := validator.NewValidator(validationPolicy, false) // verbose=false for MCP
	v.SetLLMClient(llmClient)
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
// 2. user policy has more rules than code policy (indicating new rules added), OR
// 3. user policy has rule IDs that don't exist in code policy
func (s *Server) needsConversion(codePolicyPath string) bool {
	// If no code policy exists, conversion is needed
	if s.codePolicy == nil {
		return true
	}

	// If no user policy, no conversion needed
	if s.userPolicy == nil {
		return false
	}

	// Check if user policy has more rules
	if len(s.userPolicy.Rules) > len(s.codePolicy.Rules) {
		return true
	}

	// Check if all user policy rule IDs exist in code policy
	codePolicyRuleIDs := make(map[string]bool)
	for _, rule := range s.codePolicy.Rules {
		codePolicyRuleIDs[rule.ID] = true
	}

	for _, userRule := range s.userPolicy.Rules {
		if !codePolicyRuleIDs[userRule.ID] {
			// Found a user rule that doesn't exist in code policy
			return true
		}
	}

	return false
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
