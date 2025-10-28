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
	Short: "Symphony - LLM-friendly convention linter",
	Long: `Symphony는 자연어로 정의된 컨벤션을 검증하는 LLM 친화적 linter입니다.
코드 스타일, 아키텍처 규칙, RBAC 정책 등을 자연어로 정의하고 자동 검증할 수 있습니다.`,
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
