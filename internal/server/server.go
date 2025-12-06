package server

import (
	"context"
	"embed"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/DevSymphony/sym-cli/internal/config"
	"github.com/DevSymphony/sym-cli/internal/converter"
	"github.com/DevSymphony/sym-cli/internal/envutil"
	"github.com/DevSymphony/sym-cli/internal/git"
	"github.com/DevSymphony/sym-cli/internal/github"
	"github.com/DevSymphony/sym-cli/internal/llm"
	"github.com/DevSymphony/sym-cli/internal/policy"
	"github.com/DevSymphony/sym-cli/internal/roles"
	"github.com/DevSymphony/sym-cli/pkg/schema" // symphonyclient integration

	"github.com/pkg/browser"
)

//go:embed static/*
var staticFiles embed.FS

type Server struct {
	port         int
	cfg          *config.Config
	token        *config.Token
	githubClient *github.Client
}

// NewServer creates a new dashboard server
func NewServer(port int) (*Server, error) {
	cfg, err := config.LoadConfig()
	if err != nil {
		return nil, err
	}

	token, err := config.LoadToken()
	if err != nil {
		return nil, err
	}

	githubClient := github.NewClient(cfg.GetGitHubHost(), token.AccessToken)

	return &Server{
		port:         port,
		cfg:          cfg,
		token:        token,
		githubClient: githubClient,
	}, nil
}

// Start starts the web server and opens the browser
func (s *Server) Start() error {
	mux := http.NewServeMux()

	// API endpoints
	mux.HandleFunc("/api/me", s.handleGetMe)
	mux.HandleFunc("/api/roles", s.handleRoles)
	mux.HandleFunc("/api/repo-info", s.handleRepoInfo)

	// Policy API endpoints
	mux.HandleFunc("/api/policy", s.handlePolicy)
	mux.HandleFunc("/api/policy/path", s.handlePolicyPath)
	mux.HandleFunc("/api/policy/templates", s.handlePolicyTemplates)
	mux.HandleFunc("/api/policy/templates/", s.handlePolicyTemplateDetail)
	mux.HandleFunc("/api/policy/convert", s.handleConvert)
	mux.HandleFunc("/api/users", s.handleUsers)

	// Static files
	staticFS, err := fs.Sub(staticFiles, "static")
	if err != nil {
		return err
	}
	mux.Handle("/", http.FileServer(http.FS(staticFS)))

	addr := fmt.Sprintf(":%d", s.port)
	url := fmt.Sprintf("http://localhost:%d", s.port)

	fmt.Printf("Starting dashboard server at %s\n", url)
	fmt.Println("Press Ctrl+C to stop")

	// Open browser
	go func() {
		if err := browser.OpenURL(url); err != nil {
			fmt.Printf("Could not open browser: %v\n", err)
			fmt.Printf("Please manually open: %s\n", url)
		}
	}()

	return http.ListenAndServe(addr, s.corsMiddleware(mux))
}

// corsMiddleware adds CORS headers
func (s *Server) corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func (s *Server) hasPermission(username, permission string) (bool, error) {
	// Load policy to check RBAC permissions
	policyData, err := policy.LoadPolicy(s.cfg.PolicyPath)
	if err != nil {
		return false, fmt.Errorf("failed to load policy: %w", err)
	}

	return s.hasPermissionWithPolicy(username, permission, policyData)
}

func (s *Server) hasPermissionWithPolicy(username, permission string, policyData *schema.UserPolicy) (bool, error) {
	// Load user's role from roles.json
	userRole, err := roles.GetUserRole(username)
	if err != nil {
		return false, fmt.Errorf("failed to get user role: %w", err)
	}

	return s.checkPermissionForRole(userRole, permission, policyData)
}

func (s *Server) hasPermissionWithRoles(username, permission string, rolesData roles.Roles) (bool, error) {
	// Find user's role from the given roles
	userRole := "none"
	for roleName, usernames := range rolesData {
		for _, user := range usernames {
			if user == username {
				userRole = roleName
				break
			}
		}
		if userRole != "none" {
			break
		}
	}

	// Load policy to check RBAC permissions
	policyData, err := policy.LoadPolicy(s.cfg.PolicyPath)
	if err != nil {
		return false, fmt.Errorf("failed to load policy: %w", err)
	}

	return s.checkPermissionForRole(userRole, permission, policyData)
}

