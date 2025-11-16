package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
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

	// Setup LLM client
	apiKey := envutil.GetAPIKey("OPENAI_API_KEY")
	if apiKey == "" {
		return fmt.Errorf("OPENAI_API_KEY not found in environment or .sym/.env")
	}

	llmClient := llm.NewClient(apiKey,
		llm.WithModel("gpt-4o-mini"),
		llm.WithTimeout(30*time.Second),
	)

	// Create converter
	conv := converter.NewConverter(converter.WithLLMClient(llmClient))

	// Setup context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(len(userPolicy.Rules)*30)*time.Second)
	defer cancel()

	fmt.Fprintf(os.Stderr, "Converting %d rules...\n", len(userPolicy.Rules))

	// Convert to all targets
	result, err := conv.ConvertMultiTarget(ctx, &userPolicy, converter.MultiTargetConvertOptions{
		Targets:             []string{"all"},
		OutputDir:           filepath.Dir(codePolicyPath),
		ConfidenceThreshold: 0.7,
	})
	if err != nil {
		return fmt.Errorf("conversion failed: %w", err)
	}

	// Write code policy
	codePolicyJSON, err := json.MarshalIndent(result.CodePolicy, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to serialize code policy: %w", err)
	}

	if err := os.WriteFile(codePolicyPath, codePolicyJSON, 0644); err != nil {
		return fmt.Errorf("failed to write code policy: %w", err)
	}

	// Write linter configs
	for linterName, config := range result.LinterConfigs {
		outputPath := filepath.Join(filepath.Dir(codePolicyPath), config.Filename)
		if err := os.WriteFile(outputPath, config.Content, 0644); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to write %s config: %v\n", linterName, err)
		} else {
			fmt.Fprintf(os.Stderr, "  ✓ Generated %s: %s\n", linterName, outputPath)
		}
	}

	return nil
}

// Server is a MCP (Model Context Protocol) server.
// It communicates via JSON-RPC over stdio or HTTP.
type Server struct {
	host       string
	port       int
	configPath string
	userPolicy *schema.UserPolicy
	codePolicy *schema.CodePolicy
	loader     *policy.Loader
}

// NewServer creates a new MCP server instance.
func NewServer(host string, port int, configPath string) *Server {
	return &Server{
		host:       host,
		port:       port,
		configPath: configPath,
		loader:     policy.NewLoader(false), // verbose = false for MCP
	}
}

// Start starts the MCP server.
// It communicates via JSON-RPC over stdio or HTTP.
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

	if s.port > 0 {
		return s.startHTTPServer()
	}

	fmt.Fprintln(os.Stderr, "Symphony MCP server started (stdio mode)")
	fmt.Fprintf(os.Stderr, "Listening on: %s:%d\n", s.host, s.port)
	fmt.Fprintln(os.Stderr, "Available tools: query_conventions, validate_code")

	// Use official MCP go-sdk for stdio to ensure spec-compliant framing and lifecycle
	return s.runStdioWithSDK(context.Background())
}

// startHTTPServer starts HTTP server for JSON-RPC.
func (s *Server) startHTTPServer() error {
	addr := fmt.Sprintf("%s:%d", s.host, s.port)

	mux := http.NewServeMux()
	mux.HandleFunc("/", s.handleHTTPRequest)
	mux.HandleFunc("/health", s.handleHealthCheck)

	server := &http.Server{
		Addr:              addr,
		Handler:           mux,
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
		MaxHeaderBytes:    1 << 20, // 1MB
	}

	fmt.Fprintf(os.Stderr, "Symphony MCP server started (HTTP mode)\n")
	fmt.Fprintf(os.Stderr, "Listening on: http://%s\n", addr)
	fmt.Fprintf(os.Stderr, "Available tools: query_conventions, validate_code\n")

	return server.ListenAndServe()
}

// handleHealthCheck handles health check requests.
func (s *Server) handleHealthCheck(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(map[string]interface{}{
		"status":  "ok",
		"version": "1.0.0",
	}); err != nil {
		// Log error but don't fail - headers already sent
		_ = err
	}
}

