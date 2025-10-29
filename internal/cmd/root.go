package cmd

import (
	"github.com/spf13/cobra"
)

var (
	cfgFile string
	verbose bool
)

var rootCmd = &cobra.Command{
	Use:   "sym",
	Short: "Symphony - LLM-friendly convention validation tool",
	Long: `Symphony is an LLM-friendly tool for validating conventions defined in natural language.

Key features:
- Convert natural language rules into structured policies
- Validate code compliance with defined conventions
- Integrate with LLM tools via MCP server

Designed to help LLMs easily understand and apply conventions when writing code.`,
}

func Execute() error {
	return rootCmd.Execute()
}

func init() {
	cobra.OnInitialize(initConfig)

	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is .sym/config.yaml)")
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "verbose output")
}

func initConfig() {
	if cfgFile != "" {
		// TODO: Use config file from the flag
		_ = cfgFile // Placeholder to avoid unused variable warning
	}
}
