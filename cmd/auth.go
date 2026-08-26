// Copyright (c) 2026 Santusht Kotai
// SPDX-License-Identifier: MIT

package cmd

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"
	"github.com/santusht/ldin/internal/auth"
	"github.com/santusht/ldin/internal/capabilities"
	"github.com/santusht/ldin/internal/output"
)

var (
	flagAuthToken        string
	flagAuthClientID     string
	flagAuthClientSecret string
	flagAuthPort         int
	flagAuthName         string
	flagAuthScopes       string
	flagAuthBrowser      bool
)

var authCmd = &cobra.Command{
	Use:   "auth",
	Short: "Manage LinkedIn authentication, tokens, and multi-identity sessions",
	Long: `Authenticate ldin with LinkedIn directly via access token or OAuth.
Zero browser dependencies required by default — simply pass your token or export LINKEDIN_TOKEN.`,
}

var authTokenCmd = &cobra.Command{
	Use:   "token [token-string]",
	Short: "Set active LinkedIn access token directly from terminal (No browser required)",
	Long: `Save a LinkedIn access token directly from your terminal.
Example:
  ldin auth token AQV...
  ldin auth token --name company AQV...
  export LINKEDIN_TOKEN=AQV...`,
	RunE: func(cmd *cobra.Command, args []string) error {
		profileName := flagAuthName
		if profileName == "" {
			profileName = "default"
		}

		tok := ""
		if len(args) > 0 {
			tok = strings.TrimSpace(args[0])
		} else if flagAuthToken != "" {
			tok = strings.TrimSpace(flagAuthToken)
		} else {
			fmt.Print("Enter or paste LinkedIn Access Token: ")
			fmt.Scanln(&tok)
			tok = strings.TrimSpace(tok)
		}

		if tok == "" {
			return fmt.Errorf("token cannot be empty")
		}

		Formatter.Info("Validating token against LinkedIn API (https://api.linkedin.com/v2/userinfo)...")
		userInfo, err := auth.FetchUserInfo(tok)
		if err != nil {
			Formatter.Warning("Could not fetch user profile details with token (non-fatal): %v", err)
			userInfo = &auth.UserInfoResponse{
				Sub:  "urn:li:person:authenticated",
				Name: "LinkedIn Member",
			}
		}

		tokResp := &auth.TokenResponse{
			AccessToken: tok,
			ExpiresIn:   60 * 24 * 3600,
			Scope:       strings.Join(auth.DefaultScopes, " "),
		}

		err = auth.SaveSession(ConfigMgr, profileName, tokResp, userInfo, flagAuthClientID, flagAuthClientSecret)
		if err != nil {
			return fmt.Errorf("failed saving session: %w", err)
		}

		Formatter.Success("Token successfully saved for profile '%s'!", profileName)
		Formatter.PrintKeyValue("Member", userInfo.Name)
		Formatter.PrintKeyValue("Member URN", userInfo.Sub)
		return nil
	},
}

