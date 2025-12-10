package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

// version will be set by build flags from cmd/sym/main.go
var version = "dev"

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the version number",
	Long:  `Print the version number of sym CLI.`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("sym version %s\n", version)
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}

// SetVersion sets the version string (called from main.go)
func SetVersion(v string) {
	version = v
}

// GetVersion returns the current version string
func GetVersion() string {
	return version
}