// checkPermissionForRole checks if a role has a specific permission in the policy
func (s *Server) checkPermissionForRole(userRole, permission string, policyData *schema.UserPolicy) (bool, error) {
	// Special case: "none" role has no permissions
	if userRole == "none" {
		return false, nil
	}

	// Check if RBAC is defined
	if policyData.RBAC == nil || policyData.RBAC.Roles == nil {
		return false, fmt.Errorf("RBAC not configured")
	}

	// Get role configuration
	role, exists := policyData.RBAC.Roles[userRole]
	if !exists {
		return false, fmt.Errorf("role '%s' not found in RBAC configuration", userRole)
	}

	// Check permission based on type
	switch permission {
	case "editPolicy":
		return role.CanEditPolicy, nil
	case "editRoles":
		return role.CanEditRoles, nil
	default:
		return false, fmt.Errorf("unknown permission type: %s", permission)
	}
}

func (s *Server) handleGetMe(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	user, err := s.githubClient.GetCurrentUser()
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to get user: %v", err), http.StatusInternalServerError)
		return
	}

	// Get user role
	role, err := roles.GetUserRole(user.Login)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to get role: %v", err), http.StatusInternalServerError)
		return
	}

	// Get user permissions
	canEditPolicy, err := s.hasPermission(user.Login, "editPolicy")
	if err != nil {
		// If there's an error checking permissions, default to false
		canEditPolicy = false
	}

	canEditRoles, err := s.hasPermission(user.Login, "editRoles")
	if err != nil {
		// If there's an error checking permissions, default to false
		canEditRoles = false
	}

	response := map[string]interface{}{
		"username": user.Login,
		"name":     user.Name,
		"role":     role,
		"permissions": map[string]bool{
			"canEditPolicy": canEditPolicy,
			"canEditRoles":  canEditRoles,
		},
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(response)
}

// handleRoles handles GET and POST requests for roles
func (s *Server) handleRoles(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		s.handleGetRoles(w, r)
	case http.MethodPost:
		s.handleUpdateRoles(w, r)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// handleGetRoles returns the current roles
func (s *Server) handleGetRoles(w http.ResponseWriter, r *http.Request) {
	rolesData, err := roles.LoadRoles()
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to load roles: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(rolesData)
}

// handleUpdateRoles updates the roles (requires editRoles permission)
func (s *Server) handleUpdateRoles(w http.ResponseWriter, r *http.Request) {
	// Get current user
	user, err := s.githubClient.GetCurrentUser()
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to get user: %v", err), http.StatusInternalServerError)
		return
	}

	// Parse request body first (need to check permission against the NEW roles)
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Failed to read request body", http.StatusBadRequest)
		return
	}

	var newRoles roles.Roles
	if err := json.Unmarshal(body, &newRoles); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	// Check if user has permission to edit roles using the NEW roles
	// (because the user's role might have changed)
	canEdit, err := s.hasPermissionWithRoles(user.Login, "editRoles", newRoles)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to check permissions: %v", err), http.StatusInternalServerError)
		return
	}

	if !canEdit {
		http.Error(w, "Forbidden: You don't have permission to update roles", http.StatusForbidden)
		return
	}

	// Save roles
	if err := roles.SaveRoles(newRoles); err != nil {
		http.Error(w, fmt.Sprintf("Failed to save roles: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]string{
		"status":  "success",
		"message": "Roles updated successfully",
	})
}

func (s *Server) handleRepoInfo(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	owner, repo, err := git.GetRepoInfo()
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to get repo info: %v", err), http.StatusInternalServerError)
		return
	}

	response := map[string]string{
		"owner": owner,
		"repo":  repo,
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(response)
}

