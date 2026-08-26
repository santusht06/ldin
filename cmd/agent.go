// Copyright (c) 2026 Santusht Kotai
// SPDX-License-Identifier: MIT

package cmd

import (
	"context"
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"
	"github.com/santusht/ldin/internal/agent"
	"github.com/santusht/ldin/internal/output"
)

var agentCmd = &cobra.Command{
	Use:   "agent",
	Short: "Autonomous ReAct Agent for executing end-to-end LinkedIn workflows",
	Long: `The ldin autonomous agent can inspect your workspace, observe git history,
reason about content opportunities, draft posts, and execute multi-step workflows.
Enforces strict safety boundaries and explicit user consent.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return agentPermissionsCmd.RunE(cmd, args)
	},
}

var agentRunCmd = &cobra.Command{
	Use:   "run [instruction]",
	Args:  cobra.MinimumNArgs(1),
	Short: "Run an autonomous agent task loop with safety confirmations",
	RunE: func(cmd *cobra.Command, args []string) error {
		instruction := strings.Join(args, " ")
		Formatter.Info("Starting autonomous agent with goal: %s", instruction)

		eng, err := agent.NewEngine(ConfigMgr, LinkedInClient)
		if err != nil {
			return err
		}

		ctx := context.Background()
		result, err := eng.Run(ctx, instruction)
		if err != nil {
			return err
		}

		return Formatter.Print(result, func() {
			fmt.Println(output.TitleStyle.Render(" Agent Execution Finished "))
			fmt.Println(output.HeaderStyle.Render("Action Taken:"))
			fmt.Printf("  %s\n\n", result.ActionTaken)

			fmt.Println(output.HeaderStyle.Render("Agent Output:"))
			fmt.Println(result.Response)

			if result.DraftID != "" {
				fmt.Println()
				Formatter.Success("Post ready for review: `ldin post preview %s`", result.DraftID)
			}
		})
	},
}

var agentPermissionsCmd = &cobra.Command{
	Use:   "permissions",
	Short: "Inspect active safety permission boundaries for the autonomous agent",
	RunE: func(cmd *cobra.Command, args []string) error {
		guard := agent.NewPermissionGuard(&AppCfg.Agent)
		status := guard.GetStatus()

		return Formatter.Print(status, func() {
			fmt.Println(output.TitleStyle.Render(" Agent Safety & Permissions "))
			for scope, allowed := range status {
				badge := output.SuccessBadge.Render("✓ ALLOWED")
				if !allowed {
					badge = output.WarningBadge.Render("✗ BLOCKED")
				}
				fmt.Printf("  %s  %s\n", badge, scope)
			}
			fmt.Println()
			Formatter.Info("Use `ldin agent allow <scope>` or `ldin agent deny <scope>` to adjust boundaries.")
		})
	},
}

var agentAllowCmd = &cobra.Command{
	Use:   "allow <scope>",
	Args:  cobra.ExactArgs(1),
	Short: "Grant permission to the agent (e.g. publish, delete)",
	RunE: func(cmd *cobra.Command, args []string) error {
		scope := strings.ToLower(args[0])

		for _, s := range AppCfg.Agent.AllowedScopes {
			if s == scope {
				Formatter.Info("Scope '%s' is already allowed.", scope)
				return nil
			}
		}

		AppCfg.Agent.AllowedScopes = append(AppCfg.Agent.AllowedScopes, scope)
		if scope == "publish" {
			AppCfg.Agent.AutoPublish = true
		}

		err := ConfigMgr.SaveConfig(AppCfg)
		if err != nil {
			return err
		}

		Formatter.Success("Permission '%s' granted to ldin agent.", scope)
		return nil
	},
}

var agentDenyCmd = &cobra.Command{
	Use:   "deny <scope>",
	Args:  cobra.ExactArgs(1),
	Short: "Revoke permission from the agent",
	RunE: func(cmd *cobra.Command, args []string) error {
		scope := strings.ToLower(args[0])

		var updated []string
		for _, s := range AppCfg.Agent.AllowedScopes {
			if s != scope {
				updated = append(updated, s)
			}
		}
		AppCfg.Agent.AllowedScopes = updated
		if scope == "publish" {
			AppCfg.Agent.AutoPublish = false
		}

		err := ConfigMgr.SaveConfig(AppCfg)
		if err != nil {
			return err
		}

		Formatter.Success("Permission '%s' revoked from ldin agent.", scope)
		return nil
	},
}

var agentToolsCmd = &cobra.Command{
	Use:   "tools",
	Short: "List all tools registered in the agent engine",
	RunE: func(cmd *cobra.Command, args []string) error {
		eng, err := agent.NewEngine(ConfigMgr, LinkedInClient)
		if err != nil {
			return err
		}

		toolsList := eng.Tools.List()
		return Formatter.Print(toolsList, func() {
			fmt.Println(output.TitleStyle.Render(" Registered Agent Tools "))
			for _, t := range toolsList {
				name := lipgloss.NewStyle().Bold(true).Foreground(output.AccentCyan).Render(t["name"])
				fmt.Printf("  • %s\n    %s\n\n", name, output.DimStyle.Render(t["description"]))
			}
		})
	},
}

func init() {
	agentCmd.AddCommand(agentRunCmd)
	agentCmd.AddCommand(agentPermissionsCmd)
	agentCmd.AddCommand(agentAllowCmd)
	agentCmd.AddCommand(agentDenyCmd)
	agentCmd.AddCommand(agentToolsCmd)

	RootCmd.AddCommand(agentCmd)
}
