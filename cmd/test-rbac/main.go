package main

import (
	"fmt"
	"os"

	"github.com/DevSymphony/sym-cli/internal/roles"
)

func main() {
	// Change to test directory
	if err := os.Chdir("/tmp/rbac-test"); err != nil {
		fmt.Printf("❌ Failed to change directory: %v\n", err)
		return
	}

	fmt.Println("🧪 RBAC 검증 테스트 시작\n")
	fmt.Println("================================================================")

	// Test scenarios
	testCases := []struct {
		name     string
		username string
		files    []string
	}{
		{
			name:     "Frontend Dev - 허용된 파일만",
			username: "alice",
			files: []string{
				"src/components/Button.js",
				"src/components/ui/Modal.js",
				"src/hooks/useAuth.js",
			},
		},
		{
			name:     "Frontend Dev - 거부된 파일 포함",
			username: "alice",
			files: []string{
				"src/components/Button.js",
				"src/core/engine.js",
				"src/api/client.js",
			},
		},
		{
			name:     "Senior Dev - 모든 파일",
			username: "charlie",
			files: []string{
				"src/components/Button.js",
				"src/core/engine.js",
				"src/api/client.js",
				"src/utils/helper.js",
			},
		},
		{
			name:     "Viewer - 읽기 전용",
			username: "david",
			files: []string{
				"src/components/Button.js",
			},
		},
		{
			name:     "Frontend Dev - 혼합 케이스",
			username: "bob",
			files: []string{
				"src/hooks/useData.js",
				"src/core/config.js",
				"src/utils/format.js",
				"src/components/Header.js",
			},
		},
	}

	for i, tc := range testCases {
		fmt.Printf("\n📋 테스트 %d: %s\n", i+1, tc.name)
		fmt.Printf("   사용자: %s\n", tc.username)
		fmt.Printf("   파일 수: %d개\n", len(tc.files))

		result, err := roles.ValidateFilePermissions(tc.username, tc.files)
		if err != nil {
			fmt.Printf("   ❌ 오류: %v\n", err)
			continue
		}

		if result.Allowed {
			fmt.Printf("   ✅ 결과: 모든 파일 수정 가능\n")
		} else {
			fmt.Printf("   ❌ 결과: %d개 파일 수정 불가\n", len(result.DeniedFiles))
			fmt.Printf("   거부된 파일:\n")
			for _, file := range result.DeniedFiles {
				fmt.Printf("      - %s\n", file)
			}
		}
	}

	fmt.Println("\n================================================================")
	fmt.Println("✅ 테스트 완료!")
}
