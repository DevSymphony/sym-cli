package auth

import (
	"context"
	_ "embed"
	"fmt"
	"net/http"
	"github.com/DevSymphony/sym-cli/internal/config"
	"github.com/DevSymphony/sym-cli/internal/github"
	"time"

	"github.com/pkg/browser"
)

//go:embed static/login-success.html
var loginSuccessHTML string

const (
	// symphonyclient integration: default port 3000 → 8787
	callbackPort = 8787
	redirectURI  = "http://localhost:8787/oauth/callback"
)

// StartOAuthFlow initiates the OAuth flow and waits for the callback
func StartOAuthFlow() error {
	// Load config
	cfg, err := config.LoadConfig()
	if err != nil {
		return err
	}

	// Create a channel to receive the authorization code
	codeChan := make(chan string, 1)
	errChan := make(chan error, 1)

	// Create HTTP server for callback
	mux := http.NewServeMux()
	mux.HandleFunc("/oauth/callback", func(w http.ResponseWriter, r *http.Request) {
		code := r.URL.Query().Get("code")
		if code == "" {
			errChan <- fmt.Errorf("no authorization code received")
			http.Error(w, "Authorization failed: no code received", http.StatusBadRequest)
			return
		}

		codeChan <- code

		// Send success message to browser using embedded HTML
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Write([]byte(loginSuccessHTML))
	})

	// Serve the Tailwind CSS file
	mux.HandleFunc("/styles/output.css", func(w http.ResponseWriter, r *http.Request) {
		// Serve the built CSS file from the server's static directory
		http.ServeFile(w, r, "internal/server/static/styles/output.css")
	})

	server := &http.Server{
		Addr:    fmt.Sprintf(":%d", callbackPort),
		Handler: mux,
	}

	// Start server in goroutine
	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errChan <- err
		}
	}()

	// Give server time to start
	time.Sleep(100 * time.Millisecond)

	// Open browser
	authURL := github.GetAuthURL(cfg.GitHubHost, cfg.ClientID, redirectURI)
	fmt.Printf("Opening browser for authentication...\n")
	fmt.Printf("If browser doesn't open, visit: %s\n\n", authURL)

	if err := browser.OpenURL(authURL); err != nil {
		fmt.Printf("Could not open browser automatically: %v\n", err)
		fmt.Printf("Please manually open: %s\n", authURL)
	}

	// Wait for callback or error
	var code string
	select {
	case code = <-codeChan:
		// Success
	case err := <-errChan:
		server.Shutdown(context.Background())
		return err
	case <-time.After(5 * time.Minute):
		server.Shutdown(context.Background())
		return fmt.Errorf("authentication timeout")
	}

	// Shutdown server
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	server.Shutdown(ctx)

	// Exchange code for token
	fmt.Println("Exchanging code for access token...")
	accessToken, err := github.ExchangeCodeForToken(cfg.GitHubHost, cfg.ClientID, cfg.ClientSecret, code)
	if err != nil {
		return fmt.Errorf("failed to exchange code for token: %w", err)
	}

	// Save token
	token := &config.Token{
		AccessToken: accessToken,
	}

	if err := config.SaveToken(token); err != nil {
		return fmt.Errorf("failed to save token: %w", err)
	}

	// Verify token by getting user info
	client := github.NewClient(cfg.GitHubHost, accessToken)
	user, err := client.GetCurrentUser()
	if err != nil {
		return fmt.Errorf("failed to verify token: %w", err)
	}

	fmt.Printf("\n✓ Successfully authenticated as %s\n", user.Login)

	return nil
}
