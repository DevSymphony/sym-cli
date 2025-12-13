package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/AlecAivazis/survey/v2"
)

// Section markers for Symphony instructions in append-mode files (CLAUDE.md)
const (
	symphonySectionStart = "<!-- SYMPHONY:START -->"
	symphonySectionEnd   = "<!-- SYMPHONY:END -->"
)

// MCPRegistrationConfig represents the MCP configuration structure
// Used for Claude Desktop, Claude Code, Cursor, Cline (mcpServers format)
type MCPRegistrationConfig struct {
	MCPServers map[string]MCPServerConfig `json:"mcpServers"`
}

// VSCodeMCPConfig represents the VS Code MCP configuration structure
type VSCodeMCPConfig struct {
	Servers map[string]VSCodeServerConfig `json:"servers"`
	Inputs  []interface{}                 `json:"inputs,omitempty"`
}

// MCPServerConfig represents a single MCP server configuration
// Used for Claude Desktop, Claude Code, Cursor, Cline (mcpServers format)
type MCPServerConfig struct {
	Type    string            `json:"type,omitempty"` // Optional for Claude Code, recommended for Cursor
	Command string            `json:"command"`
	Args    []string          `json:"args"`
	Env     map[string]string `json:"env,omitempty"`
}

// VSCodeServerConfig represents VS Code MCP server configuration
type VSCodeServerConfig struct {
	Type    string            `json:"type"`
	Command string            `json:"command"`
	Args    []string          `json:"args"`
	Env     map[string]string `json:"env,omitempty"`
}

// MCP tool options for multi-select
var mcpToolOptions = []string{
	"Claude Code",
	"Cursor",
	"VS Code Copilot",
}

// mcpToolToApp maps display name to internal app identifier
var mcpToolToApp = map[string]string{
	"Claude Code":     "claude-code",
	"Cursor":          "cursor",
	"VS Code Copilot": "vscode",
}

// getNpmPackageRef returns the npm package reference with version
// Returns "@dev-symphony/sym@latest" for dev builds, "@dev-symphony/sym@<version>" otherwise
func getNpmPackageRef() string {
	v := GetVersion()
	if v == "dev" || v == "" {
		return "@dev-symphony/sym@latest"
	}
	return fmt.Sprintf("@dev-symphony/sym@%s", v)
}

// promptMCPRegistration prompts user to register Symphony as MCP server
func promptMCPRegistration() {
	// Check if npx is available
	if !checkNpxAvailable() {
		printWarn("'npx' not found. MCP features require Node.js.")
		fmt.Println(indent("Download: https://nodejs.org/"))

		var continueAnyway bool
		prompt := &survey.Confirm{
			Message: "Continue anyway?",
			Default: false,
		}
		if err := survey.AskOne(prompt, &continueAnyway); err != nil || !continueAnyway {
			fmt.Println("Skipped MCP registration")
			return
		}
	}

	fmt.Println()
	printTitle("MCP", "Register Symphony as an MCP server")
	fmt.Println(indent("Symphony MCP provides code convention tools for AI assistants"))
	fmt.Println()

	// Use Select with toggle behavior - Enter toggles selection, "Submit" confirms
	selectedTools := selectToolsWithEnterToggle(mcpToolOptions)

	// If no tools selected, skip
	if len(selectedTools) == 0 {
		fmt.Println("Skipped MCP registration")
		fmt.Println(indent("Tip: Run 'sym mcp register' to register MCP later"))
		return
	}

	// Register selected tools
	var registered []string
	var failed []string

	for _, tool := range selectedTools {
		app := mcpToolToApp[tool]
		if err := registerMCP(app); err != nil {
			failed = append(failed, fmt.Sprintf("%s: %v", tool, err))
		} else {
			registered = append(registered, tool)
		}
	}

	// Print results
	fmt.Println()
	if len(registered) > 0 {
		printOK(fmt.Sprintf("Registered: %s", strings.Join(registered, ", ")))
		fmt.Println(indent("Reload/restart the tools to use Symphony"))
	}
	for _, f := range failed {
		printError(fmt.Sprintf("Failed to register %s", f))
	}
}

