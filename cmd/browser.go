// Copyright (c) 2026 Santusht Kotai
// SPDX-License-Identifier: MIT

package cmd

import (
	"context"
	"fmt"
	"os/exec"
	"time"

	"github.com/spf13/cobra"
	"github.com/santusht/ldin/internal/cdp"
)

var (
	flagBrowserPort int
	flagBrowserHost string
)

var browserCmd = &cobra.Command{
	Use:   "browser",
	Short: "Chrome DevTools Protocol bridge — connect ldin to your real Chrome session",
	Long: `ldin browser connects to your running Chrome browser via Chrome DevTools Protocol (CDP).
This lets ldin make LinkedIn API requests INSIDE Chrome's real session, bypassing TLS fingerprinting.

Usage Flow:
  1. ldin browser launch    — Launch Chrome with remote debugging enabled
  2. ldin browser status    — Check if CDP bridge is reachable
  3. ldin profile show      — Now fetches real-time data via your Chrome session`,
}

var browserLaunchCmd = &cobra.Command{
	Use:   "launch",
	Short: "Launch Chrome with remote debugging enabled on port 9222",
	Long: `Launches Google Chrome with --remote-debugging-port=9222.
Once Chrome is running, ldin can connect to your real browser session and 
make authenticated LinkedIn requests without any TLS fingerprinting issues.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		port := flagBrowserPort

		Formatter.Info("Launching Chrome with CDP on port %d...", port)

		// Check if already running
		tabs, err := cdp.ListTabs(flagBrowserHost, port)
		if err == nil && len(tabs) > 0 {
			Formatter.Success("Chrome CDP is already running on port %d — %d tab(s) detected.", port, len(tabs))
			return nil
		}

		// Launch Chrome with remote debugging
		chromePaths := []string{
			"/Applications/Google Chrome.app/Contents/MacOS/Google Chrome",
			"/Applications/Chromium.app/Contents/MacOS/Chromium",
			"/Applications/Brave Browser.app/Contents/MacOS/Brave Browser",
		}

		launched := false
		for _, path := range chromePaths {
			c := exec.Command(path,
				fmt.Sprintf("--remote-debugging-port=%d", port),
				"--no-first-run",
				"--no-default-browser-check",
				"https://www.linkedin.com",
			)
			if err := c.Start(); err == nil {
				Formatter.Success("Chrome launched with CDP on port %d", port)
				Formatter.Info("LinkedIn.com is opening in your browser...")
				Formatter.Info("Once you see LinkedIn, run: ldin profile show")
				launched = true
				break
			}
		}

		if !launched {
			fmt.Printf("\n  Chrome not found at standard paths.\n")
			fmt.Printf("  Start Chrome manually with:\n\n")
			fmt.Printf("  open -a 'Google Chrome' --args --remote-debugging-port=%d\n\n", port)
		}

		return nil
	},
}

var browserStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Check if Chrome CDP bridge is active and reachable",
	RunE: func(cmd *cobra.Command, args []string) error {
		port := flagBrowserPort

		tabs, err := cdp.ListTabs(flagBrowserHost, port)
		if err != nil {
			Formatter.Error("Chrome CDP not reachable on %s:%d", flagBrowserHost, port)
			fmt.Printf("\n  To launch Chrome with CDP:\n\n")
			fmt.Printf("  ldin browser launch\n\n")
			fmt.Printf("  Or manually:\n")
			fmt.Printf("  open -a 'Google Chrome' --args --remote-debugging-port=%d\n\n", port)
			return nil
		}

		fmt.Println(OutputTitleStyle(" Chrome CDP Bridge Status "))
		Formatter.PrintKeyValue("CDP Port", fmt.Sprintf("%s:%d", flagBrowserHost, port))
		Formatter.PrintKeyValue("Status", "✓ Connected")
		Formatter.PrintKeyValue("Open Tabs", fmt.Sprintf("%d", len(tabs)))

		linkedInFound := false
		for _, t := range tabs {
			if t.Type == "page" {
				icon := "  "
				if containsStr(t.URL, "linkedin.com") {
					icon = "🔗"
					linkedInFound = true
				}
				fmt.Printf("  %s  %s\n     %s\n", icon, t.Title, t.URL)
			}
		}

		if linkedInFound {
			Formatter.Success("LinkedIn tab detected — ldin profile show will use real-time data!")
		} else {
			Formatter.Warning("No LinkedIn tab open. Open linkedin.com for best results.")
		}

		return nil
	},
}

var browserTestCmd = &cobra.Command{
	Use:   "test",
	Short: "Test the CDP bridge by fetching your LinkedIn profile via Chrome",
	RunE: func(cmd *cobra.Command, args []string) error {
		port := flagBrowserPort

		Formatter.Info("Connecting to Chrome CDP on %s:%d...", flagBrowserHost, port)
		tabs, err := cdp.ListTabs(flagBrowserHost, port)
		if err != nil {
			return err
		}

		tab := cdp.FindLinkedInTab(tabs)
		if tab == nil {
			return fmt.Errorf("no usable Chrome tab found. Run `ldin browser launch` first")
		}

		Formatter.Info("Connecting to tab: %s", tab.Title)
		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		defer cancel()

		bridge, err := cdp.Connect(ctx, tab)
		if err != nil {
			return err
		}
		defer bridge.Close()

		Formatter.Info("Executing test fetch via Chrome's real session...")
		result, err := bridge.Eval(ctx, `
(async () => {
  const csrf = (document.cookie.match(/JSESSIONID="?([^";]+)"?/) || [])[1] || 'none';
  const resp = await fetch('/voyager/api/me', {
    credentials: 'include',
    headers: {
      'Accept': 'application/vnd.linkedin.normalized+json+2.1',
      'X-Li-Lang': 'en_US',
      'X-Requested-With': 'XMLHttpRequest',
      'Csrf-Token': csrf,
    }
  });
  const data = await resp.json();
  return JSON.stringify({status: resp.status, csrf_found: csrf !== 'none', data: data});
})()
`)
		if err != nil {
			return fmt.Errorf("CDP test failed: %w", err)
		}

		Formatter.Success("CDP bridge is working!")
		fmt.Printf("\nRaw response (first 500 chars):\n%.500s\n", result)
		return nil
	},
}

func OutputTitleStyle(s string) string {
	return "\n  " + s + "\n"
}

func containsStr(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || len(s) > 0 && containsSubstr(s, sub))
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

	browserCmd.AddCommand(browserLaunchCmd)
	browserCmd.AddCommand(browserStatusCmd)
	browserCmd.AddCommand(browserTestCmd)

	RootCmd.AddCommand(browserCmd)
}
