package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var validateCmd = &cobra.Command{
	Use:   "validate [path]",
	Short: "코드가 정의된 컨벤션을 준수하는지 검증합니다",
	Long: `지정된 경로의 코드가 .sym/policy.json에 정의된 컨벤션을 준수하는지 검증합니다.
검증 결과는 표준 출력으로 반환되며, 위반 사항이 있으면 non-zero exit code를 반환합니다.`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		path := "."
		if len(args) > 0 {
			path = args[0]
		}

		policyPath, _ := cmd.Flags().GetString("policy")
		format, _ := cmd.Flags().GetString("format")

		fmt.Printf("Validating path: %s\n", path)
		fmt.Printf("Policy: %s\n", policyPath)
		fmt.Printf("Format: %s\n", format)

		// TODO: 실제 검증 로직 구현
		return nil
	},
}

func init() {
	rootCmd.AddCommand(validateCmd)

	validateCmd.Flags().StringP("policy", "p", ".sym/policy.json", "policy file path")
	validateCmd.Flags().StringP("format", "f", "text", "output format (text|json|junit)")
	validateCmd.Flags().String("severity", "error", "minimum severity level to report (error|warning|info)")
	validateCmd.Flags().Bool("fix", false, "automatically fix violations when possible")
}