// registerMCP registers Symphony as an MCP server for the specified app
func registerMCP(app string) error {
	configPath := getMCPConfigPath(app)

	if configPath == "" {
		fmt.Println(warn(fmt.Sprintf("%s config path could not be determined", getAppDisplayName(app))))
		return fmt.Errorf("config path not determined")
	}

	// All supported apps are now project-specific
	fmt.Println(indent(fmt.Sprintf("Configuring %s", getAppDisplayName(app))))
	fmt.Println(indent(fmt.Sprintf("Location: %s", configPath)))

	// Create config directory if it doesn't exist
	configDir := filepath.Dir(configPath)
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	// Read existing config and handle different formats
	existingData, err := os.ReadFile(configPath)
	fileExists := err == nil

	var data []byte

	if app == "vscode" {
		// VS Code Copilot uses different MCP config format
		var vscodeConfig VSCodeMCPConfig

		if fileExists {
			if err := json.Unmarshal(existingData, &vscodeConfig); err != nil {
				// Invalid JSON, create backup and start fresh
				backupPath := configPath + ".bak"
				if err := os.WriteFile(backupPath, existingData, 0644); err != nil {
					fmt.Println(indent(fmt.Sprintf("Failed to create backup: %v", err)))
				} else {
					fmt.Println(indent(fmt.Sprintf("Invalid JSON, backup created: %s", filepath.Base(backupPath))))
				}
				vscodeConfig = VSCodeMCPConfig{}
			}
			// Valid JSON: no backup needed, just update symphony entry
		} else {
			fmt.Println(indent("Creating new configuration file"))
		}

		// Initialize Servers if nil
		if vscodeConfig.Servers == nil {
			vscodeConfig.Servers = make(map[string]VSCodeServerConfig)
		}

		// Add/update Symphony server
		vscodeConfig.Servers["symphony"] = VSCodeServerConfig{
			Type:    "stdio",
			Command: "npx",
			Args:    []string{"-y", getNpmPackageRef(), "mcp"},
		}

		// Marshal
		data, err = json.MarshalIndent(vscodeConfig, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal config: %w", err)
		}
	} else {
		// Claude Desktop, Claude Code, Cursor, Cline use mcpServers format
		var config MCPRegistrationConfig

		if fileExists {
			if err := json.Unmarshal(existingData, &config); err != nil {
				// Invalid JSON, create backup and start fresh
				backupPath := configPath + ".bak"
				if err := os.WriteFile(backupPath, existingData, 0644); err != nil {
					fmt.Println(indent(fmt.Sprintf("Failed to create backup: %v", err)))
				} else {
					fmt.Println(indent(fmt.Sprintf("Invalid JSON, backup created: %s", filepath.Base(backupPath))))
				}
				config = MCPRegistrationConfig{}
			}
			// Valid JSON: no backup needed, just update symphony entry
		} else {
			fmt.Println(indent("Creating new configuration file"))
		}

		// Initialize MCPServers if nil
		if config.MCPServers == nil {
			config.MCPServers = make(map[string]MCPServerConfig)
		}

		// Add/update Symphony server
		serverConfig := MCPServerConfig{
			Command: "npx",
			Args:    []string{"-y", getNpmPackageRef(), "mcp"},
		}

		// For Cursor, add type field
		if app == "cursor" {
			serverConfig.Type = "stdio"
		}

		config.MCPServers["symphony"] = serverConfig

		// Marshal
		data, err = json.MarshalIndent(config, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal config: %w", err)
		}
	}

	// Write config
	if err := os.WriteFile(configPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write config: %w", err)
	}

	fmt.Println(indent("Symphony MCP server registered"))

	// Create instructions file for all supported apps
	if err := createInstructionsFile(app); err != nil {
		fmt.Println(indent(fmt.Sprintf("Failed to create instructions file: %v", err)))
	}

	return nil
}

// getMCPConfigPath returns the MCP config file path for the specified app
func getMCPConfigPath(app string) string {
	// For project-specific configs, get current working directory (project root)
	cwd, _ := os.Getwd()

	switch app {
	case "claude-code":
		return filepath.Join(cwd, ".mcp.json")
	case "cursor":
		return filepath.Join(cwd, ".cursor", "mcp.json")
	case "vscode":
		return filepath.Join(cwd, ".vscode", "mcp.json")
	default:
		return ""
	}
}

// getAppDisplayName returns the display name for the app
func getAppDisplayName(app string) string {
	switch app {
	case "claude-code":
		return "Claude Code"
	case "cursor":
		return "Cursor"
	case "vscode":
		return "VS Code Copilot"
	default:
		return app
	}
}

// checkNpxAvailable checks if npx is available in PATH
func checkNpxAvailable() bool {
	_, err := exec.LookPath("npx")
	return err == nil
}

