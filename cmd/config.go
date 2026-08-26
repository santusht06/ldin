// Copyright (c) 2026 Santusht Kotai
// SPDX-License-Identifier: MIT

package cmd

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"github.com/santusht/ldin/internal/output"
)

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Manage ldin CLI settings and preferences",
	Long:  `View and modify configuration in ~/.ldin/config.yaml.`,
}

var configListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all configuration keys and values",
	RunE: func(cmd *cobra.Command, args []string) error {
		return Formatter.Print(AppCfg, func() {
			fmt.Println(output.TitleStyle.Render(" ldin Configuration "))
			Formatter.PrintKeyValue("Config File", ConfigMgr.ConfigFilePath())
			Formatter.PrintKeyValue("Active Profile", AppCfg.ActiveProfile)
			Formatter.PrintKeyValue("Output Format", AppCfg.OutputFormat)
			Formatter.PrintKeyValue("LinkedIn API Version", AppCfg.LinkedInAPIVersion)
			Formatter.PrintKeyValue("Editor", AppCfg.Editor)
			Formatter.PrintKeyValue("AI Provider", AppCfg.AI.Provider)
			Formatter.PrintKeyValue("AI Model", AppCfg.AI.Model)
			Formatter.PrintKeyValue("Agent Auto-Publish", fmt.Sprintf("%v", AppCfg.Agent.AutoPublish))
			Formatter.PrintKeyValue("Agent Scopes", strings.Join(AppCfg.Agent.AllowedScopes, ", "))
		})
	},
}

var configGetCmd = &cobra.Command{
	Use:   "get <key>",
	Args:  cobra.ExactArgs(1),
	Short: "Get the value of a configuration key",
	RunE: func(cmd *cobra.Command, args []string) error {
		key := strings.ToLower(args[0])
		val := ""

		switch key {
		case "active_profile", "profile":
			val = AppCfg.ActiveProfile
		case "output_format", "output":
			val = AppCfg.OutputFormat
		case "linkedin_api_version", "api_version":
			val = AppCfg.LinkedInAPIVersion
		case "editor":
			val = AppCfg.Editor
		case "ai.provider", "ai_provider":
			val = AppCfg.AI.Provider
		case "ai.model", "ai_model":
			val = AppCfg.AI.Model
		case "ai.api_key", "ai_key":
			val = AppCfg.AI.APIKey
		default:
			return fmt.Errorf("unknown configuration key '%s'", key)
		}

		if Formatter.Format == output.FormatJSON {
			return Formatter.Print(map[string]string{key: val}, nil)
		}
		fmt.Println(val)
		return nil
	},
}

var configSetCmd = &cobra.Command{
	Use:   "set <key> <value>",
	Args:  cobra.ExactArgs(2),
	Short: "Set a configuration key to a new value",
	RunE: func(cmd *cobra.Command, args []string) error {
		key := strings.ToLower(args[0])
		val := args[1]

		switch key {
		case "active_profile", "profile":
			AppCfg.ActiveProfile = val
		case "output_format", "output":
			AppCfg.OutputFormat = val
		case "linkedin_api_version", "api_version":
			AppCfg.LinkedInAPIVersion = val
		case "editor":
			AppCfg.Editor = val
		case "ai.provider", "ai_provider":
			AppCfg.AI.Provider = val
		case "ai.model", "ai_model":
			AppCfg.AI.Model = val
		case "ai.api_key", "ai_key":
			AppCfg.AI.APIKey = val
		case "ai.base_url", "ai_base_url":
			AppCfg.AI.BaseURL = val
		default:
			return fmt.Errorf("unknown configuration key '%s'", key)
		}

		err := ConfigMgr.SaveConfig(AppCfg)
		if err != nil {
			return fmt.Errorf("failed saving config: %w", err)
		}

		Formatter.Success("Config '%s' set to '%s'", key, val)
		return nil
	},
}

var configPathCmd = &cobra.Command{
	Use:   "path",
	Short: "Print the path to the configuration file",
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Println(ConfigMgr.ConfigFilePath())
		return nil
	},
}

func init() {
	configCmd.AddCommand(configListCmd)
	configCmd.AddCommand(configGetCmd)
	configCmd.AddCommand(configSetCmd)
	configCmd.AddCommand(configPathCmd)

	RootCmd.AddCommand(configCmd)
}