// handlePolicy handles GET and POST requests for policy
func (s *Server) handlePolicy(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		s.handleGetPolicy(w, r)
	case http.MethodPost:
		s.handleSavePolicy(w, r)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// handleGetPolicy returns the current policy
func (s *Server) handleGetPolicy(w http.ResponseWriter, r *http.Request) {
	// Get policy path from .sym/.env (or use default)
	policyPath := envutil.GetPolicyPath()
	if policyPath == "" {
		policyPath = ".sym/user-policy.json"
	}
	fmt.Printf("Loading policy from: %s\n", policyPath)

	policyData, err := policy.LoadPolicy(policyPath)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to load policy: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(policyData)
}

// handleSavePolicy saves the policy (requires editPolicy permission)
func (s *Server) handleSavePolicy(w http.ResponseWriter, r *http.Request) {
	// Get current user
	user, err := s.githubClient.GetCurrentUser()
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to get user: %v", err), http.StatusInternalServerError)
		return
	}

	// Parse request body first (need to check permission against the NEW policy)
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Failed to read request body", http.StatusBadRequest)
		return
	}

	var newPolicy schema.UserPolicy
	if err := json.Unmarshal(body, &newPolicy); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	// Check if user has permission to edit policy using the NEW policy
	// (because the user's role might have changed and the new role might only exist in the new policy)
	canEdit, err := s.hasPermissionWithPolicy(user.Login, "editPolicy", &newPolicy)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to check permissions: %v", err), http.StatusInternalServerError)
		return
	}

	if !canEdit {
		http.Error(w, "Forbidden: You don't have permission to update policy", http.StatusForbidden)
		return
	}

	// Get policy path from .sym/.env (or use default)
	policyPath := envutil.GetPolicyPath()
	if policyPath == "" {
		policyPath = ".sym/user-policy.json"
	}
	fmt.Printf("Saving policy to: %s\n", policyPath)

	// Ensure directory exists
	policyDir := filepath.Dir(policyPath)
	if err := os.MkdirAll(policyDir, 0755); err != nil {
		http.Error(w, fmt.Sprintf("Failed to create directory %s: %v", policyDir, err), http.StatusInternalServerError)
		return
	}

	// Save policy
	if err := policy.SavePolicy(&newPolicy, policyPath); err != nil {
		http.Error(w, fmt.Sprintf("Failed to save policy: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]string{
		"status":  "success",
		"message": "Policy saved successfully",
	})
}

