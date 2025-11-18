package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/manifoldco/promptui"
)

// MCPRegistrationConfig represents the MCP configuration structure
// Used for Claude Code, Cursor
type MCPRegistrationConfig struct {
	MCPServers map[string]MCPServerConfig `json:"mcpServers"`
}

// VSCodeMCPConfig represents the VS Code MCP configuration structure
type VSCodeMCPConfig struct {
	Servers map[string]VSCodeServerConfig `json:"servers"`
	Inputs  []interface{}                 `json:"inputs,omitempty"`
}

// MCPServerConfig represents a single MCP server configuration
// Used for Claude Code, Cursor
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

// promptMCPRegistration prompts user to register Symphony as MCP server
func promptMCPRegistration() {
	// Check if npx is available
	if !checkNpxAvailable() {
		fmt.Println("\n⚠ Warning: 'npx' not found. MCP features require Node.js.")
		fmt.Println("  Download: https://nodejs.org/")

		confirmPrompt := promptui.Prompt{
			Label:     "Continue anyway",
			IsConfirm: true,
		}

		result, err := confirmPrompt.Run()
		if err != nil || strings.ToLower(result) != "y" {
			fmt.Println("Skipped MCP registration")
			return
		}
	}

	fmt.Println("\n📡 Would you like to register Symphony as an MCP server?")
	fmt.Println("   (Symphony MCP provides code convention tools for AI assistants)")
	fmt.Println()

	// Create selection prompt
	items := []string{
		"Claude Desktop (global)",
		"Claude Code (project)",
		"Cursor (project)",
		"VS Code Copilot (project)",
		"Cline (project)",
		"All",
		"Skip",
	}

	templates := &promptui.SelectTemplates{
		Label:    "{{ . }}?",
		Active:   "▸ {{ . | cyan }}",
		Inactive: "  {{ . }}",
		Selected: "✓ {{ . | green }}",
	}

	prompt := promptui.Select{
		Label:     "Select option",
		Items:     items,
		Templates: templates,
		Size:      6,
	}

	index, _, err := prompt.Run()
	if err != nil {
		fmt.Println("\nSkipped MCP registration")
		return
	}

	switch index {
	case 0: // Claude Desktop (global)
		if err := registerMCP("claude-desktop"); err != nil {
			fmt.Printf("❌ Failed to register Claude Desktop: %v\n", err)
		} else {
			fmt.Println("\n✅ MCP registration complete! Restart Claude Desktop to use Symphony.")
		}
	case 1: // Claude Code (project)
		if err := registerMCP("claude-code"); err != nil {
			fmt.Printf("❌ Failed to register Claude Code: %v\n", err)
		} else {
			fmt.Println("\n✅ MCP registration complete! Reload Claude Code to use Symphony.")
		}
	case 2: // Cursor (project)
		if err := registerMCP("cursor"); err != nil {
			fmt.Printf("❌ Failed to register Cursor: %v\n", err)
		} else {
			fmt.Println("\n✅ MCP registration complete! Reload Cursor to use Symphony.")
		}
	case 3: // VS Code/Cline (project)
		if err := registerMCP("vscode"); err != nil {
			fmt.Printf("❌ Failed to register VS Code: %v\n", err)
		} else {
			fmt.Println("\n✅ MCP registration complete! Reload VS Code to use Symphony.")
		}
	case 4: // All
		apps := []string{"claude-desktop", "claude-code", "cursor", "vscode"}
		successCount := 0
		for _, app := range apps {
			if registerMCP(app) == nil {
				successCount++
			}
		}
		if successCount > 0 {
			fmt.Printf("\n✅ MCP registration complete! Registered to %d app(s).\n", successCount)
			fmt.Println("   Restart/reload the apps to use Symphony.")
		}
	case 5: // Skip
		fmt.Println("Skipped MCP registration")
		fmt.Println("\n💡 Tip: Run 'sym init --register-mcp' to register MCP later")
	}
}