// updateSymphonySection updates or appends the Symphony section in content
func updateSymphonySection(existingContent, symphonyContent string) string {
	startIdx := strings.Index(existingContent, symphonySectionStart)
	endIdx := strings.Index(existingContent, symphonySectionEnd)

	// Case 1: Symphony section exists → replace
	if startIdx != -1 && endIdx != -1 && endIdx > startIdx {
		endIdx += len(symphonySectionEnd)
		return existingContent[:startIdx] + symphonyContent + existingContent[endIdx:]
	}

	// Case 2: No Symphony section → append at end
	if existingContent == "" {
		return symphonyContent
	}

	// Ensure proper spacing
	separator := "\n\n"
	if strings.HasSuffix(existingContent, "\n\n") {
		separator = ""
	} else if strings.HasSuffix(existingContent, "\n") {
		separator = "\n"
	}

	return existingContent + separator + symphonyContent
}

// createInstructionsFile creates or updates the instructions file for the specified app.
// Logic is consistent with MCP config handling:
// - File doesn't exist → create new
// - File exists → update symphony content (no backup for Symphony-dedicated files)
// - Shared files (CLAUDE.md) use section markers to preserve other content
func createInstructionsFile(app string) error {
	var instructionsPath string
	var content string
	var isSharedFile bool // true for files that may contain other content (CLAUDE.md)

	switch app {
	case "claude-code":
		instructionsPath = "CLAUDE.md"
		content = getClaudeCodeInstructions()
		isSharedFile = true // CLAUDE.md may have other project instructions
	case "cursor":
		instructionsPath = filepath.Join(".cursor", "rules", "symphony.mdc")
		content = getCursorInstructions()
		isSharedFile = false // Symphony-dedicated file
	case "vscode":
		instructionsPath = filepath.Join(".github", "instructions", "symphony.instructions.md")
		content = getVSCodeInstructions()
		isSharedFile = false // Symphony-dedicated file
	default:
		return nil
	}

	// Create directory if needed
	dir := filepath.Dir(instructionsPath)
	if dir != "." {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed to create directory: %w", err)
		}
	}

	// Check if file exists
	existingContent, err := os.ReadFile(instructionsPath)
	fileExists := err == nil

	if fileExists && isSharedFile {
		// Shared file: update/append Symphony section only, preserve other content
		existingStr := string(existingContent)
		content = updateSymphonySection(existingStr, content)

		if strings.Contains(existingStr, symphonySectionStart) {
			fmt.Println(indent(fmt.Sprintf("Updated Symphony section in %s", instructionsPath)))
		} else {
			fmt.Println(indent(fmt.Sprintf("Appended Symphony section to %s", instructionsPath)))
		}
	} else if fileExists {
		// Symphony-dedicated file: just overwrite (no backup needed)
		fmt.Println(indent(fmt.Sprintf("Updated %s", instructionsPath)))
	} else {
		// New file
		fmt.Println(indent(fmt.Sprintf("Created %s", instructionsPath)))
	}

	// Write file
	if err := os.WriteFile(instructionsPath, []byte(content), 0644); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	// Add VS Code instructions directory to .gitignore
	if app == "vscode" {
		if err := ensureGitignore(".github/instructions/"); err != nil {
			fmt.Println(indent(fmt.Sprintf("Warning: Failed to update .gitignore: %v", err)))
		} else {
			fmt.Println(indent("Added .github/instructions/ to .gitignore"))
		}
	}

	return nil
}

// getClaudeCodeInstructions returns instructions for Claude Code (CLAUDE.md)
func getClaudeCodeInstructions() string {
	return symphonySectionStart + `
# Symphony Code Conventions

**This project uses Symphony MCP for automated code convention management.**

## Critical Requirements

### 1. Before Writing Code

**Check MCP Status**: Verify Symphony MCP server is active. If unavailable, warn the user and do not proceed.

**Query Categories First**: Use ` + "`mcp__symphony__list_category`" + ` to get available categories.
- **IMPORTANT**: Do NOT invent category names. Only use categories returned by list_category.

**Query Conventions**: Use ` + "`mcp__symphony__list_convention`" + ` with a category from list_category.
- Filter by languages as needed

**After Updating Rules/Categories**: If you add/edit/remove conventions or categories, run ` + "`mcp__symphony__convert`" + ` to regenerate derived policy and linter configs (then re-run validation if needed).

### 2. After Writing Code

**Validate Changes**: Always run ` + "`mcp__symphony__validate_code`" + ` to check all changes against project conventions.

**Fix Violations**: Address any issues found before committing.

## Workflow

1. Verify Symphony MCP is active
2. Query categories (list_category)
3. Query conventions with valid category (list_convention)
4. Write code
5. Validate with Symphony
6. Fix violations
7. Commit
` + symphonySectionEnd + "\n"
}

