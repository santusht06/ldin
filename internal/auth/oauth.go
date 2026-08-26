// Copyright (c) 2026 Santusht Kotai
// SPDX-License-Identifier: MIT

package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"github.com/santusht/ldin/internal/config"
)

const (
	LinkedInAuthURL  = "https://www.linkedin.com/oauth/v2/authorization"
	LinkedInTokenURL = "https://www.linkedin.com/oauth/v2/accessToken"
	LinkedInUserInfo = "https://api.linkedin.com/v2/userinfo"
	DefaultPort      = 8085
)

// DefaultScopes requested during interactive OAuth login
var DefaultScopes = []string{
	"openid",
	"profile",
	"email",
	"w_member_social",
}

// ExtendedScopes for apps with full Community Management approval
var ExtendedScopes = []string{
	"openid",
	"profile",
	"email",
	"w_member_social",
	"r_member_social",
	"r_member_postAnalytics",
	"r_member_profileAnalytics",
	"w_organization_social",
	"r_organization_social",
}

// UserInfoResponse represents the OpenID Connect /v2/userinfo response
type UserInfoResponse struct {
	Sub           string `json:"sub"` // urn:li:person:xxxx or member identifier
	Name          string `json:"name"`
	GivenName     string `json:"given_name"`
	FamilyName    string `json:"family_name"`
	Picture       string `json:"picture"`
	Email         string `json:"email"`
	EmailVerified bool   `json:"email_verified"`
	Locale        struct {
		Country  string `json:"country"`
		Language string `json:"language"`
	} `json:"locale"`
}

// TokenResponse represents OAuth 2.0 token response
type TokenResponse struct {
	AccessToken           string `json:"access_token"`
	ExpiresIn             int64  `json:"expires_in"`
	RefreshToken          string `json:"refresh_token,omitempty"`
	RefreshTokenExpiresIn int64  `json:"refresh_token_expires_in,omitempty"`
	Scope                 string `json:"scope"`
}

// PKCEParams holds code verifier and challenge for OAuth 2.0 PKCE
type PKCEParams struct {
	Verifier  string
	Challenge string
	State     string
}

// GeneratePKCE creates cryptographically secure PKCE code verifier & challenge
func GeneratePKCE() (*PKCEParams, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return nil, err
	}
	verifier := base64.RawURLEncoding.EncodeToString(b)

	h := sha256.Sum256([]byte(verifier))
	challenge := base64.RawURLEncoding.EncodeToString(h[:])

	sb := make([]byte, 16)
	if _, err := rand.Read(sb); err != nil {
		return nil, err
	}
	state := base64.RawURLEncoding.EncodeToString(sb)

	return &PKCEParams{
		Verifier:  verifier,
		Challenge: challenge,
		State:     state,
	}, nil
}

// StartOAuthFlow launches browser authorization and listens on local callback
func StartOAuthFlow(clientID, clientSecret string, scopes []string, port int) (*TokenResponse, *UserInfoResponse, error) {
	if port <= 0 {
		port = DefaultPort
	}
	if len(scopes) == 0 {
		scopes = DefaultScopes
	}

	redirectURI := fmt.Sprintf("http://localhost:%d/callback", port)
	pkce, err := GeneratePKCE()
	if err != nil {
		return nil, nil, fmt.Errorf("failed generating PKCE: %w", err)
	}

	authURLValues := url.Values{}
	authURLValues.Set("response_type", "code")
	authURLValues.Set("client_id", clientID)
	authURLValues.Set("redirect_uri", redirectURI)
	authURLValues.Set("state", pkce.State)
	authURLValues.Set("scope", strings.Join(scopes, " "))
	authURLValues.Set("code_challenge", pkce.Challenge)
	authURLValues.Set("code_challenge_method", "S256")

	authURL := fmt.Sprintf("%s?%s", LinkedInAuthURL, authURLValues.Encode())

	codeChan := make(chan string, 1)
	errChan := make(chan error, 1)

	// Start local listener
	listener, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", port))
	if err != nil {
		return nil, nil, fmt.Errorf("could not start local server on port %d: %w", port, err)
	}

	server := &http.Server{
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/callback" {
				http.NotFound(w, r)
				return
			}

			q := r.URL.Query()
			state := q.Get("state")
			if state != pkce.State {
				http.Error(w, "Invalid state parameter", http.StatusBadRequest)
				errChan <- fmt.Errorf("state mismatch in OAuth callback")
				return
			}

			if errParam := q.Get("error"); errParam != "" {
				errDesc := q.Get("error_description")
				http.Error(w, "Authentication failed: "+errDesc, http.StatusBadRequest)
				errChan <- fmt.Errorf("oauth error: %s (%s)", errParam, errDesc)
				return
			}

			code := q.Get("code")
			if code == "" {
				http.Error(w, "Missing authorization code", http.StatusBadRequest)
				errChan <- fmt.Errorf("no authorization code returned")
				return
			}

			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			_, _ = w.Write([]byte(`<!DOCTYPE html>
<html>
<head><title>ldin Authentication Successful</title>
<style>
body { font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif; background: #0e1117; color: #fff; display: flex; align-items: center; justify-content: center; height: 100vh; margin: 0; }
.card { background: #161b22; border: 1px solid #30363d; border-radius: 12px; padding: 32px 48px; text-align: center; box-shadow: 0 8px 24px rgba(0,0,0,0.4); }
h1 { color: #00D2FF; margin-bottom: 8px; font-size: 24px; }
p { color: #8b949e; font-size: 15px; margin: 0; }
.badge { display: inline-block; background: #0A66C2; color: #fff; padding: 4px 12px; border-radius: 20px; font-size: 12px; margin-top: 16px; font-weight: 600; }
</style>
</head>
<body>
<div class="card">
  <h1>Authentication Successful</h1>
  <p>You can close this window and return to your terminal.</p>
  <div class="badge">ldin CLI Connected</div>
</div>
</body>
</html>`))
			codeChan <- code
		}),
	}

	go func() {
		_ = server.Serve(listener)
	}()

	// Open browser
	openBrowser(authURL)

	// Wait for callback or timeout
	select {
	case code := <-codeChan:
		_ = server.Shutdown(context.Background())
		// Exchange code for token
		return ExchangeCode(clientID, clientSecret, code, redirectURI, pkce.Verifier)
	case err := <-errChan:
		_ = server.Shutdown(context.Background())
		return nil, nil, err
	case <-time.After(180 * time.Second):
		_ = server.Shutdown(context.Background())
		return nil, nil, fmt.Errorf("authentication timed out after 3 minutes")
	}
}

