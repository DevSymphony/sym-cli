package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var convertCmd = &cobra.Command{
	Use:   "convert [input]",
	Short: "사용자 정책을 검증 가능한 형식으로 변환합니다",
	Long: `사용자가 작성한 자연어 정책(A 스키마)를 검증 엔진이 읽을 수 있는
정형 스키마(B 스키마)로 변환합니다.`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		input := ".sym/user-policy.json"
		if len(args) > 0 {
			input = args[0]
		}

		output, _ := cmd.Flags().GetString("output")

		fmt.Printf("Converting: %s -> %s\n", input, output)

		// TODO: 실제 변환 로직 구현
		return nil
	},
}

func init() {
	rootCmd.AddCommand(convertCmd)

	convertCmd.Flags().StringP("output", "o", ".sym/policy.json", "output file path")
	convertCmd.Flags().Bool("validate", true, "validate output against schema")
}
