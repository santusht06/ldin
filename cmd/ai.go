// Copyright (c) 2026 Santusht Kotai
// SPDX-License-Identifier: MIT

package cmd

import (
	"context"
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"github.com/santusht/ldin/internal/agent"
	"github.com/santusht/ldin/internal/output"
	"github.com/santusht/ldin/internal/tui"
)

var aiCmd = &cobra.Command{
	Use:   "ai [instruction]",
	Short: "AI workspace assistant for generating content, optimizing profiles, and crafting replies",
	Long: `ldin AI is your intelligent LinkedIn copilot.
Run natural language instructions, turn code contributions into posts, optimize your profile headline,
or craft thoughtful replies to comments.

Examples:
  ldin ai "Write a LinkedIn post about my latest open-source Go project"
  ldin ai "Make my profile headline more backend and distributed systems focused"
  ldin ai post "Announcing our new distributed caching layer"
  ldin ai reply urn:li:comment:123 "Thank them for the feedback on database sharding"`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) == 0 {
			return cmd.Help()
		}

		instruction := strings.Join(args, " ")
		Formatter.Info("Thinking & reasoning with ldin AI...")

		eng, err := agent.NewEngine(ConfigMgr, LinkedInClient)
		if err != nil {
			return err
		}

		ctx := context.Background()
		result, err := eng.Run(ctx, instruction)
		if err != nil {
			return fmt.Errorf("AI error: %w", err)
		}

		return Formatter.Print(result, func() {
			fmt.Println(output.TitleStyle.Render(" ldin AI Response "))
			fmt.Println(result.Response)
			fmt.Println()

			if result.DraftID != "" {
				Formatter.Success("Saved as draft: %s", result.DraftID)
				fmt.Println(output.DimStyle.Render(fmt.Sprintf("To preview: ldin post preview %s", result.DraftID)))
				fmt.Println(output.DimStyle.Render(fmt.Sprintf("To publish: ldin post publish %s", result.DraftID)))
			}
		})
	},
}

var aiPostCmd = &cobra.Command{
	Use:   "post [prompt]",
	Short: "Generate an optimized LinkedIn post draft from prompt or git context",
	Args:  cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		prompt := strings.Join(args, " ")
		Formatter.Info("Drafting optimized LinkedIn post...")

		eng, err := agent.NewEngine(ConfigMgr, LinkedInClient)
		if err != nil {
			return err
		}

		ctx := context.Background()
		result, err := eng.Run(ctx, "Create an engaging, technical LinkedIn post about: "+prompt)
		if err != nil {
			return err
		}

		return Formatter.Print(result, func() {
			author := "You (Active Profile)"
			if LinkedInClient != nil && LinkedInClient.Profile != nil {
				author = LinkedInClient.Profile.DisplayName
			}
			fmt.Println(tui.RenderPostPreview(author, result.Response, "PUBLIC 🌐", nil, ""))

			if result.DraftID != "" {
				Formatter.Success("Saved to drafts (~/.ldin/drafts/%s.json)", result.DraftID)
				fmt.Println(output.DimStyle.Render(fmt.Sprintf("Publish when ready: `ldin post publish %s`", result.DraftID)))
			}
		})
	},
}

var aiProfileCmd = &cobra.Command{
	Use:   "profile",
	Short: "AI audit and optimization of your LinkedIn profile",
	RunE: func(cmd *cobra.Command, args []string) error {
		return profileOptimizeCmd.RunE(cmd, args)
	},
}

var aiReplyCmd = &cobra.Command{
	Use:   "reply <comment-urn> [instructions]",
	Args:  cobra.MinimumNArgs(1),
	Short: "Draft an insightful reply to a comment",
	RunE: func(cmd *cobra.Command, args []string) error {
		commentURN := args[0]
		contextPrompt := "reply professionally and add technical depth"
		if len(args) > 1 {
			contextPrompt = strings.Join(args[1:], " ")
		}

		eng, err := agent.NewEngine(ConfigMgr, LinkedInClient)
		if err != nil {
			return err
		}

		ctx := context.Background()
		result, err := eng.Run(ctx, fmt.Sprintf("Draft a concise, thoughtful technical reply to comment %s with instruction: %s", commentURN, contextPrompt))
		if err != nil {
			return err
		}

		return Formatter.Print(result, func() {
			fmt.Println(output.TitleStyle.Render(" Suggested Reply "))
			fmt.Println(result.Response)
			fmt.Println()
			Formatter.Info("To send: `ldin comment reply <post-urn> %s \"%s\"`", commentURN, strings.ReplaceAll(result.Response, "\"", "\\\""))
		})
	},
}

func init() {
	aiCmd.AddCommand(aiPostCmd)
	aiCmd.AddCommand(aiProfileCmd)
	aiCmd.AddCommand(aiReplyCmd)

	RootCmd.AddCommand(aiCmd)
}
