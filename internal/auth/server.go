package auth

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/pkg/browser"
)

// SessionResponse is the response from /authStart
type SessionResponse struct {
	SessionCode string `json:"session_code"`
	AuthURL     string `json:"auth_url"`
	ExpiresIn   int    `json:"expires_in"`
}

// StatusResponse is the response from /authStatus
type StatusResponse struct {
	Status         string `json:"status"`
	Message        string `json:"message,omitempty"`
	Error          string `json:"error,omitempty"`
	GithubToken    string `json:"github_token,omitempty"`
	GithubUsername string `json:"github_username,omitempty"`
	GithubID       int64  `json:"github_id,omitempty"`
	GithubName     string `json:"github_name,omitempty"`
}

// AuthenticateWithServer performs authentication using the Sym auth server
func AuthenticateWithServer(serverURL string) (string, string, error) {
	// 1. Start authentication session
	session, err := startAuthSession(serverURL)
	if err != nil {
		return "", "", fmt.Errorf("failed to start auth session: %w", err)
	}

	fmt.Printf("\n🔐 Symphony CLI 인증\n")
	fmt.Printf("   세션 코드: %s\n", session.SessionCode)
	fmt.Printf("   만료 시간: %d초 후\n\n", session.ExpiresIn)

	// 2. Open browser
	fmt.Println("브라우저를 열어서 GitHub 로그인을 진행합니다...")
	fmt.Printf("URL: %s\n\n", session.AuthURL)

	if err := browser.OpenURL(session.AuthURL); err != nil {
		fmt.Printf("⚠️  브라우저를 자동으로 열 수 없습니다.\n")
		fmt.Printf("   수동으로 다음 URL을 열어주세요:\n")
		fmt.Printf("   %s\n\n", session.AuthURL)
	}

	// 3. Poll for status
	fmt.Print("승인 대기 중")
	token, username, err := pollForToken(serverURL, session.SessionCode, session.ExpiresIn)
	if err != nil {
		return "", "", err
	}

	fmt.Printf("\n\n✅ 인증 성공! (%s)\n", username)

	return token, username, nil
}

// startAuthSession starts a new authentication session
func startAuthSession(serverURL string) (*SessionResponse, error) {
	url := serverURL + "/authStart"

	requestBody := map[string]string{
		"device_name": "CLI",
	}

	jsonData, err := json.Marshal(requestBody)
	if err != nil {
		return nil, err
	}

	resp, err := http.Post(url, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to connect to auth server: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("server returned error: %s (status: %d)", string(body), resp.StatusCode)
	}

	var session SessionResponse
	if err := json.NewDecoder(resp.Body).Decode(&session); err != nil {
		return nil, fmt.Errorf("failed to parse server response: %w", err)
	}

	return &session, nil
}

// pollForToken polls the server for authentication status
func pollForToken(serverURL, sessionCode string, expiresIn int) (string, string, error) {
	url := fmt.Sprintf("%s/authStatus/%s", serverURL, sessionCode)

	// Calculate timeout
	timeout := time.Now().Add(time.Duration(expiresIn) * time.Second)

	// Poll every 3 seconds
	ticker := time.NewTicker(3 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			// Check if timeout
			if time.Now().After(timeout) {
				return "", "", fmt.Errorf("authentication timeout (%d초). 다시 시도해주세요", expiresIn)
			}

			// Check status
			status, err := checkAuthStatus(url)
			if err != nil {
				// Retry on error
				fmt.Print(".")
				continue
			}

			switch status.Status {
			case "pending":
				// Still waiting
				fmt.Print(".")
				continue

			case "approved":
				// Success!
				if status.GithubToken == "" {
					return "", "", fmt.Errorf("server did not return token")
				}
				return status.GithubToken, status.GithubUsername, nil

			case "denied":
				return "", "", fmt.Errorf("인증이 거부되었습니다")

			case "expired":
				return "", "", fmt.Errorf("세션이 만료되었습니다. 다시 시도해주세요")

			default:
				return "", "", fmt.Errorf("unknown status: %s", status.Status)
			}
		}
	}
}

// checkAuthStatus checks the authentication status
func checkAuthStatus(url string) (*StatusResponse, error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("invalid session code")
	}

	if resp.StatusCode == http.StatusGone {
		// Session expired
		var status StatusResponse
		json.NewDecoder(resp.Body).Decode(&status)
		return &status, nil
	}

	if resp.StatusCode == http.StatusForbidden {
		// Denied
		var status StatusResponse
		json.NewDecoder(resp.Body).Decode(&status)
		return &status, nil
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("server error: %s (status: %d)", string(body), resp.StatusCode)
	}

	var status StatusResponse
	if err := json.NewDecoder(resp.Body).Decode(&status); err != nil {
		return nil, err
	}

	return &status, nil
}
