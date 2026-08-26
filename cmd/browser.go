// Copyright (c) 2026 Santusht Kotai
// SPDX-License-Identifier: MIT

package cmd

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/spf13/cobra"
	"github.com/santusht/ldin/internal/cdp"
	"github.com/santusht/ldin/internal/output"
)

var (
	flagBrowserPort int
	flagBrowserHost string
)

var browserCmd = &cobra.Command{
	Use:   "browser",
	Short: "Chrome DevTools Protocol bridge — control LinkedIn via your regular Chrome browser",
	Long: `ldin browser connects to your regular Google Chrome browser via Chrome DevTools Protocol (CDP).
This gives ldin full access to your logged-in LinkedIn session with genuine TLS fingerprints,
bypassing all bot-detection mechanisms.

Commands:
  ldin browser launch     Launch Google Chrome with CDP remote debugging enabled
  ldin browser status     Check connection to your regular Chrome browser
  ldin browser test       Test fetching real-time data from your active LinkedIn tab`,
}

var browserLaunchCmd = &cobra.Command{
	Use:   "launch",
	Short: "Launch Google Chrome with remote debugging on port 9222",
	Long: `Starts Google Chrome with --remote-debugging-port=9222.
If Chrome is already open, it will notify you to restart Chrome with debugging enabled.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		port := flagBrowserPort

		// Check if already active
		tabs, err := cdp.ListTabs(flagBrowserHost, port)
		if err == nil && len(tabs) > 0 {
			Formatter.Success("Chrome CDP is already active on port %d (%d tabs open)!", port, len(tabs))
			return nil
		}

		Formatter.Info("Launching Google Chrome with remote debugging on port %d...", port)

		userDataDir := filepath.Join(os.Getenv("HOME"), ".ldin", "chrome_session")
		_ = os.MkdirAll(userDataDir, 0755)

		chromePaths := []string{
			"/Applications/Google Chrome.app/Contents/MacOS/Google Chrome",
			"/Applications/Chromium.app/Contents/MacOS/Chromium",
			"/Applications/Brave Browser.app/Contents/MacOS/Brave Browser",
		}

		launched := false
		for _, path := range chromePaths {
			c := exec.Command(path,
				fmt.Sprintf("--remote-debugging-port=%d", port),
				fmt.Sprintf("--user-data-dir=%s", userDataDir),
				"--no-first-run",
				"--no-default-browser-check",
				"https://www.linkedin.com",
			)
			if err := c.Start(); err == nil {
				launched = true
				break
			}
		}

		if !launched {
			c := exec.Command("open", "-a", "Google Chrome", "--args",
				fmt.Sprintf("--remote-debugging-port=%d", port),
				fmt.Sprintf("--user-data-dir=%s", userDataDir),
				"https://www.linkedin.com",
			)
			_ = c.Start()
		}

		// Wait briefly and verify
		time.Sleep(2000 * time.Millisecond)
		tabs, err = cdp.ListTabs(flagBrowserHost, port)
		if err == nil {
			Formatter.Success("Google Chrome connected on port %d ✓", port)
			Formatter.Info("LinkedIn is opening in Chrome.")
			Formatter.Info("Once you are logged into LinkedIn in that window, run: ldin profile show")
		} else {
			Formatter.Warning("Chrome launched. If not connected, start with:")
			fmt.Printf("\n  /Applications/Google\\ Chrome.app/Contents/MacOS/Google\\ Chrome --remote-debugging-port=%d --user-data-dir=\"%s\" https://www.linkedin.com &\n\n", port, userDataDir)
		}

		return nil
	},
}

var browserStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Check connection to your regular Chrome browser",
	RunE: func(cmd *cobra.Command, args []string) error {
		port := flagBrowserPort

		fmt.Println(output.TitleStyle.Render(" Chrome Browser Bridge Status "))

		tabs, err := cdp.ListTabs(flagBrowserHost, port)
		if err != nil {
			Formatter.PrintKeyValue("Connection", "✗ Disconnected")
			Formatter.PrintKeyValue("Port", fmt.Sprintf("%s:%d", flagBrowserHost, port))
			fmt.Printf("\n  👉 To enable Chrome bridge:\n")
			fmt.Printf("  1. Quit Chrome:  pkill -x 'Google Chrome'\n")
			fmt.Printf("  2. Start Chrome: open -a 'Google Chrome' --args --remote-debugging-port=%d\n\n", port)
			return nil
		}

		Formatter.PrintKeyValue("Connection", "✓ Connected")
		Formatter.PrintKeyValue("CDP Address", fmt.Sprintf("http://%s:%d", flagBrowserHost, port))
		Formatter.PrintKeyValue("Open Tabs", fmt.Sprintf("%d", len(tabs)))

		linkedInTabFound := false
		for _, t := range tabs {
			if t.Type == "page" {
				icon := "  "
				if containsString(t.URL, "linkedin.com") {
					icon = "🔗"
					linkedInTabFound = true
				}
				fmt.Printf("  %s %s\n     %s\n", icon, t.Title, t.URL)
			}
		}

		fmt.Println()
		if linkedInTabFound {
			Formatter.Success("Active LinkedIn tab detected! Ready for real-time operations.")
			fmt.Printf("  Try: ldin profile show\n\n")
		} else {
			Formatter.Warning("No LinkedIn tab open in Chrome. Navigate to linkedin.com in Chrome for best results.")
		}

		return nil
	},
}

var browserTestCmd = &cobra.Command{
	Use:   "test",
	Short: "Test real-time LinkedIn communication via your regular Chrome browser",
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		defer cancel()

		port := flagBrowserPort
		Formatter.Info("Connecting to Chrome on %s:%d...", flagBrowserHost, port)

		tabs, err := cdp.ListTabs(flagBrowserHost, port)
		if err != nil {
			return fmt.Errorf("could not connect to Chrome on port %d: %w\n\nRun: ldin browser launch", port, err)
		}

		tab := cdp.FindLinkedInTab(tabs)
		if tab == nil {
			return fmt.Errorf("no open tab found in Chrome.\n\nPlease open linkedin.com in your Chrome browser")
		}

		Formatter.Info("Connecting to Chrome tab: %s", tab.Title)
		bridge, err := cdp.Connect(ctx, tab)
		if err != nil {
			return fmt.Errorf("CDP connection failed: %w", err)
		}
		defer bridge.Close()

		Formatter.Info("Querying active session from Chrome...")
		profile, err := cdp.FetchFullProfileView(ctx, bridge, "")
		if err != nil {
			return fmt.Errorf("failed fetching profile via Chrome: %w", err)
		}

		Formatter.Success("Communication with regular Chrome is working!")
		Formatter.PrintKeyValue("Name", profile.FirstName+" "+profile.LastName)
		Formatter.PrintKeyValue("Vanity URL", profile.VanityName)
		Formatter.PrintKeyValue("Headline", profile.Headline)
		Formatter.PrintKeyValue("Skills count", fmt.Sprintf("%d", len(profile.Skills)))
		Formatter.PrintKeyValue("Experience count", fmt.Sprintf("%d", len(profile.Experience)))
		fmt.Printf("\n  Run `ldin profile show` to see the full rendered profile.\n\n")
		return nil
	},
}

func containsString(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || len(s) > 0 && containsSub(s, sub))
}

func containsSub(s, sub string) bool {
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

	browserCmd.AddCommand(browserLaunchCmd)
	browserCmd.AddCommand(browserStatusCmd)
	browserCmd.AddCommand(browserTestCmd)

	RootCmd.AddCommand(browserCmd)
}