// registerMCP registers Symphony as an MCP server for the specified app
func registerMCP(app string) error {
	configPath := getMCPConfigPath(app)

	if configPath == "" {
		fmt.Printf("\n⚠ %s config path could not be determined\n", getAppDisplayName(app))
		return fmt.Errorf("config path not determined")
	}

	// Check if this is a project-specific config
	isProjectConfig := app != "claude-desktop"

	if isProjectConfig {
		fmt.Printf("\n✓ Configuring %s (project-specific)\n", getAppDisplayName(app))
	} else {
		fmt.Printf("\n✓ Configuring %s (global)\n", getAppDisplayName(app))
	}
	fmt.Printf("  Location: %s\n", configPath)

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
		// VS Code uses different format
		var vscodeConfig VSCodeMCPConfig

		if fileExists {
			if err := json.Unmarshal(existingData, &vscodeConfig); err != nil {
				// Invalid JSON, create backup
				backupPath := configPath + ".bak"
				if err := os.WriteFile(backupPath, existingData, 0644); err != nil {
					fmt.Printf("  ⚠ Failed to create backup: %v\n", err)
				} else {
					fmt.Printf("  ⚠ Invalid JSON, backup created: %s\n", filepath.Base(backupPath))
				}
				vscodeConfig = VSCodeMCPConfig{}
			} else {
				// Valid JSON, create backup
				backupPath := configPath + ".bak"
				if err := os.WriteFile(backupPath, existingData, 0644); err != nil {
					fmt.Printf("  ⚠ Failed to create backup: %v\n", err)
				} else {
					fmt.Printf("  Backup: %s\n", filepath.Base(backupPath))
				}
			}
		} else {
			fmt.Printf("  Creating new configuration file\n")
		}

		// Initialize Servers if nil
		if vscodeConfig.Servers == nil {
			vscodeConfig.Servers = make(map[string]VSCodeServerConfig)
		}

		// Add/update Symphony server
		vscodeConfig.Servers["symphony"] = VSCodeServerConfig{
			Type:    "stdio",
			Command: "npx",
			Args:    []string{"-y", "@dev-symphony/sym@latest", "mcp"},
		}

		// Marshal
		data, err = json.MarshalIndent(vscodeConfig, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal config: %w", err)
		}
	} else {
		// Claude Code, Cursor use standard format
		var config MCPRegistrationConfig

		if fileExists {
			if err := json.Unmarshal(existingData, &config); err != nil {
				// Invalid JSON, create backup
				backupPath := configPath + ".bak"
				if err := os.WriteFile(backupPath, existingData, 0644); err != nil {
					fmt.Printf("  ⚠ Failed to create backup: %v\n", err)
				} else {
					fmt.Printf("  ⚠ Invalid JSON, backup created: %s\n", filepath.Base(backupPath))
				}
				config = MCPRegistrationConfig{}
			} else {
				// Valid JSON, create backup
				backupPath := configPath + ".bak"
				if err := os.WriteFile(backupPath, existingData, 0644); err != nil {
					fmt.Printf("  ⚠ Failed to create backup: %v\n", err)
				} else {
					fmt.Printf("  Backup: %s\n", filepath.Base(backupPath))
				}
			}
		} else {
			fmt.Printf("  Creating new configuration file\n")
		}

		// Initialize MCPServers if nil
		if config.MCPServers == nil {
			config.MCPServers = make(map[string]MCPServerConfig)
		}

		// Add/update Symphony server
		serverConfig := MCPServerConfig{
			Command: "npx",
			Args:    []string{"-y", "@dev-symphony/sym@latest", "mcp"},
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

	fmt.Printf("  ✓ Symphony MCP server registered\n")

	// Create instructions file for project-specific configs
	if isProjectConfig {
		if err := createInstructionsFile(app); err != nil {
			fmt.Printf("  ⚠ Failed to create instructions file: %v\n", err)
		}
	}

	return nil
}

// getMCPConfigPath returns the MCP config file path for the specified app
func getMCPConfigPath(app string) string {
	homeDir, _ := os.UserHomeDir()

	// For project-specific configs, get current working directory (project root)
	cwd, _ := os.Getwd()

	var path string

	switch app {
	case "claude-desktop":
		// Global configuration
		switch runtime.GOOS {
		case "windows":
			path = filepath.Join(os.Getenv("APPDATA"), "Claude", "claude_desktop_config.json")
		case "darwin":
			path = filepath.Join(homeDir, "Library", "Application Support", "Claude", "claude_desktop_config.json")
		case "linux":
			path = filepath.Join(homeDir, ".config", "Claude", "claude_desktop_config.json")
		}
	case "claude-code":
		// Project-specific configuration
		path = filepath.Join(cwd, ".mcp.json")
	case "cursor":
		// Project-specific configuration
		path = filepath.Join(cwd, ".cursor", "mcp.json")
	case "vscode":
		// Project-specific configuration
		path = filepath.Join(cwd, ".vscode", "mcp.json")
	}

	return path
}

// getAppDisplayName returns the display name for the app
func getAppDisplayName(app string) string {
	switch app {
	case "claude-desktop":
		return "Claude Desktop"
	case "claude-code":
		return "Claude Code"
	case "cursor":
		return "Cursor"
	case "vscode":
		return "VS Code/Cline"
	default:
		return app
	}
}

// checkNpxAvailable checks if npx is available in PATH
func checkNpxAvailable() bool {
	_, err := exec.LookPath("npx")
	return err == nil
}

// createInstructionsFile creates or updates the instructions file for the specified app
func createInstructionsFile(app string) error {
	var instructionsPath string
	var content string
	var appendMode bool

	switch app {
	case "claude-code":
		instructionsPath = "claude.md"
		content = getClaudeCodeInstructions()
		appendMode = true
	case "cursor":
		// Use new .cursor/rules format
		instructionsPath = filepath.Join(".cursor", "rules", "symphony.mdc")
		content = getCursorInstructions()
		appendMode = false
	case "vscode":
		// Use .github/instructions/symphony.instructions.md for VS Code Copilot
		instructionsPath = filepath.Join(".github", "instructions", "symphony.instructions.md")
		content = getVSCodeInstructions()
		appendMode = false
	default:
		return nil // No instructions file for this app
	}

	// Check if file exists
	existingContent, err := os.ReadFile(instructionsPath)
	fileExists := err == nil

	if fileExists {
		if appendMode {
			// Check if Symphony instructions already exist
			if strings.Contains(string(existingContent), "# Symphony Code Conventions") {
				fmt.Printf("  ✓ Instructions already exist in %s\n", instructionsPath)
				return nil
			}
			// Append to existing file
			content = string(existingContent) + "\n\n" + content
			fmt.Printf("  ✓ Appended Symphony instructions to %s\n", instructionsPath)
		} else {
			// Create backup
			backupPath := instructionsPath + ".bak"
			if err := os.WriteFile(backupPath, existingContent, 0644); err != nil {
				fmt.Printf("  ⚠ Failed to create backup: %v\n", err)
			} else {
				fmt.Printf("  Backup: %s\n", filepath.Base(backupPath))
			}
			fmt.Printf("  ✓ Created %s\n", instructionsPath)
		}
	} else {
		// Create new file
		fmt.Printf("  ✓ Created %s\n", instructionsPath)
	}

	// Create directory if needed
	dir := filepath.Dir(instructionsPath)
	if dir != "." {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed to create directory: %w", err)
		}
	}

	// Write file
	if err := os.WriteFile(instructionsPath, []byte(content), 0644); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	return nil
}