// getCursorInstructions returns instructions for Cursor (.cursor/rules/symphony.mdc)
func getCursorInstructions() string {
	return `---
description: Symphony code convention validation
globs:
  - "**/*"
alwaysApply: true
---

# Symphony Code Conventions

**This project uses Symphony MCP for code convention management.**

## Required Steps

### Before Code Generation
1. **Verify Symphony MCP is active** - If not available, stop and warn user
2. **Query categories first** - Use ` + "`symphony/list_category`" + ` to get available categories
   - **IMPORTANT**: Do NOT invent category names. Only use categories returned by list_category.
3. **Query conventions** - Use ` + "`symphony/list_convention`" + ` with a category from step 2
4. **After updating conventions/categories** - Use ` + "`symphony/convert`" + ` to regenerate derived policy and linter configs

### After Code Generation
1. **Validate all changes** - Use ` + "`symphony/validate_code`" + `
2. **Fix violations** - Address issues before committing

---

*Auto-generated by Symphony*
`
}

// getVSCodeInstructions returns instructions for VS Code Copilot (.github/instructions/*.instructions.md)
func getVSCodeInstructions() string {
	return `---
description: Symphony code convention validation
name: Symphony Conventions
applyTo: "**"
---

# Symphony Code Conventions

This project uses Symphony MCP for automated code convention management.

## Critical Requirements

### Before Writing Code
1. Verify Symphony MCP server is active. If not available, warn user and stop.
2. Query available categories using symphony/list_category tool.
   - **IMPORTANT**: Do NOT invent category names. Only use categories returned by list_category.
3. Query relevant conventions using symphony/list_convention tool with a category from step 2.
   - Filter by programming language as needed
4. If you add/edit/remove conventions or categories, run symphony/convert (then validate again if needed).

### After Writing Code
1. Always validate changes using symphony/validate_code tool (validates all git changes)
2. Fix any violations found
3. Only commit after validation passes

## Workflow
Check MCP → Query Categories → Query Conventions → Write Code → Validate → Fix → Commit

---
Auto-generated by Symphony
`
}

// selectToolsWithEnterToggle allows users to select tools using Enter key to toggle
// and "Submit" option to confirm selection
func selectToolsWithEnterToggle(tools []string) []string {
	selected := make(map[string]bool)
	lastChoice := "" // Track last selected option to maintain cursor position

	// Use custom template to hide message output
	restore := useSelectTemplateNoMessage()
	defer restore()

	// Print header once with cyan hint
	fmt.Printf("Select tools to integrate:  %s\n", colorize(cyan, "[Enter: toggle]"))

	for {
		// Count selected items
		count := 0
		for _, v := range selected {
			if v {
				count++
			}
		}

		// Build submit option with count
		var submitOption string
		if count > 0 {
			submitOption = fmt.Sprintf("✓ Submit (%d selected)", count)
		} else {
			submitOption = "✓ Submit"
		}

		// Build options with selection indicators
		options := make([]string, 0, len(tools)+1)
		for _, tool := range tools {
			if selected[tool] {
				options = append(options, fmt.Sprintf("[x] %s", tool))
			} else {
				options = append(options, fmt.Sprintf("[ ] %s", tool))
			}
		}
		options = append(options, submitOption)

		// Find default option index based on last choice
		defaultOption := options[0]
		if lastChoice != "" {
			for _, opt := range options {
				// Match by tool name (ignore [x]/[ ] prefix and submit option changes)
				if strings.HasPrefix(lastChoice, "✓") && strings.HasPrefix(opt, "✓") {
					defaultOption = opt
					break
				}
				for _, tool := range tools {
					if strings.Contains(lastChoice, tool) && strings.Contains(opt, tool) {
						defaultOption = opt
						break
					}
				}
			}
		}

		// Show selection prompt
		var choice string
		prompt := &survey.Select{
			Message: "",
			Options: options,
			Default: defaultOption,
		}

		if err := survey.AskOne(prompt, &choice); err != nil {
			// User cancelled
			return nil
		}

		lastChoice = choice

		// Check if Submit was selected
		if strings.HasPrefix(choice, "✓ Submit") {
			break
		}

		// Toggle the selected tool
		for _, tool := range tools {
			if strings.Contains(choice, tool) {
				selected[tool] = !selected[tool]
				break
			}
		}
	}

	// Collect selected tools
	var result []string
	for _, tool := range tools {
		if selected[tool] {
			result = append(result, tool)
		}
	}
	return result
}
