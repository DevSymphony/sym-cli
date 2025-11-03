package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"github.com/DevSymphony/sym-cli/internal/config"

	"github.com/spf13/cobra"
)

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Configure Symphony settings",
	Long: `Configure Symphony with your GitHub host and OAuth application credentials.

Examples:
  sym config                          # Interactive configuration
  sym config --show                   # Show current configuration
  sym config --reset                  # Reset to default (server mode)
  sym config --use-custom-oauth       # Switch to custom OAuth mode`,
	Run: runConfig,
}

var (
	configHost         string
	configShow         bool
	configReset        bool
	configID           string
	configSecret       string
	useCustomOAuth     bool
	configServerURL    string
)

func init() {
	configCmd.Flags().StringVar(&configHost, "host", "", "GitHub host (e.g., github.com or ghes.company.com)")
	configCmd.Flags().StringVar(&configID, "client-id", "", "OAuth App Client ID")
	configCmd.Flags().StringVar(&configSecret, "client-secret", "", "OAuth App Client Secret")
	configCmd.Flags().BoolVar(&useCustomOAuth, "use-custom-oauth", false, "Use custom OAuth App (for GitHub Enterprise)")
	configCmd.Flags().StringVar(&configServerURL, "server-url", "", "Symphony auth server URL")
	configCmd.Flags().BoolVar(&configShow, "show", false, "Show current configuration")
	configCmd.Flags().BoolVar(&configReset, "reset", false, "Reset configuration to default (server mode)")
}