// ExchangeCode trades authorization code for access token
func ExchangeCode(clientID, clientSecret, code, redirectURI, codeVerifier string) (*TokenResponse, *UserInfoResponse, error) {
	data := url.Values{}
	data.Set("grant_type", "authorization_code")
	data.Set("code", code)
	data.Set("redirect_uri", redirectURI)
	data.Set("client_id", clientID)
	data.Set("client_secret", clientSecret)
	if codeVerifier != "" {
		data.Set("code_verifier", codeVerifier)
	}

	req, err := http.NewRequest("POST", LinkedInTokenURL, strings.NewReader(data.Encode()))
	if err != nil {
		return nil, nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, nil, fmt.Errorf("token request failed: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return nil, nil, fmt.Errorf("token error (HTTP %d): %s", resp.StatusCode, string(body))
	}

	var tok TokenResponse
	if err := json.Unmarshal(body, &tok); err != nil {
		return nil, nil, fmt.Errorf("invalid token JSON: %w", err)
	}

	// Fetch userinfo
	userInfo, err := FetchUserInfo(tok.AccessToken)
	if err != nil {
		// Non-fatal if userinfo fails, construct fallback
		userInfo = &UserInfoResponse{
			Sub:  "urn:li:person:authenticated",
			Name: "LinkedIn Member",
		}
	}

	return &tok, userInfo, nil
}

// FetchUserInfo calls the OpenID /v2/userinfo endpoint
func FetchUserInfo(accessToken string) (*UserInfoResponse, error) {
	req, err := http.NewRequest("GET", LinkedInUserInfo, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("userinfo request failed (HTTP %d): %s", resp.StatusCode, string(body))
	}

	var u UserInfoResponse
	if err := json.NewDecoder(resp.Body).Decode(&u); err != nil {
		return nil, err
	}

	if !strings.HasPrefix(u.Sub, "urn:li:person:") && u.Sub != "" {
		u.Sub = "urn:li:person:" + u.Sub
	}

	return &u, nil
}

// RefreshToken requests a new access token using a refresh token
func RefreshToken(clientID, clientSecret, refreshToken string) (*TokenResponse, error) {
	data := url.Values{}
	data.Set("grant_type", "refresh_token")
	data.Set("refresh_token", refreshToken)
	data.Set("client_id", clientID)
	data.Set("client_secret", clientSecret)

	req, err := http.NewRequest("POST", LinkedInTokenURL, strings.NewReader(data.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("refresh token failed (HTTP %d): %s", resp.StatusCode, string(body))
	}

	var tok TokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&tok); err != nil {
		return nil, err
	}
	return &tok, nil
}

// SaveSession writes an authenticated session to the profile credentials store
func SaveSession(cm *config.ConfigManager, profileName string, tok *TokenResponse, u *UserInfoResponse, clientID, clientSecret string) error {
	var scopes []string
	if tok.Scope != "" {
		scopes = strings.Split(tok.Scope, " ")
	} else {
		scopes = DefaultScopes
	}

	expiresAt := time.Now().Unix() + tok.ExpiresIn
	if tok.ExpiresIn == 0 {
		expiresAt = time.Now().Add(60 * 24 * time.Hour).Unix() // default ~60 days
	}

	creds := &config.ProfileCredentials{
		Name:         profileName,
		MemberID:     u.Sub,
		MemberURN:    u.Sub,
		DisplayName:  u.Name,
		Email:        u.Email,
		AccessToken:  tok.AccessToken,
		RefreshToken: tok.RefreshToken,
		ExpiresAt:    expiresAt,
		Scopes:       scopes,
		ClientID:     clientID,
		ClientSecret: clientSecret,
	}

	if err := cm.SaveProfile(creds); err != nil {
		return err
	}

	// Update active profile in config.yaml
	cfg, err := cm.LoadConfig()
	if err == nil {
		cfg.ActiveProfile = profileName
		_ = cm.SaveConfig(cfg)
	}

	return nil
}

func openBrowser(urlStr string) {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", urlStr)
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", urlStr)
	default:
		cmd = exec.Command("xdg-open", urlStr)
	}
	_ = cmd.Start()
}