// handleHTTPRequest handles HTTP JSON-RPC requests.
func (s *Server) handleHTTPRequest(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req JSONRPCRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(JSONRPCResponse{
			JSONRPC: "2.0",
			Error: &RPCError{
				Code:    -32700,
				Message: "parse error",
			},
			ID: nil,
		})
		return
	}

	var result interface{}
	var rpcErr *RPCError

	switch req.Method {
	case "initialize":
		result, rpcErr = s.handleInitialize(req.Params)
	case "initialized":
		// Notification - no response needed, but we'll send empty result
		result = nil
	case "tools/list":
		result, rpcErr = s.handleToolsList(req.Params)
	case "tools/call":
		result, rpcErr = s.handleToolsCall(req.Params)
	case "query_conventions":
		result, rpcErr = s.handleQueryConventions(req.Params)
	case "validate_code":
		result, rpcErr = s.handleValidateCode(req.Params)
	default:
		rpcErr = &RPCError{
			Code:    -32601,
			Message: fmt.Sprintf("method not found: %s", req.Method),
		}
	}

	resp := JSONRPCResponse{
		JSONRPC: "2.0",
		Result:  result,
		Error:   rpcErr,
		ID:      req.ID,
	}

	w.Header().Set("Content-Type", "application/json")
	if rpcErr != nil {
		w.WriteHeader(http.StatusBadRequest)
	} else {
		w.WriteHeader(http.StatusOK)
	}
	_ = json.NewEncoder(w).Encode(resp)
}

// JSONRPCRequest is a JSON-RPC 2.0 request.
type JSONRPCRequest struct {
	JSONRPC string                 `json:"jsonrpc"`
	Method  string                 `json:"method"`
	Params  map[string]interface{} `json:"params"`
	ID      interface{}            `json:"id"`
}

// JSONRPCResponse is a JSON-RPC 2.0 response.
type JSONRPCResponse struct {
	JSONRPC string      `json:"jsonrpc"`
	Result  interface{} `json:"result,omitempty"`
	Error   *RPCError   `json:"error,omitempty"`
	ID      interface{} `json:"id"`
}

