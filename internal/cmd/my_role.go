package cmd

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"

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
			fmt.Println("❌ roles.json not found")
			fmt.Println("Run 'sym init' first")
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
		handleError("Failed to get current role", err, myRoleJSON)
		os.Exit(1)
	}

	if myRoleJSON {
		output := map[string]string{
			"role": role,
		}
		_ = json.NewEncoder(os.Stdout).Encode(output)
	} else {
		if role == "" {
			fmt.Println("⚠ No role selected")
			fmt.Println("Run 'sym my-role --select' to select a role")
			fmt.Println("Or use the dashboard: 'sym dashboard'")
		} else {
			fmt.Printf("Current role: %s\n", role)
			fmt.Println("\nTo change your role:")
			fmt.Println("  sym my-role --select")
		}
	}
}

func selectNewRole() {
	availableRoles, err := roles.GetAvailableRoles()
	if err != nil {
		fmt.Printf("❌ Failed to get available roles: %v\n", err)
		os.Exit(1)
	}

	if len(availableRoles) == 0 {
		fmt.Println("❌ No roles defined in roles.json")
		os.Exit(1)
	}

	currentRole, _ := roles.GetCurrentRole()

	fmt.Println("🎭 Select your role:")
	fmt.Println()
	for i, role := range availableRoles {
		marker := "  "
		if role == currentRole {
			marker = "→ "
		}
		fmt.Printf("%s%d. %s\n", marker, i+1, role)
	}
	fmt.Println()

	reader := bufio.NewReader(os.Stdin)
	fmt.Print("Enter number (1-" + strconv.Itoa(len(availableRoles)) + "): ")
	input, _ := reader.ReadString('\n')
	input = strings.TrimSpace(input)

	if input == "" {
		fmt.Println("⚠ No selection made")
		return
	}

	num, err := strconv.Atoi(input)
	if err != nil || num < 1 || num > len(availableRoles) {
		fmt.Println("❌ Invalid selection")
		os.Exit(1)
	}

	selectedRole := availableRoles[num-1]
	if err := roles.SetCurrentRole(selectedRole); err != nil {
		fmt.Printf("❌ Failed to save role: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("✓ Your role has been changed to: %s\n", selectedRole)
}

func handleError(msg string, err error, jsonMode bool) {
	if jsonMode {
		output := map[string]string{"error": fmt.Sprintf("%s: %v", msg, err)}
		_ = json.NewEncoder(os.Stdout).Encode(output)
	} else {
		fmt.Printf("❌ %s: %v\n", msg, err)
	}
}
