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
)

var authCmd = &cobra.Command{
	Use:   "auth",
	Short: "Manage LinkedIn authentication, tokens, and multi-identity sessions",
	Long:  `Authenticate ldin with LinkedIn via OAuth 2.0 PKCE, view active session, inspect scopes, or manage multiple identities.`,
}

var authLoginCmd = &cobra.Command{
	Use:   "login",
	Short: "Authenticate with LinkedIn",
	Long: `Start an interactive OAuth 2.0 authentication flow in your browser, or supply a pre-generated token.
Example:
  ldin auth login
  ldin auth login --name company --client-id <id> --client-secret <secret>
  ldin auth login --token <access_token>`,
	RunE: func(cmd *cobra.Command, args []string) error {
		profileName := flagAuthName
		if profileName == "" {
			profileName = "default"
		}

		// Direct Token Flow
		if flagAuthToken != "" {
			userInfo, err := auth.FetchUserInfo(flagAuthToken)
			if err != nil {
				Formatter.Warning("Could not fetch user details with provided token (non-fatal): %v", err)
				userInfo = &auth.UserInfoResponse{
					Sub:  "urn:li:person:custom",
					Name: "LinkedIn Member",
				}
			}

			tokResp := &auth.TokenResponse{
				AccessToken: flagAuthToken,
				ExpiresIn:   60 * 24 * 3600,
				Scope:       strings.Join(auth.DefaultScopes, " "),
			}

			err = auth.SaveSession(ConfigMgr, profileName, tokResp, userInfo, flagAuthClientID, flagAuthClientSecret)
			if err != nil {
				return fmt.Errorf("failed saving session: %w", err)
			}

			Formatter.Success("Successfully authenticated profile '%s' for %s (%s)", profileName, userInfo.Name, userInfo.Sub)
			return nil
		}

		// Interactive OAuth 2.0 PKCE Flow
		clientID := flagAuthClientID
		clientSecret := flagAuthClientSecret
		if clientID == "" {
			clientID = os.Getenv("LINKEDIN_CLIENT_ID")
		}
		if clientSecret == "" {
			clientSecret = os.Getenv("LINKEDIN_CLIENT_SECRET")
		}

		if clientID == "" {
			Formatter.Info("No LINKEDIN_CLIENT_ID provided. You can pass --client-id or set LINKEDIN_CLIENT_ID env var.")
			Formatter.Info("Starting browser authorization...")
			clientID = "ldin-client-id" // Fallback standard client
		}

		var scopes []string
		if flagAuthScopes != "" {
			scopes = strings.Split(flagAuthScopes, ",")
		} else {
			scopes = auth.DefaultScopes
		}

		Formatter.Info("Opening LinkedIn authorization page in your browser on port %d...", flagAuthPort)
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
		Formatter.PrintKeyValue("Email", userInfo.Email)
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

		creds, err := ConfigMgr.LoadProfile(profileName)
		if err != nil {
			Formatter.Error("Not logged in. Run 'ldin auth login' to authenticate.")
			return nil
		}

		remaining := time.Until(time.Unix(creds.ExpiresAt, 0))
		statusText := "valid"
		if remaining <= 0 {
			statusText = "expired (run 'ldin auth refresh' or 'ldin auth login')"
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
	authLoginCmd.Flags().StringVar(&flagAuthToken, "token", "", "Direct personal access token for non-interactive or CI usage")
	authLoginCmd.Flags().StringVar(&flagAuthClientID, "client-id", "", "LinkedIn App Client ID")
	authLoginCmd.Flags().StringVar(&flagAuthClientSecret, "client-secret", "", "LinkedIn App Client Secret")
	authLoginCmd.Flags().IntVar(&flagAuthPort, "port", auth.DefaultPort, "Local redirect listener port")
	authLoginCmd.Flags().StringVar(&flagAuthName, "name", "default", "Profile name alias for multiple identities")
	authLoginCmd.Flags().StringVar(&flagAuthScopes, "scopes", "", "Comma-separated list of scopes to request")

	authLogoutCmd.Flags().StringVar(&flagAuthName, "name", "", "Specific profile to delete")

	authCmd.AddCommand(authLoginCmd)
	authCmd.AddCommand(authStatusCmd)
	authCmd.AddCommand(authScopesCmd)
	authCmd.AddCommand(authWhoamiCmd)
	authCmd.AddCommand(authRefreshCmd)
	authCmd.AddCommand(authLogoutCmd)
	authCmd.AddCommand(authSwitchCmd)

	RootCmd.AddCommand(authCmd)
}