// RPCError is a JSON-RPC error.
type RPCError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
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
		result, rpcErr := s.handleValidateCode(params)
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
	fmt.Fprintf(os.Stderr, "[DEBUG] handleQueryConventions called with params: %+v\n", params)

	if s.userPolicy == nil && s.codePolicy == nil {
		fmt.Fprintf(os.Stderr, "[DEBUG] No policy loaded\n")
		return map[string]interface{}{
			"conventions": []ConventionItem{},
			"message":     "policy not loaded",
		}, nil
	}

	var req QueryConventionsRequest
	paramBytes, _ := json.Marshal(params)
	if err := json.Unmarshal(paramBytes, &req); err != nil {
		fmt.Fprintf(os.Stderr, "[ERROR] Failed to parse parameters: %v\n", err)
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

	fmt.Fprintf(os.Stderr, "[DEBUG] Parsed request: category=%s, languages=%v\n",
		req.Category, req.Languages)

	conventions := s.filterConventions(req)
	fmt.Fprintf(os.Stderr, "[DEBUG] Found %d conventions\n", len(conventions))

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
func (s *Server) handleValidateCode(params map[string]interface{}) (interface{}, *RPCError) {
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

	// Always validate git changes (staged + unstaged)
	// This is the most efficient and relevant approach for AI coding workflows

	// Get unstaged changes
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

	// Also check staged changes
	stagedChanges, err := validator.GetStagedChanges()
	if err == nil {
		changes = append(changes, stagedChanges...)
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

	// Setup LLM client for validation
	apiKey := envutil.GetAPIKey("ANTHROPIC_API_KEY")
	if apiKey == "" {
		apiKey = envutil.GetAPIKey("OPENAI_API_KEY")
	}
	if apiKey == "" {
		return nil, &RPCError{
			Code:    -32000,
			Message: "LLM API key not found (ANTHROPIC_API_KEY or OPENAI_API_KEY required for validation in environment or .sym/.env)",
		}
	}

	llmClient := llm.NewClient(apiKey)

	// Create unified validator that handles all engines + RBAC
	v := validator.NewValidator(validationPolicy, false) // verbose=false for MCP
	v.SetLLMClient(llmClient)
	defer v.Close()

	// Validate git changes using unified validator
	ctx := context.Background()
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

// handleInitialize handles MCP initialize request.
// This is the first request from the client to establish protocol version and capabilities.
func (s *Server) handleInitialize(params map[string]interface{}) (interface{}, *RPCError) {
	return map[string]interface{}{
		"protocolVersion": "2024-11-05",
		"capabilities": map[string]interface{}{
			"tools": map[string]interface{}{},
		},
		"serverInfo": map[string]interface{}{
			"name":    "symphony",
			"version": "1.0.0",
		},
		"instructions": `Symphony Code Convention Enforcer

MANDATORY WORKFLOW for all coding tasks:

STEP 1 [BEFORE CODE]: Query Conventions
→ Call query_conventions tool FIRST before writing any code
→ Filter by category (security, style, architecture, etc.)
→ Filter by language/files you'll work with
→ Review and understand the conventions

STEP 2 [DURING CODE]: Write Code
→ Implement your code following the conventions from Step 1
→ Keep security, style, and architecture guidelines in mind

STEP 3 [AFTER CODE]: Validate Code
→ Call validate_code tool LAST after completing implementation
→ MANDATORY: Must validate before marking task complete
→ Fix any violations found and re-validate
→ Only proceed when validation passes with no errors

This 3-step workflow ensures all code meets project standards. Never skip steps 1 and 3.`,
	}, nil
}

// handleToolsList handles tools/list request.
// Returns the list of available tools that clients can call.
func (s *Server) handleToolsList(params map[string]interface{}) (interface{}, *RPCError) {
	tools := []map[string]interface{}{
		{
			"name": "query_conventions",
			"description": `⚠️  CALL THIS FIRST - BEFORE WRITING ANY CODE ⚠️

This tool is MANDATORY before you start coding. Query project conventions to understand what rules your code must follow.

Usage:
- Filter by category: security, style, error_handling, architecture, performance, testing, documentation
- Filter by languages: javascript, typescript, python, go, java, etc.

Example: Before adding a login feature, call query_conventions(category="security") first.

DO NOT write code before calling this tool. Violations will be caught by validate_code later.`,
			"inputSchema": map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"category": map[string]interface{}{
						"type":        "string",
						"description": "Filter by category (optional). Leave empty or use 'all' to fetch all categories. Options: security, style, documentation, error_handling, architecture, performance, testing",
					},
					"languages": map[string]interface{}{
						"type":        "array",
						"items":       map[string]string{"type": "string"},
						"description": "Programming languages to filter by (optional). Leave empty to get conventions for all languages. Examples: go, javascript, typescript, python, java",
					},
				},
			},
		},
		{
			"name": "validate_code",
			"description": `[STEP 3 - ALWAYS CALL LAST] Validate your git changes against all project conventions.

CRITICAL WORKFLOW:
1. Call this tool AFTER you have written or modified code
2. MANDATORY: Always validate before considering the task complete
3. If violations are found, fix them and validate again
4. Only mark the task as done after validation passes with no errors

This tool automatically validates:
- All STAGED changes (git add)
- All UNSTAGED changes (modified but not staged)
- Only checks the ADDED/MODIFIED lines in your diffs (not entire files)

This is the final quality gate. Never skip this validation step.

The tool will check your changes for:
- Security violations (hardcoded secrets, SQL injection, XSS, etc.)
- Style violations (formatting, naming, documentation)
- Architecture violations (separation of concerns, patterns)
- Error handling violations (missing error checks, empty catch blocks)

If violations are found, you MUST fix them before proceeding.`,
			"inputSchema": map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"role": map[string]interface{}{
						"type":        "string",
						"description": "RBAC role for validation (optional)",
					},
				},
			},
		},
	}

	return map[string]interface{}{
		"tools": tools,
	}, nil
}

// handleToolsCall handles tools/call request.
// This routes tool calls to the appropriate handler based on tool name.
func (s *Server) handleToolsCall(params map[string]interface{}) (interface{}, *RPCError) {
	toolName, ok := params["name"].(string)
	if !ok {
		return nil, &RPCError{
			Code:    -32602,
			Message: "tool name is required",
		}
	}

	arguments, ok := params["arguments"].(map[string]interface{})
	if !ok {
		arguments = make(map[string]interface{})
	}

	switch toolName {
	case "query_conventions":
		return s.handleQueryConventions(arguments)
	case "validate_code":
		return s.handleValidateCode(arguments)
	default:
		return nil, &RPCError{
			Code:    -32601,
			Message: fmt.Sprintf("unknown tool: %s", toolName),
		}
	}
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
