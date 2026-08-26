// Copyright (c) 2026 Santusht Kotai
// SPDX-License-Identifier: MIT

package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/santusht/ldin/internal/config"
	"github.com/santusht/ldin/internal/linkedin"
	"github.com/santusht/ldin/internal/output"
)

var (
	// Global flag holders
	flagProfileName string
	flagOutputJSON  bool
	flagOutputYAML  bool
	flagOutputCSV   bool
	flagQuiet       bool
	flagVerbose     bool
	flagDebug       bool
	flagConfigFile  string

	// Global runtime singletons initialized in PersistentPreRun
	ConfigMgr  *config.ConfigManager
	AppCfg     *config.AppConfig
	Formatter  *output.Formatter
	LinkedInClient *linkedin.Client
)

// RootCmd is the base command for ldin
var RootCmd = &cobra.Command{
	Use:   "ldin",
	Short: "ldin — The Developer-First LinkedIn CLI & AI Platform",
	Long: `ldin is a high-performance command-line workspace for LinkedIn.
Manage your professional identity, Profile-as-Code, content publishing,
social graph, and analytics from the terminal with an autonomous AI agent layer.

Simple things should be simple. Complex things should be possible.`,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		var err error
		ConfigMgr, err = config.NewConfigManager()
		if err != nil {
			return err
		}

		AppCfg, err = ConfigMgr.LoadConfig()
		if err != nil {
			return err
		}

		// Determine output format
		outFormat := AppCfg.OutputFormat
		if flagOutputJSON {
			outFormat = "json"
		} else if flagOutputYAML {
			outFormat = "yaml"
		} else if flagOutputCSV {
			outFormat = "csv"
		} else if flagQuiet {
			outFormat = "quiet"
		}

		Formatter = output.NewFormatter(outFormat, flagVerbose, flagDebug)

		// Profile selection priority: Flag > Config > "default"
		targetProfile := flagProfileName
		if targetProfile == "" && AppCfg.ActiveProfile != "" {
			targetProfile = AppCfg.ActiveProfile
		}

		creds, _ := ConfigMgr.LoadProfile(targetProfile)
		LinkedInClient = linkedin.NewClient(creds, AppCfg.LinkedInAPIVersion)

		return nil
	},
}

func init() {
	RootCmd.PersistentFlags().StringVarP(&flagProfileName, "profile", "p", "", "Active LinkedIn identity/profile name (e.g. personal, company)")
	RootCmd.PersistentFlags().BoolVar(&flagOutputJSON, "json", false, "Output results in formatted JSON for unix scripting")
	RootCmd.PersistentFlags().BoolVar(&flagOutputYAML, "yaml", false, "Output results in YAML")
	RootCmd.PersistentFlags().BoolVar(&flagOutputCSV, "csv", false, "Output tabular results in CSV")
	RootCmd.PersistentFlags().BoolVarP(&flagQuiet, "quiet", "q", false, "Suppress non-essential terminal output")
	RootCmd.PersistentFlags().BoolVarP(&flagVerbose, "verbose", "v", false, "Enable verbose output")
	RootCmd.PersistentFlags().BoolVar(&flagDebug, "debug", false, "Enable debug mode with full HTTP logs")
	RootCmd.PersistentFlags().StringVar(&flagConfigFile, "config", "", "Custom configuration file path")
}

// Execute runs the root CLI command
func Execute() {
	if err := RootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
