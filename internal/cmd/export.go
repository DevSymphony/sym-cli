package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var exportCmd = &cobra.Command{
	Use:   "export [path]",
	Short: "현재 작업에 필요한 컨벤션을 추출하여 반환합니다",
	Long: `현재 작업 컨텍스트에 맞는 관련 컨벤션만 추출하여 반환합니다.
LLM이 작업 시 컨텍스트에 포함할 수 있도록 최적화된 형태로 제공됩니다.`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		path := "."
		if len(args) > 0 {
			path = args[0]
		}

		context, _ := cmd.Flags().GetString("context")
		format, _ := cmd.Flags().GetString("format")

		fmt.Printf("Exporting conventions for: %s\n", path)
		fmt.Printf("Context: %s\n", context)
		fmt.Printf("Format: %s\n", format)

		// TODO: 실제 내보내기 로직 구현
		return nil
	},
}

func init() {
	rootCmd.AddCommand(exportCmd)

	exportCmd.Flags().StringP("context", "c", "", "work context description")
	exportCmd.Flags().StringP("format", "f", "text", "output format (text|json|markdown)")
	exportCmd.Flags().StringSlice("files", []string{}, "files being modified")
	exportCmd.Flags().StringSlice("languages", []string{}, "programming languages involved")
}
