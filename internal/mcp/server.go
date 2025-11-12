package mcp

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/DevSymphony/sym-cli/internal/policy"
	"github.com/DevSymphony/sym-cli/internal/validator"
	"github.com/DevSymphony/sym-cli/pkg/schema"
)

// Server is a MCP (Model Context Protocol) server.
// It communicates via JSON-RPC over stdio or HTTP.
type Server struct {
	host       string
	port       int
	configPath string
	userPolicy *schema.UserPolicy  // Original user-written policy
	codePolicy *schema.CodePolicy  // Converted validation policy
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
	if s.configPath != "" {
		// Determine the directory and try to load both user-policy and code-policy
		dir := filepath.Dir(s.configPath)

		// Try to load user-policy.json for natural language descriptions
		userPolicyPath := filepath.Join(dir, "user-policy.json")
		if userPolicy, err := s.loader.LoadUserPolicy(userPolicyPath); err == nil {
			s.userPolicy = userPolicy
			fmt.Fprintf(os.Stderr, "✓ User policy loaded: %s (%d rules)\n", userPolicyPath, len(userPolicy.Rules))
		}

		// Try to load code-policy.json for validation
		codePolicyPath := filepath.Join(dir, "code-policy.json")
		if codePolicy, err := s.loader.LoadCodePolicy(codePolicyPath); err == nil {
			s.codePolicy = codePolicy
			fmt.Fprintf(os.Stderr, "✓ Code policy loaded: %s (%d rules)\n", codePolicyPath, len(codePolicy.Rules))
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

	return s.handleRequests(os.Stdin, os.Stdout)
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

// handleRequests handles incoming requests via stdio.
func (s *Server) handleRequests(in io.Reader, out io.Writer) error {
	scanner := bufio.NewScanner(in)
	encoder := json.NewEncoder(out)

	for scanner.Scan() {
		line := scanner.Bytes()

		var req JSONRPCRequest
		if err := json.Unmarshal(line, &req); err != nil {
			continue
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

		if err := encoder.Encode(resp); err != nil {
			return fmt.Errorf("failed to encode response: %w", err)
		}
	}

	return scanner.Err()
}

// QueryConventionsRequest is a request to query conventions.
type QueryConventionsRequest struct {
	Category  string   `json:"category"` // filter by category
	Files     []string `json:"files"`
	Languages []string `json:"languages"`
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

	conventions := s.filterConventions(req)

	return map[string]interface{}{
		"conventions": conventions,
		"total":       len(conventions),
		"next_step":   "Now implement your code following these conventions. After completion, MUST call validate_code to verify compliance.",
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
			if len(req.Languages) > 0 && len(rule.Languages) > 0 {
				if !containsAny(rule.Languages, req.Languages) {
					continue
				}
			}

			conventions = append(conventions, ConventionItem{
				ID:          rule.ID,
				Category:    rule.Category,
				Description: rule.Say,      // Use natural language description
				Message:     rule.Message,
				Severity:    rule.Severity,
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

	if len(rule.When.Include) > 0 && len(req.Files) > 0 {
		matched := false
		for _, file := range req.Files {
			for _, pattern := range rule.When.Include {
				if match, _ := filepath.Match(pattern, file); match {
					matched = true
					break
				}
			}
			if matched {
				break
			}
		}
		if !matched {
			return false
		}
	}

	if len(rule.When.Exclude) > 0 && len(req.Files) > 0 {
		for _, file := range req.Files {
			for _, pattern := range rule.When.Exclude {
				if match, _ := filepath.Match(pattern, file); match {
					return false
				}
			}
		}
	}

	return true
}

// ValidateCodeRequest is a code validation request.
type ValidateCodeRequest struct {
	Files []string `json:"files"` // file paths to validate
	Role  string   `json:"role"`  // RBAC role
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
// It uses the existing validator to validate code.
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

	if len(req.Files) == 0 {
		req.Files = []string{"."}
	}

	v := validator.NewValidator(validationPolicy, false) // verbose = false for MCP

	var allViolations []ViolationItem
	var hasErrors bool

	for _, filePath := range req.Files {
		result, err := v.Validate(filePath)
		if err != nil {
			return nil, &RPCError{
				Code:    -32000,
				Message: fmt.Sprintf("validation failed: %v", err),
			}
		}

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
	}

	var message string
	if hasErrors {
		message = "VALIDATION FAILED: Found error-level violations. You MUST fix these issues and re-validate before proceeding."
	} else if len(allViolations) > 0 {
		message = "VALIDATION WARNING: Found non-critical violations. Consider fixing these warnings for better code quality."
	} else {
		message = "✓ VALIDATION PASSED: Code complies with all conventions. Task can be marked as complete."
	}

	return map[string]interface{}{
		"valid":      !hasErrors,
		"violations": allViolations,
		"total":      len(allViolations),
		"message":    message,
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
			"description": `[STEP 1 - ALWAYS CALL FIRST] Query coding conventions and best practices before writing any code.

CRITICAL WORKFLOW:
1. ALWAYS call this tool FIRST when starting any coding task
2. Query relevant conventions by category (security, style, architecture, etc.)
3. Query conventions for specific files/languages you'll be working with
4. Use the returned conventions to guide your code implementation

This ensures your code follows project standards from the start. Never skip this step.

Categories available: security, style, documentation, error_handling, architecture, performance, testing

Example: Before implementing authentication, query security conventions first.`,
			"inputSchema": map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"category": map[string]interface{}{
						"type":        "string",
						"description": "Filter by category (security, style, documentation, error_handling, architecture, performance, testing)",
					},
					"files": map[string]interface{}{
						"type":        "array",
						"items":       map[string]string{"type": "string"},
						"description": "File paths to check conventions for",
					},
					"languages": map[string]interface{}{
						"type":        "array",
						"items":       map[string]string{"type": "string"},
						"description": "Programming languages to filter by (go, javascript, python, java, etc.)",
					},
				},
			},
		},
		{
			"name": "validate_code",
			"description": `[STEP 3 - ALWAYS CALL LAST] Validate that your code complies with all project conventions.

CRITICAL WORKFLOW:
1. Call this tool AFTER you have written or modified code
2. MANDATORY: Always validate before considering the task complete
3. If violations are found, fix them and validate again
4. Only mark the task as done after validation passes with no errors

This is the final quality gate. Never skip this validation step.

The tool will check:
- Security violations (hardcoded secrets, SQL injection, XSS, etc.)
- Style violations (formatting, naming, documentation)
- Architecture violations (separation of concerns, patterns)
- Error handling violations (missing error checks, empty catch blocks)

If violations are found, you MUST fix them before proceeding.`,
			"inputSchema": map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"files": map[string]interface{}{
						"type":        "array",
						"items":       map[string]string{"type": "string"},
						"description": "File paths to validate (required)",
						"required":    true,
					},
					"role": map[string]interface{}{
						"type":        "string",
						"description": "RBAC role for validation (optional)",
					},
				},
				"required": []string{"files"},
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
