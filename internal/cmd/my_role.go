package cmd

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/AlecAivazis/survey/v2"
	"github.com/DevSymphony/sym-cli/internal/roles"

	"github.com/spf13/cobra"
)

var myRoleCmd = &cobra.Command{
	Use:   "my-role",
	Short: "Check or change your currently selected role",
	Long: `Display your currently selected role or change it.

Output can be formatted as JSON using --json flag for scripting purposes.
Use --select flag to interactively select a new role.`,
	Run: runMyRole,
}

var (
	myRoleJSON   bool
	myRoleSelect bool
)

func init() {
	myRoleCmd.Flags().BoolVar(&myRoleJSON, "json", false, "Output in JSON format")
	myRoleCmd.Flags().BoolVar(&myRoleSelect, "select", false, "Interactively select a new role")
}

func runMyRole(cmd *cobra.Command, args []string) {
	// Check if roles.json exists
	exists, err := roles.RolesExists()
	if err != nil || !exists {
		if myRoleJSON {
			output := map[string]string{"error": "roles.json not found"}
			_ = json.NewEncoder(os.Stdout).Encode(output)
		} else {
			printError("roles.json not found")
			fmt.Println(indent("Run 'sym init' first"))
		}
		os.Exit(1)
	}

	// If --select flag is provided, prompt for role selection
	if myRoleSelect {
		selectNewRole()
		return
	}

	// Get current role
	role, err := roles.GetCurrentRole()
	if err != nil {
		if myRoleJSON {
			_ = json.NewEncoder(os.Stdout).Encode(map[string]string{"error": fmt.Sprintf("Failed to get current role: %v", err)})
		} else {
			printError(fmt.Sprintf("Failed to get current role: %v", err))
		}
		os.Exit(1)
	}

	if myRoleJSON {
		output := map[string]string{
			"role": role,
		}
		_ = json.NewEncoder(os.Stdout).Encode(output)
	} else {
		if role == "" {
			printWarn("No role selected")
			fmt.Println(indent("Run 'sym my-role --select' to select a role"))
		} else {
			fmt.Printf("Current role: %s\n", role)
			fmt.Println(indent("Run 'sym my-role --select' to change"))
		}
	}
}

func selectNewRole() {
	// Use custom template to hide "type to filter" and typed characters
	restore := useSelectTemplateNoFilter()
	defer restore()

	availableRoles, err := roles.GetAvailableRoles()
	if err != nil {
		printError(fmt.Sprintf("Failed to get available roles: %v", err))
		os.Exit(1)
	}

	if len(availableRoles) == 0 {
		printError("No roles defined in roles.json")
		os.Exit(1)
	}

	currentRole, _ := roles.GetCurrentRole()

	fmt.Println()
	printTitle("Role", "Select your role")
	fmt.Println()

	// Use survey.Select for consistent UI
	var selectedRole string
	prompt := &survey.Select{
		Message: "Select role:",
		Options: availableRoles,
		Default: currentRole,
	}

	if err := survey.AskOne(prompt, &selectedRole); err != nil {
		printWarn("No selection made")
		return
	}

	if err := roles.SetCurrentRole(selectedRole); err != nil {
		printError(fmt.Sprintf("Failed to save role: %v", err))
		os.Exit(1)
	}

	printOK(fmt.Sprintf("Your role has been changed to: %s", selectedRole))
}