// handlePolicyPath handles GET and POST requests for policy path
func (s *Server) handlePolicyPath(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		// Load policy path from .sym/.env
		policyPath := envutil.GetPolicyPath()
		if policyPath == "" {
			// Default to .sym/user-policy.json if not set
			policyPath = ".sym/user-policy.json"
			fmt.Printf("No POLICY_PATH in .sym/.env, using default: %s\n", policyPath)
		} else {
			fmt.Printf("Loaded POLICY_PATH from .sym/.env: %s\n", policyPath)
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{
			"policyPath": policyPath,
		})
	case http.MethodPost:
		s.handleSetPolicyPath(w, r)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// handleSetPolicyPath sets the policy path (requires editPolicy permission)
func (s *Server) handleSetPolicyPath(w http.ResponseWriter, r *http.Request) {
	// Get current user
	user, err := s.githubClient.GetCurrentUser()
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to get user: %v", err), http.StatusInternalServerError)
		return
	}

	// Check if user has permission to edit policy
	canEdit, err := s.hasPermission(user.Login, "editPolicy")
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to check permissions: %v", err), http.StatusInternalServerError)
		return
	}

	if !canEdit {
		http.Error(w, "Forbidden: You don't have permission to change policy path", http.StatusForbidden)
		return
	}

	// Parse request body
	var req struct {
		PolicyPath string `json:"policyPath"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		fmt.Printf("Failed to decode request body: %v\n", err)
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	fmt.Printf("Received policy path from client: '%s' (length: %d)\n", req.PolicyPath, len(req.PolicyPath))

	// Get current policy path from .env
	oldPolicyPath := envutil.GetPolicyPath()
	if oldPolicyPath == "" {
		oldPolicyPath = ".sym/user-policy.json" // default
	}

	// If path is changing and old file exists, move it to new location
	if oldPolicyPath != req.PolicyPath {
		fmt.Printf("Policy path changing from '%s' to '%s'\n", oldPolicyPath, req.PolicyPath)

		if _, err := os.Stat(oldPolicyPath); err == nil {
			fmt.Printf("Moving existing policy file from '%s' to '%s'\n", oldPolicyPath, req.PolicyPath)

			// Create directory for new path
			newDir := filepath.Dir(req.PolicyPath)
			if err := os.MkdirAll(newDir, 0755); err != nil {
				fmt.Printf("Warning: Failed to create directory '%s': %v\n", newDir, err)
			}

			// Read old file
			oldData, err := os.ReadFile(oldPolicyPath)
			if err != nil {
				fmt.Printf("Warning: Failed to read old policy file: %v\n", err)
			} else {
				// Write to new location
				if err := os.WriteFile(req.PolicyPath, oldData, 0644); err != nil {
					fmt.Printf("Warning: Failed to write to new location: %v\n", err)
				} else {
					fmt.Printf("Successfully copied policy to new location\n")

					// Remove old file
					if err := os.Remove(oldPolicyPath); err != nil {
						fmt.Printf("Warning: Failed to remove old policy file: %v\n", err)
					} else {
						fmt.Printf("Successfully removed old policy file\n")
					}
				}
			}
		} else {
			fmt.Printf("Old policy file not found at '%s', skipping move\n", oldPolicyPath)
		}
	}

	// Save to .sym/.env file
	fmt.Printf("Saving policy path to .sym/.env: %s\n", req.PolicyPath)
	if err := envutil.SaveKeyToEnvFile(filepath.Join(".sym", ".env"), "POLICY_PATH", req.PolicyPath); err != nil {
		fmt.Printf("Failed to save policy path: %v\n", err)
		http.Error(w, fmt.Sprintf("Failed to save policy path: %v", err), http.StatusInternalServerError)
		return
	}
	fmt.Printf("Policy path saved successfully to .sym/.env\n")

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]string{
		"status":  "success",
		"message": "Policy path updated successfully",
	})
}

// handlePolicyTemplates returns the list of available templates
func (s *Server) handlePolicyTemplates(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	templates, err := policy.GetTemplates()
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to get templates: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(templates)
}

// handlePolicyTemplateDetail returns a specific template
func (s *Server) handlePolicyTemplateDetail(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract template name from URL path
	templateName := strings.TrimPrefix(r.URL.Path, "/api/policy/templates/")
	if templateName == "" {
		http.Error(w, "Template name required", http.StatusBadRequest)
		return
	}

	template, err := policy.GetTemplate(templateName)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to get template: %v", err), http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(template)
}

// handleUsers returns all users from roles.json
func (s *Server) handleUsers(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	rolesData, err := roles.LoadRoles()
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to load roles: %v", err), http.StatusInternalServerError)
		return
	}

	// Collect all unique users dynamically from all roles
	usersMap := make(map[string]string)

	for roleName, usernames := range rolesData {
		for _, username := range usernames {
			// First role wins (in case a user is in multiple roles)
			if _, exists := usersMap[username]; !exists {
				usersMap[username] = roleName
			}
		}
	}

	// Convert to array
	type UserRole struct {
		Username string `json:"username"`
		Role     string `json:"role"`
	}

	users := make([]UserRole, 0, len(usersMap))
	for username, role := range usersMap {
		users = append(users, UserRole{
			Username: username,
			Role:     role,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(users)
}

// handleConvert runs the convert command to generate linter configs
func (s *Server) handleConvert(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Get current user
	user, err := s.githubClient.GetCurrentUser()
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to get user: %v", err), http.StatusInternalServerError)
		return
	}

	// Check if user has permission to edit policy
	canEdit, err := s.hasPermission(user.Login, "editPolicy")
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to check permissions: %v", err), http.StatusInternalServerError)
		return
	}

	if !canEdit {
		http.Error(w, "Forbidden: You don't have permission to convert policy", http.StatusForbidden)
		return
	}

	fmt.Println("Starting policy conversion...")

	// Get policy path from .env
	policyPath := envutil.GetPolicyPath()
	if policyPath == "" {
		policyPath = ".sym/user-policy.json"
	}

	fmt.Printf("Converting policy from: %s\n", policyPath)

	// Load user policy
	userPolicy, err := policy.LoadPolicy(policyPath)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to load policy: %v", err), http.StatusInternalServerError)
		return
	}

	// Determine output directory (same as input file)
	outputDir := filepath.Dir(policyPath)

	// Setup LLM client (backend auto-selection via @llm)
	timeout := 30 * time.Second
	llmClient := llm.NewClient(
		llm.WithTimeout(timeout),
	)

	// Create converter with LLM client and output directory
	conv := converter.NewConverter(llmClient, outputDir)

	// Setup context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(30*10)*time.Second)
	defer cancel()

	// Convert using new API
	convResult, err := conv.Convert(ctx, userPolicy)
	if err != nil {
		http.Error(w, fmt.Sprintf("Conversion failed: %v", err), http.StatusInternalServerError)
		return
	}

	// Files are already written by converter
	filesWritten := []string{}
	for _, filePath := range convResult.GeneratedFiles {
		// Extract just the filename
		filesWritten = append(filesWritten, filepath.Base(filePath))
	}

	result := map[string]interface{}{
		"status":       "success",
		"policyPath":   policyPath,
		"outputDir":    outputDir,
		"filesWritten": filesWritten,
		"message":      fmt.Sprintf("Conversion completed: %d files written", len(filesWritten)),
	}

	if len(convResult.Warnings) > 0 {
		result["warnings"] = convResult.Warnings
	}

	fmt.Printf("Conversion completed: %d files written\n", len(filesWritten))

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(result)
}