// getClaudeCodeInstructions returns instructions for Claude Code (claude.md)
func getClaudeCodeInstructions() string {
	return `# Symphony Code Conventions

**This project uses Symphony MCP for automated code convention management.**

## Critical Requirements

### 1. Before Writing Code

**Check MCP Status**: Verify Symphony MCP server is active. If unavailable, warn the user and do not proceed.

**Query Conventions**: Use ` + "`symphony/query_conventions`" + ` to retrieve relevant rules.
- Select appropriate category: security, style, documentation, error_handling, architecture, performance, testing
- Filter by languages as needed

### 2. After Writing Code

**Validate Changes**: Always run ` + "`symphony/validate_code`" + ` to check all changes against project conventions.

**Fix Violations**: Address any issues found before committing.

## Workflow

1. Verify Symphony MCP is active
2. Query conventions for your task
3. Write code
4. Validate with Symphony
5. Fix violations
6. Commit

---

*Auto-generated by Symphony*
`
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
2. **Query conventions** - Use ` + "`symphony/query_conventions`" + ` with appropriate category and language

### After Code Generation
1. **Validate all changes** - Use ` + "`symphony/validate_code`" + `
2. **Fix violations** - Address issues before committing

## Convention Categories
- security
- style
- documentation
- error_handling
- architecture
- performance
- testing

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
2. Query relevant conventions using symphony/query_conventions tool.
   - Categories: security, style, documentation, error_handling, architecture, performance, testing
   - Filter by programming language

### After Writing Code
1. Always validate changes using symphony/validate_code tool (validates all git changes)
2. Fix any violations found
3. Only commit after validation passes

## Workflow
Check MCP → Query Conventions → Write Code → Validate → Fix → Commit

---
Auto-generated by Symphony
`
}
