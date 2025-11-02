package cmd

import (
	"fmt"
	"os"
	"github.com/DevSymphony/sym-cli/internal/auth"
	"github.com/DevSymphony/sym-cli/internal/config"

	"github.com/spf13/cobra"
)

var loginCmd = &cobra.Command{
	Use:   "login",
	Short: "Authenticate with GitHub",
	Long: `Start the OAuth flow to authenticate with GitHub.

This will open your browser to complete the authentication process.`,
	Run: runLogin,
}

func runLogin(cmd *cobra.Command, args []string) {
	// Check if already logged in
	if config.IsLoggedIn() {
		fmt.Println("⚠️  이미 로그인되어 있습니다")
		fmt.Println("   다시 인증하려면 먼저 'sym logout'을 실행하세요")
		os.Exit(0)
	}

	// Load or create config
	cfg, err := config.LoadConfig()
	if err != nil {
		// Config doesn't exist - create default config with server mode
		cfg = &config.Config{
			AuthMode: "server",
		}
		if err := config.SaveConfig(cfg); err != nil {
			fmt.Printf("❌ Failed to create config: %v\n", err)
			os.Exit(1)
		}
	}

	// Choose authentication method based on mode
	if cfg.IsCustomOAuth() {
		// Custom OAuth mode (Enterprise)
		loginWithCustomOAuth(cfg)
	} else {
		// Server mode (default)
		loginWithServer(cfg)
	}
}

// loginWithServer authenticates using Symphony auth server
func loginWithServer(cfg *config.Config) {
	serverURL := cfg.GetServerURL()

	fmt.Println("🎵 Symphony CLI 인증")
	fmt.Printf("   서버: %s\n", serverURL)
	fmt.Println()

	// Authenticate with server
	token, username, err := auth.AuthenticateWithServer(serverURL)
	if err != nil {
		fmt.Printf("\n❌ 인증 실패: %v\n", err)
		fmt.Println()
		fmt.Println("💡 문제가 계속되면 다음을 시도해보세요:")
		fmt.Println("   1. 네트워크 연결 확인")
		fmt.Println("   2. 서버 상태 확인: " + serverURL)
		fmt.Println("   3. Enterprise 사용자는 --use-custom-oauth 옵션 사용")
		os.Exit(1)
	}

	// Save token
	if err := config.SaveToken(&config.Token{AccessToken: token}); err != nil {
		fmt.Printf("❌ Failed to save token: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("\n환영합니다, %s!\n", username)
	fmt.Println("\n이제 Symphony 명령어를 사용할 수 있습니다:")
	fmt.Println("  sym whoami     - 현재 사용자 확인")
	fmt.Println("  sym init       - 저장소 초기화")
	fmt.Println("  sym dashboard  - 웹 대시보드 실행")
}

// loginWithCustomOAuth authenticates using custom OAuth app (Enterprise)
func loginWithCustomOAuth(cfg *config.Config) {
	// Validate custom OAuth config
	if cfg.GitHubHost == "" || cfg.ClientID == "" || cfg.ClientSecret == "" {
		fmt.Println("❌ Custom OAuth 설정이 완료되지 않았습니다")
		fmt.Println()
		fmt.Println("다음 명령어로 설정을 완료하세요:")
		fmt.Println("  sym config --use-custom-oauth")
		os.Exit(1)
	}

	fmt.Println("🔐 Custom OAuth 인증")
	fmt.Printf("   GitHub: %s\n", cfg.GitHubHost)
	fmt.Println()

	// Start OAuth flow
	if err := auth.StartOAuthFlow(); err != nil {
		fmt.Printf("❌ Authentication failed: %v\n", err)
		os.Exit(1)
	}
}
