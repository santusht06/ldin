// Copyright (c) 2026 Santusht Kotai
// SPDX-License-Identifier: MIT

package cmd

import (
	"context"
	"fmt"
	"time"

	"github.com/go-rod/rod/lib/launcher"
	"github.com/spf13/cobra"
	"github.com/santusht/ldin/internal/browser"
	"github.com/santusht/ldin/internal/cdp"
)

var (
	flagBrowserPort     int
	flagBrowserHost     string
	flagBrowserHeadless bool
)

var browserCmd = &cobra.Command{
	Use:   "browser",
	Short: "Headless Chrome engine — real-time LinkedIn data without opening a window",
	Long: `ldin browser runs Chromium invisibly inside your terminal.
No GUI window. No manual Chrome setup. Just pure terminal.

  ldin browser setup      Download Chromium (first time only)
  ldin browser test       Run a headless fetch test against LinkedIn
  ldin browser status     Check if CDP bridge is reachable
  ldin browser launch     Launch Chrome with remote debugging (optional GUI mode)`,
}

var browserSetupCmd = &cobra.Command{
	Use:   "setup",
	Short: "Download headless Chromium (run once — ~120MB)",
	Long: `Downloads the headless Chromium binary for your platform.
This is required once. After setup, all ldin commands that need a real browser
will use this invisible Chromium instance automatically.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		Formatter.Info("Downloading headless Chromium for your platform...")
		Formatter.Info("This is a one-time download (~120MB). Please wait...")

		pather := launcher.NewBrowser()
		if err := pather.Download(); err != nil {
			return fmt.Errorf("download failed: %w", err)
		}

		Formatter.Success("Chromium downloaded successfully!")
		Formatter.PrintKeyValue("Location", pather.BinPath())
		fmt.Printf("\n  You can now run:\n  ldin browser test\n  ldin profile show\n\n")
		return nil
	},
}

var browserTestCmd = &cobra.Command{
	Use:   "test",
	Short: "Test headless Chromium by fetching your LinkedIn profile invisibly",
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		if !flagBrowserHeadless {
			// Try CDP to existing Chrome first
			Formatter.Info("Checking for existing Chrome CDP on port %d...", flagBrowserPort)
			tabs, err := cdp.ListTabs(flagBrowserHost, flagBrowserPort)
			if err == nil && len(tabs) > 0 {
				tab := cdp.FindLinkedInTab(tabs)
				if tab != nil {
					bridge, err := cdp.Connect(ctx, tab)
					if err == nil {
						defer bridge.Close()
						result, err := bridge.Eval(ctx, `
(async () => {
  const csrf = (document.cookie.match(/JSESSIONID="?([^";]+)"?/) || [])[1] || 'none';
  const resp = await fetch('/voyager/api/me', {
    credentials: 'include',
    headers: {'Accept': 'application/vnd.linkedin.normalized+json+2.1', 'X-Requested-With': 'XMLHttpRequest', 'Csrf-Token': csrf}
  });
  return JSON.stringify({status: resp.status, ok: resp.ok, csrf_found: csrf !== 'none'});
})()
`)
						if err == nil {
							Formatter.Success("CDP bridge working via existing Chrome!")
							fmt.Printf("Response: %s\n", result)
							return nil
						}
					}
				}
			}
		}

		// Headless Chromium test
		Formatter.Info("Launching headless Chromium...")
		hb, err := browser.Launch(ctx)
		if err != nil {
			return fmt.Errorf("%w\n\nRun first: ldin browser setup", err)
		}
		defer hb.Close()

		Formatter.Info("Injecting your LinkedIn session cookie...")
		creds, credErr := ConfigMgr.LoadProfile(AppCfg.ActiveProfile)
		if credErr != nil || creds.SessionCookie == "" {
			return fmt.Errorf("not authenticated. Run: ldin auth token <your_li_at_token>")
		}

		if err := hb.InjectSession(creds.SessionCookie, creds.CSRFToken); err != nil {
			return fmt.Errorf("failed injecting session: %w", err)
		}

		Formatter.Info("Opening LinkedIn silently in headless Chromium...")
		if err := hb.OpenPage(ctx, "https://www.linkedin.com"); err != nil {
			return fmt.Errorf("failed to load LinkedIn: %w", err)
		}

		vanityName, name, _ := hb.GetCurrentUser(ctx)
		if name != "" || vanityName != "" {
			Formatter.Success("Headless Chromium authenticated as: %s (%s)", name, vanityName)
		} else {
			Formatter.Warning("Session may have expired. Try: ldin auth token <fresh_li_at>")
		}

		Formatter.Info("Fetching profile invisibly via headless Chromium...")
		profile, err := hb.FetchVoyagerProfile(ctx, vanityName)
		if err != nil {
			return fmt.Errorf("headless profile fetch failed: %w", err)
		}

		Formatter.Success("Headless browser is working!")
		Formatter.PrintKeyValue("Name", profile.FirstName+" "+profile.LastName)
		Formatter.PrintKeyValue("Headline", profile.Headline)
		Formatter.PrintKeyValue("Skills fetched", fmt.Sprintf("%d", len(profile.Skills)))
		Formatter.PrintKeyValue("Experience entries", fmt.Sprintf("%d", len(profile.Experience)))
		return nil
	},
}

var browserStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Check browser bridge status (CDP or headless Chromium)",
	RunE: func(cmd *cobra.Command, args []string) error {
		port := flagBrowserPort

		fmt.Println(outputTitleStyle(" ldin Browser Bridge Status "))

		// Check Chromium binary
		pather := launcher.NewBrowser()
		binPath := pather.BinPath()
		if binPath != "" {
			Formatter.PrintKeyValue("Headless Chromium", "✓ Installed at "+binPath)
		} else {
			Formatter.PrintKeyValue("Headless Chromium", "✗ Not installed (run: ldin browser setup)")
		}

		// Check existing Chrome CDP
		tabs, err := cdp.ListTabs(flagBrowserHost, port)
		if err != nil {
			Formatter.PrintKeyValue("Chrome CDP Bridge", fmt.Sprintf("✗ Not running on port %d", port))
		} else {
			Formatter.PrintKeyValue("Chrome CDP Bridge", fmt.Sprintf("✓ Connected — %d tab(s)", len(tabs)))
			for _, t := range tabs {
				if t.Type == "page" {
					icon := "  "
					if containsSubstr(t.URL, "linkedin.com") {
						icon = "🔗"
					}
					fmt.Printf("    %s %s\n", icon, t.Title)
				}
			}
		}

		return nil
	},
}

var browserLaunchCmd = &cobra.Command{
	Use:   "launch",
	Short: "Launch Chrome with remote debugging enabled (optional GUI mode)",
	RunE: func(cmd *cobra.Command, args []string) error {
		port := flagBrowserPort

		if flagBrowserHeadless {
			Formatter.Info("Use `ldin browser test` to run headless Chromium directly.")
			return nil
		}

		Formatter.Info("Launching Chrome with CDP on port %d...", port)
		tabs, err := cdp.ListTabs(flagBrowserHost, port)
		if err == nil && len(tabs) > 0 {
			Formatter.Success("Chrome CDP already running on port %d — %d tab(s) detected.", port, len(tabs))
			return nil
		}

		chromePaths := []string{
			"/Applications/Google Chrome.app/Contents/MacOS/Google Chrome",
			"/Applications/Chromium.app/Contents/MacOS/Chromium",
			"/Applications/Brave Browser.app/Contents/MacOS/Brave Browser",
		}

		for _, path := range chromePaths {
			_ = path
		}
		fmt.Printf("\n  Manually start Chrome with:\n\n")
		fmt.Printf("  open -a 'Google Chrome' --args --remote-debugging-port=%d\n\n", port)
		fmt.Printf("  Or use headless mode (no window needed):\n")
		fmt.Printf("  ldin browser test\n\n")

		return nil
	},
}

func outputTitleStyle(s string) string {
	return "\n  " + s + "\n"
}

func containsSubstr(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}

func init() {
	browserCmd.PersistentFlags().IntVar(&flagBrowserPort, "port", cdp.DefaultCDPPort, "Chrome CDP debugging port")
	browserCmd.PersistentFlags().StringVar(&flagBrowserHost, "host", cdp.DefaultCDPHost, "Chrome CDP host")
	browserCmd.PersistentFlags().BoolVar(&flagBrowserHeadless, "headless", true, "Use headless Chromium (no window)")

	browserCmd.AddCommand(browserSetupCmd)
	browserCmd.AddCommand(browserTestCmd)
	browserCmd.AddCommand(browserStatusCmd)
	browserCmd.AddCommand(browserLaunchCmd)

	RootCmd.AddCommand(browserCmd)
}
