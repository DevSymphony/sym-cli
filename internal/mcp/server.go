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
	policy     *schema.CodePolicy
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
		codePolicy, err := s.loader.LoadCodePolicy(s.configPath)
		if err != nil {
			return fmt.Errorf("failed to load policy: %w", err)
		}
		s.policy = codePolicy
		fmt.Fprintf(os.Stderr, "policy loaded: %s\n", s.configPath)
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
	if s.policy == nil {
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
	}, nil
}

// filterConventions filters conventions that match the request.
func (s *Server) filterConventions(req QueryConventionsRequest) []ConventionItem {
	var conventions []ConventionItem

	for _, rule := range s.policy.Rules {
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
	if s.policy == nil {
		return nil, &RPCError{
			Code:    -32000,
			Message: "policy not loaded",
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

	v := validator.NewValidator(s.policy, false) // verbose = false for MCP

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

	return map[string]interface{}{
		"valid":      !hasErrors,
		"violations": allViolations,
		"total":      len(allViolations),
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