var authLoginCmd = &cobra.Command{
	Use:   "login",
	Short: "Authenticate with LinkedIn (Direct token or optional browser OAuth)",
	Long: `Authenticate ldin with LinkedIn.
By default, prompts for token directly in terminal. Pass --browser for OAuth 2.0 PKCE browser flow.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if !flagAuthBrowser && flagAuthToken == "" && len(args) == 0 {
			return authTokenCmd.RunE(cmd, args)
		}
		if flagAuthToken != "" || len(args) > 0 {
			return authTokenCmd.RunE(cmd, args)
		}

		profileName := flagAuthName
		if profileName == "" {
			profileName = "default"
		}

		// Optional OAuth 2.0 PKCE Flow
		clientID := flagAuthClientID
		clientSecret := flagAuthClientSecret
		if clientID == "" {
			clientID = os.Getenv("LINKEDIN_CLIENT_ID")
		}
		if clientSecret == "" {
			clientSecret = os.Getenv("LINKEDIN_CLIENT_SECRET")
		}

		if clientID == "" {
			clientID = "ldin-client-id"
		}

		var scopes []string
		if flagAuthScopes != "" {
			scopes = strings.Split(flagAuthScopes, ",")
		} else {
			scopes = auth.DefaultScopes
		}

		Formatter.Info("Opening LinkedIn authorization page in browser on port %d...", flagAuthPort)
		tok, userInfo, err := auth.StartOAuthFlow(clientID, clientSecret, scopes, flagAuthPort)
		if err != nil {
			return fmt.Errorf("authentication error: %w", err)
		}

		err = auth.SaveSession(ConfigMgr, profileName, tok, userInfo, clientID, clientSecret)
		if err != nil {
			return fmt.Errorf("failed storing credentials: %w", err)
		}

		Formatter.Success("Authentication complete! Active profile: %s", profileName)
		Formatter.PrintKeyValue("Member", userInfo.Name)
		Formatter.PrintKeyValue("Member URN", userInfo.Sub)
		return nil
	},
}

var authStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "View active authentication status, scopes, and token validity",
	RunE: func(cmd *cobra.Command, args []string) error {
		profileName := flagProfileName
		if profileName == "" {
			profileName = AppCfg.ActiveProfile
		}

		// Check env var first
		if envTok := os.Getenv("LINKEDIN_TOKEN"); envTok != "" {
			fmt.Println(output.TitleStyle.Render(" ldin Authentication Status "))
			Formatter.PrintKeyValue("Source", "LINKEDIN_TOKEN environment variable")
			Formatter.PrintKeyValue("Token", fmt.Sprintf("Active (Length: %d chars)", len(envTok)))
			return nil
		}

		creds, err := ConfigMgr.LoadProfile(profileName)
		if err != nil {
			Formatter.Error("Not logged in. Run 'ldin auth token <token>' or export LINKEDIN_TOKEN.")
			return nil
		}

		remaining := time.Until(time.Unix(creds.ExpiresAt, 0))
		statusText := "valid"
		if remaining <= 0 {
			statusText = "expired (run 'ldin auth token' or 'ldin auth refresh')"
		} else {
			statusText = fmt.Sprintf("valid (%d days remaining)", int(remaining.Hours()/24))
		}

		data := map[string]interface{}{
			"authenticated":   true,
			"profile_name":    creds.Name,
			"member_name":     creds.DisplayName,
			"member_urn":      creds.MemberURN,
			"email":           creds.Email,
			"token_status":    statusText,
			"granted_scopes":  creds.Scopes,
		}

		return Formatter.Print(data, func() {
			fmt.Println(output.TitleStyle.Render(" ldin Authentication Status "))
			Formatter.PrintKeyValue("Profile", creds.Name)
			Formatter.PrintKeyValue("Member", creds.DisplayName)
			Formatter.PrintKeyValue("Member URN", creds.MemberURN)
			Formatter.PrintKeyValue("Email", creds.Email)
			Formatter.PrintKeyValue("Token", statusText)
			fmt.Println()

			fmt.Println(output.HeaderStyle.Render("Granted Scopes:"))
			for _, s := range creds.Scopes {
				fmt.Printf("  %s %s\n", output.SuccessBadge.Render("✓"), s)
			}
		})
	},
}

var authScopesCmd = &cobra.Command{
	Use:   "scopes",
	Short: "Audit available vs restricted LinkedIn scopes for active session",
	RunE: func(cmd *cobra.Command, args []string) error {
		var scopes []string
		if LinkedInClient != nil && LinkedInClient.Profile != nil {
			scopes = LinkedInClient.Profile.Scopes
		}

		eval := capabilities.EvaluateCapabilities(scopes)
		return Formatter.Print(eval, func() {
			fmt.Println(output.TitleStyle.Render(" LinkedIn API Capability & Scope Matrix "))

			var rows [][]string
			for _, item := range eval {
				status := output.SuccessBadge.Render("✓ Available")
				if !item.Available {
					status = output.WarningBadge.Render("✗ Restricted")
				}
				rows = append(rows, []string{
					item.Capability.Category,
					item.Capability.Name,
					status,
					string(item.Capability.Tier),
					strings.Join(item.Capability.RequiredScopes, ", "),
				})
			}

			Formatter.PrintTable([]string{"Category", "Capability", "Status", "Access Tier", "Required Scopes"}, rows)
		})
	},
}

var authWhoamiCmd = &cobra.Command{
	Use:   "whoami",
	Short: "Display authenticated LinkedIn member identity",
	RunE: func(cmd *cobra.Command, args []string) error {
		if LinkedInClient == nil || LinkedInClient.Profile == nil {
			Formatter.Error("Not logged in. Run 'ldin auth login'.")
			return nil
		}
		p := LinkedInClient.Profile
		data := map[string]string{
			"profile":    p.Name,
			"member_urn": p.MemberURN,
			"name":       p.DisplayName,
			"email":      p.Email,
		}
		return Formatter.Print(data, func() {
			fmt.Printf("%s (%s) — %s\n", lipgloss.NewStyle().Bold(true).Render(p.DisplayName), p.Email, p.MemberURN)
		})
	},
}

var authRefreshCmd = &cobra.Command{
	Use:   "refresh",
	Short: "Refresh LinkedIn OAuth access token",
	RunE: func(cmd *cobra.Command, args []string) error {
		if LinkedInClient == nil || LinkedInClient.Profile == nil {
			return fmt.Errorf("no active profile to refresh")
		}
		p := LinkedInClient.Profile
		if p.RefreshToken == "" {
			return fmt.Errorf("no refresh token stored for profile '%s'. Run 'ldin auth login' to re-authenticate", p.Name)
		}

		tok, err := auth.RefreshToken(p.ClientID, p.ClientSecret, p.RefreshToken)
		if err != nil {
			return fmt.Errorf("token refresh failed: %w", err)
		}

		p.AccessToken = tok.AccessToken
		if tok.RefreshToken != "" {
			p.RefreshToken = tok.RefreshToken
		}
		p.ExpiresAt = time.Now().Unix() + tok.ExpiresIn

		err = ConfigMgr.SaveProfile(p)
		if err != nil {
			return err
		}

		Formatter.Success("Access token refreshed successfully for profile '%s'!", p.Name)
		return nil
	},
}

var authLogoutCmd = &cobra.Command{
	Use:   "logout",
	Short: "Log out and remove stored profile credentials",
	RunE: func(cmd *cobra.Command, args []string) error {
		target := flagAuthName
		if target == "" {
			target = AppCfg.ActiveProfile
		}

		err := ConfigMgr.DeleteProfile(target)
		if err != nil {
			return fmt.Errorf("failed deleting profile '%s': %w", target, err)
		}

		Formatter.Success("Logged out and deleted credentials for profile '%s'.", target)
		return nil
	},
}

var authSwitchCmd = &cobra.Command{
	Use:   "switch [profile-name]",
	Short: "Switch the active default profile identity",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]
		_, err := ConfigMgr.LoadProfile(name)
		if err != nil {
			return fmt.Errorf("profile '%s' does not exist. Run 'ldin auth login --name %s'", name, name)
		}

		AppCfg.ActiveProfile = name
		err = ConfigMgr.SaveConfig(AppCfg)
		if err != nil {
			return err
		}

		Formatter.Success("Switched active profile to '%s'.", name)
		return nil
	},
}

func init() {
	authTokenCmd.Flags().StringVar(&flagAuthName, "name", "default", "Profile name alias")
	authTokenCmd.Flags().StringVar(&flagAuthToken, "token", "", "Token string (or pass as first argument)")

	authLoginCmd.Flags().StringVar(&flagAuthToken, "token", "", "Direct personal access token")
	authLoginCmd.Flags().BoolVar(&flagAuthBrowser, "browser", false, "Launch interactive browser OAuth flow")
	authLoginCmd.Flags().StringVar(&flagAuthClientID, "client-id", "", "LinkedIn App Client ID")
	authLoginCmd.Flags().StringVar(&flagAuthClientSecret, "client-secret", "", "LinkedIn App Client Secret")
	authLoginCmd.Flags().IntVar(&flagAuthPort, "port", auth.DefaultPort, "Local redirect listener port")
	authLoginCmd.Flags().StringVar(&flagAuthName, "name", "default", "Profile name alias for multiple identities")
	authLoginCmd.Flags().StringVar(&flagAuthScopes, "scopes", "", "Comma-separated list of scopes to request")

	authLogoutCmd.Flags().StringVar(&flagAuthName, "name", "", "Specific profile to delete")

	authCmd.AddCommand(authTokenCmd)
	authCmd.AddCommand(authLoginCmd)
	authCmd.AddCommand(authStatusCmd)
	authCmd.AddCommand(authScopesCmd)
	authCmd.AddCommand(authWhoamiCmd)
	authCmd.AddCommand(authRefreshCmd)
	authCmd.AddCommand(authLogoutCmd)
	authCmd.AddCommand(authSwitchCmd)

	RootCmd.AddCommand(authCmd)
}