func runConfig(cmd *cobra.Command, args []string) {
	if configShow {
		showConfig()
		return
	}

	if configReset {
		resetConfig()
		return
	}

	// Load existing config or create new one
	cfg, err := config.LoadConfig()
	if err != nil {
		cfg = &config.Config{
			AuthMode: "server", // default
		}
	}

	// Handle server URL configuration
	if configServerURL != "" {
		cfg.ServerURL = configServerURL
		if err := config.SaveConfig(cfg); err != nil {
			fmt.Printf("❌ Failed to save configuration: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("✓ Server URL updated successfully!")
		fmt.Printf("  Server URL: %s\n", cfg.ServerURL)
		return
	}

	// Handle custom OAuth configuration
	if useCustomOAuth {
		configureCustomOAuth(cfg)
		return
	}

	// No flags provided - show interactive menu
	showConfigMenu(cfg)
}

func showConfigMenu(cfg *config.Config) {
	fmt.Println("\n🎵 Symphony 설정")
	fmt.Println()
	fmt.Println("인증 모드를 선택하세요:")
	fmt.Println("  1. 서버 인증 (기본, 권장)")
	fmt.Println("     - OAuth App 설정 불필요")
	fmt.Println("     - 브라우저에서 GitHub 로그인만 하면 됨")
	fmt.Println()
	fmt.Println("  2. Custom OAuth (GitHub Enterprise 사용자)")
	fmt.Println("     - 자체 OAuth App 필요")
	fmt.Println("     - 기업용 GitHub 서버 지원")
	fmt.Println()

	reader := bufio.NewReader(os.Stdin)
	fmt.Print("선택 [1]: ")
	input, _ := reader.ReadString('\n')
	choice := strings.TrimSpace(input)

	if choice == "" {
		choice = "1"
	}

	switch choice {
	case "1":
		// Server mode
		cfg.AuthMode = "server"
		fmt.Println("\n✓ 서버 인증 모드로 설정되었습니다")
		fmt.Println()
		fmt.Println("다음 단계:")
		fmt.Println("  sym login   # GitHub 로그인")

	case "2":
		// Custom OAuth mode
		configureCustomOAuth(cfg)
		return

	default:
		fmt.Println("잘못된 선택입니다")
		os.Exit(1)
	}

	// Save config
	if err := config.SaveConfig(cfg); err != nil {
		fmt.Printf("❌ Failed to save configuration: %v\n", err)
		os.Exit(1)
	}
}

func configureCustomOAuth(cfg *config.Config) {
	fmt.Println("\n🔧 Custom OAuth 설정")
	fmt.Println()

	reader := bufio.NewReader(os.Stdin)

	// GitHub Host
	if configHost != "" {
		cfg.GitHubHost = configHost
	} else {
		defaultHost := cfg.GitHubHost
		if defaultHost == "" {
			defaultHost = "github.com"
		}
		fmt.Printf("GitHub Host [%s]: ", defaultHost)
		input, _ := reader.ReadString('\n')
		input = strings.TrimSpace(input)
		if input != "" {
			cfg.GitHubHost = input
		} else {
			cfg.GitHubHost = defaultHost
		}
	}

	// Client ID
	if configID != "" {
		cfg.ClientID = configID
	} else {
		fmt.Printf("OAuth Client ID [%s]: ", maskString(cfg.ClientID))
		input, _ := reader.ReadString('\n')
		input = strings.TrimSpace(input)
		if input != "" {
			cfg.ClientID = input
		}
	}

	// Client Secret
	if configSecret != "" {
		cfg.ClientSecret = configSecret
	} else {
		fmt.Printf("OAuth Client Secret [%s]: ", maskString(cfg.ClientSecret))
		input, _ := reader.ReadString('\n')
		input = strings.TrimSpace(input)
		if input != "" {
			cfg.ClientSecret = input
		}
	}

	// Validate
	if cfg.GitHubHost == "" || cfg.ClientID == "" || cfg.ClientSecret == "" {
		fmt.Println("\n❌ Error: All fields are required")
		os.Exit(1)
	}

	// Set mode to custom
	cfg.AuthMode = "custom"

	// Save config
	if err := config.SaveConfig(cfg); err != nil {
		fmt.Printf("❌ Failed to save configuration: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("\n✓ Configuration saved successfully!")
	fmt.Printf("  Mode: Custom OAuth\n")
	fmt.Printf("  GitHub Host: %s\n", cfg.GitHubHost)
	fmt.Printf("  Client ID: %s\n", maskString(cfg.ClientID))
	fmt.Println("\nNext step: Run 'sym login' to authenticate")
}

func showConfig() {
	cfg, err := config.LoadConfig()
	if err != nil {
		fmt.Printf("❌ No configuration found: %v\n", err)
		fmt.Println("Run 'sym config' to set up")
		os.Exit(1)
	}

	fmt.Println("Current Configuration:")
	fmt.Printf("  Authentication Mode: %s\n", cfg.GetAuthMode())

	if cfg.IsCustomOAuth() {
		// Custom OAuth mode
		fmt.Printf("  GitHub Host: %s\n", cfg.GitHubHost)
		fmt.Printf("  Client ID: %s\n", maskString(cfg.ClientID))
		fmt.Printf("  Client Secret: %s\n", maskString(cfg.ClientSecret))
	} else {
		// Server mode
		fmt.Printf("  Server URL: %s\n", cfg.GetServerURL())
	}

	fmt.Printf("\nConfig file: %s\n", config.GetConfigPath())

	if config.IsLoggedIn() {
		fmt.Println("\n✓ Logged in")
		fmt.Printf("Token file: %s\n", config.GetTokenPath())
	} else {
		fmt.Println("\n⚠ Not logged in")
		fmt.Println("Run 'sym login' to authenticate")
	}
}

func resetConfig() {
	fmt.Println("🔄 Resetting configuration to default...")
	fmt.Println()

	// Check if already logged in
	if config.IsLoggedIn() {
		fmt.Println("⚠️  Warning: You are currently logged in.")
		fmt.Print("   Resetting config will keep your token, but you may need to login again.\n")
		fmt.Print("   Continue? (y/N): ")

		reader := bufio.NewReader(os.Stdin)
		response, _ := reader.ReadString('\n')
		response = strings.TrimSpace(strings.ToLower(response))

		if response != "y" && response != "yes" {
			fmt.Println("\n❌ Reset cancelled")
			os.Exit(0)
		}
		fmt.Println()
	}

	// Create default config (server mode)
	defaultCfg := &config.Config{
		AuthMode: "server",
	}

	// Save config
	if err := config.SaveConfig(defaultCfg); err != nil {
		fmt.Printf("❌ Failed to reset configuration: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("✓ Configuration reset to default!")
	fmt.Println()
	fmt.Println("  Authentication Mode: server (default)")
	fmt.Printf("  Server URL: %s\n", defaultCfg.GetServerURL())
	fmt.Println()
	fmt.Println("Next step: Run 'sym login' to authenticate")
}

func maskString(s string) string {
	if s == "" {
		return ""
	}
	if len(s) <= 8 {
		return "****"
	}
	return s[:4] + "****" + s[len(s)-4:]
}
